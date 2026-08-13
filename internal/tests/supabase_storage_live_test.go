package tests

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/auth"
	"github.com/fredsaggio/bondrota-api/internal/crypto"
	"github.com/fredsaggio/bondrota-api/internal/publicid"
	"github.com/fredsaggio/bondrota-api/internal/storage"
	"github.com/go-chi/chi/v5"
)

func TestLiveSupabaseStorageSignedUploadDownload(t *testing.T) {
	if strings.TrimSpace(os.Getenv("SUPABASE_STORAGE_LIVE_TEST")) != "1" {
		t.Skip("set SUPABASE_STORAGE_LIVE_TEST=1 to run live Supabase storage tests")
	}

	supabaseURL := strings.TrimSpace(os.Getenv("SUPABASE_URL"))
	serviceKey := strings.TrimSpace(os.Getenv("SUPABASE_SERVICE_KEY"))
	if supabaseURL == "" || serviceKey == "" {
		t.Fatal("SUPABASE_URL and SUPABASE_SERVICE_KEY are required for live Supabase storage tests")
	}

	authSvc := auth.NewAuthService(crypto.NewBcryptHasher(crypto.DefaultCost), "live-supabase-storage-secret")
	router := newLiveStorageRouter(authSvc, storage.SupabaseConfig{
		URL:        supabaseURL,
		ServiceKey: serviceKey,
	})

	clienteID := "cli_012345678901234567890"
	clienteToken, err := authSvc.GenerateToken(clienteID, auth.RoleCliente)
	if err != nil {
		t.Fatalf("generate cliente token: %v", err)
	}

	suffix := time.Now().UnixNano()
	cases := []struct {
		name        string
		bucket      string
		path        string
		contentType string
		payload     []byte
	}{
		{
			name:        "foto",
			bucket:      storage.BucketFotos,
			path:        fmt.Sprintf("clientes/%s/e2e/live-%d.png", clienteID, suffix),
			contentType: "image/png",
			payload:     []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a},
		},
		{
			name:        "documento",
			bucket:      storage.BucketDocumentos,
			path:        fmt.Sprintf("clientes/%s/e2e/live-%d.pdf", clienteID, suffix),
			contentType: "application/pdf",
			payload:     []byte("%PDF-1.4\n% bondrota e2e\n"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			upload := doJSON[map[string]any](t, router, http.MethodPost, "/api/v1/storage/signed-upload-url", clienteToken, map[string]any{
				"bucket":       tc.bucket,
				"path":         tc.path,
				"content_type": tc.contentType,
				"upsert":       true,
			}, http.StatusCreated)
			if upload["bucket"] != tc.bucket {
				t.Fatalf("unexpected upload bucket: %v", upload["bucket"])
			}
			if upload["path"] != tc.path {
				t.Fatalf("unexpected upload path: %v", upload["path"])
			}

			putReq, err := http.NewRequest(http.MethodPut, upload["signed_url"].(string), bytes.NewReader(tc.payload))
			if err != nil {
				t.Fatalf("create upload request: %v", err)
			}
			putReq.Header.Set("Content-Type", tc.contentType)

			httpClient := &http.Client{Timeout: 30 * time.Second}
			putResp, err := httpClient.Do(putReq)
			if err != nil {
				t.Fatalf("upload to Supabase signed URL: %v", err)
			}
			defer putResp.Body.Close()
			if putResp.StatusCode < 200 || putResp.StatusCode >= 300 {
				body, _ := io.ReadAll(io.LimitReader(putResp.Body, 4096))
				t.Fatalf("upload to Supabase signed URL: status %d: %s", putResp.StatusCode, strings.TrimSpace(string(body)))
			}

			download := doJSON[map[string]any](t, router, http.MethodPost, "/api/v1/storage/signed-download-url", clienteToken, map[string]any{
				"bucket":             tc.bucket,
				"path":               tc.path,
				"expires_in_seconds": 900,
			}, http.StatusOK)

			getResp, err := httpClient.Get(download["signed_url"].(string))
			if err != nil {
				t.Fatalf("download from Supabase signed URL: %v", err)
			}
			defer getResp.Body.Close()
			if getResp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(io.LimitReader(getResp.Body, 4096))
				t.Fatalf("download from Supabase signed URL: status %d: %s", getResp.StatusCode, strings.TrimSpace(string(body)))
			}

			got, err := io.ReadAll(getResp.Body)
			if err != nil {
				t.Fatalf("read downloaded object: %v", err)
			}
			if !bytes.Equal(got, tc.payload) {
				t.Fatalf("downloaded payload mismatch: got %q, want %q", string(got), string(tc.payload))
			}

			t.Logf("uploaded live Supabase object: bucket=%s path=%s", tc.bucket, tc.path)
		})
	}
}

func newLiveStorageRouter(authSvc *auth.AuthService, config storage.SupabaseConfig) http.Handler {
	authSvc.SetIdentityResolver(liveStorageIdentityResolver{})
	storageHandler := storage.NewHandler(storage.NewService(storage.NewSupabaseClient(config, &http.Client{
		Timeout: 30 * time.Second,
	})))

	r := chi.NewRouter()
	r.Route("/api/v1/storage", func(r chi.Router) {
		r.Use(authSvc.Authenticate)
		r.Use(authSvc.RequireRole(auth.RoleAdmin, auth.RoleCliente, auth.RoleMotorista))
		r.Post("/signed-upload-url", storageHandler.CreateSignedUploadURL)
		r.Post("/signed-download-url", storageHandler.CreateSignedDownloadURL)
	})
	return r
}

type liveStorageIdentityResolver struct{}

func (liveStorageIdentityResolver) Resolve(_ context.Context, _ publicid.Prefix, _ string) (int64, error) {
	return 777777, nil
}
