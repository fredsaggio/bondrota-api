package clientes_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/fredsaggio/bondrota-api/internal/auth"
	"github.com/fredsaggio/bondrota-api/internal/clientes"
	"github.com/fredsaggio/bondrota-api/internal/mocks"
)

type fakeHasher struct {
	hashErr error
	ok      bool
}

func (h fakeHasher) Hash(s string) (string, error) {
	if h.hashErr != nil {
		return "", h.hashErr
	}
	return "hashed:" + s, nil
}

func (h fakeHasher) CompareHashAndPassword(_, _ string) (bool, error) {
	return h.ok, nil
}

func newTestAuth(ok bool) *auth.AuthService {
	return auth.NewAuthService(fakeHasher{ok: ok}, "test-secret")
}

func sampleCliente() *clientes.Cliente {
	return &clientes.Cliente{
		ID:                     1,
		Nome:                   "Maria",
		CPF:                    "12345678900",
		Senha:                  "hashed:secret",
		Telefone:               "82999999999",
		DataNasc:               time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC),
		DocumentoIdentificacao: "identidade.pdf",
		ComprovanteResidencia:  "residencia.pdf",
	}
}

func sampleVinculo() *clientes.Vinculo {
	return &clientes.Vinculo{
		ID:            10,
		ClienteID:     1,
		Tipo:          clientes.TipoEstudante,
		Turno:         clientes.TurnoNoturno,
		DestinoID:     2,
		RotaInternaID: 3,
		Curso:         "Computacao",
		Comprovante:   "doc.pdf",
		Validade:      time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		HorariosFixos: []clientes.HorarioFixo{{ID: 1, VinculoID: 10, DiaSemana: clientes.Segunda}},
	}
}

func TestClienteService_Create(t *testing.T) {
	t.Run("hashes password and delegates to store", func(t *testing.T) {
		store := mocks.NewMockClienteStore(t)
		svc := clientes.NewClienteService(store, newTestAuth(true))

		input := clientes.ClienteInput{
			Nome:     "Maria",
			CPF:      "123",
			Senha:    "secret",
			DataNasc: sampleCliente().DataNasc,
		}

		store.EXPECT().Create(mock.Anything, mock.MatchedBy(func(in clientes.ClienteInput) bool {
			return in.Senha == "hashed:secret" && in.CPF == "123"
		})).Return(sampleCliente(), nil)

		cliente, err := svc.Create(context.Background(), input)

		assert.NoError(t, err)
		assert.Equal(t, int64(1), cliente.ID)
	})

	t.Run("returns hash error", func(t *testing.T) {
		store := mocks.NewMockClienteStore(t)
		authSvc := auth.NewAuthService(fakeHasher{hashErr: errors.New("hash err")}, "test-secret")
		svc := clientes.NewClienteService(store, authSvc)

		_, err := svc.Create(context.Background(), clientes.ClienteInput{Senha: "secret"})

		assert.Error(t, err)
	})
}

func TestClienteService_Login(t *testing.T) {
	t.Run("valid credentials return token", func(t *testing.T) {
		store := mocks.NewMockClienteStore(t)
		authSvc := newTestAuth(true)
		svc := clientes.NewClienteService(store, authSvc)

		store.EXPECT().GetByCPF(mock.Anything, "123").Return(sampleCliente(), nil)

		token, err := svc.Login(context.Background(), "123", "secret")

		assert.NoError(t, err)
		assert.NotEmpty(t, token)
		claims, err := authSvc.ValidateToken(token)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), claims.UserID)
		assert.Equal(t, "cliente", claims.Role)
	})

	t.Run("invalid password returns ErrInvalidCredentials", func(t *testing.T) {
		store := mocks.NewMockClienteStore(t)
		svc := clientes.NewClienteService(store, newTestAuth(false))

		store.EXPECT().GetByCPF(mock.Anything, "123").Return(sampleCliente(), nil)

		_, err := svc.Login(context.Background(), "123", "wrong")

		assert.ErrorIs(t, err, auth.ErrInvalidCredentials)
	})

	t.Run("store error is returned", func(t *testing.T) {
		store := mocks.NewMockClienteStore(t)
		svc := clientes.NewClienteService(store, newTestAuth(true))

		store.EXPECT().GetByCPF(mock.Anything, "404").Return(nil, clientes.ErrNotFound)

		_, err := svc.Login(context.Background(), "404", "secret")

		assert.ErrorIs(t, err, clientes.ErrNotFound)
	})
}

func TestClienteService_UpdateDelegates(t *testing.T) {
	store := mocks.NewMockClienteStore(t)
	svc := clientes.NewClienteService(store, newTestAuth(true))

	store.EXPECT().Update(mock.Anything, int64(1), mock.Anything).RunAndReturn(
		func(_ context.Context, clienteID int64, updateFunc func(*clientes.Cliente) (bool, error)) (*clientes.Cliente, error) {
			c := sampleCliente()
			changed, err := updateFunc(c)
			assert.NoError(t, err)
			assert.True(t, changed)
			return c, nil
		},
	)

	cliente, err := svc.Update(context.Background(), 1, func(c *clientes.Cliente) (bool, error) {
		c.Nome = "Ana"
		return true, nil
	})

	assert.NoError(t, err)
	assert.Equal(t, "Ana", cliente.Nome)
}
