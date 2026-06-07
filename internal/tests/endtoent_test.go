package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/admin"
	"github.com/fredsaggio/bondrota-api/internal/auth"
	"github.com/fredsaggio/bondrota-api/internal/clientes"
	"github.com/fredsaggio/bondrota-api/internal/crypto"
	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/fredsaggio/bondrota-api/internal/destinos"
	"github.com/fredsaggio/bondrota-api/internal/geo"
	"github.com/fredsaggio/bondrota-api/internal/motoristas"
	"github.com/fredsaggio/bondrota-api/internal/paradas"
	"github.com/fredsaggio/bondrota-api/internal/reservas"
	"github.com/fredsaggio/bondrota-api/internal/rotasdinamicas"
	"github.com/fredsaggio/bondrota-api/internal/rotasinternas"
	"github.com/fredsaggio/bondrota-api/internal/server"
	"github.com/fredsaggio/bondrota-api/internal/storage"
	"github.com/fredsaggio/bondrota-api/internal/veiculos"
	"github.com/fredsaggio/bondrota-api/internal/viagens"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestEndToEndPlanejamentoViagem(t *testing.T) {
	dbURL := strings.TrimSpace(os.Getenv("E2E_DATABASE_URL"))
	if dbURL == "" {
		t.Skip("set E2E_DATABASE_URL to run end-to-end tests")
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect e2e database: %v", err)
	}
	t.Cleanup(pool.Close)

	jwtSecret := "e2e-secret"
	authSvc := auth.NewAuthService(crypto.NewBcryptHasher(crypto.DefaultCost), jwtSecret)
	osrmServer := newFakeOSRMServer(t)
	defer osrmServer.Close()
	router := buildE2ERouter(pool, authSvc, e2eRouterOptions{OSRMBaseURL: osrmServer.URL})

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	adminEmail := "admin-e2e-" + suffix + "@bondrota.test"
	adminPassword := "admin123"
	motoristaCPF := "900" + suffix[len(suffix)-8:]
	clienteCPF := "800" + suffix[len(suffix)-8:]
	placa := "E2E" + suffix[len(suffix)-4:]
	cidade := "e2e-cidade-" + suffix
	destinoNome := "UFAL E2E " + suffix

	t.Cleanup(func() {
		cleanupE2EData(context.Background(), t, pool, e2eCleanupData{
			AdminEmail:   adminEmail,
			MotoristaCPF: motoristaCPF,
			ClienteCPF:   clienteCPF,
			Placa:        placa,
			Cidade:       cidade,
			DestinoNome:  destinoNome,
		})
	})

	seedAdmin(t, ctx, pool, adminEmail, adminPassword)

	adminToken := loginAdmin(t, router, adminEmail, adminPassword)

	destinoID := createDestino(t, router, adminToken, map[string]any{
		"nome":      destinoNome,
		"rua":       "Av. Teste",
		"cidade":    "maceio",
		"latitude":  -9.5584,
		"longitude": -35.7777,
	})
	paradaID := createParada(t, router, adminToken, map[string]any{
		"nome":      "Praca E2E",
		"latitude":  -9.7812,
		"longitude": -36.3501,
		"cidade":    cidade,
	})
	rotaInternaID := createRotaInterna(t, router, adminToken, map[string]any{
		"cidade": cidade,
		"paradas": []map[string]any{
			{"parada_id": paradaID, "ordem": 1},
		},
	})
	createVeiculo(t, router, adminToken, map[string]any{
		"placa":           placa,
		"modelo":          "Van E2E",
		"categoria":       "carro_7_lugares",
		"capacidade":      7,
		"cidade_base":     cidade,
		"status":          "ativo",
		"ar_condicionado": false,
		"banheiro":        false,
		"persiana":        false,
		"luz_leitura":     false,
		"tomada":          false,
	})
	motoristaID := createMotorista(t, router, adminToken, map[string]any{
		"nome":            "Motorista E2E",
		"cpf":             motoristaCPF,
		"senha":           "senha123",
		"telefone":        "82999990000",
		"data_nasc":       "1980-05-20",
		"turno":           "NT",
		"cidade_trabalho": cidade,
		"residencia":      cidade,
		"foto":            "",
	})
	clienteID := createCliente(t, router, adminToken, map[string]any{
		"nome":      "Cliente E2E",
		"cpf":       clienteCPF,
		"senha":     "senha123",
		"telefone":  "82999991111",
		"data_nasc": "2002-08-10",
		"foto":      "",
	})
	vinculoID := createVinculo(t, router, adminToken, clienteID, map[string]any{
		"tipo":            "estudante",
		"turno":           "NT",
		"destino_id":      destinoID,
		"rota_interna_id": rotaInternaID,
		"curso":           "Sistemas",
		"comprovante":     "clientes/1/vinculos/1/comprovante.pdf",
		"validade":        "2027-12-31",
		"horarios_fixos":  []int{1, 2, 3, 4, 5},
	})

	dataViagem := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	createReserva(t, router, adminToken, clienteID, vinculoID, map[string]any{
		"data_viagem": dataViagem,
		"turno":       "NT",
		"sentido":     "ida",
	})
	createReserva(t, router, adminToken, clienteID, vinculoID, map[string]any{
		"data_viagem": dataViagem,
		"turno":       "NT",
		"sentido":     "volta",
	})
	createHorarioTurno(t, router, adminToken, map[string]any{
		"cidade":        cidade,
		"turno":         "NT",
		"horario_ida":   "17:00",
		"horario_volta": "22:00",
	})

	planejamento := doJSON[map[string]any](t, router, http.MethodPost, "/api/v1/planejamentos/viagens", adminToken, map[string]any{
		"data_viagem":     dataViagem,
		"turno":           "NT",
		"cidade":          cidade,
		"rota_interna_id": rotaInternaID,
	}, http.StatusCreated)

	ciclos := planejamento["ciclos"].([]any)
	if len(ciclos) != 1 {
		t.Fatalf("expected 1 ciclo, got %d", len(ciclos))
	}
	ciclo := ciclos[0].(map[string]any)
	viagensResp := ciclo["viagens"].([]any)
	if len(viagensResp) != 2 {
		t.Fatalf("expected ida and volta viagens, got %d", len(viagensResp))
	}
	cicloData := ciclo["ciclo"].(map[string]any)
	cicloID := int64(cicloData["id"].(float64))
	veiculoID := int64(cicloData["veiculo_id"].(float64))
	cicloMotoristaID := int64(cicloData["motorista_id"].(float64))
	if veiculoID <= 0 {
		t.Fatalf("expected ciclo veiculo_id to be set, got %d", veiculoID)
	}
	if cicloMotoristaID != motoristaID {
		t.Fatalf("expected same motorista on ciclo: got %d, want %d", cicloMotoristaID, motoristaID)
	}
	for _, viagem := range viagensResp {
		viagemData := viagem.(map[string]any)
		if int64(viagemData["ciclo_viagem_id"].(float64)) != cicloID {
			t.Fatalf("expected ida/volta to share ciclo %d, got viagem %+v", cicloID, viagemData)
		}
	}

	viagemID := int64(viagensResp[0].(map[string]any)["id"].(float64))
	viagemVoltaID := int64(viagensResp[1].(map[string]any)["id"].(float64))
	idaComCiclo := doJSON[map[string]any](t, router, http.MethodGet, fmt.Sprintf("/api/v1/viagens/%d", viagemID), adminToken, nil, http.StatusOK)
	voltaComCiclo := doJSON[map[string]any](t, router, http.MethodGet, fmt.Sprintf("/api/v1/viagens/%d", viagemVoltaID), adminToken, nil, http.StatusOK)
	idaCiclo := idaComCiclo["ciclo"].(map[string]any)
	voltaCiclo := voltaComCiclo["ciclo"].(map[string]any)
	if idaCiclo["veiculo_id"] != voltaCiclo["veiculo_id"] || idaCiclo["motorista_id"] != voltaCiclo["motorista_id"] {
		t.Fatalf("expected ida/volta to share veiculo and motorista: ida=%+v volta=%+v", idaCiclo, voltaCiclo)
	}
	motoristaToken := loginMotorista(t, router, motoristaCPF, "senha123")

	rota := doJSON[map[string]any](t, router, http.MethodPost, fmt.Sprintf("/api/v1/viagens/%d/rota-dinamica/calcular", viagemID), motoristaToken, nil, http.StatusCreated)
	rotaData := rota["rota"].(map[string]any)
	if int64(rotaData["distancia_metros"].(float64)) != 12346 {
		t.Fatalf("unexpected dynamic route distance: %v", rotaData["distancia_metros"])
	}
	if osrmServer.Requests() != 1 {
		t.Fatalf("expected 1 OSRM request, got %d", osrmServer.Requests())
	}

	doJSON[map[string]any](t, router, http.MethodPost, fmt.Sprintf("/api/v1/viagens/%d/iniciar", viagemID), motoristaToken, nil, http.StatusOK)

	reservasViagem := doJSON[[]map[string]any](t, router, http.MethodGet, fmt.Sprintf("/api/v1/viagens/%d/reservas/", viagemID), motoristaToken, nil, http.StatusOK)
	if len(reservasViagem) != 1 {
		t.Fatalf("expected 1 reserva on selected viagem, got %d", len(reservasViagem))
	}
	reservaID := int64(reservasViagem[0]["reserva_id"].(float64))

	doJSON[map[string]any](t, router, http.MethodPut, fmt.Sprintf("/api/v1/viagens/%d/reservas/%d/presenca", viagemID, reservaID), motoristaToken, map[string]any{
		"status_presenca": "embarcou",
	}, http.StatusOK)

	localizacao := doJSON[map[string]any](t, router, http.MethodPut, fmt.Sprintf("/api/v1/viagens/%d/localizacao", viagemID), motoristaToken, map[string]any{
		"motorista_id":    motoristaID,
		"latitude":        -9.7812,
		"longitude":       -36.3501,
		"velocidade_kmh":  42.5,
		"direcao_graus":   180,
		"precisao_metros": 8,
	}, http.StatusOK)
	if int64(localizacao["motorista_id"].(float64)) != motoristaID {
		t.Fatalf("unexpected motorista_id in localizacao: %v", localizacao["motorista_id"])
	}

	clienteToken := loginCliente(t, router, clienteCPF, "senha123")
	doJSON[map[string]any](t, router, http.MethodGet, fmt.Sprintf("/api/v1/viagens/%d/localizacao", viagemID), clienteToken, nil, http.StatusOK)

	doJSON[map[string]any](t, router, http.MethodPost, fmt.Sprintf("/api/v1/viagens/%d/concluir", viagemID), motoristaToken, nil, http.StatusOK)

	doJSON[map[string]any](t, router, http.MethodPost, fmt.Sprintf("/api/v1/viagens/%d/iniciar", viagemVoltaID), motoristaToken, nil, http.StatusOK)
	reservasVolta := doJSON[[]map[string]any](t, router, http.MethodGet, fmt.Sprintf("/api/v1/viagens/%d/reservas/", viagemVoltaID), motoristaToken, nil, http.StatusOK)
	if len(reservasVolta) != 1 {
		t.Fatalf("expected 1 reserva on volta viagem, got %d", len(reservasVolta))
	}
	reservaVoltaID := int64(reservasVolta[0]["reserva_id"].(float64))
	doJSON[map[string]any](t, router, http.MethodPut, fmt.Sprintf("/api/v1/viagens/%d/reservas/%d/presenca", viagemVoltaID, reservaVoltaID), motoristaToken, map[string]any{
		"status_presenca": "embarcou",
	}, http.StatusOK)
	doJSON[map[string]any](t, router, http.MethodPost, fmt.Sprintf("/api/v1/viagens/%d/concluir", viagemVoltaID), motoristaToken, nil, http.StatusOK)
}

func TestEndToEndSupabaseStorageSignedURLs(t *testing.T) {
	dbURL := strings.TrimSpace(os.Getenv("E2E_DATABASE_URL"))
	if dbURL == "" {
		t.Skip("set E2E_DATABASE_URL to run end-to-end tests")
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect e2e database: %v", err)
	}
	t.Cleanup(pool.Close)

	supabase := newFakeSupabaseStorageServer(t)
	defer supabase.Close()

	authSvc := auth.NewAuthService(crypto.NewBcryptHasher(crypto.DefaultCost), "e2e-secret")
	router := buildE2ERouter(pool, authSvc, e2eRouterOptions{
		StorageConfig: storage.SupabaseConfig{
			URL:        supabase.URL,
			ServiceKey: "service-key",
		},
	})

	clienteToken, err := authSvc.GenerateToken(1, auth.RoleCliente)
	if err != nil {
		t.Fatalf("generate cliente token: %v", err)
	}
	motoristaToken, err := authSvc.GenerateToken(1, auth.RoleMotorista)
	if err != nil {
		t.Fatalf("generate motorista token: %v", err)
	}

	upload := doJSON[map[string]any](t, router, http.MethodPost, "/api/v1/storage/signed-upload-url", clienteToken, map[string]any{
		"bucket":       "fotos",
		"path":         "clientes/1/foto.png",
		"content_type": "image/png",
		"upsert":       true,
	}, http.StatusCreated)
	if upload["path"] != "clientes/1/foto.png" {
		t.Fatalf("unexpected signed upload path: %v", upload["path"])
	}

	req, err := http.NewRequest(http.MethodPut, upload["signed_url"].(string), strings.NewReader("fake-image"))
	if err != nil {
		t.Fatalf("create fake upload request: %v", err)
	}
	req.Header.Set("Content-Type", "image/png")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("send fake upload: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("fake upload: want 200, got %d", resp.StatusCode)
	}

	download := doJSON[map[string]any](t, router, http.MethodPost, "/api/v1/storage/signed-download-url", clienteToken, map[string]any{
		"bucket":             "fotos",
		"path":               "clientes/1/foto.png",
		"expires_in_seconds": 900,
	}, http.StatusOK)
	if download["signed_url"] == "" {
		t.Fatalf("expected signed download url")
	}

	doStatus(t, router, http.MethodPost, "/api/v1/storage/signed-upload-url", clienteToken, map[string]any{
		"bucket":       "fotos",
		"path":         "clientes/2/foto.png",
		"content_type": "image/png",
	}, http.StatusForbidden)
	doStatus(t, router, http.MethodPost, "/api/v1/storage/signed-upload-url", clienteToken, map[string]any{
		"bucket":       "arquivos",
		"path":         "clientes/1/foto.png",
		"content_type": "image/png",
	}, http.StatusUnprocessableEntity)
	doStatus(t, router, http.MethodPost, "/api/v1/storage/signed-upload-url", clienteToken, map[string]any{
		"bucket":       "fotos",
		"path":         "clientes/1/foto.exe",
		"content_type": "image/png",
	}, http.StatusUnprocessableEntity)
	doStatus(t, router, http.MethodPost, "/api/v1/storage/signed-upload-url", motoristaToken, map[string]any{
		"bucket":       "documentos",
		"path":         "motoristas/1/cnh.pdf",
		"content_type": "application/pdf",
	}, http.StatusForbidden)

	if supabase.SignUploadRequests() != 1 {
		t.Fatalf("expected 1 signed upload request, got %d", supabase.SignUploadRequests())
	}
	if supabase.UploadRequests() != 1 {
		t.Fatalf("expected 1 direct upload request, got %d", supabase.UploadRequests())
	}
	if supabase.SignDownloadRequests() != 1 {
		t.Fatalf("expected 1 signed download request, got %d", supabase.SignDownloadRequests())
	}
}

func TestEndToEndAutorizacaoRoles(t *testing.T) {
	dbURL := strings.TrimSpace(os.Getenv("E2E_DATABASE_URL"))
	if dbURL == "" {
		t.Skip("set E2E_DATABASE_URL to run end-to-end tests")
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect e2e database: %v", err)
	}
	t.Cleanup(pool.Close)

	authSvc := auth.NewAuthService(crypto.NewBcryptHasher(crypto.DefaultCost), "e2e-secret")
	router := buildE2ERouter(pool, authSvc, e2eRouterOptions{})

	clienteToken, err := authSvc.GenerateToken(1, auth.RoleCliente)
	if err != nil {
		t.Fatalf("generate cliente token: %v", err)
	}
	motoristaToken, err := authSvc.GenerateToken(1, auth.RoleMotorista)
	if err != nil {
		t.Fatalf("generate motorista token: %v", err)
	}

	doStatus(t, router, http.MethodGet, "/api/v1/veiculos/", "", nil, http.StatusUnauthorized)
	doStatus(t, router, http.MethodPost, "/api/v1/veiculos/", clienteToken, map[string]any{}, http.StatusForbidden)
	doStatus(t, router, http.MethodPost, "/api/v1/clientes/1/vinculos/", motoristaToken, map[string]any{}, http.StatusForbidden)
	doStatus(t, router, http.MethodPut, "/api/v1/viagens/1/localizacao", clienteToken, map[string]any{}, http.StatusForbidden)
}

func TestEndToEndPlanejamentoMultiplosVeiculosPorCapacidade(t *testing.T) {
	h := newE2EHarness(t, e2eRouterOptions{})
	suffix := h.Suffix
	cidade := "e2e-capacidade-" + suffix
	h.Cleanup(e2eCleanupData{
		AdminEmail:      h.AdminEmail,
		MotoristaPrefix: "91" + suffix[len(suffix)-6:],
		ClientePrefix:   "81" + suffix[len(suffix)-6:],
		PlacaPrefix:     "C" + suffix[len(suffix)-5:],
		Cidade:          cidade,
		DestinoPrefix:   "Destino Capacidade E2E " + suffix,
	})

	destinoID := createDestino(t, h.Router, h.AdminToken, map[string]any{
		"nome":      "Destino Capacidade E2E " + suffix,
		"rua":       "Av. Teste",
		"cidade":    "maceio",
		"latitude":  -9.5584,
		"longitude": -35.7777,
	})
	paradaID := createParada(t, h.Router, h.AdminToken, map[string]any{
		"nome":      "Parada Capacidade E2E",
		"latitude":  -9.7812,
		"longitude": -36.3501,
		"cidade":    cidade,
	})
	rotaInternaID := createRotaInterna(t, h.Router, h.AdminToken, map[string]any{
		"cidade":  cidade,
		"paradas": []map[string]any{{"parada_id": paradaID, "ordem": 1}},
	})

	createVeiculo(t, h.Router, h.AdminToken, map[string]any{
		"placa":       "C" + suffix[len(suffix)-5:] + "E",
		"modelo":      "Executivo E2E",
		"categoria":   "executivo",
		"capacidade":  46,
		"cidade_base": cidade,
		"status":      "ativo",
	})
	createVeiculo(t, h.Router, h.AdminToken, map[string]any{
		"placa":       "C" + suffix[len(suffix)-5:] + "7",
		"modelo":      "Carro E2E",
		"categoria":   "carro_7_lugares",
		"capacidade":  7,
		"cidade_base": cidade,
		"status":      "ativo",
	})
	for i := 0; i < 2; i++ {
		createMotorista(t, h.Router, h.AdminToken, map[string]any{
			"nome":            fmt.Sprintf("Motorista Capacidade %d", i),
			"cpf":             fmt.Sprintf("91%s%02d", suffix[len(suffix)-6:], i),
			"senha":           "senha123",
			"telefone":        "82999990000",
			"data_nasc":       "1980-05-20",
			"turno":           "NT",
			"cidade_trabalho": cidade,
			"residencia":      cidade,
			"foto":            "",
		})
	}

	dataViagem := time.Now().AddDate(0, 0, 2).Format("2006-01-02")
	for i := 0; i < 48; i++ {
		clienteID := createCliente(t, h.Router, h.AdminToken, map[string]any{
			"nome":      fmt.Sprintf("Cliente Capacidade %02d", i),
			"cpf":       fmt.Sprintf("81%s%02d", suffix[len(suffix)-6:], i),
			"senha":     "senha123",
			"telefone":  "82999991111",
			"data_nasc": "2002-08-10",
			"foto":      "",
		})
		vinculoID := createVinculo(t, h.Router, h.AdminToken, clienteID, map[string]any{
			"tipo":            "estudante",
			"turno":           "NT",
			"destino_id":      destinoID,
			"rota_interna_id": rotaInternaID,
			"curso":           "Sistemas",
			"comprovante":     fmt.Sprintf("clientes/%d/e2e/comprovante.pdf", clienteID),
			"validade":        "2027-12-31",
			"horarios_fixos":  []int{1, 2, 3, 4, 5},
		})
		createReserva(t, h.Router, h.AdminToken, clienteID, vinculoID, map[string]any{
			"data_viagem": dataViagem,
			"turno":       "NT",
			"sentido":     "ida",
		})
	}
	createHorarioTurno(t, h.Router, h.AdminToken, map[string]any{
		"cidade":        cidade,
		"turno":         "NT",
		"horario_ida":   "17:00",
		"horario_volta": "22:00",
	})

	planejamento := doJSON[map[string]any](t, h.Router, http.MethodPost, "/api/v1/planejamentos/viagens", h.AdminToken, map[string]any{
		"data_viagem":     dataViagem,
		"turno":           "NT",
		"cidade":          cidade,
		"rota_interna_id": rotaInternaID,
	}, http.StatusCreated)

	ciclos := planejamento["ciclos"].([]any)
	if len(ciclos) != 2 {
		t.Fatalf("expected 2 ciclos for 48 students, got %d", len(ciclos))
	}
	if int(planejamento["quantidade_reservas_ida"].(float64)) != 48 {
		t.Fatalf("unexpected ida reservation count: %v", planejamento["quantidade_reservas_ida"])
	}
	if int(planejamento["capacidade_total"].(float64)) != 53 {
		t.Fatalf("expected capacity 53, got %v", planejamento["capacidade_total"])
	}
}

func TestEndToEndPlanejamentoIgnoraRecursosIndisponiveis(t *testing.T) {
	h := newE2EHarness(t, e2eRouterOptions{})
	suffix := h.Suffix
	cidade := "e2e-recursos-" + suffix
	h.Cleanup(e2eCleanupData{
		AdminEmail:      h.AdminEmail,
		MotoristaPrefix: "90" + suffix[len(suffix)-6:],
		ClientePrefix:   "80" + suffix[len(suffix)-6:],
		PlacaPrefix:     "R" + suffix[len(suffix)-5:],
		Cidade:          cidade,
		DestinoPrefix:   "Destino Recursos E2E " + suffix,
	})

	destinoID := createDestino(t, h.Router, h.AdminToken, map[string]any{
		"nome":      "Destino Recursos E2E " + suffix,
		"rua":       "Av. Teste",
		"cidade":    "maceio",
		"latitude":  -9.5584,
		"longitude": -35.7777,
	})
	paradaID := createParada(t, h.Router, h.AdminToken, map[string]any{
		"nome":      "Parada Recursos E2E",
		"latitude":  -9.7812,
		"longitude": -36.3501,
		"cidade":    cidade,
	})
	rotaInternaID := createRotaInterna(t, h.Router, h.AdminToken, map[string]any{
		"cidade":  cidade,
		"paradas": []map[string]any{{"parada_id": paradaID, "ordem": 1}},
	})

	inactiveCarID := createVeiculo(t, h.Router, h.AdminToken, map[string]any{
		"placa":       "R" + suffix[len(suffix)-5:] + "I",
		"modelo":      "Carro Inativo E2E",
		"categoria":   "carro_7_lugares",
		"capacidade":  7,
		"cidade_base": cidade,
		"status":      "inativo",
	})
	activeEscolarID := createVeiculo(t, h.Router, h.AdminToken, map[string]any{
		"placa":       "R" + suffix[len(suffix)-5:] + "A",
		"modelo":      "Escolar Ativo E2E",
		"categoria":   "escolar",
		"capacidade":  24,
		"cidade_base": cidade,
		"status":      "ativo",
	})
	wrongTurnoMotoristaID := createMotorista(t, h.Router, h.AdminToken, map[string]any{
		"nome":            "Motorista Turno Errado E2E",
		"cpf":             "90" + suffix[len(suffix)-6:] + "00",
		"senha":           "senha123",
		"telefone":        "82999990000",
		"data_nasc":       "1980-05-20",
		"turno":           "MT",
		"cidade_trabalho": cidade,
		"residencia":      cidade,
		"foto":            "",
	})
	correctMotoristaID := createMotorista(t, h.Router, h.AdminToken, map[string]any{
		"nome":            "Motorista Turno Certo E2E",
		"cpf":             "90" + suffix[len(suffix)-6:] + "01",
		"senha":           "senha123",
		"telefone":        "82999990001",
		"data_nasc":       "1980-05-20",
		"turno":           "NT",
		"cidade_trabalho": cidade,
		"residencia":      cidade,
		"foto":            "",
	})

	clienteID := createCliente(t, h.Router, h.AdminToken, map[string]any{
		"nome":      "Cliente Recursos E2E",
		"cpf":       "80" + suffix[len(suffix)-6:] + "00",
		"senha":     "senha123",
		"telefone":  "82999991111",
		"data_nasc": "2002-08-10",
		"foto":      "",
	})
	vinculoID := createVinculo(t, h.Router, h.AdminToken, clienteID, map[string]any{
		"tipo":            "estudante",
		"turno":           "NT",
		"destino_id":      destinoID,
		"rota_interna_id": rotaInternaID,
		"curso":           "Sistemas",
		"comprovante":     fmt.Sprintf("clientes/%d/e2e/comprovante.pdf", clienteID),
		"validade":        "2027-12-31",
		"horarios_fixos":  []int{1, 2, 3, 4, 5},
	})
	dataViagem := time.Now().AddDate(0, 0, 7).Format("2006-01-02")
	createReserva(t, h.Router, h.AdminToken, clienteID, vinculoID, map[string]any{
		"data_viagem": dataViagem,
		"turno":       "NT",
		"sentido":     "ida",
	})
	createHorarioTurno(t, h.Router, h.AdminToken, map[string]any{
		"cidade":        cidade,
		"turno":         "NT",
		"horario_ida":   "17:00",
		"horario_volta": "22:00",
	})

	planejamento := doJSON[map[string]any](t, h.Router, http.MethodPost, "/api/v1/planejamentos/viagens", h.AdminToken, map[string]any{
		"data_viagem":     dataViagem,
		"turno":           "NT",
		"cidade":          cidade,
		"rota_interna_id": rotaInternaID,
	}, http.StatusCreated)
	ciclo := planejamento["ciclos"].([]any)[0].(map[string]any)["ciclo"].(map[string]any)
	if got := int64(ciclo["veiculo_id"].(float64)); got != activeEscolarID {
		t.Fatalf("expected active escolar fallback vehicle %d, got %d; inactive car was %d", activeEscolarID, got, inactiveCarID)
	}
	if got := int64(ciclo["motorista_id"].(float64)); got != correctMotoristaID {
		t.Fatalf("expected correct turno motorista %d, got %d; wrong turno motorista was %d", correctMotoristaID, got, wrongTurnoMotoristaID)
	}
}

func TestEndToEndPlanejamentoNaoReutilizaRecursosJaAlocados(t *testing.T) {
	h := newE2EHarness(t, e2eRouterOptions{})
	suffix := h.Suffix
	cidade := "e2e-recursos-alocados-" + suffix
	h.Cleanup(e2eCleanupData{
		AdminEmail:      h.AdminEmail,
		MotoristaPrefix: "76" + suffix[len(suffix)-6:],
		ClientePrefix:   "76" + suffix[len(suffix)-6:],
		PlacaPrefix:     "U" + suffix[len(suffix)-5:],
		Cidade:          cidade,
		DestinoPrefix:   "Destino Reuso E2E " + suffix,
	})

	destinoID := createDestino(t, h.Router, h.AdminToken, map[string]any{
		"nome":      "Destino Reuso E2E " + suffix,
		"rua":       "Av. Teste",
		"cidade":    "maceio",
		"latitude":  -9.5584,
		"longitude": -35.7777,
	})
	paradaAID := createParada(t, h.Router, h.AdminToken, map[string]any{
		"nome":      "Parada Reuso A E2E",
		"latitude":  -9.7812,
		"longitude": -36.3501,
		"cidade":    cidade,
	})
	paradaBID := createParada(t, h.Router, h.AdminToken, map[string]any{
		"nome":      "Parada Reuso B E2E",
		"latitude":  -9.7902,
		"longitude": -36.3601,
		"cidade":    cidade,
	})
	rotaInternaAID := createRotaInterna(t, h.Router, h.AdminToken, map[string]any{
		"cidade":  cidade,
		"paradas": []map[string]any{{"parada_id": paradaAID, "ordem": 1}},
	})
	rotaInternaBID := createRotaInterna(t, h.Router, h.AdminToken, map[string]any{
		"cidade":  cidade,
		"paradas": []map[string]any{{"parada_id": paradaBID, "ordem": 1}},
	})
	veiculoID := createVeiculo(t, h.Router, h.AdminToken, map[string]any{
		"placa":       "U" + suffix[len(suffix)-5:] + "7",
		"modelo":      "Carro Reuso E2E",
		"categoria":   "carro_7_lugares",
		"capacidade":  7,
		"cidade_base": cidade,
		"status":      "ativo",
	})
	motoristaID := createMotorista(t, h.Router, h.AdminToken, map[string]any{
		"nome":            "Motorista Reuso E2E",
		"cpf":             "76" + suffix[len(suffix)-6:] + "00",
		"senha":           "senha123",
		"telefone":        "82999990000",
		"data_nasc":       "1980-05-20",
		"turno":           "NT",
		"cidade_trabalho": cidade,
		"residencia":      cidade,
		"foto":            "",
	})

	dataViagem := time.Now().AddDate(0, 0, 10).Format("2006-01-02")
	for i, rotaInternaID := range []int64{rotaInternaAID, rotaInternaBID} {
		clienteID := createCliente(t, h.Router, h.AdminToken, map[string]any{
			"nome":      fmt.Sprintf("Cliente Reuso E2E %d", i),
			"cpf":       fmt.Sprintf("76%s%02d", suffix[len(suffix)-6:], i+1),
			"senha":     "senha123",
			"telefone":  "82999991111",
			"data_nasc": "2002-08-10",
			"foto":      "",
		})
		vinculoID := createVinculo(t, h.Router, h.AdminToken, clienteID, map[string]any{
			"tipo":            "estudante",
			"turno":           "NT",
			"destino_id":      destinoID,
			"rota_interna_id": rotaInternaID,
			"curso":           "Sistemas",
			"comprovante":     fmt.Sprintf("clientes/%d/e2e/comprovante.pdf", clienteID),
			"validade":        "2027-12-31",
			"horarios_fixos":  []int{1, 2, 3, 4, 5},
		})
		createReserva(t, h.Router, h.AdminToken, clienteID, vinculoID, map[string]any{
			"data_viagem": dataViagem,
			"turno":       "NT",
			"sentido":     "ida",
		})
	}
	createHorarioTurno(t, h.Router, h.AdminToken, map[string]any{
		"cidade":        cidade,
		"turno":         "NT",
		"horario_ida":   "17:00",
		"horario_volta": "22:00",
	})

	primeiro := doJSON[map[string]any](t, h.Router, http.MethodPost, "/api/v1/planejamentos/viagens", h.AdminToken, map[string]any{
		"data_viagem":     dataViagem,
		"turno":           "NT",
		"cidade":          cidade,
		"rota_interna_id": rotaInternaAID,
	}, http.StatusCreated)
	ciclo := primeiro["ciclos"].([]any)[0].(map[string]any)["ciclo"].(map[string]any)
	if got := int64(ciclo["veiculo_id"].(float64)); got != veiculoID {
		t.Fatalf("expected first ciclo to use vehicle %d, got %d", veiculoID, got)
	}
	if got := int64(ciclo["motorista_id"].(float64)); got != motoristaID {
		t.Fatalf("expected first ciclo to use motorista %d, got %d", motoristaID, got)
	}

	doStatus(t, h.Router, http.MethodPost, "/api/v1/planejamentos/viagens", h.AdminToken, map[string]any{
		"data_viagem":     dataViagem,
		"turno":           "NT",
		"cidade":          cidade,
		"rota_interna_id": rotaInternaBID,
	}, http.StatusNotFound)
}

func TestEndToEndRotaDinamicaMultiplosDestinos(t *testing.T) {
	if strings.TrimSpace(os.Getenv("E2E_DATABASE_URL")) == "" {
		t.Skip("set E2E_DATABASE_URL to run end-to-end tests")
	}

	osrmServer := newFakeOSRMServer(t)
	defer osrmServer.Close()
	h := newE2EHarness(t, e2eRouterOptions{OSRMBaseURL: osrmServer.URL})
	suffix := h.Suffix
	cidade := "e2e-multidestino-" + suffix
	h.Cleanup(e2eCleanupData{
		AdminEmail:      h.AdminEmail,
		MotoristaPrefix: "92" + suffix[len(suffix)-6:],
		ClientePrefix:   "82" + suffix[len(suffix)-6:],
		PlacaPrefix:     "M" + suffix[len(suffix)-5:],
		Cidade:          cidade,
		DestinoPrefix:   "Destino Multi E2E " + suffix,
	})

	destinosIDs := make([]int64, 0, 3)
	for i, coords := range []struct {
		lat float64
		lon float64
	}{
		{-9.5584, -35.7777},
		{-9.6481, -35.7089},
		{-9.6658, -35.7353},
	} {
		destinosIDs = append(destinosIDs, createDestino(t, h.Router, h.AdminToken, map[string]any{
			"nome":      fmt.Sprintf("Destino Multi E2E %s %d", suffix, i),
			"rua":       "Av. Teste",
			"cidade":    "maceio",
			"latitude":  coords.lat,
			"longitude": coords.lon,
		}))
	}
	paradaID := createParada(t, h.Router, h.AdminToken, map[string]any{
		"nome":      "Parada Multi E2E",
		"latitude":  -9.7812,
		"longitude": -36.3501,
		"cidade":    cidade,
	})
	rotaInternaID := createRotaInterna(t, h.Router, h.AdminToken, map[string]any{
		"cidade":  cidade,
		"paradas": []map[string]any{{"parada_id": paradaID, "ordem": 1}},
	})
	createVeiculo(t, h.Router, h.AdminToken, map[string]any{
		"placa":       "M" + suffix[len(suffix)-5:] + "7",
		"modelo":      "Carro Multi E2E",
		"categoria":   "carro_7_lugares",
		"capacidade":  7,
		"cidade_base": cidade,
		"status":      "ativo",
	})
	createMotorista(t, h.Router, h.AdminToken, map[string]any{
		"nome":            "Motorista Multi E2E",
		"cpf":             "92" + suffix[len(suffix)-6:] + "00",
		"senha":           "senha123",
		"telefone":        "82999990000",
		"data_nasc":       "1980-05-20",
		"turno":           "NT",
		"cidade_trabalho": cidade,
		"residencia":      cidade,
		"foto":            "",
	})

	dataViagem := time.Now().AddDate(0, 0, 3).Format("2006-01-02")
	for i, destinoID := range destinosIDs {
		clienteID := createCliente(t, h.Router, h.AdminToken, map[string]any{
			"nome":      fmt.Sprintf("Cliente Multi %d", i),
			"cpf":       fmt.Sprintf("82%s%02d", suffix[len(suffix)-6:], i),
			"senha":     "senha123",
			"telefone":  "82999991111",
			"data_nasc": "2002-08-10",
			"foto":      "",
		})
		vinculoID := createVinculo(t, h.Router, h.AdminToken, clienteID, map[string]any{
			"tipo":            "estudante",
			"turno":           "NT",
			"destino_id":      destinoID,
			"rota_interna_id": rotaInternaID,
			"curso":           "Sistemas",
			"comprovante":     fmt.Sprintf("clientes/%d/e2e/comprovante.pdf", clienteID),
			"validade":        "2027-12-31",
			"horarios_fixos":  []int{1, 2, 3, 4, 5},
		})
		createReserva(t, h.Router, h.AdminToken, clienteID, vinculoID, map[string]any{
			"data_viagem": dataViagem,
			"turno":       "NT",
			"sentido":     "ida",
		})
	}
	createHorarioTurno(t, h.Router, h.AdminToken, map[string]any{
		"cidade":        cidade,
		"turno":         "NT",
		"horario_ida":   "17:00",
		"horario_volta": "22:00",
	})
	planejamento := doJSON[map[string]any](t, h.Router, http.MethodPost, "/api/v1/planejamentos/viagens", h.AdminToken, map[string]any{
		"data_viagem":     dataViagem,
		"turno":           "NT",
		"cidade":          cidade,
		"rota_interna_id": rotaInternaID,
	}, http.StatusCreated)
	viagemID := int64(planejamento["ciclos"].([]any)[0].(map[string]any)["viagens"].([]any)[0].(map[string]any)["id"].(float64))

	rota := doJSON[map[string]any](t, h.Router, http.MethodPost, fmt.Sprintf("/api/v1/viagens/%d/rota-dinamica/calcular", viagemID), h.AdminToken, nil, http.StatusCreated)
	destinos := rota["destinos"].([]any)
	if len(destinos) != 3 {
		t.Fatalf("expected 3 route destinations, got %d", len(destinos))
	}
	ordens := map[int]bool{}
	for _, item := range destinos {
		ordem := int(item.(map[string]any)["ordem"].(float64))
		ordens[ordem] = true
	}
	for i := 1; i <= 3; i++ {
		if !ordens[i] {
			t.Fatalf("expected route destination order %d in %+v", i, destinos)
		}
	}
	if osrmServer.Requests() != 1 {
		t.Fatalf("expected 1 OSRM request, got %d", osrmServer.Requests())
	}

	doStatus(t, h.Router, http.MethodPost, fmt.Sprintf("/api/v1/viagens/%d/rota-dinamica/calcular", viagemID), h.AdminToken, nil, http.StatusConflict)
	if osrmServer.Requests() != 2 {
		t.Fatalf("expected duplicate calculation to reach OSRM once more, got %d requests", osrmServer.Requests())
	}
}

func TestEndToEndCancelarReservaInvalidaRotaDinamica(t *testing.T) {
	if strings.TrimSpace(os.Getenv("E2E_DATABASE_URL")) == "" {
		t.Skip("set E2E_DATABASE_URL to run end-to-end tests")
	}

	osrmServer := newFakeOSRMServer(t)
	defer osrmServer.Close()
	h := newE2EHarness(t, e2eRouterOptions{OSRMBaseURL: osrmServer.URL})
	suffix := h.Suffix
	cidade := "e2e-invalida-rota-" + suffix
	h.Cleanup(e2eCleanupData{
		AdminEmail:      h.AdminEmail,
		MotoristaPrefix: "97" + suffix[len(suffix)-6:],
		ClientePrefix:   "79" + suffix[len(suffix)-6:],
		PlacaPrefix:     "I" + suffix[len(suffix)-5:],
		Cidade:          cidade,
		DestinoPrefix:   "Destino Invalida E2E " + suffix,
	})

	destinoID := createDestino(t, h.Router, h.AdminToken, map[string]any{
		"nome":      "Destino Invalida E2E " + suffix,
		"rua":       "Av. Teste",
		"cidade":    "maceio",
		"latitude":  -9.5584,
		"longitude": -35.7777,
	})
	paradaID := createParada(t, h.Router, h.AdminToken, map[string]any{
		"nome":      "Parada Invalida E2E",
		"latitude":  -9.7812,
		"longitude": -36.3501,
		"cidade":    cidade,
	})
	rotaInternaID := createRotaInterna(t, h.Router, h.AdminToken, map[string]any{
		"cidade":  cidade,
		"paradas": []map[string]any{{"parada_id": paradaID, "ordem": 1}},
	})
	createVeiculo(t, h.Router, h.AdminToken, map[string]any{
		"placa":       "I" + suffix[len(suffix)-5:] + "7",
		"modelo":      "Carro Invalida E2E",
		"categoria":   "carro_7_lugares",
		"capacidade":  7,
		"cidade_base": cidade,
		"status":      "ativo",
	})
	createMotorista(t, h.Router, h.AdminToken, map[string]any{
		"nome":            "Motorista Invalida E2E",
		"cpf":             "97" + suffix[len(suffix)-6:] + "00",
		"senha":           "senha123",
		"telefone":        "82999990000",
		"data_nasc":       "1980-05-20",
		"turno":           "NT",
		"cidade_trabalho": cidade,
		"residencia":      cidade,
		"foto":            "",
	})
	clienteID := createCliente(t, h.Router, h.AdminToken, map[string]any{
		"nome":      "Cliente Invalida E2E",
		"cpf":       "79" + suffix[len(suffix)-6:] + "00",
		"senha":     "senha123",
		"telefone":  "82999991111",
		"data_nasc": "2002-08-10",
		"foto":      "",
	})
	vinculoID := createVinculo(t, h.Router, h.AdminToken, clienteID, map[string]any{
		"tipo":            "estudante",
		"turno":           "NT",
		"destino_id":      destinoID,
		"rota_interna_id": rotaInternaID,
		"curso":           "Sistemas",
		"comprovante":     fmt.Sprintf("clientes/%d/e2e/comprovante.pdf", clienteID),
		"validade":        "2027-12-31",
		"horarios_fixos":  []int{1, 2, 3, 4, 5},
	})
	dataViagem := time.Now().AddDate(0, 0, 8).Format("2006-01-02")
	reservaID := createReserva(t, h.Router, h.AdminToken, clienteID, vinculoID, map[string]any{
		"data_viagem": dataViagem,
		"turno":       "NT",
		"sentido":     "ida",
	})
	createHorarioTurno(t, h.Router, h.AdminToken, map[string]any{
		"cidade":        cidade,
		"turno":         "NT",
		"horario_ida":   "17:00",
		"horario_volta": "22:00",
	})
	planejamento := doJSON[map[string]any](t, h.Router, http.MethodPost, "/api/v1/planejamentos/viagens", h.AdminToken, map[string]any{
		"data_viagem":     dataViagem,
		"turno":           "NT",
		"cidade":          cidade,
		"rota_interna_id": rotaInternaID,
	}, http.StatusCreated)
	viagemID := int64(planejamento["ciclos"].([]any)[0].(map[string]any)["viagens"].([]any)[0].(map[string]any)["id"].(float64))

	doJSON[map[string]any](t, h.Router, http.MethodPost, fmt.Sprintf("/api/v1/viagens/%d/rota-dinamica/calcular", viagemID), h.AdminToken, nil, http.StatusCreated)
	doJSON[map[string]any](t, h.Router, http.MethodGet, fmt.Sprintf("/api/v1/viagens/%d/rota-dinamica", viagemID), h.AdminToken, nil, http.StatusOK)

	cancelada := doJSON[map[string]any](t, h.Router, http.MethodPost, fmt.Sprintf("/api/v1/reservas/%d/cancelar", reservaID), h.AdminToken, nil, http.StatusOK)
	if cancelada["status"] != "cancelada" {
		t.Fatalf("expected reserva cancelada, got %v", cancelada["status"])
	}
	doStatus(t, h.Router, http.MethodGet, fmt.Sprintf("/api/v1/viagens/%d/rota-dinamica", viagemID), h.AdminToken, nil, http.StatusNotFound)
}

func TestEndToEndReservaCanceladaAntesDoPlanejamentoNaoEntraNaViagem(t *testing.T) {
	h := newE2EHarness(t, e2eRouterOptions{})
	suffix := h.Suffix
	cidade := "e2e-reserva-cancelada-planejamento-" + suffix
	h.Cleanup(e2eCleanupData{
		AdminEmail:      h.AdminEmail,
		MotoristaPrefix: "78" + suffix[len(suffix)-6:],
		ClientePrefix:   "78" + suffix[len(suffix)-6:],
		PlacaPrefix:     "N" + suffix[len(suffix)-5:],
		Cidade:          cidade,
		DestinoPrefix:   "Destino Reserva Cancelada E2E " + suffix,
	})

	destinoID := createDestino(t, h.Router, h.AdminToken, map[string]any{
		"nome":      "Destino Reserva Cancelada E2E " + suffix,
		"rua":       "Av. Teste",
		"cidade":    "maceio",
		"latitude":  -9.5584,
		"longitude": -35.7777,
	})
	paradaID := createParada(t, h.Router, h.AdminToken, map[string]any{
		"nome":      "Parada Reserva Cancelada E2E",
		"latitude":  -9.7812,
		"longitude": -36.3501,
		"cidade":    cidade,
	})
	rotaInternaID := createRotaInterna(t, h.Router, h.AdminToken, map[string]any{
		"cidade":  cidade,
		"paradas": []map[string]any{{"parada_id": paradaID, "ordem": 1}},
	})
	createVeiculo(t, h.Router, h.AdminToken, map[string]any{
		"placa":       "N" + suffix[len(suffix)-5:] + "7",
		"modelo":      "Carro Reserva Cancelada E2E",
		"categoria":   "carro_7_lugares",
		"capacidade":  7,
		"cidade_base": cidade,
		"status":      "ativo",
	})
	createMotorista(t, h.Router, h.AdminToken, map[string]any{
		"nome":            "Motorista Reserva Cancelada E2E",
		"cpf":             "78" + suffix[len(suffix)-6:] + "00",
		"senha":           "senha123",
		"telefone":        "82999990000",
		"data_nasc":       "1980-05-20",
		"turno":           "NT",
		"cidade_trabalho": cidade,
		"residencia":      cidade,
		"foto":            "",
	})

	dataViagem := time.Now().AddDate(0, 0, 9).Format("2006-01-02")
	var reservaConfirmadaID, reservaCanceladaID int64
	for i := 0; i < 2; i++ {
		clienteID := createCliente(t, h.Router, h.AdminToken, map[string]any{
			"nome":      fmt.Sprintf("Cliente Reserva Cancelada E2E %d", i),
			"cpf":       fmt.Sprintf("78%s%02d", suffix[len(suffix)-6:], i+1),
			"senha":     "senha123",
			"telefone":  "82999991111",
			"data_nasc": "2002-08-10",
			"foto":      "",
		})
		vinculoID := createVinculo(t, h.Router, h.AdminToken, clienteID, map[string]any{
			"tipo":            "estudante",
			"turno":           "NT",
			"destino_id":      destinoID,
			"rota_interna_id": rotaInternaID,
			"curso":           "Sistemas",
			"comprovante":     fmt.Sprintf("clientes/%d/e2e/comprovante.pdf", clienteID),
			"validade":        "2027-12-31",
			"horarios_fixos":  []int{1, 2, 3, 4, 5},
		})
		reservaID := createReserva(t, h.Router, h.AdminToken, clienteID, vinculoID, map[string]any{
			"data_viagem": dataViagem,
			"turno":       "NT",
			"sentido":     "ida",
		})
		if i == 0 {
			reservaConfirmadaID = reservaID
			continue
		}
		reservaCanceladaID = reservaID
		doJSON[map[string]any](t, h.Router, http.MethodPost, fmt.Sprintf("/api/v1/reservas/%d/cancelar", reservaCanceladaID), h.AdminToken, nil, http.StatusOK)
	}
	createHorarioTurno(t, h.Router, h.AdminToken, map[string]any{
		"cidade":        cidade,
		"turno":         "NT",
		"horario_ida":   "17:00",
		"horario_volta": "22:00",
	})

	planejamento := doJSON[map[string]any](t, h.Router, http.MethodPost, "/api/v1/planejamentos/viagens", h.AdminToken, map[string]any{
		"data_viagem":     dataViagem,
		"turno":           "NT",
		"cidade":          cidade,
		"rota_interna_id": rotaInternaID,
	}, http.StatusCreated)
	if int(planejamento["quantidade_reservas_ida"].(float64)) != 1 {
		t.Fatalf("expected only 1 confirmed ida reservation, got %v", planejamento["quantidade_reservas_ida"])
	}
	viagemID := int64(planejamento["ciclos"].([]any)[0].(map[string]any)["viagens"].([]any)[0].(map[string]any)["id"].(float64))
	reservasViagem := doJSON[[]map[string]any](t, h.Router, http.MethodGet, fmt.Sprintf("/api/v1/viagens/%d/reservas/", viagemID), h.AdminToken, nil, http.StatusOK)
	if len(reservasViagem) != 1 {
		t.Fatalf("expected 1 viagem_reserva, got %d", len(reservasViagem))
	}
	gotReservaID := int64(reservasViagem[0]["reserva_id"].(float64))
	if gotReservaID != reservaConfirmadaID {
		t.Fatalf("expected confirmed reserva %d on viagem, got %d", reservaConfirmadaID, gotReservaID)
	}
	if gotReservaID == reservaCanceladaID {
		t.Fatalf("canceled reserva %d should not enter viagem", reservaCanceladaID)
	}
}

func TestEndToEndFalhaOSRMNaoPersisteRotaDinamica(t *testing.T) {
	if strings.TrimSpace(os.Getenv("E2E_DATABASE_URL")) == "" {
		t.Skip("set E2E_DATABASE_URL to run end-to-end tests")
	}

	osrmServer := newFailingOSRMServer(t)
	defer osrmServer.Close()
	h := newE2EHarness(t, e2eRouterOptions{OSRMBaseURL: osrmServer.URL})
	suffix := h.Suffix
	cidade := "e2e-osrm-fail-" + suffix
	h.Cleanup(e2eCleanupData{
		AdminEmail:      h.AdminEmail,
		MotoristaPrefix: "77" + suffix[len(suffix)-6:],
		ClientePrefix:   "77" + suffix[len(suffix)-6:],
		PlacaPrefix:     "F" + suffix[len(suffix)-5:],
		Cidade:          cidade,
		DestinoPrefix:   "Destino Base E2E " + suffix,
	})

	rotaInternaID, dataViagem := setupPlanejamentoBase(t, h, planejamentoBaseOptions{
		Cidade:          cidade,
		Prefixo:         suffix,
		MotoristaPrefix: "77" + suffix[len(suffix)-6:],
		ClientePrefix:   "77" + suffix[len(suffix)-6:],
		PlacaPrefix:     "F" + suffix[len(suffix)-5:],
		CriarVeiculo:    true,
		CriarMotorista:  true,
		CriarHorario:    true,
	})

	planejamento := doJSON[map[string]any](t, h.Router, http.MethodPost, "/api/v1/planejamentos/viagens", h.AdminToken, map[string]any{
		"data_viagem":     dataViagem,
		"turno":           "NT",
		"cidade":          cidade,
		"rota_interna_id": rotaInternaID,
	}, http.StatusCreated)
	viagemID := int64(planejamento["ciclos"].([]any)[0].(map[string]any)["viagens"].([]any)[0].(map[string]any)["id"].(float64))

	doStatus(t, h.Router, http.MethodPost, fmt.Sprintf("/api/v1/viagens/%d/rota-dinamica/calcular", viagemID), h.AdminToken, nil, http.StatusInternalServerError)
	if osrmServer.Requests() != 1 {
		t.Fatalf("expected 1 failing OSRM request, got %d", osrmServer.Requests())
	}
	doStatus(t, h.Router, http.MethodGet, fmt.Sprintf("/api/v1/viagens/%d/rota-dinamica", viagemID), h.AdminToken, nil, http.StatusNotFound)
}

func TestEndToEndAutorizacaoPorDono(t *testing.T) {
	h := newE2EHarness(t, e2eRouterOptions{})
	suffix := h.Suffix
	cidade := "e2e-dono-" + suffix
	h.Cleanup(e2eCleanupData{
		AdminEmail:      h.AdminEmail,
		MotoristaPrefix: "93" + suffix[len(suffix)-6:],
		ClientePrefix:   "83" + suffix[len(suffix)-6:],
		PlacaPrefix:     "D" + suffix[len(suffix)-5:],
		Cidade:          cidade,
		DestinoPrefix:   "Destino Dono E2E " + suffix,
	})

	destinoID := createDestino(t, h.Router, h.AdminToken, map[string]any{
		"nome":      "Destino Dono E2E " + suffix,
		"rua":       "Av. Teste",
		"cidade":    "maceio",
		"latitude":  -9.5584,
		"longitude": -35.7777,
	})
	paradaID := createParada(t, h.Router, h.AdminToken, map[string]any{
		"nome":      "Parada Dono E2E",
		"latitude":  -9.7812,
		"longitude": -36.3501,
		"cidade":    cidade,
	})
	rotaInternaID := createRotaInterna(t, h.Router, h.AdminToken, map[string]any{
		"cidade":  cidade,
		"paradas": []map[string]any{{"parada_id": paradaID, "ordem": 1}},
	})
	createVeiculo(t, h.Router, h.AdminToken, map[string]any{
		"placa":       "D" + suffix[len(suffix)-5:] + "7",
		"modelo":      "Carro Dono E2E",
		"categoria":   "carro_7_lugares",
		"capacidade":  7,
		"cidade_base": cidade,
		"status":      "ativo",
	})
	motoristaCPF := "93" + suffix[len(suffix)-6:] + "00"
	outroMotoristaCPF := "93" + suffix[len(suffix)-6:] + "01"
	createMotorista(t, h.Router, h.AdminToken, map[string]any{
		"nome":            "Motorista Dono E2E",
		"cpf":             motoristaCPF,
		"senha":           "senha123",
		"telefone":        "82999990000",
		"data_nasc":       "1980-05-20",
		"turno":           "NT",
		"cidade_trabalho": cidade,
		"residencia":      cidade,
		"foto":            "",
	})
	createMotorista(t, h.Router, h.AdminToken, map[string]any{
		"nome":            "Outro Motorista Dono E2E",
		"cpf":             outroMotoristaCPF,
		"senha":           "senha123",
		"telefone":        "82999990001",
		"data_nasc":       "1980-05-20",
		"turno":           "NT",
		"cidade_trabalho": cidade + "-fora",
		"residencia":      cidade + "-fora",
		"foto":            "",
	})
	clienteID := createCliente(t, h.Router, h.AdminToken, map[string]any{
		"nome":      "Cliente Dono E2E",
		"cpf":       "83" + suffix[len(suffix)-6:] + "00",
		"senha":     "senha123",
		"telefone":  "82999991111",
		"data_nasc": "2002-08-10",
		"foto":      "",
	})
	outroClienteID := createCliente(t, h.Router, h.AdminToken, map[string]any{
		"nome":      "Outro Cliente Dono E2E",
		"cpf":       "83" + suffix[len(suffix)-6:] + "01",
		"senha":     "senha123",
		"telefone":  "82999991112",
		"data_nasc": "2002-08-10",
		"foto":      "",
	})
	vinculoID := createVinculo(t, h.Router, h.AdminToken, clienteID, map[string]any{
		"tipo":            "estudante",
		"turno":           "NT",
		"destino_id":      destinoID,
		"rota_interna_id": rotaInternaID,
		"curso":           "Sistemas",
		"comprovante":     fmt.Sprintf("clientes/%d/e2e/comprovante.pdf", clienteID),
		"validade":        "2027-12-31",
		"horarios_fixos":  []int{1, 2, 3, 4, 5},
	})
	dataViagem := time.Now().AddDate(0, 0, 4).Format("2006-01-02")
	createReserva(t, h.Router, h.AdminToken, clienteID, vinculoID, map[string]any{
		"data_viagem": dataViagem,
		"turno":       "NT",
		"sentido":     "ida",
	})
	createHorarioTurno(t, h.Router, h.AdminToken, map[string]any{
		"cidade":        cidade,
		"turno":         "NT",
		"horario_ida":   "17:00",
		"horario_volta": "22:00",
	})
	planejamento := doJSON[map[string]any](t, h.Router, http.MethodPost, "/api/v1/planejamentos/viagens", h.AdminToken, map[string]any{
		"data_viagem":     dataViagem,
		"turno":           "NT",
		"cidade":          cidade,
		"rota_interna_id": rotaInternaID,
	}, http.StatusCreated)
	viagemID := int64(planejamento["ciclos"].([]any)[0].(map[string]any)["viagens"].([]any)[0].(map[string]any)["id"].(float64))
	motoristaToken := loginMotorista(t, h.Router, motoristaCPF, "senha123")
	outroMotoristaToken := loginMotorista(t, h.Router, outroMotoristaCPF, "senha123")
	outroClienteToken := loginCliente(t, h.Router, "83"+suffix[len(suffix)-6:]+"01", "senha123")

	doStatus(t, h.Router, http.MethodPut, fmt.Sprintf("/api/v1/viagens/%d/localizacao", viagemID), motoristaToken, map[string]any{
		"latitude":        -9.7812,
		"longitude":       -36.3501,
		"velocidade_kmh":  42.5,
		"direcao_graus":   180,
		"precisao_metros": 8,
	}, http.StatusForbidden)
	doJSON[map[string]any](t, h.Router, http.MethodPost, fmt.Sprintf("/api/v1/viagens/%d/iniciar", viagemID), motoristaToken, nil, http.StatusOK)
	doStatus(t, h.Router, http.MethodPut, fmt.Sprintf("/api/v1/viagens/%d/localizacao", viagemID), outroMotoristaToken, map[string]any{
		"latitude":        -9.7812,
		"longitude":       -36.3501,
		"velocidade_kmh":  42.5,
		"direcao_graus":   180,
		"precisao_metros": 8,
	}, http.StatusForbidden)

	doJSON[map[string]any](t, h.Router, http.MethodPut, fmt.Sprintf("/api/v1/viagens/%d/localizacao", viagemID), motoristaToken, map[string]any{
		"latitude":        -9.7812,
		"longitude":       -36.3501,
		"velocidade_kmh":  42.5,
		"direcao_graus":   180,
		"precisao_metros": 8,
	}, http.StatusOK)
	doStatus(t, h.Router, http.MethodGet, fmt.Sprintf("/api/v1/viagens/%d/localizacao", viagemID), outroClienteToken, nil, http.StatusForbidden)

	if outroClienteID <= 0 {
		t.Fatalf("expected other cliente to be created")
	}
}

func TestEndToEndReservaDuplicadaECancelada(t *testing.T) {
	h := newE2EHarness(t, e2eRouterOptions{})
	suffix := h.Suffix
	cidade := "e2e-reserva-" + suffix
	h.Cleanup(e2eCleanupData{
		AdminEmail:    h.AdminEmail,
		ClientePrefix: "84" + suffix[len(suffix)-6:],
		Cidade:        cidade,
		DestinoPrefix: "Destino Reserva E2E " + suffix,
	})

	destinoID := createDestino(t, h.Router, h.AdminToken, map[string]any{
		"nome":      "Destino Reserva E2E " + suffix,
		"rua":       "Av. Teste",
		"cidade":    "maceio",
		"latitude":  -9.5584,
		"longitude": -35.7777,
	})
	paradaID := createParada(t, h.Router, h.AdminToken, map[string]any{
		"nome":      "Parada Reserva E2E",
		"latitude":  -9.7812,
		"longitude": -36.3501,
		"cidade":    cidade,
	})
	rotaInternaID := createRotaInterna(t, h.Router, h.AdminToken, map[string]any{
		"cidade":  cidade,
		"paradas": []map[string]any{{"parada_id": paradaID, "ordem": 1}},
	})
	clienteID := createCliente(t, h.Router, h.AdminToken, map[string]any{
		"nome":      "Cliente Reserva E2E",
		"cpf":       "84" + suffix[len(suffix)-6:] + "00",
		"senha":     "senha123",
		"telefone":  "82999991111",
		"data_nasc": "2002-08-10",
		"foto":      "",
	})
	outroClienteID := createCliente(t, h.Router, h.AdminToken, map[string]any{
		"nome":      "Outro Cliente Reserva E2E",
		"cpf":       "84" + suffix[len(suffix)-6:] + "01",
		"senha":     "senha123",
		"telefone":  "82999991112",
		"data_nasc": "2002-08-10",
		"foto":      "",
	})
	vinculoID := createVinculo(t, h.Router, h.AdminToken, clienteID, map[string]any{
		"tipo":            "estudante",
		"turno":           "NT",
		"destino_id":      destinoID,
		"rota_interna_id": rotaInternaID,
		"curso":           "Sistemas",
		"comprovante":     fmt.Sprintf("clientes/%d/e2e/comprovante.pdf", clienteID),
		"validade":        "2027-12-31",
		"horarios_fixos":  []int{1, 2, 3, 4, 5},
	})
	body := map[string]any{
		"data_viagem": time.Now().AddDate(0, 0, 5).Format("2006-01-02"),
		"turno":       "NT",
		"sentido":     "ida",
	}

	reservaID := createReserva(t, h.Router, h.AdminToken, clienteID, vinculoID, body)
	doStatus(t, h.Router, http.MethodPost, fmt.Sprintf("/api/v1/clientes/%d/vinculos/%d/reservas/", clienteID, vinculoID), h.AdminToken, body, http.StatusConflict)
	outroClienteToken := loginCliente(t, h.Router, "84"+suffix[len(suffix)-6:]+"01", "senha123")
	doStatus(t, h.Router, http.MethodPost, fmt.Sprintf("/api/v1/reservas/%d/cancelar", reservaID), outroClienteToken, nil, http.StatusForbidden)

	clienteToken := loginCliente(t, h.Router, "84"+suffix[len(suffix)-6:]+"00", "senha123")
	cancelada := doJSON[map[string]any](t, h.Router, http.MethodPost, fmt.Sprintf("/api/v1/reservas/%d/cancelar", reservaID), clienteToken, nil, http.StatusOK)
	if cancelada["status"] != "cancelada" {
		t.Fatalf("expected reserva cancelada, got %v", cancelada["status"])
	}
	if int64(cancelada["cliente_id"].(float64)) != clienteID {
		t.Fatalf("expected owner cliente_id %d, got %v", clienteID, cancelada["cliente_id"])
	}
	if outroClienteID <= 0 {
		t.Fatalf("expected other cliente to be created")
	}

	nova := doJSON[map[string]any](t, h.Router, http.MethodPost, fmt.Sprintf("/api/v1/clientes/%d/vinculos/%d/reservas/", clienteID, vinculoID), h.AdminToken, body, http.StatusCreated)
	if nova["status"] != "confirmada" {
		t.Fatalf("expected recreated reserva confirmada, got %v", nova["status"])
	}
}

func TestEndToEndPlanejamentoErrosSemRecursos(t *testing.T) {
	t.Run("sem horario configurado", func(t *testing.T) {
		h := newE2EHarness(t, e2eRouterOptions{})
		suffix := h.Suffix
		cidade := "e2e-sem-horario-" + suffix
		h.Cleanup(e2eCleanupData{
			AdminEmail:      h.AdminEmail,
			MotoristaPrefix: "95" + suffix[len(suffix)-6:],
			ClientePrefix:   "85" + suffix[len(suffix)-6:],
			PlacaPrefix:     "H" + suffix[len(suffix)-5:],
			Cidade:          cidade,
			DestinoPrefix:   "Destino Base E2E " + suffix,
		})

		rotaInternaID, dataViagem := setupPlanejamentoBase(t, h, planejamentoBaseOptions{
			Cidade:          cidade,
			Prefixo:         suffix,
			MotoristaPrefix: "95" + suffix[len(suffix)-6:],
			ClientePrefix:   "85" + suffix[len(suffix)-6:],
			PlacaPrefix:     "H" + suffix[len(suffix)-5:],
			CriarVeiculo:    true,
			CriarMotorista:  true,
		})

		doStatus(t, h.Router, http.MethodPost, "/api/v1/planejamentos/viagens", h.AdminToken, map[string]any{
			"data_viagem":     dataViagem,
			"turno":           "NT",
			"cidade":          cidade,
			"rota_interna_id": rotaInternaID,
		}, http.StatusNotFound)
	})

	t.Run("sem veiculo disponivel", func(t *testing.T) {
		h := newE2EHarness(t, e2eRouterOptions{})
		suffix := h.Suffix
		cidade := "e2e-sem-veiculo-" + suffix
		h.Cleanup(e2eCleanupData{
			AdminEmail:      h.AdminEmail,
			MotoristaPrefix: "96" + suffix[len(suffix)-6:],
			ClientePrefix:   "86" + suffix[len(suffix)-6:],
			Cidade:          cidade,
			DestinoPrefix:   "Destino Base E2E " + suffix,
		})

		rotaInternaID, dataViagem := setupPlanejamentoBase(t, h, planejamentoBaseOptions{
			Cidade:          cidade,
			Prefixo:         suffix,
			MotoristaPrefix: "96" + suffix[len(suffix)-6:],
			ClientePrefix:   "86" + suffix[len(suffix)-6:],
			CriarMotorista:  true,
			CriarHorario:    true,
		})

		doStatus(t, h.Router, http.MethodPost, "/api/v1/planejamentos/viagens", h.AdminToken, map[string]any{
			"data_viagem":     dataViagem,
			"turno":           "NT",
			"cidade":          cidade,
			"rota_interna_id": rotaInternaID,
		}, http.StatusNotFound)
	})

	t.Run("sem motorista disponivel", func(t *testing.T) {
		h := newE2EHarness(t, e2eRouterOptions{})
		suffix := h.Suffix
		cidade := "e2e-sem-motorista-" + suffix
		h.Cleanup(e2eCleanupData{
			AdminEmail:    h.AdminEmail,
			ClientePrefix: "87" + suffix[len(suffix)-6:],
			PlacaPrefix:   "S" + suffix[len(suffix)-5:],
			Cidade:        cidade,
			DestinoPrefix: "Destino Base E2E " + suffix,
		})

		rotaInternaID, dataViagem := setupPlanejamentoBase(t, h, planejamentoBaseOptions{
			Cidade:        cidade,
			Prefixo:       suffix,
			ClientePrefix: "87" + suffix[len(suffix)-6:],
			PlacaPrefix:   "S" + suffix[len(suffix)-5:],
			CriarVeiculo:  true,
			CriarHorario:  true,
		})

		doStatus(t, h.Router, http.MethodPost, "/api/v1/planejamentos/viagens", h.AdminToken, map[string]any{
			"data_viagem":     dataViagem,
			"turno":           "NT",
			"cidade":          cidade,
			"rota_interna_id": rotaInternaID,
		}, http.StatusNotFound)
	})
}

func TestEndToEndViagemCanceladaNaoInicia(t *testing.T) {
	h := newE2EHarness(t, e2eRouterOptions{})
	suffix := h.Suffix
	cidade := "e2e-cancelar-viagem-" + suffix
	h.Cleanup(e2eCleanupData{
		AdminEmail:      h.AdminEmail,
		MotoristaPrefix: "98" + suffix[len(suffix)-6:],
		ClientePrefix:   "88" + suffix[len(suffix)-6:],
		PlacaPrefix:     "V" + suffix[len(suffix)-5:],
		Cidade:          cidade,
		DestinoPrefix:   "Destino Base E2E " + suffix,
	})

	rotaInternaID, dataViagem := setupPlanejamentoBase(t, h, planejamentoBaseOptions{
		Cidade:          cidade,
		Prefixo:         suffix,
		MotoristaPrefix: "98" + suffix[len(suffix)-6:],
		ClientePrefix:   "88" + suffix[len(suffix)-6:],
		PlacaPrefix:     "V" + suffix[len(suffix)-5:],
		CriarVeiculo:    true,
		CriarMotorista:  true,
		CriarHorario:    true,
	})

	planejamento := doJSON[map[string]any](t, h.Router, http.MethodPost, "/api/v1/planejamentos/viagens", h.AdminToken, map[string]any{
		"data_viagem":     dataViagem,
		"turno":           "NT",
		"cidade":          cidade,
		"rota_interna_id": rotaInternaID,
	}, http.StatusCreated)
	viagemID := int64(planejamento["ciclos"].([]any)[0].(map[string]any)["viagens"].([]any)[0].(map[string]any)["id"].(float64))

	cancelada := doJSON[map[string]any](t, h.Router, http.MethodPost, fmt.Sprintf("/api/v1/viagens/%d/cancelar", viagemID), h.AdminToken, nil, http.StatusOK)
	if cancelada["status"] != "cancelada" {
		t.Fatalf("expected viagem cancelada, got %v", cancelada["status"])
	}

	doStatus(t, h.Router, http.MethodPost, fmt.Sprintf("/api/v1/viagens/%d/iniciar", viagemID), h.AdminToken, nil, http.StatusConflict)
}

type planejamentoBaseOptions struct {
	Cidade          string
	Prefixo         string
	MotoristaPrefix string
	ClientePrefix   string
	PlacaPrefix     string
	CriarVeiculo    bool
	CriarMotorista  bool
	CriarHorario    bool
}

func setupPlanejamentoBase(t *testing.T, h *e2eHarness, options planejamentoBaseOptions) (int64, string) {
	t.Helper()

	destinoID := createDestino(t, h.Router, h.AdminToken, map[string]any{
		"nome":      "Destino Base E2E " + options.Prefixo,
		"rua":       "Av. Teste",
		"cidade":    "maceio",
		"latitude":  -9.5584,
		"longitude": -35.7777,
	})
	paradaID := createParada(t, h.Router, h.AdminToken, map[string]any{
		"nome":      "Parada Base E2E",
		"latitude":  -9.7812,
		"longitude": -36.3501,
		"cidade":    options.Cidade,
	})
	rotaInternaID := createRotaInterna(t, h.Router, h.AdminToken, map[string]any{
		"cidade":  options.Cidade,
		"paradas": []map[string]any{{"parada_id": paradaID, "ordem": 1}},
	})

	if options.CriarVeiculo {
		createVeiculo(t, h.Router, h.AdminToken, map[string]any{
			"placa":       options.PlacaPrefix + "7",
			"modelo":      "Carro Base E2E",
			"categoria":   "carro_7_lugares",
			"capacidade":  7,
			"cidade_base": options.Cidade,
			"status":      "ativo",
		})
	}
	if options.CriarMotorista {
		createMotorista(t, h.Router, h.AdminToken, map[string]any{
			"nome":            "Motorista Base E2E",
			"cpf":             options.MotoristaPrefix + "00",
			"senha":           "senha123",
			"telefone":        "82999990000",
			"data_nasc":       "1980-05-20",
			"turno":           "NT",
			"cidade_trabalho": options.Cidade,
			"residencia":      options.Cidade,
			"foto":            "",
		})
	}

	clienteID := createCliente(t, h.Router, h.AdminToken, map[string]any{
		"nome":      "Cliente Base E2E",
		"cpf":       options.ClientePrefix + "00",
		"senha":     "senha123",
		"telefone":  "82999991111",
		"data_nasc": "2002-08-10",
		"foto":      "",
	})
	vinculoID := createVinculo(t, h.Router, h.AdminToken, clienteID, map[string]any{
		"tipo":            "estudante",
		"turno":           "NT",
		"destino_id":      destinoID,
		"rota_interna_id": rotaInternaID,
		"curso":           "Sistemas",
		"comprovante":     fmt.Sprintf("clientes/%d/e2e/comprovante.pdf", clienteID),
		"validade":        "2027-12-31",
		"horarios_fixos":  []int{1, 2, 3, 4, 5},
	})

	dataViagem := time.Now().AddDate(0, 0, 6).Format("2006-01-02")
	createReserva(t, h.Router, h.AdminToken, clienteID, vinculoID, map[string]any{
		"data_viagem": dataViagem,
		"turno":       "NT",
		"sentido":     "ida",
	})
	if options.CriarHorario {
		createHorarioTurno(t, h.Router, h.AdminToken, map[string]any{
			"cidade":        options.Cidade,
			"turno":         "NT",
			"horario_ida":   "17:00",
			"horario_volta": "22:00",
		})
	}

	return rotaInternaID, dataViagem
}

type e2eRouterOptions struct {
	OSRMBaseURL   string
	StorageConfig storage.SupabaseConfig
}

type e2eHarness struct {
	T          *testing.T
	Pool       *pgxpool.Pool
	Router     http.Handler
	AuthSvc    *auth.AuthService
	Suffix     string
	AdminEmail string
	AdminToken string
}

func newE2EHarness(t *testing.T, options e2eRouterOptions) *e2eHarness {
	t.Helper()

	dbURL := strings.TrimSpace(os.Getenv("E2E_DATABASE_URL"))
	if dbURL == "" {
		t.Skip("set E2E_DATABASE_URL to run end-to-end tests")
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect e2e database: %v", err)
	}
	t.Cleanup(pool.Close)

	authSvc := auth.NewAuthService(crypto.NewBcryptHasher(crypto.DefaultCost), "e2e-secret")
	router := buildE2ERouter(pool, authSvc, options)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	adminEmail := "admin-e2e-" + suffix + "@bondrota.test"
	adminPassword := "admin123"
	seedAdmin(t, ctx, pool, adminEmail, adminPassword)

	h := &e2eHarness{
		T:          t,
		Pool:       pool,
		Router:     router,
		AuthSvc:    authSvc,
		Suffix:     suffix,
		AdminEmail: adminEmail,
		AdminToken: loginAdmin(t, router, adminEmail, adminPassword),
	}
	t.Cleanup(func() {
		cleanupE2EData(context.Background(), t, pool, e2eCleanupData{
			AdminEmail: adminEmail,
		})
	})
	return h
}

func (h *e2eHarness) Cleanup(data e2eCleanupData) {
	h.T.Helper()
	h.T.Cleanup(func() {
		cleanupE2EData(context.Background(), h.T, h.Pool, data)
	})
}

func buildE2ERouter(pool *pgxpool.Pool, authSvc *auth.AuthService, options e2eRouterOptions) http.Handler {
	adminStore := admin.NewAdminStore(pool)
	adminSvc := admin.NewAdminService(adminStore, authSvc)

	veiculoStore := veiculos.NewVeiculoStore(pool)
	alocacaoVeiculoStore := veiculos.NewAlocacaoVeiculoStore(pool)
	alocacaoVeiculoSvc := veiculos.NewAlocacaoService(alocacaoVeiculoStore)

	destinoStore := destinos.NewDestinoStore(pool)
	paradaStore := paradas.NewParadaStore(pool)

	rotaInternaStore := rotasinternas.NewRotaInternaStore(pool)
	rotaInternaSvc := rotasinternas.NewRotaInternaService(rotaInternaStore)

	motoristaStore := motoristas.NewMotoristaStore(pool)
	alocacaoMotoristaStore := motoristas.NewAlocacaoMotoristaStore(pool)
	alocacaoMotoristaSvc := motoristas.NewAlocacaoService(alocacaoMotoristaStore)
	motoristaSvc := motoristas.NewMotoristaService(motoristaStore, authSvc)

	clienteStore := clientes.NewClienteStore(pool)
	clienteSvc := clientes.NewClienteService(clienteStore, authSvc)
	vinculoStore := clientes.NewVinculoStore(pool)
	vinculoSvc := clientes.NewVinculoService(vinculoStore)

	calculadorRotaDinamicaStore := rotasdinamicas.NewCalculadorRotaDinamicaStore(pool)
	rotaDinamicaInvalidator := rotasdinamicas.NewInvalidadorRotaDinamicaService(calculadorRotaDinamicaStore, rotasdinamicas.DefaultJanelaBloqueioRotaDinamica)

	reservaStore := reservas.NewReservaStore(pool)
	reservaSvc := reservas.NewReservaService(reservaStore, rotaDinamicaInvalidator)

	cicloViagemStore := viagens.NewCicloViagemStore(pool)
	horarioTurnoStore := viagens.NewHorarioTurnoViagemStore(pool)
	horarioTurnoSvc := viagens.NewHorarioTurnoViagemService(horarioTurnoStore)
	planejamentoSvc := viagens.NewPlanejamentoService(cicloViagemStore, horarioTurnoStore, alocacaoVeiculoSvc, alocacaoMotoristaSvc)

	viagemStore := viagens.NewViagemStore(pool)
	viagemSvc := viagens.NewViagemService(viagemStore)
	viagemReservaStore := viagens.NewViagemReservaStore(pool)
	presencaSvc := viagens.NewPresencaService(viagemReservaStore)
	viagemLocalizacaoStore := viagens.NewViagemLocalizacaoStore(pool)
	viagemLocalizacaoSvc := viagens.NewViagemLocalizacaoService(viagemLocalizacaoStore)

	rotaDinamicaStore := rotasdinamicas.NewRotaDinamicaStore(pool)
	rotaDinamicaSvc := rotasdinamicas.NewRotaDinamicaService(rotaDinamicaStore)
	calculadorRotaDinamicaSvc := rotasdinamicas.NewCalculadorRotaDinamicaService(
		calculadorRotaDinamicaStore,
		rotaDinamicaSvc,
		geo.NewOSRMClient(nil, options.OSRMBaseURL),
		geo.NewOtimizadorRota(),
	)

	storageHandler := storage.NewHandler(storage.NewService(storage.NewSupabaseClient(options.StorageConfig, nil)))

	handlers := server.Handlers{
		AdminHandler:        admin.NewAdminHandler(adminSvc),
		VeiculoHandler:      veiculos.NewVeiculoHandler(veiculoStore),
		DestinoHandler:      destinos.NewDestinoHandler(destinoStore),
		ParadaHandler:       paradas.NewParadaHandler(paradaStore),
		RotaInternaHandler:  rotasinternas.NewRotaInternaHandler(rotaInternaSvc),
		MotoristaHandler:    motoristas.NewMotoristaHandler(motoristaSvc),
		ClienteHandler:      clientes.NewClienteHandler(clienteSvc),
		VinculoHandler:      clientes.NewVinculoHandler(vinculoSvc),
		ReservaHandler:      reservas.NewReservaHandler(reservaSvc),
		ViagemHandler:       viagens.NewViagemHandler(viagemSvc, presencaSvc, viagemLocalizacaoSvc),
		PlanejamentoHandler: viagens.NewPlanejamentoHandler(planejamentoSvc),
		HorarioTurnoHandler: viagens.NewHorarioTurnoViagemHandler(horarioTurnoSvc),
		RotaDinamicaHandler: rotasdinamicas.NewRotaDinamicaHandler(rotaDinamicaSvc, calculadorRotaDinamicaSvc),
		StorageHandler:      storageHandler,
	}

	apiRouter := chi.NewRouter()
	server.NewServer(handlers, authSvc).RegisterRoutes(apiRouter)

	root := chi.NewRouter()
	root.Mount("/api/v1", apiRouter)
	return root
}

func seedAdmin(t *testing.T, ctx context.Context, pool *pgxpool.Pool, email, password string) {
	t.Helper()
	hash, err := crypto.NewBcryptHasher(crypto.DefaultCost).Hash(password)
	if err != nil {
		t.Fatalf("hash admin password: %v", err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO administrador (email, senha)
		VALUES ($1, $2)
		ON CONFLICT (email) DO UPDATE SET senha = EXCLUDED.senha
	`, email, hash)
	if err != nil {
		t.Fatalf("seed admin: %v", err)
	}
}

type e2eCleanupData struct {
	AdminEmail      string
	MotoristaCPF    string
	MotoristaPrefix string
	ClienteCPF      string
	ClientePrefix   string
	Placa           string
	PlacaPrefix     string
	Cidade          string
	DestinoNome     string
	DestinoPrefix   string
}

func cleanupE2EData(ctx context.Context, t *testing.T, pool *pgxpool.Pool, data e2eCleanupData) {
	t.Helper()

	statements := []struct {
		query string
		args  []any
	}{
		{query: `DELETE FROM ciclos_viagem WHERE cidade = $1`, args: []any{data.Cidade}},
		{query: `DELETE FROM reservas WHERE cidade = $1`, args: []any{data.Cidade}},
		{query: `DELETE FROM cliente_vinculos WHERE cliente_id IN (SELECT id FROM clientes WHERE cpf = $1)`, args: []any{data.ClienteCPF}},
		{query: `DELETE FROM cliente_vinculos WHERE cliente_id IN (SELECT id FROM clientes WHERE cpf LIKE $1 || '%')`, args: []any{data.ClientePrefix}},
		{query: `DELETE FROM clientes WHERE cpf = $1`, args: []any{data.ClienteCPF}},
		{query: `DELETE FROM clientes WHERE cpf LIKE $1 || '%'`, args: []any{data.ClientePrefix}},
		{query: `DELETE FROM horarios_turno_viagem WHERE cidade = $1`, args: []any{data.Cidade}},
		{query: `DELETE FROM rotas_internas WHERE cidade = $1`, args: []any{data.Cidade}},
		{query: `DELETE FROM paradas WHERE cidade = $1`, args: []any{data.Cidade}},
		{query: `DELETE FROM destinos WHERE nome = $1`, args: []any{data.DestinoNome}},
		{query: `DELETE FROM destinos WHERE nome LIKE $1 || '%'`, args: []any{data.DestinoPrefix}},
		{query: `DELETE FROM veiculos WHERE placa = $1`, args: []any{data.Placa}},
		{query: `DELETE FROM veiculos WHERE placa LIKE $1 || '%'`, args: []any{data.PlacaPrefix}},
		{query: `DELETE FROM motoristas WHERE cpf = $1`, args: []any{data.MotoristaCPF}},
		{query: `DELETE FROM motoristas WHERE cpf LIKE $1 || '%'`, args: []any{data.MotoristaPrefix}},
		{query: `DELETE FROM administrador WHERE email = $1`, args: []any{data.AdminEmail}},
	}

	for _, statement := range statements {
		if len(statement.args) > 0 {
			if value, ok := statement.args[0].(string); ok && value == "" {
				continue
			}
		}
		if _, err := pool.Exec(ctx, statement.query, statement.args...); err != nil {
			t.Logf("cleanup failed for %q: %v", statement.query, err)
		}
	}
}

func loginAdmin(t *testing.T, router http.Handler, email, senha string) string {
	t.Helper()
	resp := doJSON[map[string]any](t, router, http.MethodPost, "/api/v1/admin/login", "", map[string]any{
		"email": email,
		"senha": senha,
	}, http.StatusOK)
	return resp["token"].(string)
}

func loginMotorista(t *testing.T, router http.Handler, cpf, senha string) string {
	t.Helper()
	resp := doJSON[map[string]any](t, router, http.MethodPost, "/api/v1/motoristas/login", "", map[string]any{
		"cpf":   cpf,
		"senha": senha,
	}, http.StatusOK)
	return resp["token"].(string)
}

func loginCliente(t *testing.T, router http.Handler, cpf, senha string) string {
	t.Helper()
	resp := doJSON[map[string]any](t, router, http.MethodPost, "/api/v1/clientes/login", "", map[string]any{
		"cpf":   cpf,
		"senha": senha,
	}, http.StatusOK)
	return resp["token"].(string)
}

func createDestino(t *testing.T, router http.Handler, token string, body map[string]any) int64 {
	t.Helper()
	resp := doJSON[map[string]any](t, router, http.MethodPost, "/api/v1/destinos/", token, body, http.StatusCreated)
	return int64(resp["id"].(float64))
}

func createParada(t *testing.T, router http.Handler, token string, body map[string]any) int64 {
	t.Helper()
	resp := doJSON[map[string]any](t, router, http.MethodPost, "/api/v1/paradas/", token, body, http.StatusCreated)
	return int64(resp["id"].(float64))
}

func createRotaInterna(t *testing.T, router http.Handler, token string, body map[string]any) int64 {
	t.Helper()
	resp := doJSON[map[string]any](t, router, http.MethodPost, "/api/v1/rotas-internas/", token, body, http.StatusCreated)
	return int64(resp["id"].(float64))
}

func createVeiculo(t *testing.T, router http.Handler, token string, body map[string]any) int64 {
	t.Helper()
	resp := doJSON[map[string]any](t, router, http.MethodPost, "/api/v1/veiculos/", token, body, http.StatusCreated)
	return int64(resp["id"].(float64))
}

func createMotorista(t *testing.T, router http.Handler, token string, body map[string]any) int64 {
	t.Helper()
	resp := doJSON[map[string]any](t, router, http.MethodPost, "/api/v1/motoristas/", token, body, http.StatusCreated)
	return int64(resp["id"].(float64))
}

func createCliente(t *testing.T, router http.Handler, token string, body map[string]any) int64 {
	t.Helper()
	resp := doJSON[map[string]any](t, router, http.MethodPost, "/api/v1/clientes/", token, body, http.StatusCreated)
	return int64(resp["id"].(float64))
}

func createVinculo(t *testing.T, router http.Handler, token string, clienteID int64, body map[string]any) int64 {
	t.Helper()
	resp := doJSON[map[string]any](t, router, http.MethodPost, fmt.Sprintf("/api/v1/clientes/%d/vinculos/", clienteID), token, body, http.StatusCreated)
	return int64(resp["id"].(float64))
}

func createReserva(t *testing.T, router http.Handler, token string, clienteID, vinculoID int64, body map[string]any) int64 {
	t.Helper()
	resp := doJSON[map[string]any](t, router, http.MethodPost, fmt.Sprintf("/api/v1/clientes/%d/vinculos/%d/reservas/", clienteID, vinculoID), token, body, http.StatusCreated)
	return int64(resp["id"].(float64))
}

func createHorarioTurno(t *testing.T, router http.Handler, token string, body map[string]any) int64 {
	t.Helper()
	resp := doJSON[map[string]any](t, router, http.MethodPost, "/api/v1/horarios-turno-viagem/", token, body, http.StatusCreated)
	return int64(resp["id"].(float64))
}

func doStatus(t *testing.T, router http.Handler, method, path, token string, body any, wantStatus int) {
	t.Helper()

	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		reader = bytes.NewReader(data)
	}

	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != wantStatus {
		t.Fatalf("%s %s: want status %d, got %d: %s", method, path, wantStatus, rr.Code, rr.Body.String())
	}
}

func doJSON[T any](t *testing.T, router http.Handler, method, path, token string, body any, wantStatus int) T {
	t.Helper()

	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		reader = bytes.NewReader(data)
	}

	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != wantStatus {
		t.Fatalf("%s %s: want status %d, got %d: %s", method, path, wantStatus, rr.Code, rr.Body.String())
	}

	var out T
	if rr.Body.Len() == 0 {
		return out
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("%s %s: decode response: %v\nbody: %s", method, path, err, rr.Body.String())
	}
	return out
}

type fakeOSRMServer struct {
	*httptest.Server
	requests int
}

func newFakeOSRMServer(t *testing.T) *fakeOSRMServer {
	t.Helper()
	fake := &fakeOSRMServer{}
	fake.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("unexpected OSRM method: %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if !strings.HasPrefix(r.URL.Path, "/route/v1/driving/") {
			t.Errorf("unexpected OSRM path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		fake.requests++
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{
			"code": "Ok",
			"routes": [
				{
					"distance": 12345.6,
					"duration": 789.1,
					"geometry": {
						"type": "LineString",
						"coordinates": [
							[-36.3501, -9.7812],
							[-35.7777, -9.5584]
						]
					}
				}
			]
		}`)
	}))
	return fake
}

func (s *fakeOSRMServer) Requests() int {
	return s.requests
}

type failingOSRMServer struct {
	*httptest.Server
	requests int
}

func newFailingOSRMServer(t *testing.T) *failingOSRMServer {
	t.Helper()
	fake := &failingOSRMServer{}
	fake.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("unexpected OSRM method: %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if !strings.HasPrefix(r.URL.Path, "/route/v1/driving/") {
			t.Errorf("unexpected OSRM path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		fake.requests++
		http.Error(w, "osrm unavailable", http.StatusInternalServerError)
	}))
	return fake
}

func (s *failingOSRMServer) Requests() int {
	return s.requests
}

type fakeSupabaseStorageServer struct {
	*httptest.Server
	signUploadRequests   int
	uploadRequests       int
	signDownloadRequests int
}

func newFakeSupabaseStorageServer(t *testing.T) *fakeSupabaseStorageServer {
	t.Helper()
	fake := &fakeSupabaseStorageServer{}
	var baseURL string
	fake.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/storage/v1/object/upload/sign/fotos/clientes/1/foto.png"):
			fake.signUploadRequests++
			if r.Header.Get("Authorization") != "Bearer service-key" {
				t.Errorf("missing service key authorization header")
			}
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, fmt.Sprintf(`{
				"signedURL": %q,
				"path": "clientes/1/foto.png",
				"token": "upload-token"
			}`, baseURL+"/upload-target"))
		case r.Method == http.MethodPut && r.URL.Path == "/upload-target":
			fake.uploadRequests++
			if r.Header.Get("Content-Type") != "image/png" {
				t.Errorf("unexpected upload content type: %s", r.Header.Get("Content-Type"))
			}
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/storage/v1/object/sign/fotos/clientes/1/foto.png"):
			fake.signDownloadRequests++
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, fmt.Sprintf(`{
				"signedURL": %q
			}`, baseURL+"/download-target"))
		default:
			t.Errorf("unexpected Supabase storage request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	baseURL = fake.URL
	return fake
}

func (s *fakeSupabaseStorageServer) SignUploadRequests() int {
	return s.signUploadRequests
}

func (s *fakeSupabaseStorageServer) UploadRequests() int {
	return s.uploadRequests
}

func (s *fakeSupabaseStorageServer) SignDownloadRequests() int {
	return s.signDownloadRequests
}
