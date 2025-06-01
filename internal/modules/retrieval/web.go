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

	"github.com/alan-mat/awe/internal/api"
	"github.com/alan-mat/awe/internal/executor"
	"github.com/alan-mat/awe/internal/provider"
	"github.com/alan-mat/awe/internal/registry"
)

var webRetrieverExecutorDescriptor = "retrieval.Web"

func init() {
	exec, err := NewWebExecutor()
	if err != nil {
		slog.Error("failed to initialize executor", "name", webRetrieverExecutorDescriptor, "err", err)
		return
	}

	err = registry.RegisterExecutor(webRetrieverExecutorDescriptor, exec)
	if err != nil {
		slog.Error("failed to register executor", "name", webRetrieverExecutorDescriptor)
	}
}

type WebExecutor struct {
	DefaultWebSearcher provider.WebSearcher
	operators          map[string]func(context.Context, *executor.ExecutorParams) (map[string]any, error)
}

func NewWebExecutor() (*WebExecutor, error) {
	wp, err := provider.NewWebSearcher(provider.WebSearcherTypeTavily)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize default providers: %w", err)
	}

	e := &WebExecutor{
		DefaultWebSearcher: wp,
	}
	e.operators = map[string]func(context.Context, *executor.ExecutorParams) (map[string]any, error){
		"search": e.webSearch,
	}
	return e, nil
}

func (e WebExecutor) Execute(ctx context.Context, p *executor.ExecutorParams) *executor.ExecutorResult {
	if p.Operator == "" {
		p.Operator = "search"
	}
	slog.Info("executing", "name", webRetrieverExecutorDescriptor, "op", p.Operator, "query", p.GetQuery(), "id", p.GetTaskID())

	opFunc, exists := e.operators[p.Operator]
	if !exists {
		return &executor.ExecutorResult{
			Name:     webRetrieverExecutorDescriptor,
			Operator: p.Operator,
			Err: executor.ErrOperatorNotFound{
				ExecutorName: webRetrieverExecutorDescriptor,
				OperatorName: p.Operator,
			},
			Values: nil,
		}
	}

	vals, err := opFunc(ctx, p)

	return &executor.ExecutorResult{
		Name:     webRetrieverExecutorDescriptor,
		Operator: p.Operator,
		Err:      err,
		Values:   vals,
	}
}

func (e WebExecutor) webSearch(ctx context.Context, p *executor.ExecutorParams) (map[string]any, error) {
	req := api.WebSearchRequest{
		Query: p.GetQuery(),
	}

	// Optional
	// top_n - limit the amount of documents returned after reranking
	topN, err := executor.GetTypedArg[uint64](p, "top_n")
	if err == nil {
		if topN > uint64(math.MaxInt64) {
			return nil, fmt.Errorf("top_n value is of out int64 range")
		}
		req.Limit = int(topN)
	}

	resp, err := e.DefaultWebSearcher.Search(ctx, req)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"context_docs": resp.Results,
	}, nil
}
