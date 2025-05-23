package jina

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"regexp"
	"strings"

	"github.com/alan-mat/awe/internal/api"
	"github.com/alan-mat/awe/internal/http"
	"golang.org/x/sync/errgroup"
)

const (
	Endpoint                = "https://api.jina.ai"
	SegmentMaxContentLength = 64000
	EmbedItemsMaxLength     = 2048
)

type segmentResponse struct {
	NumTokens int    `json:"num_tokens"`
	Tokenizer string `json:"tokenizer"`
	Usage     struct {
		Tokens int `json:"tokens"`
	} `json:"usage"`
	NumChunks      int      `json:"num_chunks"`
	ChunkPositions [][]int  `json:"chunk_positions"`
	Chunks         []string `json:"chunks"`
}

type embeddingResponse struct {
	Model     string `json:"model"`
	UsageInfo struct {
		TotalTokens  int `json:"total_tokens"`
		PromptTokens int `json:"prompt_tokens"`
	} `json:"usage"`
	Data []struct {
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

type JinaAIProvider struct {
	client     http.Client
	vectorDims uint
}

func New() *JinaAIProvider {
	c := http.NewClient(
		Endpoint,
		http.WithMaxRetries(3),
		http.WithApiKey(os.Getenv("JINA_API_KEY")),
	)
	p := &JinaAIProvider{
		client:     c,
		vectorDims: 1024,
	}
	return p
}

func (p JinaAIProvider) ChunkDocument(ctx context.Context, doc *api.DocumentContent) ([]string, error) {
	contents := p.splitContentLen(SegmentMaxContentLength, doc)

	responses := make([]*segmentResponse, 0, len(contents))

	var g errgroup.Group
	for _, c := range contents {
		g.Go(func() error {
			resp, err := p.requestSegmenter(c)
			if err == nil {
				responses = append(responses, resp)
			}
			return err
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	regex := regexp.MustCompile(`\w+`)
	chunks := make([]string, 0, len(responses))
	var acc string
	for _, resp := range responses {
		for _, c := range resp.Chunks {
			if !regex.MatchString(c) {
				// no words, skip it
				continue
			}

			if strings.TrimSpace(c) == "[^0]" {
				// ignore this
				continue
			}

			acc += c
			if strings.HasPrefix(strings.TrimSpace(c), "#") {
				// interpret # as markdown headings
				continue
			} else {
				chunks = append(chunks, strings.TrimSpace(acc))
				acc = ""
			}
		}
	}

	return chunks, nil
}

func (p JinaAIProvider) EmbedQuery(ctx context.Context, q string) ([]float32, error) {
	resp, err := p.requestEmbedding([]string{q})
	if err != nil {
		return nil, err
	}

	if len(resp.Data) == 0 {
		return nil, errors.New("failed to deserialize embeddings")
	}

	return resp.Data[0].Embedding, nil
}

func (p JinaAIProvider) EmbedDocuments(ctx context.Context, docs []*api.EmbedDocumentRequest) ([]*api.DocumentEmbedding, error) {
	docs = p.splitDocsReqLen(EmbedItemsMaxLength, docs)
	embeddings := make([]*api.DocumentEmbedding, 0, len(docs))

	for _, doc := range docs {
		slog.Info("embedding document", "name", doc.Title, "chunks", len(doc.Chunks))
		total := 0
		largest := 0
		for _, c := range doc.Chunks {
			total += len(c)
			if len(c) > largest {
				largest = len(c)
			}
		}
		slog.Info("msg", "largest chunk", largest, "total", total)

		resp, err := p.requestEmbedding(doc.Chunks)
		if err != nil {
			slog.Error("error", "err", err)
			return nil, err
		}

		vals := make([][]float32, len(resp.Data))
		for _, e := range resp.Data {
			vals[e.Index] = e.Embedding
		}

		docEmbedding := &api.DocumentEmbedding{
			Title:  doc.Title,
			Chunks: doc.Chunks,
			Values: vals,
		}
		embeddings = append(embeddings, docEmbedding)
	}

	return embeddings, nil
}

func (p JinaAIProvider) GetDimensions() uint {
	return p.vectorDims
}

func (p JinaAIProvider) requestSegmenter(content string) (*segmentResponse, error) {
	requestData := map[string]any{
		"return_chunks":    true,
		"max_chunk_length": 768,
		"content":          content,
	}

	resp, err := p.client.Request(http.MethodPost, "/v1/segment", requestData)
	if err != nil {
		return nil, err
	}

	jsonData, err := json.Marshal(resp)
	if err != nil {
		return nil, err
	}

	var segmentResponse segmentResponse
	err = json.Unmarshal(jsonData, &segmentResponse)
	if err != nil {
		return nil, err
	}

	return &segmentResponse, nil
}

func (p JinaAIProvider) requestEmbedding(input []string) (*embeddingResponse, error) {
	requestData := map[string]any{
		"input":      input,
		"model":      "jina-embeddings-v3",
		"task":       "retrieval.passage",
		"dimensions": p.vectorDims,
	}

	resp, err := p.client.Request(http.MethodPost, "/v1/embeddings", requestData)
	if err != nil {
		return nil, err
	}

	jsonData, err := json.Marshal(resp)
	if err != nil {
		return nil, err
	}

	var embeddingResponse embeddingResponse
	err = json.Unmarshal(jsonData, &embeddingResponse)
	if err != nil {
		return nil, err
	}

	return &embeddingResponse, nil
}

func (p JinaAIProvider) splitContentLen(maxLen int, doc *api.DocumentContent) []string {
	cts := make([]string, 0, 1)
	full := doc.Text()

	if len(full) < maxLen {
		cts = append(cts, full)
		return cts
	}

	acc := ""
	for _, page := range doc.Pages {
		if (len(acc) + len(page.Text)) >= maxLen {
			cts = append(cts, acc)
			acc = ""
		}

		acc += page.Text
	}

	return cts
}

func (p JinaAIProvider) splitDocsReqLen(maxLen int, docs []*api.EmbedDocumentRequest) []*api.EmbedDocumentRequest {
	newDocs := make([]*api.EmbedDocumentRequest, 0, len(docs))

	for _, doc := range docs {
		if len(doc.Chunks) < maxLen {
			newDocs = append(newDocs, doc)
			continue
		}

		nParts := (len(doc.Chunks) / maxLen) + 1
		for i := range nParts {
			start, end := i*maxLen, (i+1)*maxLen
			if end > len(doc.Chunks) {
				end = len(doc.Chunks)
			}

			newDocs = append(newDocs, &api.EmbedDocumentRequest{
				Title:  doc.Title,
				Chunks: doc.Chunks[start:end],
			})
		}
	}

	return newDocs
}
