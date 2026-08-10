package viagens_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
	"github.com/fredsaggio/bondrota-api/internal/viagens"
)

type fakeAgendadorPlanejamentoStore struct {
	listFn func(context.Context, time.Time, time.Time) ([]viagens.CandidatoPlanejamento, error)
}

func (s fakeAgendadorPlanejamentoStore) ListCandidatos(ctx context.Context, dataInicio, dataFim time.Time) ([]viagens.CandidatoPlanejamento, error) {
	return s.listFn(ctx, dataInicio, dataFim)
}

type fakeExecucaoPlanejamentoStore struct {
	tentarIniciarFn func(context.Context, viagens.IniciarExecucaoPlanejamentoInput) (*viagens.ExecucaoPlanejamento, bool, error)
	getByChaveFn    func(context.Context, viagens.ChaveExecucaoPlanejamento) (*viagens.ExecucaoPlanejamento, error)
	finalizarFn     func(context.Context, int64, viagens.StatusExecucaoPlanejamento) (*viagens.ExecucaoPlanejamento, error)
	falharFn        func(context.Context, viagens.FalharExecucaoPlanejamentoInput) (*viagens.ExecucaoPlanejamento, error)
}

func (s fakeExecucaoPlanejamentoStore) TentarIniciar(ctx context.Context, input viagens.IniciarExecucaoPlanejamentoInput) (*viagens.ExecucaoPlanejamento, bool, error) {
	return s.tentarIniciarFn(ctx, input)
}

func (s fakeExecucaoPlanejamentoStore) GetByChave(ctx context.Context, chave viagens.ChaveExecucaoPlanejamento) (*viagens.ExecucaoPlanejamento, error) {
	return s.getByChaveFn(ctx, chave)
}

func (s fakeExecucaoPlanejamentoStore) Finalizar(ctx context.Context, execucaoID int64, resultado viagens.StatusExecucaoPlanejamento) (*viagens.ExecucaoPlanejamento, error) {
	return s.finalizarFn(ctx, execucaoID, resultado)
}

func (s fakeExecucaoPlanejamentoStore) Falhar(ctx context.Context, input viagens.FalharExecucaoPlanejamentoInput) (*viagens.ExecucaoPlanejamento, error) {
	return s.falharFn(ctx, input)
}

func TestProcessadorPlanejamento_Processar(t *testing.T) {
	location := time.FixedZone("America/Maceio", -3*60*60)
	agora := time.Date(2026, time.August, 12, 21, 30, 0, 0, location)
	candidatos := []viagens.CandidatoPlanejamento{
		novoCandidatoPlanejamento(agora, 1, viagens.SentidoIda, 22*time.Hour),
		novoCandidatoPlanejamento(agora, 2, viagens.SentidoIda, 22*time.Hour),
		novoCandidatoPlanejamento(agora, 3, viagens.SentidoVolta, 22*time.Hour),
	}

	finalizados := make(map[int64]viagens.StatusExecucaoPlanejamento)
	processador := viagens.NewProcessadorPlanejamento(
		fakeAgendadorPlanejamentoStore{
			listFn: func(_ context.Context, dataInicio, dataFim time.Time) ([]viagens.CandidatoPlanejamento, error) {
				wantInicio := time.Date(2026, time.August, 12, 0, 0, 0, 0, location)
				wantFim := wantInicio.AddDate(0, 0, 1)
				if !dataInicio.Equal(wantInicio) || !dataFim.Equal(wantFim) {
					t.Fatalf("unexpected candidate range: %v - %v", dataInicio, dataFim)
				}
				return candidatos, nil
			},
		},
		fakeExecucaoPlanejamentoStore{
			tentarIniciarFn: func(_ context.Context, input viagens.IniciarExecucaoPlanejamentoInput) (*viagens.ExecucaoPlanejamento, bool, error) {
				wantPartida := time.Date(2026, time.August, 12, 22, 0, 0, 0, location)
				if !input.PartidaEm.Equal(wantPartida) || !input.FechamentoEm.Equal(agora) {
					t.Fatalf("unexpected planning window: %+v", input)
				}
				if !input.BloqueioExpiraEm.Equal(agora.Add(5 * time.Minute)) {
					t.Fatalf("unexpected lock expiration: %v", input.BloqueioExpiraEm)
				}
				return &viagens.ExecucaoPlanejamento{ID: input.Chave.RotaInternaID}, true, nil
			},
			finalizarFn: func(_ context.Context, execucaoID int64, resultado viagens.StatusExecucaoPlanejamento) (*viagens.ExecucaoPlanejamento, error) {
				finalizados[execucaoID] = resultado
				return &viagens.ExecucaoPlanejamento{ID: execucaoID, Status: resultado}, nil
			},
			falharFn: func(context.Context, viagens.FalharExecucaoPlanejamentoInput) (*viagens.ExecucaoPlanejamento, error) {
				t.Fatal("no execution should fail")
				return nil, nil
			},
		},
		fakePlanejamentoService{
			planejarFn: func(_ context.Context, input viagens.PlanejamentoViagensInput) (*viagens.PlanejamentoViagens, error) {
				switch input.RotaInternaID {
				case 1:
					return &viagens.PlanejamentoViagens{}, nil
				case 2:
					return nil, viagens.ErrSemDemandaPlanejamento
				case 3:
					return nil, brerror.ErrAlreadyExists
				default:
					t.Fatalf("unexpected planning input: %+v", input)
					return nil, nil
				}
			},
		},
		viagens.ProcessadorPlanejamentoConfig{Location: location, Now: func() time.Time { return agora }},
	)

	resumo, err := processador.Processar(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resumo.Candidatos != 3 || resumo.Devidos != 3 || resumo.Adquiridos != 3 || resumo.Concluidos != 2 || resumo.SemDemanda != 1 || resumo.Falhos != 0 {
		t.Fatalf("unexpected summary: %+v", resumo)
	}
	if finalizados[1] != viagens.StatusExecucaoConcluido || finalizados[2] != viagens.StatusExecucaoSemDemanda || finalizados[3] != viagens.StatusExecucaoConcluido {
		t.Fatalf("unexpected final statuses: %+v", finalizados)
	}
}

func TestProcessadorPlanejamento_IgnoresCandidatesOutsideWindowOrAlreadyClaimed(t *testing.T) {
	location := time.FixedZone("America/Maceio", -3*60*60)
	agora := time.Date(2026, time.August, 12, 21, 30, 0, 0, location)
	candidatos := []viagens.CandidatoPlanejamento{
		novoCandidatoPlanejamento(agora, 1, viagens.SentidoIda, 23*time.Hour),
		novoCandidatoPlanejamento(agora, 2, viagens.SentidoIda, 21*time.Hour),
		novoCandidatoPlanejamento(agora, 3, viagens.SentidoVolta, 22*time.Hour),
	}
	claims := 0
	processador := viagens.NewProcessadorPlanejamento(
		fakeAgendadorPlanejamentoStore{listFn: func(context.Context, time.Time, time.Time) ([]viagens.CandidatoPlanejamento, error) {
			return candidatos, nil
		}},
		fakeExecucaoPlanejamentoStore{
			tentarIniciarFn: func(context.Context, viagens.IniciarExecucaoPlanejamentoInput) (*viagens.ExecucaoPlanejamento, bool, error) {
				claims++
				return nil, false, nil
			},
		},
		fakePlanejamentoService{planejarFn: func(context.Context, viagens.PlanejamentoViagensInput) (*viagens.PlanejamentoViagens, error) {
			t.Fatal("planner should not be called")
			return nil, nil
		}},
		viagens.ProcessadorPlanejamentoConfig{Location: location, Now: func() time.Time { return agora }},
	)

	resumo, err := processador.Processar(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claims != 1 || resumo.Candidatos != 3 || resumo.Devidos != 1 || resumo.Adquiridos != 0 {
		t.Fatalf("unexpected processing: claims=%d summary=%+v", claims, resumo)
	}
}

func TestProcessadorPlanejamento_ProcessesNextDayDepartureAtPreviousDayDeadline(t *testing.T) {
	location := time.FixedZone("America/Maceio", -3*60*60)
	agora := time.Date(2026, time.August, 12, 23, 45, 0, 0, location)
	dataViagem := agora.AddDate(0, 0, 1)
	processador := viagens.NewProcessadorPlanejamento(
		fakeAgendadorPlanejamentoStore{listFn: func(context.Context, time.Time, time.Time) ([]viagens.CandidatoPlanejamento, error) {
			return []viagens.CandidatoPlanejamento{novoCandidatoPlanejamento(dataViagem, 1, viagens.SentidoIda, 15*time.Minute)}, nil
		}},
		fakeExecucaoPlanejamentoStore{
			tentarIniciarFn: func(_ context.Context, input viagens.IniciarExecucaoPlanejamentoInput) (*viagens.ExecucaoPlanejamento, bool, error) {
				wantPartida := time.Date(2026, time.August, 13, 0, 15, 0, 0, location)
				if !input.PartidaEm.Equal(wantPartida) || !input.FechamentoEm.Equal(agora) {
					t.Fatalf("unexpected overnight window: %+v", input)
				}
				return &viagens.ExecucaoPlanejamento{ID: 1}, true, nil
			},
			finalizarFn: func(context.Context, int64, viagens.StatusExecucaoPlanejamento) (*viagens.ExecucaoPlanejamento, error) {
				return &viagens.ExecucaoPlanejamento{}, nil
			},
		},
		fakePlanejamentoService{planejarFn: func(context.Context, viagens.PlanejamentoViagensInput) (*viagens.PlanejamentoViagens, error) {
			return &viagens.PlanejamentoViagens{}, nil
		}},
		viagens.ProcessadorPlanejamentoConfig{Location: location, Now: func() time.Time { return agora }},
	)

	resumo, err := processador.Processar(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resumo.Devidos != 1 || resumo.Concluidos != 1 {
		t.Fatalf("unexpected summary: %+v", resumo)
	}
}

func TestProcessadorPlanejamento_PersistsPlanningFailure(t *testing.T) {
	location := time.FixedZone("America/Maceio", -3*60*60)
	agora := time.Date(2026, time.August, 12, 21, 30, 0, 0, location)
	planningErr := errors.New("vehicles unavailable")
	failureMessage := ""
	var proximaTentativa time.Time
	processador := viagens.NewProcessadorPlanejamento(
		fakeAgendadorPlanejamentoStore{listFn: func(context.Context, time.Time, time.Time) ([]viagens.CandidatoPlanejamento, error) {
			return []viagens.CandidatoPlanejamento{novoCandidatoPlanejamento(agora, 4, viagens.SentidoIda, 22*time.Hour)}, nil
		}},
		fakeExecucaoPlanejamentoStore{
			tentarIniciarFn: func(context.Context, viagens.IniciarExecucaoPlanejamentoInput) (*viagens.ExecucaoPlanejamento, bool, error) {
				return &viagens.ExecucaoPlanejamento{ID: 40, Tentativas: 3}, true, nil
			},
			falharFn: func(_ context.Context, input viagens.FalharExecucaoPlanejamentoInput) (*viagens.ExecucaoPlanejamento, error) {
				if input.ExecucaoID != 40 {
					t.Fatalf("unexpected execution id: %d", input.ExecucaoID)
				}
				if !input.FalhouEm.Equal(agora) {
					t.Fatalf("unexpected failure time: %v", input.FalhouEm)
				}
				failureMessage = input.Mensagem
				proximaTentativa = input.ProximaTentativaEm
				return &viagens.ExecucaoPlanejamento{ID: input.ExecucaoID, Status: viagens.StatusExecucaoFalhou}, nil
			},
		},
		fakePlanejamentoService{planejarFn: func(context.Context, viagens.PlanejamentoViagensInput) (*viagens.PlanejamentoViagens, error) {
			return nil, planningErr
		}},
		viagens.ProcessadorPlanejamentoConfig{Location: location, Now: func() time.Time { return agora }},
	)

	resumo, err := processador.Processar(t.Context())
	if !errors.Is(err, planningErr) {
		t.Fatalf("expected planning error, got %v", err)
	}
	if !strings.Contains(failureMessage, planningErr.Error()) || resumo.Falhos != 1 {
		t.Fatalf("failure was not persisted: message=%q summary=%+v", failureMessage, resumo)
	}
	if !proximaTentativa.Equal(agora.Add(4 * time.Minute)) {
		t.Fatalf("unexpected retry time: %v", proximaTentativa)
	}
}

func TestProcessadorPlanejamento_ReturnsCandidateStoreError(t *testing.T) {
	storeErr := errors.New("db")
	processador := viagens.NewProcessadorPlanejamento(
		fakeAgendadorPlanejamentoStore{listFn: func(context.Context, time.Time, time.Time) ([]viagens.CandidatoPlanejamento, error) {
			return nil, storeErr
		}},
		fakeExecucaoPlanejamentoStore{},
		fakePlanejamentoService{},
		viagens.ProcessadorPlanejamentoConfig{},
	)

	_, err := processador.Processar(t.Context())
	if !errors.Is(err, storeErr) {
		t.Fatalf("expected candidate store error, got %v", err)
	}
}

func novoCandidatoPlanejamento(data time.Time, rotaInternaID int64, sentido viagens.SentidoViagem, horario time.Duration) viagens.CandidatoPlanejamento {
	return viagens.CandidatoPlanejamento{
		Chave: viagens.ChaveExecucaoPlanejamento{
			DataViagem:         data,
			Turno:              viagens.TurnoNoturno,
			MunicipioDestinoID: 2704302,
			RotaInternaID:      rotaInternaID,
			Sentido:            sentido,
		},
		HorarioPartida: horario,
	}
}
