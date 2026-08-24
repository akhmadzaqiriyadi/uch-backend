package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var (
	ErrLockNotAcquired = errors.New("unable to acquire distributed lock")
	ErrLockNotHeld     = errors.New("distributed lock is not held or already expired")
)

// Locker defines the interface for distributed locking mechanisms
type Locker interface {
	Acquire(ctx context.Context, key string, ttl time.Duration) (string, error)
	Release(ctx context.Context, key, lockToken string) error
	WithLock(ctx context.Context, key string, ttl time.Duration, fn func(ctx context.Context) error) error
}

// RedisLocker implements distributed locking with Redis using atomic SET NX and Lua release
type RedisLocker struct {
	client *redis.Client
}

func NewRedisLocker(client *redis.Client) *RedisLocker {
	return &RedisLocker{
		client: client,
	}
}

// Acquire acquires a lock for a given key with a specified TTL.
// Returns a unique lockToken required to safely release the lock.
func (l *RedisLocker) Acquire(ctx context.Context, key string, ttl time.Duration) (string, error) {
	if l.client == nil {
		return "noop-lock-token", nil
	}

	lockToken := uuid.New().String()
	lockKey := fmt.Sprintf("lock:%s", key)

	success, err := l.client.SetNX(ctx, lockKey, lockToken, ttl).Result()
	if err != nil {
		return "", fmt.Errorf("failed to execute lock command: %w", err)
	}

	if !success {
		return "", ErrLockNotAcquired
	}

	return lockToken, nil
}

// Release safely releases a lock only if the lockToken matches the current owner (via atomic Lua script)
func (l *RedisLocker) Release(ctx context.Context, key, lockToken string) error {
	if l.client == nil || lockToken == "noop-lock-token" {
		return nil
	}

	lockKey := fmt.Sprintf("lock:%s", key)

	// Atomic check-and-delete Lua script
	luaScript := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`

	res, err := l.client.Eval(ctx, luaScript, []string{lockKey}, lockToken).Result()
	if err != nil {
		return fmt.Errorf("failed to execute lock release script: %w", err)
	}

	if val, ok := res.(int64); !ok || val == 0 {
		return ErrLockNotHeld
	}

	return nil
}

// WithLock executes a callback within an acquired distributed lock and automatically releases it
func (l *RedisLocker) WithLock(ctx context.Context, key string, ttl time.Duration, fn func(ctx context.Context) error) error {
	token, err := l.Acquire(ctx, key, ttl)
	if err != nil {
		return err
	}

	defer func() {
		_ = l.Release(ctx, key, token)
	}()

	return fn(ctx)
}
