package municipios

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"unicode"

	"github.com/fredsaggio/bondrota-api/internal/conv"
	"github.com/fredsaggio/bondrota-api/internal/httputils"
)

type Handler struct {
	store Store
}

type Response struct {
	CodigoIBGE int64  `json:"codigo_ibge"`
	Nome       string `json:"nome"`
	UF         string `json:"uf"`
}

func NewHandler(store Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) ListByUF(w http.ResponseWriter, r *http.Request) {
	uf := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("uf")))
	if !validUF(uf) {
		http.Error(w, "uf must contain exactly two letters", http.StatusBadRequest)
		return
	}

	items, err := h.store.ListByUF(r.Context(), uf)
	if err != nil {
		slog.Error("failed to list municipios", "error", err, "uf", uf)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	response := make([]Response, 0, len(items))
	for _, municipio := range items {
		response = append(response, Response{
			CodigoIBGE: municipio.CodigoIBGE,
			Nome:       municipio.Nome,
			UF:         municipio.UF,
		})
	}
	httputils.Respond(w, http.StatusOK, response)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	codigoIBGE, err := conv.ParseInt(r, "codigoIBGE")
	if err != nil {
		http.Error(w, "invalid codigoIBGE", http.StatusBadRequest)
		return
	}

	municipio, err := h.store.GetByID(r.Context(), codigoIBGE)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "municipio not found", http.StatusNotFound)
			return
		}
		slog.Error("failed to get municipio", "error", err, "codigo_ibge", codigoIBGE)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	httputils.Respond(w, http.StatusOK, Response{
		CodigoIBGE: municipio.CodigoIBGE,
		Nome:       municipio.Nome,
		UF:         municipio.UF,
	})
}

func validUF(uf string) bool {
	if len(uf) != 2 {
		return false
	}
	for _, char := range uf {
		if !unicode.IsLetter(char) || !unicode.IsUpper(char) {
			return false
		}
	}
	return true
}
