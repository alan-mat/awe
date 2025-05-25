package server

import (
	"context"
	"log/slog"
	"time"

	"github.com/alan-mat/awe/internal/transport"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type messageResponseFunc[T any] func(msg *transport.MessageStreamPayload, traceID string) *T

func handleMessageStream[T any](
	ctx context.Context,
	traceID string,
	tstream transport.MessageStream,
	stream grpc.ServerStreamingServer[T],
	respFunc messageResponseFunc[T],
) error {
	readFails := 0
	for {
		msg, err := tstream.Recv(ctx)

		if err != nil {
			slog.Warn("failed to read from stream", "stream", traceID)
			readFails += 1
			if readFails >= 10 {
				slog.Error("exceeded stream read attempts, failed", "id", traceID)
				return status.Errorf(codes.Internal, "internal server error")
			}
			time.Sleep(100 * time.Millisecond)
			continue
		}
		readFails = 0

		switch msg.Status {
		case "ERR":
			return status.Errorf(codes.Internal, "message stream failed")
		case "DONE":
			slog.Debug("message stream done", "trace", traceID)
			return nil
		}

		resp := respFunc(msg, traceID)
		if err := stream.Send(resp); err != nil {
			return err
		}
	}
}
