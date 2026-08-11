package admin_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/fredsaggio/bondrota-api/internal/admin"
	"github.com/fredsaggio/bondrota-api/internal/auth"
	"github.com/fredsaggio/bondrota-api/internal/mocks"
)

// stubHasher é um PasswordHasher simples para testes que não dependem do bcrypt.
type stubHasher struct {
	hashFn    func(string) (string, error)
	compareFn func(string, string) (bool, error)
}

func (s *stubHasher) Hash(p string) (string, error) { return s.hashFn(p) }
func (s *stubHasher) CompareHashAndPassword(hash, plain string) (bool, error) {
	return s.compareFn(hash, plain)
}

// newAuthSvc cria um AuthService com o hasher fornecido e uma chave JWT de teste.
func newAuthSvc(h *stubHasher) *auth.AuthService {
	return auth.NewAuthService(h, "test-secret-key-that-is-at-least-32-bytes-long!")
}

// okHasher aceita qualquer senha e retorna "hashed:<senha>".
func okHasher() *stubHasher {
	return &stubHasher{
		hashFn:    func(p string) (string, error) { return "hashed:" + p, nil },
		compareFn: func(hash, plain string) (bool, error) { return hash == "hashed:"+plain, nil },
	}
}

// failHasher retorna erro no Hash.
func failHasher() *stubHasher {
	return &stubHasher{
		hashFn:    func(p string) (string, error) { return "", errors.New("hash failed") },
		compareFn: func(hash, plain string) (bool, error) { return false, nil },
	}
}

var bgCtx = context.Background()

// --- Login ---

func TestAdminService_Login(t *testing.T) {
	tests := []struct {
		name      string
		email     string
		password  string
		setup     func(*mocks.MockAdminStore)
		hasher    *stubHasher
		wantErr   error
		wantToken bool
	}{
		{
			name:     "sucesso — credenciais corretas",
			email:    "admin@bondrota.com",
			password: "secret",
			setup: func(st *mocks.MockAdminStore) {
				st.EXPECT().GetByEmail(mock.Anything, "admin@bondrota.com").
					Return(&admin.Admin{ID: 1, Email: "admin@bondrota.com", Senha: "hashed:secret"}, nil)
			},
			hasher:    okHasher(),
			wantToken: true,
		},
		{
			name:     "senha errada → ErrInvalidCredentials",
			email:    "admin@bondrota.com",
			password: "wrong",
			setup: func(st *mocks.MockAdminStore) {
				st.EXPECT().GetByEmail(mock.Anything, "admin@bondrota.com").
					Return(&admin.Admin{ID: 1, Senha: "hashed:secret"}, nil)
			},
			hasher:  okHasher(),
			wantErr: auth.ErrInvalidCredentials,
		},
		{
			name:     "admin não encontrado → ErrNotFound",
			email:    "ghost@bondrota.com",
			password: "pw",
			setup: func(st *mocks.MockAdminStore) {
				st.EXPECT().GetByEmail(mock.Anything, "ghost@bondrota.com").
					Return(nil, admin.ErrNotFound)
			},
			hasher:  okHasher(),
			wantErr: admin.ErrNotFound,
		},
		{
			name:     "erro do store → propaga",
			email:    "a@a.com",
			password: "pw",
			setup: func(st *mocks.MockAdminStore) {
				st.EXPECT().GetByEmail(mock.Anything, "a@a.com").
					Return(nil, errors.New("db err"))
			},
			hasher: okHasher(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := mocks.NewMockAdminStore(t)
			tc.setup(store)

			svc := admin.NewAdminService(store, newAuthSvc(tc.hasher))
			token, err := svc.Login(bgCtx, tc.email, tc.password)

			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
				assert.Empty(t, token)
			} else if tc.wantToken {
				assert.NoError(t, err)
				assert.NotEmpty(t, token)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

// --- Create ---

func TestAdminService_Create(t *testing.T) {
	tests := []struct {
		name    string
		input   admin.AdminInput
		setup   func(*mocks.MockAdminStore)
		hasher  *stubHasher
		wantErr bool
	}{
		{
			name:  "sucesso — senha é hashada antes de persistir",
			input: admin.AdminInput{Email: "new@bondrota.com", Senha: "plain"},
			setup: func(st *mocks.MockAdminStore) {
				// store deve receber a senha hashada, não o plain text
				st.EXPECT().Create(mock.Anything, mock.MatchedBy(func(in admin.AdminInput) bool {
					return in.Email == "new@bondrota.com" && in.Senha == "hashed:plain"
				})).Return(&admin.Admin{ID: 1, Email: "new@bondrota.com"}, nil)
			},
			hasher: okHasher(),
		},
		{
			name:    "falha no hash → não chama store",
			input:   admin.AdminInput{Email: "x@x.com", Senha: "plain"},
			setup:   func(_ *mocks.MockAdminStore) {},
			hasher:  failHasher(),
			wantErr: true,
		},
		{
			name:  "falha no store → propaga erro",
			input: admin.AdminInput{Email: "x@x.com", Senha: "plain"},
			setup: func(st *mocks.MockAdminStore) {
				st.EXPECT().Create(mock.Anything, mock.Anything).Return(nil, errors.New("db err"))
			},
			hasher:  okHasher(),
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := mocks.NewMockAdminStore(t)
			tc.setup(store)

			svc := admin.NewAdminService(store, newAuthSvc(tc.hasher))
			a, err := svc.Create(bgCtx, tc.input)

			if tc.wantErr {
				assert.Error(t, err)
				assert.Nil(t, a)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, a)
			}
		})
	}
}

// --- Update ---

func TestAdminService_Update(t *testing.T) {
	tests := []struct {
		name    string
		adminID int64
		email   string
		setup   func(*mocks.MockAdminStore)
		wantErr error
	}{
		{
			name:    "sucesso — email atualizado",
			adminID: 1,
			email:   "novo@bondrota.com",
			setup: func(st *mocks.MockAdminStore) {
				st.EXPECT().Update(mock.Anything, int64(1), mock.Anything).
					RunAndReturn(func(_ context.Context, _ int64, fn func(*admin.Admin) (bool, error)) (*admin.Admin, error) {
						a := &admin.Admin{ID: 1, Email: "old@bondrota.com"}
						fn(a)
						return a, nil
					})
			},
		},
		{
			name:    "email igual — sem mudança",
			adminID: 1,
			email:   "same@bondrota.com",
			setup: func(st *mocks.MockAdminStore) {
				st.EXPECT().Update(mock.Anything, int64(1), mock.Anything).
					RunAndReturn(func(_ context.Context, _ int64, fn func(*admin.Admin) (bool, error)) (*admin.Admin, error) {
						a := &admin.Admin{ID: 1, Email: "same@bondrota.com"}
						fn(a) // changed=false, nenhuma modificação
						return a, nil
					})
			},
		},
		{
			name:    "não encontrado → ErrNotFound",
			adminID: 99,
			email:   "x@x.com",
			setup: func(st *mocks.MockAdminStore) {
				st.EXPECT().Update(mock.Anything, int64(99), mock.Anything).Return(nil, admin.ErrNotFound)
			},
			wantErr: admin.ErrNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := mocks.NewMockAdminStore(t)
			tc.setup(store)

			svc := admin.NewAdminService(store, newAuthSvc(okHasher()))
			_, err := svc.Update(bgCtx, tc.adminID, tc.email)

			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// --- GetByID ---

func TestAdminService_GetByID(t *testing.T) {
	tests := []struct {
		name    string
		id      int64
		setup   func(*mocks.MockAdminStore)
		wantErr error
	}{
		{
			name: "sucesso",
			id:   1,
			setup: func(st *mocks.MockAdminStore) {
				st.EXPECT().GetByID(mock.Anything, int64(1)).Return(sampleAdmin(), nil)
			},
		},
		{
			name: "não encontrado",
			id:   99,
			setup: func(st *mocks.MockAdminStore) {
				st.EXPECT().GetByID(mock.Anything, int64(99)).Return(nil, admin.ErrNotFound)
			},
			wantErr: admin.ErrNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := mocks.NewMockAdminStore(t)
			tc.setup(store)
			svc := admin.NewAdminService(store, newAuthSvc(okHasher()))
			_, err := svc.GetByID(bgCtx, tc.id)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// --- Delete ---

func TestAdminService_Delete(t *testing.T) {
	tests := []struct {
		name    string
		id      int64
		setup   func(*mocks.MockAdminStore)
		wantErr error
	}{
		{
			name:  "sucesso",
			id:    1,
			setup: func(st *mocks.MockAdminStore) { st.EXPECT().Delete(mock.Anything, int64(1)).Return(nil) },
		},
		{
			name:    "não encontrado",
			id:      99,
			setup:   func(st *mocks.MockAdminStore) { st.EXPECT().Delete(mock.Anything, int64(99)).Return(admin.ErrNotFound) },
			wantErr: admin.ErrNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := mocks.NewMockAdminStore(t)
			tc.setup(store)
			svc := admin.NewAdminService(store, newAuthSvc(okHasher()))
			err := svc.Delete(bgCtx, tc.id)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// --- List ---

func TestAdminService_List(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*mocks.MockAdminStore)
		wantLen int
		wantErr bool
	}{
		{
			name: "retorna lista",
			setup: func(st *mocks.MockAdminStore) {
				st.EXPECT().List(mock.Anything).Return([]admin.Admin{*sampleAdmin()}, nil)
			},
			wantLen: 1,
		},
		{
			name:    "lista vazia",
			setup:   func(st *mocks.MockAdminStore) { st.EXPECT().List(mock.Anything).Return([]admin.Admin{}, nil) },
			wantLen: 0,
		},
		{
			name:    "erro do store",
			setup:   func(st *mocks.MockAdminStore) { st.EXPECT().List(mock.Anything).Return(nil, errors.New("db")) },
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := mocks.NewMockAdminStore(t)
			tc.setup(store)
			svc := admin.NewAdminService(store, newAuthSvc(okHasher()))
			list, err := svc.List(bgCtx)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, list, tc.wantLen)
			}
		})
	}
}

// --- ChangePassword ---

func TestAdminService_ChangePassword(t *testing.T) {
	const (
		senhaAtual = "senha-atual"
		novaSenha  = "nova-senha-1"
	)

	t.Run("sucesso — grava o hash da nova senha e devolve token do proprio admin", func(t *testing.T) {
		var gravada string
		store := mocks.NewMockAdminStore(t)
		store.EXPECT().Update(mock.Anything, int64(7), mock.Anything).
			RunAndReturn(func(_ context.Context, _ int64, fn func(*admin.Admin) (bool, error)) (*admin.Admin, error) {
				a := &admin.Admin{ID: 7, Email: "a@b.com", Senha: "hashed:" + senhaAtual}
				updated, err := fn(a)
				assert.NoError(t, err)
				assert.True(t, updated)
				gravada = a.Senha
				return a, nil
			})

		authSvc := newAuthSvc(okHasher())
		token, err := admin.NewAdminService(store, authSvc).ChangePassword(bgCtx, 7, senhaAtual, novaSenha)

		assert.NoError(t, err)
		assert.Equal(t, "hashed:"+novaSenha, gravada)

		claims, err := authSvc.ValidateToken(token)
		assert.NoError(t, err)
		assert.Equal(t, int64(7), claims.UserID)
		assert.Equal(t, auth.RoleAdmin, claims.Role)
	})

	// Sem isso, um cookie roubado bastaria para tomar a conta de vez: o atacante
	// trocaria a senha e trancaria o admin legitimo do lado de fora.
	t.Run("senha atual errada — nao grava e devolve ErrInvalidCredentials", func(t *testing.T) {
		store := mocks.NewMockAdminStore(t)
		store.EXPECT().Update(mock.Anything, int64(7), mock.Anything).
			RunAndReturn(func(_ context.Context, _ int64, fn func(*admin.Admin) (bool, error)) (*admin.Admin, error) {
				a := &admin.Admin{ID: 7, Senha: "hashed:" + senhaAtual}
				updated, err := fn(a)
				assert.False(t, updated)
				assert.Equal(t, "hashed:"+senhaAtual, a.Senha, "a senha nao pode ser tocada")
				return nil, err
			})

		token, err := admin.NewAdminService(store, newAuthSvc(okHasher())).
			ChangePassword(bgCtx, 7, "senha-errada", novaSenha)

		assert.ErrorIs(t, err, auth.ErrInvalidCredentials)
		assert.Empty(t, token)
	})

	// O store nem chega a ser chamado: MockAdminStore falha o teste se receber algo
	// que nao foi declarado com EXPECT.
	t.Run("nova senha curta — rejeita antes de tocar no banco", func(t *testing.T) {
		store := mocks.NewMockAdminStore(t)

		token, err := admin.NewAdminService(store, newAuthSvc(okHasher())).
			ChangePassword(bgCtx, 7, senhaAtual, "curta")

		assert.ErrorIs(t, err, admin.ErrSenhaFraca)
		assert.Empty(t, token)
	})

	t.Run("admin inexistente — ErrNotFound", func(t *testing.T) {
		store := mocks.NewMockAdminStore(t)
		store.EXPECT().Update(mock.Anything, int64(99), mock.Anything).Return(nil, admin.ErrNotFound)

		_, err := admin.NewAdminService(store, newAuthSvc(okHasher())).
			ChangePassword(bgCtx, 99, senhaAtual, novaSenha)

		assert.ErrorIs(t, err, admin.ErrNotFound)
	})
}

func TestValidarSenha(t *testing.T) {
	assert.ErrorIs(t, admin.ValidarSenha(""), admin.ErrSenhaFraca)
	assert.ErrorIs(t, admin.ValidarSenha("1234567"), admin.ErrSenhaFraca)
	assert.NoError(t, admin.ValidarSenha("12345678"))
	// Acentuada com 8 letras tem mais de 8 bytes: a regra conta caracteres.
	assert.NoError(t, admin.ValidarSenha("senhaçãí"))
}
