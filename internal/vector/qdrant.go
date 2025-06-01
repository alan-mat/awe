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

	"github.com/alan-mat/awe/internal/api"
	"github.com/qdrant/go-client/qdrant"
)

type QdrantStore struct {
	client     *qdrant.Client
	host       string
	port       int
	waitUpsert bool
}

func NewQdrantStore(host string, port int) (*QdrantStore, error) {
	c, err := qdrant.NewClient(&qdrant.Config{
		Host: host,
		Port: port,
	})
	if err != nil {
		return nil, err
	}

	s := &QdrantStore{
		client:     c,
		host:       host,
		port:       port,
		waitUpsert: true,
	}
	return s, nil
}

func NewQdrantStoreDefault() (*QdrantStore, error) {
	return NewQdrantStore("localhost", 6334)
}

func (s QdrantStore) CollectionExists(ctx context.Context, collectionName string) (bool, error) {
	return s.client.CollectionExists(ctx, collectionName)
}

func (s QdrantStore) CreateCollection(ctx context.Context, collection Collection) error {
	err := s.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: collection.Name,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     uint64(collection.Dimensions),
			Distance: qdrant.Distance_Cosine,
		}),
	})
	return err
}

func (s QdrantStore) Upsert(ctx context.Context, collectionName string, points []*Point) error {
	upsertPoints := make([]*qdrant.PointStruct, 0, len(points))
	for _, point := range points {
		upsertPoints = append(upsertPoints, &qdrant.PointStruct{
			Id:      qdrant.NewIDUUID(point.ID),
			Vectors: qdrant.NewVectors(point.Vector...),
			Payload: qdrant.NewValueMap(point.Payload),
		})
	}

	_, err := s.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: collectionName,
		Wait:           &s.waitUpsert,
		Points:         upsertPoints,
	})

	return err
}

func (s QdrantStore) Query(ctx context.Context, params *QueryParams) ([]*api.ScoredDocument, error) {
	queryPoints := &qdrant.QueryPoints{
		CollectionName: params.collection,
		Query:          qdrant.NewQuery(params.query...),
		WithPayload:    qdrant.NewWithPayload(params.withPayload),
	}

	if params.limit > 0 {
		limit := uint64(params.limit)
		queryPoints.Limit = &limit
	}

	if len(params.filters) > 0 {
		conds := make([]*qdrant.Condition, 0, len(params.filters))
		for _, filter := range params.filters {
			conds = append(conds, qdrant.NewMatch(filter.Key, filter.Value))
		}

		filter := &qdrant.Filter{
			Must: conds,
		}
		queryPoints.Filter = filter
	}

	res, err := s.client.Query(ctx, queryPoints)
	if err != nil {
		return nil, err
	}

	scoredDocs := make([]*api.ScoredDocument, 0, len(res))
	for _, sp := range res {
		payload := make(map[string]string)
		for k, v := range sp.Payload {
			if textValue := v.GetStringValue(); textValue != "" {
				payload[k] = textValue
			}
		}

		doc := &api.ScoredDocument{
			Score: float64(sp.Score),
		}

		if content, ok := payload["text"]; ok {
			doc.Content = content
		}
		if title, ok := payload["title"]; ok {
			doc.Title = title
		}
		if url, ok := payload["source_url"]; ok {
			doc.Url = url
		}

		scoredDocs = append(scoredDocs, doc)
	}

	return scoredDocs, nil
}

func (s QdrantStore) Close() error {
	return s.client.Close()
}
