package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
)

const (
	maxRedirectURIs  = 10
	maxLargoRedirect = 2048
	maxLargoState    = 1024
)

type Usuarios interface {
	ObtenerUsuarioPorID(ctx context.Context, id string) (*auth.UsuarioAuth, error)
}

type Negocios interface {
	NombreNegocio(ctx context.Context, tenantID string) (string, error)
}

// AccesoPlan resuelve si el plan (o una excepción) del negocio incluye MCP.
type AccesoPlan interface {
	AccesoMCP(ctx context.Context, tenantID string) (bool, error)
}

type ParamsAutorizacion struct {
	ClientID, RedirectURI, ResponseType, CodeChallenge, CodeChallengeMethod, State, Scope, Resource string
}

type AccesoMCP struct {
	Claims     *auth.Claims
	ConexionID string
	Scope      string
}

type Service struct {
	store    Store
	usuarios Usuarios
	negocios Negocios
	acceso   AccesoPlan
	cfg      Config
	ahora    func() time.Time
}

func NuevoService(store Store, usuarios Usuarios, negocios Negocios, acceso AccesoPlan, cfg Config) *Service {
	return &Service{store: store, usuarios: usuarios, negocios: negocios, acceso: acceso, cfg: cfg, ahora: time.Now}
}

func (s *Service) RegistrarCliente(ctx context.Context, nombre string, redirectURIs []string) (*Cliente, error) {
	nombre = strings.TrimSpace(nombre)
	if nombre == "" {
		nombre = "Cliente MCP"
	}
	if runas := []rune(nombre); len(runas) > 100 {
		nombre = string(runas[:100])
	}
	if len(redirectURIs) == 0 || len(redirectURIs) > maxRedirectURIs {
		return nil, ErrRedirectInvalido
	}
	for _, u := range redirectURIs {
		if !redirectValido(u) {
			return nil, ErrRedirectInvalido
		}
	}

	c := &Cliente{
		ClientID:     "mcp_" + generarAleatorio(24),
		ClientName:   nombre,
		RedirectURIs: redirectURIs,
	}
	if err := s.store.CrearCliente(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

// IniciarAutorizacion devuelve el id de la solicitud y un nonce que el handler
// entrega al navegador en una cookie: la aprobación exige presentarlo.
func (s *Service) IniciarAutorizacion(ctx context.Context, p ParamsAutorizacion) (string, string, error) {
	cliente, err := s.store.ObtenerCliente(ctx, p.ClientID)
	if err != nil {
		return "", "", err
	}
	if !slices.Contains(cliente.RedirectURIs, p.RedirectURI) {
		return "", "", ErrRedirectInvalido
	}
	if p.Resource == "" {
		p.Resource = s.cfg.Resource
	}
	if p.ResponseType != "code" || p.CodeChallengeMethod != "S256" ||
		len(p.CodeChallenge) < 43 || len(p.CodeChallenge) > 128 || len(p.State) > maxLargoState ||
		strings.TrimRight(p.Resource, "/") != strings.TrimRight(s.cfg.Resource, "/") {
		return "", "", ErrSolicitudInvalida
	}
	if strings.TrimSpace(p.Scope) != "" && !slices.Contains(strings.Fields(p.Scope), ScopeLectura) {
		return "", "", ErrScopeInvalido
	}
	nonce := generarAleatorio(32)

	sol := &Solicitud{
		ClientID:      p.ClientID,
		RedirectURI:   p.RedirectURI,
		CodeChallenge: p.CodeChallenge,
		State:         p.State,
		Scope:         ScopeLectura,
		Resource:      s.cfg.Resource,
		NavegadorHash: hashToken(nonce),
		ExpiraAt:      s.ahora().Add(DuracionSolicitud),
	}
	if err := s.store.CrearSolicitud(ctx, sol); err != nil {
		return "", "", err
	}
	return sol.ID, nonce, nil
}

func (s *Service) URLConsentimiento(id string) string {
	return s.cfg.AppURL + "/conectar?solicitud=" + url.QueryEscape(id)
}

func (s *Service) solicitudPendiente(ctx context.Context, id string) (*Solicitud, error) {
	sol, err := s.store.ObtenerSolicitud(ctx, id)
	if err != nil {
		return nil, err
	}
	if sol.Estado != "pendiente" || !sol.ExpiraAt.After(s.ahora()) {
		return nil, ErrSolicitudNoEncontrada
	}
	return sol, nil
}

func (s *Service) VerSolicitud(ctx context.Context, id, nonce string, claims *auth.Claims) (*SolicitudVista, error) {
	sol, err := s.solicitudPendiente(ctx, id)
	if err != nil {
		return nil, err
	}
	cliente, err := s.store.ObtenerCliente(ctx, sol.ClientID)
	if err != nil {
		return nil, err
	}
	negocio, err := s.negocios.NombreNegocio(ctx, claims.TenantID)
	if err != nil {
		return nil, fmt.Errorf("obteniendo negocio: %w", err)
	}
	dominio := ""
	if u, err := url.Parse(sol.RedirectURI); err == nil {
		dominio = u.Host
	}
	return &SolicitudVista{
		ID:             sol.ID,
		ClientName:     cliente.ClientName,
		DominioRetorno: dominio,
		Negocio:        negocio,
		Permisos: []string{
			"Ver el estado del salón y las mesas",
			"Ver el estado de la cocina",
			"Ver el resumen del turno",
		},
		SoloLectura:    true,
		MismoNavegador: mismoNavegador(sol, nonce),
		ExpiraAt:       sol.ExpiraAt,
	}, nil
}

func (s *Service) Aprobar(ctx context.Context, id, nonce string, claims *auth.Claims) (string, error) {
	if claims.Rol != auth.RolDuenio {
		return "", ErrSoloDuenio
	}
	incluido, err := s.acceso.AccesoMCP(ctx, claims.TenantID)
	if err != nil {
		return "", fmt.Errorf("verificando acceso a MCP: %w", err)
	}
	if !incluido {
		return "", ErrPlanSinMCP
	}
	sol, err := s.solicitudPendiente(ctx, id)
	if err != nil {
		return "", err
	}
	if !mismoNavegador(sol, nonce) {
		return "", ErrOtroNavegador
	}
	codigo := generarAleatorio(32)
	if err := s.store.AprobarSolicitud(ctx, id, claims.UsuarioID, hashToken(codigo), s.ahora().Add(DuracionCodigo)); err != nil {
		return "", err
	}
	return s.urlRetorno(sol, map[string]string{"code": codigo})
}

func (s *Service) Rechazar(ctx context.Context, id string, claims *auth.Claims) (string, error) {
	sol, err := s.solicitudPendiente(ctx, id)
	if err != nil {
		return "", err
	}
	if err := s.store.RechazarSolicitud(ctx, id); err != nil {
		return "", err
	}
	return s.urlRetorno(sol, map[string]string{"error": "access_denied"})
}

func mismoNavegador(sol *Solicitud, nonce string) bool {
	return nonce != "" && sol.NavegadorHash != "" &&
		subtle.ConstantTimeCompare([]byte(hashToken(nonce)), []byte(sol.NavegadorHash)) == 1
}

// URLRetornoError arma la redirección de error para una solicitud que no
// llegó a persistirse (RFC 6749 §4.1.2.1).
func (s *Service) URLRetornoError(redirectURI, state, codigo string) string {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return redirectURI
	}
	q := u.Query()
	q.Set("error", codigo)
	if state != "" {
		q.Set("state", state)
	}
	q.Set("iss", s.cfg.Issuer)
	u.RawQuery = q.Encode()
	return u.String()
}

func (s *Service) urlRetorno(sol *Solicitud, extra map[string]string) (string, error) {
	u, err := url.Parse(sol.RedirectURI)
	if err != nil {
		return "", fmt.Errorf("redirect_uri inválido: %w", err)
	}
	q := u.Query()
	for k, v := range extra {
		q.Set(k, v)
	}
	if sol.State != "" {
		q.Set("state", sol.State)
	}
	q.Set("iss", s.cfg.Issuer)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (s *Service) CanjearCodigo(ctx context.Context, code, codeVerifier, clientID, redirectURI string) (*TokenRespuesta, error) {
	sol, err := s.store.CanjearCodigo(ctx, hashToken(code))
	if err != nil {
		return nil, err
	}
	if sol.ClientID != clientID || sol.RedirectURI != redirectURI || sol.UsuarioID == nil {
		return nil, ErrGrantInvalido
	}
	if len(codeVerifier) < 43 || len(codeVerifier) > 128 {
		return nil, ErrGrantInvalido
	}
	desafio := sha256.Sum256([]byte(codeVerifier))
	calculado := base64.RawURLEncoding.EncodeToString(desafio[:])
	if subtle.ConstantTimeCompare([]byte(calculado), []byte(sol.CodeChallenge)) != 1 {
		return nil, ErrGrantInvalido
	}

	usuario, err := s.usuarios.ObtenerUsuarioPorID(ctx, *sol.UsuarioID)
	if err != nil {
		if usuarioNoExiste(err) {
			return nil, ErrGrantInvalido
		}
		return nil, fmt.Errorf("obtener usuario: %w", err)
	}
	if usuario.Estado != "activo" || usuario.Rol != auth.RolDuenio {
		return nil, ErrGrantInvalido
	}

	conexion := &Conexion{
		ClientID:       sol.ClientID,
		UsuarioID:      usuario.ID,
		TenantID:       usuario.TenantID,
		Scope:          sol.Scope,
		SessionVersion: usuario.SessionVersion,
	}
	refresh := generarAleatorio(32)
	if err := s.store.CrearConexion(ctx, conexion, RefreshToken{
		TokenHash: hashToken(refresh),
		ExpiraAt:  s.ahora().Add(DuracionRefresh),
	}); err != nil {
		return nil, err
	}
	return s.respuestaToken(conexion, refresh)
}

func (s *Service) Refrescar(ctx context.Context, refreshToken, clientID string) (*TokenRespuesta, error) {
	nuevo := generarAleatorio(32)
	nuevoHash := hashToken(nuevo)
	conexion, err := s.store.RotarRefresh(ctx, hashToken(refreshToken), RefreshToken{
		TokenHash: nuevoHash,
		ExpiraAt:  s.ahora().Add(DuracionRefresh),
	})
	if err != nil {
		return nil, err
	}
	if conexion.ClientID != clientID {
		return nil, s.revocarYRechazar(ctx, nuevoHash)
	}
	usuario, err := s.usuarios.ObtenerUsuarioPorID(ctx, conexion.UsuarioID)
	if err != nil {
		if usuarioNoExiste(err) {
			return nil, s.revocarYRechazar(ctx, nuevoHash)
		}
		// Falla transitoria: no se revoca, el refresh nuevo sigue sirviendo.
		return nil, fmt.Errorf("obtener usuario: %w", err)
	}
	if usuario.Estado != "activo" || usuario.Rol != auth.RolDuenio ||
		usuario.TenantID != conexion.TenantID || usuario.SessionVersion != conexion.SessionVersion {
		return nil, s.revocarYRechazar(ctx, nuevoHash)
	}
	return s.respuestaToken(conexion, nuevo)
}

func (s *Service) revocarYRechazar(ctx context.Context, refreshHash string) error {
	if err := s.store.RevocarPorRefresh(ctx, refreshHash); err != nil {
		return err
	}
	return ErrGrantInvalido
}

func (s *Service) Revocar(ctx context.Context, token string) error {
	return s.store.RevocarPorRefresh(ctx, hashToken(token))
}

func (s *Service) ListarConexiones(ctx context.Context, claims *auth.Claims) ([]ConexionVista, error) {
	if claims.Rol != auth.RolDuenio {
		return nil, ErrSoloDuenio
	}
	return s.store.ListarConexiones(ctx, claims.UsuarioID)
}

func (s *Service) RevocarConexion(ctx context.Context, id string, claims *auth.Claims) error {
	if claims.Rol != auth.RolDuenio {
		return ErrSoloDuenio
	}
	return s.store.RevocarConexion(ctx, id, claims.UsuarioID)
}

func (s *Service) respuestaToken(c *Conexion, refresh string) (*TokenRespuesta, error) {
	access, err := s.emitirAccessToken(c)
	if err != nil {
		return nil, fmt.Errorf("firmando access token: %w", err)
	}
	return &TokenRespuesta{
		AccessToken:  access,
		TokenType:    "Bearer",
		ExpiresIn:    int(DuracionAccessToken.Seconds()),
		RefreshToken: refresh,
		Scope:        c.Scope,
	}, nil
}

func (s *Service) emitirAccessToken(c *Conexion) (string, error) {
	ahora := s.ahora()
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss":   s.cfg.Issuer,
		"aud":   s.cfg.Resource,
		"sub":   c.UsuarioID,
		"cid":   c.ID,
		"sv":    c.SessionVersion,
		"scope": ScopeLectura,
		"iat":   ahora.Unix(),
		"exp":   ahora.Add(DuracionAccessToken).Unix(),
	})
	return t.SignedString(s.cfg.Secreto)
}

func (s *Service) ValidarAccessToken(ctx context.Context, token string) (*AccesoMCP, error) {
	t, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		return s.cfg.Secreto, nil
	},
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithAudience(s.cfg.Resource),
		jwt.WithIssuer(s.cfg.Issuer),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(s.ahora),
	)
	if err != nil || !t.Valid {
		return nil, ErrTokenInvalido
	}
	mc, ok := t.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrTokenInvalido
	}
	sub, _ := mc["sub"].(string)
	cid, _ := mc["cid"].(string)
	svRaw, okSV := mc["sv"].(float64)
	if sub == "" || cid == "" || !okSV {
		return nil, ErrTokenInvalido
	}

	conexion, err := s.store.ObtenerConexion(ctx, cid)
	if err != nil {
		if errors.Is(err, ErrConexionNoEncontrada) {
			return nil, ErrTokenInvalido
		}
		return nil, err
	}
	if conexion.RevocadaAt != nil || conexion.UsuarioID != sub {
		return nil, ErrTokenInvalido
	}
	usuario, err := s.usuarios.ObtenerUsuarioPorID(ctx, sub)
	if err != nil {
		if usuarioNoExiste(err) {
			return nil, ErrTokenInvalido
		}
		return nil, fmt.Errorf("obtener usuario: %w", err)
	}
	if usuario.Estado != "activo" || usuario.Rol != auth.RolDuenio ||
		usuario.TenantID != conexion.TenantID || usuario.SessionVersion != int64(svRaw) {
		return nil, ErrTokenInvalido
	}

	if err := s.store.MarcarUso(ctx, cid); err != nil {
		slog.WarnContext(ctx, "no se pudo marcar el uso de la conexión MCP", "error", err, "conexion_id", cid)
	}
	return &AccesoMCP{
		Claims: &auth.Claims{
			UsuarioID:      usuario.ID,
			TenantID:       usuario.TenantID,
			Rol:            usuario.Rol,
			SessionVersion: usuario.SessionVersion,
		},
		ConexionID: cid,
		Scope:      conexion.Scope,
	}, nil
}

// usuarioNoExiste reconoce el "no hay fila" que auth.Store envuelve con %w.
func usuarioNoExiste(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

func generarAleatorio(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand no disponible: %v", err))
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func hashToken(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func redirectValido(raw string) bool {
	if len(raw) > maxLargoRedirect || strings.Contains(raw, "#") {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		return false
	}
	switch u.Scheme {
	case "https":
		return true
	case "http":
		h := u.Hostname()
		return h == "localhost" || h == "127.0.0.1" || h == "::1"
	}
	return false
}
