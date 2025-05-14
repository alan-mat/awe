package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"time"

	"github.com/hibiken/asynq"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/alan-mat/awe/internal/proto"
	"github.com/alan-mat/awe/internal/tasks"
	"github.com/alan-mat/awe/internal/transport"
)

// Server implements the AWEService
type server struct {
	pb.UnimplementedAWEServiceServer
	transport   transport.Transport
	asynqClient *asynq.Client
}

func (s *server) Chat(req *pb.ChatRequest, stream pb.AWEService_ChatServer) error {
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

	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // use default Addr
		Password: "",               // no password set
		DB:       0,                // use default DB
	})
	defer rdb.Close()

	t := transport.NewRedisTransport(rdb)

	client := asynq.NewClientFromRedisClient(rdb)
	defer client.Close()
	client.Ping()

	s := grpc.NewServer()
	pb.RegisterAWEServiceServer(s, &server{
		asynqClient: client,
		transport:   t,
	})

	slog.Info("Server starting on :50051")
	if err := s.Serve(lis); err != nil {
		slog.Error("failed to serve", "err", err)
		os.Exit(1)
	}
}
