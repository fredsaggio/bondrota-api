package rotasdinamicas

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

const defaultProvider = "osrm"

type rotaDinamicaService struct {
	store RotaDinamicaStore
}

func NewRotaDinamicaService(store RotaDinamicaStore) RotaDinamicaService {
	return &rotaDinamicaService{store: store}
}

func (s *rotaDinamicaService) Create(ctx context.Context, input RotaDinamicaInput) (*RotaDinamicaComDestinos, error) {
	normalized, err := normalizeRotaDinamicaInput(input)
	if err != nil {
		return nil, err
	}

	return s.store.Create(ctx, normalized)
}

func (s *rotaDinamicaService) GetByViagem(ctx context.Context, viagemID int64) (*RotaDinamicaComDestinos, error) {
	if viagemID <= 0 {
		return nil, errors.New("viagem_id is required")
	}

	return s.store.GetByViagem(ctx, viagemID)
}

func (s *rotaDinamicaService) DeleteByViagem(ctx context.Context, viagemID int64) error {
	if viagemID <= 0 {
		return errors.New("viagem_id is required")
	}

	return s.store.DeleteByViagem(ctx, viagemID)
}

func normalizeRotaDinamicaInput(input RotaDinamicaInput) (RotaDinamicaInput, error) {
	input.Provider = strings.TrimSpace(input.Provider)
	if input.Provider == "" {
		input.Provider = defaultProvider
	}

	input.Origem.Nome = strings.TrimSpace(input.Origem.Nome)
	input.DestinoFinal.Nome = strings.TrimSpace(input.DestinoFinal.Nome)

	if err := validateRotaDinamicaInput(input); err != nil {
		return RotaDinamicaInput{}, err
	}

	destinos := make([]RotaDinamicaDestinoInput, 0, len(input.Destinos))
	seenDestino := make(map[int64]struct{}, len(input.Destinos))
	for i, destino := range input.Destinos {
		if destino.DestinoID <= 0 {
			return RotaDinamicaInput{}, errors.New("destino_id is required")
		}
		if _, ok := seenDestino[destino.DestinoID]; ok {
			return RotaDinamicaInput{}, errors.New("destino_id duplicated")
		}
		seenDestino[destino.DestinoID] = struct{}{}

		destinos = append(destinos, RotaDinamicaDestinoInput{
			DestinoID: destino.DestinoID,
			Ordem:     i + 1,
		})
	}
	input.Destinos = destinos

	return input, nil
}

func validateRotaDinamicaInput(input RotaDinamicaInput) error {
	if input.ViagemID <= 0 {
		return errors.New("viagem_id is required")
	}
	if input.Provider == "" {
		return errors.New("provider is required")
	}
	if err := validatePontoRota("origem", input.Origem); err != nil {
		return err
	}
	if err := validatePontoRota("destino_final", input.DestinoFinal); err != nil {
		return err
	}
	if input.DistanciaMetros <= 0 {
		return errors.New("distancia_metros must be greater than zero")
	}
	if input.DuracaoSegundos <= 0 {
		return errors.New("duracao_segundos must be greater than zero")
	}
	if !json.Valid(input.Geometry) {
		return errors.New("geometry must be valid json")
	}
	if input.ExpiresAt.IsZero() {
		return errors.New("expires_at is required")
	}
	if len(input.Destinos) == 0 {
		return errors.New("destinos is required")
	}

	return nil
}

func validatePontoRota(field string, ponto PontoRota) error {
	if ponto.Nome == "" {
		return errors.New(field + ".nome is required")
	}
	if ponto.Latitude == 0 && ponto.Longitude == 0 {
		return errors.New(field + " coordinates are required")
	}
	if ponto.Latitude < -90 || ponto.Latitude > 90 {
		return errors.New(field + ".latitude must be between -90 and 90")
	}
	if ponto.Longitude < -180 || ponto.Longitude > 180 {
		return errors.New(field + ".longitude must be between -180 and 180")
	}

	return nil
}
