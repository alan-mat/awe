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

import (
	"errors"
	"fmt"
	"os"

	"github.com/alan-mat/awe/internal/executor"
	"github.com/alan-mat/awe/internal/registry"
	"github.com/goccy/go-yaml"
)

var (
	ErrInvalidNodeType      = errors.New("invalid node type")
	ErrIncompatibleNodeType = errors.New("incompatible node type")
	ErrNodeMissingChildren  = errors.New("node must contain at least one child node")
	ErrInvalidExecutor      = errors.New("invalid executor")
)

func ReadConfig(path string) WorkflowConfig {
	file, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	var wc WorkflowConfig
	if err := yaml.Unmarshal(file, &wc); err != nil {
		panic(err)
	}

	return wc
}

func ParseWorkflows(conf WorkflowConfig) (map[string]*executor.Workflow, error) {
	workflows := make(map[string]*executor.Workflow)

	for _, cw := range conf.Workflows {
		nodes, err := parseWorkflowNodes(cw.Nodes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse node on '%s' workflow (%v)", cw.Identifier, err)
		}

		var collectionName string
		if cw.CollectionName == "" {
			collectionName = "default"
		} else {
			collectionName = cw.CollectionName
		}

		workflow := executor.NewWorkflow(
			cw.Identifier,
			cw.Description,
			collectionName,
			cw.Search,
			nodes,
		)

		workflows[cw.Identifier] = workflow
	}

	return workflows, nil
}

func parseWorkflowNodes(nodes []WorkflowNode) ([]*executor.WorkflowNode, error) {
	if len(nodes) == 0 {
		return nil, ErrNodeMissingChildren
	}

	execNodes := make([]*executor.WorkflowNode, 0, len(nodes))
	for _, cnode := range nodes {
		exec, err := registry.GetExecutor(cnode.Module)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrInvalidExecutor, err)
		}

		if cnode.Type == "" {
			cnode.Type = NodeTypeLinear
		}

		wfNode := executor.NewWorkflowNode(exec, cnode.Operator, string(cnode.Type))
		if len(cnode.Args) > 0 {
			wfNode.Args = cnode.Args
		}

		switch cnode.Type {
		case NodeTypeLoop:
			if len(cnode.Nodes) == 0 {
				return nil, ErrNodeMissingChildren
			}

			children, err := parseWorkflowNodes(cnode.Nodes)
			if err != nil {
				return nil, err
			}
			wfNode.Children = children

		case NodeTypeConditional:
			if len(cnode.Routes) == 0 {
				return nil, ErrNodeMissingChildren
			}

			routes := make([]*executor.WorkflowRoute, 0, len(cnode.Routes))
			for _, r := range cnode.Routes {
				children, err := parseWorkflowNodes(r.Nodes)
				if err != nil {
					return nil, err
				}

				routes = append(routes, &executor.WorkflowRoute{
					Key:         r.Key,
					Description: r.Description,
					Nodes:       children,
				})
			}
			wfNode.Routes = routes

		case NodeTypeBranching:
			if len(cnode.Branches) == 0 {
				return nil, ErrNodeMissingChildren
			}

			branches := make([]*executor.WorkflowBranch, 0, len(cnode.Branches))
			for _, b := range cnode.Branches {
				children, err := parseWorkflowNodes(b.Nodes)
				if err != nil {
					return nil, err
				}

				branches = append(branches, &executor.WorkflowBranch{
					Name:  b.Name,
					Nodes: children,
				})
			}
			wfNode.Branches = branches

		case NodeTypeLinear:
			if len(cnode.Nodes) > 0 || len(cnode.Routes) > 0 || len(cnode.Branches) > 0 {
				return nil, ErrIncompatibleNodeType
			}

		default:
			return nil, ErrInvalidNodeType
		}

		execNodes = append(execNodes, wfNode)
	}

	return execNodes, nil
}
