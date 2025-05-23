package indexing

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/alan-mat/awe/internal/api"
	"github.com/alan-mat/awe/internal/executor"
	"github.com/alan-mat/awe/internal/provider"
	"github.com/alan-mat/awe/internal/registry"
	"github.com/alan-mat/awe/internal/vector"
)

var simpleExecutorDescriptor = "indexing.Simple"

func init() {
	exec, err := NewSimpleExecutor()
	if err != nil {
		slog.Error("failed to initialize executor", "name", simpleExecutorDescriptor, "err", err)
		return
	}

	err = registry.RegisterExecutor(simpleExecutorDescriptor, exec)
	if err != nil {
		slog.Error("failed to register executor", "name", simpleExecutorDescriptor, "err", err)
	}
}

type SimpleExecutor struct {
	DefaultParseProvider   provider.DocParser
	DefaultSegmentProvider provider.Segmenter
	DefaultEmbedProvider   provider.Embedder
	operators              map[string]func(context.Context, *executor.ExecutorParams) (map[string]any, error)
}

func NewSimpleExecutor() (*SimpleExecutor, error) {
	pp, err1 := provider.NewDocParser(provider.DocParserTypeMistral)
	sp, err2 := provider.NewSegmenter(provider.SegmenterTypeJina)
	ep, err3 := provider.NewEmbedder(provider.EmbedderTypeJina)
	joinedErr := errors.Join(err1, err2, err3)
	if joinedErr != nil {
		return nil, fmt.Errorf("failed to initialize default providers: %e", joinedErr)
	}

	e := &SimpleExecutor{
		DefaultParseProvider:   pp,
		DefaultSegmentProvider: sp,
		DefaultEmbedProvider:   ep,
	}
	e.operators = map[string]func(context.Context, *executor.ExecutorParams) (map[string]any, error){
		"index_files_base64": e.indexFilesBase64,
	}
	return e, nil
}

func (e SimpleExecutor) Execute(ctx context.Context, p *executor.ExecutorParams) *executor.ExecutorResult {
	if p.Operator == "" {
		p.Operator = "index_files_base64"
	}
	slog.Info("executing", "name", simpleExecutorDescriptor, "op", p.Operator, "query", p.GetQuery(), "id", p.GetTaskID())

	opFunc, exists := e.operators[p.Operator]
	if !exists {
		return e.buildResult(p.Operator, executor.ErrOperatorNotFound{
			ExecutorName: simpleExecutorDescriptor, OperatorName: p.Operator}, nil)
	}

	vals, err := opFunc(ctx, p)
	if err == nil {
		slog.Info("indexing results", "values", vals)
	}

	return e.buildResult(p.Operator, err, vals)
}

func (e SimpleExecutor) indexFilesBase64(ctx context.Context, p *executor.ExecutorParams) (map[string]any, error) {
	// 'index_files_base64' requires following parameter args:
	// file_contents - contains the base64 encoded files to index; type must be []*message.FileContent
	// collection_name - name of the collection to use for the vector store
	fcArg, err := p.GetArg("file_contents")
	if err != nil {
		return nil, err
	}

	files, ok := fcArg.([]*api.FileContent)
	if !ok {
		return nil, fmt.Errorf("argument 'file_contents' must be of type '[]*message.FileContent'")
	}

	cnArg, err := p.GetArg("collection_name")
	if err != nil {
		return nil, err
	}

	collectionName, ok := cnArg.(string)
	if !ok {
		return nil, fmt.Errorf("argument 'collection_name' must be of type 'string'")
	}

	if exists, err := p.VectorStore.CollectionExists(ctx, collectionName); err == nil {
		if !exists {
			slog.Info("requested collection not found", "name", collectionName)

			err := p.VectorStore.CreateCollection(ctx, vector.Collection{
				Name:       collectionName,
				Dimensions: e.DefaultEmbedProvider.GetDimensions(),
			})

			slog.Info("successfully created collection", "name", collectionName)

			if err != nil {
				return nil, fmt.Errorf("failed to create collection: %e", err)
			}
		}
	} else {
		return nil, fmt.Errorf("failed to communicate with vector store: %e", err)
	}

	var wg sync.WaitGroup
	var docReqMu sync.Mutex
	docRequests := make([]*api.EmbedDocumentRequest, 0, len(files))

	for _, file := range files {
		wg.Add(1)
		ctxTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		go func(ctx context.Context, file *api.FileContent) {
			defer wg.Done()

			chunks := e.parseAndSegmentFile(ctx, file)
			if len(chunks) > 0 {
				docReqMu.Lock()
				docRequests = append(docRequests, &api.EmbedDocumentRequest{
					Title:  file.Name,
					Chunks: chunks,
				})
				docReqMu.Unlock()
			}
		}(ctxTimeout, file)

		time.Sleep(100 * time.Millisecond)
	}
	wg.Wait()

	if len(docRequests) == 0 {
		return nil, fmt.Errorf("failed to index files: no files parsed")
	}

	embeddings, err := e.DefaultEmbedProvider.EmbedDocuments(ctx, docRequests)
	if err != nil {
		return nil, fmt.Errorf("failed to embed %d documents: %e", len(docRequests), err)
	}

	points := vector.CreatePoints(embeddings)
	err = p.VectorStore.Upsert(ctx, collectionName, points)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert points to vector store: %e", err)
	}

	return map[string]any{
		"points_indexed": len(points),
	}, nil
}

func (e SimpleExecutor) parseAndSegmentFile(ctx context.Context, file *api.FileContent) []string {
	parsed, err := e.DefaultParseProvider.Parse(ctx, file.Content)
	if err != nil {
		slog.Error("failed to parse file, skipping...", "name", file.Name, "err", err)
		return nil
	}

	chunks, err := e.DefaultSegmentProvider.ChunkDocument(ctx, parsed)
	if err != nil {
		slog.Error("failed to segment file, skipping...", "name", file.Name, "err", err)
		return nil
	}
	return chunks
}

func (e SimpleExecutor) buildResult(operator string, err error, values map[string]any) *executor.ExecutorResult {
	return &executor.ExecutorResult{
		Name:     simpleExecutorDescriptor,
		Operator: operator,
		Err:      err,
		Values:   values,
	}
}
