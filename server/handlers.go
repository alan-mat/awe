package server

import (
	"context"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/alan-mat/awe/internal/proto"
	"github.com/alan-mat/awe/internal/tasks"
	"github.com/alan-mat/awe/internal/transport"
)

func (s Server) Chat(req *pb.ChatRequest, stream pb.AWEService_ChatServer) error {
	slog.Debug("received chat request", "user", req.User, "query", req.Query, "history", req.GetHistory(), "args", req.GetArgs())
	ctx := context.Background()

	t, err := tasks.NewChatTask(req)
	if err != nil {
		slog.Error(err.Error())
		return status.Errorf(codes.Internal, "internal server error")
	}

	info, err := s.asynqClient.Enqueue(t)
	if err != nil {
		slog.Error(err.Error())
		return status.Errorf(codes.Internal, "internal server error")
	}
	slog.Info("enqueued task successfully", "id", info.ID)
	traceID := info.ID

	tstream, err := s.transport.GetMessageStream(traceID)
	if err != nil {
		slog.Error("failed to retrieve stream", "id", traceID)
		return status.Errorf(codes.Internal, "internal server error")
	}

	var respFunc messageResponseFunc[pb.ChatResponse] = func(msg *transport.MessageStreamPayload, traceID string) *pb.ChatResponse {
		return &pb.ChatResponse{
			MsgId:   int32(msg.ID),
			TraceId: traceID,
			Status:  msg.Status,
			Content: msg.Content,
		}
	}

	err = handleMessageStream(ctx, traceID, tstream, stream, respFunc)
	return err
}

func (s Server) Search(req *pb.SearchRequest, stream pb.AWEService_SearchServer) error {
	slog.Debug("received search request", "user", req.User, "query", req.Query, "args", req.GetArgs())
	ctx := context.Background()

	t, err := tasks.NewSearchTask(req)
	if err != nil {
		slog.Error(err.Error())
		return status.Errorf(codes.Internal, "internal server error")
	}

	info, err := s.asynqClient.Enqueue(t)
	if err != nil {
		slog.Error(err.Error())
		return status.Errorf(codes.Internal, "internal server error")
	}
	slog.Info("enqueued task successfully", "id", info.ID)
	traceID := info.ID

	tstream, err := s.transport.GetMessageStream(traceID)
	if err != nil {
		slog.Error("failed to retrieve stream", "id", traceID)
		return status.Errorf(codes.Internal, "internal server error")
	}

	var respFunc messageResponseFunc[pb.SearchResponse] = func(msg *transport.MessageStreamPayload, traceID string) *pb.SearchResponse {
		return &pb.SearchResponse{
			MsgId:   int32(msg.ID),
			TraceId: traceID,
			Status:  msg.Status,
			Document: &pb.Document{
				Title:   msg.Document.Title,
				Content: msg.Document.Content,
				Source:  msg.Document.Source,
			},
		}
	}

	err = handleMessageStream(ctx, traceID, tstream, stream, respFunc)
	return err
}

func (s Server) Execute(req *pb.ExecuteRequest, stream pb.AWEService_ExecuteServer) error {
	slog.Debug("received execute request", "workflowId", req.WorkflowId, "user", req.User,
		"query", req.Query, "history", req.GetHistory(), "args", req.GetArgs())
	ctx := context.Background()

	t, err := tasks.NewExecuteTask(req)
	if err != nil {
		slog.Error(err.Error())
		return status.Errorf(codes.Internal, "internal server error")
	}

	info, err := s.asynqClient.Enqueue(t)
	if err != nil {
		slog.Error(err.Error())
		return status.Errorf(codes.Internal, "internal server error")
	}
	slog.Info("enqueued task successfully", "id", info.ID)
	traceID := info.ID

	tstream, err := s.transport.GetMessageStream(traceID)
	if err != nil {
		slog.Error("failed to retrieve stream", "id", traceID)
		return status.Errorf(codes.Internal, "internal server error")
	}

	var respFunc messageResponseFunc[pb.ExecuteResponse] = func(msg *transport.MessageStreamPayload, traceID string) *pb.ExecuteResponse {
		resp := &pb.ExecuteResponse{
			MsgId:   int32(msg.ID),
			TraceId: traceID,
			Status:  msg.Status,
		}

		switch msg.Type {
		case transport.MessageTypeContent:
			resp.Payload = &pb.ExecuteResponse_Content{
				Content: msg.Content,
			}

		case transport.MessageTypeDocument:
			resp.Payload = &pb.ExecuteResponse_Document{
				Document: &pb.Document{
					Title:   msg.Document.Title,
					Content: msg.Document.Content,
					Source:  msg.Document.Source,
				},
			}
		}

		return resp
	}

	err = handleMessageStream(ctx, traceID, tstream, stream, respFunc)
	return err
}
