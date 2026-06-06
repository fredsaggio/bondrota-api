package clientes_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/fredsaggio/bondrota-api/internal/clientes"
	"github.com/fredsaggio/bondrota-api/internal/mocks"
)

func TestVinculoService_CreateValidation(t *testing.T) {
	validInput := clientes.VinculoInput{
		ClienteID:     1,
		Tipo:          clientes.TipoEstudante,
		Turno:         clientes.TurnoNoturno,
		DestinoID:     2,
		RotaInternaID: 3,
		Curso:         "Computacao",
		HorariosFixos: []clientes.DiaSemana{clientes.Segunda, clientes.Quarta},
	}

	tests := []struct {
		name    string
		input   clientes.VinculoInput
		setup   func(*mocks.MockVinculoStore)
		wantErr error
	}{
		{
			name:  "valid estudante",
			input: validInput,
			setup: func(store *mocks.MockVinculoStore) {
				store.EXPECT().Create(mock.Anything, validInput).Return(sampleVinculo(), nil)
			},
		},
		{
			name: "estudante requires curso",
			input: func() clientes.VinculoInput {
				in := validInput
				in.Curso = ""
				return in
			}(),
			setup:   func(_ *mocks.MockVinculoStore) {},
			wantErr: clientes.ErrCursoObrigatorio,
		},
		{
			name: "invalid turno",
			input: func() clientes.VinculoInput {
				in := validInput
				in.Turno = "XX"
				return in
			}(),
			setup:   func(_ *mocks.MockVinculoStore) {},
			wantErr: clientes.ErrTurnoInvalido,
		},
		{
			name: "duplicated day",
			input: func() clientes.VinculoInput {
				in := validInput
				in.HorariosFixos = []clientes.DiaSemana{clientes.Segunda, clientes.Segunda}
				return in
			}(),
			setup:   func(_ *mocks.MockVinculoStore) {},
			wantErr: clientes.ErrDiaDuplicado,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := mocks.NewMockVinculoStore(t)
			tc.setup(store)
			svc := clientes.NewVinculoService(store)

			_, err := svc.Create(context.Background(), tc.input)

			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestVinculoService_UpdateValidation(t *testing.T) {
	store := mocks.NewMockVinculoStore(t)
	svc := clientes.NewVinculoService(store)

	input := clientes.VinculoUpdateInput{
		Tipo:          clientes.TipoEstagio,
		Turno:         clientes.TurnoMatutino,
		DestinoID:     2,
		RotaInternaID: 3,
		HorariosFixos: []clientes.DiaSemana{clientes.Terca},
	}

	store.EXPECT().Update(mock.Anything, int64(10), input).Return(sampleVinculo(), nil)

	_, err := svc.Update(context.Background(), 10, input)

	assert.NoError(t, err)
}
