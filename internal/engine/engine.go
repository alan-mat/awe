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

import (
	"context"
	"maps"

	"github.com/alan-mat/awe/internal/api"
	"github.com/alan-mat/awe/internal/llm"
)

type Executer interface {
	Execute(c Context, p *Params) *Response
}

type ExecuterFunc func(c Context, p *Params) *Response

func (f ExecuterFunc) Execute(c Context, p *Params) *Response {
	return f(c, p)
}

type Middleware func(ExecuterFunc) ExecuterFunc

type Invoker struct {
	context Context
}

func NewInvoker(context Context) *Invoker {
	return &Invoker{
		context: context,
	}
}

func (i *Invoker) Call(e Executer, p *Params) error {
	resp := e.Execute(i.context, p)
	if resp.Err != nil {
		return InvokeError{Cause: resp.Err}
	}

	i.context = i.context.WithState(resp.State)
	return nil
}

type Context struct {
	ctx context.Context

	taskId     string
	workflowId string
	collection string

	originalQuery Query
	caller        CallerMeta

	state State

	values map[string]any
}

func NewContext(
	ctx context.Context,
	taskId string,
	workflowId string,
	collection string,
	query Query,
	caller CallerMeta,
) Context {
	return Context{
		ctx:           ctx,
		taskId:        taskId,
		workflowId:    workflowId,
		collection:    collection,
		originalQuery: query,
		caller:        caller,
		state:         NewState(query),
	}
}

func (c Context) ID() string {
	return c.taskId
}

func (c Context) WorkflowID() string {
	return c.workflowId
}

func (c Context) OriginalQuery() Query {
	return c.originalQuery
}

func (c Context) Caller() CallerMeta {
	return c.caller
}

func (c Context) State() State {
	return c.state
}

func (c Context) Value(key string) any {
	return c.values[key]
}

func (c Context) WithState(state State) Context {
	c.state = state
	return c
}

func (c Context) WithValues(values map[string]any) Context {
	newVals := make(map[string]any)
	maps.Copy(newVals, c.values)
	maps.Copy(newVals, values)

	c.values = newVals
	return c
}

type CallerMeta struct {
	Name string
}

// State holds data related to the workflow execution
// throughout its entire lifecycle. Fields it contains
// are meant to be accessed by each Executer in a workflow.
type State struct {
	Query        Query
	QueryHistory QueryList

	Contents    GeneratedContents
	ContextDocs []*api.ScoredDocument

	MultiQueries *QueryList
	SubQueries   *QueryList
}

func NewState(initialQuery Query) State {
	state := State{}
	state.Query = initialQuery
	state.QueryHistory = make(QueryList, 0)
	state.Contents = GeneratedContents{}

	return state
}

func (s *State) AddToHistory(queries ...Query) {
	s.QueryHistory = append(s.QueryHistory, queries...)
}

func (s *State) AddContents(contents *GeneratedContents) {
	s.Contents.Merge(contents)
}

func (s *State) AddContextDocs(docs ...*api.ScoredDocument) {
	if len(docs) == 0 {
		return
	}
	s.ContextDocs = append(s.ContextDocs, docs...)
}

func (s *State) AddMultiQueries(queries ...Query) {
	if len(queries) == 0 {
		return
	}

	if s.MultiQueries == nil {
		list := QueryList(queries)
		s.MultiQueries = &list
	} else {
		*s.MultiQueries = append(*s.MultiQueries, queries...)
	}
}

func (s *State) AddSubQueries(queries ...Query) {
	if len(queries) == 0 {
		return
	}

	if s.SubQueries == nil {
		list := QueryList(queries)
		s.SubQueries = &list
	} else {
		*s.SubQueries = append(*s.SubQueries, queries...)
	}
}

func (s *State) Merge(states ...*State) {
	for _, state := range states {
		s.QueryHistory = append(s.QueryHistory, state.QueryHistory...)
		s.Contents.Merge(&state.Contents)
		s.ContextDocs = append(s.ContextDocs, state.ContextDocs...)

		if s.MultiQueries == nil && state.MultiQueries != nil {
			s.MultiQueries = state.MultiQueries
		} else if s.MultiQueries != nil && state.MultiQueries != nil {
			*s.MultiQueries = append(*s.MultiQueries, *state.MultiQueries...)
		}

		if s.SubQueries == nil && state.SubQueries != nil {
			s.SubQueries = state.SubQueries
		} else if s.SubQueries != nil && state.SubQueries != nil {
			*s.SubQueries = append(*s.SubQueries, *state.SubQueries...)
		}
	}
}

type GeneratedContents struct {
	Messages []llm.Message
}

func ContentsFromMessages(messages ...llm.Message) *GeneratedContents {
	contents := &GeneratedContents{Messages: messages}
	return contents
}

func (c *GeneratedContents) Merge(contents ...*GeneratedContents) {
	for _, content := range contents {
		c.Messages = append(c.Messages, content.Messages...)
	}
}

type Query struct {
	Text string
}

func TextQuery(text string) Query {
	return Query{
		Text: text,
	}
}

type QueryList []Query

// Params contain data limited in scope to the currently
// invoked executer only.
type Params struct {
	Args Arguments

	Children []*WorkflowNode
	Routes   []*WorkflowRoute
	Branches []*WorkflowBranch
}

func (p Params) SetArgument(key string, val any) {
	if p.Args == nil {
		p.Args = make(Arguments)
	}
	p.Args[key] = val
}

func DefaultParams() *Params {
	return &Params{
		Args: make(Arguments),
	}
}

type Arguments map[string]any

func GetTypedArgument[T any](args Arguments, name string) (T, bool) {
	arg, ok := args[name]
	if !ok {
		return *new(T), false
	}

	typedArg, ok := arg.(T)
	return typedArg, ok
}

func GetTypedArgumentWithDefault[T any](args Arguments, name string, defaultValue T) T {
	arg, ok := args[name]
	if !ok {
		return defaultValue
	}

	typedArg, ok := arg.(T)
	if !ok {
		return defaultValue
	}

	return typedArg
}

// Response contains the execution response from an [Executor].
type Response struct {
	// Err holds errors that may occur during execution.
	// If this is not-nil, the value of State may not be trusted.
	Err error

	// State contains the new state post-execution, desired by the Executor.
	// If Err is nil, this must hold a valid not-nil value.
	// Executers are responsible for returning valid new state,
	// according to its execution results. Corrupted state may result in
	// failed invocations of furhter Executors.
	State State
}
