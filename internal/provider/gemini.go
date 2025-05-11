package provider

import (
	"context"
	"io"
	"iter"
	"os"

	"github.com/alan-mat/awe/internal/message"
	"google.golang.org/genai"
)

type GeminiProvider struct {
	client *genai.Client
}

func NewGeminiProvider() *GeminiProvider {
	// New methods might need error return
	// to handle error returns from client libs like genai
	c, _ := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:  os.Getenv("GEMINI_API_KEY"),
		Backend: genai.BackendGeminiAPI,
	})
	return &GeminiProvider{
		client: c,
	}
}

func (p *GeminiProvider) CreateCompletionStream(ctx context.Context, req CompletionRequest) (CompletionStream, error) {
	contents := p.parseRequestHistory(req.History)
	contents = append(contents, genai.NewContentFromText(req.Query, genai.RoleUser))

	i := p.client.Models.GenerateContentStream(
		ctx,
		"gemini-2.0-flash",
		contents,
		nil,
	)

	next, stop := iter.Pull2(i)
	return &GeminiCompletionStream{
		next: next,
		stop: stop,
	}, nil
}

func (p *GeminiProvider) parseRequestHistory(h []*message.Chat) []*genai.Content {
	contents := make([]*genai.Content, len(h))
	roleTypes := map[message.ChatRole]genai.Role{
		message.RoleUser:      genai.RoleUser,
		message.RoleAssistant: genai.RoleModel,
	}
	for i, m := range h {
		c := genai.NewContentFromText(m.Content, roleTypes[m.Role])
		contents[i] = c
	}
	return contents
}

type GeminiCompletionStream struct {
	next func() (*genai.GenerateContentResponse, error, bool)
	stop func()
}

func (s *GeminiCompletionStream) Recv() (string, error) {
	res, err, valid := s.next()
	if !valid {
		//iterator is finished
		return "", io.EOF
	}

	if err != nil {
		return "", err
	}

	return res.Text(), nil
}

func (s *GeminiCompletionStream) Close() error {
	s.stop()
	return nil
}
