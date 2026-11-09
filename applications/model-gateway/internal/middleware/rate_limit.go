package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

var rateLimitScript = redis.NewScript(`
local count = redis.call("INCR", KEYS[1])
if count == 1 then
    redis.call("EXPIRE", KEYS[1], ARGV[1])
end
return count
`)

func RateLimit(
	client *redis.Client,
	apiKey string,
	limit int64,
	next http.Handler,
) http.Handler {
	fingerprint := sha256.Sum256([]byte(apiKey))
	clientID := hex.EncodeToString(fingerprint[:])

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		window := time.Now().Unix() / 60
		key := "gateway:ratelimit:" + clientID + ":" +
			strconv.FormatInt(window, 10)

		count, err := rateLimitScript.Run(
			r.Context(),
			client,
			[]string{key},
			120,
		).Int64()

		if err != nil {
			http.Error(w, "rate limiter unavailable",
				http.StatusServiceUnavailable)
			return
		}

		remaining := limit - count
		if remaining < 0 {
			remaining = 0
		}

		w.Header().Set("X-RateLimit-Limit", strconv.FormatInt(limit, 10))
		w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))

		if count > limit {
			retryAfter := 60 - time.Now().Unix()%60
			w.Header().Set("Retry-After",
				strconv.FormatInt(retryAfter, 10))
			http.Error(w, "rate limit exceeded",
				http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func NewRedisClient(addr string) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:         addr,
		DialTimeout:  3 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})
}

func CheckRedis(ctx context.Context, client *redis.Client) error {
	return client.Ping(ctx).Err()
}
