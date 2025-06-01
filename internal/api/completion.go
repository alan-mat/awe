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
