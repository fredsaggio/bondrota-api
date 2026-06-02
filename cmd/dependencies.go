package main

import (
	"github.com/fredsaggio/bondrota-api/internal/admin"
	"github.com/fredsaggio/bondrota-api/internal/auth"
	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/fredsaggio/bondrota-api/internal/pontos"
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

	return server.Handlers{
		AdminHandler:   adminHandler,
		VeiculoHandler: veiculoHandler,
		PontoHandler:   pontoHandler,
	}
}
