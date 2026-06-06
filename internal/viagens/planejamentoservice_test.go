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

type fakeCicloViagemStore struct {
	createCicloFn            func(ctx context.Context, input viagens.CicloViagemInput) (*viagens.CicloViagem, error)
	createCicloComViagensFn  func(ctx context.Context, input viagens.CicloViagemInput, partidas map[viagens.SentidoViagem]time.Time) (*viagens.CicloComViagens, error)
	createCiclosComViagensFn func(ctx context.Context, inputs []viagens.CicloViagemComReservasInput, partidas map[viagens.SentidoViagem]time.Time) (*viagens.PlanejamentoViagens, error)
	listReservasFn           func(ctx context.Context, filtro viagens.PlanejamentoReservasFiltro) ([]viagens.PlanejamentoReserva, error)
	getCicloByIDFn           func(ctx context.Context, cicloID int64) (*viagens.CicloViagem, error)
	listCiclosFn             func(ctx context.Context) ([]viagens.CicloViagem, error)
	updateCicloFn            func(ctx context.Context, cicloID int64, updateFunc func(*viagens.CicloViagem) (bool, error)) (*viagens.CicloViagem, error)
}

func (s fakeCicloViagemStore) CreateCiclo(ctx context.Context, input viagens.CicloViagemInput) (*viagens.CicloViagem, error) {
	return s.createCicloFn(ctx, input)
}

func (s fakeCicloViagemStore) CreateCicloComViagens(ctx context.Context, input viagens.CicloViagemInput, partidas map[viagens.SentidoViagem]time.Time) (*viagens.CicloComViagens, error) {
	return s.createCicloComViagensFn(ctx, input, partidas)
}

func (s fakeCicloViagemStore) CreateCiclosComViagens(ctx context.Context, inputs []viagens.CicloViagemComReservasInput, partidas map[viagens.SentidoViagem]time.Time) (*viagens.PlanejamentoViagens, error) {
	return s.createCiclosComViagensFn(ctx, inputs, partidas)
}

func (s fakeCicloViagemStore) ListReservasConfirmadasParaPlanejamento(ctx context.Context, filtro viagens.PlanejamentoReservasFiltro) ([]viagens.PlanejamentoReserva, error) {
	return s.listReservasFn(ctx, filtro)
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
	getByCidadeTurnoFn func(ctx context.Context, cidade string, turno viagens.TurnoViagem) (*viagens.HorarioTurnoViagem, error)
}

func (s fakeHorarioTurnoViagemStore) Create(ctx context.Context, input viagens.HorarioTurnoViagemInput) (*viagens.HorarioTurnoViagem, error) {
	return nil, nil
}

func (s fakeHorarioTurnoViagemStore) GetByID(ctx context.Context, id int64) (*viagens.HorarioTurnoViagem, error) {
	return nil, nil
}

func (s fakeHorarioTurnoViagemStore) GetByCidadeTurno(ctx context.Context, cidade string, turno viagens.TurnoViagem) (*viagens.HorarioTurnoViagem, error) {
	return s.getByCidadeTurnoFn(ctx, cidade, turno)
}

func (s fakeHorarioTurnoViagemStore) List(ctx context.Context) ([]viagens.HorarioTurnoViagem, error) {
	return nil, nil
}

func (s fakeHorarioTurnoViagemStore) Update(ctx context.Context, id int64, updateFunc func(*viagens.HorarioTurnoViagem) (bool, error)) (*viagens.HorarioTurnoViagem, error) {
	return nil, nil
}

func (s fakeHorarioTurnoViagemStore) Delete(ctx context.Context, id int64) error {
	return nil
}

type fakeVeiculoAlocador struct {
	alocarFn func(ctx context.Context, input veiculos.AlocarVeiculosInput) (*veiculos.AlocacaoVeiculos, error)
}

func (a fakeVeiculoAlocador) Alocar(ctx context.Context, input veiculos.AlocarVeiculosInput) (*veiculos.AlocacaoVeiculos, error) {
	return a.alocarFn(ctx, input)
}

type fakeMotoristaAlocador struct {
	alocarFn func(ctx context.Context, input motoristas.AlocarMotoristasInput) ([]motoristas.Motorista, error)
}

func (a fakeMotoristaAlocador) Alocar(ctx context.Context, input motoristas.AlocarMotoristasInput) ([]motoristas.Motorista, error) {
	return a.alocarFn(ctx, input)
}

func validPlanejamentoInput() viagens.PlanejamentoViagensInput {
	return viagens.PlanejamentoViagensInput{
		DataViagem:    time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
		Turno:         viagens.TurnoNoturno,
		Cidade:        "Campo Alegre",
		RotaInternaID: 2,
		ExpiresAt:     time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC),
	}
}

func validPartidas() map[viagens.SentidoViagem]time.Time {
	return map[viagens.SentidoViagem]time.Time{
		viagens.SentidoIda:   time.Date(2026, 6, 10, 18, 0, 0, 0, time.UTC),
		viagens.SentidoVolta: time.Date(2026, 6, 10, 22, 0, 0, 0, time.UTC),
	}
}

func validHorarioStore() fakeHorarioTurnoViagemStore {
	return fakeHorarioTurnoViagemStore{
		getByCidadeTurnoFn: func(_ context.Context, cidade string, turno viagens.TurnoViagem) (*viagens.HorarioTurnoViagem, error) {
			if cidade != "Campo Alegre" {
				return nil, errors.New("unexpected cidade")
			}
			if turno != viagens.TurnoNoturno {
				return nil, errors.New("unexpected turno")
			}
			return &viagens.HorarioTurnoViagem{
				Cidade:       cidade,
				Turno:        turno,
				HorarioIda:   18 * time.Hour,
				HorarioVolta: 22 * time.Hour,
			}, nil
		},
	}
}

func TestPlanejamentoService_Planejar(t *testing.T) {
	t.Run("valid input delegates to store", func(t *testing.T) {
		input := validPlanejamentoInput()
		partidas := validPartidas()
		svc := viagens.NewPlanejamentoService(fakeCicloViagemStore{
			listReservasFn: func(_ context.Context, filtro viagens.PlanejamentoReservasFiltro) ([]viagens.PlanejamentoReserva, error) {
				if filtro.Sentido == viagens.SentidoIda {
					return []viagens.PlanejamentoReserva{
						{ID: 1, DestinoID: 100},
						{ID: 2, DestinoID: 100},
						{ID: 3, DestinoID: 100},
					}, nil
				}
				return []viagens.PlanejamentoReserva{{ID: 4, DestinoID: 100}}, nil
			},
			createCiclosComViagensFn: func(_ context.Context, gotInputs []viagens.CicloViagemComReservasInput, gotPartidas map[viagens.SentidoViagem]time.Time) (*viagens.PlanejamentoViagens, error) {
				if len(gotInputs) != 1 {
					t.Fatalf("expected 1 ciclo input, got %d", len(gotInputs))
				}
				if gotInputs[0].Ciclo.VeiculoID != 10 {
					t.Fatalf("unexpected veiculo id: %d", gotInputs[0].Ciclo.VeiculoID)
				}
				if gotInputs[0].Ciclo.MotoristaID != 20 {
					t.Fatalf("unexpected motorista id: %d", gotInputs[0].Ciclo.MotoristaID)
				}
				if len(gotInputs[0].ReservaIDsIda) != 3 {
					t.Fatalf("unexpected ida reservas: %+v", gotInputs[0].ReservaIDsIda)
				}
				if !gotPartidas[viagens.SentidoVolta].Equal(partidas[viagens.SentidoVolta]) {
					t.Fatalf("unexpected partidas: %+v", gotPartidas)
				}
				ciclo := sampleCicloComViagens()
				return &viagens.PlanejamentoViagens{Ciclos: []viagens.CicloComViagens{ciclo}}, nil
			},
		}, validHorarioStore(), fakeVeiculoAlocador{
			alocarFn: func(_ context.Context, gotInput veiculos.AlocarVeiculosInput) (*veiculos.AlocacaoVeiculos, error) {
				if gotInput.QuantidadeAlunos != 3 {
					t.Fatalf("unexpected quantidade alunos: %d", gotInput.QuantidadeAlunos)
				}
				return &veiculos.AlocacaoVeiculos{
					Veiculos:        []veiculos.Veiculo{{ID: 10, Capacidade: 7, Categoria: veiculos.CategoriaCarroSeteLugares}},
					CapacidadeTotal: 7,
				}, nil
			},
		}, fakeMotoristaAlocador{
			alocarFn: func(_ context.Context, gotInput motoristas.AlocarMotoristasInput) ([]motoristas.Motorista, error) {
				if gotInput.Quantidade != 1 {
					t.Fatalf("unexpected quantidade motoristas: %d", gotInput.Quantidade)
				}
				return []motoristas.Motorista{{ID: 20}}, nil
			},
		})

		planejamento, err := svc.Planejar(context.Background(), input)

		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if len(planejamento.Ciclos) != 1 {
			t.Fatalf("expected 1 ciclo, got %d", len(planejamento.Ciclos))
		}
		if planejamento.QuantidadeReservasIda != 3 {
			t.Fatalf("expected 3 ida reservas, got %d", planejamento.QuantidadeReservasIda)
		}
	})

	t.Run("keeps same destino reservations together when possible", func(t *testing.T) {
		svc := viagens.NewPlanejamentoService(fakeCicloViagemStore{
			listReservasFn: func(_ context.Context, filtro viagens.PlanejamentoReservasFiltro) ([]viagens.PlanejamentoReserva, error) {
				if filtro.Sentido == viagens.SentidoVolta {
					return nil, nil
				}
				return []viagens.PlanejamentoReserva{
					{ID: 1, DestinoID: 100},
					{ID: 2, DestinoID: 200},
					{ID: 3, DestinoID: 100},
					{ID: 4, DestinoID: 200},
				}, nil
			},
			createCiclosComViagensFn: func(_ context.Context, gotInputs []viagens.CicloViagemComReservasInput, _ map[viagens.SentidoViagem]time.Time) (*viagens.PlanejamentoViagens, error) {
				if len(gotInputs) != 2 {
					t.Fatalf("expected 2 ciclo inputs, got %d", len(gotInputs))
				}

				assertSameIDs(t, gotInputs[0].ReservaIDsIda, []int64{1, 3})
				assertSameIDs(t, gotInputs[1].ReservaIDsIda, []int64{2, 4})

				return &viagens.PlanejamentoViagens{}, nil
			},
		}, validHorarioStore(), fakeVeiculoAlocador{
			alocarFn: func(_ context.Context, _ veiculos.AlocarVeiculosInput) (*veiculos.AlocacaoVeiculos, error) {
				return &veiculos.AlocacaoVeiculos{
					Veiculos: []veiculos.Veiculo{
						{ID: 10, Capacidade: 2, Categoria: veiculos.CategoriaCarroSeteLugares},
						{ID: 11, Capacidade: 2, Categoria: veiculos.CategoriaCarroSeteLugares},
					},
					CapacidadeTotal: 4,
				}, nil
			},
		}, fakeMotoristaAlocador{
			alocarFn: func(_ context.Context, _ motoristas.AlocarMotoristasInput) ([]motoristas.Motorista, error) {
				return []motoristas.Motorista{{ID: 20}, {ID: 21}}, nil
			},
		})

		_, err := svc.Planejar(context.Background(), validPlanejamentoInput())

		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
	})

	t.Run("invalid input does not call store", func(t *testing.T) {
		svc := viagens.NewPlanejamentoService(fakeCicloViagemStore{
			listReservasFn: func(_ context.Context, _ viagens.PlanejamentoReservasFiltro) ([]viagens.PlanejamentoReserva, error) {
				t.Fatal("store should not be called")
				return nil, nil
			},
		}, validHorarioStore(), fakeVeiculoAlocador{}, fakeMotoristaAlocador{})

		_, err := svc.Planejar(context.Background(), viagens.PlanejamentoViagensInput{})

		if err == nil {
			t.Fatal("expected validation error")
		}
	})

	t.Run("store error is returned", func(t *testing.T) {
		svc := viagens.NewPlanejamentoService(fakeCicloViagemStore{
			listReservasFn: func(_ context.Context, filtro viagens.PlanejamentoReservasFiltro) ([]viagens.PlanejamentoReserva, error) {
				if filtro.Sentido == viagens.SentidoIda {
					return []viagens.PlanejamentoReserva{{ID: 1, DestinoID: 100}}, nil
				}
				return nil, nil
			},
			createCiclosComViagensFn: func(_ context.Context, _ []viagens.CicloViagemComReservasInput, _ map[viagens.SentidoViagem]time.Time) (*viagens.PlanejamentoViagens, error) {
				return nil, brerror.ErrAlreadyExists
			},
		}, validHorarioStore(), fakeVeiculoAlocador{
			alocarFn: func(_ context.Context, _ veiculos.AlocarVeiculosInput) (*veiculos.AlocacaoVeiculos, error) {
				return &veiculos.AlocacaoVeiculos{
					Veiculos:        []veiculos.Veiculo{{ID: 10, Capacidade: 7, Categoria: veiculos.CategoriaCarroSeteLugares}},
					CapacidadeTotal: 7,
				}, nil
			},
		}, fakeMotoristaAlocador{
			alocarFn: func(_ context.Context, _ motoristas.AlocarMotoristasInput) ([]motoristas.Motorista, error) {
				return []motoristas.Motorista{{ID: 20}}, nil
			},
		})

		_, err := svc.Planejar(context.Background(), validPlanejamentoInput())

		if !errors.Is(err, brerror.ErrAlreadyExists) {
			t.Fatalf("expected already exists, got %v", err)
		}
	})
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
