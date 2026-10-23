package oauth

import (
	"encoding/json"
	"errors"
	"html"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/ratelimit"
)

const maxBodyBytes = 16 << 10

type Handlers struct {
	svc             *Service
	limiteRegistro  *ratelimit.Limitador
	limiteToken     *ratelimit.Limitador
	limiteAutorizar *ratelimit.Limitador
}

func NuevosHandlers(svc *Service) *Handlers {
	return &Handlers{
		svc:             svc,
		limiteRegistro:  ratelimit.Nuevo(10, time.Minute),
		limiteToken:     ratelimit.Nuevo(30, time.Minute),
		limiteAutorizar: ratelimit.Nuevo(30, time.Minute),
	}
}

func (h *Handlers) MetadataRecurso(w http.ResponseWriter, r *http.Request) {
	jsonOK(w, http.StatusOK, map[string]any{
		"resource":                 h.svc.cfg.Resource,
		"authorization_servers":    []string{h.svc.cfg.Issuer},
		"scopes_supported":         []string{ScopeLectura},
		"bearer_methods_supported": []string{"header"},
		"resource_name":            "Fudd",
	})
}

func (h *Handlers) MetadataServidor(w http.ResponseWriter, r *http.Request) {
	iss := h.svc.cfg.Issuer
	jsonOK(w, http.StatusOK, map[string]any{
		"issuer":                                         iss,
		"authorization_endpoint":                         iss + "/oauth/authorize",
		"token_endpoint":                                 iss + "/oauth/token",
		"registration_endpoint":                          iss + "/oauth/register",
		"revocation_endpoint":                            iss + "/oauth/revoke",
		"response_types_supported":                       []string{"code"},
		"grant_types_supported":                          []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":               []string{"S256"},
		"token_endpoint_auth_methods_supported":          []string{"none"},
		"revocation_endpoint_auth_methods_supported":     []string{"none"},
		"scopes_supported":                               []string{ScopeLectura},
		"authorization_response_iss_parameter_supported": true,
	})
}

func (h *Handlers) Registrar(w http.ResponseWriter, r *http.Request) {
	if !h.limiteRegistro.Permitir(ratelimit.IPCliente(r)) {
		rateLimited(w)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	var in struct {
		ClientName              string   `json:"client_name"`
		RedirectURIs            []string `json:"redirect_uris"`
		TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		oauthError(w, http.StatusBadRequest, "invalid_client_metadata", "body inválido")
		return
	}
	if in.TokenEndpointAuthMethod != "" && in.TokenEndpointAuthMethod != "none" {
		oauthError(w, http.StatusBadRequest, "invalid_client_metadata", "solo se admite token_endpoint_auth_method none")
		return
	}

	c, err := h.svc.RegistrarCliente(r.Context(), in.ClientName, in.RedirectURIs)
	if err != nil {
		if errors.Is(err, ErrRedirectInvalido) {
			oauthError(w, http.StatusBadRequest, "invalid_redirect_uri", "redirect_uris inválidas")
			return
		}
		errorServidor(w, r, "error registrando cliente oauth", err)
		return
	}
	jsonOK(w, http.StatusCreated, map[string]any{
		"client_id":                  c.ClientID,
		"client_name":                c.ClientName,
		"redirect_uris":              c.RedirectURIs,
		"token_endpoint_auth_method": "none",
		"grant_types":                []string{"authorization_code", "refresh_token"},
		"response_types":             []string{"code"},
		"client_id_issued_at":        c.CreatedAt.Unix(),
	})
}

func (h *Handlers) Autorizar(w http.ResponseWriter, r *http.Request) {
	if !h.limiteAutorizar.Permitir(ratelimit.IPCliente(r)) {
		w.Header().Set("Retry-After", "60")
		paginaError(w, http.StatusTooManyRequests, "Demasiados intentos, reintentá en un minuto.")
		return
	}
	q := r.URL.Query()
	p := ParamsAutorizacion{
		ClientID:            q.Get("client_id"),
		RedirectURI:         q.Get("redirect_uri"),
		ResponseType:        q.Get("response_type"),
		CodeChallenge:       q.Get("code_challenge"),
		CodeChallengeMethod: q.Get("code_challenge_method"),
		State:               q.Get("state"),
		Scope:               q.Get("scope"),
		Resource:            q.Get("resource"),
	}
	id, nonce, err := h.svc.IniciarAutorizacion(r.Context(), p)
	switch {
	case err == nil:
		http.SetCookie(w, &http.Cookie{
			Name:     cookieSolicitud + id,
			Value:    nonce,
			Path:     "/oauth/solicitudes/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   int(DuracionSolicitud.Seconds()),
			Secure:   r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
		})
		http.Redirect(w, r, h.svc.URLConsentimiento(id), http.StatusFound)
	case errors.Is(err, ErrClienteInvalido), errors.Is(err, ErrRedirectInvalido):
		paginaError(w, http.StatusBadRequest, "No se pudo iniciar la conexión: la aplicación no está registrada correctamente.")
	case errors.Is(err, ErrSolicitudInvalida):
		http.Redirect(w, r, h.svc.URLRetornoError(p.RedirectURI, p.State, "invalid_request"), http.StatusFound)
	case errors.Is(err, ErrScopeInvalido):
		http.Redirect(w, r, h.svc.URLRetornoError(p.RedirectURI, p.State, "invalid_scope"), http.StatusFound)
	default:
		errorServidor(w, r, "error iniciando autorización oauth", err)
	}
}

func (h *Handlers) VerSolicitud(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		jsonError(w, "no autorizado", http.StatusUnauthorized)
		return
	}
	vista, err := h.svc.VerSolicitud(r.Context(), r.PathValue("id"), nonceSolicitud(r), claims)
	if err != nil {
		h.errorSolicitud(w, r, err)
		return
	}
	jsonOK(w, http.StatusOK, vista)
}

func (h *Handlers) Aprobar(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		jsonError(w, "no autorizado", http.StatusUnauthorized)
		return
	}
	url, err := h.svc.Aprobar(r.Context(), r.PathValue("id"), nonceSolicitud(r), claims)
	if err != nil {
		h.errorSolicitud(w, r, err)
		return
	}
	jsonOK(w, http.StatusOK, map[string]string{"redirect_url": url})
}

func (h *Handlers) Rechazar(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		jsonError(w, "no autorizado", http.StatusUnauthorized)
		return
	}
	url, err := h.svc.Rechazar(r.Context(), r.PathValue("id"), claims)
	if err != nil {
		h.errorSolicitud(w, r, err)
		return
	}
	jsonOK(w, http.StatusOK, map[string]string{"redirect_url": url})
}

func (h *Handlers) errorSolicitud(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrSoloDuenio):
		errorSoloDuenio(w)
	case errors.Is(err, ErrOtroNavegador):
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":  "Esta solicitud se inició en otro navegador. Volvé a conectar desde tu asistente de IA.",
			"codigo": "SOLICITUD_OTRO_NAVEGADOR",
		})
	case errors.Is(err, ErrSolicitudNoEncontrada):
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":  "La solicitud no existe o venció",
			"codigo": "SOLICITUD_INVALIDA",
		})
	default:
		slog.ErrorContext(r.Context(), "error en solicitud oauth", "error", err)
		jsonError(w, "error interno", http.StatusInternalServerError)
	}
}

func (h *Handlers) Token(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if !h.limiteToken.Permitir(ratelimit.IPCliente(r)) {
		rateLimited(w)
		return
	}
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type"))), "application/x-www-form-urlencoded") {
		oauthError(w, http.StatusBadRequest, "invalid_request", "Content-Type debe ser application/x-www-form-urlencoded")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := r.ParseForm(); err != nil {
		oauthError(w, http.StatusBadRequest, "invalid_request", "")
		return
	}
	f := r.PostForm

	var resp *TokenRespuesta
	var err error
	switch f.Get("grant_type") {
	case "authorization_code":
		code, verifier, clientID, redirectURI := f.Get("code"), f.Get("code_verifier"), f.Get("client_id"), f.Get("redirect_uri")
		if code == "" || verifier == "" || clientID == "" || redirectURI == "" {
			oauthError(w, http.StatusBadRequest, "invalid_request", "faltan parámetros")
			return
		}
		resp, err = h.svc.CanjearCodigo(r.Context(), code, verifier, clientID, redirectURI)
	case "refresh_token":
		refresh, clientID := f.Get("refresh_token"), f.Get("client_id")
		if refresh == "" || clientID == "" {
			oauthError(w, http.StatusBadRequest, "invalid_request", "faltan parámetros")
			return
		}
		resp, err = h.svc.Refrescar(r.Context(), refresh, clientID)
	default:
		oauthError(w, http.StatusBadRequest, "unsupported_grant_type", "")
		return
	}
	if err != nil {
		if errors.Is(err, ErrGrantInvalido) {
			oauthError(w, http.StatusBadRequest, "invalid_grant", "")
			return
		}
		errorServidor(w, r, "error en token oauth", err)
		return
	}
	jsonOK(w, http.StatusOK, resp)
}

func (h *Handlers) Revocar(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := r.ParseForm(); err != nil {
		oauthError(w, http.StatusBadRequest, "invalid_request", "")
		return
	}
	token := r.PostForm.Get("token")
	if token == "" {
		oauthError(w, http.StatusBadRequest, "invalid_request", "falta token")
		return
	}
	if err := h.svc.Revocar(r.Context(), token); err != nil {
		errorServidor(w, r, "error revocando token oauth", err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) ListarConexiones(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		jsonError(w, "no autorizado", http.StatusUnauthorized)
		return
	}
	lista, err := h.svc.ListarConexiones(r.Context(), claims)
	if err != nil {
		if errors.Is(err, ErrSoloDuenio) {
			errorSoloDuenio(w)
			return
		}
		slog.ErrorContext(r.Context(), "error listando conexiones oauth", "error", err)
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}
	if lista == nil {
		lista = []ConexionVista{}
	}
	jsonOK(w, http.StatusOK, lista)
}

func (h *Handlers) RevocarConexion(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		jsonError(w, "no autorizado", http.StatusUnauthorized)
		return
	}
	err := h.svc.RevocarConexion(r.Context(), r.PathValue("id"), claims)
	switch {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, ErrSoloDuenio):
		errorSoloDuenio(w)
	case errors.Is(err, ErrConexionNoEncontrada):
		jsonError(w, "conexión no encontrada", http.StatusNotFound)
	default:
		slog.ErrorContext(r.Context(), "error revocando conexión oauth", "error", err)
		jsonError(w, "error interno", http.StatusInternalServerError)
	}
}

const cookieSolicitud = "mcp_sol_"

// nonceSolicitud lee la cookie que Autorizar dejó en el navegador que inició la
// solicitud; sin cookie devuelve "" y el service lo trata como otro navegador.
func nonceSolicitud(r *http.Request) string {
	c, err := r.Cookie(cookieSolicitud + r.PathValue("id"))
	if err != nil {
		return ""
	}
	return c.Value
}

func paginaError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte("<!doctype html><html lang=\"es\"><meta charset=\"utf-8\"><title>Fudd</title><body><p>" +
		html.EscapeString(msg) + "</p></body></html>"))
}

func errorSoloDuenio(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": ErrSoloDuenio.Error(), "codigo": "MCP_SOLO_DUENIO"})
}

func rateLimited(w http.ResponseWriter) {
	w.Header().Set("Retry-After", "60")
	oauthError(w, http.StatusTooManyRequests, "rate_limited", "Demasiadas solicitudes, reintentá en un minuto")
}

func errorServidor(w http.ResponseWriter, r *http.Request, msg string, err error) {
	slog.ErrorContext(r.Context(), msg, "error", err)
	oauthError(w, http.StatusInternalServerError, "server_error", "")
}

func oauthError(w http.ResponseWriter, status int, codigo, descripcion string) {
	body := map[string]string{"error": codigo}
	if descripcion != "" {
		body["error_description"] = descripcion
	}
	jsonOK(w, status, body)
}

func jsonError(w http.ResponseWriter, msg string, status int) {
	jsonOK(w, status, map[string]string{"error": msg})
}

func jsonOK(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
