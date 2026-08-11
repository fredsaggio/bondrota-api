//go:build integration

package repositories

import (
	"testing"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/clientes"
	"github.com/stretchr/testify/require"
)

func TestVinculoRepository_ListAcrossClientes(t *testing.T) {
	ctx, tx := beginTestTx(t)
	fixture := seedBaseFixture(t, ctx, tx)
	store := clientes.NewVinculoStore(tx)

	empty, err := store.List(ctx)
	require.NoError(t, err)
	require.Empty(t, empty)

	// Bruno e inserido antes de Ana para provar que a ordenacao vem do SQL.
	brunoID := seedClienteComNome(t, ctx, tx, "20000000002", "Bruno Lima")
	anaID := seedClienteComNome(t, ctx, tx, "20000000003", "Ana Costa")

	_, err = store.Create(ctx, clientes.VinculoInput{
		ClienteID: brunoID, Tipo: clientes.TipoEstudante, Turno: clientes.TurnoNoturno,
		DestinoID: fixture.DestinoID, RotaInternaID: fixture.RotaInternaID,
		Curso: "Direito", Comprovante: "bruno.pdf",
		Validade:      time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC),
		HorariosFixos: []clientes.DiaSemana{clientes.Segunda, clientes.Quarta, clientes.Sexta},
	})
	require.NoError(t, err)

	_, err = store.Create(ctx, clientes.VinculoInput{
		ClienteID: anaID, Tipo: clientes.TipoEstagio, Turno: clientes.TurnoMatutino,
		DestinoID: fixture.DestinoID, RotaInternaID: fixture.RotaInternaID,
		Comprovante: "ana.pdf",
		Validade:    time.Date(2031, 2, 2, 0, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)

	listed, err := store.List(ctx)
	require.NoError(t, err)
	require.Len(t, listed, 2)

	require.Equal(t, "Ana Costa", listed[0].ClienteNome)
	require.Equal(t, anaID, listed[0].ClienteID)
	require.Empty(t, listed[0].HorariosFixos)

	// O LEFT JOIN repete o vinculo uma vez por horario fixo; o coletor precisa
	// agrupar as linhas em vez de devolver o mesmo vinculo tres vezes.
	require.Equal(t, "Bruno Lima", listed[1].ClienteNome)
	require.Equal(t, brunoID, listed[1].ClienteID)
	require.Equal(t, "Direito", listed[1].Curso)
	require.Len(t, listed[1].HorariosFixos, 3)
	require.Equal(t, []clientes.DiaSemana{clientes.Segunda, clientes.Quarta, clientes.Sexta}, []clientes.DiaSemana{
		listed[1].HorariosFixos[0].DiaSemana,
		listed[1].HorariosFixos[1].DiaSemana,
		listed[1].HorariosFixos[2].DiaSemana,
	})
}

func TestVinculoRepository_CRUDWithHorarios(t *testing.T) {
	ctx, tx := beginTestTx(t)
	fixture := seedBaseFixture(t, ctx, tx)
	store := clientes.NewVinculoStore(tx)

	created, err := store.Create(ctx, clientes.VinculoInput{
		ClienteID: fixture.ClienteID, Tipo: clientes.TipoEstudante, Turno: clientes.TurnoNoturno,
		DestinoID: fixture.DestinoID, RotaInternaID: fixture.RotaInternaID,
		Curso: "Computacao", Comprovante: "comprovante.pdf",
		Validade:      time.Date(2030, 12, 31, 0, 0, 0, 0, time.UTC),
		HorariosFixos: []clientes.DiaSemana{clientes.Segunda, clientes.Quarta},
	})
	require.NoError(t, err)
	require.Len(t, created.HorariosFixos, 2)

	got, err := store.GetByID(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, []clientes.DiaSemana{clientes.Segunda, clientes.Quarta}, []clientes.DiaSemana{
		got.HorariosFixos[0].DiaSemana, got.HorariosFixos[1].DiaSemana,
	})

	updated, err := store.Update(ctx, created.ID, clientes.VinculoUpdateInput{
		Tipo: clientes.TipoEstagio, Turno: clientes.TurnoVespertino,
		DestinoID: fixture.DestinoID, RotaInternaID: fixture.RotaInternaID,
		Curso: "", Comprovante: "novo.pdf", Validade: time.Date(2031, 1, 1, 0, 0, 0, 0, time.UTC),
		HorariosFixos: []clientes.DiaSemana{clientes.Sexta},
	})
	require.NoError(t, err)
	require.Equal(t, clientes.TipoEstagio, updated.Tipo)
	require.Len(t, updated.HorariosFixos, 1)

	listed, err := store.ListByCliente(ctx, fixture.ClienteID)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.Equal(t, clientes.Sexta, listed[0].HorariosFixos[0].DiaSemana)

	require.NoError(t, store.Delete(ctx, created.ID))
	_, err = store.GetByID(ctx, created.ID)
	require.ErrorIs(t, err, clientes.ErrVinculoNotFound)
}
