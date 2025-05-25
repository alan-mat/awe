package executor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"

	"github.com/alan-mat/awe/internal/api"
	"github.com/alan-mat/awe/internal/transport"
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
	search         bool

	nodes []*WorkflowNode
}

func NewWorkflow(
	identifier string,
	description string,
	collectionName string,
	search bool,
	nodes []*WorkflowNode,
) *Workflow {
	workflow := &Workflow{
		identifier:     identifier,
		description:    description,
		collectionName: collectionName,
		search:         search,
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

		params = ProcessResult(params, result)

		nodeIdx++
		if nodeIdx >= len(nodes) {
			break
		}
	}

	var err error = nil
	// if search-only workflow, stream context_docs
	if w.search {
		err = w.sendContextDocs(ctx, params)
	}

	return &ExecutorResult{
		Name:   w.identifier,
		Err:    err,
		Values: params.Args,
	}
}

func (w Workflow) sendContextDocs(ctx context.Context, params *ExecutorParams) error {
	docs, ok := params.Args["context_docs"].([]*api.ScoredDocument)
	if !ok {
		// no docs found during search
		return errors.New("no search results found")
	}

	ms, err := params.Transport.GetMessageStream(params.taskID)
	if err != nil {
		return err
	}

	for i, doc := range docs {
		payload := transport.MessageStreamPayload{
			ID:     i,
			Status: "OK",
			Type:   transport.MessageTypeDocument,
			Document: transport.Document{
				Title:   doc.Title,
				Content: doc.Content,
				Source:  "",
			},
		}

		err := ms.Send(ctx, payload)

		if err != nil {
			return errors.New("failed to send to message stream")
		}
	}
	return nil
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
