package oauth

import (
	"encoding/json"
	"errors"
	"html"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/ratelimit"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/tenant"
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
	if !h.limiteRegistro.Permitir(ratelimit.ClaveCliente(r)) {
		logRechazo(r, "register", "rate limit excedido")
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
		logRechazo(r, "register", "body de registro con JSON inválido: "+err.Error())
		oauthError(w, http.StatusBadRequest, "invalid_client_metadata", "body inválido")
		return
	}
	if in.TokenEndpointAuthMethod != "" && in.TokenEndpointAuthMethod != "none" {
		logRechazo(r, "register", "token_endpoint_auth_method "+strconv.Quote(in.TokenEndpointAuthMethod)+" no admitido, solo none",
			"client_name", in.ClientName)
		oauthError(w, http.StatusBadRequest, "invalid_client_metadata", "solo se admite token_endpoint_auth_method none")
		return
	}

	c, err := h.svc.RegistrarCliente(r.Context(), in.ClientName, in.RedirectURIs)
	if err != nil {
		if errors.Is(err, ErrRedirectInvalido) {
			logRechazo(r, "register", "redirect_uris inválidas: deben ser entre 1 y "+strconv.Itoa(maxRedirectURIs)+", https (o http en localhost), sin fragmento",
				"client_name", in.ClientName, "cantidad_redirect_uris", len(in.RedirectURIs))
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
	if !h.limiteAutorizar.Permitir(ratelimit.ClaveCliente(r)) {
		logRechazo(r, "authorize", "rate limit excedido", "client_id", r.URL.Query().Get("client_id"))
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
		// La web consume la API vía /api-proxy desde otro host: Domain hace que la
		// cookie viaje al dominio de la web, y Path "/" cubre la ruta proxeada.
		http.SetCookie(w, &http.Cookie{
			Name:     cookieSolicitud + id,
			Value:    nonce,
			Path:     "/",
			Domain:   h.svc.cfg.DominioCookie,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   int(DuracionSolicitud.Seconds()),
			Secure: r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" ||
				strings.EqualFold(os.Getenv("APP_ENV"), "production"),
		})
		slog.InfoContext(r.Context(), "oauth authorize",
			"client_id", p.ClientID,
			"client_name", h.svc.NombreCliente(r.Context(), p.ClientID),
			"redirect_host", hostDe(p.RedirectURI),
			"scope", p.Scope,
			"con_resource", p.Resource != "")
		http.Redirect(w, r, h.svc.URLConsentimiento(id), http.StatusFound)
	case errors.Is(err, ErrClienteInvalido):
		logRechazo(r, "authorize", err.Error(), "client_id", p.ClientID)
		paginaError(w, http.StatusBadRequest, "No se pudo iniciar la conexión: la aplicación no está registrada correctamente.")
	case errors.Is(err, ErrRedirectInvalido):
		logRechazo(r, "authorize", err.Error(), h.atributosCliente(r, p.ClientID)...)
		paginaError(w, http.StatusBadRequest, "No se pudo iniciar la conexión: la aplicación no está registrada correctamente.")
	case errors.Is(err, ErrSolicitudInvalida):
		logRechazo(r, "authorize", err.Error(), h.atributosCliente(r, p.ClientID)...)
		http.Redirect(w, r, h.svc.URLRetornoError(p.RedirectURI, p.State, "invalid_request"), http.StatusFound)
	case errors.Is(err, ErrScopeInvalido):
		logRechazo(r, "authorize", err.Error(), h.atributosCliente(r, p.ClientID)...)
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
		h.errorSolicitud(w, r, "solicitud", err)
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
		h.errorSolicitud(w, r, "aprobar", err)
		return
	}
	slog.InfoContext(r.Context(), "oauth aprobar", "solicitud_id", r.PathValue("id"), "usuario_id", claims.UsuarioID)
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
		h.errorSolicitud(w, r, "rechazar", err)
		return
	}
	slog.InfoContext(r.Context(), "oauth rechazar", "solicitud_id", r.PathValue("id"), "usuario_id", claims.UsuarioID)
	jsonOK(w, http.StatusOK, map[string]string{"redirect_url": url})
}

func (h *Handlers) errorSolicitud(w http.ResponseWriter, r *http.Request, endpoint string, err error) {
	id := r.PathValue("id")
	switch {
	case errors.Is(err, ErrSoloDuenio):
		logRechazo(r, endpoint, "el usuario no es el dueño del negocio", "solicitud_id", id)
		errorSoloDuenio(w)
	case errors.Is(err, ErrPlanSinMCP):
		logRechazo(r, endpoint, "el plan del negocio no incluye MCP", "solicitud_id", id)
		errorPlanSinMCP(w)
	case errors.Is(err, ErrOtroNavegador):
		logRechazo(r, endpoint, "la solicitud se inició en otro navegador o falta la cookie de la solicitud", "solicitud_id", id)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":  "Esta solicitud se inició en otro navegador. Volvé a conectar desde tu asistente de IA.",
			"codigo": "SOLICITUD_OTRO_NAVEGADOR",
		})
	case errors.Is(err, ErrSolicitudNoEncontrada):
		logRechazo(r, endpoint, "solicitud no encontrada, vencida o ya resuelta", "solicitud_id", id)
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
	if !h.limiteToken.Permitir(ratelimit.ClaveCliente(r)) {
		logRechazo(r, "token", "rate limit excedido")
		rateLimited(w)
		return
	}
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type"))), "application/x-www-form-urlencoded") {
		logRechazo(r, "token", "Content-Type debe ser application/x-www-form-urlencoded", "content_type", r.Header.Get("Content-Type"))
		oauthError(w, http.StatusBadRequest, "invalid_request", "Content-Type debe ser application/x-www-form-urlencoded")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := r.ParseForm(); err != nil {
		logRechazo(r, "token", "body del form ilegible: "+err.Error())
		oauthError(w, http.StatusBadRequest, "invalid_request", "")
		return
	}
	f := r.PostForm
	grantType := f.Get("grant_type")
	clientID, motivo := clientIDDeRequest(r, f.Get("client_id"))
	if motivo != "" {
		logRechazo(r, "token", motivo, "grant_type", grantType)
		oauthError(w, http.StatusBadRequest, "invalid_request", "")
		return
	}

	var resp *TokenRespuesta
	var err error
	switch grantType {
	case "authorization_code":
		code, verifier, redirectURI := f.Get("code"), f.Get("code_verifier"), f.Get("redirect_uri")
		if faltan := parametrosFaltantes(map[string]string{"code": code, "code_verifier": verifier, "client_id": clientID, "redirect_uri": redirectURI}); faltan != "" {
			logRechazo(r, "token", "faltan parámetros: "+faltan, "grant_type", grantType, "client_id", clientID)
			oauthError(w, http.StatusBadRequest, "invalid_request", "faltan parámetros")
			return
		}
		resp, err = h.svc.CanjearCodigo(r.Context(), code, verifier, clientID, redirectURI)
	case "refresh_token":
		refresh := f.Get("refresh_token")
		if faltan := parametrosFaltantes(map[string]string{"refresh_token": refresh, "client_id": clientID}); faltan != "" {
			logRechazo(r, "token", "faltan parámetros: "+faltan, "grant_type", grantType, "client_id", clientID)
			oauthError(w, http.StatusBadRequest, "invalid_request", "faltan parámetros")
			return
		}
		resp, err = h.svc.Refrescar(r.Context(), refresh, clientID)
	default:
		logRechazo(r, "token", "grant_type no soportado: "+strconv.Quote(grantType), "client_id", clientID)
		oauthError(w, http.StatusBadRequest, "unsupported_grant_type", "")
		return
	}
	if err != nil {
		if errors.Is(err, ErrGrantInvalido) {
			logRechazo(r, "token", err.Error(), append([]any{"grant_type", grantType}, h.atributosCliente(r, clientID)...)...)
			oauthError(w, http.StatusBadRequest, "invalid_grant", "")
			return
		}
		errorServidor(w, r, "error en token oauth", err)
		return
	}
	slog.InfoContext(r.Context(), "oauth token", "grant_type", grantType,
		"client_id", clientID, "client_name", h.svc.NombreCliente(r.Context(), clientID))
	jsonOK(w, http.StatusOK, resp)
}

func (h *Handlers) Revocar(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := r.ParseForm(); err != nil {
		logRechazo(r, "revoke", "body del form ilegible: "+err.Error())
		oauthError(w, http.StatusBadRequest, "invalid_request", "")
		return
	}
	// El client_id no autentica nada (clientes públicos); se resuelve solo para
	// rechazar la ambigüedad Basic vs form y para dejar rastro en el log.
	clientID, motivo := clientIDDeRequest(r, r.PostForm.Get("client_id"))
	if motivo != "" {
		logRechazo(r, "revoke", motivo)
		oauthError(w, http.StatusBadRequest, "invalid_request", "")
		return
	}
	token := r.PostForm.Get("token")
	if token == "" {
		logRechazo(r, "revoke", "faltan parámetros: token", "client_id", clientID)
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

// clientIDDeRequest toma el client_id del form o, si no viene, del usuario de
// Authorization: Basic (RFC 6749 §2.3.1: va codificado como form-urlencoded; la
// contraseña se ignora porque los clientes son públicos). Devuelve un motivo
// no vacío si el request es ambiguo o ilegible.
func clientIDDeRequest(r *http.Request, delForm string) (string, string) {
	usuario, _, ok := r.BasicAuth()
	if !ok || usuario == "" {
		return delForm, ""
	}
	delBasic, err := url.QueryUnescape(usuario)
	if err != nil {
		return "", "client_id de Authorization Basic con codificación inválida"
	}
	if delForm != "" && delForm != delBasic {
		return "", "client_id del form (" + delForm + ") distinto del de Authorization Basic (" + delBasic + ")"
	}
	return delBasic, ""
}

// parametrosFaltantes lista, en orden alfabético, los parámetros vacíos.
func parametrosFaltantes(params map[string]string) string {
	var faltan []string
	for nombre, valor := range params {
		if valor == "" {
			faltan = append(faltan, nombre)
		}
	}
	slices.Sort(faltan)
	return strings.Join(faltan, ", ")
}

// atributosCliente arma los atributos de log client_id y client_name (si se conoce).
func (h *Handlers) atributosCliente(r *http.Request, clientID string) []any {
	attrs := []any{"client_id", clientID}
	if nombre := h.svc.NombreCliente(r.Context(), clientID); nombre != "" {
		attrs = append(attrs, "client_name", nombre)
	}
	return attrs
}

// logRechazo deja rastro de todo rechazo OAuth. Los args nunca deben incluir
// tokens, códigos, code_verifier, code_challenge, cookies ni nonces.
func logRechazo(r *http.Request, endpoint, motivo string, args ...any) {
	slog.WarnContext(r.Context(), "oauth rechazo", append([]any{"endpoint", endpoint, "motivo", motivo}, args...)...)
}

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

// errorPlanSinMCP usa el mismo formato que tenant.RequerirCuota para que el
// frontend abra el modal de upgrade.
func errorPlanSinMCP(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_ = json.NewEncoder(w).Encode(tenant.ErrorCuotaEnvelope{
		Error:  "La conexión con asistentes de IA es exclusiva del plan Pro.",
		Codigo: "PLAN_LIMIT_REACHED",
		Detalle: tenant.DetalleErrorCuota{
			Recurso: string(tenant.RecursoMCP),
			Nombre:  tenant.NombreRecurso(tenant.RecursoMCP),
			Plan:    tenant.PlanFree,
			Accion:  "upgrade",
		},
	})
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
