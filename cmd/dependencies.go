package main

import (
	"github.com/fredsaggio/bondrota-api/internal/admin"
	"github.com/fredsaggio/bondrota-api/internal/auth"
	"github.com/fredsaggio/bondrota-api/internal/clientes"
	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/fredsaggio/bondrota-api/internal/destinos"
	"github.com/fredsaggio/bondrota-api/internal/geo"
	"github.com/fredsaggio/bondrota-api/internal/motoristas"
	"github.com/fredsaggio/bondrota-api/internal/paradas"
	"github.com/fredsaggio/bondrota-api/internal/reservas"
	"github.com/fredsaggio/bondrota-api/internal/rotasdinamicas"
	"github.com/fredsaggio/bondrota-api/internal/rotasinternas"
	"github.com/fredsaggio/bondrota-api/internal/server"
	"github.com/fredsaggio/bondrota-api/internal/veiculos"
	"github.com/fredsaggio/bondrota-api/internal/viagens"
)

func buildHandlers(pool db.DB, authSvc *auth.AuthService) (server.Handlers, *rotasdinamicas.RotaDinamicaWorker) {
	adminStore := admin.NewAdminStore(pool)
	adminSvc := admin.NewAdminService(adminStore, authSvc)
	adminHandler := admin.NewAdminHandler(adminSvc)

	veiculoStore := veiculos.NewVeiculoStore(pool)
	alocacaoVeiculoStore := veiculos.NewAlocacaoVeiculoStore(pool)
	alocacaoVeiculoSvc := veiculos.NewAlocacaoService(alocacaoVeiculoStore)
	veiculoHandler := veiculos.NewVeiculoHandler(veiculoStore)

	destinoStore := destinos.NewDestinoStore(pool)
	destinoHandler := destinos.NewDestinoHandler(destinoStore)

	paradaStore := paradas.NewParadaStore(pool)
	paradaHandler := paradas.NewParadaHandler(paradaStore)

	rotaInternaStore := rotasinternas.NewRotaInternaStore(pool)
	rotaInternaSvc := rotasinternas.NewRotaInternaService(rotaInternaStore)
	rotaInternaHandler := rotasinternas.NewRotaInternaHandler(rotaInternaSvc)

	motoristaStore := motoristas.NewMotoristaStore(pool)
	alocacaoMotoristaStore := motoristas.NewAlocacaoMotoristaStore(pool)
	alocacaoMotoristaSvc := motoristas.NewAlocacaoService(alocacaoMotoristaStore)
	motoristaSvc := motoristas.NewMotoristaService(motoristaStore, authSvc)
	motoristaHandler := motoristas.NewMotoristaHandler(motoristaSvc)

	clienteStore := clientes.NewClienteStore(pool)
	clienteSvc := clientes.NewClienteService(clienteStore, authSvc)
	vinculoStore := clientes.NewVinculoStore(pool)
	vinculoSvc := clientes.NewVinculoService(vinculoStore)
	clienteHandler := clientes.NewClienteHandler(clienteSvc)
	vinculoHandler := clientes.NewVinculoHandler(vinculoSvc)

	calculadorRotaDinamicaStore := rotasdinamicas.NewCalculadorRotaDinamicaStore(pool)
	rotaDinamicaInvalidator := rotasdinamicas.NewInvalidadorRotaDinamicaService(
		calculadorRotaDinamicaStore,
		rotasdinamicas.DefaultJanelaBloqueioRotaDinamica,
	)

	reservaStore := reservas.NewReservaStore(pool)
	reservaSvc := reservas.NewReservaService(reservaStore, rotaDinamicaInvalidator)
	reservaHandler := reservas.NewReservaHandler(reservaSvc)

	cicloViagemStore := viagens.NewCicloViagemStore(pool)
	planejamentoSvc := viagens.NewPlanejamentoService(cicloViagemStore, alocacaoVeiculoSvc, alocacaoMotoristaSvc)
	planejamentoHandler := viagens.NewPlanejamentoHandler(planejamentoSvc)

	viagemStore := viagens.NewViagemStore(pool)
	viagemSvc := viagens.NewViagemService(viagemStore)
	viagemReservaStore := viagens.NewViagemReservaStore(pool)
	presencaSvc := viagens.NewPresencaService(viagemReservaStore)
	viagemHandler := viagens.NewViagemHandler(viagemSvc, presencaSvc)

	rotaDinamicaStore := rotasdinamicas.NewRotaDinamicaStore(pool)
	rotaDinamicaSvc := rotasdinamicas.NewRotaDinamicaService(rotaDinamicaStore)
	roteador := geo.NewOSRMClient(nil, "")
	otimizadorRota := geo.NewOtimizadorRota()
	calculadorRotaDinamicaSvc := rotasdinamicas.NewCalculadorRotaDinamicaService(
		calculadorRotaDinamicaStore,
		rotaDinamicaSvc,
		roteador,
		otimizadorRota,
	)
	rotaDinamicaHandler := rotasdinamicas.NewRotaDinamicaHandler(rotaDinamicaSvc, calculadorRotaDinamicaSvc)
	rotaDinamicaWorker := rotasdinamicas.NewRotaDinamicaWorker(
		calculadorRotaDinamicaStore,
		calculadorRotaDinamicaSvc,
		rotasdinamicas.RotaDinamicaWorkerConfig{},
	)

	return server.Handlers{
		AdminHandler:        adminHandler,
		VeiculoHandler:      veiculoHandler,
		DestinoHandler:      destinoHandler,
		ParadaHandler:       paradaHandler,
		RotaInternaHandler:  rotaInternaHandler,
		MotoristaHandler:    motoristaHandler,
		ClienteHandler:      clienteHandler,
		VinculoHandler:      vinculoHandler,
		ReservaHandler:      reservaHandler,
		ViagemHandler:       viagemHandler,
		PlanejamentoHandler: planejamentoHandler,
		RotaDinamicaHandler: rotaDinamicaHandler,
	}, rotaDinamicaWorker
}
