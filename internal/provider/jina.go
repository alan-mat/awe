package provider

import (
	"context"
	"encoding/json"
	"os"

	"github.com/alan-mat/awe/internal/http"
	"golang.org/x/sync/errgroup"
)

const (
	JinaAIEndpoint              = "https://api.jina.ai"
	JinaSegmentMaxContentLength = 64000
)

type jinaSegmentResponse struct {
	NumTokens int    `json:"num_tokens"`
	Tokenizer string `json:"tokenizer"`
	Usage     struct {
		Tokens int `json:"tokens"`
	} `json:"usage"`
	NumChunks      int      `json:"num_chunks"`
	ChunkPositions [][]int  `json:"chunk_positions"`
	Chunks         []string `json:"chunks"`
}

type JinaAIProvider struct {
	client http.Client
}

func NewJinaAIProvider() *JinaAIProvider {
	c := http.NewClient(
		JinaAIEndpoint,
		http.WithMaxRetries(3),
		http.WithApiKey(os.Getenv("JINA_API_KEY")),
	)
	p := &JinaAIProvider{
		client: c,
	}
	return p
}

func (p JinaAIProvider) ChunkDocument(ctx context.Context, doc *DocumentContent) ([]string, error) {
	contents := p.splitContentLen(JinaSegmentMaxContentLength, doc)

	responses := make([]*jinaSegmentResponse, 0, len(contents))

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

	chunks := make([]string, 0, len(responses))
	for _, resp := range responses {
		chunks = append(chunks, resp.Chunks...)
	}

	return chunks, nil
}

func (p JinaAIProvider) requestSegmenter(content string) (*jinaSegmentResponse, error) {
	requestData := map[string]any{
		"return_chunks":    true,
		"max_chunk_length": 1000,
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

	var segmentResponse jinaSegmentResponse
	err = json.Unmarshal(jsonData, &segmentResponse)
	if err != nil {
		return nil, err
	}

	return &segmentResponse, nil
}

func (p JinaAIProvider) splitContentLen(maxLen int, doc *DocumentContent) []string {
	cts := make([]string, 0, 1)
	full := doc.Text()

	if len(full) < maxLen {
		cts = append(cts, full)
		return cts
	}

	nParts := (len(full) / (maxLen + 1)) * 2
	nPages := len(doc.Pages) / nParts

	start, end := 0, nPages
	for _ = range nParts {
		t := ""
		ps := doc.Pages[start:end]

		for _, p := range ps {
			t += p.Text
		}
		cts = append(cts, t)

		start = end
		if (start + nPages) > len(doc.Pages) {
			end = len(doc.Pages)
		} else {
			end = start + nPages
		}
	}

	return cts
}
