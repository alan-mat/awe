package main

import (
	"context"
	"log"
	"math/rand"
	"net"
	"time"

	pb "github.com/alan-mat/awe/internal/proto"
	"google.golang.org/grpc"
)

// Server implements the AWEService
type server struct {
	pb.UnimplementedAWEServiceServer
}

// Post implements the unary RPC method
func (s *server) Post(ctx context.Context, req *pb.PostRequest) (*pb.PostResponse, error) {
	log.Printf("Received Post request with data: %s", req.Data)
	return &pb.PostResponse{
		Success: true,
		Message: "Request processed successfully",
	}, nil
}

// Stream implements the server streaming RPC method
func (s *server) Stream(req *pb.StreamRequest, stream pb.AWEService_StreamServer) error {
	log.Printf("Starting stream, will send %d responses", req.Count)

	// Default to 10 responses if count is 0
	count := req.Count
	if count <= 0 {
		count = 10
	}

	// Create random number generator
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Send random integers every 0.5 seconds
	for i := int32(0); i < count; i++ {
		// Generate random integer
		randomInt := rng.Int31n(1000)

		// Create response
		resp := &pb.StreamResponse{
			Data:      randomInt,
			Timestamp: time.Now().Unix(),
		}

		// Send the response
		if err := stream.Send(resp); err != nil {
			return err
		}

		log.Printf("Sent random data: %d", randomInt)

		// Wait 0.5 seconds before sending next data
		time.Sleep(500 * time.Millisecond)
	}

	log.Println("Stream completed")
	return nil
}

func main() {
	// Listen on port 50051
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Create gRPC server
	s := grpc.NewServer()
	pb.RegisterAWEServiceServer(s, &server{})

	log.Println("Server starting on :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
