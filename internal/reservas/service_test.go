package reservas_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/fredsaggio/bondrota-api/internal/mocks"
	"github.com/fredsaggio/bondrota-api/internal/reservas"
)

var ctx = context.Background()

func baseInput() reservas.ReservaInput {
	return reservas.ReservaInput{
		ClienteID:  10,
		VinculoID:  20,
		DataViagem: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		Sentido:    reservas.SentidoIda,
	}
}

func snapshotFor(clienteID int64, turno reservas.TurnoReserva) reservas.VinculoSnapshot {
	return reservas.VinculoSnapshot{
		ClienteID:     clienteID,
		Turno:         turno,
		DestinoID:     5,
		RotaInternaID: 3,
		Cidade:        "Recife",
	}
}

// --- Create ---

func TestService_Create(t *testing.T) {
	tests := []struct {
		name      string
		input     func() reservas.ReservaInput
		setup     func(*mocks.MockReservaStore)
		wantErr   error
		wantTurno reservas.TurnoReserva
	}{
		{
			name:  "sucesso — vinculo com turno fixo",
			input: baseInput,
			setup: func(st *mocks.MockReservaStore) {
				st.EXPECT().GetVinculoSnapshot(mock.Anything, int64(20)).Return(snapshotFor(10, reservas.TurnoMatutino), nil)
				st.EXPECT().Create(mock.Anything, mock.MatchedBy(func(in reservas.ReservaInput) bool {
					return in.Turno == reservas.TurnoMatutino && in.Cidade == "Recife"
				})).Return(sampleReserva(), nil)
			},
			wantTurno: reservas.TurnoMatutino,
		},
		{
			name: "sucesso — vinculo integral com turno informado",
			input: func() reservas.ReservaInput {
				in := baseInput()
				in.Turno = reservas.TurnoVespertino
				return in
			},
			setup: func(st *mocks.MockReservaStore) {
				st.EXPECT().GetVinculoSnapshot(mock.Anything, int64(20)).Return(snapshotFor(10, reservas.TurnoIntegral), nil)
				st.EXPECT().Create(mock.Anything, mock.MatchedBy(func(in reservas.ReservaInput) bool {
					return in.Turno == reservas.TurnoVespertino
				})).Return(sampleReserva(), nil)
			},
			wantTurno: reservas.TurnoVespertino,
		},
		{
			name: "vinculo integral sem turno → ErrTurnoObrigatorio",
			input: func() reservas.ReservaInput {
				in := baseInput()
				in.Turno = ""
				return in
			},
			setup: func(st *mocks.MockReservaStore) {
				st.EXPECT().GetVinculoSnapshot(mock.Anything, int64(20)).Return(snapshotFor(10, reservas.TurnoIntegral), nil)
			},
			wantErr: reservas.ErrTurnoObrigatorio,
		},
		{
			name: "turno incompatível com vinculo → ErrTurnoIncompativel",
			input: func() reservas.ReservaInput {
				in := baseInput()
				in.Turno = reservas.TurnoNoturno
				return in
			},
			setup: func(st *mocks.MockReservaStore) {
				st.EXPECT().GetVinculoSnapshot(mock.Anything, int64(20)).Return(snapshotFor(10, reservas.TurnoMatutino), nil)
			},
			wantErr: reservas.ErrTurnoIncompativel,
		},
		{
			name: "clienteID não pertence ao vínculo → ErrVinculoNotFound",
			input: func() reservas.ReservaInput {
				in := baseInput()
				in.ClienteID = 99 // clienteID diferente do snapshot
				return in
			},
			setup: func(st *mocks.MockReservaStore) {
				st.EXPECT().GetVinculoSnapshot(mock.Anything, int64(20)).Return(snapshotFor(10, reservas.TurnoMatutino), nil)
			},
			wantErr: reservas.ErrVinculoNotFound,
		},
		{
			name: "vinculo_id ausente → ErrVinculoIDObrigatorio",
			input: func() reservas.ReservaInput {
				in := baseInput()
				in.VinculoID = 0
				return in
			},
			setup:   func(_ *mocks.MockReservaStore) {},
			wantErr: reservas.ErrVinculoIDObrigatorio,
		},
		{
			name: "data_viagem zero → ErrDataObrigatoria",
			input: func() reservas.ReservaInput {
				in := baseInput()
				in.DataViagem = time.Time{}
				return in
			},
			setup:   func(_ *mocks.MockReservaStore) {},
			wantErr: reservas.ErrDataObrigatoria,
		},
		{
			name: "sentido inválido → ErrSentidoInvalido",
			input: func() reservas.ReservaInput {
				in := baseInput()
				in.Sentido = "nenhum"
				return in
			},
			setup:   func(_ *mocks.MockReservaStore) {},
			wantErr: reservas.ErrSentidoInvalido,
		},
		{
			name:  "erro do store ao buscar snapshot",
			input: baseInput,
			setup: func(st *mocks.MockReservaStore) {
				st.EXPECT().GetVinculoSnapshot(mock.Anything, int64(20)).Return(reservas.VinculoSnapshot{}, errors.New("db err"))
			},
			// erro genérico não tem sentinel, só verificamos que houve erro
			wantErr: nil, // handled via assert.Error below
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := mocks.NewMockReservaStore(t)
			tc.setup(store)

			svc := reservas.NewReservaService(store)
			_, err := svc.Create(ctx, tc.input())

			switch {
			case tc.name == "erro do store ao buscar snapshot":
				assert.Error(t, err)
			case tc.wantErr != nil:
				assert.ErrorIs(t, err, tc.wantErr)
			default:
				assert.NoError(t, err)
			}
		})
	}
}

// --- GetByID ---

func TestService_GetByID(t *testing.T) {
	tests := []struct {
		name      string
		reservaID int64
		setup     func(*mocks.MockReservaStore)
		wantErr   error
	}{
		{
			name:      "sucesso",
			reservaID: 1,
			setup: func(st *mocks.MockReservaStore) {
				st.EXPECT().GetByID(mock.Anything, int64(1)).Return(sampleReserva(), nil)
			},
		},
		{
			name:      "não encontrado",
			reservaID: 99,
			setup: func(st *mocks.MockReservaStore) {
				st.EXPECT().GetByID(mock.Anything, int64(99)).Return(nil, reservas.ErrReservaNotFound)
			},
			wantErr: reservas.ErrReservaNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := mocks.NewMockReservaStore(t)
			tc.setup(store)
			svc := reservas.NewReservaService(store)
			_, err := svc.GetByID(ctx, tc.reservaID)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// --- ListByVinculo ---

func TestService_ListByVinculo(t *testing.T) {
	tests := []struct {
		name      string
		clienteID int64
		vinculoID int64
		setup     func(*mocks.MockReservaStore)
		wantErr   error
	}{
		{
			name:      "sucesso",
			clienteID: 10, vinculoID: 20,
			setup: func(st *mocks.MockReservaStore) {
				st.EXPECT().GetVinculoSnapshot(mock.Anything, int64(20)).Return(snapshotFor(10, reservas.TurnoMatutino), nil)
				st.EXPECT().ListByVinculo(mock.Anything, int64(10), int64(20)).Return([]reservas.Reserva{*sampleReserva()}, nil)
			},
		},
		{
			name:      "clienteID diverge → ErrVinculoNotFound",
			clienteID: 99, vinculoID: 20,
			setup: func(st *mocks.MockReservaStore) {
				st.EXPECT().GetVinculoSnapshot(mock.Anything, int64(20)).Return(snapshotFor(10, reservas.TurnoMatutino), nil)
			},
			wantErr: reservas.ErrVinculoNotFound,
		},
		{
			name:      "snapshot não existe → ErrVinculoNotFound",
			clienteID: 10, vinculoID: 99,
			setup: func(st *mocks.MockReservaStore) {
				st.EXPECT().GetVinculoSnapshot(mock.Anything, int64(99)).Return(reservas.VinculoSnapshot{}, reservas.ErrVinculoNotFound)
			},
			wantErr: reservas.ErrVinculoNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := mocks.NewMockReservaStore(t)
			tc.setup(store)
			svc := reservas.NewReservaService(store)
			_, err := svc.ListByVinculo(ctx, tc.clienteID, tc.vinculoID)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// --- Cancel ---

func TestService_Cancel(t *testing.T) {
	tests := []struct {
		name      string
		reservaID int64
		setup     func(*mocks.MockReservaStore)
		wantErr   error
	}{
		{
			name:      "sucesso — muda status para cancelada",
			reservaID: 1,
			setup: func(st *mocks.MockReservaStore) {
				st.EXPECT().Update(mock.Anything, int64(1), mock.Anything).RunAndReturn(
					func(_ context.Context, _ int64, fn func(*reservas.Reserva) (bool, error)) (*reservas.Reserva, error) {
						r := sampleReserva()
						changed, err := fn(r)
						if err != nil {
							return nil, err
						}
						if changed {
							r.Status = reservas.StatusCancelada
						}
						return r, nil
					},
				)
			},
		},
		{
			name:      "já cancelada — nenhuma mudança",
			reservaID: 1,
			setup: func(st *mocks.MockReservaStore) {
				st.EXPECT().Update(mock.Anything, int64(1), mock.Anything).RunAndReturn(
					func(_ context.Context, _ int64, fn func(*reservas.Reserva) (bool, error)) (*reservas.Reserva, error) {
						r := sampleReserva()
						r.Status = reservas.StatusCancelada
						_, err := fn(r)
						return r, err
					},
				)
			},
		},
		{
			name:      "não encontrado → ErrReservaNotFound",
			reservaID: 99,
			setup: func(st *mocks.MockReservaStore) {
				st.EXPECT().Update(mock.Anything, int64(99), mock.Anything).Return(nil, reservas.ErrReservaNotFound)
			},
			wantErr: reservas.ErrReservaNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := mocks.NewMockReservaStore(t)
			tc.setup(store)
			svc := reservas.NewReservaService(store)
			_, err := svc.Cancel(ctx, tc.reservaID)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// --- Delete ---

func TestService_Delete(t *testing.T) {
	tests := []struct {
		name      string
		reservaID int64
		setup     func(*mocks.MockReservaStore)
		wantErr   error
	}{
		{
			name:      "sucesso",
			reservaID: 1,
			setup:     func(st *mocks.MockReservaStore) { st.EXPECT().Delete(mock.Anything, int64(1)).Return(nil) },
		},
		{
			name:      "não encontrado",
			reservaID: 99,
			setup: func(st *mocks.MockReservaStore) {
				st.EXPECT().Delete(mock.Anything, int64(99)).Return(reservas.ErrReservaNotFound)
			},
			wantErr: reservas.ErrReservaNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := mocks.NewMockReservaStore(t)
			tc.setup(store)
			svc := reservas.NewReservaService(store)
			err := svc.Delete(ctx, tc.reservaID)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// --- resolveTurno (via Create) — casos de turno ---

func TestService_ResolveTurno(t *testing.T) {
	tests := []struct {
		name         string
		vinculoTurno reservas.TurnoReserva
		inputTurno   reservas.TurnoReserva
		wantErr      error
	}{
		{"fixo MT sem turno solicitado → usa MT", reservas.TurnoMatutino, "", nil},
		{"fixo MT com turno MT → usa MT", reservas.TurnoMatutino, reservas.TurnoMatutino, nil},
		{"fixo MT com turno VT → incompatível", reservas.TurnoMatutino, reservas.TurnoVespertino, reservas.ErrTurnoIncompativel},
		{"integral com MT → usa MT", reservas.TurnoIntegral, reservas.TurnoMatutino, nil},
		{"integral sem turno → obrigatório", reservas.TurnoIntegral, "", reservas.ErrTurnoObrigatorio},
		// IN é barrado por validateCreateInput antes de chegar ao store, por isso vinculoTurno não importa
		{"turno IN (não-operacional) informado → inválido", reservas.TurnoMatutino, reservas.TurnoIntegral, reservas.ErrTurnoInvalido},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := mocks.NewMockReservaStore(t)

			// turno não-operacional informado no input é barrado antes do store
			inputIsInvalid := tc.inputTurno != "" && tc.inputTurno != reservas.TurnoMatutino &&
				tc.inputTurno != reservas.TurnoVespertino && tc.inputTurno != reservas.TurnoNoturno
			if !inputIsInvalid {
				store.EXPECT().GetVinculoSnapshot(mock.Anything, int64(20)).
					Return(snapshotFor(10, tc.vinculoTurno), nil)
			}
			if tc.wantErr == nil {
				store.EXPECT().Create(mock.Anything, mock.Anything).Return(sampleReserva(), nil)
			}

			svc := reservas.NewReservaService(store)
			in := baseInput()
			in.Turno = tc.inputTurno
			_, err := svc.Create(ctx, in)

			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
