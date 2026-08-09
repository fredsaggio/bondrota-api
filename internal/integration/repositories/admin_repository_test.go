//go:build integration

package repositories

import (
	"testing"

	"github.com/fredsaggio/bondrota-api/internal/admin"
	"github.com/stretchr/testify/require"
)

func TestAdminRepository_CRUD(t *testing.T) {
	ctx, tx := beginTestTx(t)
	store := admin.NewAdminStore(tx)

	created, err := store.Create(ctx, admin.AdminInput{Email: "admin@integration.test", Senha: "hash-secreto"})
	require.NoError(t, err)
	require.NotZero(t, created.ID)

	byEmail, err := store.GetByEmail(ctx, created.Email)
	require.NoError(t, err)
	require.Equal(t, "hash-secreto", byEmail.Senha)

	updated, err := store.Update(ctx, created.ID, func(current *admin.Admin) (bool, error) {
		current.Email = "admin-updated@integration.test"
		return true, nil
	})
	require.NoError(t, err)
	require.Equal(t, "admin-updated@integration.test", updated.Email)

	byEmail, err = store.GetByEmail(ctx, updated.Email)
	require.NoError(t, err)
	require.Equal(t, "hash-secreto", byEmail.Senha, "updating the email must preserve the password hash")

	admins, err := store.List(ctx)
	require.NoError(t, err)
	require.Len(t, admins, 1)

	require.NoError(t, store.Delete(ctx, created.ID))
	_, err = store.GetByID(ctx, created.ID)
	require.ErrorIs(t, err, admin.ErrNotFound)
}
