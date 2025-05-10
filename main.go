package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"

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
	log.Printf("received chat request from user '%s' with query '%s'", req.User, req.Query)

	prov, err := provider.NewLMProvider(provider.LMProviderTypeOpenAI)
	if err != nil {
		log.Printf("error creating new lmprovider, cancelling Chat request")
		return status.Errorf(codes.Internal, "something went wrong")
	}

	creq := provider.CompletionRequest{
		Query: req.Query,
	}
	cs, err := prov.CreateCompletionStream(context.Background(), creq)
	if err != nil {
		log.Printf("error creating chat completion stream, cancelling Chat request")
		return status.Errorf(codes.Internal, "something went wrong")
	}
	defer cs.Close()

	trace_id := "test-trace"
	msg_id := int32(0)
	for {
		chunk, err := cs.Recv()
		if errors.Is(err, io.EOF) {
			log.Printf("provider stream finished")
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

	log.Println("Stream completed")
	return nil
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterAWEServiceServer(s, &server{})

	log.Println("Server starting on :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
