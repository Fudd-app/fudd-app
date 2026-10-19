package mesa

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
)

type Handlers struct {
	svc *Service
}

func NuevosHandlers(svc *Service) *Handlers { return &Handlers{svc: svc} }

func (h *Handlers) Listar(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	mesas, err := h.svc.ListarSucursal(r.Context(), claims.TenantID, claims.SucursalID)
	if err != nil {
		slog.ErrorContext(r.Context(), "error listando mesas", "err", err)
		jsonError(w, "error listando mesas", http.StatusInternalServerError)
		return
	}
	jsonOK(w, mesas)
}

func (h *Handlers) Crear(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	var input MesaInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, "body inválido", http.StatusBadRequest)
		return
	}
	sucursalID, err := auth.ResolverSucursal(claims, input.SucursalID)
	if err != nil {
		jsonError(w, err.Error(), http.StatusForbidden)
		return
	}
	input.SucursalID = sucursalID
	m, err := h.svc.Crear(r.Context(), claims.TenantID, input)
	if err != nil {
		if errors.Is(err, ErrValidation) {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, ErrNumeroDuplicado) {
			jsonError(w, err.Error(), http.StatusConflict)
			return
		}
		slog.ErrorContext(r.Context(), "error creando mesa", "err", err)
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(m)
}

func (h *Handlers) Actualizar(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	id := r.PathValue("id")
	if err := h.svc.ValidarAlcance(r.Context(), id, claims.TenantID, claims.SucursalID); err != nil {
		jsonError(w, "mesa no encontrada", http.StatusNotFound)
		return
	}
	var u MesaUpdate
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		jsonError(w, "body inválido", http.StatusBadRequest)
		return
	}
	m, err := h.svc.Actualizar(r.Context(), id, claims.TenantID, u)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			jsonError(w, "mesa no encontrada", http.StatusNotFound)
			return
		}
		if errors.Is(err, ErrValidation) {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, ErrNumeroDuplicado) {
			jsonError(w, err.Error(), http.StatusConflict)
			return
		}
		slog.ErrorContext(r.Context(), "error actualizando mesa", "err", err)
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}
	jsonOK(w, m)
}

func (h *Handlers) CerrarCuenta(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	id := r.PathValue("id")
	if err := h.svc.ValidarAlcance(r.Context(), id, claims.TenantID, claims.SucursalID); err != nil {
		jsonError(w, "mesa no encontrada", http.StatusNotFound)
		return
	}
	m, err := h.svc.CerrarCuenta(r.Context(), id, claims.TenantID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			jsonError(w, "mesa no encontrada", http.StatusNotFound)
			return
		}
		slog.ErrorContext(r.Context(), "error cerrando cuenta", "err", err)
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}
	jsonOK(w, m)
}

func (h *Handlers) HabilitarPago(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	id := r.PathValue("id")
	if err := h.svc.ValidarAlcance(r.Context(), id, claims.TenantID, claims.SucursalID); err != nil {
		jsonError(w, "mesa no encontrada", http.StatusNotFound)
		return
	}
	if id == "" {
		jsonError(w, "id de mesa requerido", http.StatusBadRequest)
		return
	}

	habilitado := true
	if r.Body != nil {
		var input struct {
			Habilitado *bool `json:"habilitado"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err == nil && input.Habilitado != nil {
			habilitado = *input.Habilitado
		}
	}

	m, err := h.svc.HabilitarPago(r.Context(), id, claims.TenantID, habilitado)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			jsonError(w, "mesa no encontrada", http.StatusNotFound)
			return
		}
		slog.ErrorContext(r.Context(), "error habilitando pago en mesa", "err", err)
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}
	jsonOK(w, m)
}

func (h *Handlers) SolicitarCuenta(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("qr_token")
	if token == "" {
		jsonError(w, "qr_token requerido", http.StatusBadRequest)
		return
	}

	mp, err := h.svc.SolicitarCuentaPorQRToken(r.Context(), token)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			jsonError(w, "mesa no encontrada", http.StatusNotFound)
			return
		}
		if errors.Is(err, ErrMesaInactiva) {
			jsonError(w, "mesa inactiva", http.StatusConflict)
			return
		}
		slog.ErrorContext(r.Context(), "error solicitando cuenta", "err", err)
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}
	jsonOK(w, mp)
}

func (h *Handlers) Eliminar(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	id := r.PathValue("id")
	if err := h.svc.ValidarAlcance(r.Context(), id, claims.TenantID, claims.SucursalID); err != nil {
		jsonError(w, "mesa no encontrada", http.StatusNotFound)
		return
	}
	if err := h.svc.Eliminar(r.Context(), id, claims.TenantID); err != nil {
		if errors.Is(err, ErrNotFound) {
			jsonError(w, "mesa no encontrada", http.StatusNotFound)
			return
		}
		slog.ErrorContext(r.Context(), "error eliminando mesa", "err", err)
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) MesaPorQR(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("qr_token")
	if token == "" {
		jsonError(w, "qr_token requerido", http.StatusBadRequest)
		return
	}

	mp, err := h.svc.ObtenerPorQRToken(r.Context(), token)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			jsonError(w, "mesa no encontrada", http.StatusNotFound)
			return
		}
		slog.ErrorContext(r.Context(), "error obteniendo mesa por QR", "err", err)
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}
	jsonOK(w, mp)
}

func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
