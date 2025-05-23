package provider

import (
	"context"
	"errors"

	"github.com/alan-mat/awe/internal/api"
	cohere "github.com/alan-mat/awe/internal/provider/cohere"
	"github.com/alan-mat/awe/internal/provider/gemini"
	"github.com/alan-mat/awe/internal/provider/jina"
	"github.com/alan-mat/awe/internal/provider/mistral"
	"github.com/alan-mat/awe/internal/provider/ollama"
	"github.com/alan-mat/awe/internal/provider/openai"
	"github.com/alan-mat/awe/internal/provider/tavily"
)

var (
	ErrInvalidProviderType = errors.New("no provider found for given type")
)

const (
	LMTypeOpenai = iota
	LMTypeGemini
	LMTypeCohere
	LMTypeOllama
)

const (
	EmbedderTypeGemini = iota
	EmbedderTypeJina
	EmbedderTypeCohere
	EmbedderTypeOpenai
)

const (
	DocParserTypeMistral = iota
)

const (
	SegmenterTypeJina = iota
	SegmenterTypeGemini
)

const (
	RerankerTypeCohere = iota
)

const (
	WebSearcherTypeTavily = iota
)

type LMType int
type EmbedderType int
type DocParserType int
type SegmenterType int
type RerankerType int
type WebSearcherType int

type LM interface {
	Generate(ctx context.Context, req api.GenerationRequest) (api.CompletionStream, error)
	Chat(ctx context.Context, req api.ChatRequest) (api.CompletionStream, error)
}

func NewLM(t LMType) (LM, error) {
	switch t {
	case LMTypeOpenai:
		return openai.New(), nil
	case LMTypeGemini:
		return gemini.New(), nil
	case LMTypeCohere:
		return cohere.New(), nil
	case LMTypeOllama:
		return ollama.New(), nil
	default:
		return nil, ErrInvalidProviderType
	}
}

type Embedder interface {
	EmbedQuery(ctx context.Context, q string) ([]float32, error)
	EmbedDocuments(ctx context.Context, docs []*api.EmbedDocumentRequest) ([]*api.DocumentEmbedding, error)

	GetDimensions() uint
}

func NewEmbedder(t EmbedderType) (Embedder, error) {
	switch t {
	case EmbedderTypeGemini:
		return gemini.New(), nil
	case EmbedderTypeJina:
		return jina.New(), nil
	case EmbedderTypeCohere:
		return cohere.New(), nil
	case EmbedderTypeOpenai:
		return openai.New(), nil
	default:
		return nil, ErrInvalidProviderType
	}
}

type DocParser interface {
	Parse(ctx context.Context, base64file string) (*api.DocumentContent, error)
}

func NewDocParser(t DocParserType) (DocParser, error) {
	switch t {
	case DocParserTypeMistral:
		return mistral.New(), nil
	default:
		return nil, ErrInvalidProviderType
	}
}

type Segmenter interface {
	ChunkDocument(ctx context.Context, doc *api.DocumentContent) ([]string, error)
}

func NewSegmenter(t SegmenterType) (Segmenter, error) {
	switch t {
	case SegmenterTypeJina:
		return jina.New(), nil
	case SegmenterTypeGemini:
		return gemini.New(), nil
	default:
		return nil, ErrInvalidProviderType
	}
}

type Reranker interface {
	Rerank(ctx context.Context, req api.RerankRequest) (*api.RerankResponse, error)
}

func NewReranker(t RerankerType) (Reranker, error) {
	switch t {
	case RerankerTypeCohere:
		return cohere.New(), nil
	default:
		return nil, ErrInvalidProviderType
	}
}

type WebSearcher interface {
	Search(ctx context.Context, req api.WebSearchRequest) (*api.WebSearchResponse, error)
}

func NewWebSearcher(t WebSearcherType) (WebSearcher, error) {
	switch t {
	case WebSearcherTypeTavily:
		return tavily.New(), nil
	default:
		return nil, ErrInvalidProviderType
	}
}
