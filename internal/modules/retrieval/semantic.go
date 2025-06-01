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

package retrieval

import (
	"context"
	"fmt"
	"log/slog"
	"math"

	"github.com/alan-mat/awe/internal/executor"
	"github.com/alan-mat/awe/internal/provider"
	"github.com/alan-mat/awe/internal/registry"
	"github.com/alan-mat/awe/internal/vector"
)

var semanticExecutorDescriptor = "retrieval.Semantic"

func init() {
	exec, err := NewSemanticExecutor()
	if err != nil {
		slog.Error("failed to initialize executor", "name", semanticExecutorDescriptor, "err", err)
		return
	}

	err = registry.RegisterExecutor(semanticExecutorDescriptor, exec)
	if err != nil {
		slog.Error("failed to register executor", "name", semanticExecutorDescriptor)
	}
}

type SemanticExecutor struct {
	DefaultEmbedProvider provider.Embedder
	operators            map[string]func(context.Context, *executor.ExecutorParams) (map[string]any, error)
}

func NewSemanticExecutor() (*SemanticExecutor, error) {
	ep, err := provider.NewEmbedder(provider.EmbedderTypeOpenai)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize default providers: %e", err)
	}

	e := &SemanticExecutor{
		DefaultEmbedProvider: ep,
	}
	e.operators = map[string]func(context.Context, *executor.ExecutorParams) (map[string]any, error){
		"dense": e.denseRetrieval,
	}
	return e, nil
}

func (e *SemanticExecutor) Execute(ctx context.Context, p *executor.ExecutorParams) *executor.ExecutorResult {
	if p.Operator == "" {
		p.Operator = "dense"
	}
	slog.Info("executing", "name", semanticExecutorDescriptor, "op", p.Operator, "query", p.GetQuery(), "id", p.GetTaskID())

	opFunc, exists := e.operators[p.Operator]
	if !exists {
		return e.buildResult(p.Operator, executor.ErrOperatorNotFound{
			ExecutorName: semanticExecutorDescriptor, OperatorName: p.Operator}, nil)
	}

	vals, err := opFunc(ctx, p)

	return e.buildResult(p.Operator, err, vals)
}

func (e *SemanticExecutor) denseRetrieval(ctx context.Context, p *executor.ExecutorParams) (map[string]any, error) {
	// 'dense' requires following parameter args:
	// collection_name - name of the collection to use for the vector store
	collectionName, err := executor.GetTypedArg[string](p, "collection_name")
	if err != nil {
		return nil, err
	}

	if p.VectorStore == nil {
		return nil, fmt.Errorf("operator failed: vector store is not initialized")
	}

	vec, err := e.DefaultEmbedProvider.EmbedQuery(ctx, p.GetQuery())
	if err != nil {
		return nil, fmt.Errorf("failed to embed query '%s': %e", p.GetQuery(), err)
	}

	// Optional
	// top_n - limit the amount of documents returned after reranking
	var topN uint = 25
	topN_raw, err := executor.GetTypedArg[uint64](p, "top_n")
	if err == nil {
		if topN_raw > uint64(math.MaxInt64) {
			return nil, fmt.Errorf("top_n value is of out int64 range")
		}
		topN = uint(topN_raw)
	}

	queryParams := vector.NewQueryParams(
		collectionName,
		vec,
		vector.WithPayload(true),
		vector.WithLimit(topN),
	)

	docs, err := p.VectorStore.Query(ctx, queryParams)
	if err != nil {
		return nil, fmt.Errorf("failed to get results for query '%s': %e", p.GetQuery(), err)
	}

	return map[string]any{
		"context_docs": docs,
	}, nil
}

func (e *SemanticExecutor) buildResult(operator string, err error, values map[string]any) *executor.ExecutorResult {
	return &executor.ExecutorResult{
		Name:     semanticExecutorDescriptor,
		Operator: operator,
		Err:      err,
		Values:   values,
	}
}
