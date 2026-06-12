package observability

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Timeout returns middleware that cancels requests exceeding d.
func Timeout(d time.Duration) func(http.Handler) http.Handler {
	if d <= 0 {
		return func(next http.Handler) http.Handler { return next }
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), d)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RateLimit limits requests per second (0 disables).
func RateLimit(rps float64, burst int) func(http.Handler) http.Handler {
	if rps <= 0 {
		return func(next http.Handler) http.Handler { return next }
	}
	if burst <= 0 {
		burst = int(rps)
		if burst < 1 {
			burst = 1
		}
	}
	lim := rate.NewLimiter(rate.Limit(rps), burst)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !lim.Allow() {
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Breaker is a minimal circuit breaker for outbound calls.
type Breaker struct {
	mu       sync.Mutex
	failures int
	open     bool
	until    time.Time
	maxFails int
	cooldown time.Duration
}

func NewBreaker(maxFails int, cooldown time.Duration) *Breaker {
	if maxFails <= 0 {
		maxFails = 5
	}
	if cooldown <= 0 {
		cooldown = 30 * time.Second
	}
	return &Breaker{maxFails: maxFails, cooldown: cooldown}
}

func (b *Breaker) Allow() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.open {
		return nil
	}
	if time.Now().After(b.until) {
		b.open = false
		b.failures = 0
		return nil
	}
	return errBreakerOpen
}

func (b *Breaker) Success() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures = 0
	b.open = false
}

func (b *Breaker) Fail() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures++
	if b.failures >= b.maxFails {
		b.open = true
		b.until = time.Now().Add(b.cooldown)
	}
}

var errBreakerOpen = &breakerError{}

type breakerError struct{}

func (e *breakerError) Error() string { return "circuit breaker open" }

// ConfigFromEnv reads optional observability tuning from the environment.
type Config struct {
	RequestTimeout time.Duration
	RateLimitRPS   float64
	RateLimitBurst int
	MetricsEnabled bool
	OTelEnabled    bool
	BreakerMaxFail int
	BreakerCooldown time.Duration
}

func ConfigFromEnv() Config {
	cfg := Config{
		RequestTimeout: 60 * time.Second,
		BreakerMaxFail: 5,
		BreakerCooldown: 30 * time.Second,
	}
	if v := os.Getenv("DATASPAN_REQUEST_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.RequestTimeout = d
		}
	}
	if v := os.Getenv("DATASPAN_RATE_LIMIT_RPS"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.RateLimitRPS = f
		}
	}
	if v := os.Getenv("DATASPAN_RATE_LIMIT_BURST"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.RateLimitBurst = n
		}
	}
	cfg.MetricsEnabled = os.Getenv("DATASPAN_METRICS") == "1"
	cfg.OTelEnabled = os.Getenv("DATASPAN_OTEL") == "1"
	if v := os.Getenv("DATASPAN_BREAKER_MAX_FAILS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.BreakerMaxFail = n
		}
	}
	return cfg
}

// Middleware chains timeout and rate limiting.
func Middleware(cfg Config, next http.Handler) http.Handler {
	h := next
	h = Timeout(cfg.RequestTimeout)(h)
	h = RateLimit(cfg.RateLimitRPS, cfg.RateLimitBurst)(h)
	if cfg.MetricsEnabled {
		h = MetricsHandler()(h)
	}
	if cfg.OTelEnabled {
		h = OTelHandler("dataspan")(h)
	}
	return h
}
