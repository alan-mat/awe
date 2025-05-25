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
