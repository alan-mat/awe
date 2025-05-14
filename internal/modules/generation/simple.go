package generation

import (
	"log/slog"

	"github.com/alan-mat/awe/internal/executor"
	"github.com/alan-mat/awe/internal/provider"
	"github.com/alan-mat/awe/internal/registry"
)

var simpleExecutorDescriptor = "generation.Simple"

func init() {
	e := NewSimpleExecutor()
	err := registry.RegisterExecutor(simpleExecutorDescriptor, e)
	if err != nil {
		slog.Error("failed to register executor", "name", simpleExecutorDescriptor)
	}
}

type SimpleExecutor struct {
	Provider  provider.LMProviderType
	operators map[string]func(...any) error
}

func NewSimpleExecutor() *SimpleExecutor {
	e := &SimpleExecutor{
		Provider: provider.LMProviderTypeGemini,
	}
	e.operators = map[string]func(...any) error{
		"generate": e.generate,
	}
	return e
}

func (e *SimpleExecutor) Execute(operator string, args ...any) executor.ExecutorResult {
	if operator == "" {
		operator = "generate"
	}
	slog.Info("executing", "name", simpleExecutorDescriptor, "op", operator, "args", args)

	opFunc, exists := e.operators[operator]
	if !exists {
		return e.buildResult(operator, executor.ErrOperatorNotFound{
			ExecutorName: simpleExecutorDescriptor, OperatorName: operator}, nil)
	}

	err := opFunc(args...)
	return e.buildResult(operator, err, nil)
}

func (e *SimpleExecutor) generate(...any) error {
	slog.Info("I AM GENERATING")
	slog.Info("SOME TEXT !!!")
	return nil
}

func (e *SimpleExecutor) buildResult(operator string, err error, values map[string]any) executor.ExecutorResult {
	return executor.ExecutorResult{
		Name:     simpleExecutorDescriptor,
		Operator: operator,
		Err:      err,
		Values:   values,
	}
}
