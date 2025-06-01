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

package executor

import (
	"context"
	"log/slog"
	"maps"
	"reflect"

	"github.com/alan-mat/awe/internal/api"
	"github.com/alan-mat/awe/internal/transport"
	"github.com/alan-mat/awe/internal/vector"
)

type Executor interface {
	Execute(ctx context.Context, params *ExecutorParams) *ExecutorResult
}

type ExecutorParams struct {
	taskID string
	query  string

	Operator    string
	Transport   transport.Transport
	VectorStore vector.Store
	Args        map[string]any

	Children []*WorkflowNode
	Routes   []*WorkflowRoute
	Branches []*WorkflowBranch
}

type ExecutorParamOption func(*ExecutorParams)

func NewExecutorParams(id string, query string, options ...ExecutorParamOption) *ExecutorParams {
	ep := &ExecutorParams{
		taskID:   id,
		query:    query,
		Operator: "",
		Args:     make(map[string]any),
	}
	for _, opt := range options {
		opt(ep)
	}
	return ep
}

func (p *ExecutorParams) SetChildren(children []*WorkflowNode) {
	p.Children = children
}

func (p *ExecutorParams) SetRoutes(routes []*WorkflowRoute) {
	p.Routes = routes
}

func (p *ExecutorParams) SetBranches(branches []*WorkflowBranch) {
	p.Branches = branches
}

func (p *ExecutorParams) SetQuery(q string) {
	p.query = q
}

func (p ExecutorParams) GetTaskID() string {
	return p.taskID
}

func (p ExecutorParams) GetQuery() string {
	return p.query
}

func (p ExecutorParams) WithQuery(q string) *ExecutorParams {
	newArgs := make(map[string]any)
	maps.Copy(newArgs, p.Args)

	newParams := &ExecutorParams{
		query: q,

		taskID:      p.taskID,
		Operator:    p.Operator,
		Transport:   p.Transport,
		VectorStore: p.VectorStore,
		Args:        newArgs,
	}

	return newParams
}

// Copy creates a copy of the ExecutorParams object
// The returned copy excludes the following fields:
//
//	Children, Routes, Branches
func (p ExecutorParams) Copy() *ExecutorParams {
	newArgs := make(map[string]any)
	maps.Copy(newArgs, p.Args)

	return &ExecutorParams{
		query:       p.query,
		taskID:      p.taskID,
		Operator:    p.Operator,
		Transport:   p.Transport,
		VectorStore: p.VectorStore,
		Args:        newArgs,
	}
}

func (p ExecutorParams) GetArg(argName string) (any, error) {
	arg, ok := p.Args[argName]
	if !ok {
		return nil, ErrArgMissing{ArgName: argName}
	}
	return arg, nil
}

func WithOperator(op string) ExecutorParamOption {
	return func(ep *ExecutorParams) {
		ep.Operator = op
	}
}

func WithTransport(t transport.Transport) ExecutorParamOption {
	return func(ep *ExecutorParams) {
		ep.Transport = t
	}
}

func WithVectorStore(vs vector.Store) ExecutorParamOption {
	return func(ep *ExecutorParams) {
		ep.VectorStore = vs
	}
}

func WithArgs(args map[string]any) ExecutorParamOption {
	return func(ep *ExecutorParams) {
		ep.Args = args
	}
}

type ExecutorResult struct {
	Name     string
	Operator string
	Err      error
	Values   map[string]any
}

func (res *ExecutorResult) Get(valueName string) (any, bool) {
	val, ok := res.Values[valueName]
	if !ok {
		return nil, false
	}
	return val, true
}

func ProcessResult(params *ExecutorParams, result *ExecutorResult) *ExecutorParams {
	newParams := params.Copy()

	if query_transformed, ok := result.Values["query_transformed"].(string); ok {
		// node executor returned a new transformed query
		// set it as new query in params
		newParams.SetQuery(query_transformed)
	}

	if new_context, ok := result.Values["context_docs"].([]*api.ScoredDocument); ok {
		// check if the context should be replaced
		if replace, ok := result.Values["replace_context"].(bool); ok {
			if replace {
				newParams.Args["context_docs"] = new_context
			}
		} else {
			// otherwise append
			context, ok := newParams.Args["context_docs"]
			if !ok {
				// no context_docs yet, create
				newParams.Args["context_docs"] = new_context
			} else {
				context_typed, ok := context.([]*api.ScoredDocument)
				if !ok {
					slog.Error("workflow error", "msg", "invalid type of context docs in params")
				}
				newParams.Args["context_docs"] = append(context_typed, new_context...)
			}
		}
	}

	return newParams
}

func GetTypedArg[T any](p *ExecutorParams, argName string) (T, error) {
	arg, err := p.GetArg(argName)
	if err != nil {
		return *new(T), err
	}

	typedArg, ok := arg.(T)
	if !ok {
		expectedType := reflect.TypeOf((*T)(nil)).Elem()
		receivedType := reflect.TypeOf(arg)

		return *new(T), ErrInvalidArgumentType{
			Name:     argName,
			Expected: expectedType.String(),
			Received: receivedType.String(),
		}
	}

	return typedArg, nil
}

func GetTypedResult[T any](res *ExecutorResult, argName string) (T, bool) {
	arg, ok := res.Get(argName)
	if !ok {
		return *new(T), false
	}

	typedArg, ok := arg.(T)
	return typedArg, ok
}
