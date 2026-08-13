package clientes

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/fredsaggio/bondrota-api/internal/auth"
	"github.com/fredsaggio/bondrota-api/internal/conv"
	"github.com/fredsaggio/bondrota-api/internal/db"
	"github.com/fredsaggio/bondrota-api/internal/httputils"
	"github.com/fredsaggio/bondrota-api/internal/validation"
)

// ArquivoMovedor move um objeto ja enviado ao Storage do caminho de espera
// (onde um documento entra antes do registro ter ID) para o caminho
// definitivo, depois que o registro e criado. Definida aqui, e nao importada
// de internal/storage, para este pacote nao depender de detalhes do provedor
// de armazenamento — so precisa saber mover um arquivo de um lugar a outro.
type ArquivoMovedor interface {
	MoveObject(ctx context.Context, bucket, from, to string) error
}

type ClienteHandler struct {
	clienteSvc ClienteService
	arquivos   ArquivoMovedor
}

// NewClienteHandler aceita o movedor de arquivos como variadico de proposito:
// a maioria dos testes deste pacote nem chega a exercitar documentos, e
// forcar todos eles a passar um mock so pra satisfazer a assinatura seria
// ruido. Sem o argumento, o documento simplesmente fica no caminho que veio na
// requisicao — igual ao comportamento de antes desta funcionalidade existir.
func NewClienteHandler(clienteSvc ClienteService, arquivos ...ArquivoMovedor) *ClienteHandler {
	h := &ClienteHandler{clienteSvc: clienteSvc}
	if len(arquivos) > 0 {
		h.arquivos = arquivos[0]
	}
	return h
}

type CreateClienteRequest struct {
	Nome                   string `json:"nome"`
	CPF                    string `json:"cpf"`
	Senha                  string `json:"senha"`
	Telefone               string `json:"telefone"`
	DataNasc               string `json:"data_nasc"`
	DocumentoIdentificacao string `json:"documento_identificacao"`
	ComprovanteResidencia  string `json:"comprovante_residencia"`
}

type UpdateClienteRequest struct {
	Nome string `json:"nome"`
	// Telefone pode ser limpo com string vazia. Documentos podem ser substituidos,
	// mas nao apagados: uma conta administrativa cria clientes ja conferidos.
	Telefone               *string `json:"telefone"`
	DataNasc               string  `json:"data_nasc"`
	DocumentoIdentificacao *string `json:"documento_identificacao"`
	ComprovanteResidencia  *string `json:"comprovante_residencia"`
}

type ClienteResponse struct {
	ID                     int64  `json:"id"`
	Nome                   string `json:"nome"`
	CPF                    string `json:"cpf"`
	Telefone               string `json:"telefone"`
	DataNasc               string `json:"data_nasc"`
	DocumentoIdentificacao string `json:"documento_identificacao"`
	ComprovanteResidencia  string `json:"comprovante_residencia"`
}

type ClienteListResponse struct {
	Items      []ClienteResponse `json:"items"`
	NextCursor string            `json:"next_cursor,omitempty"`
	HasMore    bool              `json:"has_more"`
}

type ClienteResumoResponse struct {
	Total int64 `json:"total"`
}

type ClienteComVinculosResponse struct {
	ID                     int64             `json:"id"`
	Nome                   string            `json:"nome"`
	CPF                    string            `json:"cpf"`
	Telefone               string            `json:"telefone"`
	DataNasc               string            `json:"data_nasc"`
	DocumentoIdentificacao string            `json:"documento_identificacao"`
	ComprovanteResidencia  string            `json:"comprovante_residencia"`
	Vinculos               []VinculoResponse `json:"vinculos"`
}

type LoginRequest struct {
	CPF   string `json:"cpf"`
	Senha string `json:"senha"`
}

func (h *ClienteHandler) Login(w http.ResponseWriter, r *http.Request) {
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

	token, err := h.clienteSvc.Login(ctx, strings.TrimSpace(req.CPF), req.Senha)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) || errors.Is(err, ErrNotFound) {
			http.Error(w, "Credenciais inválidas.", http.StatusUnauthorized)
			return
		}
		slog.Error("failed to login cliente", "error", err)
		http.Error(w, "Erro inesperado no servidor. Tente novamente em instantes.", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusOK, map[string]string{"token": token})
}

func (h *ClienteHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateClienteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Não foi possível processar os dados enviados.", http.StatusBadRequest)
		return
	}

	input, err := toClienteInput(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cliente, err := h.clienteSvc.Create(ctx, input)
	if err != nil {
		if db.IsUniqueViolation(err, "clientes_cpf_key") {
			http.Error(w, "Já existe um cadastro com este CPF.", http.StatusConflict)
			return
		}
		slog.Error("failed to create cliente", "error", err)
		http.Error(w, "Erro inesperado no servidor. Tente novamente em instantes.", http.StatusInternalServerError)
		return
	}

	cliente = h.organizarDocumentos(ctx, cliente, input)

	httputils.Respond(w, http.StatusCreated, toClienteResponse(cliente))
}

// organizarDocumentos leva os dois arquivos da pasta de espera para os paths
// definitivos do cliente. A operacao e best-effort pelo mesmo motivo dos demais
// uploads: o cadastro ja existe quando o Storage e chamado, e responder erro
// incentivaria um retry que esbarraria no CPF duplicado.
func (h *ClienteHandler) organizarDocumentos(ctx context.Context, cliente *Cliente, input ClienteInput) *Cliente {
	if h.arquivos == nil {
		return cliente
	}
	prefixo := "clientes/" + strconv.FormatInt(cliente.ID, 10) + "/"
	documentoIdentificacao := input.DocumentoIdentificacao
	comprovanteResidencia := input.ComprovanteResidencia
	moveu := false

	destinoIdentificacao := prefixo + "documento-identificacao" + path.Ext(input.DocumentoIdentificacao)
	if err := h.arquivos.MoveObject(ctx, "documentos", input.DocumentoIdentificacao, destinoIdentificacao); err != nil {
		slog.Error("failed to organize cliente identification document", "error", err, "clienteID", cliente.ID)
	} else {
		documentoIdentificacao = destinoIdentificacao
		moveu = true
	}

	destinoResidencia := prefixo + "comprovante-residencia" + path.Ext(input.ComprovanteResidencia)
	if err := h.arquivos.MoveObject(ctx, "documentos", input.ComprovanteResidencia, destinoResidencia); err != nil {
		slog.Error("failed to organize cliente residence document", "error", err, "clienteID", cliente.ID)
	} else {
		comprovanteResidencia = destinoResidencia
		moveu = true
	}
	if !moveu {
		return cliente
	}
	atualizado, err := h.clienteSvc.Update(ctx, cliente.ID, func(c *Cliente) (bool, error) {
		changed := c.DocumentoIdentificacao != documentoIdentificacao || c.ComprovanteResidencia != comprovanteResidencia
		c.DocumentoIdentificacao = documentoIdentificacao
		c.ComprovanteResidencia = comprovanteResidencia
		return changed, nil
	})
	if err != nil {
		slog.Error("failed to persist organized cliente documents", "error", err, "clienteID", cliente.ID)
		return cliente
	}
	return atualizado
}

func (h *ClienteHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	clienteID, err := conv.ParseInt(r, "clienteID")
	if err != nil {
		http.Error(w, "Registro não encontrado.", http.StatusBadRequest)
		return
	}

	cliente, err := h.clienteSvc.GetByID(ctx, clienteID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "Cliente não encontrado.", http.StatusNotFound)
			return
		}
		slog.Error("failed to get cliente", "error", err, "clienteID", clienteID)
		http.Error(w, "Erro inesperado no servidor. Tente novamente em instantes.", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusOK, toClienteComVinculosResponse(cliente))
}

func (h *ClienteHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	params, err := parseClienteListParams(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := h.clienteSvc.List(ctx, params)
	if err != nil {
		slog.Error("failed to list clientes", "error", err)
		http.Error(w, "Erro inesperado no servidor. Tente novamente em instantes.", http.StatusInternalServerError)
		return
	}

	items := make([]ClienteResponse, 0, len(result.Items))
	for _, c := range result.Items {
		items = append(items, toClienteResponse(&c))
	}

	resp := ClienteListResponse{Items: items, HasMore: result.HasMore}
	if result.NextCursorID > 0 {
		resp.NextCursor = encodeClienteCursor(result.NextCursorID)
	}

	httputils.Respond(w, http.StatusOK, resp)
}

func (h *ClienteHandler) Resumo(w http.ResponseWriter, r *http.Request) {
	resumo, err := h.clienteSvc.Resumo(r.Context())
	if err != nil {
		slog.Error("failed to summarize clientes", "error", err)
		http.Error(w, "Erro inesperado no servidor. Tente novamente em instantes.", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusOK, ClienteResumoResponse{Total: resumo.Total})
}

func parseClienteListParams(r *http.Request) (ClienteListParams, error) {
	query := r.URL.Query()
	params := ClienteListParams{Busca: query.Get("q")}

	if raw := query.Get("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit <= 0 {
			return ClienteListParams{}, errors.New("Parâmetro de listagem inválido.")
		}
		params.Limit = limit
	}

	if raw := query.Get("cursor"); raw != "" {
		cursorID, err := decodeClienteCursor(raw)
		if err != nil {
			return ClienteListParams{}, errors.New("Parâmetro de listagem inválido.")
		}
		params.CursorID = cursorID
	}

	return params, nil
}

// O cursor e opaco para o consumidor, mesmo carregando so o id: o formato pode
// mudar (para incluir outra chave de ordenacao) sem quebrar contrato.
func encodeClienteCursor(id int64) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(id, 10)))
}

func decodeClienteCursor(value string) (int64, error) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return 0, fmt.Errorf("decode cursor: %w", err)
	}
	id, err := strconv.ParseInt(string(raw), 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("Parâmetro de listagem inválido.")
	}
	return id, nil
}

func (h *ClienteHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	clienteID, err := conv.ParseInt(r, "clienteID")
	if err != nil {
		http.Error(w, "Registro não encontrado.", http.StatusBadRequest)
		return
	}

	var req UpdateClienteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Não foi possível processar os dados enviados.", http.StatusBadRequest)
		return
	}

	cliente, err := h.clienteSvc.Update(ctx, clienteID, func(c *Cliente) (bool, error) {
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
			if nome != c.Nome {
				c.Nome = nome
				updated = true
			}
		}
		if req.Telefone != nil {
			telefone, err := validation.Telefone(*req.Telefone)
			if err != nil {
				return false, err
			}
			if telefone != c.Telefone {
				c.Telefone = telefone
				updated = true
			}
		}
		if req.DataNasc != "" {
			dataNasc, err := parseDate(req.DataNasc)
			if err != nil {
				return false, ErrDataInvalida
			}
			if !dataNasc.Equal(c.DataNasc) {
				c.DataNasc = dataNasc
				updated = true
			}
		}
		if req.DocumentoIdentificacao != nil {
			if strings.TrimSpace(*req.DocumentoIdentificacao) == "" {
				return false, ErrDocumentoIdentificacaoObrigatorio
			}
			documento, err := validation.CaminhoDocumento(*req.DocumentoIdentificacao)
			if err != nil {
				return false, err
			}
			if documento != c.DocumentoIdentificacao {
				c.DocumentoIdentificacao = documento
				updated = true
			}
		}
		if req.ComprovanteResidencia != nil {
			if strings.TrimSpace(*req.ComprovanteResidencia) == "" {
				return false, ErrComprovanteResidenciaObrigatorio
			}
			comprovante, err := validation.CaminhoDocumento(*req.ComprovanteResidencia)
			if err != nil {
				return false, err
			}
			if comprovante != c.ComprovanteResidencia {
				c.ComprovanteResidencia = comprovante
				updated = true
			}
		}
		return updated, nil
	})
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "Cliente não encontrado.", http.StatusNotFound)
			return
		}
		if errors.Is(err, ErrNomeObrigatorio) ||
			errors.Is(err, validation.ErrNomeInvalido) ||
			errors.Is(err, validation.ErrTelefoneInvalido) ||
			errors.Is(err, validation.ErrCaminhoDocumentoInvalido) ||
			errors.Is(err, ErrDocumentoIdentificacaoObrigatorio) ||
			errors.Is(err, ErrComprovanteResidenciaObrigatorio) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, ErrDataInvalida) {
			http.Error(w, "data_nasc must be in format YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		slog.Error("failed to update cliente", "error", err, "clienteID", clienteID)
		http.Error(w, "Erro inesperado no servidor. Tente novamente em instantes.", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusOK, toClienteResponse(cliente))
}

func (h *ClienteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	clienteID, err := conv.ParseInt(r, "clienteID")
	if err != nil {
		http.Error(w, "Registro não encontrado.", http.StatusBadRequest)
		return
	}

	if err := h.clienteSvc.Delete(ctx, clienteID); err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "Cliente não encontrado.", http.StatusNotFound)
			return
		}
		// Vinculos e reservas do cliente somem em cascata, mas uma reserva ja alocada
		// a uma viagem e protegida por RESTRICT em viagem_reservas.
		if db.IsAnyForeignKeyViolation(err) {
			http.Error(w, "Este cliente tem reservas em viagens e não pode ser removido.", http.StatusConflict)
			return
		}
		slog.Error("failed to delete cliente", "error", err, "clienteID", clienteID)
		http.Error(w, "Erro inesperado no servidor. Tente novamente em instantes.", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func toClienteInput(req CreateClienteRequest) (ClienteInput, error) {
	nomeBruto := strings.TrimSpace(req.Nome)
	if nomeBruto == "" {
		return ClienteInput{}, errors.New("Informe o nome.")
	}
	nome, err := validation.Nome(nomeBruto)
	if err != nil {
		return ClienteInput{}, err
	}
	if strings.TrimSpace(req.CPF) == "" {
		return ClienteInput{}, errors.New("Informe o CPF.")
	}
	cpf, err := validation.CPF(req.CPF)
	if err != nil {
		return ClienteInput{}, err
	}
	if strings.TrimSpace(req.Senha) == "" {
		return ClienteInput{}, errors.New("Informe a senha.")
	}
	if req.DataNasc == "" {
		return ClienteInput{}, errors.New("Informe a data de nascimento.")
	}

	dataNasc, err := parseDate(req.DataNasc)
	if err != nil {
		return ClienteInput{}, errors.New("data_nasc must be in format YYYY-MM-DD")
	}

	telefone, err := validation.Telefone(req.Telefone)
	if err != nil {
		return ClienteInput{}, err
	}
	if strings.TrimSpace(req.DocumentoIdentificacao) == "" {
		return ClienteInput{}, ErrDocumentoIdentificacaoObrigatorio
	}
	documentoIdentificacao, err := validation.CaminhoDocumento(req.DocumentoIdentificacao)
	if err != nil {
		return ClienteInput{}, err
	}
	if strings.TrimSpace(req.ComprovanteResidencia) == "" {
		return ClienteInput{}, ErrComprovanteResidenciaObrigatorio
	}
	comprovanteResidencia, err := validation.CaminhoDocumento(req.ComprovanteResidencia)
	if err != nil {
		return ClienteInput{}, err
	}

	return ClienteInput{
		Nome:                   nome,
		CPF:                    cpf,
		Senha:                  req.Senha,
		Telefone:               telefone,
		DataNasc:               dataNasc,
		DocumentoIdentificacao: documentoIdentificacao,
		ComprovanteResidencia:  comprovanteResidencia,
	}, nil
}

func parseDate(value string) (time.Time, error) {
	return time.Parse("2006-01-02", value)
}

func toClienteResponse(c *Cliente) ClienteResponse {
	return ClienteResponse{
		ID:                     c.ID,
		Nome:                   c.Nome,
		CPF:                    c.CPF,
		Telefone:               c.Telefone,
		DataNasc:               c.DataNasc.Format("2006-01-02"),
		DocumentoIdentificacao: c.DocumentoIdentificacao,
		ComprovanteResidencia:  c.ComprovanteResidencia,
	}
}

func toClienteComVinculosResponse(c *ClienteComVinculos) ClienteComVinculosResponse {
	vinculos := make([]VinculoResponse, 0, len(c.Vinculos))
	for _, v := range c.Vinculos {
		vinculos = append(vinculos, toVinculoResponse(&v))
	}

	return ClienteComVinculosResponse{
		ID:                     c.ID,
		Nome:                   c.Nome,
		CPF:                    c.CPF,
		Telefone:               c.Telefone,
		DataNasc:               c.DataNasc.Format("2006-01-02"),
		DocumentoIdentificacao: c.DocumentoIdentificacao,
		ComprovanteResidencia:  c.ComprovanteResidencia,
		Vinculos:               vinculos,
	}
}
