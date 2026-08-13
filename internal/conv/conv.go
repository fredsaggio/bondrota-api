package conv

import (
	"net/http"
	"strconv"

	"github.com/fredsaggio/bondrota-api/internal/publicid"
	"github.com/go-chi/chi/v5"
)

func ParseInt(r *http.Request, param string) (int64, error) {
	if id, ok := publicid.ResolvedParam(r.Context(), param); ok {
		return id, nil
	}
	return strconv.ParseInt(chi.URLParam(r, param), 10, 64)
}
