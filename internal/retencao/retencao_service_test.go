package retencao

import (
	"context"
	"errors"
	"testing"
	"time"
)

type storeSpy struct {
	chamadas  []string
	corte     time.Time
	limite    int
	ciclos    int64
	reservas  int64
	execucoes int64
	erroEm    string
}

func (s *storeSpy) registrar(nome string, corte time.Time, limite int) error {
	s.chamadas = append(s.chamadas, nome)
	s.corte = corte
	s.limite = limite
	if s.erroEm == nome {
		return errors.New("falha no banco")
	}
	return nil
}

func (s *storeSpy) RemoverCiclosAntigos(_ context.Context, corte time.Time, limite int) (int64, error) {
	if err := s.registrar("ciclos", corte, limite); err != nil {
		return 0, err
	}
	return s.ciclos, nil
}

func (s *storeSpy) RemoverReservasAntigas(_ context.Context, corte time.Time, limite int) (int64, error) {
	if err := s.registrar("reservas", corte, limite); err != nil {
		return 0, err
	}
	return s.reservas, nil
}

func (s *storeSpy) RemoverExecucoesAntigas(_ context.Context, corte time.Time, limite int) (int64, error) {
	if err := s.registrar("execucoes", corte, limite); err != nil {
		return 0, err
	}
	return s.execucoes, nil
}

func novoServico(t *testing.T, spy *storeSpy, config Config, agora time.Time) Service {
	t.Helper()
	svc := NewService(spy, config).(*service)
	svc.agora = func() time.Time { return agora }
	return svc
}

// Os ciclos precisam sair antes das reservas: viagem_reservas referencia reservas
// com ON DELETE RESTRICT, e sao os ciclos que derrubam as viagem_reservas em cascata.
// Inverter a ordem faria a limpeza falhar por chave estrangeira.
func TestLimparRemoveCiclosAntesDasReservas(t *testing.T) {
	spy := &storeSpy{}
	local := time.FixedZone("America/Maceio", -3*60*60)
	svc := novoServico(t, spy, Config{Location: local}, time.Date(2030, 6, 15, 10, 0, 0, 0, local))

	if _, err := svc.Limpar(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"ciclos", "reservas", "execucoes"}
	if len(spy.chamadas) != len(want) {
		t.Fatalf("want %v, got %v", want, spy.chamadas)
	}
	for i, nome := range want {
		if spy.chamadas[i] != nome {
			t.Fatalf("want order %v, got %v", want, spy.chamadas)
		}
	}
}

func TestCorteUsaFusoDaOperacaoEJanelaConfigurada(t *testing.T) {
	local := time.FixedZone("America/Maceio", -3*60*60)

	tests := []struct {
		name  string
		meses int
		agora time.Time
		want  time.Time
	}{
		{
			name:  "padrao de tres meses",
			meses: 0, // cai no MesesPadrao
			agora: time.Date(2030, 6, 15, 10, 0, 0, 0, local),
			want:  time.Date(2030, 3, 15, 0, 0, 0, 0, local),
		},
		{
			name:  "janela configuravel",
			meses: 6,
			agora: time.Date(2030, 6, 15, 10, 0, 0, 0, local),
			want:  time.Date(2029, 12, 15, 0, 0, 0, 0, local),
		},
		{
			// 22h em Maceio ja e o dia seguinte em UTC. O corte precisa seguir o dia
			// local, senao a janela desliza um dia conforme a hora da execucao.
			name:  "noite local nao antecipa o corte para o dia seguinte",
			meses: 3,
			agora: time.Date(2030, 6, 15, 22, 0, 0, 0, local),
			want:  time.Date(2030, 3, 15, 0, 0, 0, 0, local),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			spy := &storeSpy{}
			svc := novoServico(t, spy, Config{Meses: tc.meses, Location: local}, tc.agora)

			if _, err := svc.Limpar(context.Background()); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !spy.corte.Equal(tc.want) {
				t.Fatalf("want corte %s, got %s", tc.want, spy.corte)
			}
		})
	}
}

func TestLimparSinalizaLoteSaturado(t *testing.T) {
	local := time.UTC
	agora := time.Date(2030, 6, 15, 10, 0, 0, 0, local)

	t.Run("lote cheio indica trabalho restante", func(t *testing.T) {
		spy := &storeSpy{reservas: 10}
		svc := novoServico(t, spy, Config{LoteMaximo: 10, Location: local}, agora)

		resumo, err := svc.Limpar(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !resumo.LoteSaturado {
			t.Fatal("want LoteSaturado=true when a table hits the batch limit")
		}
		if spy.limite != 10 {
			t.Fatalf("want limite 10 propagated to the store, got %d", spy.limite)
		}
	})

	t.Run("lote parcial nao sinaliza", func(t *testing.T) {
		spy := &storeSpy{ciclos: 2, reservas: 3, execucoes: 1}
		svc := novoServico(t, spy, Config{LoteMaximo: 10, Location: local}, agora)

		resumo, err := svc.Limpar(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resumo.LoteSaturado {
			t.Fatal("want LoteSaturado=false when no table hits the batch limit")
		}
		if resumo.CiclosRemovidos != 2 || resumo.ReservasRemovidas != 3 || resumo.ExecucoesRemovidas != 1 {
			t.Fatalf("unexpected resumo: %+v", resumo)
		}
	})
}

// Uma falha no meio precisa interromper a limpeza e preservar o que ja foi contado,
// para que o log mostre ate onde a execucao chegou.
func TestLimparInterrompeEPreservaContagemParcial(t *testing.T) {
	local := time.UTC
	spy := &storeSpy{ciclos: 7, erroEm: "reservas"}
	svc := novoServico(t, spy, Config{LoteMaximo: 100, Location: local}, time.Date(2030, 6, 15, 10, 0, 0, 0, local))

	resumo, err := svc.Limpar(context.Background())

	if err == nil {
		t.Fatal("want error when the store fails")
	}
	if resumo.CiclosRemovidos != 7 {
		t.Fatalf("want partial count preserved, got %d", resumo.CiclosRemovidos)
	}
	if len(spy.chamadas) != 2 {
		t.Fatalf("want cleanup to stop at the failing step, got %v", spy.chamadas)
	}
}
