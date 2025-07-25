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

package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/alan-mat/awe/internal/executor"
	"github.com/alan-mat/awe/internal/registry"
	"github.com/alan-mat/awe/internal/transport"
	"github.com/alan-mat/awe/internal/vector"
	"github.com/hibiken/asynq"
)

type TaskHandler struct {
	transport   transport.Transport
	vectorStore vector.Store
}

func NewTaskHandler(transport transport.Transport, vectorStore vector.Store) *TaskHandler {
	return &TaskHandler{
		transport:   transport,
		vectorStore: vectorStore,
	}
}

func (h TaskHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var query, workflowId, user string
	args := make(map[string]any)

	switch t.Type() {
	case TypeChat:
		var p chatTaskPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return err
		}
		slog.Info("received chat task", "user", p.User, "query", p.Query, "history", p.History)

		for k, v := range p.Args {
			args[k] = v
		}
		if len(p.History) > 0 {
			args["history"] = p.History
		}
		query = p.Query
		user = p.User
		workflowId = DefaultWorkflowChat

	case TypeSearch:
		var p searchTaskPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return err
		}
		slog.Info("received search task", "user", p.User, "query", p.Query)

		for k, v := range p.Args {
			args[k] = v
		}
		query = p.Query
		user = p.User
		workflowId = DefaultWorkflowSearch

	case TypeExecute:
		var p executeTaskPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return err
		}
		slog.Info("received execute task", "workflowId", p.WorkflowId, "user", p.User, "query", p.Query, "history", p.History)

		for k, v := range p.Args {
			args[k] = v
		}
		if len(p.History) > 0 {
			args["history"] = p.History
		}
		query = p.Query
		user = p.User
		workflowId = p.WorkflowId

	default:
		return fmt.Errorf("unrecognized task type (%w)", asynq.SkipRetry)
	}

	id := t.ResultWriter().TaskID()
	slog.Info("task id", "id", id)
	ms, err := h.transport.GetMessageStream(id)
	if err != nil {
		slog.Error("failed to initialize message stream", "err", err)
		return fmt.Errorf("failed to initialize message stream: %v (%w)", err, asynq.SkipRetry)
	}

	trace := &transport.RequestTrace{
		ID:          id,
		Status:      transport.TraceStatusRunning,
		StartedAt:   time.Now().UnixNano(),
		CompletedAt: 0,
		Query:       query,
		User:        user,
	}
	err = h.transport.SetTrace(ctx, trace)
	if err != nil {
		slog.Error("failed to set trace", "id", id, "err", err)
	}

	workflow, err := registry.GetWorkflow(workflowId)
	if err != nil {
		errf := fmt.Errorf("workflow not found: %v (%w)", err, asynq.SkipRetry)
		slog.Error(fmt.Sprintf("%v", errf))
		ms.Send(ctx, transport.MessageStreamPayload{
			Content: "workflow not found",
			Status:  "ERR",
		})

		trace.CompletedAt = time.Now().Unix()
		trace.Status = transport.TraceStatusFailed
		err = h.transport.SetTrace(ctx, trace)
		if err != nil {
			slog.Error("failed to set trace", "id", id, "err", err)
		}

		return errf
	}

	params := executor.NewExecutorParams(
		id,
		query,
		executor.WithTransport(h.transport),
		executor.WithVectorStore(h.vectorStore),
		executor.WithArgs(args),
	)

	res := workflow.Execute(ctx, params)
	if res.Err != nil {
		ms.Send(ctx, transport.MessageStreamPayload{
			Content: "workflow execution failed",
			Status:  "ERR",
		})

		trace.CompletedAt = time.Now().UnixNano()
		trace.Status = transport.TraceStatusFailed
		err = h.transport.SetTrace(ctx, trace)
		if err != nil {
			slog.Error("failed to set trace", "id", id, "err", err)
		}

		return fmt.Errorf("workflow execution failed: %w", asynq.SkipRetry)
	}

	err = ms.Send(ctx, transport.MessageStreamPayload{
		Content: "task finished",
		Status:  "DONE",
	})
	if err != nil {
		slog.Warn("failed to write DONE message to stream", "id", id)
	}

	trace.CompletedAt = time.Now().UnixNano()
	trace.Status = transport.TraceStatusCompleted
	err = h.transport.SetTrace(ctx, trace)
	if err != nil {
		slog.Error("failed to set trace", "id", id, "err", err)
	}

	return nil
}
