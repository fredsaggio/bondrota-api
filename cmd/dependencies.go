package main

import (
	"time"

	"github.com/fredsaggio/bondrota-api/internal/admin"
	"github.com/fredsaggio/bondrota-api/internal/auth"
	"github.com/fredsaggio/bondrota-api/internal/clientes"
	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/fredsaggio/bondrota-api/internal/destinos"
	"github.com/fredsaggio/bondrota-api/internal/geo"
	"github.com/fredsaggio/bondrota-api/internal/motoristas"
	"github.com/fredsaggio/bondrota-api/internal/municipios"
	"github.com/fredsaggio/bondrota-api/internal/paradas"
	"github.com/fredsaggio/bondrota-api/internal/reservas"
	"github.com/fredsaggio/bondrota-api/internal/rotasdinamicas"
	"github.com/fredsaggio/bondrota-api/internal/rotasinternas"
	"github.com/fredsaggio/bondrota-api/internal/server"
	"github.com/fredsaggio/bondrota-api/internal/storage"
	"github.com/fredsaggio/bondrota-api/internal/veiculos"
	"github.com/fredsaggio/bondrota-api/internal/viagens"
)

func buildHandlers(pool db.DB, authSvc *auth.AuthService, storageConfig storage.SupabaseConfig, osrmBaseURL string, appLocation *time.Location) (server.Handlers, *rotasdinamicas.RotaDinamicaWorker) {
	adminStore := admin.NewAdminStore(pool)
	adminSvc := admin.NewAdminService(adminStore, authSvc)
	adminHandler := admin.NewAdminHandler(adminSvc)

	veiculoStore := veiculos.NewVeiculoStore(pool)
	alocacaoVeiculoStore := veiculos.NewAlocacaoVeiculoStore(pool)
	alocacaoVeiculoSvc := veiculos.NewAlocacaoService(alocacaoVeiculoStore)
	veiculoHandler := veiculos.NewVeiculoHandler(veiculoStore)

	destinoStore := destinos.NewDestinoStore(pool)
	destinoHandler := destinos.NewDestinoHandler(destinoStore)
	municipioStore := municipios.NewStore(pool)
	municipioHandler := municipios.NewHandler(municipioStore)

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
	reservaSvc := reservas.NewReservaService(reservaStore, reservas.ReservaServiceConfig{Location: appLocation}, rotaDinamicaInvalidator)
	reservaHandler := reservas.NewReservaHandler(reservaSvc)

	cicloViagemStore := viagens.NewCicloViagemStore(pool)
	horarioTurnoStore := viagens.NewHorarioTurnoViagemStore(pool)
	horarioTurnoSvc := viagens.NewHorarioTurnoViagemService(horarioTurnoStore)
	horarioTurnoHandler := viagens.NewHorarioTurnoViagemHandler(horarioTurnoSvc)
	planejamentoSvc := viagens.NewPlanejamentoService(cicloViagemStore, horarioTurnoStore, alocacaoVeiculoSvc, alocacaoMotoristaSvc, viagens.PlanejamentoServiceConfig{Location: appLocation})
	planejamentoHandler := viagens.NewPlanejamentoHandler(planejamentoSvc)
	agendadorPlanejamentoStore := viagens.NewAgendadorPlanejamentoStore(pool)
	execucaoPlanejamentoStore := viagens.NewExecucaoPlanejamentoStore(pool)
	execucaoPlanejamentoHandler := viagens.NewExecucaoPlanejamentoHandler(execucaoPlanejamentoStore)
	processadorPlanejamento := viagens.NewProcessadorPlanejamento(
		agendadorPlanejamentoStore,
		execucaoPlanejamentoStore,
		planejamentoSvc,
		viagens.ProcessadorPlanejamentoConfig{Location: appLocation},
	)
	processadorPlanejamentoHandler := viagens.NewProcessadorPlanejamentoHandler(processadorPlanejamento)

	viagemStore := viagens.NewViagemStore(pool)
	viagemSvc := viagens.NewViagemService(viagemStore)
	viagemReservaStore := viagens.NewViagemReservaStore(pool)
	presencaSvc := viagens.NewPresencaService(viagemReservaStore)
	viagemLocalizacaoStore := viagens.NewViagemLocalizacaoStore(pool)
	viagemLocalizacaoSvc := viagens.NewViagemLocalizacaoService(viagemLocalizacaoStore)
	viagemHandler := viagens.NewViagemHandler(viagemSvc, presencaSvc, viagemLocalizacaoSvc)

	rotaDinamicaStore := rotasdinamicas.NewRotaDinamicaStore(pool)
	rotaDinamicaSvc := rotasdinamicas.NewRotaDinamicaService(rotaDinamicaStore)
	roteador := geo.NewOSRMClient(nil, osrmBaseURL)
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

	storageClient := storage.NewSupabaseClient(storageConfig, nil)
	storageSvc := storage.NewService(storageClient)
	storageHandler := storage.NewHandler(storageSvc)

	return server.Handlers{
		AdminHandler:        adminHandler,
		VeiculoHandler:      veiculoHandler,
		DestinoHandler:      destinoHandler,
		MunicipioHandler:    municipioHandler,
		ParadaHandler:       paradaHandler,
		RotaInternaHandler:  rotaInternaHandler,
		MotoristaHandler:    motoristaHandler,
		ClienteHandler:      clienteHandler,
		VinculoHandler:      vinculoHandler,
		ReservaHandler:      reservaHandler,
		ViagemHandler:       viagemHandler,
		PlanejamentoHandler: planejamentoHandler,
		ProcessadorHandler:  processadorPlanejamentoHandler,
		ExecucaoHandler:     execucaoPlanejamentoHandler,
		HorarioTurnoHandler: horarioTurnoHandler,
		RotaDinamicaHandler: rotaDinamicaHandler,
		StorageHandler:      storageHandler,
	}, rotaDinamicaWorker
}
