package motoristas_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/fredsaggio/bondrota-api/internal/auth"
	"github.com/fredsaggio/bondrota-api/internal/mocks"
	"github.com/fredsaggio/bondrota-api/internal/motoristas"
)

// ---- hasher stub (igual ao do admin_test, local ao package) ----

type stubHasher struct {
	hashFn    func(string) (string, error)
	compareFn func(string, string) (bool, error)
}

func (s *stubHasher) Hash(p string) (string, error)                    { return s.hashFn(p) }
func (s *stubHasher) CompareHashAndPassword(h, p string) (bool, error) { return s.compareFn(h, p) }

func okHasher() *stubHasher {
	return &stubHasher{
		hashFn:    func(p string) (string, error) { return "hashed:" + p, nil },
		compareFn: func(hash, plain string) (bool, error) { return hash == "hashed:"+plain, nil },
	}
}

func failHasher() *stubHasher {
	return &stubHasher{
		hashFn:    func(_ string) (string, error) { return "", errors.New("hash failed") },
		compareFn: func(_, _ string) (bool, error) { return false, nil },
	}
}

func newAuthSvc(h *stubHasher) *auth.AuthService {
	return auth.NewAuthService(h, "test-secret-key-that-is-at-least-32-bytes-long!")
}

var svcCtx = context.Background()

func baseMotoristaInput() motoristas.MotoristaInput {
	return motoristas.MotoristaInput{
		Nome:     "João Silva",
		CPF:      "123.456.789-00",
		Senha:    "plain",
		DataNasc: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		Turno:    motoristas.TurnoMatutino,
	}
}

// --- Login ---

func TestMotoristaService_Login(t *testing.T) {
	tests := []struct {
		name      string
		cpf       string
		senha     string
		setup     func(*mocks.MockMotoristaStore)
		hasher    *stubHasher
		wantToken bool
		wantErr   error
	}{
		{
			name:  "sucesso — credenciais corretas",
			cpf:   "123.456.789-00",
			senha: "plain",
			setup: func(st *mocks.MockMotoristaStore) {
				st.EXPECT().GetByCPF(mock.Anything, "123.456.789-00").
					Return(&motoristas.Motorista{ID: 1, PublicID: "mot_012345678901234567890", CPF: "123.456.789-00", Senha: "hashed:plain"}, nil)
			},
			hasher:    okHasher(),
			wantToken: true,
		},
		{
			name:  "senha incorreta → ErrInvalidCredentials",
			cpf:   "123.456.789-00",
			senha: "wrong",
			setup: func(st *mocks.MockMotoristaStore) {
				st.EXPECT().GetByCPF(mock.Anything, "123.456.789-00").
					Return(&motoristas.Motorista{ID: 1, PublicID: "mot_012345678901234567890", Senha: "hashed:plain"}, nil)
			},
			hasher:  okHasher(),
			wantErr: auth.ErrInvalidCredentials,
		},
		{
			name:  "motorista não encontrado → propaga erro do store",
			cpf:   "000.000.000-00",
			senha: "pw",
			setup: func(st *mocks.MockMotoristaStore) {
				st.EXPECT().GetByCPF(mock.Anything, "000.000.000-00").
					Return(nil, motoristas.ErrNotFound)
			},
			hasher: okHasher(),
		},
		{
			name:  "erro do store → propaga",
			cpf:   "a@a.com",
			senha: "pw",
			setup: func(st *mocks.MockMotoristaStore) {
				st.EXPECT().GetByCPF(mock.Anything, "a@a.com").Return(nil, errors.New("db err"))
			},
			hasher: okHasher(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := mocks.NewMockMotoristaStore(t)
			tc.setup(store)

			svc := motoristas.NewMotoristaService(store, newAuthSvc(tc.hasher))
			token, err := svc.Login(svcCtx, tc.cpf, tc.senha)

			switch {
			case tc.wantErr != nil:
				assert.ErrorIs(t, err, tc.wantErr)
				assert.Empty(t, token)
			case tc.wantToken:
				assert.NoError(t, err)
				assert.NotEmpty(t, token)
			default:
				assert.Error(t, err)
			}
		})
	}
}

// --- Create ---

func TestMotoristaService_Create(t *testing.T) {
	tests := []struct {
		name    string
		input   motoristas.MotoristaInput
		setup   func(*mocks.MockMotoristaStore)
		hasher  *stubHasher
		wantErr bool
	}{
		{
			name:  "sucesso — senha hashada antes de persistir",
			input: baseMotoristaInput(),
			setup: func(st *mocks.MockMotoristaStore) {
				st.EXPECT().Create(mock.Anything, mock.MatchedBy(func(in motoristas.MotoristaInput) bool {
					return in.Senha == "hashed:plain" && in.Nome == "João Silva"
				})).Return(sampleMotorista(), nil)
			},
			hasher: okHasher(),
		},
		{
			name:    "falha no hash → não chama store",
			input:   baseMotoristaInput(),
			setup:   func(_ *mocks.MockMotoristaStore) {},
			hasher:  failHasher(),
			wantErr: true,
		},
		{
			name:  "erro do store → propaga",
			input: baseMotoristaInput(),
			setup: func(st *mocks.MockMotoristaStore) {
				st.EXPECT().Create(mock.Anything, mock.Anything).Return(nil, errors.New("db err"))
			},
			hasher:  okHasher(),
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := mocks.NewMockMotoristaStore(t)
			tc.setup(store)

			svc := motoristas.NewMotoristaService(store, newAuthSvc(tc.hasher))
			m, err := svc.Create(svcCtx, tc.input)

			if tc.wantErr {
				assert.Error(t, err)
				assert.Nil(t, m)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, m)
			}
		})
	}
}

// --- GetByID ---

func TestMotoristaService_GetByID(t *testing.T) {
	tests := []struct {
		name    string
		id      int64
		setup   func(*mocks.MockMotoristaStore)
		wantErr error
	}{
		{
			name: "sucesso",
			id:   1,
			setup: func(st *mocks.MockMotoristaStore) {
				st.EXPECT().GetByID(mock.Anything, int64(1)).Return(sampleMotorista(), nil)
			},
		},
		{
			name: "não encontrado → ErrNotFound",
			id:   99,
			setup: func(st *mocks.MockMotoristaStore) {
				st.EXPECT().GetByID(mock.Anything, int64(99)).Return(nil, motoristas.ErrNotFound)
			},
			wantErr: motoristas.ErrNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := mocks.NewMockMotoristaStore(t)
			tc.setup(store)
			svc := motoristas.NewMotoristaService(store, newAuthSvc(okHasher()))
			_, err := svc.GetByID(svcCtx, tc.id)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// --- Update (delegação direta ao store) ---

func TestMotoristaService_Update(t *testing.T) {
	tests := []struct {
		name    string
		id      int64
		setup   func(*mocks.MockMotoristaStore)
		wantErr error
	}{
		{
			name: "sucesso",
			id:   1,
			setup: func(st *mocks.MockMotoristaStore) {
				st.EXPECT().Update(mock.Anything, int64(1), mock.Anything).
					RunAndReturn(func(_ context.Context, _ int64, fn func(*motoristas.Motorista) (bool, error)) (*motoristas.Motorista, error) {
						m := sampleMotorista()
						fn(m)
						return m, nil
					})
			},
		},
		{
			name: "não encontrado → ErrNotFound",
			id:   99,
			setup: func(st *mocks.MockMotoristaStore) {
				st.EXPECT().Update(mock.Anything, int64(99), mock.Anything).Return(nil, motoristas.ErrNotFound)
			},
			wantErr: motoristas.ErrNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := mocks.NewMockMotoristaStore(t)
			tc.setup(store)
			svc := motoristas.NewMotoristaService(store, newAuthSvc(okHasher()))
			_, err := svc.Update(svcCtx, tc.id, func(m *motoristas.Motorista) (bool, error) {
				m.Nome = "Novo Nome"
				return true, nil
			})
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// --- Delete ---

func TestMotoristaService_Delete(t *testing.T) {
	tests := []struct {
		name    string
		id      int64
		setup   func(*mocks.MockMotoristaStore)
		wantErr error
	}{
		{
			name:  "sucesso",
			id:    1,
			setup: func(st *mocks.MockMotoristaStore) { st.EXPECT().Delete(mock.Anything, int64(1)).Return(nil) },
		},
		{
			name: "não encontrado → ErrNotFound",
			id:   99,
			setup: func(st *mocks.MockMotoristaStore) {
				st.EXPECT().Delete(mock.Anything, int64(99)).Return(motoristas.ErrNotFound)
			},
			wantErr: motoristas.ErrNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := mocks.NewMockMotoristaStore(t)
			tc.setup(store)
			svc := motoristas.NewMotoristaService(store, newAuthSvc(okHasher()))
			err := svc.Delete(svcCtx, tc.id)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// --- List ---

func TestMotoristaService_List(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*mocks.MockMotoristaStore)
		wantLen int
		wantErr bool
	}{
		{
			name: "retorna lista",
			setup: func(st *mocks.MockMotoristaStore) {
				st.EXPECT().List(mock.Anything).Return([]motoristas.Motorista{*sampleMotorista()}, nil)
			},
			wantLen: 1,
		},
		{
			name: "lista vazia",
			setup: func(st *mocks.MockMotoristaStore) {
				st.EXPECT().List(mock.Anything).Return([]motoristas.Motorista{}, nil)
			},
			wantLen: 0,
		},
		{
			name:    "erro do store",
			setup:   func(st *mocks.MockMotoristaStore) { st.EXPECT().List(mock.Anything).Return(nil, errors.New("db")) },
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := mocks.NewMockMotoristaStore(t)
			tc.setup(store)
			svc := motoristas.NewMotoristaService(store, newAuthSvc(okHasher()))
			list, err := svc.List(svcCtx)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, list, tc.wantLen)
			}
		})
	}
}
