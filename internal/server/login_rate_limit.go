package server

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/auth"
)

const (
	defaultLoginRequestsPerIP       = 20
	defaultLoginRequestsPerIdentity = 5
	defaultLoginRateLimitWindow     = time.Minute
)

type LoginRateLimitConfig struct {
	RequestsPerIP       int
	RequestsPerIdentity int
	Window              time.Duration
	TrustProxyHeaders   bool
}

func (config LoginRateLimitConfig) withDefaults() LoginRateLimitConfig {
	if config.RequestsPerIP <= 0 {
		config.RequestsPerIP = defaultLoginRequestsPerIP
	}
	if config.RequestsPerIdentity <= 0 {
		config.RequestsPerIdentity = defaultLoginRequestsPerIdentity
	}
	if config.Window <= 0 {
		config.Window = defaultLoginRateLimitWindow
	}
	return config
}

type rateLimitEntry struct {
	count   int
	resetAt time.Time
}

type fixedWindowLimiter struct {
	mu          sync.Mutex
	entries     map[string]rateLimitEntry
	limit       int
	window      time.Duration
	now         func() time.Time
	lastCleanup time.Time
}

func newFixedWindowLimiter(limit int, window time.Duration) *fixedWindowLimiter {
	return &fixedWindowLimiter{
		entries: make(map[string]rateLimitEntry),
		limit:   limit,
		window:  window,
		now:     time.Now,
	}
}

func (limiter *fixedWindowLimiter) allow(key string) (bool, time.Duration) {
	now := limiter.now()
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	if limiter.lastCleanup.IsZero() || now.Sub(limiter.lastCleanup) >= limiter.window {
		for existingKey, entry := range limiter.entries {
			if !now.Before(entry.resetAt) {
				delete(limiter.entries, existingKey)
			}
		}
		limiter.lastCleanup = now
	}

	entry, exists := limiter.entries[key]
	if !exists || !now.Before(entry.resetAt) {
		limiter.entries[key] = rateLimitEntry{count: 1, resetAt: now.Add(limiter.window)}
		return true, 0
	}
	if entry.count >= limiter.limit {
		return false, entry.resetAt.Sub(now)
	}

	entry.count++
	limiter.entries[key] = entry
	return true, 0
}

type loginRateLimiter struct {
	byIP              *fixedWindowLimiter
	byIdentity        *fixedWindowLimiter
	trustProxyHeaders bool
}

func newLoginRateLimiter(config LoginRateLimitConfig) *loginRateLimiter {
	config = config.withDefaults()
	return &loginRateLimiter{
		byIP:              newFixedWindowLimiter(config.RequestsPerIP, config.Window),
		byIdentity:        newFixedWindowLimiter(config.RequestsPerIdentity, config.Window),
		trustProxyHeaders: config.TrustProxyHeaders,
	}
}

func (limiter *loginRateLimiter) middleware(loginType, identityField string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r, limiter.trustProxyHeaders)
			if allowed, retryAfter := limiter.byIP.allow(ip); !allowed {
				respondRateLimited(w, retryAfter)
				return
			}

			identity := readLoginIdentity(r, identityField)
			if identity != "" {
				key := hashedIdentityKey(loginType, identity)
				if allowed, retryAfter := limiter.byIdentity.allow(key); !allowed {
					respondRateLimited(w, retryAfter)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// authenticatedMiddleware limita por usuario autenticado, para endpoints que conferem
// credencial mas nao trazem identidade no corpo — hoje so a troca de senha. Sem isso
// uma sessao roubada poderia forcar a senha atual no limite do teto por IP.
func (limiter *loginRateLimiter) authenticatedMiddleware(scope string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r, limiter.trustProxyHeaders)
			if allowed, retryAfter := limiter.byIP.allow(ip); !allowed {
				respondRateLimited(w, retryAfter)
				return
			}

			if claims, ok := r.Context().Value(auth.ClaimsKey).(*auth.Claims); ok && claims.UserID > 0 {
				key := hashedIdentityKey(scope, strconv.FormatInt(claims.UserID, 10))
				if allowed, retryAfter := limiter.byIdentity.allow(key); !allowed {
					respondRateLimited(w, retryAfter)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func readLoginIdentity(r *http.Request, field string) string {
	if r.Body == nil {
		return ""
	}
	body, err := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewReader(body))
	if err != nil {
		return ""
	}

	var payload map[string]string
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	identity := strings.ToLower(strings.TrimSpace(payload[field]))
	if field == "cpf" {
		var digits strings.Builder
		for _, char := range identity {
			if char >= '0' && char <= '9' {
				digits.WriteRune(char)
			}
		}
		return digits.String()
	}
	return identity
}

func hashedIdentityKey(loginType, identity string) string {
	sum := sha256.Sum256([]byte(identity))
	return loginType + ":" + hex.EncodeToString(sum[:])
}

func clientIP(r *http.Request, trustProxyHeaders bool) string {
	if trustProxyHeaders {
		for _, candidate := range strings.Split(r.Header.Get("X-Forwarded-For"), ",") {
			if ip := net.ParseIP(strings.TrimSpace(candidate)); ip != nil {
				return ip.String()
			}
		}
		if ip := net.ParseIP(strings.TrimSpace(r.Header.Get("X-Real-IP"))); ip != nil {
			return ip.String()
		}
	}

	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil {
		if ip := net.ParseIP(host); ip != nil {
			return ip.String()
		}
		return host
	}
	if ip := net.ParseIP(strings.TrimSpace(r.RemoteAddr)); ip != nil {
		return ip.String()
	}
	return strings.TrimSpace(r.RemoteAddr)
}

func respondRateLimited(w http.ResponseWriter, retryAfter time.Duration) {
	seconds := int(math.Ceil(retryAfter.Seconds()))
	if seconds < 1 {
		seconds = 1
	}
	w.Header().Set("Retry-After", strconv.Itoa(seconds))
	http.Error(w, "too many login attempts", http.StatusTooManyRequests)
}
