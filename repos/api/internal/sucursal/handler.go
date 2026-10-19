package sucursal

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
	sucs, err := h.svc.Listar(r.Context(), claims.TenantID)
	if err != nil {
		slog.ErrorContext(r.Context(), "error listando sucursales", "err", err)
		jsonError(w, "error listando sucursales", http.StatusInternalServerError)
		return
	}
	if claims.Rol != auth.RolDuenio {
		filtradas := sucs[:0]
		for _, suc := range sucs {
			if suc.ID == claims.SucursalID {
				filtradas = append(filtradas, suc)
			}
		}
		sucs = filtradas
	}
	jsonOK(w, sucs)
}

func (h *Handlers) ObtenerPorID(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	id := r.PathValue("id")
	if claims.Rol != auth.RolDuenio && id != claims.SucursalID {
		jsonError(w, "sucursal no encontrada", http.StatusNotFound)
		return
	}
	suc, err := h.svc.ObtenerPorID(r.Context(), id, claims.TenantID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			jsonError(w, "sucursal no encontrada", http.StatusNotFound)
			return
		}
		slog.ErrorContext(r.Context(), "error obteniendo sucursal", "err", err)
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}
	jsonOK(w, suc)
}

func (h *Handlers) Crear(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	var input SucursalInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, "body inválido", http.StatusBadRequest)
		return
	}

	suc, err := h.svc.Crear(r.Context(), claims.TenantID, input)
	if err != nil {
		if errors.Is(err, ErrValidation) {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		slog.ErrorContext(r.Context(), "error creando sucursal", "err", err)
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(suc)
}

func (h *Handlers) Actualizar(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	id := r.PathValue("id")
	if claims.Rol != auth.RolDuenio && id != claims.SucursalID {
		jsonError(w, "sucursal no encontrada", http.StatusNotFound)
		return
	}
	var input SucursalInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, "body inválido", http.StatusBadRequest)
		return
	}

	suc, err := h.svc.Actualizar(r.Context(), id, claims.TenantID, input)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			jsonError(w, "sucursal no encontrada", http.StatusNotFound)
			return
		}
		if errors.Is(err, ErrValidation) {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		slog.ErrorContext(r.Context(), "error actualizando sucursal", "err", err)
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}

	jsonOK(w, suc)
}

func (h *Handlers) Eliminar(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	id := r.PathValue("id")

	err := h.svc.Eliminar(r.Context(), id, claims.TenantID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			jsonError(w, "sucursal no encontrada", http.StatusNotFound)
			return
		}
		slog.ErrorContext(r.Context(), "error eliminando sucursal", "err", err)
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) CrearSector(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	sucursalID := r.PathValue("sucursal_id")
	if claims.Rol != auth.RolDuenio && sucursalID != claims.SucursalID {
		jsonError(w, "sucursal no encontrada", http.StatusNotFound)
		return
	}
	var input SectorInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, "body inválido", http.StatusBadRequest)
		return
	}

	sec, err := h.svc.CrearSector(r.Context(), sucursalID, claims.TenantID, input)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			jsonError(w, "sucursal no encontrada", http.StatusNotFound)
			return
		}
		if errors.Is(err, ErrValidation) {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		slog.ErrorContext(r.Context(), "error creando sector", "err", err)
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(sec)
}

func (h *Handlers) ListarSectores(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	sucursalID := r.PathValue("sucursal_id")
	if claims.Rol != auth.RolDuenio && sucursalID != claims.SucursalID {
		jsonError(w, "sucursal no encontrada", http.StatusNotFound)
		return
	}

	secs, err := h.svc.ListarSectores(r.Context(), sucursalID, claims.TenantID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			jsonError(w, "sucursal no encontrada", http.StatusNotFound)
			return
		}
		slog.ErrorContext(r.Context(), "error listando sectores", "err", err)
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}

	jsonOK(w, secs)
}

func (h *Handlers) EliminarSector(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	id := r.PathValue("id")

	err := h.svc.EliminarSector(r.Context(), id, claims.TenantID, claims.SucursalID)
	if err != nil {
		if errors.Is(err, ErrSectorNotFound) {
			jsonError(w, "sector no encontrado", http.StatusNotFound)
			return
		}
		slog.ErrorContext(r.Context(), "error eliminando sector", "err", err)
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
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
