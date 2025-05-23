package api

type WebSearchRequest struct {
	// Required
	Query string

	// Optional
	Limit int
}

type WebSearchResponse struct {
	Query   string
	Results []*ScoredDocument
}
