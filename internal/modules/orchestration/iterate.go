package orchestration

import (
	"context"
	"fmt"
	"log/slog"
	"math"

	"github.com/alan-mat/awe/internal/executor"
	"github.com/alan-mat/awe/internal/registry"
)

var iterateExecutorDescriptor = "orchestration.Iterate"

func init() {
	exec, err := NewIterateExecutor()
	if err != nil {
		slog.Error("failed to initialize executor", "name", iterateExecutorDescriptor, "err", err)
		return
	}

	err = registry.RegisterExecutor(iterateExecutorDescriptor, exec)
	if err != nil {
		slog.Error("failed to register executor", "name", iterateExecutorDescriptor)
	}
}

type IterateExecutor struct {
	operators map[string]func(context.Context, *executor.ExecutorParams) (map[string]any, error)
}

func NewIterateExecutor() (*IterateExecutor, error) {
	e := &IterateExecutor{}
	e.operators = map[string]func(context.Context, *executor.ExecutorParams) (map[string]any, error){
		"fixed_iters": e.fixedIterations,
	}
	return e, nil
}

func (e IterateExecutor) Execute(ctx context.Context, p *executor.ExecutorParams) *executor.ExecutorResult {
	if p.Operator == "" {
		p.Operator = "fixed_iters"
	}
	slog.Info("executing", "name", iterateExecutorDescriptor, "op", p.Operator, "query", p.GetQuery(), "id", p.GetTaskID())

	opFunc, exists := e.operators[p.Operator]
	if !exists {
		return &executor.ExecutorResult{
			Name:     iterateExecutorDescriptor,
			Operator: p.Operator,
			Err: executor.ErrOperatorNotFound{
				ExecutorName: iterateExecutorDescriptor,
				OperatorName: p.Operator,
			},
			Values: nil,
		}
	}

	if len(p.Children) == 0 {
		return &executor.ExecutorResult{
			Name:     iterateExecutorDescriptor,
			Operator: p.Operator,
			Err: executor.ErrInvalidParams{
				ExecutorName:  iterateExecutorDescriptor,
				MissingParams: []string{"children"},
			},
			Values: nil,
		}
	}

	vals, err := opFunc(ctx, p)

	return &executor.ExecutorResult{
		Name:     iterateExecutorDescriptor,
		Operator: p.Operator,
		Err:      err,
		Values:   vals,
	}
}

func (e IterateExecutor) fixedIterations(ctx context.Context, p *executor.ExecutorParams) (map[string]any, error) {
	// Optional
	// num_iters - sets the amount of iterations (default: 3)
	numIters := 3
	numItersArg, err := executor.GetTypedArg[uint64](p, "num_iters")
	if err == nil {
		if numItersArg > uint64(math.MaxInt64) {
			return nil, fmt.Errorf("num_iters value is of out int64 range")
		}
		numIters = int(numItersArg)
	}

	runtimeParams := p.Copy()
	for i := range numIters {

		for j, node := range p.Children {
			slog.Info("running iteration", "i", i, "nodeIdx", j)

			nodeParams := executor.MakeNodeParams(node, runtimeParams)

			result := node.Executor.Execute(ctx, nodeParams)

			if result.Err != nil {
				slog.Error("failed to execute node", "error", fmt.Sprintf("(%T): %v", result.Err, result.Err))
				return nil, fmt.Errorf("iteration failed on node '%s': %w", node.Operator, result.Err)
			}

			runtimeParams = executor.ProcessResult(runtimeParams, result)
		}

	}

	return runtimeParams.Args, nil
}
