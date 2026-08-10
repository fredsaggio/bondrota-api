package viagens_test

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/viagens"
)

func testTime() time.Time {
	return time.Date(2026, 6, 6, 8, 0, 0, 0, time.UTC)
}

func body(v any) *bytes.Buffer {
	var buf bytes.Buffer
	_ = json.NewEncoder(&buf).Encode(v)
	return &buf
}

func sampleCiclo() viagens.CicloViagem {
	now := testTime()
	return viagens.CicloViagem{
		ID:                 1,
		DataViagem:         time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
		Turno:              viagens.TurnoNoturno,
		MunicipioDestinoID: 2704302,
		RotaInternaID:      2,
		VeiculoID:          3,
		MotoristaID:        4,
		Status:             viagens.StatusCicloPlanejado,
		ExpiresAt:          now.AddDate(0, 3, 0),
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

func sampleViagem() viagens.Viagem {
	now := testTime()
	return viagens.Viagem{
		ID:            10,
		CicloViagemID: 1,
		Sentido:       viagens.SentidoIda,
		Status:        viagens.StatusViagemProgramada,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func sampleViagemComCiclo() viagens.ViagemComCiclo {
	return viagens.ViagemComCiclo{
		Viagem: sampleViagem(),
		Ciclo:  sampleCiclo(),
	}
}

func sampleCicloComViagens() viagens.CicloComViagens {
	ida := sampleViagem()
	volta := sampleViagem()
	volta.ID = 11
	volta.Sentido = viagens.SentidoVolta

	return viagens.CicloComViagens{
		Ciclo:   sampleCiclo(),
		Viagens: []viagens.Viagem{ida, volta},
	}
}

func sampleViagemReserva() viagens.ViagemReserva {
	now := testTime()
	return viagens.ViagemReserva{
		ID:             100,
		ViagemID:       10,
		ReservaID:      20,
		StatusPresenca: viagens.StatusPresencaAguardando,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func sampleViagemReservaComReserva() viagens.ViagemReservaComReserva {
	return viagens.ViagemReservaComReserva{
		ViagemReserva: sampleViagemReserva(),
		ClienteID:     30,
		VinculoID:     40,
		DataViagem:    time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
		Turno:         viagens.TurnoNoturno,
		DestinoID:     50,
		RotaInternaID: 2,
		Sentido:       viagens.SentidoIda,
	}
}
