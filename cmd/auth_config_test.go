package main

import (
	"net/http"
	"testing"
	"time"
)

func TestParseAllowedOrigins(t *testing.T) {
	origins, err := parseAllowedOrigins(" https://admin.example.com/,http://localhost:3000,https://admin.example.com ")
	if err != nil {
		t.Fatal(err)
	}
	if len(origins) != 2 || origins[0] != "https://admin.example.com" || origins[1] != "http://localhost:3000" {
		t.Fatalf("unexpected origins: %#v", origins)
	}
	if _, err := parseAllowedOrigins("*"); err == nil {
		t.Fatal("expected wildcard origin to be rejected")
	}
}

func TestLoadAdminCookieConfig(t *testing.T) {
	values := map[string]string{
		"AUTH_COOKIE_SECURE":    "true",
		"AUTH_COOKIE_SAME_SITE": "none",
		"AUTH_COOKIE_DOMAIN":    ".bondrota.com",
	}
	config, err := loadAdminCookieConfig(func(key string) string { return values[key] }, []string{"https://admin.bondrota.com"})
	if err != nil {
		t.Fatal(err)
	}
	if !config.Secure || config.SameSite != http.SameSiteNoneMode || config.Domain != ".bondrota.com" {
		t.Fatalf("unexpected cookie config: %#v", config)
	}

	values["AUTH_COOKIE_SECURE"] = "false"
	if _, err := loadAdminCookieConfig(func(key string) string { return values[key] }, []string{"https://admin.bondrota.com"}); err == nil {
		t.Fatal("expected SameSite=None without Secure to be rejected")
	}
}

func TestCookieIsSecureWhenProductionAndLocalOriginsAreMixed(t *testing.T) {
	config, err := loadAdminCookieConfig(
		func(string) string { return "" },
		[]string{"https://admin.bondrota.com", "http://localhost:3000"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !config.Secure {
		t.Fatal("production origin must keep the session cookie secure")
	}
}

func TestLoadLoginRateLimitConfig(t *testing.T) {
	values := map[string]string{
		"LOGIN_RATE_LIMIT_PER_IP":              "30",
		"LOGIN_RATE_LIMIT_PER_IDENTITY":        "6",
		"LOGIN_RATE_LIMIT_WINDOW":              "2m",
		"LOGIN_RATE_LIMIT_TRUST_PROXY_HEADERS": "true",
	}
	config, err := loadLoginRateLimitConfig(func(key string) string { return values[key] })
	if err != nil {
		t.Fatal(err)
	}
	if config.RequestsPerIP != 30 || config.RequestsPerIdentity != 6 || config.Window != 2*time.Minute || !config.TrustProxyHeaders {
		t.Fatalf("unexpected rate limit config: %#v", config)
	}

	values["LOGIN_RATE_LIMIT_PER_IP"] = "0"
	if _, err := loadLoginRateLimitConfig(func(key string) string { return values[key] }); err == nil {
		t.Fatal("expected invalid per-IP limit to be rejected")
	}
}
