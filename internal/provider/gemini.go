package provider

import (
	"context"
	"encoding/json"
	"io"
	"iter"
	"os"

	"github.com/alan-mat/awe/internal/message"
	"google.golang.org/genai"
)

const geminiSegmentPrompt = `You are an expert document chunker, responsible for segmenting complex documents into semantically coherent chunks suitable for indexing in a vector database. Your goal is to create chunks that are informative and useful for semantic search. Follow these guidelines meticulously:

1.  **Semantic Coherence:** Maintain semantic meaning within each chunk. Avoid splitting sentences, paragraphs, or logical units of information across chunk boundaries. Ensure a smooth and natural flow of information within each chunk.

2.  **Minimum Length:** Each chunk must contain at least one complete sentence *or* a semantically relevant object (e.g., a table row with context, a code snippet with a description, a well-defined mathematical expression with an explanation). Avoid creating chunks that consist solely of headings, subheadings, isolated LaTeX formulas, or other content fragments lacking independent meaning.

3.  **Maximum Length:** Chunks must not exceed 768 characters in length (including spaces). If a semantic unit exceeds this limit, split it at the most logical break point while preserving as much context as possible in each resulting chunk.

4.  **Heading/Subheading Integration:** Always merge headings and subheadings with the immediately following sentence, paragraph, list, table or other semantically connected content. A chunk may not include only a heading or subheading. Include a line-break between a heading and the following content.

5.  **Artifact Removal & Repair:** Identify and remove any nonsensical artifacts or inconsistencies that may have resulted from document parsing (e.g., broken characters, redundant whitespace, misplaced punctuation, OCR errors). Repair minor grammatical errors or inconsistencies to improve readability and searchability.
`

type GeminiProvider struct {
	client     *genai.Client
	vectorDims *int32
}

func NewGeminiProvider() *GeminiProvider {
	// New methods might need error return
	// to handle error returns from client libs like genai
	c, _ := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:  os.Getenv("GEMINI_API_KEY"),
		Backend: genai.BackendGeminiAPI,
	})
	p := &GeminiProvider{
		client:     c,
		vectorDims: new(int32),
	}
	*(p.vectorDims) = 1536
	return p
}

func (p GeminiProvider) CreateCompletionStream(ctx context.Context, req CompletionRequest) (CompletionStream, error) {
	contents := p.parseRequestHistory(req.History)
	contents = append(contents, genai.NewContentFromText(req.Query, genai.RoleUser))

	config := &genai.GenerateContentConfig{}
	if req.SystemPrompt != "" {
		config.SystemInstruction = genai.NewContentFromText(req.SystemPrompt, "")
	}

	i := p.client.Models.GenerateContentStream(
		ctx,
		"gemini-2.0-flash",
		contents,
		config,
	)

	next, stop := iter.Pull2(i)
	return &GeminiCompletionStream{
		next: next,
		stop: stop,
	}, nil
}

func (p GeminiProvider) EmbedQuery(ctx context.Context, q string) ([]float32, error) {
	contents := genai.Text(q)

	config := &genai.EmbedContentConfig{
		TaskType:             "RETRIEVAL_QUERY",
		OutputDimensionality: p.vectorDims,
	}

	res, err := p.client.Models.EmbedContent(ctx, "gemini-embedding-exp-03-07", contents, config)
	if err != nil {
		return nil, err
	}

	vals := res.Embeddings[0].Values
	return vals, nil
}

func (p GeminiProvider) EmbedDocuments(ctx context.Context, docs []*EmbedDocumentRequest) ([]*DocumentEmbedding, error) {
	embeddings := make([]*DocumentEmbedding, 0, len(docs))

	for _, doc := range docs {
		contents := make([]*genai.Content, 0, len(doc.Chunks))
		for _, chunk := range doc.Chunks {
			content := genai.NewContentFromText(chunk, genai.RoleUser)
			contents = append(contents, content)
		}

		config := &genai.EmbedContentConfig{
			TaskType:             "RETRIEVAL_DOCUMENT",
			Title:                doc.Title,
			OutputDimensionality: p.vectorDims,
		}

		res, err := p.client.Models.EmbedContent(ctx, "gemini-embedding-exp-03-07", contents, config)
		if err != nil {
			return nil, err
		}

		values := make([][]float32, 0, len(res.Embeddings))
		for _, rEmbedding := range res.Embeddings {
			values = append(values, rEmbedding.Values)
		}

		docEmbed := &DocumentEmbedding{
			Title:  doc.Title,
			Values: values,
			Chunks: doc.Chunks,
		}
		embeddings = append(embeddings, docEmbed)
	}

	return embeddings, nil
}

func (p GeminiProvider) GetDimensions() uint {
	return uint(*p.vectorDims)
}

func (p GeminiProvider) ChunkDocument(ctx context.Context, doc *DocumentContent) ([]string, error) {
	content := doc.Text()

	schema := &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"chunks": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type: genai.TypeString,
				},
			},
		},
		Title:    "List of chunks.",
		Required: []string{"chunks"},
	}

	temperature := float32(0)
	reqConfig := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(geminiSegmentPrompt, ""),
		ResponseMIMEType:  "application/json",
		ResponseSchema:    schema,
		Temperature:       &temperature,
	}

	resp, err := p.client.Models.GenerateContent(
		ctx,
		"models/gemini-2.5-flash-preview-05-20",
		genai.Text(content),
		reqConfig,
	)
	if err != nil {
		return nil, err
	}

	var respChunks struct {
		Chunks []string `json:"chunks"`
	}
	respBytes := []byte(resp.Text())
	err = json.Unmarshal(respBytes, &respChunks)
	if err != nil {
		return nil, err
	}

	return respChunks.Chunks, nil
}

func (p GeminiProvider) parseRequestHistory(h []*message.Chat) []*genai.Content {
	contents := make([]*genai.Content, len(h))
	roleTypes := map[message.ChatRole]genai.Role{
		message.RoleUser:      genai.RoleUser,
		message.RoleAssistant: genai.RoleModel,
	}
	for i, m := range h {
		c := genai.NewContentFromText(m.Content, roleTypes[m.Role])
		contents[i] = c
	}
	return contents
}

type GeminiCompletionStream struct {
	next func() (*genai.GenerateContentResponse, error, bool)
	stop func()
}

func (s GeminiCompletionStream) Recv() (string, error) {
	res, err, valid := s.next()
	if !valid {
		//iterator is finished
		return "", io.EOF
	}

	if err != nil {
		return "", err
	}

	return res.Text(), nil
}

func (s GeminiCompletionStream) Close() error {
	s.stop()
	return nil
}
