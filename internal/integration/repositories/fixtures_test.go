//go:build integration

package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

const testCity = "Maceio"
const testMunicipioID int64 = 2704302

type baseFixture struct {
	DestinoID     int64
	ParadaID      int64
	RotaInternaID int64
	VeiculoID     int64
	MotoristaID   int64
	ClienteID     int64
	VinculoID     int64
}

func beginTestTx(t *testing.T) (context.Context, pgx.Tx) {
	t.Helper()
	ctx := t.Context()
	tx, err := testPool.Begin(ctx)
	require.NoError(t, err)
	_, err = tx.Exec(ctx, `
		INSERT INTO municipios (codigo_ibge, nome, uf)
		VALUES ($1, $2, 'AL')
		ON CONFLICT (codigo_ibge) DO NOTHING`, testMunicipioID, testCity)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, tx.Rollback(context.Background()))
	})
	return ctx, tx
}

func seedBaseFixture(t *testing.T, ctx context.Context, tx pgx.Tx) baseFixture {
	t.Helper()
	destinoID := seedDestino(t, ctx, tx, "UFAL", testMunicipioID)
	paradaID := seedParada(t, ctx, tx, "Terminal")
	rotaID := seedRotaInterna(t, ctx, tx, paradaID)
	return baseFixture{
		DestinoID:     destinoID,
		ParadaID:      paradaID,
		RotaInternaID: rotaID,
		VeiculoID:     seedVeiculo(t, ctx, tx, "INT0001", "ativo"),
		MotoristaID:   seedMotorista(t, ctx, tx, "10000000001", testMunicipioID, "NT"),
		ClienteID:     seedCliente(t, ctx, tx, "20000000001"),
	}
}

func seedFixtureWithVinculo(t *testing.T, ctx context.Context, tx pgx.Tx) baseFixture {
	t.Helper()
	fixture := seedBaseFixture(t, ctx, tx)
	fixture.VinculoID = seedVinculo(t, ctx, tx, fixture.ClienteID, fixture.DestinoID, fixture.RotaInternaID)
	return fixture
}

func seedDestino(t *testing.T, ctx context.Context, tx pgx.Tx, nome string, municipioID int64) int64 {
	t.Helper()
	var id int64
	err := tx.QueryRow(ctx, `
		INSERT INTO destinos (nome, rua, municipio_id, latitude, longitude)
		VALUES ($1, 'Av. Principal', $2, -9.6658, -35.7353)
		RETURNING id`, nome, municipioID).Scan(&id)
	require.NoError(t, err)
	return id
}

func seedParada(t *testing.T, ctx context.Context, tx pgx.Tx, nome string) int64 {
	t.Helper()
	var id int64
	err := tx.QueryRow(ctx, `
		INSERT INTO paradas (nome, latitude, longitude)
		VALUES ($1, -9.6500, -35.7200)
		RETURNING id`, nome).Scan(&id)
	require.NoError(t, err)
	return id
}

func seedRotaInterna(t *testing.T, ctx context.Context, tx pgx.Tx, paradaIDs ...int64) int64 {
	t.Helper()
	var id int64
	require.NoError(t, tx.QueryRow(ctx, `INSERT INTO rotas_internas DEFAULT VALUES RETURNING id`).Scan(&id))
	for index, paradaID := range paradaIDs {
		_, err := tx.Exec(ctx,
			`INSERT INTO rota_interna_paradas (rota_interna_id, parada_id, ordem) VALUES ($1, $2, $3)`,
			id, paradaID, index+1,
		)
		require.NoError(t, err)
	}
	return id
}

func seedVeiculo(t *testing.T, ctx context.Context, tx pgx.Tx, placa, status string) int64 {
	t.Helper()
	var id int64
	err := tx.QueryRow(ctx, `
		INSERT INTO veiculos (placa, modelo, categoria, capacidade, status)
		VALUES ($1, 'Van', 'carro_7_lugares', 7, $2)
		RETURNING id`, placa, status).Scan(&id)
	require.NoError(t, err)
	return id
}

func seedMotorista(t *testing.T, ctx context.Context, tx pgx.Tx, cpf string, municipioTrabalhoID int64, turno string) int64 {
	t.Helper()
	var id int64
	err := tx.QueryRow(ctx, `
		INSERT INTO motoristas (
			nome, cpf, senha, telefone, data_nasc, turno, municipio_trabalho_id, residencia, foto
		) VALUES ('Motorista Teste', $1, 'hash', '82999990000', '1985-05-20', $2, $3, $4, '')
		RETURNING id`, cpf, turno, municipioTrabalhoID, testCity).Scan(&id)
	require.NoError(t, err)
	return id
}

func seedCliente(t *testing.T, ctx context.Context, tx pgx.Tx, cpf string) int64 {
	t.Helper()
	var id int64
	err := tx.QueryRow(ctx, `
		INSERT INTO clientes (nome, cpf, senha, telefone, data_nasc, foto)
		VALUES ('Cliente Teste', $1, 'hash', '82999991111', '2002-08-10', '')
		RETURNING id`, cpf).Scan(&id)
	require.NoError(t, err)
	return id
}

func seedVinculo(t *testing.T, ctx context.Context, tx pgx.Tx, clienteID, destinoID, rotaID int64) int64 {
	t.Helper()
	var id int64
	err := tx.QueryRow(ctx, `
		INSERT INTO cliente_vinculos (
			cliente_id, tipo, turno, destino_id, rota_interna_id, curso, comprovante, validade
		) VALUES ($1, 'estudante', 'NT', $2, $3, 'Computacao', 'comprovante.pdf', '2030-12-31')
		RETURNING id`, clienteID, destinoID, rotaID).Scan(&id)
	require.NoError(t, err)
	return id
}

func seedReserva(t *testing.T, ctx context.Context, tx pgx.Tx, fixture baseFixture, sentido string, data time.Time) int64 {
	t.Helper()
	var id int64
	err := tx.QueryRow(ctx, `
		INSERT INTO reservas (
			cliente_id, vinculo_id, data_viagem, turno, destino_id, rota_interna_id, sentido
		) VALUES ($1, $2, $3, 'NT', $4, $5, $6)
		RETURNING id`, fixture.ClienteID, fixture.VinculoID, data, fixture.DestinoID,
		fixture.RotaInternaID, sentido).Scan(&id)
	require.NoError(t, err)
	return id
}

func seedCiclo(t *testing.T, ctx context.Context, tx pgx.Tx, fixture baseFixture, data time.Time) int64 {
	t.Helper()
	var id int64
	expiresAt := time.Date(data.Year(), data.Month(), data.Day(), 23, 59, 0, 0, time.UTC)
	err := tx.QueryRow(ctx, `
		INSERT INTO ciclos_viagem (
			data_viagem, turno, municipio_destino_id, rota_interna_id, veiculo_id, motorista_id, expires_at
		) VALUES ($1, 'NT', $2, $3, $4, $5, $6)
		RETURNING id`, data, testMunicipioID, fixture.RotaInternaID, fixture.VeiculoID,
		fixture.MotoristaID, expiresAt).Scan(&id)
	require.NoError(t, err)
	return id
}

func seedViagem(t *testing.T, ctx context.Context, tx pgx.Tx, cicloID int64, sentido string) int64 {
	t.Helper()
	var id int64
	require.NoError(t, tx.QueryRow(ctx, `
		INSERT INTO viagens (ciclo_viagem_id, sentido)
		VALUES ($1, $2)
		RETURNING id`, cicloID, sentido).Scan(&id))
	return id
}

func seedViagemReserva(t *testing.T, ctx context.Context, tx pgx.Tx, viagemID, reservaID int64) int64 {
	t.Helper()
	var id int64
	require.NoError(t, tx.QueryRow(ctx, `
		INSERT INTO viagem_reservas (viagem_id, reserva_id)
		VALUES ($1, $2)
		RETURNING id`, viagemID, reservaID).Scan(&id))
	return id
}

func futureTripDate() time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day()+7, 0, 0, 0, 0, time.UTC)
}
