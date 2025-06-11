package engine_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/alan-mat/awe/internal/engine"
	"github.com/alan-mat/awe/internal/llm"
)

func TestWorkflowExecute(t *testing.T) {
	query := "My query"
	workflow := createWorkflow("wf-1", 3)
	c := createContext("wf-1", "My query")
	inv := engine.NewInvoker(c)
	err := inv.Call(workflow.Executer(nil), engine.DefaultParams())
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
}

func createWorkflow(workflowId string, nodeCnt int) *engine.Workflow {
	nodes := make([]*engine.WorkflowNode, 0)
	m := mockModule{}
	for i := range nodeCnt {
		n := engine.NewWorkflowNode(m, fmt.Sprintf("op%d", i+1), engine.WorkflowNodeLinear)
		nodes = append(nodes, n)
	}

	return engine.NewWorkflow(
		workflowId,
		"Test workflow",
		"mycollection",
		nodes,
	)
}

func createWorkflowWithGenerator(workflowId string, nodeCnt int) *engine.Workflow {
	nodes := make([]*engine.WorkflowNode, 0)
	m := mockModule{}
	for i := range nodeCnt {
		n := engine.NewWorkflowNode(m, fmt.Sprintf("op%d", i+1), engine.WorkflowNodeLinear)
		nodes = append(nodes, n)
	}
	nodes = append(nodes, engine.NewWorkflowNode(m, "mock_gen", engine.WorkflowNodeLinear))

	return engine.NewWorkflow(
		workflowId,
		"Generation workflow",
		"mycollection",
		nodes,
	)
}

func createContext(workflowId, query string) engine.Context {
	c := engine.NewContext(
		context.Background(),
		"task-001",
		workflowId,
		"mycollection",
		engine.TextQuery(query),
		engine.CallerMeta{
			Name: "user",
		},
	)
	return c
}

type mockModule struct{}

func (m mockModule) Operator(name string) (engine.Executer, error) {
	var fn func(c engine.Context, p *engine.Params) *engine.Response
	if name == "mock_gen" {
		fn = func(c engine.Context, p *engine.Params) *engine.Response {
			return &engine.Response{
				State:             c.State(),
				GenerationChannel: mockGenerationStream(5),
			}
		}
	} else {
		fn = func(c engine.Context, p *engine.Params) *engine.Response {
			state := c.State()
			state.AddContents(engine.ContentsFromMessages(llm.TextMessage(llm.MessageRoleAssistant, "Message from "+name)))
			return &engine.Response{State: state}
		}
	}
	return engine.ExecuterFunc(fn), nil
}

func mockGenerationStream(cnt int) <-chan engine.GenerationEvent {
	ch := make(chan engine.GenerationEvent, cnt+3)

	go func() {
		defer close(ch)
		ch <- engine.ContentStartEvent()
		for i := range cnt {
			ch <- engine.ContentDeltaEvent(fmt.Sprintf("Message %d", i+1))
		}
		ch <- engine.ContentStopEvent()
		ch <- engine.GenerationCompleteEvent()
	}()

	return ch
}
