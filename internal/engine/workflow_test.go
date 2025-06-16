package engine_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/alan-mat/awe/internal/engine"
	"github.com/alan-mat/awe/internal/llm"
)

func TestWorkflowExecute(t *testing.T) {
	query := "My query"
	workflow := createWorkflow("wf-1", 3)
	c := createContext("wf-1", query)
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

func TestWorkflowWithConditional(t *testing.T) {
	// default route route_a
	query := "My query"
	workflow := createWorkflowWithConditional("wf-1", "")
	c := createContext("wf-1", query)
	inv := engine.NewInvoker(c)
	err := inv.Call(workflow.Executer(nil), engine.DefaultParams())
	if err != nil {
		t.Errorf("expected not-nil error, got '%v'", err)
	}

	expected := engine.NewState(engine.TextQuery(query))
	expected.AddContents(engine.ContentsFromMessages(
		llm.TextMessage(llm.MessageRoleAssistant, "Message from op1"),
		llm.TextMessage(llm.MessageRoleAssistant, "Message from op2"),
		llm.TextMessage(llm.MessageRoleAssistant, "Router -> route_a"),
		llm.TextMessage(llm.MessageRoleAssistant, "Message from route_a_op1"),
		llm.TextMessage(llm.MessageRoleAssistant, "Message from route_a_op2"),
	))

	if !reflect.DeepEqual(inv.State(), expected) {
		t.Errorf("invalid state after workflow execution, expected '%+v', got '%+v'", expected, inv.State())
	}

	// check other route
	workflow = createWorkflowWithConditional("wf-1", "route_b")
	c = createContext("wf-1", query)
	inv = engine.NewInvoker(c)
	err = inv.Call(workflow.Executer(nil), engine.DefaultParams())
	if err != nil {
		t.Errorf("expected not-nil error, got '%v'", err)
	}

	expected = engine.NewState(engine.TextQuery(query))
	expected.AddContents(engine.ContentsFromMessages(
		llm.TextMessage(llm.MessageRoleAssistant, "Message from op1"),
		llm.TextMessage(llm.MessageRoleAssistant, "Message from op2"),
		llm.TextMessage(llm.MessageRoleAssistant, "Router -> route_b"),
		llm.TextMessage(llm.MessageRoleAssistant, "Message from route_b_op1"),
		llm.TextMessage(llm.MessageRoleAssistant, "Message from route_b_op2"),
		llm.TextMessage(llm.MessageRoleAssistant, "Message from route_b_op3"),
	))

	if !reflect.DeepEqual(inv.State(), expected) {
		t.Errorf("invalid state after workflow execution, expected '%+v', got '%+v'", expected, inv.State())
	}

	// not existent route throws error
	workflow = createWorkflowWithConditional("wf-1", "route_X")
	c = createContext("wf-1", query)
	inv = engine.NewInvoker(c)
	err = inv.Call(workflow.Executer(nil), engine.DefaultParams())
	expected_err := engine.ErrNodeRouteNotFound
	if !errors.Is(err, expected_err) {
		t.Errorf("expected error of type %T, got '%v' of type %T", expected_err, err, err)
	}
}

func TestWorkflowWithNestedWorkflow(t *testing.T) {
	// 1 level of nesting
	query := "My query"
	workflow := createWorkflowWithNested("wf-1", 1)
	c := createContext("wf-1", query)
	inv := engine.NewInvoker(c)
	err := inv.Call(workflow.Executer(nil), engine.DefaultParams())
	if err != nil {
		t.Errorf("expected not-nil error, got '%v'", err)
	}

	expected := engine.NewState(engine.TextQuery(query))
	expected.AddContents(engine.ContentsFromMessages(
		llm.TextMessage(llm.MessageRoleAssistant, "Message from op1"),
		llm.TextMessage(llm.MessageRoleAssistant, "Message from op2"),
		llm.TextMessage(llm.MessageRoleAssistant, "Running nested nested_1"),
		llm.TextMessage(llm.MessageRoleAssistant, "Message from op1"),
		llm.TextMessage(llm.MessageRoleAssistant, "Message from op2"),
		llm.TextMessage(llm.MessageRoleAssistant, "Message from op3"),
		llm.TextMessage(llm.MessageRoleAssistant, "Message from end of nested_1"),
	))

	if !reflect.DeepEqual(inv.State(), expected) {
		t.Errorf("invalid state after workflow execution, expected '%+v', got '%+v'", expected, inv.State())
	}

	// multiple levels of nesting
	workflow = createWorkflowWithNested("wf-1", 3)
	c = createContext("wf-1", query)
	inv = engine.NewInvoker(c)
	err = inv.Call(workflow.Executer(nil), engine.DefaultParams())
	if err != nil {
		t.Errorf("expected not-nil error, got '%v'", err)
	}

	expected = engine.NewState(engine.TextQuery(query))
	expected.AddContents(engine.ContentsFromMessages(
		llm.TextMessage(llm.MessageRoleAssistant, "Message from op1"),
		llm.TextMessage(llm.MessageRoleAssistant, "Message from op2"),
		llm.TextMessage(llm.MessageRoleAssistant, "Running nested nested_3"),
		llm.TextMessage(llm.MessageRoleAssistant, "Message from op1"),
		llm.TextMessage(llm.MessageRoleAssistant, "Message from op2"),
		llm.TextMessage(llm.MessageRoleAssistant, "Running nested nested_2"),
		llm.TextMessage(llm.MessageRoleAssistant, "Message from op1"),
		llm.TextMessage(llm.MessageRoleAssistant, "Message from op2"),
		llm.TextMessage(llm.MessageRoleAssistant, "Running nested nested_1"),
		llm.TextMessage(llm.MessageRoleAssistant, "Message from op1"),
		llm.TextMessage(llm.MessageRoleAssistant, "Message from op2"),
		llm.TextMessage(llm.MessageRoleAssistant, "Message from op3"),
		llm.TextMessage(llm.MessageRoleAssistant, "Message from end of nested_1"),
		llm.TextMessage(llm.MessageRoleAssistant, "Message from end of nested_2"),
		llm.TextMessage(llm.MessageRoleAssistant, "Message from end of nested_3"),
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

func createWorkflowWithConditional(workflowId string, toRoute string) *engine.Workflow {
	m := mockModule{}
	nodes := []*engine.WorkflowNode{
		engine.NewWorkflowNode(m, "op1", engine.WorkflowNodeLinear),
		engine.NewWorkflowNode(m, "op2", engine.WorkflowNodeLinear),
	}

	cond := engine.NewWorkflowNode(m, "mock_router", engine.WorkflowNodeConditional)
	routes := []*engine.WorkflowRoute{
		{
			Key: "route_a",
			Nodes: []*engine.WorkflowNode{
				engine.NewWorkflowNode(m, "route_a_op1", engine.WorkflowNodeLinear),
				engine.NewWorkflowNode(m, "route_a_op2", engine.WorkflowNodeLinear),
			},
		},
		{
			Key: "route_b",
			Nodes: []*engine.WorkflowNode{
				engine.NewWorkflowNode(m, "route_b_op1", engine.WorkflowNodeLinear),
				engine.NewWorkflowNode(m, "route_b_op2", engine.WorkflowNodeLinear),
				engine.NewWorkflowNode(m, "route_b_op3", engine.WorkflowNodeLinear),
			},
		},
	}
	if toRoute != "" {
		cond.SetArgument("route", toRoute)
	}
	cond.AddRoutes(routes...)
	nodes = append(nodes, cond)

	return engine.NewWorkflow(
		workflowId,
		"Workflow with conditional node",
		"mycollection",
		nodes,
	)
}

func createWorkflowWithNested(workflowId string, levels int) *engine.Workflow {
	if levels < 1 {
		panic("levels must be equal to 1 or greater")
	}

	m := mockModule{}
	nodes := []*engine.WorkflowNode{
		engine.NewWorkflowNode(m, "op1", engine.WorkflowNodeLinear),
		engine.NewWorkflowNode(m, "op2", engine.WorkflowNodeLinear),
		engine.NewWorkflowNode(m, fmt.Sprintf("nested_%d", levels), engine.WorkflowNodeLinear),
		engine.NewWorkflowNode(m, fmt.Sprintf("end of nested_%d", levels), engine.WorkflowNodeLinear),
	}

	return engine.NewWorkflow(
		workflowId,
		"Workflow with nested workflow",
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

	switch {
	case name == "mock_gen":
		fn = func(c engine.Context, p *engine.Params) *engine.Response {
			return &engine.Response{
				State:             c.State(),
				GenerationChannel: mockGenerationStream(5),
			}
		}
	case name == "mock_router":
		fn = func(c engine.Context, p *engine.Params) *engine.Response {
			route := engine.GetTypedArgumentWithDefault(p.Args, "route", "route_a")
			state := c.State()
			state.AddContents(engine.ContentsFromMessages(llm.TextMessage(llm.MessageRoleAssistant, "Router -> "+route)))
			state.SetNextRoute(route)
			return &engine.Response{State: state}
		}
	case strings.HasPrefix(name, "nested_"):
		levels, err := strconv.Atoi(strings.TrimPrefix(name, "nested_"))
		if err != nil {
			panic(err)
		}

		fn = func(c engine.Context, p *engine.Params) *engine.Response {
			state := c.State()

			var wf *engine.Workflow
			if (levels - 1) > 0 {
				wf = createWorkflowWithNested(fmt.Sprintf("wf_nested-%d", levels), levels-1)
			} else {
				wf = createWorkflow(fmt.Sprintf("wf_nested-%d", levels), 3)
			}

			state.AddContents(engine.ContentsFromMessages(llm.TextMessage(llm.MessageRoleAssistant, "Running nested "+name)))
			state.SetNextWorkflow(wf)
			return &engine.Response{State: state}
		}
	default:
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
