package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type SupabaseConfig struct {
	URL        string
	ServiceKey string
}

type supabaseClient struct {
	baseURL    string
	serviceKey string
	httpClient *http.Client
}

func NewSupabaseClient(config SupabaseConfig, httpClient *http.Client) SupabaseClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &supabaseClient{
		baseURL:    strings.TrimRight(strings.TrimSpace(config.URL), "/"),
		serviceKey: strings.TrimSpace(config.ServiceKey),
		httpClient: httpClient,
	}
}

func (c *supabaseClient) CreateSignedUploadURL(ctx context.Context, input SignedUploadURLInput) (*SignedUploadURL, error) {
	if err := c.validateConfig(); err != nil {
		return nil, err
	}

	endpoint := c.storageURL("/object/upload/sign/" + url.PathEscape(input.Bucket) + "/" + escapeObjectPath(input.Path))
	body := map[string]any{}
	if input.Upsert {
		body["upsert"] = true
	}

	var raw map[string]any
	if err := c.doJSON(ctx, http.MethodPost, endpoint, body, &raw); err != nil {
		return nil, err
	}

	signedURL := firstString(raw, "signedURL", "signedUrl", "signed_url", "url")
	if signedURL == "" {
		return nil, fmt.Errorf("%w: supabase did not return signed upload url", errSupabaseStorage)
	}

	return &SignedUploadURL{
		Bucket:    input.Bucket,
		Path:      firstNonEmpty(firstString(raw, "path"), input.Path),
		SignedURL: c.absoluteStorageURL(signedURL),
		Token:     firstString(raw, "token"),
	}, nil
}

func (c *supabaseClient) CreateSignedDownloadURL(ctx context.Context, input SignedDownloadURLInput) (*SignedDownloadURL, error) {
	if err := c.validateConfig(); err != nil {
		return nil, err
	}

	endpoint := c.storageURL("/object/sign/" + url.PathEscape(input.Bucket) + "/" + escapeObjectPath(input.Path))
	body := map[string]any{
		"expiresIn": input.ExpiresInSeconds,
	}

	var raw map[string]any
	if err := c.doJSON(ctx, http.MethodPost, endpoint, body, &raw); err != nil {
		return nil, err
	}

	signedURL := firstString(raw, "signedURL", "signedUrl", "signed_url", "url")
	if signedURL == "" {
		return nil, fmt.Errorf("%w: supabase did not return signed download url", errSupabaseStorage)
	}

	return &SignedDownloadURL{
		Bucket:           input.Bucket,
		Path:             input.Path,
		SignedURL:        c.absoluteStorageURL(signedURL),
		ExpiresInSeconds: input.ExpiresInSeconds,
	}, nil
}

func (c *supabaseClient) validateConfig() error {
	if c == nil || c.baseURL == "" || c.serviceKey == "" {
		return fmt.Errorf("%w: supabase storage is not configured", errSupabaseStorage)
	}
	return nil
}

func (c *supabaseClient) storageURL(path string) string {
	return c.baseURL + "/storage/v1" + path
}

func (c *supabaseClient) absoluteStorageURL(value string) string {
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value
	}
	if strings.HasPrefix(value, "/") {
		return c.baseURL + "/storage/v1" + value
	}
	return c.baseURL + "/storage/v1/" + value
}

func (c *supabaseClient) doJSON(ctx context.Context, method, endpoint string, body any, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.serviceKey)
	req.Header.Set("apikey", c.serviceKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("%w: supabase storage returned %d: %s", errSupabaseStorage, resp.StatusCode, strings.TrimSpace(string(data)))
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return err
	}
	return nil
}

func escapeObjectPath(value string) string {
	parts := strings.Split(value, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

func firstString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := values[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

type storageError string

func (e storageError) Error() string {
	return string(e)
}

const errSupabaseStorage storageError = "supabase storage error"
