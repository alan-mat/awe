package system

import (
	"context"
	"log/slog"

	"github.com/alan-mat/awe/internal/executor"
	"github.com/alan-mat/awe/internal/registry"
)

var loggerExecutorDescriptor = "system.Logger"

func init() {
	e := NewLoggerExecutor()
	err := registry.RegisterExecutor(loggerExecutorDescriptor, e)
	if err != nil {
		slog.Error("failed to register executor", "name", loggerExecutorDescriptor)
	}
}

type LoggerExecutor struct {
	operators map[string]func(context.Context, *executor.ExecutorParams) (map[string]any, error)
}

func NewLoggerExecutor() *LoggerExecutor {
	e := &LoggerExecutor{}
	e.operators = map[string]func(context.Context, *executor.ExecutorParams) (map[string]any, error){
		"acc_stream": e.accStream,
	}
	return e
}

func (e *LoggerExecutor) Execute(ctx context.Context, p *executor.ExecutorParams) executor.ExecutorResult {
	if p.Operator == "" {
		p.Operator = "acc_stream"
	}
	slog.Info("executing", "name", loggerExecutorDescriptor, "op", p.Operator, "query", p.GetQuery(), "id", p.GetTaskID())

	opFunc, exists := e.operators[p.Operator]
	if !exists {
		return e.buildResult(p.Operator, executor.ErrOperatorNotFound{
			ExecutorName: loggerExecutorDescriptor, OperatorName: p.Operator}, nil)
	}

	vals, err := opFunc(ctx, p)
	if err == nil {
		slog.Info("logger results", "values", vals)
	}

	return e.buildResult(p.Operator, err, vals)
}

func (e *LoggerExecutor) accStream(ctx context.Context, p *executor.ExecutorParams) (map[string]any, error) {
	ms, err := p.Transport.GetMessageStream(p.GetTaskID())
	if err != nil {
		slog.Warn("failed to create message stream", "id", p.GetTaskID())
		return nil, err
	}

	slog.Debug("accStream got args", "args", p.Args)

	text, err := ms.Text(ctx)
	if err != nil {
		return nil, err
	}

	return map[string]any{"content": text}, nil
}

func (e *LoggerExecutor) buildResult(operator string, err error, values map[string]any) executor.ExecutorResult {
	return executor.ExecutorResult{
		Name:     loggerExecutorDescriptor,
		Operator: operator,
		Err:      err,
		Values:   values,
	}
}
