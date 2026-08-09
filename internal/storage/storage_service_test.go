package storage_test

import (
	"context"
	"testing"

	"github.com/fredsaggio/bondrota-api/internal/auth"
	"github.com/fredsaggio/bondrota-api/internal/brerror"
	"github.com/fredsaggio/bondrota-api/internal/storage"
	"github.com/stretchr/testify/require"
)

type fakeSupabaseClient struct {
	uploadInput   storage.SignedUploadURLInput
	downloadInput storage.SignedDownloadURLInput
}

func (c *fakeSupabaseClient) CreateSignedUploadURL(_ context.Context, input storage.SignedUploadURLInput) (*storage.SignedUploadURL, error) {
	c.uploadInput = input
	return &storage.SignedUploadURL{
		Bucket:    input.Bucket,
		Path:      input.Path,
		SignedURL: "https://example.com/upload",
		Token:     "token",
	}, nil
}

func (c *fakeSupabaseClient) CreateSignedDownloadURL(_ context.Context, input storage.SignedDownloadURLInput) (*storage.SignedDownloadURL, error) {
	c.downloadInput = input
	return &storage.SignedDownloadURL{
		Bucket:           input.Bucket,
		Path:             input.Path,
		SignedURL:        "https://example.com/download",
		ExpiresInSeconds: input.ExpiresInSeconds,
	}, nil
}

func TestService_CreateSignedUploadURLClienteOwnPath(t *testing.T) {
	client := &fakeSupabaseClient{}
	svc := storage.NewService(client)

	resp, err := svc.CreateSignedUploadURL(context.Background(), storage.Actor{
		UserID: 10,
		Role:   auth.RoleCliente,
	}, storage.SignedUploadURLInput{
		Bucket:      "fotos",
		Path:        "clientes/10/foto.png",
		ContentType: "image/png",
	})

	require.NoError(t, err)
	require.Equal(t, "https://example.com/upload", resp.SignedURL)
	require.Equal(t, "clientes/10/foto.png", client.uploadInput.Path)
}

func TestService_CreateSignedUploadURLClienteOtherPathForbidden(t *testing.T) {
	svc := storage.NewService(&fakeSupabaseClient{})

	_, err := svc.CreateSignedUploadURL(context.Background(), storage.Actor{
		UserID: 10,
		Role:   auth.RoleCliente,
	}, storage.SignedUploadURLInput{
		Bucket:      "fotos",
		Path:        "clientes/11/foto.png",
		ContentType: "image/png",
	})

	require.ErrorIs(t, err, brerror.ErrForbidden)
}

func TestService_CreateSignedUploadURLMotoristaCannotUseDocumentos(t *testing.T) {
	svc := storage.NewService(&fakeSupabaseClient{})

	_, err := svc.CreateSignedUploadURL(context.Background(), storage.Actor{
		UserID: 5,
		Role:   auth.RoleMotorista,
	}, storage.SignedUploadURLInput{
		Bucket:      "documentos",
		Path:        "motoristas/5/cnh.pdf",
		ContentType: "application/pdf",
	})

	require.ErrorIs(t, err, brerror.ErrForbidden)
}

func TestService_CreateSignedDownloadURLDefaultsExpiration(t *testing.T) {
	client := &fakeSupabaseClient{}
	svc := storage.NewService(client)

	resp, err := svc.CreateSignedDownloadURL(context.Background(), storage.Actor{
		UserID: 1,
		Role:   auth.RoleAdmin,
	}, storage.SignedDownloadURLInput{
		Bucket: "documentos",
		Path:   "clientes/10/vinculos/20/comprovante.pdf",
	})

	require.NoError(t, err)
	require.Equal(t, 15*60, resp.ExpiresInSeconds)
	require.Equal(t, 15*60, client.downloadInput.ExpiresInSeconds)
}

func TestService_CreateSignedUploadURLInvalidContentType(t *testing.T) {
	svc := storage.NewService(&fakeSupabaseClient{})

	_, err := svc.CreateSignedUploadURL(context.Background(), storage.Actor{
		UserID: 1,
		Role:   auth.RoleAdmin,
	}, storage.SignedUploadURLInput{
		Bucket:      "fotos",
		Path:        "clientes/10/foto.exe",
		ContentType: "application/octet-stream",
	})

	require.ErrorIs(t, err, brerror.ErrInvalidInput)
}

func TestService_CreateSignedUploadURLRejectsDotDotPath(t *testing.T) {
	svc := storage.NewService(&fakeSupabaseClient{})

	_, err := svc.CreateSignedUploadURL(context.Background(), storage.Actor{
		UserID: 10,
		Role:   auth.RoleCliente,
	}, storage.SignedUploadURLInput{
		Bucket:      "fotos",
		Path:        "clientes/10/../foto.png",
		ContentType: "image/png",
	})

	require.ErrorIs(t, err, brerror.ErrInvalidInput)
}
