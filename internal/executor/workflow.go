package executor

import (
	"context"
	"errors"
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

func (w Workflow) Execute(ctx context.Context, params *ExecutorParams) *ExecutorResult {
	params.Args["collection_name"] = w.collectionName
	nodes := w.nodes
	nodeIdx := 0

	slog.Info("executing workflow", "workflowId", w.identifier, "params", params)

	for {
		node := nodes[nodeIdx]
		nodeParams := MakeNodeParams(node, params)

		result := node.Executor.Execute(ctx, nodeParams)
		// slog.Info(fmt.Sprintf("%v\n", result))

		if result.Err != nil {
			slog.Error("failed to execute node", "error", fmt.Sprintf("(%T): %v", result.Err, result.Err))
			return result
		}

		if node.NodeType == "conditional" {
			// conditional nodes MUST return a route_key
			if routeKey, ok := result.Values["route_key"].(string); ok {
				// set nodes to route nodes
				// and reset nodeIdx
				var nextRoute *WorkflowRoute
				for _, r := range node.Routes {
					if r.Key == routeKey {
						nextRoute = r
						break
					}
				}

				if nextRoute == nil {
					// invalid route key
					slog.Error("failed to execute workflow: no route found for given key")
					return &ExecutorResult{
						Name: w.identifier,
						Err:  errors.New("failed to execute workflow: no route found for given key"),
					}
				}

				nodes = nextRoute.Nodes
				nodeIdx = 0
				continue
			} else {
				slog.Error("failed to execute workflow: conditional type node did not return a route key")
				return &ExecutorResult{
					Name: w.identifier,
					Err:  errors.New("failed to execute workflow: conditional type node did not return a route key"),
				}
			}
		}

		nodeIdx++
		if nodeIdx >= len(nodes) {
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
