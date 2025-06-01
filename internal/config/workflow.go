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

package config

const (
	NodeTypeLinear      = "default"
	NodeTypeLoop        = "loop"
	NodeTypeConditional = "conditional"
	NodeTypeBranching   = "branching"
)

type WorkflowNodeType string

/* var workflowNodeTypeMap = map[string]WorkflowNodeType{
	"default":     NodeTypeLinear,
	"loop":        NodeTypeLoop,
	"conditional": NodeTypeConditional,
	"branching":   NodeTypeBranching,
} */

type WorkflowNode struct {
	Module   string           `yaml:"module"`
	Operator string           `yaml:"operator"`
	Type     WorkflowNodeType `yaml:"type"`
	Args     map[string]any   `yaml:"args"`

	Nodes    []WorkflowNode   `yaml:"nodes"`
	Routes   []WorkflowRoute  `yaml:"routes"`
	Branches []WorkflowBranch `yaml:"branches"`
}

type WorkflowRoute struct {
	Key         string         `yaml:"key"`
	Description string         `yaml:"description"`
	Nodes       []WorkflowNode `yaml:"nodes"`
}

type WorkflowBranch struct {
	Name  string         `yaml:"name"`
	Nodes []WorkflowNode `yaml:"nodes"`
}

type Workflow struct {
	Identifier     string `yaml:"name"`
	Description    string `yaml:"description"`
	CollectionName string `yaml:"collection"`
	Search         bool   `yaml:"search"`

	Nodes []WorkflowNode `yaml:"nodes"`
}

type WorkflowConfig struct {
	Workflows map[string]Workflow `yaml:"workflows"`
}
