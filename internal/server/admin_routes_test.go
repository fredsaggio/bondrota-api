package server

import (
	"net/http"
	"strings"
	"testing"

	"github.com/fredsaggio/bondrota-api/internal/auth"
	"github.com/fredsaggio/bondrota-api/internal/crypto"
	"github.com/go-chi/chi/v5"
)

// Sob /admin, escrita so pode agir sobre quem faz a requisicao, com a identidade
// vinda do JWT. Uma escrita que aceite {adminID} deixaria um admin agir sobre outro:
// enquanto houver uma unica role de admin, isso significa que qualquer sessao roubada
// fabrica acessos proprios e apaga os legitimos. Criar, alterar e remover conta ficam
// em cmd/admin, que exige acesso direto ao banco.
//
// Ao liberar uma rota nova aqui, confirme que ela nao aceita alvo por path nem por
// corpo. Este teste falha se alguem reintroduzir o CRUD.
func TestAdminWritesOnlyTargetTheCaller(t *testing.T) {
	srv := NewServer(Handlers{}, auth.NewAuthService(crypto.NewBcryptHasher(crypto.DefaultCost), "chave-de-teste"), Config{})
	router := chi.NewRouter()
	srv.RegisterRoutes(router)

	selfServiceWrites := map[string]bool{
		"/admin/login":  true, // abre a propria sessao
		"/admin/logout": true, // encerra a propria sessao
		"/admin/senha":  true, // troca a propria senha, adminID vem do JWT
	}

	var found int
	err := chi.Walk(router, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if !strings.HasPrefix(route, "/admin") {
			return nil
		}
		found++
		if method == http.MethodGet || selfServiceWrites[route] {
			return nil
		}
		t.Errorf("escrita em administrador fora da lista de auto-servico: %s %s", method, route)
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

// A lista acima so e segura enquanto nenhuma rota de auto-servico aceitar um alvo
// pelo path. Se alguem trocar /admin/senha por /admin/{adminID}/senha, a intencao
// muda de "minha senha" para "a senha de qualquer um" sem o teste acima notar.
func TestAdminSelfServiceRoutesTakeNoTargetParam(t *testing.T) {
	srv := NewServer(Handlers{}, auth.NewAuthService(crypto.NewBcryptHasher(crypto.DefaultCost), "chave-de-teste"), Config{})
	router := chi.NewRouter()
	srv.RegisterRoutes(router)

	err := chi.Walk(router, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if !strings.HasPrefix(route, "/admin") || method == http.MethodGet {
			return nil
		}
		if strings.Contains(route, "{") {
			t.Errorf("rota de escrita em administrador aceita alvo pelo path: %s %s", method, route)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk routes: %v", err)
	}
}
