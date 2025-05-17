package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/alan-mat/awe/internal/http"
)

const (
	MistralEndpoint = "https://api.mistral.ai"
)

type mistralPage struct {
	Index      int              `json:"index"`
	Markdown   string           `json:"markdown"`
	Images     []map[string]any `json:"images"`
	Dimensions map[string]any   `json:"dimensions"`
}

type mistralUsageInfo struct {
	PagesProcessed int `json:"pages_processed"`
	DocSizeBytes   int `json:"doc_size_bytes"`
}

type MistralOCRResponse struct {
	Pages     []mistralPage    `json:"pages"`
	Model     string           `json:"model"`
	UsageInfo mistralUsageInfo `json:"usage_info"`
}

type MistralProvider struct {
	client http.Client
}

func NewMistralProvider() *MistralProvider {
	c := http.NewClient(
		MistralEndpoint,
		http.WithMaxRetries(3),
		http.WithApiKey(os.Getenv("MISTRAL_API_KEY")),
	)
	p := &MistralProvider{
		client: c,
	}
	return p
}

func (p MistralProvider) Parse(ctx context.Context, base64file string) (*DocumentContent, error) {
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

	var ocrResponse MistralOCRResponse
	err = json.Unmarshal(jsonData, &ocrResponse)
	if err != nil {
		return nil, err
	}

	doc := &DocumentContent{
		Pages: make([]DocumentPage, 0, len(ocrResponse.Pages)),
	}
	for _, page := range ocrResponse.Pages {
		dp := DocumentPage{
			Index: page.Index,
			Text:  page.Markdown,
		}
		doc.Pages = append(doc.Pages, dp)
	}

	return doc, nil
}
