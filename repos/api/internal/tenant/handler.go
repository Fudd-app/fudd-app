package tenant

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

func (h *Handlers) Crear(w http.ResponseWriter, r *http.Request) {
	var input OnboardingInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, "body inválido", http.StatusBadRequest)
		return
	}

	t, err := h.svc.Crear(r.Context(), input)
	if err != nil {
		// Validation errors from service layer are safe to expose
		if errors.Is(err, ErrValidation) {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, ErrSlugConflict) {
			jsonError(w, "Ese nombre en URL ya está en uso. Probá con otro.", http.StatusConflict)
			return
		}
		if errors.Is(err, ErrEmailAdminConflict) {
			jsonError(w, "Ese correo de acceso ya está registrado. Iniciá sesión o usá otro correo.", http.StatusConflict)
			return
		}

		slog.ErrorContext(r.Context(), "error creando tenant", "err", err)
		jsonError(w, "error creando negocio", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(t)
}

func (h *Handlers) ObtenerMe(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		jsonError(w, "no autorizado", http.StatusUnauthorized)
		return
	}

	t, err := h.svc.ObtenerPorID(r.Context(), claims.TenantID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			jsonError(w, "tenant no encontrado", http.StatusNotFound)
		} else {
			slog.ErrorContext(r.Context(), "error obteniendo tenant", "err", err)
			jsonError(w, "error interno", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if claims.Rol != auth.RolDuenio {
		json.NewEncoder(w).Encode(t.VistaOperativa())
		return
	}
	json.NewEncoder(w).Encode(t)
}

func (h *Handlers) ActualizarMe(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		jsonError(w, "no autorizado", http.StatusUnauthorized)
		return
	}

	var input ActualizarTenantInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, "body inválido", http.StatusBadRequest)
		return
	}

	t, err := h.svc.Actualizar(r.Context(), claims.TenantID, input)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			jsonError(w, "tenant no encontrado", http.StatusNotFound)
			return
		}
		if errors.Is(err, ErrValidation) {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, ErrPlanRequired) {
			jsonError(w, err.Error(), http.StatusForbidden, "PLAN_LIMIT_REACHED", map[string]any{
				"recurso": "personalizacion",
				"limite":  0,
				"actual":  0,
			})
			return
		}
		slog.ErrorContext(r.Context(), "error actualizando tenant", "err", err)
		jsonError(w, "error actualizando negocio", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)
}

func (h *Handlers) EmailAdminDisponible(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")

	disponible, err := h.svc.EmailAdminDisponible(r.Context(), email)
	if err != nil {
		if errors.Is(err, ErrValidation) {
			jsonError(w, "Ingresá un correo de acceso válido.", http.StatusBadRequest)
			return
		}

		slog.ErrorContext(r.Context(), "error verificando disponibilidad de email", "err", err)
		jsonError(w, "No pudimos validar el correo de acceso.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(map[string]bool{"disponible": disponible})
}

func (h *Handlers) SlugDisponible(w http.ResponseWriter, r *http.Request) {
	disponible, err := h.svc.SlugDisponible(r.Context(), r.URL.Query().Get("slug"))
	if err != nil {
		if errors.Is(err, ErrValidation) {
			jsonError(w, "Usá solo letras, números y guiones en el nombre en URL.", http.StatusBadRequest)
			return
		}

		slog.ErrorContext(r.Context(), "error verificando disponibilidad de slug", "err", err)
		jsonError(w, "No pudimos validar el nombre en URL.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(map[string]bool{"disponible": disponible})
}

func (h *Handlers) ObtenerMiPlan(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		jsonError(w, "no autenticado", http.StatusUnauthorized)
		return
	}

	dto, err := h.svc.EstadoPlan(r.Context(), claims.TenantID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			jsonError(w, "tenant no encontrado", http.StatusNotFound)
			return
		}
		slog.ErrorContext(r.Context(), "error obteniendo estado de plan", "tenant_id", claims.TenantID, "error", err)
		jsonError(w, "error interno al obtener estado del plan", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(dto)
}

type SolicitarUpgradeInput struct {
	Nota string `json:"nota"`
}

func (h *Handlers) SolicitarUpgrade(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		jsonError(w, "no autenticado", http.StatusUnauthorized)
		return
	}

	var input SolicitarUpgradeInput
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&input)
	}

	t, err := h.svc.SolicitarUpgrade(r.Context(), claims.TenantID, input.Nota)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			jsonError(w, "tenant no encontrado", http.StatusNotFound)
			return
		}
		slog.ErrorContext(r.Context(), "error solicitando upgrade", "tenant_id", claims.TenantID, "error", err)
		jsonError(w, "error interno al solicitar upgrade", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(t)
}

func jsonError(w http.ResponseWriter, msg string, status int, extras ...any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	res := map[string]any{"error": msg}
	if len(extras) > 0 {
		if codigo, ok := extras[0].(string); ok && codigo != "" {
			res["codigo"] = codigo
		}
	}
	if len(extras) > 1 && extras[1] != nil {
		res["detalle"] = extras[1]
	}
	_ = json.NewEncoder(w).Encode(res)
}
