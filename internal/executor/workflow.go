package executor

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
)

type WorkflowNode struct {
	executor Executor
	operator string
	args     map[string]any
}

func NewWorkflowNode(executor Executor, operator string, args map[string]any) WorkflowNode {
	node := WorkflowNode{
		executor: executor,
		operator: operator,
		args:     args,
	}
	return node
}

func (n *WorkflowNode) Execute(ctx context.Context, params *ExecutorParams) *ExecutorResult {
	return n.executor.Execute(ctx, params)
}

type Workflow struct {
	identifier     string
	description    string
	collectionName string

	nodes []WorkflowNode
}

func NewWorkflow(identifier string, description string, collectionName string, nodes []WorkflowNode) *Workflow {
	workflow := &Workflow{
		identifier:     identifier,
		description:    description,
		collectionName: collectionName,
		nodes:          nodes,
	}
	return workflow
}

func (w *Workflow) Execute(ctx context.Context, params *ExecutorParams) *ExecutorResult {
	nodeIdx := 0
	params.Args["collection_name"] = w.collectionName

	slog.Info("executing workflow", "workflowId", w.identifier, "params", params)

	for {
		node := w.nodes[nodeIdx]

		params.Operator = node.operator
		maps.Copy(params.Args, node.args)

		slog.Info("executing node", "executor", node.executor, "op", node.operator, "args", params.Args)

		result := node.executor.Execute(ctx, params)
		slog.Debug(fmt.Sprintf("%v\n", result))

		if result.Err != nil {
			slog.Error("failed to execute node", "error", fmt.Sprintf("(%T): %v", result.Err, result.Err))
			return result
		}

		if query_transformed, ok := result.Values["query_transformed"].(string); ok {
			// node executor returned a new transformed query
			// set it as new query in params
			params = params.WithQuery(query_transformed)
		}

		nodeIdx++
		if nodeIdx >= len(w.nodes) {
			break
		}

		maps.Copy(params.Args, result.Values)
	}

	return &ExecutorResult{
		Name: w.identifier,
		Err:  nil,
	}
}
