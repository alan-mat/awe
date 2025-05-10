package provider

import (
	"context"
	"os"

	"github.com/sashabaranov/go-openai"
)

type OpenAIProvider struct {
	client *openai.Client
}

func NewOpenAIProvider() *OpenAIProvider {
	c := openai.NewClient(os.Getenv("OPENAI_API_KEY"))
	return &OpenAIProvider{
		client: c,
	}
}

func (p *OpenAIProvider) CreateCompletionStream(ctx context.Context, req CompletionRequest) (CompletionStream, error) {
	openaiReq := openai.ChatCompletionRequest{
		Model: openai.GPT4Dot1Mini,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: req.Query,
			},
		},
		Stream: true,
	}

	s, err := p.client.CreateChatCompletionStream(ctx, openaiReq)
	if err != nil {
		return nil, err
	}

	completionStream := &OpenAICompletionStream{
		stream: s,
	}
	return completionStream, nil
}

type OpenAICompletionStream struct {
	stream *openai.ChatCompletionStream
}

func (s *OpenAICompletionStream) Recv() (string, error) {
	res, err := s.stream.Recv()
	if err != nil {
		return "", err
	}

	return res.Choices[0].Delta.Content, nil
}

func (s *OpenAICompletionStream) Close() error {
	return s.stream.Close()
}
