package provider

import (
	"context"
	"fmt"
)

var (
	ErrInvalidLMProviderType = fmt.Errorf("no lmprovider found for given type")
)

const (
	LMProviderTypeOpenAI = iota
	LMProviderTypeGemini
)

type LMProviderType int

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
