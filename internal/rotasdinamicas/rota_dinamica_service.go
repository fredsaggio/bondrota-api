package rotasdinamicas

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
)

const defaultProvider = "osrm"

type rotaDinamicaService struct {
	store RotaDinamicaStore
}

func NewRotaDinamicaService(store RotaDinamicaStore) RotaDinamicaService {
	return &rotaDinamicaService{store: store}
}

func (s *rotaDinamicaService) Create(ctx context.Context, input RotaDinamicaInput) (*RotaDinamicaComDestinos, error) {
	if input.ExpiresAt.IsZero() {
		expiresAt, err := s.store.GetExpiresAtByViagem(ctx, input.ViagemID)
		if err != nil {
			return nil, err
		}
		input.ExpiresAt = expiresAt
	}

	normalized, err := normalizeRotaDinamicaInput(input)
	if err != nil {
		return nil, err
	}

	return s.store.Create(ctx, normalized)
}

func (s *rotaDinamicaService) GetByViagem(ctx context.Context, viagemID int64) (*RotaDinamicaComDestinos, error) {
	if viagemID <= 0 {
		return nil, invalidInput("Selecione a viagem.")
	}

	return s.store.GetByViagem(ctx, viagemID)
}

func (s *rotaDinamicaService) DeleteByViagem(ctx context.Context, viagemID int64) error {
	if viagemID <= 0 {
		return invalidInput("Selecione a viagem.")
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
		return RotaDinamicaInput{}, wrapInvalidInput(err)
	}

	destinos := make([]RotaDinamicaDestinoInput, 0, len(input.Destinos))
	seenDestino := make(map[int64]struct{}, len(input.Destinos))
	for i, destino := range input.Destinos {
		if destino.DestinoID <= 0 {
			return RotaDinamicaInput{}, invalidInput("Selecione o destino.")
		}
		if _, ok := seenDestino[destino.DestinoID]; ok {
			return RotaDinamicaInput{}, invalidInput("O mesmo destino foi informado mais de uma vez.")
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
		return errors.New("Selecione a viagem.")
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
		return errors.New("Selecione ao menos um destino.")
	}

	return nil
}

func invalidInput(message string) error {
	return fmt.Errorf("%w: %s", brerror.ErrInvalidInput, message)
}

func wrapInvalidInput(err error) error {
	return fmt.Errorf("%w: %v", brerror.ErrInvalidInput, err)
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
