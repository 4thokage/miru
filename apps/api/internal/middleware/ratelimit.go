package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
)

type RateLimitConfig struct {
	RequestsPerMinute int
	Burst             int
}

func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		RequestsPerMinute: 60,
		Burst:             10,
	}
}

func RateLimiter(config *RateLimitConfig) func(http.Handler) http.Handler {
	if config == nil {
		config = DefaultRateLimitConfig()
	}

	return httprate.Limit(
		config.RequestsPerMinute,
		time.Minute,
		httprate.WithKeyFuncs(
			httprate.KeyByIP,
			httprate.KeyByRealIP,
		),
	)
}

func IPBasedRateLimiter(requestsPerMinute int) func(http.Handler) http.Handler {
	return httprate.LimitByIP(requestsPerMinute, time.Minute)
}

func TieredRateLimiter() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path

			var limit int
			switch {
			case strings.Contains(path, "/search"):
				limit = 30
			case strings.Contains(path, "/chapter"):
				limit = 60
			case strings.Contains(path, "/sources"):
				limit = 120
			default:
				limit = 100
			}

			httprate.Limit(limit, time.Minute)(next).ServeHTTP(w, r)
		})
	}
}

func NewRateLimiterFromEnv() func(http.Handler) http.Handler {
	return TieredRateLimiter()
}

func RateLimitRoutes(r *chi.Mux) {
	r.Use(NewRateLimitFromEnv())
}

func NewRateLimitFromEnv() func(http.Handler) http.Handler {
	return TieredRateLimiter()
}
