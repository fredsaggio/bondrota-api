//go:build integration

package repositories

import (
	"testing"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/reservas"
	"github.com/fredsaggio/bondrota-api/internal/viagens"
	"github.com/stretchr/testify/require"
)

func TestReservaRepository_CRUDAndVinculoSnapshot(t *testing.T) {
	ctx, tx := beginTestTx(t)
	fixture := seedFixtureWithVinculo(t, ctx, tx)
	store := reservas.NewReservaStore(tx)
	tripDate := futureTripDate()

	created, err := store.Create(ctx, reservas.ReservaInput{
		ClienteID: fixture.ClienteID, VinculoID: fixture.VinculoID, DataViagem: tripDate,
		Turno: reservas.TurnoNoturno, DestinoID: fixture.DestinoID,
		RotaInternaID: fixture.RotaInternaID, Sentido: reservas.SentidoIda,
	})
	require.NoError(t, err)
	require.Equal(t, reservas.StatusConfirmada, created.Status)

	snapshot, err := store.GetVinculoSnapshot(ctx, fixture.VinculoID)
	require.NoError(t, err)
	require.Equal(t, fixture.ClienteID, snapshot.ClienteID)

	byCliente, err := store.ListByCliente(ctx, fixture.ClienteID)
	require.NoError(t, err)
	require.Len(t, byCliente, 1)
	byVinculo, err := store.ListByVinculo(ctx, fixture.ClienteID, fixture.VinculoID)
	require.NoError(t, err)
	require.Len(t, byVinculo, 1)

	updated, err := store.Update(ctx, created.ID, func(current *reservas.Reserva) (bool, error) {
		current.Status = reservas.StatusCancelada
		return true, nil
	})
	require.NoError(t, err)
	require.Equal(t, reservas.StatusCancelada, updated.Status)

	require.NoError(t, store.Delete(ctx, created.ID))
	_, err = store.GetByID(ctx, created.ID)
	require.ErrorIs(t, err, reservas.ErrReservaNotFound)
}

func TestReservaRepository_ListPagina(t *testing.T) {
	ctx, tx := beginTestTx(t)
	fixture := seedFixtureWithVinculo(t, ctx, tx)
	store := reservas.NewReservaStore(tx)

	base := futureTripDate()
	dates := []time.Time{base, base.AddDate(0, 0, 1), base.AddDate(0, 0, 2)}
	var ids []int64
	for _, data := range dates {
		created, err := store.Create(ctx, reservas.ReservaInput{
			ClienteID: fixture.ClienteID, VinculoID: fixture.VinculoID, DataViagem: data,
			Turno: reservas.TurnoNoturno, DestinoID: fixture.DestinoID,
			RotaInternaID: fixture.RotaInternaID, Sentido: reservas.SentidoIda,
		})
		require.NoError(t, err)
		ids = append(ids, created.ID)
	}
	// ORDER BY data_viagem DESC, id DESC: a mais recente primeiro.
	newest, middle, oldest := ids[2], ids[1], ids[0]

	first, err := store.List(ctx, reservas.ReservaListParams{Limit: 2})
	require.NoError(t, err)
	require.Len(t, first.Items, 2)
	require.Equal(t, newest, first.Items[0].ID)
	require.Equal(t, middle, first.Items[1].ID)
	require.True(t, first.HasMore)
	require.NotNil(t, first.NextCursor)

	second, err := store.List(ctx, reservas.ReservaListParams{Limit: 2, Cursor: first.NextCursor})
	require.NoError(t, err)
	require.Len(t, second.Items, 1)
	require.Equal(t, oldest, second.Items[0].ID)
	require.False(t, second.HasMore)
	require.Nil(t, second.NextCursor)
}

func TestReservaRepository_ListBuscaPorNomeDeClienteEDestino(t *testing.T) {
	ctx, tx := beginTestTx(t)
	store := reservas.NewReservaStore(tx)

	paradaID := seedParada(t, ctx, tx, "Terminal")
	rotaID := seedRotaInterna(t, ctx, tx, paradaID)

	destinoAna := seedDestino(t, ctx, tx, "Campus Central", testMunicipioID)
	clienteAna := seedClienteComNome(t, ctx, tx, "30000000001", "Ana Beatriz")
	vinculoAna := seedVinculo(t, ctx, tx, clienteAna, destinoAna, rotaID)

	destinoBruno := seedDestino(t, ctx, tx, "Terminal Sul", testMunicipioID)
	clienteBruno := seedClienteComNome(t, ctx, tx, "30000000002", "Bruno Costa")
	vinculoBruno := seedVinculo(t, ctx, tx, clienteBruno, destinoBruno, rotaID)

	data := futureTripDate()
	reservaAna, err := store.Create(ctx, reservas.ReservaInput{
		ClienteID: clienteAna, VinculoID: vinculoAna, DataViagem: data,
		Turno: reservas.TurnoNoturno, DestinoID: destinoAna, RotaInternaID: rotaID, Sentido: reservas.SentidoIda,
	})
	require.NoError(t, err)
	reservaBruno, err := store.Create(ctx, reservas.ReservaInput{
		ClienteID: clienteBruno, VinculoID: vinculoBruno, DataViagem: data,
		Turno: reservas.TurnoNoturno, DestinoID: destinoBruno, RotaInternaID: rotaID, Sentido: reservas.SentidoIda,
	})
	require.NoError(t, err)

	t.Run("por nome do cliente, case-insensitive e parcial", func(t *testing.T) {
		result, err := store.List(ctx, reservas.ReservaListParams{Busca: "ana beat"})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		require.Equal(t, reservaAna.ID, result.Items[0].ID)
		require.Equal(t, "Ana Beatriz", result.Items[0].ClienteNome)
	})

	t.Run("por nome do destino", func(t *testing.T) {
		result, err := store.List(ctx, reservas.ReservaListParams{Busca: "terminal sul"})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		require.Equal(t, reservaBruno.ID, result.Items[0].ID)
		require.Equal(t, "Terminal Sul", result.Items[0].DestinoNome)
	})

	t.Run("sem termo, lista tudo", func(t *testing.T) {
		result, err := store.List(ctx, reservas.ReservaListParams{})
		require.NoError(t, err)
		require.Len(t, result.Items, 2)
	})

	// A busca livre nao cobre data de proposito: quem filtra por data e o
	// intervalo dedicado (DataInicio/DataFim), nao o campo de texto.
	t.Run("nao bate na data, mesmo em formato exato", func(t *testing.T) {
		result, err := store.List(ctx, reservas.ReservaListParams{Busca: data.Format("2006-01-02")})
		require.NoError(t, err)
		require.Empty(t, result.Items)
	})
}

func TestReservaRepository_ListFiltraPorIntervaloDeData(t *testing.T) {
	ctx, tx := beginTestTx(t)
	fixture := seedFixtureWithVinculo(t, ctx, tx)
	store := reservas.NewReservaStore(tx)

	base := futureTripDate()
	dates := []time.Time{base, base.AddDate(0, 0, 1), base.AddDate(0, 0, 2)}
	var ids []int64
	for _, data := range dates {
		created, err := store.Create(ctx, reservas.ReservaInput{
			ClienteID: fixture.ClienteID, VinculoID: fixture.VinculoID, DataViagem: data,
			Turno: reservas.TurnoNoturno, DestinoID: fixture.DestinoID,
			RotaInternaID: fixture.RotaInternaID, Sentido: reservas.SentidoIda,
		})
		require.NoError(t, err)
		ids = append(ids, created.ID)
	}

	middle := dates[1]
	result, err := store.List(ctx, reservas.ReservaListParams{DataInicio: &middle, DataFim: &middle})
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	require.Equal(t, ids[1], result.Items[0].ID)
}

func TestReservaRepository_ResumoContaSomenteConfirmadas(t *testing.T) {
	ctx, tx := beginTestTx(t)
	fixture := seedFixtureWithVinculo(t, ctx, tx)
	store := reservas.NewReservaStore(tx)

	base := futureTripDate()
	criar := func(data time.Time, turno reservas.TurnoReserva) *reservas.Reserva {
		created, err := store.Create(ctx, reservas.ReservaInput{
			ClienteID: fixture.ClienteID, VinculoID: fixture.VinculoID, DataViagem: data,
			Turno: turno, DestinoID: fixture.DestinoID,
			RotaInternaID: fixture.RotaInternaID, Sentido: reservas.SentidoIda,
		})
		require.NoError(t, err)
		return created
	}

	criar(base, reservas.TurnoNoturno)
	criar(base.AddDate(0, 0, 1), reservas.TurnoNoturno)
	criar(base.AddDate(0, 0, 2), reservas.TurnoMatutino)
	cancelada := criar(base.AddDate(0, 0, 3), reservas.TurnoMatutino)

	_, err := store.Update(ctx, cancelada.ID, func(current *reservas.Reserva) (bool, error) {
		current.Status = reservas.StatusCancelada
		return true, nil
	})
	require.NoError(t, err)

	resumo, err := store.Resumo(ctx)
	require.NoError(t, err)
	// A cancelada fica de fora: o painel conta reservas ativas, nao historico.
	require.Equal(t, int64(3), resumo.ConfirmadasTotal)
	require.Equal(t, int64(2), resumo.ConfirmadasPorTurno[reservas.TurnoNoturno])
	require.Equal(t, int64(1), resumo.ConfirmadasPorTurno[reservas.TurnoMatutino])
}

func TestReservaRepository_GetHorarioPartidaPorSentido(t *testing.T) {
	ctx, tx := beginTestTx(t)
	fixture := seedFixtureWithVinculo(t, ctx, tx)
	store := reservas.NewReservaStore(tx)

	_, err := store.GetHorarioPartida(ctx, fixture.DestinoID, reservas.TurnoNoturno, reservas.SentidoIda)
	require.ErrorIs(t, err, reservas.ErrHorarioNaoConfigurado)

	_, err = tx.Exec(ctx, `
		INSERT INTO horarios_turno_viagem (
			municipio_destino_id, turno, horario_ida, horario_volta
		) VALUES ($1, 'NT', '17:00', '22:00')
	`, testMunicipioID)
	require.NoError(t, err)

	horarioIda, err := store.GetHorarioPartida(ctx, fixture.DestinoID, reservas.TurnoNoturno, reservas.SentidoIda)
	require.NoError(t, err)
	require.Equal(t, 17*time.Hour, horarioIda)

	horarioVolta, err := store.GetHorarioPartida(ctx, fixture.DestinoID, reservas.TurnoNoturno, reservas.SentidoVolta)
	require.NoError(t, err)
	require.Equal(t, 22*time.Hour, horarioVolta)
}

func TestReservaRepository_EnforcesOneActiveReservationPerDirection(t *testing.T) {
	ctx, tx := beginTestTx(t)
	fixture := seedFixtureWithVinculo(t, ctx, tx)
	store := reservas.NewReservaStore(tx)
	input := reservas.ReservaInput{
		ClienteID: fixture.ClienteID, VinculoID: fixture.VinculoID, DataViagem: futureTripDate(),
		Turno: reservas.TurnoNoturno, DestinoID: fixture.DestinoID,
		RotaInternaID: fixture.RotaInternaID, Sentido: reservas.SentidoIda,
	}

	_, err := store.Create(ctx, input)
	require.NoError(t, err)
	_, err = store.Create(ctx, input)
	require.Error(t, err)
}

func TestReservaRepository_RejectsConfirmedReservationAfterPlanningStarts(t *testing.T) {
	ctx, tx := beginTestTx(t)
	fixture := seedFixtureWithVinculo(t, ctx, tx)
	reservaStore := reservas.NewReservaStore(tx)
	execucaoStore := viagens.NewExecucaoPlanejamentoStore(tx)
	dataViagem := futureTripDate()
	agora := time.Date(dataViagem.Year(), dataViagem.Month(), dataViagem.Day(), 16, 30, 0, 0, time.UTC)

	_, adquirida, err := execucaoStore.TentarIniciar(ctx, viagens.IniciarExecucaoPlanejamentoInput{
		Chave: viagens.ChaveExecucaoPlanejamento{
			DataViagem:         dataViagem,
			Turno:              viagens.TurnoNoturno,
			MunicipioDestinoID: testMunicipioID,
			RotaInternaID:      fixture.RotaInternaID,
			Sentido:            viagens.SentidoIda,
		},
		PartidaEm:        agora.Add(30 * time.Minute),
		FechamentoEm:     agora,
		Agora:            agora,
		BloqueioExpiraEm: agora.Add(2 * time.Minute),
	})
	require.NoError(t, err)
	require.True(t, adquirida)

	_, err = reservaStore.Create(ctx, reservas.ReservaInput{
		ClienteID:     fixture.ClienteID,
		VinculoID:     fixture.VinculoID,
		DataViagem:    dataViagem,
		Turno:         reservas.TurnoNoturno,
		DestinoID:     fixture.DestinoID,
		RotaInternaID: fixture.RotaInternaID,
		Sentido:       reservas.SentidoIda,
	})
	require.ErrorIs(t, err, reservas.ErrPrazoReservaEncerrado)
}

func TestReservaRepository_RejectsReactivationAfterPlanningStarts(t *testing.T) {
	ctx, tx := beginTestTx(t)
	fixture := seedFixtureWithVinculo(t, ctx, tx)
	reservaStore := reservas.NewReservaStore(tx)
	execucaoStore := viagens.NewExecucaoPlanejamentoStore(tx)
	dataViagem := futureTripDate()

	created, err := reservaStore.Create(ctx, reservas.ReservaInput{
		ClienteID:     fixture.ClienteID,
		VinculoID:     fixture.VinculoID,
		DataViagem:    dataViagem,
		Turno:         reservas.TurnoNoturno,
		DestinoID:     fixture.DestinoID,
		RotaInternaID: fixture.RotaInternaID,
		Sentido:       reservas.SentidoVolta,
	})
	require.NoError(t, err)

	_, err = reservaStore.Update(ctx, created.ID, func(current *reservas.Reserva) (bool, error) {
		current.Status = reservas.StatusCancelada
		return true, nil
	})
	require.NoError(t, err)

	agora := time.Date(dataViagem.Year(), dataViagem.Month(), dataViagem.Day(), 21, 30, 0, 0, time.UTC)
	_, adquirida, err := execucaoStore.TentarIniciar(ctx, viagens.IniciarExecucaoPlanejamentoInput{
		Chave: viagens.ChaveExecucaoPlanejamento{
			DataViagem:         dataViagem,
			Turno:              viagens.TurnoNoturno,
			MunicipioDestinoID: testMunicipioID,
			RotaInternaID:      fixture.RotaInternaID,
			Sentido:            viagens.SentidoVolta,
		},
		PartidaEm:        agora.Add(30 * time.Minute),
		FechamentoEm:     agora,
		Agora:            agora,
		BloqueioExpiraEm: agora.Add(2 * time.Minute),
	})
	require.NoError(t, err)
	require.True(t, adquirida)

	_, err = reservaStore.Update(ctx, created.ID, func(current *reservas.Reserva) (bool, error) {
		current.Status = reservas.StatusConfirmada
		return true, nil
	})
	require.ErrorIs(t, err, reservas.ErrPrazoReservaEncerrado)

	stored, err := reservaStore.GetByID(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, reservas.StatusCancelada, stored.Status)
}
