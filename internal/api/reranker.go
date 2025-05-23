package api

const RerankScoreThreshold = 0.5

type RerankRequest struct {
	// Required params
	Query     string
	Documents []string

	// Optional params
	Limit     int
	ModelName string
}

type RerankResponse struct {
	Query     string
	Documents []*ScoredDocument

	ModelName string
}
