package transport

import (
	"context"
	"errors"
	"io"
	"log/slog"

	"github.com/alan-mat/awe/internal/provider"
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

func ProcessCompletionStream(ctx context.Context, ms MessageStream, cs provider.CompletionStream) (string, error) {
	acc := ""
	msgId := 0
	for {
		chunk, err := cs.Recv()
		if errors.Is(err, io.EOF) {
			return acc, nil
		}

		if err != nil {
			ms.Send(ctx, MessageStreamPayload{
				ID:      msgId,
				Content: "something went wrong",
				Status:  "ERR",
			})
			return acc, err
		}

		acc += chunk

		err = ms.Send(ctx, MessageStreamPayload{
			ID:      msgId,
			Content: chunk,
			Status:  "OK",
		})
		if err != nil {
			slog.Debug("failed sending chunk to message stream", "chunk", chunk)
		}

		msgId += 1
	}
}
