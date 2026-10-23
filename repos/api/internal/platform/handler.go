package platform

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
)

type Handlers struct {
	svc *Service
}

func NuevosHandlers(svc *Service) *Handlers {
	return &Handlers{svc: svc}
}

func (h *Handlers) Status(w http.ResponseWriter, r *http.Request) {
	st, err := h.svc.ObtenerStatus(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "error obteniendo status de plataforma", "err", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "error obteniendo status de plataforma"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(st)
}

func (h *Handlers) ListarTenants(w http.ResponseWriter, r *http.Request) {
	tenants, err := h.svc.ListarTenants(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "error listando tenants de plataforma", "err", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "error listando tenants"})
		return
	}

	if tenants == nil {
		tenants = []TenantResumen{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(tenants)
}

func (h *Handlers) DetalleActividad(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "id de tenant requerido"})
		return
	}

	detalle, err := h.svc.ObtenerActividadTenant(r.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "no encontrado") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "tenant no encontrado"})
			return
		}
		slog.ErrorContext(r.Context(), "error obteniendo detalle de tenant", "tenant_id", id, "err", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "error obteniendo detalle de actividad"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(detalle)
}
