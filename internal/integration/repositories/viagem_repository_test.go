//go:build integration

package repositories

import (
	"testing"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/viagens"
	"github.com/stretchr/testify/require"
)

func TestViagemRepository_CreatesListsAndTransitionsTrip(t *testing.T) {
	ctx, tx := beginTestTx(t)
	fixture := seedBaseFixture(t, ctx, tx)
	tripDate := futureTripDate()
	cicloID := seedCiclo(t, ctx, tx, fixture, tripDate)
	store := viagens.NewViagemStore(tx)
	partida := tripDate.Add(17 * time.Hour)

	created, err := store.CreateViagem(ctx, viagens.ViagemInput{
		CicloViagemID: cicloID, Sentido: viagens.SentidoIda, PartidaPrevista: partida,
	})
	require.NoError(t, err)
	require.Equal(t, viagens.StatusViagemProgramada, created.Status)

	got, err := store.GetViagemByID(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, cicloID, got.Ciclo.ID)

	byCycle, err := store.ListViagensByCiclo(ctx, cicloID)
	require.NoError(t, err)
	require.Len(t, byCycle, 1)
	hours, err := store.ListHorariosByViagem(ctx, created.ID)
	require.NoError(t, err)
	require.Len(t, hours, 1)
	require.Equal(t, viagens.TipoHorarioPartidaPrevista, hours[0].Tipo)

	startedAt := partida.Add(10 * time.Minute)
	started, err := store.AtualizarStatusERegistrarHorarioViagem(
		ctx, created.ID, viagens.StatusViagemProgramada, viagens.StatusViagemEmAndamento,
		viagens.TipoHorarioInicioReal, startedAt,
	)
	require.NoError(t, err)
	require.Equal(t, viagens.StatusViagemEmAndamento, started.Status)

	hours, err = store.ListHorariosByViagem(ctx, created.ID)
	require.NoError(t, err)
	require.Len(t, hours, 2)
}

// municipio_destino_id era exposto no JSON mas nunca era selecionado: voltava
// sempre 0, e o painel exibia "Municipio #0".
func TestViagemRepository_GetViagemByIDTrazMunicipioDestino(t *testing.T) {
	ctx, tx := beginTestTx(t)
	fixture := seedBaseFixture(t, ctx, tx)
	cicloID := seedCiclo(t, ctx, tx, fixture, futureTripDate())
	store := viagens.NewViagemStore(tx)
	viagemID := seedViagem(t, ctx, tx, cicloID, "ida")

	got, err := store.GetViagemByID(ctx, viagemID)
	require.NoError(t, err)
	require.Equal(t, testMunicipioID, got.Ciclo.MunicipioDestinoID)
}

func TestViagemRepository_ListViagensPagina(t *testing.T) {
	ctx, tx := beginTestTx(t)
	fixture := seedBaseFixture(t, ctx, tx)
	store := viagens.NewViagemStore(tx)

	base := futureTripDate()
	var ids []int64
	for offset := range 3 {
		cicloID := seedCiclo(t, ctx, tx, fixture, base.AddDate(0, 0, offset))
		ids = append(ids, seedViagem(t, ctx, tx, cicloID, "ida"))
	}
	newest, middle, oldest := ids[2], ids[1], ids[0]

	first, err := store.ListViagens(ctx, viagens.ViagemListParams{Limit: 2})
	require.NoError(t, err)
	require.Len(t, first.Items, 2)
	require.Equal(t, newest, first.Items[0].Viagem.ID)
	require.Equal(t, middle, first.Items[1].Viagem.ID)
	require.True(t, first.HasMore)

	// Os nomes vem resolvidos pelo JOIN, nao como id cru.
	require.Equal(t, testCity, first.Items[0].MunicipioNome)
	require.NotEmpty(t, first.Items[0].VeiculoPlaca)

	second, err := store.ListViagens(ctx, viagens.ViagemListParams{Limit: 2, Cursor: first.NextCursor})
	require.NoError(t, err)
	require.Len(t, second.Items, 1)
	require.Equal(t, oldest, second.Items[0].Viagem.ID)
	require.False(t, second.HasMore)
}

func TestViagemRepository_ListViagensFiltros(t *testing.T) {
	ctx, tx := beginTestTx(t)
	fixture := seedBaseFixture(t, ctx, tx)
	store := viagens.NewViagemStore(tx)

	outroMotorista := seedMotorista(t, ctx, tx, "10000000002", testMunicipioID, "NT")
	base := futureTripDate()

	cicloA := seedCiclo(t, ctx, tx, fixture, base)
	viagemA := seedViagem(t, ctx, tx, cicloA, "ida")

	outraFixture := fixture
	outraFixture.MotoristaID = outroMotorista
	cicloB := seedCiclo(t, ctx, tx, outraFixture, base.AddDate(0, 0, 1))
	viagemB := seedViagem(t, ctx, tx, cicloB, "volta")

	t.Run("por motorista", func(t *testing.T) {
		result, err := store.ListViagens(ctx, viagens.ViagemListParams{MotoristaID: outroMotorista})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		require.Equal(t, viagemB, result.Items[0].Viagem.ID)
	})

	t.Run("por nome do municipio", func(t *testing.T) {
		result, err := store.ListViagens(ctx, viagens.ViagemListParams{Busca: "macei"})
		require.NoError(t, err)
		require.Len(t, result.Items, 2)
	})

	t.Run("por sentido", func(t *testing.T) {
		result, err := store.ListViagens(ctx, viagens.ViagemListParams{Busca: "volta"})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		require.Equal(t, viagemB, result.Items[0].Viagem.ID)
	})

	t.Run("por intervalo de data", func(t *testing.T) {
		result, err := store.ListViagens(ctx, viagens.ViagemListParams{DataInicio: &base, DataFim: &base})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		require.Equal(t, viagemA, result.Items[0].Viagem.ID)
	})

	// A data fica fora da busca livre: quem recorta por data e o intervalo.
	t.Run("busca nao bate na data", func(t *testing.T) {
		result, err := store.ListViagens(ctx, viagens.ViagemListParams{Busca: base.Format("2006-01-02")})
		require.NoError(t, err)
		require.Empty(t, result.Items)
	})
}

// O monitoramento pede as viagens acompanhaveis em ordem crescente. Com a ordem
// padrao (mais recente primeiro), as de hoje ficariam depois de todas as futuras.
func TestViagemRepository_ListViagensStatusEOrdemAscendente(t *testing.T) {
	ctx, tx := beginTestTx(t)
	fixture := seedBaseFixture(t, ctx, tx)
	store := viagens.NewViagemStore(tx)

	base := futureTripDate()
	cicloProxima := seedCiclo(t, ctx, tx, fixture, base)
	proxima := seedViagem(t, ctx, tx, cicloProxima, "ida")

	cicloDistante := seedCiclo(t, ctx, tx, fixture, base.AddDate(0, 0, 30))
	distante := seedViagem(t, ctx, tx, cicloDistante, "ida")

	cicloCancelada := seedCiclo(t, ctx, tx, fixture, base.AddDate(0, 0, 1))
	cancelada := seedViagem(t, ctx, tx, cicloCancelada, "ida")
	_, err := tx.Exec(ctx, `UPDATE viagens SET status = 'cancelada' WHERE id = $1`, cancelada)
	require.NoError(t, err)

	result, err := store.ListViagens(ctx, viagens.ViagemListParams{
		Status:     []viagens.StatusViagem{viagens.StatusViagemProgramada, viagens.StatusViagemEmAndamento},
		Ascendente: true,
	})
	require.NoError(t, err)
	require.Len(t, result.Items, 2)
	require.Equal(t, proxima, result.Items[0].Viagem.ID, "a mais proxima vem primeiro")
	require.Equal(t, distante, result.Items[1].Viagem.ID)
}

func TestViagemRepository_ListViagensCursorAscendente(t *testing.T) {
	ctx, tx := beginTestTx(t)
	fixture := seedBaseFixture(t, ctx, tx)
	store := viagens.NewViagemStore(tx)

	base := futureTripDate()
	var ids []int64
	for offset := range 3 {
		cicloID := seedCiclo(t, ctx, tx, fixture, base.AddDate(0, 0, offset))
		ids = append(ids, seedViagem(t, ctx, tx, cicloID, "ida"))
	}

	first, err := store.ListViagens(ctx, viagens.ViagemListParams{Limit: 2, Ascendente: true})
	require.NoError(t, err)
	require.Len(t, first.Items, 2)
	require.Equal(t, ids[0], first.Items[0].Viagem.ID)
	require.Equal(t, ids[1], first.Items[1].Viagem.ID)
	require.True(t, first.HasMore)

	// O cursor precisa comparar no sentido certo: com ">" invertido, a segunda
	// pagina voltaria vazia ou repetiria a primeira.
	second, err := store.ListViagens(ctx, viagens.ViagemListParams{Limit: 2, Ascendente: true, Cursor: first.NextCursor})
	require.NoError(t, err)
	require.Len(t, second.Items, 1)
	require.Equal(t, ids[2], second.Items[0].Viagem.ID)
}

func TestViagemRepository_ResumoViagens(t *testing.T) {
	ctx, tx := beginTestTx(t)
	fixture := seedBaseFixture(t, ctx, tx)
	store := viagens.NewViagemStore(tx)

	hoje := futureTripDate()
	cicloHoje := seedCiclo(t, ctx, tx, fixture, hoje)
	viagemHoje := seedViagem(t, ctx, tx, cicloHoje, "ida")
	seedViagem(t, ctx, tx, cicloHoje, "volta")

	cicloOutroDia := seedCiclo(t, ctx, tx, fixture, hoje.AddDate(0, 0, 5))
	seedViagem(t, ctx, tx, cicloOutroDia, "ida")

	_, err := tx.Exec(ctx, `UPDATE viagens SET status = 'em_andamento' WHERE id = $1`, viagemHoje)
	require.NoError(t, err)

	resumo, err := store.ResumoViagens(ctx, hoje)
	require.NoError(t, err)

	require.Equal(t, int64(2), resumo.PorStatus[viagens.StatusViagemProgramada])
	require.Equal(t, int64(1), resumo.PorStatus[viagens.StatusViagemEmAndamento])
	require.Equal(t, int64(3), resumo.PorTurno[viagens.TurnoNoturno])
	require.Equal(t, int64(2), resumo.HojeTotal)
	require.Equal(t, int64(1), resumo.HojeEmAndamento)

	// Proximas vem em ordem crescente — o oposto da listagem.
	require.Len(t, resumo.Proximas, 3)
	require.True(t, !resumo.Proximas[0].Ciclo.DataViagem.After(resumo.Proximas[2].Ciclo.DataViagem))
	require.Equal(t, testCity, resumo.Proximas[0].MunicipioNome)
}
