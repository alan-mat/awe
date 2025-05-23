package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/alan-mat/awe/internal/api"
	"github.com/alan-mat/awe/internal/executor"
	pb "github.com/alan-mat/awe/internal/proto"
	"github.com/alan-mat/awe/internal/registry"
	"github.com/alan-mat/awe/internal/transport"
	"github.com/alan-mat/awe/internal/vector"

	"github.com/hibiken/asynq"
)

const (
	TypeChat = "awe:chat"
)

type chatTaskPayload struct {
	Query   string
	User    string
	History []*api.ChatMessage
	Args    map[string]string
}

func NewChatTask(req *pb.ChatRequest) (*asynq.Task, error) {
	tp := chatTaskPayload{
		Query:   req.Query,
		User:    req.User,
		History: api.ParseChatHistory(req.History),
		Args:    req.Args,
	}
	payload, err := json.Marshal(tp)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeChat, payload), nil
}

type ChatTaskHandler struct {
	transport   transport.Transport
	vectorStore vector.Store
}

func NewChatTaskHandler(transport transport.Transport, vectorStore vector.Store) *ChatTaskHandler {
	return &ChatTaskHandler{
		transport:   transport,
		vectorStore: vectorStore,
	}
}

func (h *ChatTaskHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var p chatTaskPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}
	id := t.ResultWriter().TaskID()

	slog.Info("received chat task", "user", p.User, "query", p.Query, "history", p.History)
	slog.Info("task id", "id", id)

	workflow, _ := registry.GetWorkflow("naive_rag")

	args := make(map[string]any)
	for k, v := range p.Args {
		args[k] = v
	}

	if len(p.History) > 0 {
		args["history"] = p.History
	}

	params := executor.NewExecutorParams(
		id,
		p.Query,
		executor.WithTransport(h.transport),
		executor.WithVectorStore(h.vectorStore),
		executor.WithArgs(args),
	)

	res := workflow.Execute(ctx, params)
	if res.Err != nil {
		return fmt.Errorf("workflow execution failed: %w", asynq.SkipRetry)
	}

	ms, _ := h.transport.GetMessageStream(id)
	err := ms.Send(ctx, transport.MessageStreamPayload{
		Content: "task finished",
		Status:  "DONE",
	})
	if err != nil {
		slog.Warn("failed to write DONE message to stream", "id", id)
	}

	return nil
}
