package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

type RedisTransport struct {
	rdb *redis.Client
}

func NewRedisTransport(rdb *redis.Client) *RedisTransport {
	return &RedisTransport{
		rdb: rdb,
	}
}

func (t *RedisTransport) GetMessageStream(id string) (MessageStream, error) {
	if len(id) == 0 {
		return nil, fmt.Errorf("invalid stream ID")
	}
	rs := &RedisStream{
		id:          id,
		lastRedisID: "0",
		rdb:         t.rdb,
	}
	return rs, nil
}

type RedisStream struct {
	id          string
	lastRedisID string

	rdb *redis.Client
}

func (s RedisStream) Send(ctx context.Context, payload MessageStreamPayload) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	res, err := s.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: s.id,
		ID:     "*",
		Values: map[string]any{
			"payload": string(payloadJSON),
		},
	}).Result()

	if err != nil {
		return err
	}

	slog.Debug("received result from redis", "res", res)
	return nil
}

func (s *RedisStream) Recv(ctx context.Context) (*MessageStreamPayload, error) {
	rstreams, err := s.rdb.XRead(ctx, &redis.XReadArgs{
		Streams: []string{s.id, s.lastRedisID},
		Count:   1,
		Block:   0,
	}).Result()
	if err != nil {
		return nil, err
	}

	msg := rstreams[0].Messages[0]
	s.lastRedisID = msg.ID
	payloadJSON, ok := msg.Values["payload"].(string)
	if !ok {
		return nil, fmt.Errorf("failed to read payload from stream message")
	}

	var payload MessageStreamPayload
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		return nil, fmt.Errorf("failed to deserialize stream message payload")
	}

	return &payload, nil
}

func (s *RedisStream) GetID() string {
	return s.id
}
