package main

import (
	"fmt"
	"log"

	"github.com/alan-mat/awe/internal/config"
	"github.com/alan-mat/awe/internal/tasks"
	"github.com/alan-mat/awe/internal/transport"
	"github.com/hibiken/asynq"

	"github.com/redis/go-redis/v9"

	_ "github.com/alan-mat/awe/internal/modules/generation"
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

	wc := config.ParseWorkflowConfig("workflows.yaml")
	fmt.Printf("%v\n", wc)

	mux := asynq.NewServeMux()
	mux.Handle("awe:chat", tasks.NewChatTaskHandler(transport))

	if err := srv.Run(mux); err != nil {
		log.Fatal(err)
	}
}
