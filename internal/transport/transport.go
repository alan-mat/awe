package transport

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/alan-mat/awe/internal/api"
)

var (
	TraceExpiry = time.Hour * 24
)

type Transport interface {
	GetMessageStream(id string) (MessageStream, error)
	SetTrace(ctx context.Context, trace *RequestTrace) error
	GetTrace(ctx context.Context, traceId string) (*RequestTrace, error)
}

type MessageStream interface {
	Send(ctx context.Context, payload MessageStreamPayload) error

	Recv(ctx context.Context) (*MessageStreamPayload, error)

	// Text reads the entire message stream and returns its content
	//
	// Note this will not retrieve any Documents sent in the stream
	Text(ctx context.Context) (string, error)

	GetID() string
}

type MessageStreamPayload struct {
	ID     int         `json:"id"`
	Status string      `json:"status"`
	Type   MessageType `json:"type"`

	Content  string   `json:"content"`
	Document Document `json:"document"`
}

type MessageType int

const (
	MessageTypeOther = iota
	MessageTypeContent
	MessageTypeDocument
)

type Document struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Source  string `json:"source"`
}

type RequestTrace struct {
	ID          string `redis:"id"`
	Status      int    `redis:"status"`
	StartedAt   int64  `redis:"started_at"`
	CompletedAt int64  `redis:"completed_at"`
	Query       string `redis:"query"`
	User        string `redis:"user"`
}

type TraceStatus int

const (
	TraceStatusUnspecified = iota
	TraceStatusRunning
	TraceStatusCompleted
	TraceStatusFailed
)

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
				Status:  "ERR",
				Content: "something went wrong",
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
			Type:    MessageTypeContent,
			Status:  "OK",
			Content: acc,
		})
		if err != nil {
			slog.Debug("failed sending chunk to message stream", "chunk", acc)
		}

		acc = ""
		msgId += 1
	}
}
