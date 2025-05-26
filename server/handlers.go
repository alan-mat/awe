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

	err = handleMessageStream(stream.Context(), traceID, tstream, stream, respFunc)
	return err
}

func (s Server) Search(req *pb.SearchRequest, stream pb.AWEService_SearchServer) error {
	slog.Debug("received search request", "user", req.User, "query", req.Query, "args", req.GetArgs())

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

	err = handleMessageStream(stream.Context(), traceID, tstream, stream, respFunc)
	return err
}

func (s Server) Execute(req *pb.ExecuteRequest, stream pb.AWEService_ExecuteServer) error {
	slog.Debug("received execute request", "workflowId", req.WorkflowId, "user", req.User,
		"query", req.Query, "history", req.GetHistory(), "args", req.GetArgs())

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

	err = handleMessageStream(stream.Context(), traceID, tstream, stream, respFunc)
	return err
}

func (s Server) Trace(ctx context.Context, req *pb.TraceRequest) (*pb.TraceResponse, error) {
	trace, err := s.transport.GetTrace(ctx, req.TraceId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "trace with given id does not exist")
	}

	resp := &pb.TraceResponse{
		TraceId:     trace.ID,
		Status:      pb.TraceStatus(trace.Status),
		StartedAt:   trace.StartedAt,
		CompletedAt: trace.CompletedAt,
		Query:       trace.Query,
		User:        trace.User,
	}
	return resp, nil
}

func (s Server) Attach(req *pb.AttachRequest, stream pb.AWEService_AttachServer) error {
	trace, err := s.transport.GetTrace(stream.Context(), req.TraceId)
	if err != nil {
		return status.Errorf(codes.NotFound, "trace with given id does not exist")
	}

	tstream, err := s.transport.GetMessageStream(trace.ID)
	if err != nil {
		slog.Error("failed to retrieve stream", "id", trace.ID)
		return status.Errorf(codes.Internal, "internal server error")
	}

	if trace.Status != transport.TraceStatusRunning {
		text, err := tstream.Text(stream.Context())
		if err != nil {
			slog.Error("failed to read from stream", "id", trace.ID)
			return status.Errorf(codes.Internal, "internal server error")
		}

		resp := &pb.ExecuteResponse{
			MsgId:   0,
			TraceId: trace.ID,
			Status:  "OK",
			Payload: &pb.ExecuteResponse_Content{
				Content: text,
			},
		}
		if err := stream.Send(resp); err != nil {
			return err
		}
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

	err = handleMessageStream(stream.Context(), trace.ID, tstream, stream, respFunc)
	return err
}
