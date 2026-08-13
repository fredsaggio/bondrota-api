package rotasdinamicas

import (
	"context"
	"time"
)

type invalidadorRotaDinamicaService struct {
	store          InvalidadorRotaDinamicaStore
	janelaBloqueio time.Duration
	now            func() time.Time
}

func NewInvalidadorRotaDinamicaService(store InvalidadorRotaDinamicaStore, janelaBloqueio time.Duration) InvalidadorRotaDinamicaService {
	if janelaBloqueio <= 0 {
		janelaBloqueio = DefaultJanelaBloqueioRotaDinamica
	}

	return &invalidadorRotaDinamicaService{
		store:          store,
		janelaBloqueio: janelaBloqueio,
		now:            time.Now,
	}
}

func (s *invalidadorRotaDinamicaService) InvalidarPorReserva(ctx context.Context, reservaID int64) error {
	if reservaID <= 0 {
		return invalidInput("Selecione a reserva.")
	}

	return s.store.DeleteRotasPorReservaAntesDoBloqueio(ctx, reservaID, s.now(), s.janelaBloqueio)
}
