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

package engine

type WorkflowNodeType string

const (
	WorkflowNodeLinear      WorkflowNodeType = "linear"
	WorkflowNodeLoop        WorkflowNodeType = "loop"
	WorkflowNodeConditional WorkflowNodeType = "conditional"
	WorkflowNodeBranching   WorkflowNodeType = "branching"
)

type WorkflowNode struct {
	Module       Module
	OperatorName string

	NodeType WorkflowNodeType
	Args     Arguments

	Children []*WorkflowNode
	Routes   []*WorkflowRoute
	Branches []*WorkflowBranch
}

func NewWorkflowNode(module Module, operatorName string, nodeType WorkflowNodeType) *WorkflowNode {
	node := &WorkflowNode{
		Module:       module,
		OperatorName: operatorName,
		NodeType:     nodeType,
		Args:         make(Arguments),
	}
	return node
}

func (node WorkflowNode) SetArgument(key string, val any) {
	node.Args[key] = val
}

func (node *WorkflowNode) AddChildren(nodes ...*WorkflowNode) {
	if node.Children == nil {
		node.Children = make([]*WorkflowNode, 0, len(nodes))
	}
	node.Children = append(node.Children, nodes...)
}

func (node *WorkflowNode) AddRoutes(routes ...*WorkflowRoute) {
	if node.Routes == nil {
		node.Routes = make([]*WorkflowRoute, 0, len(routes))
	}
	node.Routes = append(node.Routes, routes...)
}

func (node *WorkflowNode) AddBranches(branches ...*WorkflowBranch) {
	if node.Branches == nil {
		node.Branches = make([]*WorkflowBranch, 0, len(branches))
	}
	node.Branches = append(node.Branches, branches...)
}

// Route retrieves a WorkflowRoute from the WorkflowNode based on the route key.
// If no route was found for the given key, the return value will be nil.
func (node WorkflowNode) Route(key string) *WorkflowRoute {
	var found *WorkflowRoute
	for _, route := range node.Routes {
		if route.Key == key {
			found = route
			break
		}
	}
	return found
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

type Workflow struct {
	identifier     string
	description    string
	collectionName string

	nodes []*WorkflowNode
}

func NewWorkflow(
	identifier string,
	description string,
	collectionName string,
	nodes []*WorkflowNode,
) *Workflow {
	workflow := &Workflow{
		identifier:     identifier,
		description:    description,
		collectionName: collectionName,
		nodes:          nodes,
	}
	return workflow
}

func (w Workflow) Executer(t Transport) ExecuterFunc {
	return func(c Context, p *Params) *Response {
		inv := NewInvoker(c)
		if t != nil {
			inv.Use(TransportGenerationMiddleware(t, c.TaskId()))
		}

		nodes := w.nodes
		nodeIdx := 0

		for {
			node := nodes[nodeIdx]
			nodeParams := &Params{
				Args:     node.Args,
				Children: node.Children,
				Branches: node.Branches,
				Routes:   node.Routes,
			}

			op, err := node.Module.Operator(node.OperatorName)
			if err != nil {
				return ErrorResponse(inv.State(), err)
			}

			err = inv.Call(op, nodeParams)
			if err != nil {
				return ErrorResponse(inv.State(), err)
			}

			// workflow nodes with conditional type
			// must set a next route
			if node.NodeType == WorkflowNodeConditional {
				route, err := parseNextRoute(inv, node)
				if err != nil {
					return ErrorResponse(inv.State(), err)
				}

				nodes = route.Nodes
				nodeIdx = 0
				continue
			}

			// if next workflow has been set, execute it
			sub := inv.NextWorkflow()
			if sub != nil {
				err = inv.Call(sub.Executer(t), nodeParams)
				if err != nil {
					return ErrorResponse(inv.State(), err)
				}
			}

			nodeIdx += 1
			if nodeIdx >= len(nodes) {
				break
			}
		}

		return &Response{State: inv.State()}
	}
}

func parseNextRoute(inv *Invoker, node *WorkflowNode) (*WorkflowRoute, error) {
	key := inv.NextRoute()
	if key == nil {
		return nil, ErrNextRouteMissing
	}

	route := node.Route(*key)
	if route == nil {
		return nil, ErrNodeRouteNotFound
	}

	return route, nil
}
