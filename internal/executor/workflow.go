package executor

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
)

type WorkflowNode struct {
	Executor Executor
	Operator string
	NodeType string
	Args     map[string]any

	Children []*WorkflowNode
	Routes   []*WorkflowRoute
	Branches []*WorkflowBranch
}

type WorkflowRoute struct {
	Key         string
	Description string
	Nodes       []*WorkflowNode
}

type WorkflowBranch struct {
	Name  string
	Nodes []*WorkflowNode
}

func NewWorkflowNode(executor Executor, operator string, nodeType string) *WorkflowNode {
	node := &WorkflowNode{
		Executor: executor,
		Operator: operator,
		NodeType: nodeType,
	}
	return node
}

func (n WorkflowNode) Execute(ctx context.Context, params *ExecutorParams) *ExecutorResult {
	return n.Executor.Execute(ctx, params)
}

type Workflow struct {
	identifier     string
	description    string
	collectionName string

	nodes []*WorkflowNode
}

func NewWorkflow(identifier string, description string, collectionName string, nodes []*WorkflowNode) *Workflow {
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
		nodeParams := MakeNodeParams(node, params)

		result := node.Executor.Execute(ctx, nodeParams)
		// slog.Info(fmt.Sprintf("%v\n", result))

		if result.Err != nil {
			slog.Error("failed to execute node", "error", fmt.Sprintf("(%T): %v", result.Err, result.Err))
			return result
		}

		nodeIdx++
		if nodeIdx >= len(w.nodes) {
			break
		}

		params = ProcessResult(params, result)
	}

	return &ExecutorResult{
		Name: w.identifier,
		Err:  nil,
	}
}

func MakeNodeParams(node *WorkflowNode, sourceParams *ExecutorParams) *ExecutorParams {
	nodeParams := sourceParams.Copy()

	nodeParams.Operator = node.Operator
	nodeParams.SetChildren(node.Children)
	nodeParams.SetRoutes(node.Routes)
	nodeParams.SetBranches(node.Branches)

	maps.Copy(nodeParams.Args, node.Args)

	return nodeParams
}
