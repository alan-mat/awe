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

func (n *WorkflowNode) Execute(ctx context.Context, params ExecutorParams) ExecutorResult {
	return n.executor.Execute(ctx, params)
}

type Workflow struct {
	identifier  string
	description string

	nodes []WorkflowNode
}

func NewWorkflow(identifier string, description string, nodes []WorkflowNode) *Workflow {
	workflow := &Workflow{
		identifier:  identifier,
		description: description,
		nodes:       nodes,
	}
	return workflow
}

func (w *Workflow) Execute(ctx context.Context, params ExecutorParams) ExecutorResult {
	nodeIdx := 0
	for {
		node := w.nodes[nodeIdx]

		params.Operator = node.operator
		maps.Copy(params.Args, node.args)

		result := node.executor.Execute(ctx, params)
		slog.Debug(fmt.Sprintf("%v\n", result))

		if result.Err != nil {
			return result
		}

		nodeIdx++
		if nodeIdx >= len(w.nodes) {
			break
		}
	}

	return ExecutorResult{
		Name: w.identifier,
		Err:  nil,
	}
}
