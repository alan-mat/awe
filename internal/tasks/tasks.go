package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/alan-mat/awe/internal/message"
	pb "github.com/alan-mat/awe/internal/proto"
	"github.com/alan-mat/awe/internal/provider"
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

	slog.Info("received chat task", "user", p.User, "query", p.Query, "history", p.History, "args", p.Args)
	slog.Info("task id", "id", id)

	msgId := 0
	s, err := h.transport.GetMessageStream(id)
	if err != nil {
		slog.Warn("failed to create message stream", "id", id)
		return err
	}

	prov, err := provider.NewLMProvider(provider.LMProviderTypeGemini)
	if err != nil {
		slog.Warn("error creating new lmprovider, cancelling task")
		s.Send(ctx, transport.MessageStreamPayload{
			ID:      msgId,
			Content: "something went wrong",
			Status:  "ERR",
		})
		return err
	}

	creq := provider.CompletionRequest{
		Query:   p.Query,
		History: p.History,
	}

	slog.Info(fmt.Sprintf("%v\n", creq))
	cs, err := prov.CreateCompletionStream(ctx, creq)
	if err != nil {
		slog.Warn("error creating chat completion stream, cancelling task")
		s.Send(ctx, transport.MessageStreamPayload{
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
			slog.Debug("provider stream finished", "id", id)
			break
		}

		if err != nil {
			slog.Debug("provider stream error", "id", id, "error", err)
			s.Send(ctx, transport.MessageStreamPayload{
				ID:      msgId,
				Content: "something went wrong",
				Status:  "ERR",
			})
			return err
		}

		err = s.Send(ctx, transport.MessageStreamPayload{
			ID:      msgId,
			Content: chunk,
			Status:  "OK",
		})
		if err != nil {
			slog.Debug("failed sending chunk to stream", "id", id, "chunk", chunk)
		}

		msgId += 1
	}

	err = s.Send(ctx, transport.MessageStreamPayload{
		ID:      msgId,
		Content: "task finished",
		Status:  "DONE",
	})
	if err != nil {
		slog.Warn("failed to write DONE message to stream", "id", id)
	}

	return nil
}
