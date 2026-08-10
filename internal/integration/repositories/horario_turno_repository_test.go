//go:build integration

package repositories

import (
	"testing"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
	"github.com/fredsaggio/bondrota-api/internal/viagens"
	"github.com/stretchr/testify/require"
)

func TestHorarioTurnoRepository_CRUD(t *testing.T) {
	ctx, tx := beginTestTx(t)
	store := viagens.NewHorarioTurnoViagemStore(tx)
	input := viagens.HorarioTurnoViagemInput{
		MunicipioDestinoID: testMunicipioID, Turno: viagens.TurnoNoturno,
		HorarioIda: 17 * time.Hour, HorarioVolta: 22 * time.Hour,
	}

	created, err := store.Create(ctx, input)
	require.NoError(t, err)
	require.Equal(t, testMunicipioID, created.MunicipioDestinoID)

	got, err := store.GetByMunicipioDestinoTurno(ctx, testMunicipioID, viagens.TurnoNoturno)
	require.NoError(t, err)
	require.Equal(t, created.ID, got.ID)

	updated, err := store.Update(ctx, created.ID, func(current *viagens.HorarioTurnoViagem) (bool, error) {
		current.HorarioIda = 18 * time.Hour
		return true, nil
	})
	require.NoError(t, err)
	require.Equal(t, 18*time.Hour, updated.HorarioIda)

	require.NoError(t, store.Delete(ctx, created.ID))
	_, err = store.GetByID(ctx, created.ID)
	require.ErrorIs(t, err, brerror.ErrNotFound)
}

func TestHorarioTurnoRepository_RejectsDuplicateMunicipioShift(t *testing.T) {
	ctx, tx := beginTestTx(t)
	store := viagens.NewHorarioTurnoViagemStore(tx)
	input := viagens.HorarioTurnoViagemInput{
		MunicipioDestinoID: testMunicipioID, Turno: viagens.TurnoNoturno,
		HorarioIda: 17 * time.Hour, HorarioVolta: 22 * time.Hour,
	}

	_, err := store.Create(ctx, input)
	require.NoError(t, err)
	_, err = store.Create(ctx, input)
	require.ErrorIs(t, err, brerror.ErrAlreadyExists)
}
