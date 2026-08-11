package server

import (
	"net/http"
	"strings"
	"testing"

	"github.com/fredsaggio/bondrota-api/internal/auth"
	"github.com/fredsaggio/bondrota-api/internal/crypto"
	"github.com/go-chi/chi/v5"
)

// Contas de administrador so podem ser criadas, alteradas e removidas por cmd/admin,
// com acesso direto ao banco. Enquanto existir uma unica role de admin, expor essas
// operacoes por HTTP deixaria qualquer sessao roubada fabricar os proprios acessos.
// Este teste falha se alguem reintroduzir a rota.
func TestAdminRoutesAreReadOnly(t *testing.T) {
	srv := NewServer(Handlers{}, auth.NewAuthService(crypto.NewBcryptHasher(crypto.DefaultCost), "chave-de-teste"), Config{})
	router := chi.NewRouter()
	srv.RegisterRoutes(router)

	// Login e logout escrevem so na sessao do proprio requisitante, nao no cadastro.
	sessionRoutes := map[string]bool{"/admin/login": true, "/admin/logout": true}

	var found int
	err := chi.Walk(router, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if !strings.HasPrefix(route, "/admin") || sessionRoutes[route] {
			return nil
		}
		found++
		if method != http.MethodGet {
			t.Errorf("rota de escrita em administrador: %s %s", method, route)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk routes: %v", err)
	}
	// Guarda contra o teste passar porque o prefixo mudou e nada foi inspecionado.
	if found == 0 {
		t.Fatal("nenhuma rota /admin encontrada: o teste parou de inspecionar o que deveria")
	}
}
