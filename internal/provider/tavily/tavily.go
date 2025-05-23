package tavily

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/alan-mat/awe/internal/api"
	"github.com/alan-mat/awe/internal/http"
)

const (
	Endpoint           = "https://api.tavily.com"
	SearchDefaultLimit = 5
)

type SearchResponse struct {
	Query  string `json:"query"`
	Answer string `json:"answer"`
	Images []struct {
		Url         string `json:"url"`
		Description string `json:"description"`
	} `json:"images"`
	Results      []*SearchResult `json:"results"`
	ResponseTime float32         `json:"response_time"`
}

type SearchResult struct {
	Title   string  `json:"title"`
	Url     string  `json:"url"`
	Content string  `json:"content"`
	Score   float64 `json:"score"`
	Raw     string  `json:"raw_content"`
}

type TavilyProvider struct {
	client http.Client
}

func New() *TavilyProvider {
	c := http.NewClient(
		Endpoint,
		http.WithMaxRetries(3),
		http.WithApiKey(os.Getenv("TAVILY_API_KEY")),
	)
	p := &TavilyProvider{
		client: c,
	}
	return p
}

func (p TavilyProvider) Search(ctx context.Context, req api.WebSearchRequest) (*api.WebSearchResponse, error) {
	if req.Query == "" {
		return nil, fmt.Errorf("query must not be empty")
	}

	var limit int
	if req.Limit != 0 {
		limit = req.Limit
	} else {
		limit = SearchDefaultLimit
	}

	requestData := map[string]any{
		"query":                      req.Query,
		"topic":                      "general",
		"search":                     "basic",
		"max_results":                limit,
		"include_answer":             false,
		"include_raw_content":        false,
		"include_images":             false,
		"include_image_descriptions": false,
	}

	resp, err := p.client.Request(http.MethodPost, "/search", requestData)
	if err != nil {
		return nil, fmt.Errorf("web search request failed: %w", err)
	}

	jsonData, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize web search response: %w", err)
	}

	var searchResponse SearchResponse
	err = json.Unmarshal(jsonData, &searchResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize web search response: %w", err)
	}

	docs := make([]*api.ScoredDocument, 0, len(searchResponse.Results))
	for _, result := range searchResponse.Results {
		docs = append(docs, &api.ScoredDocument{
			Content: result.Content,
			Score:   result.Score,
			Title:   result.Title,
			Url:     result.Url,
		})
	}

	return &api.WebSearchResponse{
		Query:   searchResponse.Query,
		Results: docs,
	}, nil
}
