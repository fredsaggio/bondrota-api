package storage

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/fredsaggio/bondrota-api/internal/auth"
	"github.com/fredsaggio/bondrota-api/internal/brerror"
)

const (
	defaultDownloadExpirationSeconds = 15 * 60
	minDownloadExpirationSeconds     = 60
	maxDownloadExpirationSeconds     = 60 * 60
)

var (
	fotoContentTypes = map[string]struct{}{
		"image/jpeg": {},
		"image/png":  {},
		"image/webp": {},
	}
	documentoContentTypes = map[string]struct{}{
		"application/pdf": {},
		"image/jpeg":      {},
		"image/png":       {},
		"image/webp":      {},
	}
	fotoExtensoes = map[string]struct{}{
		".jpg":  {},
		".jpeg": {},
		".png":  {},
		".webp": {},
	}
	documentoExtensoes = map[string]struct{}{
		".pdf":  {},
		".jpg":  {},
		".jpeg": {},
		".png":  {},
		".webp": {},
	}
)

type service struct {
	client SupabaseClient
}

func NewService(client SupabaseClient) Service {
	return &service{client: client}
}

func (s *service) CreateSignedUploadURL(ctx context.Context, actor Actor, input SignedUploadURLInput) (*SignedUploadURL, error) {
	if s.client == nil {
		return nil, fmt.Errorf("%w: storage client is not configured", brerror.ErrInvalidInput)
	}
	if err := validateActor(actor); err != nil {
		return nil, err
	}
	input = normalizeUploadInput(input)
	if err := validateUploadInput(actor, input); err != nil {
		return nil, err
	}
	return s.client.CreateSignedUploadURL(ctx, input)
}

func (s *service) CreateSignedDownloadURL(ctx context.Context, actor Actor, input SignedDownloadURLInput) (*SignedDownloadURL, error) {
	if s.client == nil {
		return nil, fmt.Errorf("%w: storage client is not configured", brerror.ErrInvalidInput)
	}
	if err := validateActor(actor); err != nil {
		return nil, err
	}
	input = normalizeDownloadInput(input)
	if err := validateDownloadInput(actor, input); err != nil {
		return nil, err
	}
	return s.client.CreateSignedDownloadURL(ctx, input)
}

func validateActor(actor Actor) error {
	if actor.UserID <= 0 {
		return brerror.ErrUnauthenticated
	}
	switch actor.Role {
	case auth.RoleAdmin, auth.RoleCliente, auth.RoleMotorista:
		return nil
	default:
		return brerror.ErrForbidden
	}
}

func normalizeUploadInput(input SignedUploadURLInput) SignedUploadURLInput {
	input.Bucket = strings.TrimSpace(strings.ToLower(input.Bucket))
	input.Path = normalizeObjectPath(input.Path)
	input.ContentType = strings.TrimSpace(strings.ToLower(input.ContentType))
	return input
}

func normalizeDownloadInput(input SignedDownloadURLInput) SignedDownloadURLInput {
	input.Bucket = strings.TrimSpace(strings.ToLower(input.Bucket))
	input.Path = normalizeObjectPath(input.Path)
	if input.ExpiresInSeconds == 0 {
		input.ExpiresInSeconds = defaultDownloadExpirationSeconds
	}
	return input
}

func validateUploadInput(actor Actor, input SignedUploadURLInput) error {
	if err := validateBucket(input.Bucket); err != nil {
		return err
	}
	if err := validateObjectPath(input.Path); err != nil {
		return err
	}
	if err := validateBucketAccess(actor, input.Bucket, input.Path); err != nil {
		return err
	}
	if err := validateContentType(input.Bucket, input.ContentType); err != nil {
		return err
	}
	return validateExtension(input.Bucket, input.Path)
}

func validateDownloadInput(actor Actor, input SignedDownloadURLInput) error {
	if err := validateBucket(input.Bucket); err != nil {
		return err
	}
	if err := validateObjectPath(input.Path); err != nil {
		return err
	}
	if err := validateBucketAccess(actor, input.Bucket, input.Path); err != nil {
		return err
	}
	if input.ExpiresInSeconds < minDownloadExpirationSeconds || input.ExpiresInSeconds > maxDownloadExpirationSeconds {
		return fmt.Errorf("%w: expires_in_seconds must be between %d and %d", brerror.ErrInvalidInput, minDownloadExpirationSeconds, maxDownloadExpirationSeconds)
	}
	return nil
}

func validateBucket(bucket string) error {
	switch bucket {
	case BucketFotos, BucketDocumentos:
		return nil
	default:
		return fmt.Errorf("%w: bucket must be fotos or documentos", brerror.ErrInvalidInput)
	}
}

func normalizeObjectPath(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "/")
	return value
}

func validateObjectPath(value string) error {
	if value == "" || value == "." {
		return fmt.Errorf("%w: path is required", brerror.ErrInvalidInput)
	}
	if strings.HasPrefix(value, "/") {
		return fmt.Errorf("%w: path must be relative", brerror.ErrInvalidInput)
	}
	parts := strings.Split(value, "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf("%w: path is invalid", brerror.ErrInvalidInput)
		}
	}
	if strings.Contains(value, "\\") {
		return fmt.Errorf("%w: path is invalid", brerror.ErrInvalidInput)
	}
	return nil
}

func validateBucketAccess(actor Actor, bucket, objectPath string) error {
	if actor.Role == auth.RoleAdmin {
		return nil
	}

	switch actor.Role {
	case auth.RoleCliente:
		prefix := "clientes/" + strconv.FormatInt(actor.UserID, 10) + "/"
		if !strings.HasPrefix(objectPath, prefix) {
			return brerror.ErrForbidden
		}
		return nil
	case auth.RoleMotorista:
		if bucket != BucketFotos {
			return brerror.ErrForbidden
		}
		prefix := "motoristas/" + strconv.FormatInt(actor.UserID, 10) + "/"
		if !strings.HasPrefix(objectPath, prefix) {
			return brerror.ErrForbidden
		}
		return nil
	default:
		return brerror.ErrForbidden
	}
}

func validateContentType(bucket, contentType string) error {
	if contentType == "" {
		return fmt.Errorf("%w: content_type is required", brerror.ErrInvalidInput)
	}
	var allowed map[string]struct{}
	switch bucket {
	case BucketFotos:
		allowed = fotoContentTypes
	case BucketDocumentos:
		allowed = documentoContentTypes
	default:
		return errors.New("unreachable bucket")
	}
	if _, ok := allowed[contentType]; !ok {
		return fmt.Errorf("%w: content_type is not allowed for this bucket", brerror.ErrInvalidInput)
	}
	return nil
}

func validateExtension(bucket, objectPath string) error {
	ext := strings.ToLower(path.Ext(objectPath))
	var allowed map[string]struct{}
	switch bucket {
	case BucketFotos:
		allowed = fotoExtensoes
	case BucketDocumentos:
		allowed = documentoExtensoes
	default:
		return errors.New("unreachable bucket")
	}
	if _, ok := allowed[ext]; !ok {
		return fmt.Errorf("%w: file extension is not allowed for this bucket", brerror.ErrInvalidInput)
	}
	return nil
}
