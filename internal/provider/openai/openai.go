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

package openai

import (
	"context"
	"fmt"
	"os"

	"github.com/alan-mat/awe/internal/api"
	"github.com/sashabaranov/go-openai"
)

const embedMaxDocsLength = 2048

type OpenAIProvider struct {
	client     *openai.Client
	vectorDims int
}

func New() *OpenAIProvider {
	c := openai.NewClient(os.Getenv("OPENAI_API_KEY"))
	return &OpenAIProvider{
		client:     c,
		vectorDims: 1024,
	}
}

func (p OpenAIProvider) Generate(ctx context.Context, req api.GenerationRequest) (api.CompletionStream, error) {
	openaiReq := openai.ChatCompletionRequest{
		Model:       openai.O4Mini,
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

	if req.ResponseSchema != nil {
		openaiReq.ResponseFormat = &openai.ChatCompletionResponseFormat{
			Type:       openai.ChatCompletionResponseFormatTypeJSONSchema,
			JSONSchema: p.parseResponseSchema(req.ResponseSchema),
		}
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

func (p OpenAIProvider) Chat(ctx context.Context, req api.ChatRequest) (api.CompletionStream, error) {
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
		Model:    openai.O4Mini,
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

func (p OpenAIProvider) EmbedQuery(ctx context.Context, q string) ([]float32, error) {
	openaiReq := &openai.EmbeddingRequestStrings{
		Input:          []string{q},
		Model:          "text-embedding-3-small",
		EncodingFormat: "float",
		Dimensions:     p.vectorDims,
	}

	res, err := p.client.CreateEmbeddings(ctx, openaiReq)
	if err != nil {
		return nil, err
	}

	return res.Data[0].Embedding, nil
}

func (p OpenAIProvider) EmbedDocuments(ctx context.Context, docs []*api.EmbedDocumentRequest) ([]*api.DocumentEmbedding, error) {
	docEmbeddings := make([]*api.DocumentEmbedding, 0, len(docs))

	for _, doc := range docs {
		if len(doc.Chunks) > embedMaxDocsLength {
			return nil, fmt.Errorf("length of chunks exceeds limit: accepts '%d', received '%d'", embedMaxDocsLength, len(doc.Chunks))
		}

		openaiReq := &openai.EmbeddingRequestStrings{
			Input:          doc.Chunks,
			Model:          "text-embedding-3-small",
			EncodingFormat: "float",
			Dimensions:     p.vectorDims,
		}

		res, err := p.client.CreateEmbeddings(ctx, openaiReq)
		if err != nil {
			return nil, fmt.Errorf("failed to create embeddings for document '%s': %w", doc.Title, err)
		}

		vals := make([][]float32, 0, len(res.Data))
		for _, e := range res.Data {
			vals = append(vals, e.Embedding)
		}

		docEmbeddings = append(docEmbeddings, &api.DocumentEmbedding{
			Title:  doc.Title,
			Chunks: doc.Chunks,
			Values: vals,
		})
	}

	return docEmbeddings, nil
}

func (p OpenAIProvider) GetDimensions() uint {
	return uint(p.vectorDims)
}

func (p OpenAIProvider) parseRequestHistory(h []*api.ChatMessage) []openai.ChatCompletionMessage {
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

func (p OpenAIProvider) parseResponseSchema(s *api.Schema) *openai.ChatCompletionResponseFormatJSONSchema {
	schema := &openai.ChatCompletionResponseFormatJSONSchema{
		Name:   s.Title,
		Schema: s,
	}
	return schema
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
