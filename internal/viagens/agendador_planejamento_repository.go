package viagens

import (
	"context"
	"fmt"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/jackc/pgx/v5"
)

type agendadorPlanejamentoStore struct {
	db db.DB
}

func NewAgendadorPlanejamentoStore(database db.DB) AgendadorPlanejamentoStore {
	return &agendadorPlanejamentoStore{db: database}
}

func (s *agendadorPlanejamentoStore) ListCandidatos(ctx context.Context, dataInicio, dataFim time.Time) ([]CandidatoPlanejamento, error) {
	const op = "db/agendadorPlanejamentoStore.ListCandidatos"

	const q = `
		WITH candidatos AS (
			SELECT DISTINCT
				r.data_viagem,
				r.turno,
				d.municipio_id AS municipio_destino_id,
				r.rota_interna_id,
				'ida'::sentido_viagem AS sentido,
				EXTRACT(EPOCH FROM h.horario_ida)::BIGINT AS horario_partida_segundos
			FROM reservas r
			JOIN destinos d ON d.id = r.destino_id
			JOIN horarios_turno_viagem h
				ON h.municipio_destino_id = d.municipio_id
				AND h.turno = r.turno
			WHERE r.data_viagem BETWEEN @data_inicio::DATE AND @data_fim::DATE
				AND r.sentido = 'ida'
				AND r.status = 'confirmada'

			UNION

			SELECT DISTINCT
				c.data_viagem,
				c.turno,
				c.municipio_destino_id,
				c.rota_interna_id,
				'volta'::sentido_viagem AS sentido,
				EXTRACT(EPOCH FROM h.horario_volta)::BIGINT AS horario_partida_segundos
			FROM ciclos_viagem c
			JOIN viagens ida
				ON ida.ciclo_viagem_id = c.id
				AND ida.sentido = 'ida'
			JOIN horarios_turno_viagem h
				ON h.municipio_destino_id = c.municipio_destino_id
				AND h.turno = c.turno
			WHERE c.data_viagem BETWEEN @data_inicio::DATE AND @data_fim::DATE
				AND c.status <> 'cancelado'
				AND ida.status <> 'cancelada'
		)
		SELECT
			data_viagem,
			turno,
			municipio_destino_id,
			rota_interna_id,
			sentido,
			horario_partida_segundos
		FROM candidatos
		ORDER BY data_viagem, horario_partida_segundos, municipio_destino_id, rota_interna_id, sentido
	`

	rows, err := s.db.Query(ctx, q, pgx.StrictNamedArgs{
		"data_inicio": dataInicio.Format("2006-01-02"),
		"data_fim":    dataFim.Format("2006-01-02"),
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	candidatos, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (CandidatoPlanejamento, error) {
		var candidato CandidatoPlanejamento
		var horarioPartidaSegundos int64
		err := row.Scan(
			&candidato.Chave.DataViagem,
			&candidato.Chave.Turno,
			&candidato.Chave.MunicipioDestinoID,
			&candidato.Chave.RotaInternaID,
			&candidato.Chave.Sentido,
			&horarioPartidaSegundos,
		)
		candidato.HorarioPartida = time.Duration(horarioPartidaSegundos) * time.Second
		return candidato, err
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if candidatos == nil {
		return []CandidatoPlanejamento{}, nil
	}

	return candidatos, nil
}
