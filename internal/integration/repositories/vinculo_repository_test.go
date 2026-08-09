//go:build integration

package repositories

import (
	"testing"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/clientes"
	"github.com/stretchr/testify/require"
)

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
