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

func (p OpenAIProvider) Generate(ctx context.Context, req GenerationRequest) (CompletionStream, error) {
	openaiReq := openai.ChatCompletionRequest{
		Model:       openai.GPT4Dot1Nano,
		Temperature: req.Temperature,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: req.Prompt,
			},
		},
		Stream: true,
	}

	if req.ModelName != "" {
		openaiReq.Model = req.ModelName
	}

	s, err := p.client.CreateChatCompletionStream(ctx, openaiReq)
	if err != nil {
		return nil, err
	}

	completionStream := &OpenAIChatStream{
		stream: s,
	}
	return completionStream, nil
}

func (p OpenAIProvider) Chat(ctx context.Context, req ChatRequest) (CompletionStream, error) {
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

	completionStream := &OpenAIChatStream{
		stream: s,
	}
	return completionStream, nil
}

func (p OpenAIProvider) parseRequestHistory(h []*message.Chat) []openai.ChatCompletionMessage {
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

/* type OpenAIGenerationStream struct {
	stream *openai.CompletionStream
}

func (s OpenAIGenerationStream) Recv() (string, error) {
	res, err := s.stream.Recv()
	if err != nil {
		return "", err
	}

	return res.Choices[0].Text, nil
}

func (s OpenAIGenerationStream) Close() error {
	return s.stream.Close()
} */

type OpenAIChatStream struct {
	stream *openai.ChatCompletionStream
}

func (s OpenAIChatStream) Recv() (string, error) {
	res, err := s.stream.Recv()
	if err != nil {
		return "", err
	}

	return res.Choices[0].Delta.Content, nil
}

func (s OpenAIChatStream) Close() error {
	return s.stream.Close()
}
