//go:build integration

package repositories

import (
	"context"
	"testing"

	"github.com/fredsaggio/bondrota-api/internal/clientes"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

// seedHorariosFixos adiciona dias da semana ao vinculo. Eles multiplicam as linhas
// da consulta de listagem, que e exatamente o que a paginacao precisa tratar.
func seedHorariosFixos(t *testing.T, ctx context.Context, tx pgx.Tx, vinculoID int64, dias ...int) {
	t.Helper()
	for _, dia := range dias {
		_, err := tx.Exec(ctx,
			`INSERT INTO horarios_fixos (vinculo_id, dia_semana) VALUES ($1, $2)`,
			vinculoID, dia,
		)
		require.NoError(t, err)
	}
}

func TestVinculoRepository_ListPagina(t *testing.T) {
	ctx, tx := beginTestTx(t)
	fixture := seedBaseFixture(t, ctx, tx)
	store := clientes.NewVinculoStore(tx)

	// Nomes fora de ordem alfabetica de proposito: a listagem ordena por nome.
	carla := seedClienteComNome(t, ctx, tx, "40000000003", "Carla Dias")
	ana := seedClienteComNome(t, ctx, tx, "40000000001", "Ana Beatriz")
	bruno := seedClienteComNome(t, ctx, tx, "40000000002", "Bruno Costa")

	vinculoCarla := seedVinculo(t, ctx, tx, carla, fixture.DestinoID, fixture.RotaInternaID)
	vinculoAna := seedVinculo(t, ctx, tx, ana, fixture.DestinoID, fixture.RotaInternaID)
	vinculoBruno := seedVinculo(t, ctx, tx, bruno, fixture.DestinoID, fixture.RotaInternaID)

	first, err := store.List(ctx, clientes.VinculoListParams{Limit: 2})
	require.NoError(t, err)
	require.Len(t, first.Items, 2)
	require.Equal(t, vinculoAna, first.Items[0].ID, "ordena por nome do cliente")
	require.Equal(t, vinculoBruno, first.Items[1].ID)
	require.True(t, first.HasMore)
	require.NotNil(t, first.NextCursor)

	second, err := store.List(ctx, clientes.VinculoListParams{Limit: 2, Cursor: first.NextCursor})
	require.NoError(t, err)
	require.Len(t, second.Items, 1)
	require.Equal(t, vinculoCarla, second.Items[0].ID)
	require.False(t, second.HasMore)
}

/**
 * O LEFT JOIN de horarios_fixos multiplica as linhas: um vinculo com 5 dias vira
 * 5 linhas. Sem a CTE que pagina os vinculos antes, um LIMIT baixo cortaria no
 * meio e o vinculo viria com parte dos dias.
 */
func TestVinculoRepository_ListNaoCortaHorariosFixos(t *testing.T) {
	ctx, tx := beginTestTx(t)
	fixture := seedBaseFixture(t, ctx, tx)
	store := clientes.NewVinculoStore(tx)

	ana := seedClienteComNome(t, ctx, tx, "40000000001", "Ana Beatriz")
	bruno := seedClienteComNome(t, ctx, tx, "40000000002", "Bruno Costa")

	vinculoAna := seedVinculo(t, ctx, tx, ana, fixture.DestinoID, fixture.RotaInternaID)
	vinculoBruno := seedVinculo(t, ctx, tx, bruno, fixture.DestinoID, fixture.RotaInternaID)

	seedHorariosFixos(t, ctx, tx, vinculoAna, 1, 2, 3, 4, 5)
	seedHorariosFixos(t, ctx, tx, vinculoBruno, 1, 2)

	// limit=1 e menor que os 5 dias da Ana: se o recorte fosse por linha, ela
	// voltaria com um dia so — e o Bruno nem apareceria na proxima pagina.
	first, err := store.List(ctx, clientes.VinculoListParams{Limit: 1})
	require.NoError(t, err)
	require.Len(t, first.Items, 1)
	require.Equal(t, vinculoAna, first.Items[0].ID)
	require.Len(t, first.Items[0].HorariosFixos, 5, "o vinculo precisa vir com todos os dias")
	require.True(t, first.HasMore)

	second, err := store.List(ctx, clientes.VinculoListParams{Limit: 1, Cursor: first.NextCursor})
	require.NoError(t, err)
	require.Len(t, second.Items, 1)
	require.Equal(t, vinculoBruno, second.Items[0].ID)
	require.Len(t, second.Items[0].HorariosFixos, 2)
}

func TestVinculoRepository_ListBusca(t *testing.T) {
	ctx, tx := beginTestTx(t)
	fixture := seedBaseFixture(t, ctx, tx)
	store := clientes.NewVinculoStore(tx)

	outroDestino := seedDestino(t, ctx, tx, "Campus Norte", testMunicipioID)

	ana := seedClienteComNome(t, ctx, tx, "40000000001", "Ana Beatriz")
	bruno := seedClienteComNome(t, ctx, tx, "40000000002", "Bruno Costa")

	vinculoAna := seedVinculo(t, ctx, tx, ana, fixture.DestinoID, fixture.RotaInternaID)
	vinculoBruno := seedVinculo(t, ctx, tx, bruno, outroDestino, fixture.RotaInternaID)

	t.Run("por nome do cliente", func(t *testing.T) {
		result, err := store.List(ctx, clientes.VinculoListParams{Busca: "ana beat"})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		require.Equal(t, vinculoAna, result.Items[0].ID)
	})

	t.Run("por nome do destino", func(t *testing.T) {
		result, err := store.List(ctx, clientes.VinculoListParams{Busca: "campus norte"})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		require.Equal(t, vinculoBruno, result.Items[0].ID)
		require.Equal(t, "Campus Norte", result.Items[0].DestinoNome)
	})

	t.Run("por curso", func(t *testing.T) {
		result, err := store.List(ctx, clientes.VinculoListParams{Busca: "computacao"})
		require.NoError(t, err)
		require.Len(t, result.Items, 2)
	})

	t.Run("sem termo, lista tudo", func(t *testing.T) {
		result, err := store.List(ctx, clientes.VinculoListParams{})
		require.NoError(t, err)
		require.Len(t, result.Items, 2)
	})
}
