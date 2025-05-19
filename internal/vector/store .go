package vector

import (
	"context"
	"errors"
	"fmt"

	"github.com/alan-mat/awe/internal/provider"
	"github.com/google/uuid"
)

var (
	ErrInvalidStoreType      = errors.New("no vector store found for given type")
	ErrFailedStoreInitialize = errors.New("failed to initialise vector store")
)

const (
	StoreTypeQdrant = iota
)

var storeTypeMap = map[string]StoreType{
	"qdrant": StoreTypeQdrant,
}

type StoreType int

type Store interface {
	CollectionExists(ctx context.Context, collectionName string) (bool, error)
	CreateCollection(ctx context.Context, collection Collection) error
	//DeleteCollection()

	Upsert(ctx context.Context, collectionName string, points []*Point) error
	//Delete()

	Query(ctx context.Context, params *QueryParams) ([]*ScoredPoint, error)

	Close() error
}

func NewStore(storeName string) (Store, error) {
	storeType, ok := storeTypeMap[storeName]
	if !ok {
		return nil, ErrInvalidStoreType
	}

	switch storeType {
	case StoreTypeQdrant:
		store, err := NewQdrantStoreDefault()
		if err != nil {
			return nil, fmt.Errorf("%e: %e", ErrFailedStoreInitialize, err)
		}

		return store, nil
	default:
		return nil, ErrInvalidStoreType
	}
}

type Collection struct {
	Name       string
	Dimensions uint
	//Distance   any
}

type Point struct {
	ID      string
	Vector  []float32
	Payload map[string]any
}

func CreatePoints(docs []*provider.DocumentEmbedding) []*Point {
	points := make([]*Point, 0, len(docs))
	for _, doc := range docs {
		for i := range len(doc.Chunks) {
			points = append(points, &Point{
				ID:     uuid.NewString(),
				Vector: doc.Values[i],
				Payload: map[string]any{
					"title": doc.Title,
					"text":  doc.Chunks[i],
				},
			})
		}
	}
	return points
}

type QueryMatch struct {
	Key   string
	Value string
}

type QueryParams struct {
	collection  string
	query       []float32
	withPayload bool
	limit       uint
	filters     []*QueryMatch
}

type QueryParamsOption func(*QueryParams)

func NewQueryParams(collection string, query []float32, opts ...QueryParamsOption) *QueryParams {
	qp := &QueryParams{
		collection:  collection,
		query:       query,
		withPayload: false,
		limit:       0,
		filters:     make([]*QueryMatch, 0),
	}

	for _, opt := range opts {
		opt(qp)
	}
	return qp
}

func WithPayload(w bool) QueryParamsOption {
	return func(qp *QueryParams) {
		qp.withPayload = w
	}
}

func WithLimit(limit uint) QueryParamsOption {
	return func(qp *QueryParams) {
		qp.limit = limit
	}
}

func WithFilter(filter *QueryMatch) QueryParamsOption {
	return func(qp *QueryParams) {
		qp.filters = append(qp.filters, filter)
	}
}

type ScoredPoint struct {
	ID      string
	Score   float32
	Payload map[string]string
}
