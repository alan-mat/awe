package api

import (
	"context"
	"errors"
	"io"
)

type ChatRequest struct {
	// Required
	Query string

	// Optional params
	ModelName    string
	SystemPrompt string
	History      []*ChatMessage
}

type GenerationRequest struct {
	// Required
	Prompt string

	// Optional params
	ModelName      string
	ResponseSchema *Schema
	Temperature    float32
}

func FromPrompt(prompt string) *GenerationRequest {
	return &GenerationRequest{
		Prompt:      prompt,
		ModelName:   "",
		Temperature: 0.7,
	}
}

type CompletionStream interface {
	Recv() (string, error)
	Close() error
}

type completionStreamPayload struct {
	content string
	err     error
}

// StreamReadAll receives from a completion stream accumulating the results
// and returning the streamed chunks as a whole. This function will return an error
// if one is received from the CompletionStream. Calling this function will always
// close the underlying stream.
func StreamReadAll(ctx context.Context, stream CompletionStream) (string, error) {
	defer stream.Close()
	dataChan := make(chan completionStreamPayload)

	go func() {
		defer close(dataChan)

		for {
			chunk, err := stream.Recv()

			if errors.Is(err, io.EOF) {
				return
			}

			if err != nil {
				dataChan <- completionStreamPayload{err: err}
				return
			}

			dataChan <- completionStreamPayload{content: chunk}
		}
	}()

	var acc string

	for {
		select {
		case <-ctx.Done():
			return acc, nil
		case payload, ok := <-dataChan:
			if !ok {
				// data stream closed
				return acc, nil
			}

			if payload.err != nil {
				return acc, nil
			}

			acc += payload.content
		}
	}
}
