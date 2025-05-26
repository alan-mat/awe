package transport

import (
	"context"
	"encoding/json"
	"fmt"

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

func (t RedisTransport) SetTrace(ctx context.Context, trace *RequestTrace) error {
	key := fmt.Sprintf("awe:trace:%s", trace.ID)
	_, err := t.rdb.HSet(ctx, key, trace).Result()
	if err != nil {
		return fmt.Errorf("failed to set trace: %w", err)
	}

	_, err = t.rdb.Expire(ctx, key, TraceExpiry).Result()
	if err != nil {
		return fmt.Errorf("failed to set trace expiry: %w", err)
	}

	return nil
}

func (t RedisTransport) GetTrace(ctx context.Context, traceId string) (*RequestTrace, error) {
	key := fmt.Sprintf("awe:trace:%s", traceId)
	var trace RequestTrace
	err := t.rdb.HGetAll(ctx, key).Scan(&trace)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve trace with id '%s': %w", traceId, err)
	}

	return &trace, nil
}

func (t *RedisTransport) GetMessageStream(id string) (MessageStream, error) {
	if len(id) == 0 {
		return nil, fmt.Errorf("invalid stream ID")
	}
	rs := &RedisStream{
		id:          fmt.Sprintf("awe:stream:%s", id),
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

	_, err = s.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: s.id,
		ID:     "*",
		Values: map[string]any{
			"payload": string(payloadJSON),
		},
	}).Result()

	if err != nil {
		return err
	}
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

func (s *RedisStream) Text(ctx context.Context) (string, error) {
	rmsgs, err := s.rdb.XRange(ctx, s.id, "-", "+").Result()
	if err != nil {
		return "", err
	}

	var text string
	for _, msg := range rmsgs {
		payloadJSON, ok := msg.Values["payload"].(string)
		if !ok {
			return "", fmt.Errorf("failed to read payload from stream message")
		}

		var payload MessageStreamPayload
		if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
			return "", fmt.Errorf("failed to deserialize stream message payload")
		}

		if payload.Status == "OK" {
			text += payload.Content
		}
	}

	return text, nil
}

func (s *RedisStream) GetID() string {
	return s.id
}
