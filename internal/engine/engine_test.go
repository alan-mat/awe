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
	"fmt"
	"testing"

	"github.com/alan-mat/awe/internal/engine"
	"github.com/alan-mat/awe/internal/llm"
)

func TestInvoker(t *testing.T) {
	inv := createInvoker()
	exec := mockExecutor{"tester"}
	err := inv.Call(exec, engine.DefaultParams())
	if err != nil {
		t.Errorf("expected not-nil error, got %v", err)
	}
}

func createInvoker() *engine.Invoker {
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
	return invoker
}

type mockExecutor struct {
	Name string
}

func (e mockExecutor) Execute(c engine.Context, p *engine.Params) *engine.Response {
	text := fmt.Sprintf("executing '%s' with query '%s'", e.Name, c.State().Query)
	state := c.State()
	state.AddContents(
		engine.ContentsFromMessages(
			llm.TextMessage(llm.MessageRoleAssistant, text)))
	return &engine.Response{
		State: state,
	}
}
