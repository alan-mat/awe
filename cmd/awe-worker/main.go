package main

import (
	"log"

	"github.com/alan-mat/awe/internal/tasks"
	"github.com/hibiken/asynq"

	"github.com/redis/go-redis/v9"
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

	mux := asynq.NewServeMux()
	mux.Handle("awe:chat", tasks.NewChatTaskHandler(rdb))

	if err := srv.Run(mux); err != nil {
		log.Fatal(err)
	}
}
