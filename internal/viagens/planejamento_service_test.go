package viagens_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
	"github.com/fredsaggio/bondrota-api/internal/motoristas"
	"github.com/fredsaggio/bondrota-api/internal/veiculos"
	"github.com/fredsaggio/bondrota-api/internal/viagens"
)

var planejamentoLocation = time.FixedZone("America/Maceio", -3*60*60)

type fakeCicloViagemStore struct {
	createCicloFn       func(context.Context, viagens.CicloViagemInput) (*viagens.CicloViagem, error)
	createIdaFn         func(context.Context, []viagens.CicloIdaComReservasInput, time.Time) (*viagens.PlanejamentoViagens, error)
	createVoltaFn       func(context.Context, []viagens.CicloVoltaComReservasInput, time.Time) (*viagens.PlanejamentoViagens, error)
	listReservasFn      func(context.Context, viagens.PlanejamentoReservasFiltro) ([]viagens.PlanejamentoReserva, error)
	listReservasVoltaFn func(context.Context, viagens.PlanejamentoReservasFiltro) ([]viagens.PlanejamentoReserva, error)
	listCiclosVoltaFn   func(context.Context, viagens.PlanejamentoReservasFiltro) ([]viagens.CicloPlanejamentoVolta, error)
	getCicloByIDFn      func(context.Context, int64) (*viagens.CicloViagem, error)
	listCiclosFn        func(context.Context) ([]viagens.CicloViagem, error)
	updateCicloFn       func(context.Context, int64, func(*viagens.CicloViagem) (bool, error)) (*viagens.CicloViagem, error)
}

func (s fakeCicloViagemStore) CreateCiclo(ctx context.Context, input viagens.CicloViagemInput) (*viagens.CicloViagem, error) {
	return s.createCicloFn(ctx, input)
}

func (s fakeCicloViagemStore) CreatePlanejamentoIda(ctx context.Context, inputs []viagens.CicloIdaComReservasInput, partida time.Time) (*viagens.PlanejamentoViagens, error) {
	return s.createIdaFn(ctx, inputs, partida)
}

func (s fakeCicloViagemStore) CreatePlanejamentoVolta(ctx context.Context, inputs []viagens.CicloVoltaComReservasInput, partida time.Time) (*viagens.PlanejamentoViagens, error) {
	return s.createVoltaFn(ctx, inputs, partida)
}

func (s fakeCicloViagemStore) ListReservasConfirmadasParaPlanejamento(ctx context.Context, filtro viagens.PlanejamentoReservasFiltro) ([]viagens.PlanejamentoReserva, error) {
	return s.listReservasFn(ctx, filtro)
}

func (s fakeCicloViagemStore) ListReservasElegiveisParaVolta(ctx context.Context, filtro viagens.PlanejamentoReservasFiltro) ([]viagens.PlanejamentoReserva, error) {
	return s.listReservasVoltaFn(ctx, filtro)
}

func (s fakeCicloViagemStore) ListCiclosParaPlanejamentoVolta(ctx context.Context, filtro viagens.PlanejamentoReservasFiltro) ([]viagens.CicloPlanejamentoVolta, error) {
	return s.listCiclosVoltaFn(ctx, filtro)
}

func (s fakeCicloViagemStore) GetCicloByID(ctx context.Context, cicloID int64) (*viagens.CicloViagem, error) {
	return s.getCicloByIDFn(ctx, cicloID)
}

func (s fakeCicloViagemStore) ListCiclos(ctx context.Context) ([]viagens.CicloViagem, error) {
	return s.listCiclosFn(ctx)
}

func (s fakeCicloViagemStore) UpdateCiclo(ctx context.Context, cicloID int64, updateFunc func(*viagens.CicloViagem) (bool, error)) (*viagens.CicloViagem, error) {
	return s.updateCicloFn(ctx, cicloID, updateFunc)
}

type fakeHorarioTurnoViagemStore struct {
	getByMunicipioDestinoTurnoFn func(context.Context, int64, viagens.TurnoViagem) (*viagens.HorarioTurnoViagem, error)
}

func (s fakeHorarioTurnoViagemStore) Create(context.Context, viagens.HorarioTurnoViagemInput) (*viagens.HorarioTurnoViagem, error) {
	return nil, nil
}

func (s fakeHorarioTurnoViagemStore) GetByID(context.Context, int64) (*viagens.HorarioTurnoViagem, error) {
	return nil, nil
}

func (s fakeHorarioTurnoViagemStore) GetByMunicipioDestinoTurno(ctx context.Context, municipioDestinoID int64, turno viagens.TurnoViagem) (*viagens.HorarioTurnoViagem, error) {
	return s.getByMunicipioDestinoTurnoFn(ctx, municipioDestinoID, turno)
}

func (s fakeHorarioTurnoViagemStore) List(context.Context) ([]viagens.HorarioTurnoViagem, error) {
	return nil, nil
}

func (s fakeHorarioTurnoViagemStore) Update(context.Context, int64, func(*viagens.HorarioTurnoViagem) (bool, error)) (*viagens.HorarioTurnoViagem, error) {
	return nil, nil
}

func (s fakeHorarioTurnoViagemStore) Delete(context.Context, int64) error {
	return nil
}

type fakeVeiculoAlocador struct {
	alocarFn func(context.Context, veiculos.AlocarVeiculosInput) (*veiculos.AlocacaoVeiculos, error)
}

func (a fakeVeiculoAlocador) Alocar(ctx context.Context, input veiculos.AlocarVeiculosInput) (*veiculos.AlocacaoVeiculos, error) {
	return a.alocarFn(ctx, input)
}

type fakeMotoristaAlocador struct {
	alocarFn func(context.Context, motoristas.AlocarMotoristasInput) ([]motoristas.Motorista, error)
}

func (a fakeMotoristaAlocador) Alocar(ctx context.Context, input motoristas.AlocarMotoristasInput) ([]motoristas.Motorista, error) {
	return a.alocarFn(ctx, input)
}

func validPlanejamentoInput(sentido viagens.SentidoViagem) viagens.PlanejamentoViagensInput {
	return viagens.PlanejamentoViagensInput{
		DataViagem:         time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
		Turno:              viagens.TurnoNoturno,
		MunicipioDestinoID: 2704302,
		RotaInternaID:      2,
		Sentido:            sentido,
	}
}

func validHorarioStore() fakeHorarioTurnoViagemStore {
	return fakeHorarioTurnoViagemStore{
		getByMunicipioDestinoTurnoFn: func(_ context.Context, municipioDestinoID int64, turno viagens.TurnoViagem) (*viagens.HorarioTurnoViagem, error) {
			if municipioDestinoID != 2704302 || turno != viagens.TurnoNoturno {
				return nil, errors.New("unexpected schedule lookup")
			}
			return &viagens.HorarioTurnoViagem{
				MunicipioDestinoID: municipioDestinoID,
				Turno:              turno,
				HorarioIda:         18 * time.Hour,
				HorarioVolta:       22 * time.Hour,
			}, nil
		},
	}
}

func newPlanejamentoService(store viagens.CicloViagemStore, veiculoAlocador viagens.VeiculoAlocador, motoristaAlocador viagens.MotoristaAlocador) viagens.PlanejamentoService {
	return viagens.NewPlanejamentoService(
		store,
		validHorarioStore(),
		veiculoAlocador,
		motoristaAlocador,
		viagens.PlanejamentoServiceConfig{Location: planejamentoLocation},
	)
}

func TestPlanejamentoService_PlanejarIda(t *testing.T) {
	t.Run("creates only outbound planning", func(t *testing.T) {
		input := validPlanejamentoInput(viagens.SentidoIda)
		store := fakeCicloViagemStore{
			listReservasFn: func(_ context.Context, filtro viagens.PlanejamentoReservasFiltro) ([]viagens.PlanejamentoReserva, error) {
				requirePlanejamentoFiltro(t, filtro, viagens.SentidoIda)
				return []viagens.PlanejamentoReserva{
					{ID: 1, DestinoID: 100},
					{ID: 2, DestinoID: 100},
					{ID: 3, DestinoID: 100},
				}, nil
			},
			createIdaFn: func(_ context.Context, inputs []viagens.CicloIdaComReservasInput, partida time.Time) (*viagens.PlanejamentoViagens, error) {
				if len(inputs) != 1 {
					t.Fatalf("expected 1 cycle, got %d", len(inputs))
				}
				if inputs[0].Ciclo.VeiculoID != 10 || inputs[0].Ciclo.MotoristaID != 20 {
					t.Fatalf("unexpected resources: %+v", inputs[0].Ciclo)
				}
				assertSameIDs(t, inputs[0].ReservaIDs, []int64{1, 2, 3})
				wantExpiresAt := time.Date(2026, 9, 10, 0, 0, 0, 0, planejamentoLocation)
				if !inputs[0].Ciclo.ExpiresAt.Equal(wantExpiresAt) {
					t.Fatalf("expected expires_at %v, got %v", wantExpiresAt, inputs[0].Ciclo.ExpiresAt)
				}
				wantPartida := time.Date(2026, 6, 10, 18, 0, 0, 0, planejamentoLocation)
				if !partida.Equal(wantPartida) {
					t.Fatalf("expected departure %v, got %v", wantPartida, partida)
				}
				ciclo := sampleCicloComViagens()
				ciclo.Viagens = ciclo.Viagens[:1]
				return &viagens.PlanejamentoViagens{Ciclos: []viagens.CicloComViagens{ciclo}}, nil
			},
		}

		svc := newPlanejamentoService(store, fakeVeiculoAlocador{
			alocarFn: func(_ context.Context, got veiculos.AlocarVeiculosInput) (*veiculos.AlocacaoVeiculos, error) {
				if got.QuantidadeAlunos != 3 {
					t.Fatalf("expected 3 students, got %d", got.QuantidadeAlunos)
				}
				return &veiculos.AlocacaoVeiculos{
					Veiculos:        []veiculos.Veiculo{{ID: 10, Capacidade: 7}},
					CapacidadeTotal: 7,
				}, nil
			},
		}, fakeMotoristaAlocador{
			alocarFn: func(_ context.Context, got motoristas.AlocarMotoristasInput) ([]motoristas.Motorista, error) {
				if got.Quantidade != 1 || got.MunicipioTrabalhoID != input.MunicipioDestinoID {
					t.Fatalf("unexpected driver allocation: %+v", got)
				}
				return []motoristas.Motorista{{ID: 20}}, nil
			},
		})

		planejamento, err := svc.Planejar(t.Context(), input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if planejamento.Sentido != viagens.SentidoIda || planejamento.QuantidadeReservas != 3 || planejamento.CapacidadeTotal != 7 {
			t.Fatalf("unexpected planning result: %+v", planejamento)
		}
	})

	t.Run("keeps reservations for the same destination together when possible", func(t *testing.T) {
		store := fakeCicloViagemStore{
			listReservasFn: func(context.Context, viagens.PlanejamentoReservasFiltro) ([]viagens.PlanejamentoReserva, error) {
				return []viagens.PlanejamentoReserva{
					{ID: 1, DestinoID: 100},
					{ID: 2, DestinoID: 200},
					{ID: 3, DestinoID: 100},
					{ID: 4, DestinoID: 200},
				}, nil
			},
			createIdaFn: func(_ context.Context, inputs []viagens.CicloIdaComReservasInput, _ time.Time) (*viagens.PlanejamentoViagens, error) {
				if len(inputs) != 2 {
					t.Fatalf("expected 2 cycles, got %d", len(inputs))
				}
				assertSameIDs(t, inputs[0].ReservaIDs, []int64{1, 3})
				assertSameIDs(t, inputs[1].ReservaIDs, []int64{2, 4})
				return &viagens.PlanejamentoViagens{}, nil
			},
		}
		svc := newPlanejamentoService(store, fakeVeiculoAlocador{
			alocarFn: func(context.Context, veiculos.AlocarVeiculosInput) (*veiculos.AlocacaoVeiculos, error) {
				return &veiculos.AlocacaoVeiculos{
					Veiculos:        []veiculos.Veiculo{{ID: 10, Capacidade: 2}, {ID: 11, Capacidade: 2}},
					CapacidadeTotal: 4,
				}, nil
			},
		}, fakeMotoristaAlocador{
			alocarFn: func(context.Context, motoristas.AlocarMotoristasInput) ([]motoristas.Motorista, error) {
				return []motoristas.Motorista{{ID: 20}, {ID: 21}}, nil
			},
		})

		_, err := svc.Planejar(t.Context(), validPlanejamentoInput(viagens.SentidoIda))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns persistence error", func(t *testing.T) {
		store := fakeCicloViagemStore{
			listReservasFn: func(context.Context, viagens.PlanejamentoReservasFiltro) ([]viagens.PlanejamentoReserva, error) {
				return []viagens.PlanejamentoReserva{{ID: 1, DestinoID: 100}}, nil
			},
			createIdaFn: func(context.Context, []viagens.CicloIdaComReservasInput, time.Time) (*viagens.PlanejamentoViagens, error) {
				return nil, brerror.ErrAlreadyExists
			},
		}
		svc := newPlanejamentoService(store, fakeVeiculoAlocador{
			alocarFn: func(context.Context, veiculos.AlocarVeiculosInput) (*veiculos.AlocacaoVeiculos, error) {
				return &veiculos.AlocacaoVeiculos{Veiculos: []veiculos.Veiculo{{ID: 10, Capacidade: 7}}, CapacidadeTotal: 7}, nil
			},
		}, fakeMotoristaAlocador{
			alocarFn: func(context.Context, motoristas.AlocarMotoristasInput) ([]motoristas.Motorista, error) {
				return []motoristas.Motorista{{ID: 20}}, nil
			},
		})

		_, err := svc.Planejar(t.Context(), validPlanejamentoInput(viagens.SentidoIda))
		if !errors.Is(err, brerror.ErrAlreadyExists) {
			t.Fatalf("expected already exists, got %v", err)
		}
	})
}

func TestPlanejamentoService_PlanejarVolta(t *testing.T) {
	ciclo1 := sampleCiclo()
	ciclo1.ID = 1
	ciclo1.VeiculoID = 10
	ciclo2 := sampleCiclo()
	ciclo2.ID = 2
	ciclo2.VeiculoID = 11

	t.Run("reuses outbound cycles and only eligible reservations", func(t *testing.T) {
		store := fakeCicloViagemStore{
			listCiclosVoltaFn: func(_ context.Context, filtro viagens.PlanejamentoReservasFiltro) ([]viagens.CicloPlanejamentoVolta, error) {
				requirePlanejamentoFiltro(t, filtro, viagens.SentidoVolta)
				return []viagens.CicloPlanejamentoVolta{{Ciclo: ciclo1, Capacidade: 2}, {Ciclo: ciclo2, Capacidade: 2}}, nil
			},
			listReservasVoltaFn: func(_ context.Context, filtro viagens.PlanejamentoReservasFiltro) ([]viagens.PlanejamentoReserva, error) {
				requirePlanejamentoFiltro(t, filtro, viagens.SentidoVolta)
				return []viagens.PlanejamentoReserva{{ID: 30, DestinoID: 100}, {ID: 31, DestinoID: 100}, {ID: 32, DestinoID: 200}}, nil
			},
			createVoltaFn: func(_ context.Context, inputs []viagens.CicloVoltaComReservasInput, partida time.Time) (*viagens.PlanejamentoViagens, error) {
				if len(inputs) != 2 || inputs[0].Ciclo.ID != ciclo1.ID || inputs[1].Ciclo.ID != ciclo2.ID {
					t.Fatalf("unexpected outbound cycles: %+v", inputs)
				}
				assertSameIDs(t, inputs[0].ReservaIDs, []int64{30, 31})
				assertSameIDs(t, inputs[1].ReservaIDs, []int64{32})
				wantPartida := time.Date(2026, 6, 10, 22, 0, 0, 0, planejamentoLocation)
				if !partida.Equal(wantPartida) {
					t.Fatalf("expected departure %v, got %v", wantPartida, partida)
				}
				return &viagens.PlanejamentoViagens{Ciclos: []viagens.CicloComViagens{{Ciclo: ciclo1}, {Ciclo: ciclo2}}}, nil
			},
		}
		svc := newPlanejamentoService(store, forbiddenVeiculoAlocador(t), forbiddenMotoristaAlocador(t))

		planejamento, err := svc.Planejar(t.Context(), validPlanejamentoInput(viagens.SentidoVolta))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if planejamento.Sentido != viagens.SentidoVolta || planejamento.QuantidadeReservas != 3 || planejamento.CapacidadeTotal != 4 {
			t.Fatalf("unexpected planning result: %+v", planejamento)
		}
	})

	t.Run("creates return trips even without eligible passengers", func(t *testing.T) {
		store := fakeCicloViagemStore{
			listCiclosVoltaFn: func(context.Context, viagens.PlanejamentoReservasFiltro) ([]viagens.CicloPlanejamentoVolta, error) {
				return []viagens.CicloPlanejamentoVolta{{Ciclo: ciclo1, Capacidade: 2}, {Ciclo: ciclo2, Capacidade: 2}}, nil
			},
			listReservasVoltaFn: func(context.Context, viagens.PlanejamentoReservasFiltro) ([]viagens.PlanejamentoReserva, error) {
				return []viagens.PlanejamentoReserva{}, nil
			},
			createVoltaFn: func(_ context.Context, inputs []viagens.CicloVoltaComReservasInput, _ time.Time) (*viagens.PlanejamentoViagens, error) {
				if len(inputs) != 2 || len(inputs[0].ReservaIDs) != 0 || len(inputs[1].ReservaIDs) != 0 {
					t.Fatalf("expected two empty return trips, got %+v", inputs)
				}
				return &viagens.PlanejamentoViagens{Ciclos: []viagens.CicloComViagens{{Ciclo: ciclo1}, {Ciclo: ciclo2}}}, nil
			},
		}
		svc := newPlanejamentoService(store, forbiddenVeiculoAlocador(t), forbiddenMotoristaAlocador(t))

		planejamento, err := svc.Planejar(t.Context(), validPlanejamentoInput(viagens.SentidoVolta))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if planejamento.QuantidadeReservas != 0 || len(planejamento.Ciclos) != 2 {
			t.Fatalf("unexpected planning result: %+v", planejamento)
		}
	})

	t.Run("requires outbound cycles", func(t *testing.T) {
		store := fakeCicloViagemStore{
			listCiclosVoltaFn: func(context.Context, viagens.PlanejamentoReservasFiltro) ([]viagens.CicloPlanejamentoVolta, error) {
				return []viagens.CicloPlanejamentoVolta{}, nil
			},
			listReservasVoltaFn: func(context.Context, viagens.PlanejamentoReservasFiltro) ([]viagens.PlanejamentoReserva, error) {
				t.Fatal("return reservations should not be queried")
				return nil, nil
			},
		}
		svc := newPlanejamentoService(store, forbiddenVeiculoAlocador(t), forbiddenMotoristaAlocador(t))

		_, err := svc.Planejar(t.Context(), validPlanejamentoInput(viagens.SentidoVolta))
		if !errors.Is(err, brerror.ErrNotFound) {
			t.Fatalf("expected not found, got %v", err)
		}
	})
}

func TestPlanejamentoService_RejectsInvalidInput(t *testing.T) {
	store := fakeCicloViagemStore{
		listReservasFn: func(context.Context, viagens.PlanejamentoReservasFiltro) ([]viagens.PlanejamentoReserva, error) {
			t.Fatal("store should not be called")
			return nil, nil
		},
	}
	svc := newPlanejamentoService(store, fakeVeiculoAlocador{}, fakeMotoristaAlocador{})

	_, err := svc.Planejar(t.Context(), viagens.PlanejamentoViagensInput{})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func requirePlanejamentoFiltro(t *testing.T, got viagens.PlanejamentoReservasFiltro, sentido viagens.SentidoViagem) {
	t.Helper()
	if got.DataViagem.IsZero() || got.Turno != viagens.TurnoNoturno || got.MunicipioDestinoID != 2704302 || got.RotaInternaID != 2 || got.Sentido != sentido {
		t.Fatalf("unexpected filter: %+v", got)
	}
}

func forbiddenVeiculoAlocador(t *testing.T) fakeVeiculoAlocador {
	t.Helper()
	return fakeVeiculoAlocador{alocarFn: func(context.Context, veiculos.AlocarVeiculosInput) (*veiculos.AlocacaoVeiculos, error) {
		t.Fatal("vehicle allocator must not run for return planning")
		return nil, nil
	}}
}

func forbiddenMotoristaAlocador(t *testing.T) fakeMotoristaAlocador {
	t.Helper()
	return fakeMotoristaAlocador{alocarFn: func(context.Context, motoristas.AlocarMotoristasInput) ([]motoristas.Motorista, error) {
		t.Fatal("driver allocator must not run for return planning")
		return nil, nil
	}}
}

func assertSameIDs(t *testing.T, got, want []int64) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected %d ids, got %d: %+v", len(want), len(got), got)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("unexpected id %d: want %d, got %d", i, want[i], got[i])
		}
	}
}
