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

func TestCoordinateEnv(t *testing.T) {
	values := map[string]string{
		"BASE_CITY_LATITUDE":  " -9.7817 ",
		"BASE_CITY_LONGITUDE": "-36.3506",
	}
	getEnv := func(key string) string { return values[key] }

	latitude, err := coordinateEnv(getEnv, "BASE_CITY_LATITUDE", 90)
	if err != nil {
		t.Fatal(err)
	}
	if latitude != -9.7817 {
		t.Fatalf("unexpected latitude: %v", latitude)
	}

	longitude, err := coordinateEnv(getEnv, "BASE_CITY_LONGITUDE", 180)
	if err != nil {
		t.Fatal(err)
	}
	if longitude != -36.3506 {
		t.Fatalf("unexpected longitude: %v", longitude)
	}

	if _, err := coordinateEnv(func(string) string { return "" }, "BASE_CITY_LATITUDE", 90); err == nil {
		t.Fatal("expected a missing coordinate to be rejected")
	}
	if _, err := coordinateEnv(func(string) string { return "sul" }, "BASE_CITY_LATITUDE", 90); err == nil {
		t.Fatal("expected a non-numeric coordinate to be rejected")
	}
	// Longitude valida nao pode passar por latitude: -100 cabe em [-180, 180]
	// mas esta fora de [-90, 90], e trocar os dois campos e o erro mais provavel.
	if _, err := coordinateEnv(func(string) string { return "-100" }, "BASE_CITY_LATITUDE", 90); err == nil {
		t.Fatal("expected an out-of-range latitude to be rejected")
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
