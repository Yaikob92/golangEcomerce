package services

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimitService provides Redis-backed sliding window rate limiting.
type RateLimitService interface {
	// Allow checks if an action is allowed for the given key. Returns true if allowed.
	Allow(ctx context.Context, key string, maxAttempts int, window time.Duration) (bool, error)
}

type rateLimitService struct {
	redis *redis.Client
}

func NewRateLimitService(redisClient *redis.Client) RateLimitService {
	return &rateLimitService{redis: redisClient}
}

// Allow implements sliding window rate limiting.
// key: unique identifier (e.g. "login:192.168.1.1" or "verify:user@email.com")
// maxAttempts: max number of attempts in the window
// window: duration of the window
func (s *rateLimitService) Allow(ctx context.Context, key string, maxAttempts int, window time.Duration) (bool, error) {
	// Fail open if Redis is unavailable
	if s.redis == nil {
		return true, nil
	}

	prefixedKey := fmt.Sprintf("ratelimit:%s", key)

	// Use INCR + EXPIRE pattern for simplicity and atomicity
	pipe := s.redis.Pipeline()
	incr := pipe.Incr(ctx, prefixedKey)
	pipe.Expire(ctx, prefixedKey, window)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("rate limit check failed: %w", err)
	}

	count := incr.Val()

	// If this is the first request, the expire was already set above.
	// If count exceeds max, deny the request.
	return count <= int64(maxAttempts), nil
}
