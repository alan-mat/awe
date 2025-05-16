package transport

import "context"

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
