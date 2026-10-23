package oauth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
)

const (
	verifierTest = "verifier-de-prueba-con-largo-suficiente-para-pkce-123"
	redirectTest = "https://claude.ai/api/mcp/auth_callback"
)

type reloj struct {
	mu sync.Mutex
	t  time.Time
}

func (r *reloj) Ahora() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.t
}

func (r *reloj) Avanzar(d time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.t = r.t.Add(d)
}

type memStore struct {
	mu          sync.Mutex
	reloj       *reloj
	seq         int
	clientes    map[string]*Cliente
	solicitudes map[string]*Solicitud
	conexiones  map[string]*Conexion
	refresh     map[string]*RefreshToken
}

func nuevoMemStore(r *reloj) *memStore {
	return &memStore{
		reloj:       r,
		clientes:    map[string]*Cliente{},
		solicitudes: map[string]*Solicitud{},
		conexiones:  map[string]*Conexion{},
		refresh:     map[string]*RefreshToken{},
	}
}

func (m *memStore) id() string {
	m.seq++
	return "id-" + strconv.Itoa(m.seq)
}

func (m *memStore) CrearCliente(_ context.Context, c *Cliente) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	c.ID, c.CreatedAt = m.id(), m.reloj.Ahora()
	cp := *c
	m.clientes[c.ClientID] = &cp
	return nil
}

func (m *memStore) ObtenerCliente(_ context.Context, clientID string) (*Cliente, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.clientes[clientID]
	if !ok {
		return nil, ErrClienteInvalido
	}
	cp := *c
	return &cp, nil
}

func (m *memStore) CrearSolicitud(_ context.Context, s *Solicitud) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s.ID, s.CreatedAt, s.Estado = m.id(), m.reloj.Ahora(), "pendiente"
	cp := *s
	m.solicitudes[s.ID] = &cp
	return nil
}

func (m *memStore) ObtenerSolicitud(_ context.Context, id string) (*Solicitud, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.solicitudes[id]
	if !ok {
		return nil, ErrSolicitudNoEncontrada
	}
	cp := *s
	return &cp, nil
}

func (m *memStore) AprobarSolicitud(_ context.Context, id, usuarioID, codigoHash string, codigoExpira time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.solicitudes[id]
	if !ok || s.Estado != "pendiente" || !s.ExpiraAt.After(m.reloj.Ahora()) {
		return ErrSolicitudNoEncontrada
	}
	s.Estado, s.UsuarioID, s.CodigoHash, s.CodigoExpiraAt = "aprobada", &usuarioID, &codigoHash, &codigoExpira
	return nil
}

func (m *memStore) RechazarSolicitud(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.solicitudes[id]
	if !ok || s.Estado != "pendiente" || !s.ExpiraAt.After(m.reloj.Ahora()) {
		return ErrSolicitudNoEncontrada
	}
	s.Estado = "rechazada"
	return nil
}

func (m *memStore) CanjearCodigo(_ context.Context, codigoHash string) (*Solicitud, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.solicitudes {
		if s.CodigoHash != nil && *s.CodigoHash == codigoHash && s.Estado == "aprobada" &&
			s.CodigoExpiraAt.After(m.reloj.Ahora()) {
			s.Estado = "canjeada"
			cp := *s
			return &cp, nil
		}
	}
	return nil, ErrGrantInvalido
}

func (m *memStore) CrearConexion(_ context.Context, c *Conexion, r RefreshToken) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	c.ID, c.CreatedAt = m.id(), m.reloj.Ahora()
	cp := *c
	if cl, ok := m.clientes[c.ClientID]; ok {
		cp.ClientName = cl.ClientName
	}
	m.conexiones[c.ID] = &cp
	r.ConexionID = c.ID
	m.refresh[r.TokenHash] = &r
	return nil
}

func (m *memStore) ObtenerConexion(_ context.Context, id string) (*Conexion, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.conexiones[id]
	if !ok {
		return nil, ErrConexionNoEncontrada
	}
	cp := *c
	return &cp, nil
}

func (m *memStore) revocar(id string) {
	if c, ok := m.conexiones[id]; ok && c.RevocadaAt == nil {
		ahora := m.reloj.Ahora()
		c.RevocadaAt = &ahora
	}
}

func (m *memStore) RotarRefresh(_ context.Context, viejoHash string, nuevo RefreshToken) (*Conexion, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.refresh[viejoHash]
	if !ok {
		return nil, ErrGrantInvalido
	}
	if r.UsadoAt != nil {
		m.revocar(r.ConexionID)
		return nil, ErrGrantInvalido
	}
	if !r.ExpiraAt.After(m.reloj.Ahora()) {
		return nil, ErrGrantInvalido
	}
	c := m.conexiones[r.ConexionID]
	if c.RevocadaAt != nil {
		return nil, ErrGrantInvalido
	}
	ahora := m.reloj.Ahora()
	r.UsadoAt = &ahora
	nuevo.ConexionID = c.ID
	m.refresh[nuevo.TokenHash] = &nuevo
	c.UltimoUsoAt = &ahora
	cp := *c
	return &cp, nil
}

func (m *memStore) RevocarPorRefresh(_ context.Context, tokenHash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r, ok := m.refresh[tokenHash]; ok {
		m.revocar(r.ConexionID)
	}
	return nil
}

func (m *memStore) ListarConexiones(_ context.Context, usuarioID string) ([]ConexionVista, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	lista := []ConexionVista{}
	for _, c := range m.conexiones {
		if c.UsuarioID == usuarioID && c.RevocadaAt == nil {
			lista = append(lista, ConexionVista{ID: c.ID, ClientName: c.ClientName, CreadoAt: c.CreatedAt, UltimoUsoAt: c.UltimoUsoAt})
		}
	}
	return lista, nil
}

func (m *memStore) RevocarConexion(_ context.Context, id, usuarioID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.conexiones[id]
	if !ok || c.UsuarioID != usuarioID || c.RevocadaAt != nil {
		return ErrConexionNoEncontrada
	}
	m.revocar(id)
	return nil
}

func (m *memStore) MarcarUso(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c, ok := m.conexiones[id]; ok {
		ahora := m.reloj.Ahora()
		c.UltimoUsoAt = &ahora
	}
	return nil
}

type memUsuarios struct {
	mu sync.Mutex
	m  map[string]*auth.UsuarioAuth
}

func (u *memUsuarios) ObtenerUsuarioPorID(_ context.Context, id string) (*auth.UsuarioAuth, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	us, ok := u.m[id]
	if !ok {
		return nil, errors.New("usuario no encontrado")
	}
	cp := *us
	return &cp, nil
}

func (u *memUsuarios) modificar(id string, f func(*auth.UsuarioAuth)) {
	u.mu.Lock()
	defer u.mu.Unlock()
	f(u.m[id])
}

type memNegocios struct{}

func (memNegocios) NombreNegocio(_ context.Context, tenantID string) (string, error) {
	return "Negocio " + tenantID, nil
}

type entorno struct {
	svc      *Service
	store    *memStore
	usuarios *memUsuarios
	reloj    *reloj
}

func claimsDuenio() *auth.Claims {
	return &auth.Claims{UsuarioID: "u-duenio", TenantID: "t-1", Rol: auth.RolDuenio, SessionVersion: 3}
}

func nuevoEntorno(t *testing.T) *entorno {
	t.Helper()
	t.Setenv("JWT_SECRET", "secreto-de-test")
	t.Setenv("MCP_BASE_URL", "https://mcp.test")
	t.Setenv("APP_URL", "https://app.test")
	cfg, err := ConfigDesdeEntorno()
	if err != nil {
		t.Fatal(err)
	}
	r := &reloj{t: time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)}
	st := nuevoMemStore(r)
	us := &memUsuarios{m: map[string]*auth.UsuarioAuth{
		"u-duenio": {ID: "u-duenio", TenantID: "t-1", Rol: auth.RolDuenio, Estado: "activo", SessionVersion: 3},
		"u-otro":   {ID: "u-otro", TenantID: "t-2", Rol: auth.RolDuenio, Estado: "activo", SessionVersion: 1},
	}}
	svc := NuevoService(st, us, memNegocios{}, cfg)
	svc.ahora = r.Ahora
	return &entorno{svc: svc, store: st, usuarios: us, reloj: r}
}

func desafioTest() string {
	h := sha256.Sum256([]byte(verifierTest))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

func (e *entorno) registrar(t *testing.T) *Cliente {
	t.Helper()
	c, err := e.svc.RegistrarCliente(context.Background(), "Claude", []string{redirectTest})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func (e *entorno) paramsOK(clientID string) ParamsAutorizacion {
	return ParamsAutorizacion{
		ClientID: clientID, RedirectURI: redirectTest, ResponseType: "code",
		CodeChallenge: desafioTest(), CodeChallengeMethod: "S256", State: "xyz",
		Scope: ScopeLectura, Resource: e.svc.cfg.Resource,
	}
}

// aprobada deja una solicitud aprobada y devuelve el código en claro.
func (e *entorno) aprobada(t *testing.T, clientID string) string {
	t.Helper()
	id, err := e.svc.IniciarAutorizacion(context.Background(), e.paramsOK(clientID))
	if err != nil {
		t.Fatal(err)
	}
	redir, err := e.svc.Aprobar(context.Background(), id, claimsDuenio())
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(redir)
	return u.Query().Get("code")
}

func (e *entorno) conectar(t *testing.T) (*TokenRespuesta, string) {
	t.Helper()
	c := e.registrar(t)
	code := e.aprobada(t, c.ClientID)
	resp, err := e.svc.CanjearCodigo(context.Background(), code, verifierTest, c.ClientID, redirectTest)
	if err != nil {
		t.Fatal(err)
	}
	return resp, c.ClientID
}

func TestRegistrarCliente(t *testing.T) {
	e := nuevoEntorno(t)
	ctx := context.Background()
	casos := []struct {
		nombre string
		uris   []string
		ok     bool
	}{
		{"https", []string{"https://claude.ai/cb"}, true},
		{"http localhost", []string{"http://localhost:3333/cb"}, true},
		{"http 127.0.0.1", []string{"http://127.0.0.1:3333/cb"}, true},
		{"http no loopback", []string{"http://ejemplo.com/cb"}, false},
		{"fragmento", []string{"https://claude.ai/cb#x"}, false},
		{"cero uris", nil, false},
		{"once uris", []string{"https://a.com/1", "https://a.com/2", "https://a.com/3", "https://a.com/4", "https://a.com/5",
			"https://a.com/6", "https://a.com/7", "https://a.com/8", "https://a.com/9", "https://a.com/10", "https://a.com/11"}, false},
	}
	for _, c := range casos {
		_, err := e.svc.RegistrarCliente(ctx, "x", c.uris)
		if c.ok && err != nil {
			t.Errorf("%s: error inesperado %v", c.nombre, err)
		}
		if !c.ok && !errors.Is(err, ErrRedirectInvalido) {
			t.Errorf("%s: esperaba ErrRedirectInvalido, got %v", c.nombre, err)
		}
	}
	cl, _ := e.svc.RegistrarCliente(ctx, "   ", []string{"https://a.com/cb"})
	if cl.ClientName != "Cliente MCP" || !strings.HasPrefix(cl.ClientID, "mcp_") {
		t.Errorf("cliente: %+v", cl)
	}
}

func TestIniciarAutorizacion(t *testing.T) {
	e := nuevoEntorno(t)
	ctx := context.Background()
	c := e.registrar(t)

	if _, err := e.svc.IniciarAutorizacion(ctx, ParamsAutorizacion{ClientID: "nope"}); !errors.Is(err, ErrClienteInvalido) {
		t.Errorf("cliente inexistente: %v", err)
	}
	mod := func(f func(*ParamsAutorizacion)) error {
		p := e.paramsOK(c.ClientID)
		f(&p)
		_, err := e.svc.IniciarAutorizacion(ctx, p)
		return err
	}
	if err := mod(func(p *ParamsAutorizacion) { p.RedirectURI = "https://otro.com/cb" }); !errors.Is(err, ErrRedirectInvalido) {
		t.Errorf("redirect distinto: %v", err)
	}
	if err := mod(func(p *ParamsAutorizacion) { p.CodeChallenge, p.CodeChallengeMethod = "", "" }); !errors.Is(err, ErrSolicitudInvalida) {
		t.Errorf("sin PKCE: %v", err)
	}
	if err := mod(func(p *ParamsAutorizacion) { p.CodeChallengeMethod = "plain" }); !errors.Is(err, ErrSolicitudInvalida) {
		t.Errorf("plain: %v", err)
	}
	if err := mod(func(p *ParamsAutorizacion) { p.Resource = "https://otro.com/mcp" }); !errors.Is(err, ErrSolicitudInvalida) {
		t.Errorf("resource: %v", err)
	}
	if err := mod(func(p *ParamsAutorizacion) { p.Scope = "admin" }); !errors.Is(err, ErrScopeInvalido) {
		t.Errorf("scope: %v", err)
	}

	id, err := e.svc.IniciarAutorizacion(ctx, e.paramsOK(c.ClientID))
	if err != nil {
		t.Fatal(err)
	}
	sol, _ := e.store.ObtenerSolicitud(ctx, id)
	if sol.Estado != "pendiente" || sol.Scope != ScopeLectura {
		t.Errorf("solicitud: %+v", sol)
	}
	if got := e.svc.URLConsentimiento(id); got != "https://app.test/conectar?solicitud="+id {
		t.Errorf("url consentimiento: %s", got)
	}
}

func TestVerSolicitud(t *testing.T) {
	e := nuevoEntorno(t)
	c := e.registrar(t)
	id, _ := e.svc.IniciarAutorizacion(context.Background(), e.paramsOK(c.ClientID))
	v, err := e.svc.VerSolicitud(context.Background(), id, claimsDuenio())
	if err != nil {
		t.Fatal(err)
	}
	if v.ClientName != "Claude" || v.DominioRetorno != "claude.ai" || v.Negocio != "Negocio t-1" || !v.SoloLectura || len(v.Permisos) != 3 {
		t.Errorf("vista: %+v", v)
	}
}

func TestAprobar(t *testing.T) {
	e := nuevoEntorno(t)
	ctx := context.Background()
	c, _ := e.svc.RegistrarCliente(ctx, "Claude", []string{"https://claude.ai/cb?x=1"})
	p := e.paramsOK(c.ClientID)
	p.RedirectURI = "https://claude.ai/cb?x=1"
	id, err := e.svc.IniciarAutorizacion(ctx, p)
	if err != nil {
		t.Fatal(err)
	}

	mozo := &auth.Claims{UsuarioID: "u-mozo", TenantID: "t-1", Rol: auth.RolMozo}
	if _, err := e.svc.Aprobar(ctx, id, mozo); !errors.Is(err, ErrSoloDuenio) {
		t.Errorf("mozo: %v", err)
	}
	redir, err := e.svc.Aprobar(ctx, id, claimsDuenio())
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(redir)
	q := u.Query()
	if q.Get("x") != "1" || q.Get("code") == "" || q.Get("state") != "xyz" || q.Get("iss") != "https://mcp.test" {
		t.Errorf("redirect: %s", redir)
	}
	if _, err := e.svc.Aprobar(ctx, id, claimsDuenio()); !errors.Is(err, ErrSolicitudNoEncontrada) {
		t.Errorf("segunda aprobación: %v", err)
	}

	id2, _ := e.svc.IniciarAutorizacion(ctx, p)
	e.reloj.Avanzar(DuracionSolicitud + time.Second)
	if _, err := e.svc.Aprobar(ctx, id2, claimsDuenio()); !errors.Is(err, ErrSolicitudNoEncontrada) {
		t.Errorf("vencida: %v", err)
	}
}

func TestRechazar(t *testing.T) {
	e := nuevoEntorno(t)
	c := e.registrar(t)
	id, _ := e.svc.IniciarAutorizacion(context.Background(), e.paramsOK(c.ClientID))
	redir, err := e.svc.Rechazar(context.Background(), id, &auth.Claims{Rol: auth.RolMozo})
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(redir)
	if u.Query().Get("error") != "access_denied" || u.Query().Get("state") != "xyz" || u.Query().Get("iss") == "" {
		t.Errorf("redirect: %s", redir)
	}
}

func TestCanjearCodigo(t *testing.T) {
	ctx := context.Background()
	casos := []struct {
		nombre string
		f      func(e *entorno, code, clientID string) (string, string, string, string) // code, verifier, client, redirect
	}{
		{"PKCE incorrecto", func(e *entorno, code, cl string) (string, string, string, string) {
			return code, strings.Repeat("x", 50), cl, redirectTest
		}},
		{"redirect distinto", func(e *entorno, code, cl string) (string, string, string, string) {
			return code, verifierTest, cl, "https://otro.com/cb"
		}},
		{"client distinto", func(e *entorno, code, cl string) (string, string, string, string) {
			return code, verifierTest, "mcp_otro", redirectTest
		}},
		{"código vencido", func(e *entorno, code, cl string) (string, string, string, string) {
			e.reloj.Avanzar(61 * time.Second)
			return code, verifierTest, cl, redirectTest
		}},
		{"usuario bloqueado", func(e *entorno, code, cl string) (string, string, string, string) {
			e.usuarios.modificar("u-duenio", func(u *auth.UsuarioAuth) { u.Estado = "bloqueado" })
			return code, verifierTest, cl, redirectTest
		}},
	}
	for _, c := range casos {
		e := nuevoEntorno(t)
		cl := e.registrar(t)
		code := e.aprobada(t, cl.ClientID)
		code, ver, cid, red := c.f(e, code, cl.ClientID)
		if _, err := e.svc.CanjearCodigo(ctx, code, ver, cid, red); !errors.Is(err, ErrGrantInvalido) {
			t.Errorf("%s: esperaba ErrGrantInvalido, got %v", c.nombre, err)
		}
	}

	e := nuevoEntorno(t)
	cl := e.registrar(t)
	code := e.aprobada(t, cl.ClientID)
	resp, err := e.svc.CanjearCodigo(ctx, code, verifierTest, cl.ClientID, redirectTest)
	if err != nil {
		t.Fatal(err)
	}
	if resp.AccessToken == "" || resp.RefreshToken == "" || resp.TokenType != "Bearer" || resp.ExpiresIn != 900 || resp.Scope != ScopeLectura {
		t.Errorf("respuesta: %+v", resp)
	}
	if _, err := e.svc.CanjearCodigo(ctx, code, verifierTest, cl.ClientID, redirectTest); !errors.Is(err, ErrGrantInvalido) {
		t.Errorf("código reusado: %v", err)
	}
}

func TestRefrescar(t *testing.T) {
	ctx := context.Background()

	t.Run("rota y el viejo deja de servir", func(t *testing.T) {
		e := nuevoEntorno(t)
		r1, cid := e.conectar(t)
		r2, err := e.svc.Refrescar(ctx, r1.RefreshToken, cid)
		if err != nil {
			t.Fatal(err)
		}
		if r2.RefreshToken == r1.RefreshToken {
			t.Error("el refresh no rotó")
		}
		if _, err := e.svc.ValidarAccessToken(ctx, r2.AccessToken); err != nil {
			t.Errorf("access nuevo: %v", err)
		}
		if _, err := e.svc.Refrescar(ctx, r1.RefreshToken, cid); !errors.Is(err, ErrGrantInvalido) {
			t.Errorf("el viejo no debería servir: %v", err)
		}
	})

	t.Run("reuso revoca la conexión", func(t *testing.T) {
		e := nuevoEntorno(t)
		r1, cid := e.conectar(t)
		r2, _ := e.svc.Refrescar(ctx, r1.RefreshToken, cid)
		if _, err := e.svc.Refrescar(ctx, r1.RefreshToken, cid); !errors.Is(err, ErrGrantInvalido) {
			t.Errorf("reuso: %v", err)
		}
		if _, err := e.svc.Refrescar(ctx, r2.RefreshToken, cid); !errors.Is(err, ErrGrantInvalido) {
			t.Errorf("el nuevo debería haber quedado inválido: %v", err)
		}
		if _, err := e.svc.ValidarAccessToken(ctx, r2.AccessToken); !errors.Is(err, ErrTokenInvalido) {
			t.Errorf("access de conexión revocada: %v", err)
		}
	})

	t.Run("session_version cambiada", func(t *testing.T) {
		e := nuevoEntorno(t)
		r1, cid := e.conectar(t)
		e.usuarios.modificar("u-duenio", func(u *auth.UsuarioAuth) { u.SessionVersion++ })
		if _, err := e.svc.Refrescar(ctx, r1.RefreshToken, cid); !errors.Is(err, ErrGrantInvalido) {
			t.Errorf("sv: %v", err)
		}
		for _, c := range e.store.conexiones {
			if c.RevocadaAt == nil {
				t.Error("la conexión debería estar revocada")
			}
		}
	})

	t.Run("refresh vencido", func(t *testing.T) {
		e := nuevoEntorno(t)
		r1, cid := e.conectar(t)
		e.reloj.Avanzar(DuracionRefresh + time.Second)
		if _, err := e.svc.Refrescar(ctx, r1.RefreshToken, cid); !errors.Is(err, ErrGrantInvalido) {
			t.Errorf("vencido: %v", err)
		}
	})

	t.Run("client distinto", func(t *testing.T) {
		e := nuevoEntorno(t)
		r1, _ := e.conectar(t)
		if _, err := e.svc.Refrescar(ctx, r1.RefreshToken, "mcp_otro"); !errors.Is(err, ErrGrantInvalido) {
			t.Errorf("client: %v", err)
		}
	})
}

func TestValidarAccessToken(t *testing.T) {
	ctx := context.Background()
	e := nuevoEntorno(t)
	resp, _ := e.conectar(t)

	acceso, err := e.svc.ValidarAccessToken(ctx, resp.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	if acceso.Claims.TenantID != "t-1" || acceso.Claims.UsuarioID != "u-duenio" || acceso.Claims.Rol != auth.RolDuenio {
		t.Errorf("claims: %+v", acceso.Claims)
	}

	// Token de sesión web firmado con JWT_SECRET directo.
	sesion, _ := auth.GenerarJWT(claimsDuenio(), "secreto-de-test")
	if _, err := e.svc.ValidarAccessToken(ctx, sesion); !errors.Is(err, ErrTokenInvalido) {
		t.Errorf("token de sesión: %v", err)
	}

	firmar := func(mod func(jwt.MapClaims)) string {
		mc := jwt.MapClaims{
			"iss": e.svc.cfg.Issuer, "aud": e.svc.cfg.Resource, "sub": "u-duenio", "cid": acceso.ConexionID,
			"sv": 3, "scope": ScopeLectura, "iat": e.reloj.Ahora().Unix(), "exp": e.reloj.Ahora().Add(time.Minute).Unix(),
		}
		mod(mc)
		s, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, mc).SignedString(e.svc.cfg.Secreto)
		return s
	}
	if _, err := e.svc.ValidarAccessToken(ctx, firmar(func(mc jwt.MapClaims) {})); err != nil {
		t.Errorf("token armado a mano debería validar: %v", err)
	}
	if _, err := e.svc.ValidarAccessToken(ctx, firmar(func(mc jwt.MapClaims) { mc["aud"] = "https://otro/mcp" })); !errors.Is(err, ErrTokenInvalido) {
		t.Errorf("aud: %v", err)
	}
	if _, err := e.svc.ValidarAccessToken(ctx, firmar(func(mc jwt.MapClaims) { mc["sv"] = 2 })); !errors.Is(err, ErrTokenInvalido) {
		t.Errorf("sv: %v", err)
	}
	if _, err := e.svc.ValidarAccessToken(ctx, firmar(func(mc jwt.MapClaims) { delete(mc, "exp") })); !errors.Is(err, ErrTokenInvalido) {
		t.Errorf("sin exp: %v", err)
	}

	e.usuarios.modificar("u-duenio", func(u *auth.UsuarioAuth) { u.Rol = auth.RolMozo })
	if _, err := e.svc.ValidarAccessToken(ctx, resp.AccessToken); !errors.Is(err, ErrTokenInvalido) {
		t.Errorf("rol cambiado: %v", err)
	}
	e.usuarios.modificar("u-duenio", func(u *auth.UsuarioAuth) { u.Rol, u.Estado = auth.RolDuenio, "bloqueado" })
	if _, err := e.svc.ValidarAccessToken(ctx, resp.AccessToken); !errors.Is(err, ErrTokenInvalido) {
		t.Errorf("bloqueado: %v", err)
	}
	e.usuarios.modificar("u-duenio", func(u *auth.UsuarioAuth) { u.Estado = "activo"; u.SessionVersion = 9 })
	if _, err := e.svc.ValidarAccessToken(ctx, resp.AccessToken); !errors.Is(err, ErrTokenInvalido) {
		t.Errorf("session_version: %v", err)
	}
	e.usuarios.modificar("u-duenio", func(u *auth.UsuarioAuth) { u.SessionVersion = 3 })
	if _, err := e.svc.ValidarAccessToken(ctx, resp.AccessToken); err != nil {
		t.Errorf("restaurado: %v", err)
	}

	e.reloj.Avanzar(DuracionAccessToken + time.Second)
	if _, err := e.svc.ValidarAccessToken(ctx, resp.AccessToken); !errors.Is(err, ErrTokenInvalido) {
		t.Errorf("expirado: %v", err)
	}
}

func TestValidarAccessToken_ConexionRevocada(t *testing.T) {
	e := nuevoEntorno(t)
	resp, _ := e.conectar(t)
	e.store.mu.Lock()
	for id := range e.store.conexiones {
		e.store.revocar(id)
	}
	e.store.mu.Unlock()
	if _, err := e.svc.ValidarAccessToken(context.Background(), resp.AccessToken); !errors.Is(err, ErrTokenInvalido) {
		t.Errorf("revocada: %v", err)
	}
}

func TestAuthValidarJWT_RechazaTokenMCP(t *testing.T) {
	e := nuevoEntorno(t)
	resp, _ := e.conectar(t)
	if _, err := auth.ValidarJWT(resp.AccessToken, "secreto-de-test"); err == nil {
		t.Error("auth.ValidarJWT no debería aceptar un access token MCP")
	}
}

func TestConexiones(t *testing.T) {
	ctx := context.Background()
	e := nuevoEntorno(t)
	resp, cid := e.conectar(t)

	lista, err := e.svc.ListarConexiones(ctx, claimsDuenio())
	if err != nil || len(lista) != 1 || lista[0].ClientName != "Claude" {
		t.Fatalf("lista: %+v %v", lista, err)
	}
	otro := &auth.Claims{UsuarioID: "u-otro", TenantID: "t-2", Rol: auth.RolDuenio}
	if l, _ := e.svc.ListarConexiones(ctx, otro); len(l) != 0 {
		t.Errorf("otro usuario ve conexiones ajenas: %+v", l)
	}
	if _, err := e.svc.ListarConexiones(ctx, &auth.Claims{UsuarioID: "u-duenio", Rol: auth.RolMozo}); !errors.Is(err, ErrSoloDuenio) {
		t.Errorf("mozo: %v", err)
	}
	if err := e.svc.RevocarConexion(ctx, lista[0].ID, otro); !errors.Is(err, ErrConexionNoEncontrada) {
		t.Errorf("revocar ajena: %v", err)
	}
	if err := e.svc.RevocarConexion(ctx, lista[0].ID, claimsDuenio()); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.ValidarAccessToken(ctx, resp.AccessToken); !errors.Is(err, ErrTokenInvalido) {
		t.Errorf("access tras revocar: %v", err)
	}
	if _, err := e.svc.Refrescar(ctx, resp.RefreshToken, cid); !errors.Is(err, ErrGrantInvalido) {
		t.Errorf("refresh tras revocar: %v", err)
	}
}

func TestRevocarToken(t *testing.T) {
	ctx := context.Background()
	e := nuevoEntorno(t)
	resp, cid := e.conectar(t)
	if err := e.svc.Revocar(ctx, "desconocido"); err != nil {
		t.Errorf("token desconocido debe ser nil: %v", err)
	}
	if err := e.svc.Revocar(ctx, resp.RefreshToken); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.Refrescar(ctx, resp.RefreshToken, cid); !errors.Is(err, ErrGrantInvalido) {
		t.Errorf("refresh revocado: %v", err)
	}
}
