package vector

type Store interface {
	CollectionExists()
	CreateCollection()
	//DeleteCollection()

	Upsert()
	//Delete()

	Query()

	Close()
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
