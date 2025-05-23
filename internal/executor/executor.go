package executor

import (
	"context"
	"fmt"
	"maps"
	"reflect"

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

type ErrInvalidArgumentType struct {
	Name     string
	Expected string
	Received string
}

func (e ErrInvalidArgumentType) Error() string {
	return fmt.Sprintf("argument '%s' must be of type '%s', but received '%s'",
		e.Name, e.Expected, e.Received)
}

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
