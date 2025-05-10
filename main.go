package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"

	"github.com/joho/godotenv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/alan-mat/awe/internal/proto"
	"github.com/alan-mat/awe/internal/provider"
)

// Server implements the AWEService
type server struct {
	pb.UnimplementedAWEServiceServer
}

func (s *server) Chat(req *pb.ChatRequest, stream pb.AWEService_ChatServer) error {
	slog.Debug("received chat request", "user", req.User, "query", req.Query)

	prov, err := provider.NewLMProvider(provider.LMProviderTypeGemini)
	if err != nil {
		slog.Warn("error creating new lmprovider, cancelling Chat request")
		return status.Errorf(codes.Internal, "something went wrong")
	}

	creq := provider.CompletionRequest{
		Query: req.Query,
	}
	cs, err := prov.CreateCompletionStream(context.Background(), creq)
	if err != nil {
		slog.Warn("error creating chat completion stream, cancelling Chat request")
		return status.Errorf(codes.Internal, "something went wrong")
	}
	defer cs.Close()

	trace_id := "test-trace"
	msg_id := int32(0)
	for {
		chunk, err := cs.Recv()
		if errors.Is(err, io.EOF) {
			slog.Debug("provider stream finished")
			break
		}

		if err != nil {
			fmt.Printf("provider stream error: %v", err)
			resp := &pb.ChatResponse{
				MsgId:   msg_id,
				TraceId: trace_id,
				Status:  "Provider stream error!",
			}
			if err := stream.Send(resp); err != nil {
				return err
			}
			break
		}

		resp := &pb.ChatResponse{
			MsgId:   msg_id,
			TraceId: trace_id,
			Content: chunk,
		}
		if err := stream.Send(resp); err != nil {
			return err
		}
		msg_id += 1
	}

	slog.Debug("stream completed")
	return nil
}

func main() {
	err := godotenv.Load()
	if err != nil {
		slog.Error("Error loading .env file")
		os.Exit(1)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		slog.Error("failed to start server", "err", err)
	}

	s := grpc.NewServer()
	pb.RegisterAWEServiceServer(s, &server{})

	slog.Info("Server starting on :50051")
	if err := s.Serve(lis); err != nil {
		slog.Error("failed to serve", "err", err)
		os.Exit(1)
	}
}
