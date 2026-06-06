package rotasdinamicas

import (
	"context"
	"encoding/json"
	"time"
)

type PontoRota struct {
	Nome      string
	Latitude  float64
	Longitude float64
}

type RotaDinamica struct {
	ID                    int64
	ViagemID              int64
	Provider              string
	OrigemNome            string
	OrigemLatitude        float64
	OrigemLongitude       float64
	DestinoFinalNome      string
	DestinoFinalLatitude  float64
	DestinoFinalLongitude float64
	DistanciaMetros       int
	DuracaoSegundos       int
	Geometry              json.RawMessage
	ExpiresAt             time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type RotaDinamicaDestino struct {
	ID             int64
	RotaDinamicaID int64
	DestinoID      int64
	Ordem          int
	CreatedAt      time.Time
}

type RotaDinamicaInput struct {
	ViagemID        int64
	Provider        string
	Origem          PontoRota
	DestinoFinal    PontoRota
	DistanciaMetros int
	DuracaoSegundos int
	Geometry        json.RawMessage
	ExpiresAt       time.Time
	Destinos        []RotaDinamicaDestinoInput
}

type RotaDinamicaDestinoInput struct {
	DestinoID int64
	Ordem     int
}

type RotaDinamicaComDestinos struct {
	Rota     RotaDinamica
	Destinos []RotaDinamicaDestino
}

type RotaDinamicaStore interface {
	Create(ctx context.Context, input RotaDinamicaInput) (*RotaDinamicaComDestinos, error)
	GetByViagem(ctx context.Context, viagemID int64) (*RotaDinamicaComDestinos, error)
	ListDestinos(ctx context.Context, rotaDinamicaID int64) ([]RotaDinamicaDestino, error)
	DeleteByViagem(ctx context.Context, viagemID int64) error
}

type RotaDinamicaService interface {
	Create(ctx context.Context, input RotaDinamicaInput) (*RotaDinamicaComDestinos, error)
	GetByViagem(ctx context.Context, viagemID int64) (*RotaDinamicaComDestinos, error)
	DeleteByViagem(ctx context.Context, viagemID int64) error
}
