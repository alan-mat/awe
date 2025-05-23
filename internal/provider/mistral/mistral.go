package mistral

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/alan-mat/awe/internal/api"
	"github.com/alan-mat/awe/internal/http"
)

const (
	Endpoint = "https://api.mistral.ai"
)

type page struct {
	Index      int              `json:"index"`
	Markdown   string           `json:"markdown"`
	Images     []map[string]any `json:"images"`
	Dimensions map[string]any   `json:"dimensions"`
}

type usageInfo struct {
	PagesProcessed int `json:"pages_processed"`
	DocSizeBytes   int `json:"doc_size_bytes"`
}

type OCRResponse struct {
	Pages     []page    `json:"pages"`
	Model     string    `json:"model"`
	UsageInfo usageInfo `json:"usage_info"`
}

type MistralProvider struct {
	client http.Client
}

func New() *MistralProvider {
	c := http.NewClient(
		Endpoint,
		http.WithMaxRetries(3),
		http.WithApiKey(os.Getenv("MISTRAL_API_KEY")),
	)
	p := &MistralProvider{
		client: c,
	}
	return p
}

func (p MistralProvider) Parse(ctx context.Context, base64file string) (*api.DocumentContent, error) {
	documentUrl := map[string]any{
		"type":         "document_url",
		"document_url": fmt.Sprintf("data:application/pdf;base64,%s", base64file),
	}

	requestData := map[string]any{
		"model":    "mistral-ocr-latest",
		"document": documentUrl,
	}

	resp, err := p.client.Request(http.MethodPost, "/v1/ocr", requestData)
	if err != nil {
		return nil, err
	}

	jsonData, err := json.Marshal(resp)
	if err != nil {
		return nil, err
	}

	var ocrResponse OCRResponse
	err = json.Unmarshal(jsonData, &ocrResponse)
	if err != nil {
		return nil, err
	}

	doc := &api.DocumentContent{
		Pages: make([]api.DocumentPage, 0, len(ocrResponse.Pages)),
	}
	for _, page := range ocrResponse.Pages {
		dp := api.DocumentPage{
			Index: page.Index,
			Text:  page.Markdown,
		}
		doc.Pages = append(doc.Pages, dp)
	}

	return doc, nil
}
