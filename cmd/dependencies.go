package main

import (
	"github.com/fredsaggio/bondrota-api/internal/admin"
	"github.com/fredsaggio/bondrota-api/internal/auth"
	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/fredsaggio/bondrota-api/internal/server"
)

func buildHandlers(pool db.DB, authSvc *auth.AuthService) server.Handlers {
	adminStore := admin.NewAdminStore(pool)
	adminSvc := admin.NewAdminService(adminStore, authSvc)
	adminHandler := admin.NewAdminHandler(adminSvc)

	return server.Handlers{
		AdminHandler: adminHandler,
	}
}