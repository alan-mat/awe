package executor

import (
	"context"
	"fmt"
	"log/slog"
	"maps"

	"github.com/alan-mat/awe/internal/api"
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
		node_params := params.Copy()
		node_params.Operator = node.operator
		maps.Copy(node_params.Args, node.args)

		result := node.executor.Execute(ctx, node_params)
		// slog.Info(fmt.Sprintf("%v\n", result))

		if result.Err != nil {
			slog.Error("failed to execute node", "error", fmt.Sprintf("(%T): %v", result.Err, result.Err))
			return result
		}

		nodeIdx++
		if nodeIdx >= len(w.nodes) {
			break
		}

		if query_transformed, ok := result.Values["query_transformed"].(string); ok {
			// node executor returned a new transformed query
			// set it as new query in params
			params = params.WithQuery(query_transformed)
		}

		if new_context, ok := result.Values["context_docs"].([]*api.ScoredDocument); ok {
			// check if the context should be replaced
			if replace, ok := result.Values["replace_context"].(bool); ok {
				if replace {
					params.Args["context_docs"] = new_context
				}
			} else {
				// otherwise append
				context, ok := params.Args["context_docs"]
				if !ok {
					// no context_docs yet, create
					params.Args["context_docs"] = new_context
				} else {
					context_typed, ok := context.([]*api.ScoredDocument)
					if !ok {
						slog.Error("workflow error", "msg", "invalid type of context docs in params")
					}
					params.Args["context_docs"] = append(context_typed, new_context...)
				}
			}
		}
	}

	return &ExecutorResult{
		Name: w.identifier,
		Err:  nil,
	}
}
