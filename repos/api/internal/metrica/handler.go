package metrica

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
)

type Handlers struct {
	svc *Service
}

func NuevosHandlers(svc *Service) *Handlers { return &Handlers{svc: svc} }

func (h *Handlers) Resumen(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		jsonError(w, "no autorizado", http.StatusUnauthorized)
		return
	}

	query := r.URL.Query()
	sucursalID := strings.TrimSpace(query.Get("sucursal_id"))
	if claims.Rol != auth.RolDuenio {
		if sucursalID != "" && sucursalID != claims.SucursalID {
			jsonError(w, "sucursal fuera de alcance", http.StatusForbidden)
			return
		}
		sucursalID = claims.SucursalID
	}
	resumen, err := h.svc.ObtenerResumen(
		r.Context(),
		claims.TenantID,
		sucursalID,
		strings.TrimSpace(query.Get("desde")),
		strings.TrimSpace(query.Get("hasta")),
	)
	if err != nil {
		switch {
		case errors.Is(err, ErrValidacion):
			jsonError(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrSucursalNoEncontrada):
			jsonError(w, "sucursal no encontrada", http.StatusNotFound)
		default:
			slog.ErrorContext(r.Context(), "error obteniendo métricas", "error", err, "tenant_id", claims.TenantID)
			jsonError(w, "error interno", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "private, max-age=30")
	_ = json.NewEncoder(w).Encode(resumen)
}

func jsonError(w http.ResponseWriter, mensaje string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": mensaje})
}
