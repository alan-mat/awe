package engine

import "fmt"

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
	Execute(operator string, args ...any) ExecutorResult
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
