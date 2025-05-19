package executor

import (
	"context"
	"fmt"

	"github.com/alan-mat/awe/internal/transport"
	"github.com/alan-mat/awe/internal/vector"
)

type ErrOperatorNotFound struct {
	ExecutorName string
	OperatorName string
}

func (e ErrOperatorNotFound) Error() string {
	return fmt.Sprintf("invalid operator '%s' for executor '%s'", e.ExecutorName, e.OperatorName)
}

type ErrArgMissing struct {
	ArgName string
}

func (e ErrArgMissing) Error() string {
	return fmt.Sprintf("requested argument '%s' does not exist", e.ArgName)
}

type Executor interface {
	Execute(ctx context.Context, params *ExecutorParams) ExecutorResult
}

type ExecutorParams struct {
	taskID string
	query  string

	Operator    string
	Transport   transport.Transport
	VectorStore vector.Store
	Args        map[string]any
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

func (p ExecutorParams) GetTaskID() string {
	return p.taskID
}

func (p ExecutorParams) GetQuery() string {
	return p.query
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
