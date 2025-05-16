package main

import (
	"fmt"
	"log"

	"github.com/alan-mat/awe/internal/config"
	"github.com/alan-mat/awe/internal/executor"
	"github.com/alan-mat/awe/internal/registry"
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
	workflows, err := initWorkflows(wc)
	if err != nil {
		panic(err)
	}

	err = registerWorkflows(workflows)
	if err != nil {
		panic(err)
	}

	mux := asynq.NewServeMux()
	mux.Handle("awe:chat", tasks.NewChatTaskHandler(transport))

	if err := srv.Run(mux); err != nil {
		log.Fatal(err)
	}
}

func initWorkflows(conf config.WorkflowConfig) (map[string]*executor.Workflow, error) {
	workflows := make(map[string]*executor.Workflow)

	for _, cw := range conf.Workflows {
		nodes := make([]executor.WorkflowNode, 0, len(cw.Nodes))
		for _, cnode := range cw.Nodes {
			exec, err := registry.GetExecutor(cnode.Module)
			if err != nil {
				return nil, err
			}

			nodes = append(nodes, executor.NewWorkflowNode(exec, cnode.Operator, cnode.Args))
		}

		workflow := executor.NewWorkflow(cw.Identifier, cw.Description, nodes)
		workflows[cw.Identifier] = workflow
	}

	return workflows, nil
}

func registerWorkflows(workflows map[string]*executor.Workflow) error {
	for name, wf := range workflows {
		err := registry.RegisterWorkflow(name, wf)
		if err != nil {
			return err
		}
	}

	fmt.Println("registered workflows: ", registry.ListWorkflows())

	return nil
}
