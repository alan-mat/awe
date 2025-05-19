package generation

import (
	"context"
	"errors"
	"fmt"
	"io"
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
	e := NewSimpleExecutor()
	err := registry.RegisterExecutor(simpleExecutorDescriptor, e)
	if err != nil {
		slog.Error("failed to register executor", "name", simpleExecutorDescriptor)
	}
}

type SimpleExecutor struct {
	Provider  provider.LMProviderType
	operators map[string]func(context.Context, *executor.ExecutorParams) error
}

func NewSimpleExecutor() *SimpleExecutor {
	e := &SimpleExecutor{
		Provider: provider.LMProviderTypeGemini,
	}
	e.operators = map[string]func(context.Context, *executor.ExecutorParams) error{
		"generate": e.generate,
	}
	return e
}

func (e *SimpleExecutor) Execute(ctx context.Context, p *executor.ExecutorParams) executor.ExecutorResult {
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

func (e *SimpleExecutor) generate(ctx context.Context, p *executor.ExecutorParams) error {
	msgId := 0
	ms, err := p.Transport.GetMessageStream(p.GetTaskID())
	if err != nil {
		slog.Warn("failed to create message stream", "id", p.GetTaskID())
		return err
	}

	if len(p.GetQuery()) == 0 {
		return fmt.Errorf("<empty query>: %w", asynq.SkipRetry)
	}

	prov, err := provider.NewLMProvider(provider.LMProviderTypeGemini)
	if err != nil {
		slog.Warn("error creating new lmprovider, cancelling task")
		ms.Send(ctx, transport.MessageStreamPayload{
			ID:      msgId,
			Content: "something went wrong",
			Status:  "ERR",
		})
		return err
	}

	var history []*message.Chat
	h, ok := p.Args["history"]
	if !ok {
		history = nil
	} else {
		history = h.([]*message.Chat)
	}

	creq := provider.CompletionRequest{
		Query:   p.GetQuery(),
		History: history,
	}

	cs, err := prov.CreateCompletionStream(ctx, creq)
	if err != nil {
		slog.Warn("error creating chat completion stream, cancelling task")
		ms.Send(ctx, transport.MessageStreamPayload{
			ID:      msgId,
			Content: "something went wrong",
			Status:  "ERR",
		})
		return err
	}
	defer cs.Close()

	for {
		chunk, err := cs.Recv()
		if errors.Is(err, io.EOF) {
			slog.Debug("provider stream finished", "id", p.GetTaskID())
			break
		}

		if err != nil {
			slog.Debug("provider stream error", "id", p.GetTaskID(), "error", err)
			ms.Send(ctx, transport.MessageStreamPayload{
				ID:      msgId,
				Content: "something went wrong",
				Status:  "ERR",
			})
			return err
		}

		err = ms.Send(ctx, transport.MessageStreamPayload{
			ID:      msgId,
			Content: chunk,
			Status:  "OK",
		})
		if err != nil {
			slog.Debug("failed sending chunk to stream", "id", p.GetTaskID(), "chunk", chunk)
		}

		msgId += 1
	}

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
