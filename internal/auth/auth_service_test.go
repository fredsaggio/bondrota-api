package auth

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestGenerateTokenExposesOnlyPublicIdentity(t *testing.T) {
	const publicID = "cli_012345678901234567890"
	svc := NewAuthService(nil, "test-secret")
	token, err := svc.GenerateToken(publicID, RoleCliente)
	if err != nil {
		t.Fatal(err)
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("invalid JWT format: %q", token)
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode JWT payload: %v", err)
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		t.Fatalf("decode JWT claims: %v", err)
	}
	if claims["sub"] != publicID {
		t.Fatalf("unexpected subject: %v", claims["sub"])
	}
	if _, exposed := claims["user_id"]; exposed {
		t.Fatalf("JWT must not expose internal user_id: %s", payload)
	}
}
