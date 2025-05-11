package provider

import "github.com/alan-mat/awe/internal/message"

type CompletionRequest struct {
	Query   string
	History []*message.Chat
}

type CompletionStream interface {
	Recv() (string, error)
	Close() error
}
