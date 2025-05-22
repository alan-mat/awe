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

type ollamaChatStreamResponse struct {
	Model     string            `json:"model"`
	CreatedAt string            `json:"created_at"`
	Message   ollamaChatMessage `json:"message"`
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

func (p OllamaProvider) CreateCompletionStream(ctx context.Context, req CompletionRequest) (CompletionStream, error) {
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

	return NewOllamaCompletionStream(respBody), nil
}

type OllamaCompletionStream struct {
	body   io.ReadCloser
	reader *bufio.Reader
}

func NewOllamaCompletionStream(body io.ReadCloser) *OllamaCompletionStream {
	reader := bufio.NewReader(body)
	s := &OllamaCompletionStream{
		body:   body,
		reader: reader,
	}
	return s
}

func (s OllamaCompletionStream) Recv() (string, error) {
	line, err := s.reader.ReadBytes('\n')
	if err != nil {
		return "", err
	}

	var response ollamaChatStreamResponse
	err = json.Unmarshal(line, &response)
	if err != nil {
		return "", fmt.Errorf("failed to deserialize chat stream response: %w", err)
	}

	return response.Message.Content, nil
}

func (s OllamaCompletionStream) Close() error {
	return s.body.Close()
}
