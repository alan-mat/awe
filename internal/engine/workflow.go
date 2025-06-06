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

type WorkflowNode struct {
	Module   string
	Operator string

	NodeType string
	Args     map[string]any

	Children []*WorkflowNode
	Routes   []*WorkflowRoute
	Branches []*WorkflowBranch
}

func NewWorkflowNode(module string, operator string, nodeType string) *WorkflowNode {
	node := &WorkflowNode{
		Module:   module,
		Operator: operator,
		NodeType: nodeType,
	}
	return node
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
