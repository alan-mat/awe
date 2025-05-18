package vector

import (
	"context"

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

func (s QdrantStore) Query(ctx context.Context, params *QueryParams) ([]*ScoredPoint, error) {
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

	scoredPoints := make([]*ScoredPoint, 0, len(res))
	for _, sp := range res {
		payload := make(map[string]string)
		for k, v := range sp.Payload {
			if textValue := v.GetStringValue(); textValue != "" {
				payload[k] = textValue
			}
		}

		scoredPoints = append(scoredPoints, &ScoredPoint{
			ID:      sp.Id.GetUuid(),
			Score:   sp.Score,
			Payload: payload,
		})
	}

	return scoredPoints, nil
}

func (s QdrantStore) Close() error {
	return s.client.Close()
}
