package executor

import (
	"context"
	"fmt"

	"github.com/alan-mat/awe/internal/transport"
)

type ErrOperatorNotFound struct {
	ExecutorName string
	OperatorName string
}

func (e ErrOperatorNotFound) Error() string {
	return fmt.Sprintf("invalid operator '%s' for executor '%s'", e.ExecutorName, e.OperatorName)
}

type ErrInvalidArguments struct {
	ExecutorName string
	OperatorName string
	Accepts      string
	Given        []any
}

func (e ErrInvalidArguments) Error() string {
	return fmt.Sprintf("invalid arguments for operator '%s' in executor '%s': accepts '%s', got '%v'",
		e.OperatorName, e.ExecutorName, e.Accepts, e.Given)
}

type Executor interface {
	Execute(ctx context.Context, params ExecutorParams) ExecutorResult
}

type ExecutorParams struct {
	taskID string
	query  string

	Operator  string
	Transport transport.Transport
	Args      map[string]any
}

type ExecutorParamOption func(*ExecutorParams)

func NewExecutorParams(id string, query string, options ...ExecutorParamOption) *ExecutorParams {
	ep := &ExecutorParams{
		taskID:   id,
		query:    query,
		Operator: "",
		Args:     nil,
	}
	for _, opt := range options {
		opt(ep)
	}
	return ep
}

func (p ExecutorParams) GetTaskID() string {
	return p.taskID
}

func (p ExecutorParams) GetQuery() string {
	return p.query
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
