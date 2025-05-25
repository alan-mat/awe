package server

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/alan-mat/awe/internal/proto"
	"github.com/alan-mat/awe/internal/tasks"
)

func (s Server) Chat(req *pb.ChatRequest, stream pb.AWEService_ChatServer) error {
	slog.Debug("received chat request", "user", req.User, "query", req.Query, "history", req.GetHistory(), "args", req.GetArgs())

	t, err := tasks.NewChatTask(req)
	if err != nil {
		slog.Error(err.Error())
		panic(1)
	}

	info, err := s.asynqClient.Enqueue(t)
	if err != nil {
		slog.Error(err.Error())
		panic(1)
	}
	slog.Info("enqueued task successfully", "id", info.ID)
	traceID := info.ID

	ctx := context.Background()

	tstream, err := s.transport.GetMessageStream(traceID)
	if err != nil {
		slog.Error("failed to retrieve stream", "id", traceID)
		return status.Errorf(codes.Internal, "something went wrong")
	}

	readFails := 0
Loop:
	for {
		msg, err := tstream.Recv(ctx)

		if err != nil {
			slog.Error("failed to read from stream", "stream", traceID)
			readFails += 1
			if readFails >= 10 {
				slog.Error("exceeded stream read attempts, failed", "id", traceID)
				return status.Errorf(codes.Internal, "something went wrong")
			}
			time.Sleep(100 * time.Millisecond)
			continue
		}
		readFails = 0

		switch msg.Status {
		case "ERR":
			return status.Errorf(codes.Internal, "something went wrong")
		case "DONE":
			break Loop
		default:
		}

		resp := &pb.ChatResponse{
			MsgId:   int32(msg.ID),
			TraceId: traceID,
			Content: msg.Content,
		}
		if err := stream.Send(resp); err != nil {
			return err
		}
	}

	slog.Debug("completed", "stream", traceID)
	return nil
}
