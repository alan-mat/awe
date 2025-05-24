package main

import (
	"log"

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

func main() {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // use default Addr
		Password: "",               // no password set
		DB:       0,                // use default DB
	})
	defer rdb.Close()

	srv := asynq.NewServerFromRedisClient(
		rdb,
		asynq.Config{Concurrency: 10},
	)

	transport := transport.NewRedisTransport(rdb)

	vectorStore, err := vector.NewQdrantStoreDefault()
	if err != nil {
		panic(err)
	}
	defer vectorStore.Close()

	wc := config.ReadConfig("workflows.yaml")
	workflows, err := config.ParseWorkflows(wc)
	if err != nil {
		log.Fatalf("failed to parse workflows config: %v\n", err)
	}

	err = registry.BatchRegisterWorkflows(workflows)
	if err != nil {
		log.Fatalf("failed to register workflows: %v\n", err)
	}

	mux := asynq.NewServeMux()
	mux.Handle("awe:chat", tasks.NewChatTaskHandler(transport, vectorStore))

	if err := srv.Run(mux); err != nil {
		log.Fatal(err)
	}
}
