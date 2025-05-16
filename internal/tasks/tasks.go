package tasks

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/alan-mat/awe/internal/executor"
	"github.com/alan-mat/awe/internal/message"
	pb "github.com/alan-mat/awe/internal/proto"
	"github.com/alan-mat/awe/internal/registry"
	"github.com/alan-mat/awe/internal/transport"

	"github.com/hibiken/asynq"
)

const (
	TypeChat = "awe:chat"
)

type chatTaskPayload struct {
	Query   string
	User    string
	History []*message.Chat
	Args    map[string]string
}

func NewChatTask(req *pb.ChatRequest) (*asynq.Task, error) {
	tp := chatTaskPayload{
		Query:   req.Query,
		User:    req.User,
		History: message.ParseChatHistory(req.History),
		Args:    req.Args,
	}
	payload, err := json.Marshal(tp)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeChat, payload), nil
}

type ChatTaskHandler struct {
	transport transport.Transport
}

func NewChatTaskHandler(transport transport.Transport) *ChatTaskHandler {
	return &ChatTaskHandler{
		transport: transport,
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

	exec, _ := registry.GetExecutor("generation.Simple")
	params := *executor.NewExecutorParams(
		id,
		p.Query,
		executor.WithTransport(h.transport),
		executor.WithArgs(map[string]any{
			"history": p.History,
		}),
	)
	exec.Execute(ctx, params)

	return nil
}
