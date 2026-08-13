package storage

import "context"

const (
	BucketFotos      = "fotos"
	BucketDocumentos = "documentos"
)

type Actor struct {
	UserID   int64
	PublicID string
	Role     string
}

type SignedUploadURLInput struct {
	Bucket      string
	Path        string
	ContentType string
	Upsert      bool
}

type SignedDownloadURLInput struct {
	Bucket           string
	Path             string
	ExpiresInSeconds int
}

type SignedUploadURL struct {
	Bucket    string `json:"bucket"`
	Path      string `json:"path"`
	SignedURL string `json:"signed_url"`
	Token     string `json:"token,omitempty"`
}

type SignedDownloadURL struct {
	Bucket           string `json:"bucket"`
	Path             string `json:"path"`
	SignedURL        string `json:"signed_url"`
	ExpiresInSeconds int    `json:"expires_in_seconds"`
}

type SupabaseClient interface {
	CreateSignedUploadURL(ctx context.Context, input SignedUploadURLInput) (*SignedUploadURL, error)
	CreateSignedDownloadURL(ctx context.Context, input SignedDownloadURLInput) (*SignedDownloadURL, error)
	MoveObject(ctx context.Context, bucket, from, to string) error
}

type Service interface {
	CreateSignedUploadURL(ctx context.Context, actor Actor, input SignedUploadURLInput) (*SignedUploadURL, error)
	CreateSignedDownloadURL(ctx context.Context, actor Actor, input SignedDownloadURLInput) (*SignedDownloadURL, error)
	// MoveObject nao passa por actor de proposito: quem chama e sempre o proprio
	// backend, logo depois de criar um motorista/cliente/vinculo, para levar o
	// arquivo da pasta de espera (onde foi enviado antes do registro ter ID)
	// para o caminho definitivo. Nunca e exposto por HTTP.
	MoveObject(ctx context.Context, bucket, from, to string) error
}
