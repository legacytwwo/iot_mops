package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	rstate "rule_engine/internal/state"
)

type StateStore struct {
	client *redis.Client
}

func New(client *redis.Client) *StateStore {
	return &StateStore{client: client}
}

func (s *StateStore) GetRuleState(ctx context.Context, ruleID, metric, deviceID string) (rstate.RuleState, bool, error) {
	key := ruleStateKey(ruleID, metric, deviceID)
	val, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return rstate.RuleState{}, false, nil
		}
		return rstate.RuleState{}, false, fmt.Errorf("get rule state: %w", err)
	}

	var state rstate.RuleState
	if err := json.Unmarshal([]byte(val), &state); err != nil {
		return rstate.RuleState{}, false, fmt.Errorf("decode rule state: %w", err)
	}
	return state, true, nil
}

func (s *StateStore) SetRuleState(ctx context.Context, ruleID, metric, deviceID string, state rstate.RuleState, ttl time.Duration) error {
	key := ruleStateKey(ruleID, metric, deviceID)
	payload, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("encode rule state: %w", err)
	}

	if err := s.client.Set(ctx, key, payload, ttl).Err(); err != nil {
		return fmt.Errorf("set rule state: %w", err)
	}
	return nil
}

func (s *StateStore) DeleteRuleState(ctx context.Context, ruleID, metric, deviceID string) error {
	key := ruleStateKey(ruleID, metric, deviceID)
	if err := s.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("delete rule state: %w", err)
	}
	return nil
}

func (s *StateStore) AddToTimeWindow(ctx context.Context, ruleID, metric, deviceID, member string, ts time.Time, ttl time.Duration) error {
	key := timeWindowKey(ruleID, metric, deviceID)

	pipe := s.client.Pipeline()
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(ts.Unix()), Member: member})
	if ttl > 0 {
		pipe.Expire(ctx, key, ttl)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("zadd time window: %w", err)
	}
	return nil
}

func (s *StateStore) PruneTimeWindow(ctx context.Context, ruleID, metric, deviceID string, from time.Time) error {
	key := timeWindowKey(ruleID, metric, deviceID)
	if err := s.client.ZRemRangeByScore(ctx, key, "-inf", fmt.Sprintf("%d", from.Unix())).Err(); err != nil {
		return fmt.Errorf("prune time window: %w", err)
	}
	return nil
}

func (s *StateStore) CountTimeWindow(ctx context.Context, ruleID, metric, deviceID string) (int64, error) {
	key := timeWindowKey(ruleID, metric, deviceID)
	count, err := s.client.ZCard(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("count time window: %w", err)
	}
	return count, nil
}

func (s *StateStore) DeleteTimeWindow(ctx context.Context, ruleID, metric, deviceID string) error {
	key := timeWindowKey(ruleID, metric, deviceID)
	if err := s.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("delete time window: %w", err)
	}
	return nil
}

func (s *StateStore) GetLastSeen(ctx context.Context, deviceID string) (time.Time, bool, error) {
	key := lastSeenKey(deviceID)
	val, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return time.Time{}, false, nil
		}
		return time.Time{}, false, fmt.Errorf("get last seen: %w", err)
	}

	n, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("parse last seen: %w", err)
	}
	return time.Unix(n, 0).UTC(), true, nil
}

func (s *StateStore) SetLastSeen(ctx context.Context, deviceID string, ts time.Time, ttl time.Duration) error {
	key := lastSeenKey(deviceID)
	if err := s.client.Set(ctx, key, ts.Unix(), ttl).Err(); err != nil {
		return fmt.Errorf("set last seen: %w", err)
	}
	return nil
}

func (s *StateStore) TryMarkMessage(ctx context.Context, messageID string, ttl time.Duration) (bool, error) {
	key := messageKey(messageID)
	ok, err := s.client.SetNX(ctx, key, "1", ttl).Result()
	if err != nil {
		return false, fmt.Errorf("setnx message: %w", err)
	}
	return ok, nil
}

func (s *StateStore) DeleteMessageMark(ctx context.Context, messageID string) error {
	key := messageKey(messageID)
	if err := s.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("delete message mark: %w", err)
	}
	return nil
}

func ruleStateKey(ruleID, metric, deviceID string) string {
	return fmt.Sprintf("rule_state:%s:%s:%s", ruleID, metric, deviceID)
}

func timeWindowKey(ruleID, metric, deviceID string) string {
	return fmt.Sprintf("rule_window:%s:%s:%s", ruleID, metric, deviceID)
}

func lastSeenKey(deviceID string) string {
	return fmt.Sprintf("last_seen:%s", deviceID)
}

func messageKey(messageID string) string {
	return fmt.Sprintf("msg:%s", messageID)
}
