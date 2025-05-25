package server

import (
	"log/slog"
	"net"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"

	pb "github.com/alan-mat/awe/internal/proto"
	"github.com/alan-mat/awe/internal/transport"
)

// Server implements the AWEService
type Server struct {
	pb.UnimplementedAWEServiceServer

	rdb *redis.Client

	transport   transport.Transport
	asynqClient *asynq.Client
}

func New() *Server {
	return &Server{}
}

func (s Server) Serve() error {
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

	grpcServer := grpc.NewServer()
	pb.RegisterAWEServiceServer(grpcServer, &Server{
		rdb:         rdb,
		transport:   t,
		asynqClient: client,
	})

	slog.Info("Server starting on :50051")
	if err := grpcServer.Serve(lis); err != nil {
		slog.Error("failed to serve", "err", err)
		return err
	}
	return nil
}
