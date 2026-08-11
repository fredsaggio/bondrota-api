package retencao

import (
	"context"
	"fmt"
	"time"
)

type service struct {
	store  Store
	config Config
	agora  func() time.Time
}

func NewService(store Store, config Config) Service {
	if config.Meses <= 0 {
		config.Meses = MesesPadrao
	}
	if config.LoteMaximo <= 0 {
		config.LoteMaximo = LoteMaximoPadrao
	}
	if config.Location == nil {
		config.Location = time.UTC
	}
	return &service{store: store, config: config, agora: time.Now}
}

func (s *service) Limpar(ctx context.Context) (ResumoLimpeza, error) {
	corte := s.corte()
	resumo := ResumoLimpeza{Corte: corte}

	// Os ciclos saem primeiro: o cascata deles remove as viagem_reservas que
	// bloqueiam as reservas por chave estrangeira RESTRICT.
	ciclos, err := s.store.RemoverCiclosAntigos(ctx, corte, s.config.LoteMaximo)
	resumo.CiclosRemovidos = ciclos
	if err != nil {
		return resumo, fmt.Errorf("remover ciclos antigos: %w", err)
	}

	reservas, err := s.store.RemoverReservasAntigas(ctx, corte, s.config.LoteMaximo)
	resumo.ReservasRemovidas = reservas
	if err != nil {
		return resumo, fmt.Errorf("remover reservas antigas: %w", err)
	}

	execucoes, err := s.store.RemoverExecucoesAntigas(ctx, corte, s.config.LoteMaximo)
	resumo.ExecucoesRemovidas = execucoes
	if err != nil {
		return resumo, fmt.Errorf("remover execucoes antigas: %w", err)
	}

	limite := int64(s.config.LoteMaximo)
	resumo.LoteSaturado = ciclos >= limite || reservas >= limite || execucoes >= limite

	return resumo, nil
}

// corte devolve a meia-noite, no fuso da operacao, de N meses atras. Tudo com
// data_viagem estritamente anterior a esse instante ja passou da janela de
// auditoria e pode ser removido.
func (s *service) corte() time.Time {
	local := s.agora().In(s.config.Location)
	hoje := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, s.config.Location)
	return hoje.AddDate(0, -s.config.Meses, 0)
}
