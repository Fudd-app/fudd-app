package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
)

const msgUsuarioBloqueado = "Tu usuario ha sido suspendido temporalmente. Por favor, consultá con el encargado del local."

type Handlers struct {
	svc *Service
	// exponerLink habilita devolver el magic link en la respuesta HTTP.
	// Solo se activa fuera de producción y sin proveedor de email real, para
	// poder probar el login sin casilla de correo.
	exponerLink bool
}

func NuevosHandlers(svc *Service, exponerLink bool) *Handlers {
	return &Handlers{svc: svc, exponerLink: exponerLink}
}

func (h *Handlers) SolicitarLink(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Email == "" {
		jsonError(w, "email requerido", http.StatusBadRequest)
		return
	}

	link, err := h.svc.SolicitarLink(r.Context(), body.Email)
	if err != nil {
		if errors.Is(err, ErrUsuarioNoEncontrado) {
			jsonError(w, "No encontramos una cuenta registrada con ese correo.", http.StatusNotFound)
			return
		}
		if errors.Is(err, ErrUsuarioBloqueado) {
			jsonError(w, msgUsuarioBloqueado, http.StatusForbidden)
			return
		}
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}

	resp := map[string]string{"mensaje": "Te enviamos un enlace de acceso a tu correo."}
	if h.exponerLink && link != "" {
		resp["magic_link_dev"] = link
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (h *Handlers) VerificarToken(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		jsonError(w, "token requerido", http.StatusBadRequest)
		return
	}

	usuario, err := h.svc.VerificarToken(r.Context(), token)
	if err != nil {
		jsonError(w, err.Error(), http.StatusUnauthorized)
		return
	}

	h.crearSesion(w, r, usuario)
}

func (h *Handlers) AutenticarPassword(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "body inválido", http.StatusBadRequest)
		return
	}

	usuario, err := h.svc.AutenticarPassword(r.Context(), body.Email, body.Password)
	if err != nil {
		if errors.Is(err, ErrUsuarioBloqueado) {
			jsonError(w, msgUsuarioBloqueado, http.StatusForbidden)
			return
		}
		if errors.Is(err, ErrCredencialesInvalidas) {
			jsonError(w, "Correo o contraseña incorrectos.", http.StatusUnauthorized)
			return
		}
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}
	h.crearSesion(w, r, usuario)
}

func (h *Handlers) AutenticarGoogle(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Credencial string `json:"credential"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Credencial == "" {
		jsonError(w, "credencial de Google requerida", http.StatusBadRequest)
		return
	}

	usuario, err := h.svc.AutenticarGoogle(r.Context(), body.Credencial)
	if err != nil {
		switch {
		case errors.Is(err, ErrUsuarioBloqueado):
			jsonError(w, msgUsuarioBloqueado, http.StatusForbidden)
		case errors.Is(err, ErrUsuarioNoEncontrado):
			jsonError(w, "No encontramos una cuenta registrada con ese correo.", http.StatusNotFound)
		case errors.Is(err, ErrGoogleNoConfigurado):
			jsonError(w, "El inicio con Google no está configurado.", http.StatusServiceUnavailable)
		case errors.Is(err, ErrCredencialGoogleInvalida),
			errors.Is(err, ErrEmailGoogleNoVerificado),
			errors.Is(err, ErrIdentidadGoogleIncompleta):
			jsonError(w, "No pudimos validar tu cuenta de Google.", http.StatusUnauthorized)
		default:
			jsonError(w, "error interno", http.StatusInternalServerError)
		}
		return
	}

	h.crearSesion(w, r, usuario)
}

// ObtenerNegocioPublico devuelve los datos públicos de branding del negocio para /[slug]/login (US-84).
func (h *Handlers) ObtenerNegocioPublico(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	negocio, err := h.svc.ObtenerNegocioPublico(r.Context(), slug)
	if err != nil {
		if errors.Is(err, ErrNegocioNoEncontrado) {
			jsonError(w, "Comercio no encontrado.", http.StatusNotFound)
			return
		}
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(negocio)
}

// AutenticarStaffNegocio autentica al personal operativo dentro del ámbito estricto del tenant (US-84).
func (h *Handlers) AutenticarStaffNegocio(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	slug := r.PathValue("slug")

	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "body inválido", http.StatusBadRequest)
		return
	}

	res, err := h.svc.AutenticarStaffPorSlug(r.Context(), slug, body.Username, body.Password)
	if err != nil {
		switch {
		case errors.Is(err, ErrNegocioNoEncontrado):
			jsonError(w, "Comercio no encontrado.", http.StatusNotFound)
		case errors.Is(err, ErrUsuarioBloqueado):
			jsonError(w, msgUsuarioBloqueado, http.StatusForbidden)
		case errors.Is(err, ErrCredencialesInvalidas):
			jsonError(w, "Credenciales inválidas.", http.StatusUnauthorized)
		default:
			jsonError(w, "error interno", http.StatusInternalServerError)
		}
		return
	}

	secreto := os.Getenv("JWT_SECRET")
	if secreto == "" {
		jsonError(w, "error generando sesión", http.StatusInternalServerError)
		return
	}

	jwtToken, err := GenerarJWT(&Claims{
		UsuarioID:  res.Usuario.ID,
		TenantID:   res.Usuario.TenantID,
		SucursalID: res.Usuario.SucursalID,
		Rol:        res.Usuario.Rol,
	}, secreto)
	if err != nil {
		jsonError(w, "error generando sesión", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    jwtToken,
		HttpOnly: true,
		Secure:   r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" || os.Getenv("APP_ENV") == "production",
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		MaxAge:   60 * 60 * 24 * 30,
	})

	uname := ""
	if res.Usuario.Username != nil {
		uname = *res.Usuario.Username
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"token":       jwtToken,
		"redirect_to": res.RedirectTo,
		"usuario": UsuarioStaffSesion{
			ID:         res.Usuario.ID,
			TenantID:   res.Usuario.TenantID,
			SucursalID: res.Usuario.SucursalID,
			Username:   uname,
			Nombre:     res.Usuario.Nombre,
			Rol:        res.Usuario.Rol,
		},
	})
}

func (h *Handlers) crearSesion(w http.ResponseWriter, r *http.Request, usuario *UsuarioAuth) {
	secreto := os.Getenv("JWT_SECRET")
	if secreto == "" {
		jsonError(w, "error generando sesión", http.StatusInternalServerError)
		return
	}
	jwt, err := GenerarJWT(&Claims{
		UsuarioID:  usuario.ID,
		TenantID:   usuario.TenantID,
		SucursalID: usuario.SucursalID,
		Rol:        usuario.Rol,
	}, secreto)
	if err != nil {
		jsonError(w, "error generando sesión", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    jwt,
		HttpOnly: true,
		Secure:   r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" || os.Getenv("APP_ENV") == "production",
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		MaxAge:   60 * 60 * 24 * 30,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"token":       jwt,
		"redirect_to": RutaRedireccionPorRol(usuario.Rol),
	})
}

func jsonError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
