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
	client     *genai.Client
	vectorDims *int32
}

func NewGeminiProvider() *GeminiProvider {
	// New methods might need error return
	// to handle error returns from client libs like genai
	c, _ := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:  os.Getenv("GEMINI_API_KEY"),
		Backend: genai.BackendGeminiAPI,
	})
	p := &GeminiProvider{
		client:     c,
		vectorDims: new(int32),
	}
	*(p.vectorDims) = 1536
	return p
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

func (p *GeminiProvider) EmbedQuery(ctx context.Context, q string) ([]float32, error) {
	contents := genai.Text(q)

	config := &genai.EmbedContentConfig{
		TaskType:             "RETRIEVAL_QUERY",
		OutputDimensionality: p.vectorDims,
	}

	res, err := p.client.Models.EmbedContent(ctx, "gemini-embedding-exp-03-07", contents, config)
	if err != nil {
		return nil, err
	}

	vals := res.Embeddings[0].Values
	return vals, nil
}

func (p *GeminiProvider) EmbedDocuments(ctx context.Context, docs []*EmbedDocumentRequest) ([]*DocumentEmbedding, error) {
	embeddings := make([]*DocumentEmbedding, 0, len(docs))

	for _, doc := range docs {
		contents := make([]*genai.Content, 0, len(doc.Chunks))
		for _, chunk := range doc.Chunks {
			content := genai.NewContentFromText(chunk, genai.RoleUser)
			contents = append(contents, content)
		}

		config := &genai.EmbedContentConfig{
			TaskType:             "RETRIEVAL_DOCUMENT",
			Title:                doc.Title,
			OutputDimensionality: p.vectorDims,
		}

		res, err := p.client.Models.EmbedContent(ctx, "gemini-embedding-exp-03-07", contents, config)
		if err != nil {
			return nil, err
		}

		values := make([][]float32, 0, len(res.Embeddings))
		for _, rEmbedding := range res.Embeddings {
			values = append(values, rEmbedding.Values)
		}

		docEmbed := &DocumentEmbedding{
			Title:  doc.Title,
			Values: values,
			Chunks: doc.Chunks,
		}
		embeddings = append(embeddings, docEmbed)
	}

	return embeddings, nil
}

func (p GeminiProvider) GetDimensions() uint {
	return uint(*p.vectorDims)
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
