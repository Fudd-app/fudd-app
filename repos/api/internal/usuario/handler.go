package usuario

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

func (h *Handlers) Invitar(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	var input UsuarioInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, "body inválido", http.StatusBadRequest)
		return
	}

	resp, err := h.svc.Invitar(r.Context(), claims.TenantID, input)
	if err != nil {
		if errors.Is(err, ErrValidation) {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, ErrEmailConflict) || errors.Is(err, ErrUsernameConflict) {
			jsonError(w, err.Error(), http.StatusConflict)
			return
		}
		slog.ErrorContext(r.Context(), "error invitando usuario", "err", err)
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *Handlers) CrearStaff(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	var input CrearUsuarioStaffInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, "body inválido", http.StatusBadRequest)
		return
	}

	u, err := h.svc.CrearStaff(r.Context(), claims.TenantID, input)
	if err != nil {
		if errors.Is(err, ErrValidation) {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, ErrUsernameConflict) || errors.Is(err, ErrEmailConflict) {
			jsonError(w, err.Error(), http.StatusConflict)
			return
		}
		slog.ErrorContext(r.Context(), "error creando miembro de staff", "err", err)
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(u)
}

func (h *Handlers) Listar(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	usuarios, err := h.svc.Listar(r.Context(), claims.TenantID)
	if err != nil {
		slog.ErrorContext(r.Context(), "error listando equipo", "err", err)
		jsonError(w, "error listando equipo", http.StatusInternalServerError)
		return
	}
	jsonOK(w, usuarios)
}

func (h *Handlers) Actualizar(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	id := r.PathValue("id")
	var input ActualizarUsuarioInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, "body inválido", http.StatusBadRequest)
		return
	}

	u, err := h.svc.Actualizar(r.Context(), id, claims.TenantID, input)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			jsonError(w, "usuario no encontrado", http.StatusNotFound)
			return
		}
		if errors.Is(err, ErrValidation) {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, ErrUsernameConflict) || errors.Is(err, ErrEmailConflict) {
			jsonError(w, err.Error(), http.StatusConflict)
			return
		}
		slog.ErrorContext(r.Context(), "error actualizando usuario", "err", err)
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}

	jsonOK(w, u)
}

func (h *Handlers) ResetearPassword(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	id := r.PathValue("id")
	var input ResetPasswordInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, "body inválido", http.StatusBadRequest)
		return
	}

	err := h.svc.ResetearPassword(r.Context(), id, claims.TenantID, input.NuevaPassword)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			jsonError(w, "usuario no encontrado", http.StatusNotFound)
			return
		}
		if errors.Is(err, ErrValidation) {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		slog.ErrorContext(r.Context(), "error reseteando contraseña de usuario", "err", err)
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]bool{"ok": true})
}

func (h *Handlers) CambiarEstado(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	id := r.PathValue("id")

	if id == claims.UsuarioID {
		jsonError(w, "no puedes bloquear ni archivar tu propia cuenta", http.StatusBadRequest)
		return
	}

	var input CambiarEstadoUsuarioInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, "body inválido", http.StatusBadRequest)
		return
	}

	u, err := h.svc.CambiarEstado(r.Context(), id, claims.TenantID, input.Estado)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			jsonError(w, "usuario no encontrado", http.StatusNotFound)
			return
		}
		if errors.Is(err, ErrValidation) {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		slog.ErrorContext(r.Context(), "error cambiando estado de usuario", "err", err)
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}

	jsonOK(w, u)
}

func (h *Handlers) Eliminar(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	id := r.PathValue("id")

	// Prevenir que un usuario administrador se elimine a sí mismo mediante este endpoint
	if id == claims.UsuarioID {
		jsonError(w, "no puedes eliminarte a ti mismo del equipo", http.StatusBadRequest)
		return
	}

	err := h.svc.Eliminar(r.Context(), id, claims.TenantID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			jsonError(w, "usuario no encontrado", http.StatusNotFound)
			return
		}
		slog.ErrorContext(r.Context(), "error eliminando usuario del equipo", "err", err)
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
