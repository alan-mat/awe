package provider

import (
	"context"
	"os"

	"github.com/alan-mat/awe/internal/message"
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
	messages := make([]openai.ChatCompletionMessage, 0)

	if req.SystemPrompt != "" {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: req.SystemPrompt,
		})
	}

	msgHistory := p.parseRequestHistory(req.History)
	if len(msgHistory) > 0 {
		messages = append(messages, msgHistory...)
	}

	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: req.Query,
	})

	openaiReq := openai.ChatCompletionRequest{
		Model:    openai.GPT4Dot1Mini,
		Messages: messages,
		Stream:   true,
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

func (p *OpenAIProvider) parseRequestHistory(h []*message.Chat) []openai.ChatCompletionMessage {
	msgs := make([]openai.ChatCompletionMessage, len(h))
	for i, m := range h {
		ccm := openai.ChatCompletionMessage{
			Role:    m.Role.String(),
			Content: m.Content,
		}
		msgs[i] = ccm
	}
	return msgs
}

type OpenAICompletionStream struct {
	stream *openai.ChatCompletionStream
}

func (s OpenAICompletionStream) Recv() (string, error) {
	res, err := s.stream.Recv()
	if err != nil {
		return "", err
	}

	return res.Choices[0].Delta.Content, nil
}

func (s OpenAICompletionStream) Close() error {
	return s.stream.Close()
}
