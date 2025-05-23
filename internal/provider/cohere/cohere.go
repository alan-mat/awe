package awe_cohere

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/alan-mat/awe/internal/api"
	cohere "github.com/cohere-ai/cohere-go/v2"
	cohereclient "github.com/cohere-ai/cohere-go/v2/client"
	coherecore "github.com/cohere-ai/cohere-go/v2/core"
)

const (
	EmbedMaxTexts = 96
)

type embedRequestWrapper struct {
	Title   string
	Chunks  []string
	Request *cohere.V2EmbedRequest
}

type embedResponseWrapper struct {
	Title    string
	Chunks   []string
	Response *cohere.EmbedByTypeResponse
}

type CohereProvider struct {
	client *cohereclient.Client
}

func New() *CohereProvider {
	c := cohereclient.NewClient(
		cohereclient.WithToken(os.Getenv("COHERE_API_KEY")),
		cohereclient.WithHTTPClient(
			&http.Client{
				Timeout: 60 * time.Second,
			},
		),
	)
	return &CohereProvider{
		client: c,
	}
}

func (p CohereProvider) Generate(ctx context.Context, req api.GenerationRequest) (api.CompletionStream, error) {
	temp := float64(req.Temperature)
	cohereReq := &cohere.V2ChatStreamRequest{
		Model:       "command-r-08-2024",
		Temperature: &temp,
	}

	if req.ModelName != "" {
		cohereReq.Model = req.ModelName
	}

	cohereReq.Messages = append(cohereReq.Messages, &cohere.ChatMessageV2{
		Role: "user",
		User: &cohere.UserMessage{Content: &cohere.UserMessageContent{
			String: req.Prompt,
		}},
	})

	stream, err := p.client.V2.ChatStream(ctx, cohereReq)
	if err != nil {
		return nil, fmt.Errorf("chat streaming request failed: %w", err)
	}

	return &CohereCompletionStream{stream: stream}, nil
}

func (p CohereProvider) Chat(ctx context.Context, req api.ChatRequest) (api.CompletionStream, error) {
	if req.Query == "" {
		return nil, fmt.Errorf("completion request failed: missing parameter 'query' in request")
	}

	cohereReq := &cohere.V2ChatStreamRequest{
		Model: "command-r-08-2024",
	}

	if req.ModelName != "" {
		cohereReq.Model = req.ModelName
	}

	if req.SystemPrompt != "" {
		cohereReq.Messages = append(cohereReq.Messages, &cohere.ChatMessageV2{
			Role: "system",
			System: &cohere.SystemMessage{Content: &cohere.SystemMessageContent{
				String: req.SystemPrompt,
			}},
		})
	}

	history := p.parseRequestHistory(req.History)
	if len(history) > 0 {
		cohereReq.Messages = append(cohereReq.Messages, history...)
	}

	cohereReq.Messages = append(cohereReq.Messages, &cohere.ChatMessageV2{
		Role: "user",
		User: &cohere.UserMessage{Content: &cohere.UserMessageContent{
			String: req.Query,
		}},
	})

	stream, err := p.client.V2.ChatStream(ctx, cohereReq)
	if err != nil {
		return nil, fmt.Errorf("chat streaming request failed: %w", err)
	}

	return &CohereCompletionStream{stream: stream}, nil
}

func (p CohereProvider) EmbedQuery(ctx context.Context, q string) ([]float32, error) {
	resp, err := p.client.V2.Embed(
		ctx,
		&cohere.V2EmbedRequest{
			Texts:          []string{q},
			Model:          "embed-multilingual-v3.0",
			InputType:      cohere.EmbedInputTypeSearchQuery,
			EmbeddingTypes: []cohere.EmbeddingType{cohere.EmbeddingTypeFloat},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("embed request failed: %w", err)
	}

	f32 := make([]float32, 0, len(resp.Embeddings.Float[0]))
	for _, f := range resp.Embeddings.Float[0] {
		f32 = append(f32, float32(f))
	}

	return f32, nil
}

func (p CohereProvider) EmbedDocuments(ctx context.Context, docs []*api.EmbedDocumentRequest) ([]*api.DocumentEmbedding, error) {
	embedRequests := make([]*embedRequestWrapper, 0, len(docs))
	for _, doc := range docs {
		if len(doc.Chunks) <= EmbedMaxTexts {
			req := &cohere.V2EmbedRequest{
				Texts:          doc.Chunks,
				Model:          "embed-multilingual-v3.0",
				InputType:      cohere.EmbedInputTypeSearchDocument,
				EmbeddingTypes: []cohere.EmbeddingType{cohere.EmbeddingTypeFloat},
			}
			embedRequests = append(embedRequests, &embedRequestWrapper{
				Title:   doc.Title,
				Chunks:  doc.Chunks,
				Request: req,
			})
		}

		parts := (len(doc.Chunks) / EmbedMaxTexts) + 1
		var start, end int
		for i := range parts {
			start, end = i*EmbedMaxTexts, (i+1)*EmbedMaxTexts
			if end > len(doc.Chunks) {
				end = len(doc.Chunks)
			}

			req := &cohere.V2EmbedRequest{
				Texts:          doc.Chunks[start:end],
				Model:          "embed-multilingual-v3.0",
				InputType:      cohere.EmbedInputTypeSearchDocument,
				EmbeddingTypes: []cohere.EmbeddingType{cohere.EmbeddingTypeFloat},
			}
			embedRequests = append(embedRequests, &embedRequestWrapper{
				Title:   doc.Title,
				Chunks:  doc.Chunks,
				Request: req,
			})
		}
	}

	var wg sync.WaitGroup
	var embedRespMu sync.Mutex
	embedResponses := make([]*embedResponseWrapper, 0, len(embedRequests))

	for _, ereq := range embedRequests {
		wg.Add(1)
		ctxTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		go func(ctx context.Context, ereq *embedRequestWrapper) {
			defer wg.Done()
			resp, err := p.client.V2.Embed(ctx, ereq.Request)
			if err == nil {
				embedRespMu.Lock()
				embedResponses = append(embedResponses, &embedResponseWrapper{
					Title:    ereq.Title,
					Chunks:   ereq.Chunks,
					Response: resp,
				})
				embedRespMu.Unlock()
			}
		}(ctxTimeout, ereq)
	}
	wg.Wait()

	docEmbeddings := make([]*api.DocumentEmbedding, 0, len(embedResponses))
	for _, eresp := range embedResponses {
		vectors := make([][]float32, 0, len(eresp.Response.Embeddings.Float))
		for _, cohereVector := range eresp.Response.Embeddings.Float {
			vector := make([]float32, 0, len(cohereVector))
			for _, f64 := range cohereVector {
				vector = append(vector, float32(f64))
			}
			vectors = append(vectors, vector)
		}
		docEmbeddings = append(docEmbeddings, &api.DocumentEmbedding{
			Title:  eresp.Title,
			Chunks: eresp.Chunks,
			Values: vectors,
		})
	}

	return docEmbeddings, nil
}

func (p CohereProvider) GetDimensions() uint {
	return 1024
}

func (p CohereProvider) Rerank(ctx context.Context, req api.RerankRequest) (*api.RerankResponse, error) {
	if req.Query == "" {
		return nil, fmt.Errorf("rerank request failed: missing parameter 'query' in request")
	}

	if len(req.Documents) == 0 {
		return nil, fmt.Errorf("rerank request failed: missing parameter 'documents' in request")
	}

	returnDocuments := true
	coReq := &cohere.V2RerankRequest{
		Query:           req.Query,
		Documents:       req.Documents,
		Model:           "rerank-v3.5",
		ReturnDocuments: &returnDocuments,
	}

	if req.ModelName != "" {
		coReq.Model = req.ModelName
	}

	if req.Limit != 0 {
		coReq.TopN = &req.Limit
	}

	resp, err := p.client.V2.Rerank(ctx, coReq)
	if err != nil {
		return nil, fmt.Errorf("rerank request failed: %w", err)
	}

	scoredDocs := make([]*api.ScoredDocument, 0, len(resp.Results))
	for _, result := range resp.Results {
		if result.RelevanceScore >= api.RerankScoreThreshold {
			scoredDocs = append(scoredDocs, &api.ScoredDocument{
				Content: result.Document.Text,
				Score:   result.RelevanceScore,
			})
		}
	}

	return &api.RerankResponse{
		Query:     req.Query,
		Documents: scoredDocs,
		ModelName: coReq.Model,
	}, nil
}

func (p CohereProvider) parseRequestHistory(h []*api.ChatMessage) cohere.ChatMessages {
	messages := make([]*cohere.ChatMessageV2, 0, len(h))
	for _, chatMsg := range h {
		var coMsg *cohere.ChatMessageV2
		switch chatMsg.Role {
		case api.RoleUser:
			coMsg = &cohere.ChatMessageV2{
				Role: "user",
				User: &cohere.UserMessage{Content: &cohere.UserMessageContent{
					String: chatMsg.Content,
				}},
			}
		case api.RoleAssistant:
			coMsg = &cohere.ChatMessageV2{
				Role: "assistant",
				Assistant: &cohere.AssistantMessage{Content: &cohere.AssistantMessageContent{
					String: chatMsg.Content,
				}},
			}
		default:
			slog.Warn("failed to parse chat message from history", "role", chatMsg.Role, "content", chatMsg.Content, "err", "unrecognized role")
			continue
		}

		messages = append(messages, coMsg)
	}

	return messages
}

type CohereCompletionStream struct {
	stream *coherecore.Stream[cohere.StreamedChatResponseV2]
}

func (s CohereCompletionStream) Recv() (string, error) {
	for {
		resp, err := s.stream.Recv()

		slog.Info("cohere", "resp", resp, "err", err)

		if err != nil {
			return "", err
		}

		if resp.ContentDelta != nil {
			return *resp.ContentDelta.Delta.Message.Content.Text, nil
		}
	}
}

func (s CohereCompletionStream) Close() error {
	return s.stream.Close()
}
