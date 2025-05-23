package provider

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/alan-mat/awe/internal/http"
)

const (
	OllamaEndpoint = "http://localhost:11434"
)

type OllamaProvider struct {
	client       http.Client
	defaultModel string
}

type ollamaChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaStreamResponse struct {
	Model     string            `json:"model"`
	CreatedAt string            `json:"created_at"`
	Message   ollamaChatMessage `json:"message"`
	Response  string            `json:"response"`
	Done      bool              `json:"done"`
}

func NewOllamaProvider() *OllamaProvider {
	c := http.NewClient(
		OllamaEndpoint,
		http.WithMaxRetries(3),
	)
	p := &OllamaProvider{
		client:       c,
		defaultModel: "gemma3:4b",
	}
	return p
}

func (p OllamaProvider) Generate(ctx context.Context, req GenerationRequest) (CompletionStream, error) {
	var model string
	if req.ModelName != "" {
		model = req.ModelName
	} else {
		model = p.defaultModel
	}

	requestData := map[string]any{
		"model":  model,
		"prompt": req.Prompt,
		"options": map[string]any{
			"temperature": req.Temperature,
		},
	}

	respBody, err := p.client.RequestStream(http.MethodPost, "/api/chat", requestData)
	if err != nil {
		return nil, fmt.Errorf("completion request failed: %w", err)
	}

	return NewOllamaCompletionStream(respBody, false), nil
}

func (p OllamaProvider) Chat(ctx context.Context, req ChatRequest) (CompletionStream, error) {
	if req.Query == "" {
		return nil, fmt.Errorf("completion request failed: missing parameter 'query' in request")
	}

	var model string
	if req.ModelName != "" {
		model = req.ModelName
	} else {
		model = p.defaultModel
	}

	messages := make([]ollamaChatMessage, 0, 1)
	if req.SystemPrompt != "" {
		messages = append(messages, ollamaChatMessage{
			Role:    "system",
			Content: req.SystemPrompt,
		})
	}

	for _, cm := range req.History {
		messages = append(messages, ollamaChatMessage{
			Role:    cm.Role.String(),
			Content: cm.Content,
		})
	}

	messages = append(messages, ollamaChatMessage{
		Role:    "user",
		Content: req.Query,
	})

	requestData := map[string]any{
		"model":    model,
		"messages": messages,
	}

	respBody, err := p.client.RequestStream(http.MethodPost, "/api/chat", requestData)
	if err != nil {
		return nil, fmt.Errorf("completion request failed: %w", err)
	}

	return NewOllamaCompletionStream(respBody, true), nil
}

type OllamaCompletionStream struct {
	body   io.ReadCloser
	reader *bufio.Reader
	chat   bool
}

func NewOllamaCompletionStream(body io.ReadCloser, chat bool) *OllamaCompletionStream {
	reader := bufio.NewReader(body)
	s := &OllamaCompletionStream{
		body:   body,
		reader: reader,
		chat:   chat,
	}
	return s
}

func (s OllamaCompletionStream) Recv() (string, error) {
	line, err := s.reader.ReadBytes('\n')
	if err != nil {
		return "", err
	}

	var response ollamaStreamResponse
	err = json.Unmarshal(line, &response)
	if err != nil {
		return "", fmt.Errorf("failed to deserialize chat stream response: %w", err)
	}

	var out string
	if s.chat {
		out = response.Message.Content
	} else {
		out = response.Response
	}

	return out, nil
}

func (s OllamaCompletionStream) Close() error {
	return s.body.Close()
}
