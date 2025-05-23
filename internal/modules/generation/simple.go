package generation

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/alan-mat/awe/internal/executor"
	"github.com/alan-mat/awe/internal/message"
	"github.com/alan-mat/awe/internal/provider"
	"github.com/alan-mat/awe/internal/registry"
	"github.com/alan-mat/awe/internal/transport"
	"github.com/hibiken/asynq"
)

var simpleExecutorDescriptor = "generation.Simple"

func init() {
	e, err := NewSimpleExecutor()
	if err != nil {
		slog.Error("failed to initialize executor", "name", simpleExecutorDescriptor, "err", err)
		return
	}

	err = registry.RegisterExecutor(simpleExecutorDescriptor, e)
	if err != nil {
		slog.Error("failed to register executor", "name", simpleExecutorDescriptor)
	}
}

type SimpleExecutor struct {
	DefaultLMProvider provider.LMProvider
	operators         map[string]func(context.Context, *executor.ExecutorParams) error
}

func NewSimpleExecutor() (*SimpleExecutor, error) {
	lp, err := provider.NewLMProvider(provider.LMProviderTypeGemini)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize default providers: %w", err)
	}

	e := &SimpleExecutor{
		DefaultLMProvider: lp,
	}

	e.operators = map[string]func(context.Context, *executor.ExecutorParams) error{
		"generate": e.generate,
		"chat":     e.chat,
	}
	return e, nil
}

func (e *SimpleExecutor) Execute(ctx context.Context, p *executor.ExecutorParams) *executor.ExecutorResult {
	if p.Operator == "" {
		p.Operator = "generate"
	}
	slog.Info("executing", "name", simpleExecutorDescriptor, "op", p.Operator, "query", p.GetQuery(), "id", p.GetTaskID())

	opFunc, exists := e.operators[p.Operator]
	if !exists {
		return e.buildResult(p.Operator, executor.ErrOperatorNotFound{
			ExecutorName: simpleExecutorDescriptor, OperatorName: p.Operator}, nil)
	}

	err := opFunc(ctx, p)
	return e.buildResult(p.Operator, err, nil)
}

func (e SimpleExecutor) generate(ctx context.Context, p *executor.ExecutorParams) error {
	ms, err := p.Transport.GetMessageStream(p.GetTaskID())
	if err != nil {
		slog.Warn("failed to create message stream", "id", p.GetTaskID())
		return err
	}

	if len(p.GetQuery()) == 0 {
		return fmt.Errorf("<empty query>: %w", asynq.SkipRetry)
	}

	greq := provider.FromPrompt(p.GetQuery())
	temperature, err := executor.GetTypedArg[float64](p, "temperature")
	if err != nil {
		if _, ok := err.(executor.ErrArgMissing); !ok {
			return err
		}
	} else {
		greq.Temperature = float32(temperature)
	}

	cs, err := e.DefaultLMProvider.Generate(ctx, *greq)
	if err != nil {
		slog.Warn("error creating generation completion stream, cancelling task")
		ms.Send(ctx, transport.MessageStreamPayload{
			Content: "something went wrong",
			Status:  "ERR",
		})
		return err
	}
	defer cs.Close()

	_, err = transport.ProcessCompletionStream(ctx, ms, cs)
	if err != nil {
		return fmt.Errorf("failed to process completion stream: %w", err)
	}

	return nil
}

func (e *SimpleExecutor) chat(ctx context.Context, p *executor.ExecutorParams) error {
	ms, err := p.Transport.GetMessageStream(p.GetTaskID())
	if err != nil {
		slog.Warn("failed to create message stream", "id", p.GetTaskID())
		return err
	}

	if len(p.GetQuery()) == 0 {
		return fmt.Errorf("<empty query>: %w", asynq.SkipRetry)
	}

	var history []*message.Chat
	h, ok := p.Args["history"]
	if !ok {
		history = nil
	} else {
		history = h.([]*message.Chat)
	}

	creq := provider.ChatRequest{
		Query:   p.GetQuery(),
		History: history,
	}

	slog.Info("chat request", "history", history, "query", creq.Query)

	cs, err := e.DefaultLMProvider.Chat(ctx, creq)
	if err != nil {
		slog.Warn("error creating chat completion stream, cancelling task")
		ms.Send(ctx, transport.MessageStreamPayload{
			Content: "something went wrong",
			Status:  "ERR",
		})
		return err
	}
	defer cs.Close()

	_, err = transport.ProcessCompletionStream(ctx, ms, cs)
	if err != nil {
		return fmt.Errorf("failed to process completion stream: %w", err)
	}

	return nil
}

func (e *SimpleExecutor) buildResult(operator string, err error, values map[string]any) *executor.ExecutorResult {
	return &executor.ExecutorResult{
		Name:     simpleExecutorDescriptor,
		Operator: operator,
		Err:      err,
		Values:   values,
	}
}
