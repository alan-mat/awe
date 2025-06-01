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
