package oauth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/db"
)

type pgUsuarios struct{ u *auth.UsuarioAuth }

func (p pgUsuarios) ObtenerUsuarioPorID(_ context.Context, id string) (*auth.UsuarioAuth, error) {
	if id != p.u.ID {
		return nil, errors.New("usuario no encontrado")
	}
	copia := *p.u
	return &copia, nil
}

type pgNegocios struct{}

func (pgNegocios) NombreNegocio(context.Context, string) (string, error) { return "Negocio Test", nil }

// prepararIntegracion conecta a la DB real, migra y crea un tenant con su dueño.
func prepararIntegracion(t *testing.T) (*Service, *auth.UsuarioAuth, *auth.Claims) {
	t.Helper()
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL no configurada, saltando test de integración")
	}
	if err := db.Conectar(); err != nil {
		t.Skipf("no se pudo conectar a la base de datos: %v", err)
	}
	t.Cleanup(db.Cerrar)
	if err := db.EjecutarMigraciones("../../migrations"); err != nil {
		t.Fatalf("no se pudieron correr migraciones: %v", err)
	}

	ctx := context.Background()
	sufijo := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(1_000_000_000)
	usr := &auth.UsuarioAuth{Email: fmt.Sprintf("oauth-%d@test.com", sufijo), Nombre: "Duenio OAuth", Rol: auth.RolDuenio, Estado: "activo"}
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO tenants (nombre, nombre_fantasia, rubro, slug) VALUES ($1, $1, 'bar', $2) RETURNING id`,
		"Test OAuth", fmt.Sprintf("test-oauth-%d", sufijo)).Scan(&usr.TenantID)
	if err != nil {
		t.Fatalf("creando tenant: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Pool.Exec(ctx, "DELETE FROM tenants WHERE id = $1", usr.TenantID) })
	err = db.Pool.QueryRow(ctx,
		`INSERT INTO usuarios (tenant_id, email, nombre, rol) VALUES ($1, $2, $3, 'admin') RETURNING id`,
		usr.TenantID, usr.Email, usr.Nombre).Scan(&usr.ID)
	if err != nil {
		t.Fatalf("creando usuario: %v", err)
	}

	cfg := Config{
		Issuer:   "https://mcp.test",
		Resource: "https://mcp.test/mcp",
		AppURL:   "https://app.test",
		Secreto:  []byte("secreto-fijo-de-test-0123456789ab"),
	}
	svc := NuevoService(NuevoStore(), pgUsuarios{usr}, pgNegocios{}, cfg)
	claims := &auth.Claims{UsuarioID: usr.ID, TenantID: usr.TenantID, Rol: auth.RolDuenio}
	return svc, usr, claims
}

const (
	pgRedirect = "http://localhost:8123/callback"
	pgVerifier = "verifier-de-prueba-con-largo-suficiente-para-pkce-0123456789"
)

func pgChallenge() string {
	h := sha256.Sum256([]byte(pgVerifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// pgAprobada deja una solicitud aprobada y devuelve el code del redirect.
func pgAprobada(t *testing.T, svc *Service, claims *auth.Claims, cli *Cliente) string {
	t.Helper()
	ctx := context.Background()
	id, nonce, err := svc.IniciarAutorizacion(ctx, ParamsAutorizacion{
		ClientID: cli.ClientID, RedirectURI: pgRedirect, ResponseType: "code",
		CodeChallenge: pgChallenge(), CodeChallengeMethod: "S256", State: "st", Resource: svc.cfg.Resource,
	})
	if err != nil {
		t.Fatalf("IniciarAutorizacion: %v", err)
	}
	if _, err := svc.VerSolicitud(ctx, id, nonce, claims); err != nil {
		t.Fatalf("VerSolicitud: %v", err)
	}
	redir, err := svc.Aprobar(ctx, id, nonce, claims)
	if err != nil {
		t.Fatalf("Aprobar: %v", err)
	}
	u, err := url.Parse(redir)
	if err != nil || u.Query().Get("code") == "" {
		t.Fatalf("redirect sin code: %q (%v)", redir, err)
	}
	return u.Query().Get("code")
}

func pgCliente(t *testing.T, svc *Service) *Cliente {
	t.Helper()
	cli, err := svc.RegistrarCliente(context.Background(), "Claude Test", []string{pgRedirect})
	if err != nil {
		t.Fatalf("RegistrarCliente: %v", err)
	}
	return cli
}

func TestIntegration_OAuth_FlujoFeliz(t *testing.T) {
	svc, _, claims := prepararIntegracion(t)
	ctx := context.Background()
	cli := pgCliente(t, svc)
	code := pgAprobada(t, svc, claims, cli)

	tok, err := svc.CanjearCodigo(ctx, code, pgVerifier, cli.ClientID, pgRedirect)
	if err != nil {
		t.Fatalf("CanjearCodigo: %v", err)
	}
	acceso, err := svc.ValidarAccessToken(ctx, tok.AccessToken)
	if err != nil || acceso.Claims.UsuarioID != claims.UsuarioID || acceso.Claims.TenantID != claims.TenantID {
		t.Fatalf("ValidarAccessToken: %+v, %v", acceso, err)
	}
	tok2, err := svc.Refrescar(ctx, tok.RefreshToken, cli.ClientID)
	if err != nil {
		t.Fatalf("Refrescar: %v", err)
	}
	if _, err := svc.ValidarAccessToken(ctx, tok2.AccessToken); err != nil {
		t.Fatalf("access token refrescado inválido: %v", err)
	}
	lista, err := svc.ListarConexiones(ctx, claims)
	if err != nil || len(lista) != 1 || lista[0].ClientName != "Claude Test" {
		t.Fatalf("ListarConexiones: %+v, %v", lista, err)
	}
	if err := svc.RevocarConexion(ctx, lista[0].ID, claims); err != nil {
		t.Fatalf("RevocarConexion: %v", err)
	}
	if _, err := svc.ValidarAccessToken(ctx, tok2.AccessToken); !errors.Is(err, ErrTokenInvalido) {
		t.Fatalf("tras revocar, ValidarAccessToken = %v, quería ErrTokenInvalido", err)
	}
	if _, err := svc.Refrescar(ctx, tok2.RefreshToken, cli.ClientID); !errors.Is(err, ErrGrantInvalido) {
		t.Fatalf("tras revocar, Refrescar = %v, quería ErrGrantInvalido", err)
	}
}

func TestIntegration_OAuth_CodigoReusado(t *testing.T) {
	svc, _, claims := prepararIntegracion(t)
	ctx := context.Background()
	cli := pgCliente(t, svc)
	code := pgAprobada(t, svc, claims, cli)

	if _, err := svc.CanjearCodigo(ctx, code, pgVerifier, cli.ClientID, pgRedirect); err != nil {
		t.Fatalf("primer canje: %v", err)
	}
	if _, err := svc.CanjearCodigo(ctx, code, pgVerifier, cli.ClientID, pgRedirect); !errors.Is(err, ErrGrantInvalido) {
		t.Fatalf("segundo canje = %v, quería ErrGrantInvalido", err)
	}
}

func TestIntegration_OAuth_ReusoDeRefreshRevocaConexion(t *testing.T) {
	svc, _, claims := prepararIntegracion(t)
	ctx := context.Background()
	cli := pgCliente(t, svc)
	code := pgAprobada(t, svc, claims, cli)
	tok, err := svc.CanjearCodigo(ctx, code, pgVerifier, cli.ClientID, pgRedirect)
	if err != nil {
		t.Fatalf("CanjearCodigo: %v", err)
	}

	r1 := tok.RefreshToken
	tok2, err := svc.Refrescar(ctx, r1, cli.ClientID)
	if err != nil {
		t.Fatalf("Refrescar con R1: %v", err)
	}
	if _, err := svc.Refrescar(ctx, r1, cli.ClientID); !errors.Is(err, ErrGrantInvalido) {
		t.Fatalf("reuso de R1 = %v, quería ErrGrantInvalido", err)
	}
	if _, err := svc.Refrescar(ctx, tok2.RefreshToken, cli.ClientID); !errors.Is(err, ErrGrantInvalido) {
		t.Fatalf("R2 tras reuso = %v, quería ErrGrantInvalido (conexión revocada)", err)
	}
}

func TestIntegration_OAuth_IDsNoUUID(t *testing.T) {
	svc, _, claims := prepararIntegracion(t)
	ctx := context.Background()

	if _, err := svc.VerSolicitud(ctx, "abc", "", claims); !errors.Is(err, ErrSolicitudNoEncontrada) {
		t.Errorf("VerSolicitud = %v, quería ErrSolicitudNoEncontrada", err)
	}
	if _, err := svc.Aprobar(ctx, "abc", "", claims); !errors.Is(err, ErrSolicitudNoEncontrada) {
		t.Errorf("Aprobar = %v, quería ErrSolicitudNoEncontrada", err)
	}
	if err := svc.RevocarConexion(ctx, "abc", claims); !errors.Is(err, ErrConexionNoEncontrada) {
		t.Errorf("RevocarConexion = %v, quería ErrConexionNoEncontrada", err)
	}
}

func TestIntegration_OAuth_RevocarConexionDeOtroUsuario(t *testing.T) {
	svc, _, claims := prepararIntegracion(t)
	ctx := context.Background()
	cli := pgCliente(t, svc)
	code := pgAprobada(t, svc, claims, cli)
	if _, err := svc.CanjearCodigo(ctx, code, pgVerifier, cli.ClientID, pgRedirect); err != nil {
		t.Fatalf("CanjearCodigo: %v", err)
	}
	lista, err := svc.ListarConexiones(ctx, claims)
	if err != nil || len(lista) != 1 {
		t.Fatalf("ListarConexiones: %+v, %v", lista, err)
	}

	otro := &auth.Claims{UsuarioID: "00000000-0000-4000-8000-000000000001", TenantID: claims.TenantID, Rol: auth.RolDuenio}
	if err := svc.RevocarConexion(ctx, lista[0].ID, otro); !errors.Is(err, ErrConexionNoEncontrada) {
		t.Fatalf("RevocarConexion de otro usuario = %v, quería ErrConexionNoEncontrada", err)
	}
}

func TestIntegration_OAuth_RevocarTokenInexistente(t *testing.T) {
	svc, _, _ := prepararIntegracion(t)
	if err := svc.Revocar(context.Background(), "token-que-no-existe"); err != nil {
		t.Fatalf("Revocar = %v, quería nil", err)
	}
}

func TestIntegration_OAuth_SolicitudLigadaAlNavegador(t *testing.T) {
	svc, _, claims := prepararIntegracion(t)
	ctx := context.Background()
	cli := pgCliente(t, svc)
	id, nonce, err := svc.IniciarAutorizacion(ctx, ParamsAutorizacion{
		ClientID: cli.ClientID, RedirectURI: pgRedirect, ResponseType: "code",
		CodeChallenge: pgChallenge(), CodeChallengeMethod: "S256", State: "st", Resource: svc.cfg.Resource,
	})
	if err != nil {
		t.Fatalf("IniciarAutorizacion: %v", err)
	}
	if _, err := svc.Aprobar(ctx, id, "", claims); !errors.Is(err, ErrOtroNavegador) {
		t.Fatalf("Aprobar sin nonce = %v, quería ErrOtroNavegador", err)
	}
	if v, err := svc.VerSolicitud(ctx, id, nonce, claims); err != nil || !v.MismoNavegador {
		t.Fatalf("VerSolicitud con nonce: %+v, %v", v, err)
	}
	if _, err := svc.Aprobar(ctx, id, nonce, claims); err != nil {
		t.Fatalf("Aprobar con nonce: %v", err)
	}
}
