package provider

type EmbedDocumentRequest struct {
	Title  string
	Chunks []string
}

type DocumentEmbedding struct {
	Title  string
	Chunks []string
	Values [][]float32
}
