package postretrieval

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

var rerankExecutorDescriptor = "post.Rerank"

func init() {
	exec, err := NewRerankExecutor()
	if err != nil {
		slog.Error("failed to initialize executor", "name", rerankExecutorDescriptor, "err", err)
		return
	}

	err = registry.RegisterExecutor(rerankExecutorDescriptor, exec)
	if err != nil {
		slog.Error("failed to register executor", "name", rerankExecutorDescriptor)
	}
}

type RerankExecutor struct {
	DefaultReranker provider.Reranker
	operators       map[string]func(context.Context, *executor.ExecutorParams) (map[string]any, error)
}

func NewRerankExecutor() (*RerankExecutor, error) {
	rp, err := provider.NewReranker(provider.RerankerTypeCohere)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize default providers: %e", err)
	}

	e := &RerankExecutor{
		DefaultReranker: rp,
	}
	e.operators = map[string]func(context.Context, *executor.ExecutorParams) (map[string]any, error){
		"cohere_rerank": e.cohereRerank,
	}
	return e, nil
}

func (e RerankExecutor) Execute(ctx context.Context, p *executor.ExecutorParams) *executor.ExecutorResult {
	if p.Operator == "" {
		p.Operator = "cohere_rerank"
	}
	slog.Info("executing", "name", rerankExecutorDescriptor, "op", p.Operator, "query", p.GetQuery(), "id", p.GetTaskID())

	opFunc, exists := e.operators[p.Operator]
	if !exists {
		return &executor.ExecutorResult{
			Name:     rerankExecutorDescriptor,
			Operator: p.Operator,
			Err: executor.ErrOperatorNotFound{
				ExecutorName: rerankExecutorDescriptor,
				OperatorName: p.Operator,
			},
			Values: nil,
		}
	}

	vals, err := opFunc(ctx, p)

	return &executor.ExecutorResult{
		Name:     rerankExecutorDescriptor,
		Operator: p.Operator,
		Err:      err,
		Values:   vals,
	}
}

func (e RerankExecutor) cohereRerank(ctx context.Context, p *executor.ExecutorParams) (map[string]any, error) {
	// 'cohere_rerank' requires following parameter args:
	// context_docs - slice of scored documents to be used as context
	context, err := executor.GetTypedArg[[]*api.ScoredDocument](p, "context_docs")
	if err != nil {
		return nil, err
	}

	// Optional
	// top_n - limit the amount of documents returned after reranking
	var topN int
	topN_raw, err := executor.GetTypedArg[uint64](p, "top_n")
	if err != nil {
		if _, ok := err.(executor.ErrArgMissing); !ok {
			return nil, err
		}
	} else {
		if topN_raw > uint64(math.MaxInt64) {
			return nil, fmt.Errorf("top_n value is of out int64 range")
		}
		topN = int(topN_raw)
	}

	texts := make([]string, 0, len(context))
	for _, sp := range context {
		if sp.Content == "" {
			slog.Warn("malformed retrieved context document: missing content", "doc", sp)
		} else {
			texts = append(texts, sp.Content)
		}
	}

	rerankRequest := &api.RerankRequest{
		Query:     p.GetQuery(),
		Documents: texts,
	}
	if topN != 0 {
		rerankRequest.Limit = topN
	}

	resp, err := e.DefaultReranker.Rerank(ctx, *rerankRequest)
	if err != nil {
		return nil, fmt.Errorf("rerank request failed: %w", err)
	}

	return map[string]any{
		"context_docs":    resp.Documents,
		"replace_context": true,
	}, nil
}
