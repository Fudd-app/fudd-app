package resenia

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

func NuevosHandlers(svc *Service) *Handlers {
	return &Handlers{svc: svc}
}

func (h *Handlers) Crear(w http.ResponseWriter, r *http.Request) {
	qrToken := r.PathValue("qr_token")
	var input CrearResenaInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, "body inválido", http.StatusBadRequest)
		return
	}

	resena, err := h.svc.CrearResena(r.Context(), qrToken, input)
	if err != nil {
		switch {
		case errors.Is(err, ErrValidation):
			jsonError(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrMesaInactiva):
			jsonError(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrSinPedidos):
			jsonError(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrResenaDuplicada):
			jsonError(w, err.Error(), http.StatusConflict)
		case errors.Is(err, ErrNotFound):
			jsonError(w, "mesa no encontrada", http.StatusNotFound)
		default:
			slog.ErrorContext(r.Context(), "error creando reseña", "error", err, "qr_token", qrToken)
			jsonError(w, "error interno", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resena)
}

func (h *Handlers) RegistrarClickGoogle(w http.ResponseWriter, r *http.Request) {
	qrToken := r.PathValue("qr_token")
	err := h.svc.RegistrarClickGoogle(r.Context(), qrToken)
	if err != nil {
		switch {
		case errors.Is(err, ErrValidation):
			jsonError(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrNotFound):
			jsonError(w, "reseña no encontrada", http.StatusNotFound)
		default:
			slog.ErrorContext(r.Context(), "error registrando clic de google", "error", err, "qr_token", qrToken)
			jsonError(w, "error interno", http.StatusInternalServerError)
		}
		return
	}

	jsonOK(w, map[string]any{"ok": true})
}

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
		case errors.Is(err, ErrValidation):
			jsonError(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrSucursalInvalida):
			jsonError(w, "sucursal no encontrada", http.StatusNotFound)
		default:
			slog.ErrorContext(r.Context(), "error obteniendo resumen de reseñas", "error", err, "tenant_id", claims.TenantID)
			jsonError(w, "error interno", http.StatusInternalServerError)
		}
		return
	}

	jsonOK(w, resumen)
}

func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
