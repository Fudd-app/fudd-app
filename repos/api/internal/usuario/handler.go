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

func NuevosHandlers(svc *Service) *Handlers {
	return &Handlers{svc: svc}
}

func (h *Handlers) Invitar(w http.ResponseWriter, r *http.Request) {
	var input InvitarInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, "body inválido", http.StatusBadRequest)
		return
	}
	respuesta, err := h.svc.Invitar(r.Context(), auth.ClaimsFromContext(r.Context()), input)
	if err != nil {
		responderError(w, r, err)
		return
	}
	jsonRespuesta(w, respuesta, http.StatusCreated)
}

func (h *Handlers) Listar(w http.ResponseWriter, r *http.Request) {
	equipo, err := h.svc.Listar(r.Context(), auth.ClaimsFromContext(r.Context()))
	if err != nil {
		responderError(w, r, err)
		return
	}
	jsonRespuesta(w, equipo, http.StatusOK)
}

func (h *Handlers) Actualizar(w http.ResponseWriter, r *http.Request) {
	var input ActualizarUsuarioInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, "body inválido", http.StatusBadRequest)
		return
	}
	u, err := h.svc.Actualizar(r.Context(), auth.ClaimsFromContext(r.Context()), r.PathValue("id"), input)
	if err != nil {
		responderError(w, r, err)
		return
	}
	jsonRespuesta(w, u, http.StatusOK)
}

func (h *Handlers) Eliminar(w http.ResponseWriter, r *http.Request) {
	err := h.svc.Eliminar(r.Context(), auth.ClaimsFromContext(r.Context()), r.PathValue("id"))
	if err != nil {
		responderError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) Cancelar(w http.ResponseWriter, r *http.Request) {
	err := h.svc.Cancelar(r.Context(), auth.ClaimsFromContext(r.Context()), r.PathValue("id"))
	if err != nil {
		responderError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) Reenviar(w http.ResponseWriter, r *http.Request) {
	respuesta, err := h.svc.Reenviar(r.Context(), auth.ClaimsFromContext(r.Context()), r.PathValue("id"))
	if err != nil {
		responderError(w, r, err)
		return
	}
	jsonRespuesta(w, respuesta, http.StatusCreated)
}

func (h *Handlers) VistaPrevia(w http.ResponseWriter, r *http.Request) {
	i, err := h.svc.VistaPrevia(r.Context(), r.PathValue("token"))
	if err != nil {
		responderError(w, r, err)
		return
	}
	jsonRespuesta(w, i, http.StatusOK)
}

func (h *Handlers) Aceptar(w http.ResponseWriter, r *http.Request) {
	var input AceptarInvitacionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.CredencialGoogle == "" {
		jsonError(w, "credencial de Google requerida", http.StatusBadRequest)
		return
	}
	u, err := h.svc.Aceptar(r.Context(), r.PathValue("token"), input)
	if err != nil {
		responderError(w, r, err)
		return
	}
	auth.EscribirSesion(w, r, &auth.UsuarioAuth{
		ID: u.ID, TenantID: u.TenantID, Email: u.Email, Nombre: u.Nombre, Rol: u.Rol,
		SucursalID: u.SucursalID, Estado: u.Estado, SessionVersion: 1,
	})
}

func responderError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrValidation):
		jsonError(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, ErrForbidden):
		jsonError(w, "sin permisos para esta operación", http.StatusForbidden)
	case errors.Is(err, ErrNotFound):
		jsonError(w, "usuario no encontrado", http.StatusNotFound)
	case errors.Is(err, ErrEmailConflict):
		jsonError(w, err.Error(), http.StatusConflict)
	case errors.Is(err, ErrInvitacionExpirada):
		jsonError(w, err.Error(), http.StatusGone)
	case errors.Is(err, ErrInvitacionNoValida):
		jsonError(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, ErrEmailGoogleDistinto):
		jsonError(w, err.Error(), http.StatusForbidden)
	case errors.Is(err, auth.ErrGoogleNoConfigurado):
		jsonError(w, "El inicio con Google no está configurado.", http.StatusServiceUnavailable)
	case errors.Is(err, auth.ErrCredencialGoogleInvalida), errors.Is(err, auth.ErrEmailGoogleNoVerificado), errors.Is(err, auth.ErrIdentidadGoogleIncompleta):
		jsonError(w, "No pudimos validar tu cuenta de Google.", http.StatusUnauthorized)
	default:
		slog.ErrorContext(r.Context(), "error gestionando equipo", "err", err)
		jsonError(w, "error interno", http.StatusInternalServerError)
	}
}

func jsonRespuesta(w http.ResponseWriter, valor any, estado int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(estado)
	_ = json.NewEncoder(w).Encode(valor)
}

func jsonError(w http.ResponseWriter, mensaje string, estado int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(estado)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": mensaje})
}
