package provider

import (
	"context"
	"errors"
)

var (
	ErrInvalidLMProviderType    = errors.New("no lmprovider found for given type")
	ErrInvalidEmbedProviderType = errors.New("no embeddings provider found for given type")
)

const (
	LMProviderTypeOpenAI = iota
	LMProviderTypeGemini
)

const (
	EmbedProviderTypeGemini = iota
)

type LMProviderType int
type EmbedProviderType int

type LMProvider interface {
	CreateCompletionStream(context.Context, CompletionRequest) (CompletionStream, error)
}

func NewLMProvider(t LMProviderType) (LMProvider, error) {
	switch t {
	case LMProviderTypeOpenAI:
		return NewOpenAIProvider(), nil
	case LMProviderTypeGemini:
		return NewGeminiProvider(), nil
	default:
		return nil, ErrInvalidLMProviderType
	}
}

type EmbedProvider interface {
	EmbedQuery(ctx context.Context, q string) ([]float32, error)
	EmbedDocuments(ctx context.Context, docs []*EmbedDocumentRequest) ([]*DocumentEmbedding, error)
}

func NewEmbedProvider(t EmbedProviderType) (EmbedProvider, error) {
	switch t {
	case EmbedProviderTypeGemini:
		return NewGeminiProvider(), nil
	default:
		return nil, ErrInvalidEmbedProviderType
	}
}
