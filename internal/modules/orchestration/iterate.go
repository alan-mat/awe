// Copyright 2025 Alan Matykiewicz
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to use,
// copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the
// Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES
// OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
// HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
// WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
// OTHER DEALINGS IN THE SOFTWARE.

package orchestration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"text/template"

	"github.com/alan-mat/awe/internal/api"
	"github.com/alan-mat/awe/internal/executor"
	"github.com/alan-mat/awe/internal/provider"
	"github.com/alan-mat/awe/internal/registry"
)

var iterateExecutorDescriptor = "orchestration.Iterate"

const (
	promptLLMJudge = `You are an expert judge for a Retrieval-Augmented Generation (RAG) system. Your task is to evaluate the provided context documents against a user's original query.

**Input:**
1.  **Original User Query:** The initial question or request from the user.
2.  **Rewritten Queries (Optional):** A list of queries that have already been generated in previous turns to retrieve context. If this is the first turn, this list will be empty.
3.  **Context Documents:** A collection of text snippets retrieved from a vector database based on the current or previous queries.

**Goal:**
Determine if the 'Context Documents' are sufficient to fully, accurately, and truthfully answer the 'Original User Query'.

**Instructions:**
*   **Confidence Threshold:** You must be 100% confident that all aspects of the 'Original User Query' can be completely and correctly addressed using *only* the provided 'Context Documents'.
*   **Missing Information Detection:** Carefully analyze the 'Original User Query' and identify any information gaps that the 'Context Documents' fail to cover or cover inadequately.
*   **Query Rewriting (if needed):** If you are NOT 100% confident that the 'Context Documents' are sufficient, you MUST generate a new, expertly semantically formed query. This new query should aim to retrieve the *missing* information from the vector database. It should be concise, clear, and directly target the information gap identified. Avoid generating queries that would retrieve redundant information already present in the 'Context Documents'.
*   **Avoid Redundancy:** Do not generate a new query if an identical or semantically equivalent query is already present in the 'Rewritten Queries' list.
*   Respond using JSON.

**Original User Query:**
{{.Query}}

**Rewritten Queries (Optional):**
{{.QueryList}}

**Context Documents:**
{{.Context}}
`
)

func init() {
	exec, err := NewIterateExecutor()
	if err != nil {
		slog.Error("failed to initialize executor", "name", iterateExecutorDescriptor, "err", err)
		return
	}

	err = registry.RegisterExecutor(iterateExecutorDescriptor, exec)
	if err != nil {
		slog.Error("failed to register executor", "name", iterateExecutorDescriptor)
	}
}

type IterateExecutor struct {
	DefaultLM provider.LM

	operators        map[string]func(context.Context, *executor.ExecutorParams) (map[string]any, error)
	templateLLMJudge template.Template
}

func NewIterateExecutor() (*IterateExecutor, error) {
	lp, _ := provider.NewLM(provider.LMTypeOpenai)
	templ := template.Must(template.New("promptLLMJudge").Parse(promptLLMJudge))

	e := &IterateExecutor{
		DefaultLM:        lp,
		templateLLMJudge: *templ,
	}
	e.operators = map[string]func(context.Context, *executor.ExecutorParams) (map[string]any, error){
		"fixed_iters":               e.fixedIterations,
		"llm_judge_context_rewrite": e.llmJudgeContextRewrite,
	}
	return e, nil
}

func (e IterateExecutor) Execute(ctx context.Context, p *executor.ExecutorParams) *executor.ExecutorResult {
	if p.Operator == "" {
		p.Operator = "fixed_iters"
	}
	slog.Info("executing", "name", iterateExecutorDescriptor, "op", p.Operator, "query", p.GetQuery(), "id", p.GetTaskID())

	opFunc, exists := e.operators[p.Operator]
	if !exists {
		return &executor.ExecutorResult{
			Name:     iterateExecutorDescriptor,
			Operator: p.Operator,
			Err: executor.ErrOperatorNotFound{
				ExecutorName: iterateExecutorDescriptor,
				OperatorName: p.Operator,
			},
			Values: nil,
		}
	}

	if len(p.Children) == 0 {
		return &executor.ExecutorResult{
			Name:     iterateExecutorDescriptor,
			Operator: p.Operator,
			Err: executor.ErrInvalidParams{
				ExecutorName:  iterateExecutorDescriptor,
				MissingParams: []string{"children"},
			},
			Values: nil,
		}
	}

	vals, err := opFunc(ctx, p)

	return &executor.ExecutorResult{
		Name:     iterateExecutorDescriptor,
		Operator: p.Operator,
		Err:      err,
		Values:   vals,
	}
}

func (e IterateExecutor) fixedIterations(ctx context.Context, p *executor.ExecutorParams) (map[string]any, error) {
	// Optional
	// num_iters - sets the amount of iterations (default: 3)
	numIters := 3
	numItersArg, err := executor.GetTypedArg[uint64](p, "num_iters")
	if err == nil {
		if numItersArg > uint64(math.MaxInt64) {
			return nil, fmt.Errorf("num_iters value is of out int64 range")
		}
		numIters = int(numItersArg)
	}

	runtimeParams := p.Copy()
	for i := range numIters {

		for j, node := range p.Children {
			slog.Info("running iteration", "i", i, "nodeIdx", j)

			nodeParams := executor.MakeNodeParams(node, runtimeParams)

			result := node.Executor.Execute(ctx, nodeParams)

			if result.Err != nil {
				slog.Error("failed to execute node", "error", fmt.Sprintf("(%T): %v", result.Err, result.Err))
				return nil, fmt.Errorf("iteration failed on node '%s': %w", node.Operator, result.Err)
			}

			runtimeParams = executor.ProcessResult(runtimeParams, result)
		}

	}

	return runtimeParams.Args, nil
}

func (e IterateExecutor) llmJudgeContextRewrite(ctx context.Context, p *executor.ExecutorParams) (map[string]any, error) {
	// Optional
	// max_iters - sets the maximum amount of iterations (default: 3)
	maxIters := 3
	maxItersArg, err := executor.GetTypedArg[uint64](p, "max_iters")
	if err == nil {
		if maxItersArg > uint64(math.MaxInt64) {
			return nil, fmt.Errorf("max_iters value is of out int64 range")
		}
		maxIters = int(maxItersArg)
	}

	schema := &api.Schema{
		Title: "context_judge_with_rewrite",
		Type:  api.TypeObject,
		Properties: map[string]*api.Schema{
			"sufficient": {
				Type:        api.TypeBoolean,
				Description: "If the provided context documents are sufficient to fully, accurately, and truthfully answer the original user query.",
			},
			"new_query": {
				Type:        api.TypeString,
				Description: "New rewritten query to semantically retrieve missing information.",
			},
		},
		Required: []string{"sufficient"},
	}

	fullContext := make([]*api.ScoredDocument, 0, 10)
	queryList := make([]string, 0)
	runtimeParams := p.Copy()
	for i := range maxIters {

		for j, node := range p.Children {
			slog.Info("running iteration", "i", i, "nodeIdx", j)

			nodeParams := executor.MakeNodeParams(node, runtimeParams)

			result := node.Executor.Execute(ctx, nodeParams)

			if result.Err != nil {
				slog.Error("failed to execute node", "error", fmt.Sprintf("(%T): %v", result.Err, result.Err))
				return nil, fmt.Errorf("iteration failed on node '%s': %w", node.Operator, result.Err)
			}

			runtimeParams = executor.ProcessResult(runtimeParams, result)
		}

		// gather context_docs
		// get response from judge
		// if sufficient, break
		// if not sufficient, set rewritten query as runtimeParams
		context, err := executor.GetTypedArg[[]*api.ScoredDocument](runtimeParams, "context_docs")
		if err != nil {
			fmt.Println(err)
		}
		fullContext = append(fullContext, context...)

		// deduplicate the context slice
		seen := make(map[string]bool)
		var res []*api.ScoredDocument
		for _, c := range fullContext {
			trimmed := strings.TrimSpace(c.Content)
			if _, found := seen[trimmed]; !found {
				seen[trimmed] = true
				res = append(res, c)
			}
		}
		fullContext = res

		var c string
		for _, doc := range fullContext {
			c += fmt.Sprintf("  - %s\n", doc.Content)
		}

		type templatePayload struct {
			Query     string
			QueryList []string
			Context   string
		}
		tp := templatePayload{
			Query:     p.GetQuery(),
			QueryList: queryList,
			Context:   c,
		}

		var buf bytes.Buffer
		err = e.templateLLMJudge.Execute(&buf, tp)
		if err != nil {
			return nil, fmt.Errorf("failed to parse prompt template for query '%s': %w", p.GetQuery(), err)
		}
		parsedPrompt := buf.String()

		req := api.GenerationRequest{
			Prompt:         parsedPrompt,
			ResponseSchema: schema,
			//Temperature:    0.2,
		}

		cs, err := e.DefaultLM.Generate(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("failed to generate llm judge results: %w", err)
		}

		type judgeResponse struct {
			Sufficient bool   `json:"sufficient"`
			NewQuery   string `json:"new_query,omitzero"`
		}
		var judge judgeResponse

		resp, err := api.StreamReadAll(ctx, cs)
		if err != nil {
			return nil, fmt.Errorf("failed to read completions stream: %w", err)
		}
		err = json.Unmarshal([]byte(resp), &judge)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal judge response: %w", err)
		}

		queryList = append(queryList, judge.NewQuery)
		if judge.Sufficient {
			break
		} else {
			runtimeParams.SetQuery(judge.NewQuery)
		}

	}

	return map[string]any{
		"context_docs": fullContext,
	}, nil
}
