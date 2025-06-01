// Copyright 2025 Alan Matykiewicz
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to use,
// copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the
// Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES
// OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
// HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
// WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
// OTHER DEALINGS IN THE SOFTWARE.

package vector

import (
	"context"
	"errors"
	"fmt"

	"github.com/alan-mat/awe/internal/api"
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

	Query(ctx context.Context, params *QueryParams) ([]*api.ScoredDocument, error)

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

func CreatePoints(docs []*api.DocumentEmbedding) []*Point {
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
