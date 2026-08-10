package rotasinternas_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/fredsaggio/bondrota-api/internal/mocks"
	"github.com/fredsaggio/bondrota-api/internal/rotasinternas"
)

var svcCtx = context.Background()

// --- Create ---

func TestRotaInternaService_Create(t *testing.T) {
	tests := []struct {
		name    string
		input   rotasinternas.CreateRotaInternaInput
		setup   func(*mocks.MockRotaInternaStore)
		wantErr error
	}{
		{
			name: "sucesso",
			input: rotasinternas.CreateRotaInternaInput{
				Paradas: []rotasinternas.ParadaInput{
					{ParadaID: 1, Ordem: 1},
					{ParadaID: 2, Ordem: 2},
				},
			},
			setup: func(st *mocks.MockRotaInternaStore) {
				st.EXPECT().Create(mock.Anything, mock.Anything).Return(sampleRotaInterna(), nil)
			},
		},
		{
			name: "erro - sem paradas",
			input: rotasinternas.CreateRotaInternaInput{
				Paradas: []rotasinternas.ParadaInput{},
			},
			setup:   func(st *mocks.MockRotaInternaStore) {},
			wantErr: rotasinternas.ErrSemParadas,
		},
		{
			name: "erro - parada_id inválido",
			input: rotasinternas.CreateRotaInternaInput{
				Paradas: []rotasinternas.ParadaInput{
					{ParadaID: 0, Ordem: 1},
				},
			},
			setup:   func(st *mocks.MockRotaInternaStore) {},
			wantErr: rotasinternas.ErrParadaInvalida,
		},
		{
			name: "erro - ordem inválida",
			input: rotasinternas.CreateRotaInternaInput{
				Paradas: []rotasinternas.ParadaInput{
					{ParadaID: 1, Ordem: 0},
				},
			},
			setup:   func(st *mocks.MockRotaInternaStore) {},
			wantErr: rotasinternas.ErrParadaInvalida,
		},
		{
			name: "erro - ordem duplicada",
			input: rotasinternas.CreateRotaInternaInput{
				Paradas: []rotasinternas.ParadaInput{
					{ParadaID: 1, Ordem: 1},
					{ParadaID: 2, Ordem: 1},
				},
			},
			setup:   func(st *mocks.MockRotaInternaStore) {},
			wantErr: rotasinternas.ErrOrdemDuplicada,
		},
		{
			name: "erro do store",
			input: rotasinternas.CreateRotaInternaInput{
				Paradas: []rotasinternas.ParadaInput{
					{ParadaID: 1, Ordem: 1},
				},
			},
			setup: func(st *mocks.MockRotaInternaStore) {
				st.EXPECT().Create(mock.Anything, mock.Anything).Return(nil, errors.New("db error"))
			},
			wantErr: errors.New("service/rotaInternaService.Create: db error"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			st := mocks.NewMockRotaInternaStore(t)
			tc.setup(st)
			svc := rotasinternas.NewRotaInternaService(st)

			rota, err := svc.Create(svcCtx, tc.input)
			if tc.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tc.wantErr)
				}
				assert.Equal(t, tc.wantErr.Error(), err.Error())
				assert.Nil(t, rota)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, rota)
			}
		})
	}
}

// --- UpdateParadas ---

func TestRotaInternaService_UpdateParadas(t *testing.T) {
	tests := []struct {
		name    string
		id      int64
		input   rotasinternas.UpdateParadasInput
		setup   func(*mocks.MockRotaInternaStore)
		wantErr error
	}{
		{
			name: "sucesso",
			id:   1,
			input: rotasinternas.UpdateParadasInput{
				Paradas: []rotasinternas.ParadaInput{
					{ParadaID: 1, Ordem: 1},
				},
			},
			setup: func(st *mocks.MockRotaInternaStore) {
				st.EXPECT().UpdateParadas(mock.Anything, int64(1), mock.Anything).Return(sampleRotaInterna(), nil)
			},
		},
		{
			name: "erro - validação (sem paradas)",
			id:   1,
			input: rotasinternas.UpdateParadasInput{
				Paradas: []rotasinternas.ParadaInput{},
			},
			setup:   func(st *mocks.MockRotaInternaStore) {},
			wantErr: rotasinternas.ErrSemParadas,
		},
		{
			name: "erro do store",
			id:   1,
			input: rotasinternas.UpdateParadasInput{
				Paradas: []rotasinternas.ParadaInput{
					{ParadaID: 1, Ordem: 1},
				},
			},
			setup: func(st *mocks.MockRotaInternaStore) {
				st.EXPECT().UpdateParadas(mock.Anything, int64(1), mock.Anything).Return(nil, errors.New("db"))
			},
			wantErr: errors.New("service/rotaInternaService.UpdateParadas: db"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			st := mocks.NewMockRotaInternaStore(t)
			tc.setup(st)
			svc := rotasinternas.NewRotaInternaService(st)

			rota, err := svc.UpdateParadas(svcCtx, tc.id, tc.input)
			if tc.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tc.wantErr)
				}
				assert.Equal(t, tc.wantErr.Error(), err.Error())
				assert.Nil(t, rota)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, rota)
			}
		})
	}
}

// --- GetByID ---

func TestRotaInternaService_GetByID(t *testing.T) {
	tests := []struct {
		name    string
		id      int64
		setup   func(*mocks.MockRotaInternaStore)
		wantErr error
	}{
		{
			name: "sucesso",
			id:   1,
			setup: func(st *mocks.MockRotaInternaStore) {
				st.EXPECT().GetByID(mock.Anything, int64(1)).Return(sampleRotaInterna(), nil)
			},
		},
		{
			name: "não encontrado",
			id:   99,
			setup: func(st *mocks.MockRotaInternaStore) {
				st.EXPECT().GetByID(mock.Anything, int64(99)).Return(nil, rotasinternas.ErrNotFound)
			},
			wantErr: rotasinternas.ErrNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			st := mocks.NewMockRotaInternaStore(t)
			tc.setup(st)
			svc := rotasinternas.NewRotaInternaService(st)

			_, err := svc.GetByID(svcCtx, tc.id)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// --- List ---

func TestRotaInternaService_List(t *testing.T) {
	st := mocks.NewMockRotaInternaStore(t)
	st.EXPECT().List(mock.Anything).Return([]rotasinternas.RotaInterna{*sampleRotaInterna()}, nil)

	svc := rotasinternas.NewRotaInternaService(st)
	list, err := svc.List(svcCtx)

	assert.NoError(t, err)
	assert.Len(t, list, 1)
}

// --- Delete ---

func TestRotaInternaService_Delete(t *testing.T) {
	st := mocks.NewMockRotaInternaStore(t)
	st.EXPECT().Delete(mock.Anything, int64(1)).Return(nil)

	svc := rotasinternas.NewRotaInternaService(st)
	err := svc.Delete(svcCtx, 1)

	assert.NoError(t, err)
}
