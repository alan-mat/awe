package provider

import (
	"context"
	"io"
	"iter"
	"os"

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
	i := p.client.Models.GenerateContentStream(
		ctx,
		"gemini-2.0-flash",
		genai.Text(req.Query),
		nil,
	)

	next, stop := iter.Pull2(i)
	return &GeminiCompletionStream{
		next: next,
		stop: stop,
	}, nil
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
