package viagens

import (
	"context"
	"errors"
	"fmt"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/jackc/pgx/v5"
)

type viagemLocalizacaoStore struct {
	db db.DB
}

func NewViagemLocalizacaoStore(db db.DB) ViagemLocalizacaoStore {
	return &viagemLocalizacaoStore{db: db}
}

func (s *viagemLocalizacaoStore) Upsert(ctx context.Context, input ViagemLocalizacaoInput) (*ViagemLocalizacao, error) {
	const op = "db/viagemLocalizacaoStore.Upsert"

	const q = `
		INSERT INTO viagem_localizacoes (
			viagem_id, motorista_id, latitude, longitude, velocidade_kmh,
			direcao_graus, precisao_metros, registrada_em
		)
		VALUES (
			@viagem_id, @motorista_id, @latitude, @longitude, @velocidade_kmh,
			@direcao_graus, @precisao_metros, @registrada_em
		)
		ON CONFLICT (viagem_id) DO UPDATE
		SET motorista_id = EXCLUDED.motorista_id,
			latitude = EXCLUDED.latitude,
			longitude = EXCLUDED.longitude,
			velocidade_kmh = EXCLUDED.velocidade_kmh,
			direcao_graus = EXCLUDED.direcao_graus,
			precisao_metros = EXCLUDED.precisao_metros,
			registrada_em = EXCLUDED.registrada_em
		RETURNING
			viagem_id, motorista_id, latitude, longitude, velocidade_kmh,
			direcao_graus, precisao_metros, registrada_em, created_at, updated_at
	`

	rows, err := s.db.Query(ctx, q, pgx.StrictNamedArgs{
		"viagem_id":       input.ViagemID,
		"motorista_id":    input.MotoristaID,
		"latitude":        input.Latitude,
		"longitude":       input.Longitude,
		"velocidade_kmh":  input.VelocidadeKmh,
		"direcao_graus":   input.DirecaoGraus,
		"precisao_metros": input.PrecisaoMetros,
		"registrada_em":   input.RegistradaEm,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	localizacao, err := pgx.CollectExactlyOneRow(rows, scanViagemLocalizacao)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if err := hydrateLocalizacaoPublicIDs(ctx, s.db, &localizacao); err != nil {
		return nil, fmt.Errorf("%s: hydrate public ids: %w", op, err)
	}

	return &localizacao, nil
}

func (s *viagemLocalizacaoStore) GetByViagem(ctx context.Context, viagemID int64) (*ViagemLocalizacao, error) {
	const op = "db/viagemLocalizacaoStore.GetByViagem"

	const q = `
		SELECT
			l.viagem_id, v.public_id, l.motorista_id, m.public_id,
			l.latitude, l.longitude, l.velocidade_kmh, l.direcao_graus,
			l.precisao_metros, l.registrada_em, l.created_at, l.updated_at
		FROM viagem_localizacoes l
		JOIN viagens v ON v.id = l.viagem_id
		JOIN motoristas m ON m.id = l.motorista_id
		WHERE l.viagem_id = @viagem_id
	`

	rows, err := s.db.Query(ctx, q, pgx.StrictNamedArgs{"viagem_id": viagemID})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	localizacao, err := pgx.CollectExactlyOneRow(rows, scanViagemLocalizacaoPublic)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, brerror.ErrNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &localizacao, nil
}

func (s *viagemLocalizacaoStore) CanMotoristaAccessViagem(ctx context.Context, viagemID, motoristaID int64, requireEmAndamento bool) (bool, error) {
	const op = "db/viagemLocalizacaoStore.CanMotoristaAccessViagem"

	q := `
		SELECT EXISTS (
			SELECT 1
			FROM viagens v
			JOIN ciclos_viagem c ON c.id = v.ciclo_viagem_id
			WHERE v.id = @viagem_id
				AND c.motorista_id = @motorista_id
	`
	if requireEmAndamento {
		q += " AND v.status = 'em_andamento'"
	}
	q += ")"

	var allowed bool
	if err := s.db.QueryRow(ctx, q, pgx.StrictNamedArgs{
		"viagem_id":    viagemID,
		"motorista_id": motoristaID,
	}).Scan(&allowed); err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return allowed, nil
}

func (s *viagemLocalizacaoStore) CanClienteAccessViagem(ctx context.Context, viagemID, clienteID int64) (bool, error) {
	const op = "db/viagemLocalizacaoStore.CanClienteAccessViagem"

	const q = `
		SELECT EXISTS (
			SELECT 1
			FROM viagem_reservas vr
			JOIN reservas r ON r.id = vr.reserva_id
			WHERE vr.viagem_id = @viagem_id
				AND r.cliente_id = @cliente_id
				AND vr.status_presenca <> 'cancelado'
		)
	`

	var allowed bool
	if err := s.db.QueryRow(ctx, q, pgx.StrictNamedArgs{
		"viagem_id":  viagemID,
		"cliente_id": clienteID,
	}).Scan(&allowed); err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return allowed, nil
}

func scanViagemLocalizacao(row pgx.CollectableRow) (ViagemLocalizacao, error) {
	var localizacao ViagemLocalizacao
	err := row.Scan(
		&localizacao.ViagemID,
		&localizacao.MotoristaID,
		&localizacao.Latitude,
		&localizacao.Longitude,
		&localizacao.VelocidadeKmh,
		&localizacao.DirecaoGraus,
		&localizacao.PrecisaoMetros,
		&localizacao.RegistradaEm,
		&localizacao.CreatedAt,
		&localizacao.UpdatedAt,
	)
	return localizacao, err
}

func scanViagemLocalizacaoPublic(row pgx.CollectableRow) (ViagemLocalizacao, error) {
	var localizacao ViagemLocalizacao
	err := row.Scan(
		&localizacao.ViagemID,
		&localizacao.ViagemPublicID,
		&localizacao.MotoristaID,
		&localizacao.MotoristaPublicID,
		&localizacao.Latitude,
		&localizacao.Longitude,
		&localizacao.VelocidadeKmh,
		&localizacao.DirecaoGraus,
		&localizacao.PrecisaoMetros,
		&localizacao.RegistradaEm,
		&localizacao.CreatedAt,
		&localizacao.UpdatedAt,
	)
	return localizacao, err
}

func hydrateLocalizacaoPublicIDs(ctx context.Context, querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}, localizacao *ViagemLocalizacao) error {
	return querier.QueryRow(ctx, `
		SELECT v.public_id, m.public_id
		FROM viagens v, motoristas m
		WHERE v.id = @viagem_id AND m.id = @motorista_id
	`, pgx.StrictNamedArgs{
		"viagem_id":    localizacao.ViagemID,
		"motorista_id": localizacao.MotoristaID,
	}).Scan(&localizacao.ViagemPublicID, &localizacao.MotoristaPublicID)
}
