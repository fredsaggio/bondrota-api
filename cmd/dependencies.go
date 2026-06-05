package main

import (
	"github.com/fredsaggio/bondrota-api/internal/admin"
	"github.com/fredsaggio/bondrota-api/internal/auth"
	"github.com/fredsaggio/bondrota-api/internal/clientes"
	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/fredsaggio/bondrota-api/internal/motoristas"
	"github.com/fredsaggio/bondrota-api/internal/paradas"
	"github.com/fredsaggio/bondrota-api/internal/pontos"
	"github.com/fredsaggio/bondrota-api/internal/rotasinternas"
	"github.com/fredsaggio/bondrota-api/internal/server"
	"github.com/fredsaggio/bondrota-api/internal/veiculos"
)

func buildHandlers(pool db.DB, authSvc *auth.AuthService) server.Handlers {
	adminStore := admin.NewAdminStore(pool)
	adminSvc := admin.NewAdminService(adminStore, authSvc)
	adminHandler := admin.NewAdminHandler(adminSvc)

	veiculoStore := veiculos.NewVeiculoStore(pool)
	veiculoHandler := veiculos.NewVeiculoHandler(veiculoStore)

	pontoStore := pontos.NewPontoStore(pool)
	pontoHandler := pontos.NewPontoHandler(pontoStore)

	paradaStore := paradas.NewParadaStore(pool)
	paradaHandler := paradas.NewParadaHandler(paradaStore)

	rotaInternaStore := rotasinternas.NewRotaInternaStore(pool)
	rotaInternaSvc := rotasinternas.NewRotaInternaService(rotaInternaStore)
	rotaInternaHandler := rotasinternas.NewRotaInternaHandler(rotaInternaSvc)

	motoristaStore := motoristas.NewMotoristaStore(pool)
	motoristaSvc := motoristas.NewMotoristaService(motoristaStore, authSvc)
	motoristaHandler := motoristas.NewMotoristaHandler(motoristaSvc)

	clienteStore := clientes.NewClienteStore(pool)
	clienteSvc := clientes.NewClienteService(clienteStore, authSvc)
	vinculoStore := clientes.NewVinculoStore(pool)
	vinculoSvc := clientes.NewVinculoService(vinculoStore)
	clienteHandler := clientes.NewClienteHandler(clienteSvc)
	vinculoHandler := clientes.NewVinculoHandler(vinculoSvc)

	return server.Handlers{
		AdminHandler:       adminHandler,
		VeiculoHandler:     veiculoHandler,
		PontoHandler:       pontoHandler,
		ParadaHandler:      paradaHandler,
		RotaInternaHandler: rotaInternaHandler,
		MotoristaHandler:   motoristaHandler,
		ClienteHandler:     clienteHandler,
		VinculoHandler:     vinculoHandler,
	}
}
