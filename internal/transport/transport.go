package transport

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"

	"github.com/alan-mat/awe/internal/api"
)

type Transport interface {
	GetMessageStream(id string) (MessageStream, error)
}

type MessageStream interface {
	Send(ctx context.Context, payload MessageStreamPayload) error
	Recv(ctx context.Context) (*MessageStreamPayload, error)
	Text(ctx context.Context) (string, error)
	GetID() string
}

type MessageStreamPayload struct {
	ID      int    `json:"id"`
	Content string `json:"content"`
	Status  string `json:"status"`
}

func ProcessCompletionStream(ctx context.Context, ms MessageStream, cs api.CompletionStream) (string, error) {
	var acc, sink string
	msgId := 0

	for {
		chunk, err := cs.Recv()
		if errors.Is(err, io.EOF) {
			return sink, nil
		}

		if err != nil {
			ms.Send(ctx, MessageStreamPayload{
				ID:      msgId,
				Content: "something went wrong",
				Status:  "ERR",
			})
			return sink, err
		}

		acc += chunk
		sink += chunk

		if strings.TrimSpace(chunk) == "" {
			continue
		}

		err = ms.Send(ctx, MessageStreamPayload{
			ID:      msgId,
			Content: acc,
			Status:  "OK",
		})
		if err != nil {
			slog.Debug("failed sending chunk to message stream", "chunk", acc)
		}

		acc = ""
		msgId += 1
	}
}
