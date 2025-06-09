// Copyright 2025 Alan Matykiewicz
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to use,
// copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the
// Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES
// OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
// HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
// WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
// OTHER DEALINGS IN THE SOFTWARE.

package engine_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/alan-mat/awe/internal/api"
	"github.com/alan-mat/awe/internal/engine"
	"github.com/alan-mat/awe/internal/llm"
)

func TestInvoker(t *testing.T) {
	inv, _ := createInvoker()
	exec := mockExecutor{"tester"}
	expected := "executing 'tester' with query 'Name three things.'"
	mw := func(e engine.ExecuterFunc) engine.ExecuterFunc {
		return func(c engine.Context, p *engine.Params) *engine.Response {
			resp := e.Execute(c, p)
			got := resp.State.Contents.Messages[0].Text()
			if got != expected {
				t.Errorf("expected contents '%s', got '%s'", expected, got)
			}
			return resp
		}
	}
	err := inv.Call(mw(engine.ExecuterFunc(exec.Execute)), engine.DefaultParams())
	if err != nil {
		t.Errorf("expected not-nil error, got %v", err)
	}
}

func TestInvokerError(t *testing.T) {
	inv, _ := createInvoker()
	var err error
	// invoke non error
	err = inv.Call(engine.ExecuterFunc(executerErrorOnNeg(1)), engine.DefaultParams())
	if err != nil {
		t.Errorf("expected not-nil error, got %v", err)
	}
	// invoke with error
	err = inv.Call(engine.ExecuterFunc(executerErrorOnNeg(-1)), engine.DefaultParams())
	if err == nil {
		t.Errorf("expected error, got %v", err)
	}
}

func TestContext(t *testing.T) {
	c := engine.NewContext(
		context.Background(),
		"task-001",
		"workflow-001",
		"mycollection",
		engine.TextQuery("Name three things."),
		engine.CallerMeta{
			Name: "user",
		},
	)

	// set new state
	oldState := c.State()
	state := engine.NewState(engine.TextQuery("Name three things."))
	state.AddContents(
		engine.ContentsFromMessages(
			llm.TextMessage(llm.MessageRoleAssistant, "New text message."),
			llm.TextMessage(llm.MessageRoleAssistant, "Another one.")))
	c = c.WithState(state)

	if !reflect.DeepEqual(state, c.State()) {
		t.Errorf("context state is incorrect, expected '%+v', got '%+v'", state, c.State())
	}

	if reflect.DeepEqual(oldState, c.State()) {
		t.Errorf("old state is equal to new state")
	}

	// with values
	keys := []string{"mykey", "mykey2"}
	vals := []any{201, "unknown"}
	for _, k := range keys {
		got := c.Value(k)
		if got != nil {
			t.Errorf("value in context for key '%s', expected nil, got '%v'", k, got)
		}
	}

	c = c.WithValues(map[string]any{
		keys[0]: vals[0],
		keys[1]: vals[1],
	})

	for i, k := range keys {
		got := c.Value(k)
		expected := vals[i]
		if got != expected {
			t.Errorf("value in context for key '%s', expected '%v', got '%v'", k, expected, got)
		}
	}
}

func TestGeneratedContentsMerge(t *testing.T) {
	cont1 := &engine.GeneratedContents{}
	cont2 := engine.ContentsFromMessages(
		llm.TextMessage(llm.MessageRoleAssistant, "Message A 1"),
		llm.TextMessage(llm.MessageRoleAssistant, "Message A 2"),
	)
	cont3 := engine.ContentsFromMessages(
		llm.TextMessage(llm.MessageRoleAssistant, "Message B 1"),
		llm.TextMessage(llm.MessageRoleAssistant, "Message B 2"),
		llm.TextMessage(llm.MessageRoleAssistant, "Message B 3"),
	)

	cont1.Merge()
	if !reflect.DeepEqual(cont1, &engine.GeneratedContents{}) {
		t.Errorf("contents are not empty")
	}

	cont1.Merge(cont2)
	if !reflect.DeepEqual(cont1, cont2) {
		t.Errorf("invalid contents, expected '%+v', got '%+v'", cont2, cont1)
	}

	// reset
	cont1 = &engine.GeneratedContents{}
	cont1.Merge(cont2, cont3)
	expected := engine.ContentsFromMessages(
		llm.TextMessage(llm.MessageRoleAssistant, "Message A 1"),
		llm.TextMessage(llm.MessageRoleAssistant, "Message A 2"),
		llm.TextMessage(llm.MessageRoleAssistant, "Message B 1"),
		llm.TextMessage(llm.MessageRoleAssistant, "Message B 2"),
		llm.TextMessage(llm.MessageRoleAssistant, "Message B 3"),
	)
	if !reflect.DeepEqual(cont1, expected) {
		t.Errorf("invalid contents, expected '%+v', got '%+v'", expected, cont1)
	}
}

func TestStateMerge(t *testing.T) {
	s1 := engine.NewState(engine.TextQuery("My initial query."))

	s1.Merge()
	if !reflect.DeepEqual(s1, engine.NewState(engine.TextQuery("My initial query."))) {
		t.Errorf("state does not match initial")
	}

	s2 := engine.NewState(engine.TextQuery("My OTHER query."))
	s2.AddContents(engine.ContentsFromMessages(
		llm.TextMessage(llm.MessageRoleUser, "Hello"),
		llm.TextMessage(llm.MessageRoleAssistant, "How can I help you?"),
	))
	s2.AddContextDocs([]*api.ScoredDocument{
		{Content: "Doc A", Score: 0.9},
		{Content: "Doc B", Score: 1, Title: "Doc B Title"},
	}...)

	s1.Merge(&s2)
	expected := engine.State{
		Query:        engine.TextQuery("My initial query."),
		QueryHistory: make(engine.QueryList, 0),
		Contents: *engine.ContentsFromMessages(
			llm.TextMessage(llm.MessageRoleUser, "Hello"),
			llm.TextMessage(llm.MessageRoleAssistant, "How can I help you?"),
		),
		ContextDocs: []*api.ScoredDocument{
			{Content: "Doc A", Score: 0.9},
			{Content: "Doc B", Score: 1, Title: "Doc B Title"},
		},
	}
	if !reflect.DeepEqual(s1, expected) {
		t.Errorf("incorrect state after merge, expected '%+v', got '%+v'", expected, s1)
	}

	s3 := engine.NewState(engine.TextQuery("My OTHER query."))
	s3.AddToHistory(engine.TextQuery("Old Query."), engine.TextQuery("My initial query."))
	s3.AddContents(engine.ContentsFromMessages(
		llm.TextMessage(llm.MessageRoleUser, "What is 2 + 2?"),
	))
	s3.AddMultiQueries(engine.TextQuery("Multi query 1"))
	s3.AddSubQueries(engine.TextQuery("Sub query 1"), engine.TextQuery("Sub query 2"))

	s4 := engine.NewState(engine.TextQuery("My OTHER OTHER query."))
	s4.AddToHistory(engine.TextQuery("Even older Query."))
	s4.AddContextDocs([]*api.ScoredDocument{
		{Content: "Doc C", Score: 0.4},
		{Content: "Doc D", Score: 0.6, Title: "Doc D Title"},
	}...)
	s4.AddMultiQueries(engine.TextQuery("Multi query A"))
	s4.AddSubQueries(engine.TextQuery("Sub query A"), engine.TextQuery("Sub query B"), engine.TextQuery("Sub query C"))

	s1.Merge(&s3, &s4)
	expected = engine.State{
		Query: engine.TextQuery("My initial query."),
		QueryHistory: engine.QueryList{
			engine.TextQuery("Old Query."),
			engine.TextQuery("My initial query."),
			engine.TextQuery("Even older Query."),
		},
		Contents: *engine.ContentsFromMessages(
			llm.TextMessage(llm.MessageRoleUser, "Hello"),
			llm.TextMessage(llm.MessageRoleAssistant, "How can I help you?"),
			llm.TextMessage(llm.MessageRoleUser, "What is 2 + 2?"),
		),
		ContextDocs: []*api.ScoredDocument{
			{Content: "Doc A", Score: 0.9},
			{Content: "Doc B", Score: 1, Title: "Doc B Title"},
			{Content: "Doc C", Score: 0.4},
			{Content: "Doc D", Score: 0.6, Title: "Doc D Title"},
		},
		MultiQueries: &engine.QueryList{
			engine.TextQuery("Multi query 1"),
			engine.TextQuery("Multi query A"),
		},
		SubQueries: &engine.QueryList{
			engine.TextQuery("Sub query 1"),
			engine.TextQuery("Sub query 2"),
			engine.TextQuery("Sub query A"),
			engine.TextQuery("Sub query B"),
			engine.TextQuery("Sub query C"),
		},
	}

	if !reflect.DeepEqual(s1, expected) {
		t.Errorf("incorrect state after merge, expected '%+v', got '%+v'", expected, s1)
	}
}

func TestGetTypedArgument(t *testing.T) {
	params := engine.DefaultParams()
	params.SetArgument("top_n", 10)
	params.SetArgument("temperature", 0.5)
	params.SetArgument("collection", "mycollection")
	params.SetArgument("search", true)

	topN := engine.GetTypedArgumentWithDefault(params.Args, "top_n", 0)
	if topN != 10 {
		t.Errorf("top_n expected '%d', got '%v'", 10, topN)
	}

	newInt := engine.GetTypedArgumentWithDefault(params.Args, "new_int", 25)
	if newInt != 25 {
		t.Errorf("new_int expected default value '%d', got '%v'", 25, newInt)
	}

	topN, ok := engine.GetTypedArgument[int](params.Args, "top_n")
	if !ok {
		t.Errorf("argument top_n not found")
	}
	if topN != 10 {
		t.Errorf("top_n expected '%d', got '%v'", 10, topN)
	}

	newInt, ok = engine.GetTypedArgument[int](params.Args, "new_int")
	if ok {
		t.Errorf("found non-existant argument new_int")
	}
	if newInt != *new(int) {
		t.Errorf("not found argument top_n must equal the zero-value of int")
	}

	temperature := engine.GetTypedArgumentWithDefault(params.Args, "temperature", 1.)
	if temperature != 0.5 {
		t.Errorf("temperature expected '%v', got '%v'", 0.5, temperature)
	}

	temperatureWrongType := engine.GetTypedArgumentWithDefault(params.Args, "temperature", 1)
	if temperatureWrongType != 1 {
		t.Errorf("temperatureWrongType expected default value '%v', got '%v'", 1, temperatureWrongType)
	}

	collection := engine.GetTypedArgumentWithDefault(params.Args, "collection", "default")
	if collection != "mycollection" {
		t.Errorf("collection expected '%v', got '%v'", "mycollection", collection)
	}

	search := engine.GetTypedArgumentWithDefault(params.Args, "search", false)
	if search != true {
		t.Errorf("search expected '%v', got '%v'", "true", collection)
	}
}

func createInvoker() (*engine.Invoker, engine.Context) {
	c := engine.NewContext(
		context.Background(),
		"task-001",
		"workflow-001",
		"mycollection",
		engine.TextQuery("Name three things."),
		engine.CallerMeta{
			Name: "user",
		},
	)
	invoker := engine.NewInvoker(c)
	return invoker, c
}

type mockExecutor struct {
	Name string
}

func (e mockExecutor) Execute(c engine.Context, p *engine.Params) *engine.Response {
	text := fmt.Sprintf("executing '%s' with query '%s'", e.Name, c.State().Query.Text)
	state := c.State()
	state.AddContents(
		engine.ContentsFromMessages(
			llm.TextMessage(llm.MessageRoleAssistant, text)))
	return &engine.Response{
		State: state,
	}
}

func executerErrorOnNeg(num int) engine.ExecuterFunc {
	return func(c engine.Context, p *engine.Params) *engine.Response {
		var err error
		if num >= 0 {
			err = nil
		} else {
			err = errors.New("num must not be negative")
		}
		return &engine.Response{
			Err:   err,
			State: c.State(),
		}
	}
}
