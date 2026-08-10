package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLoginRateLimiterByIP(t *testing.T) {
	limiter := newLoginRateLimiter(LoginRateLimitConfig{
		RequestsPerIP:       2,
		RequestsPerIdentity: 100,
		Window:              time.Minute,
	})
	handler := limiter.middleware("admin", "email")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	for attempt := 1; attempt <= 3; attempt++ {
		request := httptest.NewRequest(http.MethodPost, "/admin/login", bytes.NewBufferString(
			`{"email":"admin`+string(rune('0'+attempt))+`@example.com","senha":"secret"}`,
		))
		request.RemoteAddr = "192.0.2.10:1234"
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)

		expected := http.StatusNoContent
		if attempt == 3 {
			expected = http.StatusTooManyRequests
			if response.Header().Get("Retry-After") == "" {
				t.Fatal("expected Retry-After header")
			}
		}
		if response.Code != expected {
			t.Fatalf("attempt %d: want %d, got %d", attempt, expected, response.Code)
		}
	}
}

func TestLoginRateLimiterByNormalizedIdentity(t *testing.T) {
	limiter := newLoginRateLimiter(LoginRateLimitConfig{
		RequestsPerIP:       100,
		RequestsPerIdentity: 2,
		Window:              time.Minute,
		TrustProxyHeaders:   true,
	})
	handler := limiter.middleware("cliente", "cpf")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	bodies := []string{
		`{"cpf":"123.456.789-00","senha":"one"}`,
		`{"cpf":"12345678900","senha":"two"}`,
		`{"cpf":"123 456 789 00","senha":"three"}`,
	}

	for index, body := range bodies {
		request := httptest.NewRequest(http.MethodPost, "/clientes/login", bytes.NewBufferString(body))
		request.Header.Set("X-Forwarded-For", "192.0.2."+string(rune('1'+index)))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)

		expected := http.StatusNoContent
		if index == 2 {
			expected = http.StatusTooManyRequests
		}
		if response.Code != expected {
			t.Fatalf("attempt %d: want %d, got %d", index+1, expected, response.Code)
		}
	}
}

func TestLoginRateLimiterDoesNotTrustSpoofedProxyHeaderByDefault(t *testing.T) {
	limiter := newLoginRateLimiter(LoginRateLimitConfig{
		RequestsPerIP:       1,
		RequestsPerIdentity: 100,
		Window:              time.Minute,
	})
	handler := limiter.middleware("admin", "email")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	for index, forwardedIP := range []string{"192.0.2.1", "192.0.2.2"} {
		request := httptest.NewRequest(http.MethodPost, "/admin/login", bytes.NewBufferString(
			`{"email":"admin@example.com","senha":"secret"}`,
		))
		request.RemoteAddr = "198.51.100.10:1234"
		request.Header.Set("X-Forwarded-For", forwardedIP)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)

		expected := http.StatusNoContent
		if index == 1 {
			expected = http.StatusTooManyRequests
		}
		if response.Code != expected {
			t.Fatalf("attempt %d: want %d, got %d", index+1, expected, response.Code)
		}
	}
}

func TestFixedWindowLimiterResetsAndCleansExpiredEntries(t *testing.T) {
	now := time.Date(2026, time.August, 10, 12, 0, 0, 0, time.UTC)
	limiter := newFixedWindowLimiter(1, time.Minute)
	limiter.now = func() time.Time { return now }

	if allowed, _ := limiter.allow("first"); !allowed {
		t.Fatal("first attempt should be allowed")
	}
	if allowed, _ := limiter.allow("first"); allowed {
		t.Fatal("second attempt in same window should be blocked")
	}
	now = now.Add(time.Minute)
	if allowed, _ := limiter.allow("second"); !allowed {
		t.Fatal("attempt after the window should be allowed")
	}
	if _, exists := limiter.entries["first"]; exists {
		t.Fatal("expired entry should have been cleaned")
	}
}

func TestReadLoginIdentityRestoresRequestBody(t *testing.T) {
	body := []byte(`{"email":" Admin@Example.COM ","senha":"secret"}`)
	request := httptest.NewRequest(http.MethodPost, "/admin/login", bytes.NewReader(body))
	if identity := readLoginIdentity(request, "email"); identity != "admin@example.com" {
		t.Fatalf("unexpected identity %q", identity)
	}
	restored := make([]byte, len(body))
	if _, err := request.Body.Read(restored); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(restored, body) {
		t.Fatalf("request body was not restored: %q", restored)
	}
}
