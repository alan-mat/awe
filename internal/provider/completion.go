package provider

type CompletionRequest struct {
	Query string
}

type CompletionStream interface {
	Recv() (string, error)
	Close() error
}
