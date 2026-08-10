//go:build integration

package repositories

import (
	"testing"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/clientes"
	"github.com/stretchr/testify/require"
)

func TestClienteRepository_CRUD(t *testing.T) {
	ctx, tx := beginTestTx(t)
	store := clientes.NewClienteStore(tx)
	birthDate := time.Date(2001, time.January, 10, 0, 0, 0, 0, time.UTC)

	created, err := store.Create(ctx, clientes.ClienteInput{
		Nome: "Ana", CPF: "30000000001", Senha: "hash", Telefone: "82999991111",
		DataNasc: birthDate, Foto: "ana.jpg",
	})
	require.NoError(t, err)

	byCPF, err := store.GetByCPF(ctx, created.CPF)
	require.NoError(t, err)
	require.Equal(t, "hash", byCPF.Senha)

	withLinks, err := store.GetByID(ctx, created.ID)
	require.NoError(t, err)
	require.Empty(t, withLinks.Vinculos)

	updated, err := store.Update(ctx, created.ID, func(current *clientes.Cliente) (bool, error) {
		current.Nome = "Ana Maria"
		return true, nil
	})
	require.NoError(t, err)
	require.Equal(t, "Ana Maria", updated.Nome)

	listed, err := store.List(ctx)
	require.NoError(t, err)
	require.Len(t, listed, 1)

	require.NoError(t, store.Delete(ctx, created.ID))
	_, err = store.GetByID(ctx, created.ID)
	require.ErrorIs(t, err, clientes.ErrNotFound)
}

func TestClienteRepository_RejectsDuplicateCPF(t *testing.T) {
	ctx, tx := beginTestTx(t)
	store := clientes.NewClienteStore(tx)
	input := clientes.ClienteInput{
		Nome: "Ana", CPF: "30000000002", Senha: "hash", DataNasc: time.Date(2001, 1, 10, 0, 0, 0, 0, time.UTC),
	}

	_, err := store.Create(ctx, input)
	require.NoError(t, err)
	_, err = store.Create(ctx, input)
	require.Error(t, err)
}
