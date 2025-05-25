package worker

import (
	"fmt"

	"github.com/alan-mat/awe/internal/config"
	"github.com/alan-mat/awe/internal/registry"
	"github.com/alan-mat/awe/internal/tasks"
	"github.com/alan-mat/awe/internal/transport"
	"github.com/alan-mat/awe/internal/vector"
	"github.com/hibiken/asynq"

	"github.com/redis/go-redis/v9"

	_ "github.com/alan-mat/awe/internal/modules/generation"
	_ "github.com/alan-mat/awe/internal/modules/indexing"
	_ "github.com/alan-mat/awe/internal/modules/orchestration"
	_ "github.com/alan-mat/awe/internal/modules/postretrieval"
	_ "github.com/alan-mat/awe/internal/modules/preretrieval"
	_ "github.com/alan-mat/awe/internal/modules/retrieval"
	_ "github.com/alan-mat/awe/internal/modules/system"
)

type Worker struct {
	rdb         *redis.Client
	asynqServer *asynq.Server

	transport   transport.Transport
	vectorStore vector.Store
}

func New() *Worker {
	return &Worker{}
}

func (w Worker) RegisterWorkflows(path string) error {
	wc := config.ReadConfig(path)
	workflows, err := config.ParseWorkflows(wc)
	if err != nil {
		return fmt.Errorf("failed to parse workflows config: %v", err)
	}

	err = registry.BatchRegisterWorkflows(workflows)
	if err != nil {
		return fmt.Errorf("failed to register workflows: %v", err)
	}
	return nil
}

func (w *Worker) Start() error {
	w.rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // use default Addr
		Password: "",               // no password set
		DB:       0,                // use default DB
	})
	defer w.rdb.Close()

	w.asynqServer = asynq.NewServerFromRedisClient(
		w.rdb,
		asynq.Config{
			Concurrency: 10,
		},
	)

	w.transport = transport.NewRedisTransport(w.rdb)

	vs, err := vector.NewQdrantStoreDefault()
	if err != nil {
		return fmt.Errorf("failed to initialize vector store: %w", err)
	}
	w.vectorStore = vs
	defer w.vectorStore.Close()

	mux := asynq.NewServeMux()
	mux.Handle("awe:chat", tasks.NewChatTaskHandler(w.transport, w.vectorStore))

	if err := w.asynqServer.Run(mux); err != nil {
		return err
	}
	return nil
}
