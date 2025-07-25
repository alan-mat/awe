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

package generation

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"text/template"

	"github.com/alan-mat/awe/internal/api"
	"github.com/alan-mat/awe/internal/executor"
	"github.com/alan-mat/awe/internal/provider"
	"github.com/alan-mat/awe/internal/registry"
	"github.com/alan-mat/awe/internal/transport"
)

var augmentedExecutorDescriptor = "generation.Augmented"

const (
	/* promptGenerateWithContext = `You are an AI assistant that answers user queries. You have been provided with some potentially relevant context, which you should use to inform and support your answer.

	**INSTRUCTIONS:**

	1.  Read the provided CONTEXT to understand if it is relevant to the user's QUERY and how it can supplement your knowledge.
	2.  Understand the user's QUERY.
	3.  Formulate a comprehensive answer to the QUERY, drawing upon both the provided CONTEXT and your own internal knowledge.
	4.  Use the CONTEXT to provide specific details, examples, or confirmation where applicable.
	5.  If the CONTEXT provides information that is highly relevant or crucial to the answer, integrate it seamlessly.
	6.  If the CONTEXT is not very relevant to the QUERY, rely primarily on your internal knowledge.
	7.  Format your answer clearly and use formatting (like bullet points or bolding) when appropriate for readability.

	**CONTEXT:**
	{{.Context}}

	**QUERY:**
	` */
	promptGenerateWithContext = `You are an AI assistant that answers user queries. You have been provided with some potentially relevant context, which you should use to inform and support your answer.

**INSTRUCTIONS:**

1.  Read the provided CONTEXT to understand if it is relevant to the user's QUERY and how it can supplement your knowledge.
2.  Understand the user's QUERY.
3.  Formulate a clear but concise answer to the user's QUERY, relying on the provided CONTEXT. 
4.  If you cannot answer the QUERY using the CONTEXT provided, answer with "I don't know".
5.  Respond with as little words as necessary to completely answer the question. Your answer does not have to be a complete sentence. If the QUERY is answerable in just one word, do it in one word.

**CONTEXT:**
{{.Context}}

**QUERY:**
`
)

func init() {
	exec, err := NewAugmentedExecutor()
	if err != nil {
		slog.Error("failed to initialize executor", "name", augmentedExecutorDescriptor, "err", err)
		return
	}

	err = registry.RegisterExecutor(augmentedExecutorDescriptor, exec)
	if err != nil {
		slog.Error("failed to register executor", "name", augmentedExecutorDescriptor)
	}
}

type AugmentedExecutor struct {
	DefaultEmbedProvider provider.Embedder
	DefaultLMProvider    provider.LM

	operators map[string]func(context.Context, *executor.ExecutorParams) (map[string]any, error)

	templateGenerateWithContext template.Template
}

func NewAugmentedExecutor() (*AugmentedExecutor, error) {
	ep, err := provider.NewEmbedder(provider.EmbedderTypeJina)
	lp, err2 := provider.NewLM(provider.LMTypeOpenai)
	joinedErr := errors.Join(err, err2)
	if joinedErr != nil {
		return nil, fmt.Errorf("failed to initialize default providers: %w", err)
	}

	templ := template.Must(template.New("promptGenerateWithContext").Parse(promptGenerateWithContext))

	e := &AugmentedExecutor{
		DefaultEmbedProvider:        ep,
		DefaultLMProvider:           lp,
		templateGenerateWithContext: *templ,
	}
	e.operators = map[string]func(context.Context, *executor.ExecutorParams) (map[string]any, error){
		"gen_context": e.generateWithContext,
	}
	return e, nil
}

func (e AugmentedExecutor) Execute(ctx context.Context, p *executor.ExecutorParams) *executor.ExecutorResult {
	if p.Operator == "" {
		p.Operator = "gen_context"
	}
	slog.Info("executing", "name", augmentedExecutorDescriptor, "op", p.Operator, "query", p.GetQuery(), "id", p.GetTaskID())

	opFunc, exists := e.operators[p.Operator]
	if !exists {
		return &executor.ExecutorResult{
			Name:     augmentedExecutorDescriptor,
			Operator: p.Operator,
			Err: executor.ErrOperatorNotFound{
				ExecutorName: augmentedExecutorDescriptor,
				OperatorName: p.Operator,
			},
			Values: nil,
		}
	}

	vals, err := opFunc(ctx, p)

	return &executor.ExecutorResult{
		Name:     augmentedExecutorDescriptor,
		Operator: p.Operator,
		Err:      err,
		Values:   vals,
	}
}

func (e AugmentedExecutor) generateWithContext(ctx context.Context, p *executor.ExecutorParams) (map[string]any, error) {
	// 'gen_context' requires one of the following parameter args:
	// context_docs - slice of scored documents to be used as context
	//					(from vector store or after post-retrieval)
	context, err := executor.GetTypedArg[[]*api.ScoredDocument](p, "context_docs")
	if err != nil {
		return nil, err
	}

	modelContext := ""
	for _, sp := range context {
		// slog.Info("got point", "score", sp.Score, "text", sp.Content)
		modelContext += strings.TrimSpace(sp.Content) + "\n---\n"
	}

	type templatePayload struct {
		Context string
	}
	tp := templatePayload{Context: modelContext}

	var buf bytes.Buffer
	err = e.templateGenerateWithContext.Execute(&buf, tp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse prompt template for query '%s': %w", p.GetQuery(), err)
	}
	parsedPrompt := buf.String()

	msgStream, err := p.Transport.GetMessageStream(p.GetTaskID())
	if err != nil {
		slog.Warn("failed to create message stream", "id", p.GetTaskID())
		msgStream.Send(ctx, transport.MessageStreamPayload{
			Content: "something went wrong",
			Status:  "ERR",
		})
		return nil, err
	}

	stream, err := e.DefaultLMProvider.Chat(ctx, api.ChatRequest{
		Query:        p.GetQuery(),
		SystemPrompt: parsedPrompt,
	})
	if err != nil {
		slog.Warn("error creating chat completion stream, cancelling task")
		msgStream.Send(ctx, transport.MessageStreamPayload{
			Content: "something went wrong",
			Status:  "ERR",
		})
		return nil, err
	}
	defer stream.Close()

	output, err := transport.ProcessCompletionStream(ctx, msgStream, stream)
	if err != nil {
		return nil, fmt.Errorf("failed to process completion stream: %w", err)
	}

	return map[string]any{
		"generation_results": output,
	}, nil
}
