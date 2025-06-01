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

type WorkerConfig struct {
	Workers int

	RedisAddr     string
	RedisUsername string
	RedisPassword string
	RedisDB       int

	QdrantHost string
	QdrantPort int
}

func DefaultConfig() WorkerConfig {
	return WorkerConfig{
		Workers:    10,
		RedisAddr:  "localhost:6379",
		QdrantHost: "localhost",
		QdrantPort: 6334,
	}
}

type Worker struct {
	config WorkerConfig

	rdb         *redis.Client
	asynqServer *asynq.Server

	transport   transport.Transport
	vectorStore vector.Store
}

func New(config WorkerConfig) *Worker {
	return &Worker{
		config: config,
	}
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
		Addr:     w.config.RedisAddr,
		Username: w.config.RedisUsername,
		Password: w.config.RedisPassword,
		DB:       w.config.RedisDB,
	})
	defer w.rdb.Close()

	w.asynqServer = asynq.NewServerFromRedisClient(
		w.rdb,
		asynq.Config{
			Concurrency: w.config.Workers,
		},
	)

	w.transport = transport.NewRedisTransport(w.rdb)

	vs, err := vector.NewQdrantStore(w.config.QdrantHost, w.config.QdrantPort)
	if err != nil {
		return fmt.Errorf("failed to initialize vector store: %w", err)
	}
	w.vectorStore = vs
	defer w.vectorStore.Close()

	handler := tasks.NewTaskHandler(w.transport, w.vectorStore)
	if err := w.asynqServer.Run(handler); err != nil {
		return err
	}
	return nil
}
