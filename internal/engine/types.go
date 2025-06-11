package engine

import (
	"context"
	"time"
)

type Module interface {
	Operator(name string) (Executer, error)
}

type ModuleRegistry interface {
	Get(name string) (Module, bool)
}

type GenerationEvent struct {
	Type GenerationEventType

	Content string
	Error   error
}

type GenerationEventType string

const (
	GenerationEventContentStart GenerationEventType = "content_start"
	GenerationEventContentDelta GenerationEventType = "content_delta"
	GenerationEventContentStop  GenerationEventType = "content_stop"
	GenerationEventComplete     GenerationEventType = "complete"
	GenerationEventError        GenerationEventType = "error"
)

func ContentStartEvent() GenerationEvent {
	return GenerationEvent{
		Type: GenerationEventContentStart,
	}
}

func ContentDeltaEvent(content string) GenerationEvent {
	return GenerationEvent{
		Type:    GenerationEventContentDelta,
		Content: content,
	}
}

func ContentStopEvent() GenerationEvent {
	return GenerationEvent{
		Type: GenerationEventContentStop,
	}
}

func GenerationCompleteEvent() GenerationEvent {
	return GenerationEvent{
		Type: GenerationEventComplete,
	}
}

func GenerationErrorEvent(err error) GenerationEvent {
	return GenerationEvent{
		Type:  GenerationEventError,
		Error: err,
	}
}

type Transport interface {
	MessageStream(id string) MessageStream
	SetTrace(ctx context.Context, trace *Trace) error
	GetTrace(ctx context.Context, traceId string) (*Trace, error)
}

type MessageStream interface {
	Send(ctx context.Context, payload MessageStreamPayload) error
	Recv(ctx context.Context) (*MessageStreamPayload, error)

	ID() string
}

type MessageStreamPayload struct {
	Type TransportMessageType

	Status   TransportStatus
	Content  string
	Document Document
	Error    error
}

type TransportMessageType string

const (
	TransportMessageStatus   TransportMessageType = "status"
	TransportMessageContent  TransportMessageType = "content"
	TransportMessageDocument TransportMessageType = "document"
	TransportMessageComplete TransportMessageType = "complete"
	TransportMessageError    TransportMessageType = "error"
)

type TransportStatus string

const (
	TransportStatusStreamStart  TransportStatus = "stream_start"
	TransportStatusContentStart TransportStatus = "content_start"
	TransportStatusContentEnd   TransportStatus = "content_end"
	TransportStatusStreamEnd    TransportStatus = "stream_end"
)

type Document struct {
	Title   string
	Content string
	Source  string
}

type Trace struct {
	ID          string
	Status      TraceStatus
	StartedAt   int64
	CompletedAt int64
	Query       Query
	Caller      CallerMeta

	// FailReason contains the error message related to the failing
	// of this trace. This field must be nil, unless Status is set to TraceStatusFailed.
	FailReason *string
}

func NewTrace(id string, query Query, caller CallerMeta) *Trace {
	return &Trace{
		ID:        id,
		Status:    TraceStatusRunning,
		StartedAt: time.Now().UnixNano(),
		Query:     query,
		Caller:    caller,
	}
}

func (t *Trace) Complete() {
	if t.Status != TraceStatusRunning {
		return
	}

	t.CompletedAt = time.Now().UnixNano()
	t.Status = TraceStatusCompleted
}

func (t *Trace) Fail(reason error) {
	if t.Status != TraceStatusRunning {
		return
	}

	t.CompletedAt = time.Now().UnixNano()
	t.Status = TraceStatusFailed

	errString := reason.Error()
	t.FailReason = &errString
}

type TraceStatus int

const (
	TraceStatusUnspecified = iota
	TraceStatusRunning
	TraceStatusCompleted
	TraceStatusFailed
)
