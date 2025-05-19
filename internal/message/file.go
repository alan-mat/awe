package message

// FileContent contains the name of a file and
// its base64 encoded contents.
type FileContent struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}
