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

package tasks

import (
	"encoding/json"

	"github.com/alan-mat/awe/internal/api"
	pb "github.com/alan-mat/awe/internal/proto"
	"github.com/hibiken/asynq"
)

var (
	DefaultWorkflowChat   = "qrouter"
	DefaultWorkflowSearch = "search_web"
)

const (
	TypeChat    = "awe:chat"
	TypeSearch  = "awe:search"
	TypeExecute = "awe:execute"
)

type chatTaskPayload struct {
	Query   string
	User    string
	History []*api.ChatMessage
	Args    map[string]string
}

func NewChatTask(req *pb.ChatRequest) (*asynq.Task, error) {
	tp := chatTaskPayload{
		Query:   req.Query,
		User:    req.User,
		History: api.ParseChatHistory(req.History),
		Args:    req.Args,
	}
	payload, err := json.Marshal(tp)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeChat, payload), nil
}

type searchTaskPayload struct {
	Query string
	User  string
	Args  map[string]string
}

func NewSearchTask(req *pb.SearchRequest) (*asynq.Task, error) {
	tp := searchTaskPayload{
		Query: req.Query,
		User:  req.User,
		Args:  req.Args,
	}
	payload, err := json.Marshal(tp)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeSearch, payload), nil
}

type executeTaskPayload struct {
	WorkflowId string
	Query      string
	User       string
	History    []*api.ChatMessage
	Args       map[string]string
}

func NewExecuteTask(req *pb.ExecuteRequest) (*asynq.Task, error) {
	tp := executeTaskPayload{
		WorkflowId: req.WorkflowId,
		Query:      req.Query,
		User:       req.User,
		History:    api.ParseChatHistory(req.History),
		Args:       req.Args,
	}
	payload, err := json.Marshal(tp)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeExecute, payload), nil
}
