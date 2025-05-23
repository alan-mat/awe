package api

type DocumentPage struct {
	Index int
	Text  string
}

type DocumentContent struct {
	Pages []DocumentPage
}

func (dc DocumentContent) Text() string {
	text := ""
	for _, page := range dc.Pages {
		text += page.Text
	}
	return text
}

type ScoredDocument struct {
	// Required
	Content string
	Score   float64

	// Optional
	Title string
	Url   string
}
