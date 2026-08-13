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
		DataNasc: birthDate, DocumentoIdentificacao: "ana-identidade.pdf", ComprovanteResidencia: "ana-residencia.pdf",
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

	listed, err := store.List(ctx, clientes.ClienteListParams{})
	require.NoError(t, err)
	require.Len(t, listed.Items, 1)

	require.NoError(t, store.Delete(ctx, created.ID))
	_, err = store.GetByID(ctx, created.ID)
	require.ErrorIs(t, err, clientes.ErrNotFound)
}

func TestClienteRepository_RejectsDuplicateCPF(t *testing.T) {
	ctx, tx := beginTestTx(t)
	store := clientes.NewClienteStore(tx)
	input := clientes.ClienteInput{
		Nome: "Ana", CPF: "30000000002", Senha: "hash", DataNasc: time.Date(2001, 1, 10, 0, 0, 0, 0, time.UTC),
		DocumentoIdentificacao: "identidade.pdf", ComprovanteResidencia: "residencia.pdf",
	}

	_, err := store.Create(ctx, input)
	require.NoError(t, err)
	_, err = store.Create(ctx, input)
	require.Error(t, err)
}

func TestClienteRepository_ListPaginaEBusca(t *testing.T) {
	ctx, tx := beginTestTx(t)
	store := clientes.NewClienteStore(tx)
	nascimento := time.Date(2001, time.January, 10, 0, 0, 0, 0, time.UTC)

	criar := func(nome, cpf, telefone string) int64 {
		created, err := store.Create(ctx, clientes.ClienteInput{
			Nome: nome, CPF: cpf, Senha: "hash", Telefone: telefone,
			DataNasc: nascimento, DocumentoIdentificacao: "identidade.pdf", ComprovanteResidencia: "residencia.pdf",
		})
		require.NoError(t, err)
		return created.ID
	}

	primeiro := criar("Ana Beatriz", "30000000001", "82999990001")
	segundo := criar("Bruno Costa", "30000000002", "82999990002")
	terceiro := criar("Carla Dias", "30000000003", "82988887777")

	t.Run("pagina por id decrescente", func(t *testing.T) {
		first, err := store.List(ctx, clientes.ClienteListParams{Limit: 2})
		require.NoError(t, err)
		require.Len(t, first.Items, 2)
		require.Equal(t, terceiro, first.Items[0].ID)
		require.Equal(t, segundo, first.Items[1].ID)
		require.True(t, first.HasMore)
		require.Equal(t, segundo, first.NextCursorID)

		second, err := store.List(ctx, clientes.ClienteListParams{Limit: 2, CursorID: first.NextCursorID})
		require.NoError(t, err)
		require.Len(t, second.Items, 1)
		require.Equal(t, primeiro, second.Items[0].ID)
		require.False(t, second.HasMore)
	})

	t.Run("busca por nome parcial e case-insensitive", func(t *testing.T) {
		result, err := store.List(ctx, clientes.ClienteListParams{Busca: "bruno cos"})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		require.Equal(t, segundo, result.Items[0].ID)
	})

	// O cadastro guarda o CPF so com digitos, mas quem busca costuma colar o
	// documento formatado.
	t.Run("busca por CPF com pontuacao", func(t *testing.T) {
		result, err := store.List(ctx, clientes.ClienteListParams{Busca: "300.000.000-03"})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		require.Equal(t, terceiro, result.Items[0].ID)
	})

	// Sem essa regra, "Cliente 13" viraria tambem uma busca por CPF contendo "13"
	// e traria junto o cliente 113, cujo documento tem esses digitos no meio.
	t.Run("termo com letra nao dispara busca por CPF", func(t *testing.T) {
		result, err := store.List(ctx, clientes.ClienteListParams{Busca: "Ana Beatriz 30000000002"})
		require.NoError(t, err)
		require.Empty(t, result.Items, "termo com letra so casa por nome/telefone")
	})

	t.Run("busca por telefone", func(t *testing.T) {
		result, err := store.List(ctx, clientes.ClienteListParams{Busca: "82988887777"})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		require.Equal(t, terceiro, result.Items[0].ID)
	})

	t.Run("busca combina com o cursor", func(t *testing.T) {
		// "3000000000" casa com os tres; paginado de um em um.
		first, err := store.List(ctx, clientes.ClienteListParams{Busca: "3000000000", Limit: 1})
		require.NoError(t, err)
		require.Len(t, first.Items, 1)
		require.Equal(t, terceiro, first.Items[0].ID)
		require.True(t, first.HasMore)

		second, err := store.List(ctx, clientes.ClienteListParams{Busca: "3000000000", Limit: 1, CursorID: first.NextCursorID})
		require.NoError(t, err)
		require.Len(t, second.Items, 1)
		require.Equal(t, segundo, second.Items[0].ID)
	})
}

func TestClienteRepository_Resumo(t *testing.T) {
	ctx, tx := beginTestTx(t)
	store := clientes.NewClienteStore(tx)
	nascimento := time.Date(2001, time.January, 10, 0, 0, 0, 0, time.UTC)

	resumo, err := store.Resumo(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(0), resumo.Total)

	for _, cpf := range []string{"30000000001", "30000000002"} {
		_, err := store.Create(ctx, clientes.ClienteInput{
			Nome: "Cliente", CPF: cpf, Senha: "hash", Telefone: "", DataNasc: nascimento,
			DocumentoIdentificacao: "identidade.pdf", ComprovanteResidencia: "residencia.pdf",
		})
		require.NoError(t, err)
	}

	resumo, err = store.Resumo(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(2), resumo.Total)
}
