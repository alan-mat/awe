package engine_test

import (
	"context"
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/alan-mat/awe/internal/engine"
	"github.com/alan-mat/awe/internal/llm"
)

func TestTransportGenerationMiddleware(t *testing.T) {
	query := "My query"
	workflow := createWorkflowWithGenerator("wf-1", 3)
	c := createContext("wf-1", query)
	inv := engine.NewInvoker(c)
	transport := newMockTransport()
	err := inv.Call(workflow.Executer(transport), engine.DefaultParams())
	if err != nil {
		t.Errorf("expected not-nil error, got '%v'", err)
	}

	expected := engine.NewState(engine.TextQuery(query))
	expected.AddContents(engine.ContentsFromMessages(
		llm.TextMessage(llm.MessageRoleAssistant, "Message from op1"),
		llm.TextMessage(llm.MessageRoleAssistant, "Message from op2"),
		llm.TextMessage(llm.MessageRoleAssistant, "Message from op3"),
	))

	if !reflect.DeepEqual(inv.State(), expected) {
		t.Errorf("invalid state after workflow execution, expected '%+v', got '%+v'", expected, inv.State())
	}

	ms := transport.MessageStream(c.TaskId())
	ctx := context.Background()
	for {
		payload, err := ms.Recv(ctx)
		if err != nil {
			t.Errorf("received error when reading from message stream: %v", err)
		}
		// t.Logf("got payload: %+v", payload)

		if payload.Type == engine.TransportMessageStatus &&
			payload.Status == engine.TransportStatusContentEnd {
			break
		}
	}
}

type mockTransport struct {
	streams map[string]*mockMessageStream
	traces  map[string]*engine.Trace
}

func newMockTransport() *mockTransport {
	return &mockTransport{
		streams: make(map[string]*mockMessageStream),
		traces:  make(map[string]*engine.Trace),
	}
}

func (t mockTransport) MessageStream(id string) engine.MessageStream {
	ms, ok := t.streams[id]
	if !ok {
		ms = newMockMessageStream(id)
		t.streams[id] = ms
		return ms
	}
	return ms
}

func (t mockTransport) SetTrace(ctx context.Context, trace *engine.Trace) error {
	t.traces[trace.ID] = trace
	return nil
}

func (t mockTransport) GetTrace(ctx context.Context, traceId string) (*engine.Trace, error) {
	trace, ok := t.traces[traceId]
	if !ok {
		return nil, errors.New("trace not found")
	}
	return trace, nil
}

type mockMessageStream struct {
	id  string
	buf []*engine.MessageStreamPayload
}

func newMockMessageStream(id string) *mockMessageStream {
	return &mockMessageStream{
		id:  id,
		buf: make([]*engine.MessageStreamPayload, 0),
	}
}

func (ms *mockMessageStream) Send(ctx context.Context, payload engine.MessageStreamPayload) error {
	ms.buf = append(ms.buf, &payload)
	return nil
}

func (ms *mockMessageStream) Recv(ctx context.Context) (*engine.MessageStreamPayload, error) {
	if len(ms.buf) == 0 {
		return nil, io.EOF
	}
	msg := ms.buf[0]
	ms.buf = ms.buf[1:]
	return msg, nil
}

func (ms mockMessageStream) ID() string {
	return ms.id
}
