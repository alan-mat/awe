package provider

import (
	"context"
	"errors"
)

var (
	ErrInvalidProviderType = errors.New("no provider found for given type")
)

const (
	LMProviderTypeOpenAI = iota
	LMProviderTypeGemini
	LMProviderTypeCohere
	LMProviderTypeOllama
)

const (
	EmbedProviderTypeGemini = iota
	EmbedProviderTypeJinaAI
	EmbedProviderTypeCohere
)

const (
	DocumentParseProviderTypeMistral = iota
)

const (
	DocumentSegmentProviderTypeJinaAI = iota
)

const (
	RerankerTypeCohere = iota
)

type LMProviderType int
type EmbedProviderType int
type DocumentParseProviderType int
type DocumentSegmentProviderType int

type RerankerType int

type LMProvider interface {
	CreateCompletionStream(context.Context, CompletionRequest) (CompletionStream, error)
}

func NewLMProvider(t LMProviderType) (LMProvider, error) {
	switch t {
	case LMProviderTypeOpenAI:
		return NewOpenAIProvider(), nil
	case LMProviderTypeGemini:
		return NewGeminiProvider(), nil
	case LMProviderTypeCohere:
		return NewCohereProvider(), nil
	case LMProviderTypeOllama:
		return NewOllamaProvider(), nil
	default:
		return nil, ErrInvalidProviderType
	}
}

type EmbedProvider interface {
	EmbedQuery(ctx context.Context, q string) ([]float32, error)
	EmbedDocuments(ctx context.Context, docs []*EmbedDocumentRequest) ([]*DocumentEmbedding, error)

	GetDimensions() uint
}

func NewEmbedProvider(t EmbedProviderType) (EmbedProvider, error) {
	switch t {
	case EmbedProviderTypeGemini:
		return NewGeminiProvider(), nil
	case EmbedProviderTypeJinaAI:
		return NewJinaAIProvider(), nil
	case EmbedProviderTypeCohere:
		return NewCohereProvider(), nil
	default:
		return nil, ErrInvalidProviderType
	}
}

type DocumentParseProvider interface {
	Parse(ctx context.Context, base64file string) (*DocumentContent, error)
}

func NewDocumentParseProvider(t DocumentParseProviderType) (DocumentParseProvider, error) {
	switch t {
	case DocumentParseProviderTypeMistral:
		return NewMistralProvider(), nil
	default:
		return nil, ErrInvalidProviderType
	}
}

type DocumentSegmentProvider interface {
	ChunkDocument(ctx context.Context, doc *DocumentContent) ([]string, error)
}

func NewDocumentSegmentProvider(t DocumentSegmentProviderType) (DocumentSegmentProvider, error) {
	switch t {
	case DocumentSegmentProviderTypeJinaAI:
		return NewJinaAIProvider(), nil
	default:
		return nil, ErrInvalidProviderType
	}
}

type Reranker interface {
	Rerank(ctx context.Context, req RerankRequest) (*RerankResponse, error)
}

func NewReranker(t RerankerType) (Reranker, error) {
	switch t {
	case RerankerTypeCohere:
		return NewCohereProvider(), nil
	default:
		return nil, ErrInvalidProviderType
	}
}
