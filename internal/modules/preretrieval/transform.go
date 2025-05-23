package preretrieval

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"text/template"

	"github.com/alan-mat/awe/internal/api"
	"github.com/alan-mat/awe/internal/executor"
	"github.com/alan-mat/awe/internal/provider"
	"github.com/alan-mat/awe/internal/registry"
)

var transformExecutorDescriptor = "pre.QueryTransform"

const (
	promptRewrite = `You are an expert in query reformulation for information retrieval. Your task is to rewrite the following user query to improve its clarity, specificity, and semantic relevance for search engines. Consider potential user intent, related concepts, and synonyms. Generate only one rewrite. Answer only with the rewritten query, no additional preamble or suffix.

User Query:
{{.Query}}

Rewritten Query:
`
)

func init() {
	exec, err := NewTransformExecutor()
	if err != nil {
		slog.Error("failed to initialize executor", "name", transformExecutorDescriptor, "err", err)
		return
	}

	err = registry.RegisterExecutor(transformExecutorDescriptor, exec)
	if err != nil {
		slog.Error("failed to register executor", "name", transformExecutorDescriptor)
	}
}

type TransformExecutor struct {
	DefaultLMProvider provider.LM
	promptRewrite     *template.Template
	operators         map[string]func(context.Context, *executor.ExecutorParams) (map[string]any, error)
}

func NewTransformExecutor() (*TransformExecutor, error) {
	lp, err := provider.NewLM(provider.LMTypeGemini)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize default providers: %e", err)
	}

	templ := template.Must(template.New("promptRewrite").Parse(promptRewrite))

	e := &TransformExecutor{
		DefaultLMProvider: lp,
		promptRewrite:     templ,
	}
	e.operators = map[string]func(context.Context, *executor.ExecutorParams) (map[string]any, error){
		"rewrite": e.rewriteSimple,
	}
	return e, nil
}

func (e TransformExecutor) Execute(ctx context.Context, p *executor.ExecutorParams) *executor.ExecutorResult {
	if p.Operator == "" {
		p.Operator = "rewrite"
	}
	slog.Info("executing", "name", transformExecutorDescriptor, "op", p.Operator, "query", p.GetQuery(), "id", p.GetTaskID())

	opFunc, exists := e.operators[p.Operator]
	if !exists {
		return &executor.ExecutorResult{
			Name:     transformExecutorDescriptor,
			Operator: p.Operator,
			Err: executor.ErrOperatorNotFound{
				ExecutorName: transformExecutorDescriptor,
				OperatorName: p.Operator,
			},
			Values: nil,
		}
	}

	vals, err := opFunc(ctx, p)

	return &executor.ExecutorResult{
		Name:     transformExecutorDescriptor,
		Operator: p.Operator,
		Err:      err,
		Values:   vals,
	}
}

func (e TransformExecutor) rewriteSimple(ctx context.Context, p *executor.ExecutorParams) (map[string]any, error) {
	type templatePayload struct {
		Query string
	}
	tp := templatePayload{Query: p.GetQuery()}

	var buf bytes.Buffer
	err := e.promptRewrite.Execute(&buf, tp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse prompt template for query '%s': %w", p.GetQuery(), err)
	}
	parsedPrompt := buf.String()

	req := api.GenerationRequest{
		Prompt:      parsedPrompt,
		Temperature: 0.2,
	}
	cs, err := e.DefaultLMProvider.Generate(ctx, req)
	if err != nil {
		slog.Warn("error creating generation completion stream, cancelling task")
		return nil, err
	}

	resp, err := api.StreamReadAll(ctx, cs)
	if err != nil {
		return nil, fmt.Errorf("failed to read response stream: %w", err)
	}

	return map[string]any{
		"query_original":    p.GetQuery(),
		"query_transformed": resp,
	}, nil
}
