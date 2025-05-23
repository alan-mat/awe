package retrieval

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/alan-mat/awe/internal/api"
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
	ep, err := provider.NewEmbedder(provider.EmbedderTypeJina)
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

	queryParams := vector.NewQueryParams(
		collectionName,
		vec,
		vector.WithPayload(true),
		vector.WithLimit(25),
	)

	points, err := p.VectorStore.Query(ctx, queryParams)
	if err != nil {
		return nil, fmt.Errorf("failed to get results for query '%s': %e", p.GetQuery(), err)
	}

	scoredDocs := make([]*api.ScoredDocument, 0, len(points))
	for _, p := range points {
		t, ok := p.Payload["text"]
		if !ok {
			slog.Warn("malformed retrieved context point: missing 'text' field in payload", "id", p.ID, "payload", p.Payload)
			continue
		}
		scoredDocs = append(scoredDocs, &api.ScoredDocument{
			Document: t,
			Score:    float64(p.Score),
		})
	}

	return map[string]any{
		"context_points": points,
		"context_docs":   scoredDocs,
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
