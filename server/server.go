// Copyright 2025 Alan Matykiewicz
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to use,
// copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the
// Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES
// OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
// HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
// WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
// OTHER DEALINGS IN THE SOFTWARE.

package server

import (
	"fmt"
	"log/slog"
	"net"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"

	pb "github.com/alan-mat/awe/internal/proto"
	"github.com/alan-mat/awe/internal/transport"
)

type ServerConfig struct {
	ListenHost string
	ListenPort int

	RedisAddr     string
	RedisUsername string
	RedisPassword string
	RedisDB       int
}

func DefaultConfig() ServerConfig {
	return ServerConfig{
		ListenPort: 50051,
		RedisAddr:  "localhost:6379",
	}
}

// Server implements the AWEService
type Server struct {
	pb.UnimplementedAWEServiceServer

	config ServerConfig

	rdb *redis.Client

	transport   transport.Transport
	asynqClient *asynq.Client
}

func New(config ServerConfig) *Server {
	return &Server{
		config: config,
	}
}

func (s Server) Serve() error {
	lisAddr := fmt.Sprintf("%s:%d", s.config.ListenHost, s.config.ListenPort)
	lis, err := net.Listen("tcp", lisAddr)
	if err != nil {
		slog.Error("failed to start server", "err", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     s.config.RedisAddr,
		Username: s.config.RedisUsername,
		Password: s.config.RedisPassword,
		DB:       s.config.RedisDB,
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

	slog.Info("Server starting", "listener", lisAddr)
	if err := grpcServer.Serve(lis); err != nil {
		slog.Error("failed to serve", "err", err)
		return err
	}
	return nil
}
