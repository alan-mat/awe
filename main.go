package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/hibiken/asynq"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/alan-mat/awe/internal/proto"
	"github.com/alan-mat/awe/internal/tasks"
)

// Server implements the AWEService
type server struct {
	pb.UnimplementedAWEServiceServer
	rdb         *redis.Client
	asynqClient *asynq.Client
}

func (s *server) Chat(req *pb.ChatRequest, stream pb.AWEService_ChatServer) error {
	slog.Debug("received chat request", "user", req.User, "query", req.Query, "history", req.GetHistory(), "args", req.GetArgs())
	//history := message.ParseChatHistory(req.History)

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
	lastID := "0"

OuterLoop:
	for {
		rstreams, err := s.rdb.XRead(ctx, &redis.XReadArgs{
			Streams: []string{traceID, lastID},
			Count:   1,
			Block:   0,
		}).Result()

		if err != nil {
			slog.Error("failed to read from stream", "stream", traceID)
			time.Sleep(100 * time.Millisecond)
			continue
		}

		for _, str := range rstreams {
			for _, msg := range str.Messages {
				lastID = msg.ID
				slog.Debug("received message from stream", "stream", traceID, "msgID", lastID, "values", msg.Values)

				stat, ok := msg.Values["status"].(string)
				if !ok {
					slog.Warn("failed to retrieve message status", "stream", traceID, "msgID", lastID)
					return status.Errorf(codes.Internal, "something went wrong")
				}

				msgContent, ok := msg.Values["message"].(string)
				if !ok {
					slog.Warn("failed to retrieve message contents", "stream", traceID, "msgID", lastID)
					return status.Errorf(codes.Internal, "something went wrong")
				}

				midS, ok := msg.Values["id"].(string)
				if !ok {
					slog.Warn("failed to retrieve message id", "stream", traceID, "msgID", lastID)
					return status.Errorf(codes.Internal, "something went wrong")
				}

				midI, err := strconv.ParseInt(midS, 10, 64)
				if err != nil {
					slog.Warn("failed to parse message id", "stream", traceID, "msgID", lastID)
					return status.Errorf(codes.Internal, "something went wrong")
				}

				switch stat {
				case "ERR":
					return status.Errorf(codes.Internal, "something went wrong")
				case "DONE":
					break OuterLoop
				default:
				}

				resp := &pb.ChatResponse{
					MsgId:   int32(midI),
					TraceId: traceID,
					Content: msgContent,
				}
				if err := stream.Send(resp); err != nil {
					return err
				}
			}
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

	client := asynq.NewClientFromRedisClient(rdb)
	defer client.Close()
	client.Ping()

	s := grpc.NewServer()
	pb.RegisterAWEServiceServer(s, &server{
		asynqClient: client,
		rdb:         rdb,
	})

	slog.Info("Server starting on :50051")
	if err := s.Serve(lis); err != nil {
		slog.Error("failed to serve", "err", err)
		os.Exit(1)
	}
}
