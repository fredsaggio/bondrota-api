package rotasdinamicas

import (
	"context"
	"errors"
	"fmt"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/jackc/pgx/v5"
)

type calculadorRotaDinamicaStore struct {
	db db.DB
}

func NewCalculadorRotaDinamicaStore(db db.DB) CalculadorRotaDinamicaStore {
	return &calculadorRotaDinamicaStore{db: db}
}

func (s *calculadorRotaDinamicaStore) GetDadosCalculo(ctx context.Context, viagemID int64) (*DadosCalculoRota, error) {
	const op = "db/calculadorRotaDinamicaStore.GetDadosCalculo"

	dados, rotaInternaID, err := getDadosBaseCalculo(ctx, s.db, viagemID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, brerror.ErrNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	paradas, err := listParadasCalculo(ctx, s.db, rotaInternaID)
	if err != nil {
		return nil, fmt.Errorf("%s: list paradas: %w", op, err)
	}

	destinos, err := listDestinosCalculo(ctx, s.db, viagemID)
	if err != nil {
		return nil, fmt.Errorf("%s: list destinos: %w", op, err)
	}

	dados.Paradas = paradas
	dados.Destinos = destinos

	return dados, nil
}

func getDadosBaseCalculo(ctx context.Context, querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}, viagemID int64) (*DadosCalculoRota, int64, error) {
	const q = `
		SELECT v.id, v.sentido, c.expires_at, c.rota_interna_id
		FROM viagens v
		JOIN ciclos_viagem c ON c.id = v.ciclo_viagem_id
		WHERE v.id = @viagem_id
	`

	rows, err := querier.Query(ctx, q, pgx.StrictNamedArgs{"viagem_id": viagemID})
	if err != nil {
		return nil, 0, err
	}

	var rotaInternaID int64
	dados, err := pgx.CollectExactlyOneRow(rows, func(row pgx.CollectableRow) (DadosCalculoRota, error) {
		var data DadosCalculoRota
		err := row.Scan(
			&data.ViagemID,
			&data.Sentido,
			&data.ExpiresAt,
			&rotaInternaID,
		)
		return data, err
	})
	if err != nil {
		return nil, 0, err
	}

	return &dados, rotaInternaID, nil
}

func listParadasCalculo(ctx context.Context, querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}, rotaInternaID int64) ([]PontoCalculoRota, error) {
	const q = `
		SELECT p.id, p.nome, p.latitude, p.longitude, rip.ordem
		FROM rota_interna_paradas rip
		JOIN paradas p ON p.id = rip.parada_id
		WHERE rip.rota_interna_id = @rota_interna_id
		ORDER BY rip.ordem ASC
	`

	rows, err := querier.Query(ctx, q, pgx.StrictNamedArgs{"rota_interna_id": rotaInternaID})
	if err != nil {
		return nil, err
	}

	paradas, err := pgx.CollectRows(rows, scanPontoCalculoRota)
	if err != nil {
		return nil, err
	}
	if paradas == nil {
		return []PontoCalculoRota{}, nil
	}

	return paradas, nil
}

func listDestinosCalculo(ctx context.Context, querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}, viagemID int64) ([]DestinoCalculoRota, error) {
	const q = `
		SELECT DISTINCT d.id, d.nome, d.latitude, d.longitude
		FROM viagem_reservas vr
		JOIN reservas r ON r.id = vr.reserva_id
		JOIN destinos d ON d.id = r.destino_id
		WHERE vr.viagem_id = @viagem_id
			AND vr.status_presenca <> 'cancelado'
			AND r.status = 'confirmada'
		ORDER BY d.id ASC
	`

	rows, err := querier.Query(ctx, q, pgx.StrictNamedArgs{"viagem_id": viagemID})
	if err != nil {
		return nil, err
	}

	destinos, err := pgx.CollectRows(rows, scanDestinoCalculoRota)
	if err != nil {
		return nil, err
	}
	if destinos == nil {
		return []DestinoCalculoRota{}, nil
	}

	return destinos, nil
}

func scanPontoCalculoRota(row pgx.CollectableRow) (PontoCalculoRota, error) {
	var ponto PontoCalculoRota
	err := row.Scan(
		&ponto.ID,
		&ponto.Nome,
		&ponto.Latitude,
		&ponto.Longitude,
		&ponto.Ordem,
	)
	return ponto, err
}

func scanDestinoCalculoRota(row pgx.CollectableRow) (DestinoCalculoRota, error) {
	var destino DestinoCalculoRota
	err := row.Scan(
		&destino.ID,
		&destino.Nome,
		&destino.Latitude,
		&destino.Longitude,
	)
	return destino, err
}
