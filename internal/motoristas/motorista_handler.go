package motoristas

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/auth"
	"github.com/fredsaggio/bondrota-api/internal/conv"
	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/fredsaggio/bondrota-api/internal/httputils"
	"github.com/fredsaggio/bondrota-api/internal/validation"
)

type CreateMotoristaRequest struct {
	Nome                string `json:"nome"`
	CPF                 string `json:"cpf"`
	Senha               string `json:"senha"`
	Telefone            string `json:"telefone"`
	DataNasc            string `json:"data_nasc"`
	Turno               Turno  `json:"turno"`
	MunicipioTrabalhoID int64  `json:"municipio_trabalho_id"`
	Foto                string `json:"foto"`
}

type UpdateMotoristaRequest struct {
	Nome string `json:"nome"`
	// Telefone e Foto sao ponteiros porque sao opcionais e podem ser
	// legitimamente limpos: chave ausente/null preserva o valor atual, string vazia
	// explicita apaga o campo. Os demais campos sao obrigatorios e nunca fazem
	// sentido em branco, entao continuam string simples.
	Telefone            *string `json:"telefone"`
	DataNasc            string  `json:"data_nasc"`
	Turno               Turno   `json:"turno"`
	MunicipioTrabalhoID int64   `json:"municipio_trabalho_id"`
	Foto                *string `json:"foto"`
}

type LoginRequest struct {
	CPF   string `json:"cpf"`
	Senha string `json:"senha"`
}

type MotoristaResponse struct {
	ID                  string `json:"id"`
	Nome                string `json:"nome"`
	CPF                 string `json:"cpf"`
	Telefone            string `json:"telefone"`
	DataNasc            string `json:"data_nasc"`
	Turno               Turno  `json:"turno"`
	MunicipioTrabalhoID int64  `json:"municipio_trabalho_id"`
	Foto                string `json:"foto"`
}

// ArquivoMovedor move um objeto ja enviado ao Storage do caminho de espera
// (onde a foto entra antes do motorista ter ID) para o caminho definitivo,
// depois que o registro e criado. Definida aqui, e nao importada de
// internal/storage, para este pacote nao depender de detalhes do provedor de
// armazenamento — so precisa saber mover um arquivo de um lugar a outro.
type ArquivoMovedor interface {
	MoveObject(ctx context.Context, bucket, from, to string) error
}

type MotoristaHandler struct {
	svc      MotoristaService
	arquivos ArquivoMovedor
}

// NewMotoristaHandler aceita o movedor de arquivos como variadico de proposito:
// a maioria dos testes deste pacote nem chega a exercitar o campo foto, e
// forcar todos eles a passar um mock so pra satisfazer a assinatura seria
// ruido. Sem o argumento, a foto simplesmente fica no caminho que veio na
// requisicao — igual ao comportamento de antes desta funcionalidade existir.
func NewMotoristaHandler(svc MotoristaService, arquivos ...ArquivoMovedor) *MotoristaHandler {
	h := &MotoristaHandler{svc: svc}
	if len(arquivos) > 0 {
		h.arquivos = arquivos[0]
	}
	return h
}

func (h *MotoristaHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Não foi possível processar os dados enviados.", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.CPF) == "" {
		http.Error(w, "Informe o CPF.", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Senha) == "" {
		http.Error(w, "Informe a senha.", http.StatusBadRequest)
		return
	}

	token, err := h.svc.Login(ctx, strings.TrimSpace(req.CPF), req.Senha)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			http.Error(w, "Credenciais inválidas.", http.StatusUnauthorized)
			return
		}
		slog.Error("failed to login motorista", "error", err)
		http.Error(w, "Erro inesperado no servidor. Tente novamente em instantes.", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusOK, map[string]string{"token": token})
}

func (h *MotoristaHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateMotoristaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Não foi possível processar os dados enviados.", http.StatusBadRequest)
		return
	}

	nomeBruto := strings.TrimSpace(req.Nome)
	if nomeBruto == "" {
		http.Error(w, "Informe o nome.", http.StatusBadRequest)
		return
	}
	nome, err := validation.Nome(nomeBruto)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.CPF) == "" {
		http.Error(w, "Informe o CPF.", http.StatusBadRequest)
		return
	}
	cpf, err := validation.CPF(req.CPF)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Senha) == "" {
		http.Error(w, "Informe a senha.", http.StatusBadRequest)
		return
	}
	if req.Turno == "" {
		http.Error(w, "Selecione o turno.", http.StatusBadRequest)
		return
	}
	if req.DataNasc == "" {
		http.Error(w, "Informe a data de nascimento.", http.StatusBadRequest)
		return
	}
	if req.MunicipioTrabalhoID <= 0 {
		http.Error(w, "Selecione a cidade de trabalho.", http.StatusBadRequest)
		return
	}

	switch req.Turno {
	case TurnoMatutino, TurnoVespertino, TurnoNoturno, TurnoIntegral:

	default:
		http.Error(w, "turno must be MT, VT, NT or IN", http.StatusBadRequest)
		return
	}

	dataNasc, err := time.Parse("2006-01-02", req.DataNasc)
	if err != nil {
		http.Error(w, "data_nasc must be in format YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	telefone, err := validation.Telefone(req.Telefone)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	input := MotoristaInput{
		Nome:                nome,
		CPF:                 cpf,
		Senha:               req.Senha,
		Telefone:            telefone,
		DataNasc:            dataNasc,
		Turno:               req.Turno,
		MunicipioTrabalhoID: req.MunicipioTrabalhoID,
		Foto:                strings.TrimSpace(req.Foto),
	}

	motorista, err := h.svc.Create(ctx, input)
	if err != nil {
		if db.IsUniqueViolation(err, "telefones_cadastrados_pkey") {
			http.Error(w, "Já existe outro cadastro com este telefone.", http.StatusConflict)
			return
		}
		if db.IsUniqueViolation(err, "motoristas_cpf_key") {
			http.Error(w, "Já existe um cadastro com este CPF.", http.StatusConflict)
			return
		}
		slog.Error("failed to create motorista", "error", err)
		http.Error(w, "Erro inesperado no servidor. Tente novamente em instantes.", http.StatusInternalServerError)
		return
	}

	if input.Foto != "" {
		motorista = h.organizarFoto(ctx, motorista, input.Foto)
	}

	httputils.Respond(w, http.StatusCreated, toMotoristaResponse(motorista))
}

// organizarFoto leva a foto da pasta de espera (onde o admin envia antes do
// motorista ter ID) para o caminho definitivo "motoristas/{id}/foto{ext}".
// Best-effort: se o Storage falhar aqui, o motorista ja foi criado com
// sucesso — falhar a resposta agora deixaria o admin sem saber se o cadastro
// existe, e um retry esbarraria no CPF duplicado. A foto so fica no caminho
// de espera nesse caso raro, em vez de ficar organizada.
func (h *MotoristaHandler) organizarFoto(ctx context.Context, motorista *Motorista, caminhoEnviado string) *Motorista {
	if h.arquivos == nil {
		return motorista
	}
	destino := fmt.Sprintf("motoristas/%s/foto%s", motorista.PublicID, path.Ext(caminhoEnviado))
	if err := h.arquivos.MoveObject(ctx, "fotos", caminhoEnviado, destino); err != nil {
		slog.Error("failed to organize motorista foto", "error", err, "motoristaID", motorista.ID)
		return motorista
	}
	atualizado, err := h.svc.Update(ctx, motorista.ID, func(m *Motorista) (bool, error) {
		m.Foto = destino
		return true, nil
	})
	if err != nil {
		slog.Error("failed to persist organized motorista foto", "error", err, "motoristaID", motorista.ID)
		return motorista
	}
	return atualizado
}

func (h *MotoristaHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	motoristaID, err := conv.ParseInt(r, "id")
	if err != nil {
		http.Error(w, "Registro não encontrado.", http.StatusBadRequest)
		return
	}

	motorista, err := h.svc.GetByID(ctx, motoristaID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "Motorista não encontrado.", http.StatusNotFound)
			return
		}
		slog.Error("failed to get motorista", "error", err, "motoristaID", motoristaID)
		http.Error(w, "Erro inesperado no servidor. Tente novamente em instantes.", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusOK, toMotoristaResponse(motorista))
}

func (h *MotoristaHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	motoristas, err := h.svc.List(ctx)
	if err != nil {
		slog.Error("failed to list motoristas", "error", err)
		http.Error(w, "Erro inesperado no servidor. Tente novamente em instantes.", http.StatusInternalServerError)
		return
	}

	resp := make([]MotoristaResponse, 0, len(motoristas))
	for _, m := range motoristas {
		resp = append(resp, toMotoristaResponse(&m))
	}

	httputils.Respond(w, http.StatusOK, resp)
}

func (h *MotoristaHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	motoristaID, err := conv.ParseInt(r, "id")
	if err != nil {
		http.Error(w, "Registro não encontrado.", http.StatusBadRequest)
		return
	}

	var req UpdateMotoristaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Não foi possível processar os dados enviados.", http.StatusBadRequest)
		return
	}

	motorista, err := h.svc.Update(ctx, motoristaID, func(m *Motorista) (bool, error) {
		updated := false
		if req.Nome != "" {
			nomeBruto := strings.TrimSpace(req.Nome)
			if nomeBruto == "" {
				return false, ErrNomeObrigatorio
			}
			nome, err := validation.Nome(nomeBruto)
			if err != nil {
				return false, err
			}
			if nome != m.Nome {
				m.Nome = nome
				updated = true
			}
		}
		if req.Telefone != nil {
			telefone, err := validation.Telefone(*req.Telefone)
			if err != nil {
				return false, err
			}
			if telefone != m.Telefone {
				m.Telefone = telefone
				updated = true
			}
		}
		if req.DataNasc != "" {
			dataNasc, err := time.Parse("2006-01-02", req.DataNasc)
			if err != nil {
				return false, ErrDataNascInvalida
			}
			if !dataNasc.Equal(m.DataNasc) {
				m.DataNasc = dataNasc
				updated = true
			}
		}
		if req.Turno != "" {
			switch req.Turno {
			case TurnoMatutino, TurnoVespertino, TurnoNoturno, TurnoIntegral:
			default:
				return false, ErrTurnoInvalido
			}
			if req.Turno != m.Turno {
				m.Turno = req.Turno
				updated = true
			}
		}
		if req.MunicipioTrabalhoID > 0 {
			if req.MunicipioTrabalhoID != m.MunicipioTrabalhoID {
				m.MunicipioTrabalhoID = req.MunicipioTrabalhoID
				updated = true
			}
		}
		if req.Foto != nil {
			foto := strings.TrimSpace(*req.Foto)
			if foto != m.Foto {
				m.Foto = foto
				updated = true
			}
		}
		return updated, nil
	})
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "Motorista não encontrado.", http.StatusNotFound)
			return
		}
		if db.IsUniqueViolation(err, "telefones_cadastrados_pkey") {
			http.Error(w, "Já existe outro cadastro com este telefone.", http.StatusConflict)
			return
		}
		if errors.Is(err, ErrNomeObrigatorio) ||
			errors.Is(err, ErrTurnoInvalido) ||
			errors.Is(err, ErrDataNascInvalida) ||
			errors.Is(err, validation.ErrNomeInvalido) ||
			errors.Is(err, validation.ErrTelefoneInvalido) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		slog.Error("failed to update motorista", "error", err, "motoristaID", motoristaID)
		http.Error(w, "Erro inesperado no servidor. Tente novamente em instantes.", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusOK, toMotoristaResponse(motorista))
}

func (h *MotoristaHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	motoristaID, err := conv.ParseInt(r, "id")
	if err != nil {
		http.Error(w, "Registro não encontrado.", http.StatusBadRequest)
		return
	}

	if err := h.svc.Delete(ctx, motoristaID); err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "Motorista não encontrado.", http.StatusNotFound)
			return
		}
		if db.IsAnyForeignKeyViolation(err) {
			http.Error(w, "Este motorista está alocado em viagens e não pode ser removido.", http.StatusConflict)
			return
		}
		slog.Error("failed to delete motorista", "error", err, "motoristaID", motoristaID)
		http.Error(w, "Erro inesperado no servidor. Tente novamente em instantes.", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func toMotoristaResponse(m *Motorista) MotoristaResponse {
	return MotoristaResponse{
		ID:                  m.PublicID,
		Nome:                m.Nome,
		CPF:                 m.CPF,
		Telefone:            m.Telefone,
		DataNasc:            m.DataNasc.Format("2006-01-02"),
		Turno:               m.Turno,
		MunicipioTrabalhoID: m.MunicipioTrabalhoID,
		Foto:                m.Foto,
	}
}
