package rotasdinamicas

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/brerror"
)

const (
	DefaultIntervaloWorkerRotaDinamica = 5 * time.Minute
	DefaultJanelaCalculoRotaDinamica   = 1 * time.Hour
	DefaultJanelaBloqueioRotaDinamica  = 30 * time.Minute
)

type RotaDinamicaWorkerConfig struct {
	Intervalo      time.Duration
	JanelaCalculo  time.Duration
	JanelaBloqueio time.Duration
}

type RotaDinamicaWorker struct {
	store          AgendadorRotaDinamicaStore
	calculador     CalculadorRotaDinamicaService
	intervalo      time.Duration
	janelaCalculo  time.Duration
	janelaBloqueio time.Duration
	now            func() time.Time
}

func NewRotaDinamicaWorker(
	store AgendadorRotaDinamicaStore,
	calculador CalculadorRotaDinamicaService,
	config RotaDinamicaWorkerConfig,
) *RotaDinamicaWorker {
	if config.Intervalo <= 0 {
		config.Intervalo = DefaultIntervaloWorkerRotaDinamica
	}
	if config.JanelaCalculo <= 0 {
		config.JanelaCalculo = DefaultJanelaCalculoRotaDinamica
	}
	if config.JanelaBloqueio <= 0 {
		config.JanelaBloqueio = DefaultJanelaBloqueioRotaDinamica
	}

	return &RotaDinamicaWorker{
		store:          store,
		calculador:     calculador,
		intervalo:      config.Intervalo,
		janelaCalculo:  config.JanelaCalculo,
		janelaBloqueio: config.JanelaBloqueio,
		now:            time.Now,
	}
}

func (w *RotaDinamicaWorker) Run(ctx context.Context) {
	if w == nil {
		return
	}

	w.Processar(ctx)

	ticker := time.NewTicker(w.intervalo)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.Processar(ctx)
		}
	}
}

func (w *RotaDinamicaWorker) Processar(ctx context.Context) {
	viagens, err := w.store.ListViagensPendentesCalculo(ctx, w.now(), w.janelaCalculo, w.janelaBloqueio)
	if err != nil {
		slog.Error("failed to list viagens pending dynamic route calculation", "error", err)
		return
	}

	for _, viagemID := range viagens {
		if _, err := w.calculador.Calcular(ctx, viagemID); err != nil {
			if errors.Is(err, brerror.ErrAlreadyExists) {
				continue
			}
			slog.Error("failed to calculate dynamic route", "viagem_id", viagemID, "error", err)
		}
	}
}
