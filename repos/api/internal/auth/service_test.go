package auth_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
)

type mockStore struct {
	obtenerUsuarioPorEmailFn           func(ctx context.Context, email string) (*auth.UsuarioAuth, error)
	obtenerUsuarioPorGoogleSubFn       func(ctx context.Context, googleSub string) (*auth.UsuarioAuth, error)
	obtenerUsuarioPorIDFn              func(ctx context.Context, id string) (*auth.UsuarioAuth, error)
	obtenerNegocioPorSlugFn            func(ctx context.Context, slug string) (*auth.NegocioPublicoAuth, error)
	obtenerUsuarioPorTenantYUsernameFn func(ctx context.Context, tenantID, username string) (*auth.UsuarioAuth, error)
	vincularGoogleSubFn                func(ctx context.Context, usuarioID, googleSub string) error
	guardarTokenFn                     func(ctx context.Context, usuarioID, token string, expiresAt time.Time) (string, error)
	obtenerTokenFn                     func(ctx context.Context, token string) (*auth.MagicToken, error)
	marcarTokenUsadoFn                 func(ctx context.Context, tokenID string) error
	obtenerStaffPorGoogleSubFn         func(ctx context.Context, slug, googleSub string) (*auth.UsuarioAuth, error)
	obtenerStaffPorEmailFn             func(ctx context.Context, slug, email string) (*auth.UsuarioAuth, error)
}

func (m *mockStore) ObtenerStaffPorGoogleSub(ctx context.Context, slug, googleSub string) (*auth.UsuarioAuth, error) {
	if m.obtenerStaffPorGoogleSubFn == nil {
		return nil, auth.ErrUsuarioNoEncontrado
	}
	return m.obtenerStaffPorGoogleSubFn(ctx, slug, googleSub)
}
func (m *mockStore) ObtenerStaffPorEmail(ctx context.Context, slug, email string) (*auth.UsuarioAuth, error) {
	if m.obtenerStaffPorEmailFn == nil {
		return nil, auth.ErrUsuarioNoEncontrado
	}
	return m.obtenerStaffPorEmailFn(ctx, slug, email)
}
func (m *mockStore) ObtenerNegocioPorSlug(ctx context.Context, slug string) (*auth.NegocioPublicoAuth, error) {
	if m.obtenerNegocioPorSlugFn != nil {
		return m.obtenerNegocioPorSlugFn(ctx, slug)
	}
	return nil, auth.ErrNegocioNoEncontrado
}
func (m *mockStore) ObtenerUsuarioPorTenantYUsername(ctx context.Context, tenantID, username string) (*auth.UsuarioAuth, error) {
	if m.obtenerUsuarioPorTenantYUsernameFn != nil {
		return m.obtenerUsuarioPorTenantYUsernameFn(ctx, tenantID, username)
	}
	return nil, auth.ErrUsuarioNoEncontrado
}

func (m *mockStore) ObtenerUsuarioPorEmail(ctx context.Context, email string) (*auth.UsuarioAuth, error) {
	return m.obtenerUsuarioPorEmailFn(ctx, email)
}
func (m *mockStore) ObtenerUsuarioPorGoogleSub(ctx context.Context, googleSub string) (*auth.UsuarioAuth, error) {
	return m.obtenerUsuarioPorGoogleSubFn(ctx, googleSub)
}
func (m *mockStore) ObtenerUsuarioPorID(ctx context.Context, id string) (*auth.UsuarioAuth, error) {
	return m.obtenerUsuarioPorIDFn(ctx, id)
}
func (m *mockStore) VincularGoogleSub(ctx context.Context, usuarioID, googleSub string) error {
	return m.vincularGoogleSubFn(ctx, usuarioID, googleSub)
}
func (m *mockStore) GuardarToken(ctx context.Context, usuarioID, token string, expiresAt time.Time) (string, error) {
	return m.guardarTokenFn(ctx, usuarioID, token, expiresAt)
}
func (m *mockStore) ObtenerToken(ctx context.Context, token string) (*auth.MagicToken, error) {
	return m.obtenerTokenFn(ctx, token)
}
func (m *mockStore) MarcarTokenUsado(ctx context.Context, tokenID string) error {
	return m.marcarTokenUsadoFn(ctx, tokenID)
}

type mockEmailSender struct {
	enviarMagicLinkFn func(ctx context.Context, email, link string) error
}

type mockGoogleVerifier struct {
	verificarFn func(ctx context.Context, credencial, audiencia string) (*auth.IdentidadGoogle, error)
}

func (m *mockGoogleVerifier) Verificar(ctx context.Context, credencial, audiencia string) (*auth.IdentidadGoogle, error) {
	return m.verificarFn(ctx, credencial, audiencia)
}

func (m *mockEmailSender) EnviarMagicLink(ctx context.Context, email, link string) error {
	return m.enviarMagicLinkFn(ctx, email, link)
}

func TestGenerarYValidarJWT(t *testing.T) {
	secreto := "secreto-de-prueba"
	claims := &auth.Claims{
		UsuarioID: "user-123",
		TenantID:  "tenant-456",
		Rol:       "admin",
	}

	token, err := auth.GenerarJWT(claims, secreto)
	if err != nil {
		t.Fatalf("GenerarJWT error: %v", err)
	}
	if token == "" {
		t.Fatal("token vacío")
	}

	recuperadas, err := auth.ValidarJWT(token, secreto)
	if err != nil {
		t.Fatalf("ValidarJWT error: %v", err)
	}
	if recuperadas.UsuarioID != claims.UsuarioID {
		t.Errorf("UsuarioID: got %q, want %q", recuperadas.UsuarioID, claims.UsuarioID)
	}
	if recuperadas.TenantID != claims.TenantID {
		t.Errorf("TenantID: got %q, want %q", recuperadas.TenantID, claims.TenantID)
	}
	if recuperadas.Rol != claims.Rol {
		t.Errorf("Rol: got %q, want %q", recuperadas.Rol, claims.Rol)
	}
}

func TestValidarJWT_TokenInvalido(t *testing.T) {
	_, err := auth.ValidarJWT("token.invalido.firma", "secreto")
	if err == nil {
		t.Fatal("esperaba error con token inválido")
	}
}

func TestValidarJWT_ClaimsIncompletos(t *testing.T) {
	secreto := "test-secreto"
	tok, _ := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, jwtlib.MapClaims{
		"exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte(secreto))
	_, err := auth.ValidarJWT(tok, secreto)
	if err == nil {
		t.Fatal("esperaba error por claims incompletos")
	}
}

func TestSolicitarLink_Exitoso(t *testing.T) {
	var emailEnviado string
	var linkEnviado string

	store := &mockStore{
		obtenerUsuarioPorEmailFn: func(ctx context.Context, email string) (*auth.UsuarioAuth, error) {
			return &auth.UsuarioAuth{ID: "usr-1", TenantID: "ten-1", Email: email, Rol: "admin"}, nil
		},
		guardarTokenFn: func(ctx context.Context, usuarioID, token string, expiresAt time.Time) (string, error) {
			return "tok-id-1", nil
		},
	}
	emailMock := &mockEmailSender{
		enviarMagicLinkFn: func(ctx context.Context, email, link string) error {
			emailEnviado = email
			linkEnviado = link
			return nil
		},
	}

	svc := auth.NuevoService(store, emailMock)
	link, err := svc.SolicitarLink(context.Background(), "admin@mibar.com")
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if link == "" {
		t.Error("se esperaba que SolicitarLink devolviera el link generado")
	}

	if emailEnviado != "admin@mibar.com" {
		t.Errorf("email enviado incorrecto: %q", emailEnviado)
	}
	if linkEnviado == "" {
		t.Error("se esperaba un link de acceso generado")
	}
}

func TestSolicitarLink_UsuarioNoEncontrado(t *testing.T) {
	store := &mockStore{
		obtenerUsuarioPorEmailFn: func(ctx context.Context, email string) (*auth.UsuarioAuth, error) {
			return nil, auth.ErrUsuarioNoEncontrado
		},
	}
	emailMock := &mockEmailSender{}

	svc := auth.NuevoService(store, emailMock)
	link, err := svc.SolicitarLink(context.Background(), "no-existe@mibar.com")
	if !errors.Is(err, auth.ErrUsuarioNoEncontrado) {
		t.Fatalf("se esperaba ErrUsuarioNoEncontrado, obtenido: %v", err)
	}
	if link != "" {
		t.Errorf("no se debe generar link para un email inexistente, obtenido: %q", link)
	}
}

func TestSolicitarLink_HandlerInformaUsuarioNoEncontrado(t *testing.T) {
	store := &mockStore{
		obtenerUsuarioPorEmailFn: func(ctx context.Context, email string) (*auth.UsuarioAuth, error) {
			return nil, auth.ErrUsuarioNoEncontrado
		},
	}
	handler := auth.NuevosHandlers(auth.NuevoService(store, &mockEmailSender{}), false)
	req := httptest.NewRequest(
		http.MethodPost,
		"/auth/magic-link",
		strings.NewReader(`{"email":"no-existe@mibar.com"}`),
	)
	recorder := httptest.NewRecorder()

	handler.SolicitarLink(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want %d", recorder.Code, http.StatusNotFound)
	}
	if !strings.Contains(recorder.Body.String(), "No encontramos una cuenta registrada") {
		t.Fatalf("respuesta inesperada: %s", recorder.Body.String())
	}
}

func TestSolicitarLink_NormalizaEmail(t *testing.T) {
	var emailConsultado string

	store := &mockStore{
		obtenerUsuarioPorEmailFn: func(ctx context.Context, email string) (*auth.UsuarioAuth, error) {
			emailConsultado = email
			return &auth.UsuarioAuth{ID: "usr-1", TenantID: "ten-1", Email: email, Rol: "admin"}, nil
		},
		guardarTokenFn: func(ctx context.Context, usuarioID, token string, expiresAt time.Time) (string, error) {
			return "tok-id-1", nil
		},
	}
	emailMock := &mockEmailSender{
		enviarMagicLinkFn: func(ctx context.Context, email, link string) error { return nil },
	}

	svc := auth.NuevoService(store, emailMock)
	if _, err := svc.SolicitarLink(context.Background(), "  Admin@MiBar.COM "); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	if emailConsultado != "admin@mibar.com" {
		t.Errorf("email no normalizado: got %q, want %q", emailConsultado, "admin@mibar.com")
	}
}

func TestVerificarToken_Exitoso(t *testing.T) {
	store := &mockStore{
		obtenerTokenFn: func(ctx context.Context, token string) (*auth.MagicToken, error) {
			return &auth.MagicToken{
				ID:        "tok-1",
				UsuarioID: "usr-1",
				Token:     token,
				ExpiresAt: time.Now().Add(10 * time.Minute),
				UsedAt:    nil,
			}, nil
		},
		marcarTokenUsadoFn: func(ctx context.Context, tokenID string) error {
			return nil
		},
		obtenerUsuarioPorIDFn: func(ctx context.Context, id string) (*auth.UsuarioAuth, error) {
			return &auth.UsuarioAuth{ID: id, TenantID: "ten-1", Email: "admin@mibar.com", Rol: "admin"}, nil
		},
	}

	svc := auth.NuevoService(store, &mockEmailSender{})
	usr, err := svc.VerificarToken(context.Background(), "algun-token")
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	if usr.ID != "usr-1" {
		t.Errorf("usuario retornado incorrecto: got %q, want %q", usr.ID, "usr-1")
	}
}

func TestVerificarToken_Errores(t *testing.T) {
	// 1. Token inexistente
	store1 := &mockStore{
		obtenerTokenFn: func(ctx context.Context, token string) (*auth.MagicToken, error) {
			return nil, errors.New("no encontrado")
		},
	}
	svc1 := auth.NuevoService(store1, &mockEmailSender{})
	_, err := svc1.VerificarToken(context.Background(), "invalido")
	if err == nil || err.Error() != "token inválido" {
		t.Errorf("se esperaba error de token inválido, obtenido: %v", err)
	}

	// 2. Token ya usado
	ahora := time.Now()
	store2 := &mockStore{
		obtenerTokenFn: func(ctx context.Context, token string) (*auth.MagicToken, error) {
			return &auth.MagicToken{
				ID:        "tok-1",
				UsuarioID: "usr-1",
				Token:     token,
				ExpiresAt: time.Now().Add(10 * time.Minute),
				UsedAt:    &ahora,
			}, nil
		},
	}
	svc2 := auth.NuevoService(store2, &mockEmailSender{})
	_, err = svc2.VerificarToken(context.Background(), "ya-usado")
	if err == nil || err.Error() != "token ya usado" {
		t.Errorf("se esperaba error de token ya usado, obtenido: %v", err)
	}

	// 3. Token expirado
	store3 := &mockStore{
		obtenerTokenFn: func(ctx context.Context, token string) (*auth.MagicToken, error) {
			return &auth.MagicToken{
				ID:        "tok-1",
				UsuarioID: "usr-1",
				Token:     token,
				ExpiresAt: time.Now().Add(-5 * time.Minute),
				UsedAt:    nil,
			}, nil
		},
	}
	svc3 := auth.NuevoService(store3, &mockEmailSender{})
	_, err = svc3.VerificarToken(context.Background(), "expirado")
	if err == nil || err.Error() != "token expirado" {
		t.Errorf("se esperaba error de token expirado, obtenido: %v", err)
	}
}

func TestAutenticarGoogle_VinculaUsuarioExistente(t *testing.T) {
	var emailConsultado string
	var usuarioVinculado string
	var subjectVinculado string
	store := &mockStore{
		obtenerUsuarioPorGoogleSubFn: func(ctx context.Context, googleSub string) (*auth.UsuarioAuth, error) {
			return nil, auth.ErrUsuarioNoEncontrado
		},
		obtenerUsuarioPorEmailFn: func(ctx context.Context, email string) (*auth.UsuarioAuth, error) {
			emailConsultado = email
			return &auth.UsuarioAuth{ID: "usr-1", TenantID: "ten-1", Email: email, Rol: "admin"}, nil
		},
		vincularGoogleSubFn: func(ctx context.Context, usuarioID, googleSub string) error {
			usuarioVinculado = usuarioID
			subjectVinculado = googleSub
			return nil
		},
	}
	verifier := &mockGoogleVerifier{verificarFn: func(ctx context.Context, credencial, audiencia string) (*auth.IdentidadGoogle, error) {
		if audiencia != "client-id.apps.googleusercontent.com" {
			t.Fatalf("audiencia inesperada: %q", audiencia)
		}
		return &auth.IdentidadGoogle{
			Subject:         "google-sub-1",
			Email:           "Admin@MiBar.com",
			EmailVerificado: true,
		}, nil
	}}

	svc := auth.NuevoServiceConGoogle(store, &mockEmailSender{}, "client-id.apps.googleusercontent.com", verifier)
	usuario, err := svc.AutenticarGoogle(context.Background(), "google-id-token")
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if usuario.ID != "usr-1" {
		t.Fatalf("usuario inesperado: %q", usuario.ID)
	}
	if emailConsultado != "admin@mibar.com" {
		t.Errorf("email no normalizado: %q", emailConsultado)
	}
	if usuarioVinculado != "usr-1" || subjectVinculado != "google-sub-1" {
		t.Errorf("vinculación inesperada: usuario=%q sub=%q", usuarioVinculado, subjectVinculado)
	}
}

func TestAutenticarGoogle_UsaSubjectYaVinculado(t *testing.T) {
	googleSub := "google-sub-1"
	store := &mockStore{
		obtenerUsuarioPorGoogleSubFn: func(ctx context.Context, subject string) (*auth.UsuarioAuth, error) {
			return &auth.UsuarioAuth{
				ID: "usr-1", TenantID: "ten-1", Email: "admin@mibar.com", Rol: "admin", GoogleSub: &googleSub,
			}, nil
		},
	}
	verifier := &mockGoogleVerifier{verificarFn: func(ctx context.Context, credencial, audiencia string) (*auth.IdentidadGoogle, error) {
		return &auth.IdentidadGoogle{Subject: googleSub, Email: "admin@mibar.com", EmailVerificado: true}, nil
	}}

	svc := auth.NuevoServiceConGoogle(store, &mockEmailSender{}, "client-id", verifier)
	usuario, err := svc.AutenticarGoogle(context.Background(), "token")
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if usuario.GoogleSub == nil || *usuario.GoogleSub != googleSub {
		t.Fatalf("identidad Google inesperada: %#v", usuario.GoogleSub)
	}
}

func TestAutenticarGoogle_RechazaEmailNoVerificado(t *testing.T) {
	verifier := &mockGoogleVerifier{verificarFn: func(ctx context.Context, credencial, audiencia string) (*auth.IdentidadGoogle, error) {
		return &auth.IdentidadGoogle{Subject: "google-sub-1", Email: "admin@mibar.com", EmailVerificado: false}, nil
	}}
	svc := auth.NuevoServiceConGoogle(&mockStore{}, &mockEmailSender{}, "client-id", verifier)

	_, err := svc.AutenticarGoogle(context.Background(), "token")
	if !errors.Is(err, auth.ErrEmailGoogleNoVerificado) {
		t.Fatalf("se esperaba ErrEmailGoogleNoVerificado, obtenido: %v", err)
	}
}

func TestAutenticarGoogle_UsuarioNoRegistrado(t *testing.T) {
	store := &mockStore{
		obtenerUsuarioPorGoogleSubFn: func(ctx context.Context, googleSub string) (*auth.UsuarioAuth, error) {
			return nil, auth.ErrUsuarioNoEncontrado
		},
		obtenerUsuarioPorEmailFn: func(ctx context.Context, email string) (*auth.UsuarioAuth, error) {
			return nil, auth.ErrUsuarioNoEncontrado
		},
	}
	verifier := &mockGoogleVerifier{verificarFn: func(ctx context.Context, credencial, audiencia string) (*auth.IdentidadGoogle, error) {
		return &auth.IdentidadGoogle{
			Subject:         "google-sub-no-registrado",
			Email:           "no-existe@mibar.com",
			EmailVerificado: true,
		}, nil
	}}
	svc := auth.NuevoServiceConGoogle(store, &mockEmailSender{}, "client-id", verifier)

	_, err := svc.AutenticarGoogle(context.Background(), "token")
	if !errors.Is(err, auth.ErrUsuarioNoEncontrado) {
		t.Fatalf("se esperaba ErrUsuarioNoEncontrado, obtenido: %v", err)
	}
}

func TestAutenticarGoogle_RechazaCredencialInvalida(t *testing.T) {
	verifier := &mockGoogleVerifier{verificarFn: func(ctx context.Context, credencial, audiencia string) (*auth.IdentidadGoogle, error) {
		return nil, auth.ErrCredencialGoogleInvalida
	}}
	svc := auth.NuevoServiceConGoogle(&mockStore{}, &mockEmailSender{}, "client-id", verifier)

	_, err := svc.AutenticarGoogle(context.Background(), "token-invalido")
	if !errors.Is(err, auth.ErrCredencialGoogleInvalida) {
		t.Fatalf("se esperaba ErrCredencialGoogleInvalida, obtenido: %v", err)
	}
}

func TestAutenticarPassword_Exitoso(t *testing.T) {
	hash, err := auth.HashPassword("MesaClick2026")
	if err != nil {
		t.Fatalf("no se pudo preparar el hash: %v", err)
	}
	store := &mockStore{
		obtenerUsuarioPorEmailFn: func(ctx context.Context, email string) (*auth.UsuarioAuth, error) {
			if email != "admin@mibar.com" {
				t.Fatalf("email no normalizado: %q", email)
			}
			return &auth.UsuarioAuth{
				ID: "usr-1", TenantID: "ten-1", Email: email, Rol: "admin", PasswordHash: &hash,
			}, nil
		},
	}

	usuario, err := auth.NuevoService(store, &mockEmailSender{}).
		AutenticarPassword(context.Background(), " Admin@MiBar.com ", "MesaClick2026")
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if usuario.ID != "usr-1" {
		t.Fatalf("usuario inesperado: %q", usuario.ID)
	}
}

func TestAutenticarPassword_OcultaCorreoInexistenteYPasswordIncorrecta(t *testing.T) {
	hash, err := auth.HashPassword("MesaClick2026")
	if err != nil {
		t.Fatalf("no se pudo preparar el hash: %v", err)
	}
	casos := []struct {
		nombre string
		store  *mockStore
	}{
		{
			nombre: "correo inexistente",
			store: &mockStore{obtenerUsuarioPorEmailFn: func(context.Context, string) (*auth.UsuarioAuth, error) {
				return nil, auth.ErrUsuarioNoEncontrado
			}},
		},
		{
			nombre: "password incorrecta",
			store: &mockStore{obtenerUsuarioPorEmailFn: func(context.Context, string) (*auth.UsuarioAuth, error) {
				return &auth.UsuarioAuth{ID: "usr-1", PasswordHash: &hash}, nil
			}},
		},
		{
			nombre: "cuenta sin password",
			store: &mockStore{obtenerUsuarioPorEmailFn: func(context.Context, string) (*auth.UsuarioAuth, error) {
				return &auth.UsuarioAuth{ID: "usr-1"}, nil
			}},
		},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			_, err := auth.NuevoService(caso.store, &mockEmailSender{}).
				AutenticarPassword(context.Background(), "admin@mibar.com", "incorrecta-2026")
			if !errors.Is(err, auth.ErrCredencialesInvalidas) {
				t.Fatalf("se esperaba el mismo error genérico, obtenido: %v", err)
			}
		})
	}
}

func TestAutenticarPassword_HandlerDevuelveSesion(t *testing.T) {
	t.Setenv("JWT_SECRET", "secreto-de-prueba")
	hash, err := auth.HashPassword("MesaClick2026")
	if err != nil {
		t.Fatalf("no se pudo preparar el hash: %v", err)
	}
	store := &mockStore{obtenerUsuarioPorEmailFn: func(context.Context, string) (*auth.UsuarioAuth, error) {
		return &auth.UsuarioAuth{
			ID: "usr-1", TenantID: "ten-1", Rol: "admin", PasswordHash: &hash,
		}, nil
	}}
	handler := auth.NuevosHandlers(auth.NuevoService(store, &mockEmailSender{}), false)
	req := httptest.NewRequest(http.MethodPost, "/auth/password", strings.NewReader(
		`{"email":"admin@mibar.com","password":"MesaClick2026"}`,
	))
	req.Host = "mesa-click-api.onrender.com"
	req.Header.Set("Origin", "https://mesa-click-web.onrender.com")
	req.Header.Set("X-Forwarded-Proto", "https")
	recorder := httptest.NewRecorder()

	handler.AutenticarPassword(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if len(recorder.Result().Cookies()) == 0 || recorder.Result().Cookies()[0].Name != "session" || !recorder.Result().Cookies()[0].HttpOnly {
		t.Fatalf("respuesta sin cookie HttpOnly: %s", recorder.Body.String())
	}
	cookie := recorder.Result().Cookies()[0]
	if cookie.SameSite != http.SameSiteNoneMode || !cookie.Secure || !cookie.Partitioned {
		t.Fatalf("cookie cross-site inválida: SameSite=%v Secure=%v Partitioned=%v", cookie.SameSite, cookie.Secure, cookie.Partitioned)
	}
	if strings.Contains(recorder.Body.String(), `"token"`) {
		t.Fatalf("el JWT no debe exponerse al JavaScript: %s", recorder.Body.String())
	}
}

func TestAutenticarStaffGoogle_AislaPorSlug(t *testing.T) {
	var slugConsultado string
	store := &mockStore{obtenerStaffPorGoogleSubFn: func(_ context.Context, slug, sub string) (*auth.UsuarioAuth, error) {
		slugConsultado = slug
		sucursal := "s-1"
		return &auth.UsuarioAuth{ID: "u-1", TenantID: "t-1", Rol: auth.RolMozo, SucursalID: &sucursal, Estado: "activo", SessionVersion: 1}, nil
	}}
	verifier := &mockGoogleVerifier{verificarFn: func(context.Context, string, string) (*auth.IdentidadGoogle, error) {
		return &auth.IdentidadGoogle{Subject: "google-1", Email: "mozo@bar.com", EmailVerificado: true}, nil
	}}
	usuario, err := auth.NuevoServiceConGoogle(store, &mockEmailSender{}, "client-id", verifier).AutenticarStaffGoogle(context.Background(), " Mi-Bar ", "credencial")
	if err != nil {
		t.Fatal(err)
	}
	if slugConsultado != "mi-bar" || usuario.TenantID != "t-1" {
		t.Fatalf("login no aislado por negocio: slug=%q usuario=%+v", slugConsultado, usuario)
	}
}

func TestRequerirSesion_RechazaVersionRevocada(t *testing.T) {
	t.Setenv("JWT_SECRET", "secreto-revocacion")
	store := &mockStore{obtenerUsuarioPorIDFn: func(context.Context, string) (*auth.UsuarioAuth, error) {
		return &auth.UsuarioAuth{ID: "u-1", TenantID: "t-1", Rol: auth.RolMozo, Estado: "activo", SessionVersion: 2}, nil
	}}
	token, err := auth.GenerarJWT(&auth.Claims{UsuarioID: "u-1", TenantID: "t-1", Rol: auth.RolMozo, SessionVersion: 1}, "secreto-revocacion")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/protegido", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: token})
	recorder := httptest.NewRecorder()
	auth.RequerirSesion(store)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("no debía ejecutar el handler") })).ServeHTTP(recorder, req)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestAutenticarStaffPorSlug_ExitosoYRedireccionPorRol(t *testing.T) {
	hash, err := auth.HashPasswordStaff("Panda721")
	if err != nil {
		t.Fatalf("error generando hash staff: %v", err)
	}
	uname := "cocina1"
	sucID := "suc-1"

	store := &mockStore{
		obtenerNegocioPorSlugFn: func(ctx context.Context, slug string) (*auth.NegocioPublicoAuth, error) {
			if slug == "pizzeria-roma" {
				return &auth.NegocioPublicoAuth{ID: "tenant-A", Slug: slug, Nombre: "Pizzería Roma"}, nil
			}
			return nil, auth.ErrNegocioNoEncontrado
		},
		obtenerUsuarioPorTenantYUsernameFn: func(ctx context.Context, tenantID, username string) (*auth.UsuarioAuth, error) {
			if tenantID == "tenant-A" && username == "cocina1" {
				return &auth.UsuarioAuth{
					ID:           "usr-cocina-1",
					TenantID:     "tenant-A",
					SucursalID:   &sucID,
					Username:     &uname,
					Nombre:       "Carlos Cocina",
					Rol:          "cocina",
					Estado:       "activo",
					PasswordHash: &hash,
				}, nil
			}
			return nil, auth.ErrUsuarioNoEncontrado
		},
	}

	svc := auth.NuevoService(store, &mockEmailSender{})
	res, err := svc.AutenticarStaffPorSlug(context.Background(), "Pizzeria-Roma", "@Cocina1", "Panda721")
	if err != nil {
		t.Fatalf("error inesperado en login staff: %v", err)
	}
	if res.RedirectTo != "/kds" {
		t.Errorf("redirect_to esperado /kds para rol cocina, got %q", res.RedirectTo)
	}
	if res.Usuario.TenantID != "tenant-A" {
		t.Errorf("tenant_id inesperado: %q", res.Usuario.TenantID)
	}
}

func TestAutenticarStaffPorSlug_RechazaUsuarioBloqueadoYAislamientoPorTenant(t *testing.T) {
	hash, err := auth.HashPasswordStaff("Panda721")
	if err != nil {
		t.Fatalf("error generando hash staff: %v", err)
	}
	uname := "mozo1"

	store := &mockStore{
		obtenerNegocioPorSlugFn: func(ctx context.Context, slug string) (*auth.NegocioPublicoAuth, error) {
			switch slug {
			case "local-a":
				return &auth.NegocioPublicoAuth{ID: "tenant-A", Slug: "local-a", Nombre: "Local A"}, nil
			case "local-b":
				return &auth.NegocioPublicoAuth{ID: "tenant-B", Slug: "local-b", Nombre: "Local B"}, nil
			default:
				return nil, auth.ErrNegocioNoEncontrado
			}
		},
		obtenerUsuarioPorTenantYUsernameFn: func(ctx context.Context, tenantID, username string) (*auth.UsuarioAuth, error) {
			if tenantID == "tenant-A" && username == "mozo1" {
				return &auth.UsuarioAuth{
					ID:           "usr-mozo-1",
					TenantID:     "tenant-A",
					Username:     &uname,
					Nombre:       "Juan Mozo",
					Rol:          "mozo",
					Estado:       "bloqueado",
					PasswordHash: &hash,
				}, nil
			}
			return nil, auth.ErrUsuarioNoEncontrado
		},
	}

	svc := auth.NuevoService(store, &mockEmailSender{})

	_, err = svc.AutenticarStaffPorSlug(context.Background(), "local-a", "mozo1", "Panda721")
	if !errors.Is(err, auth.ErrUsuarioBloqueado) {
		t.Fatalf("esperaba ErrUsuarioBloqueado, got: %v", err)
	}

	_, err = svc.AutenticarStaffPorSlug(context.Background(), "local-b", "mozo1", "Panda721")
	if !errors.Is(err, auth.ErrCredencialesInvalidas) {
		t.Fatalf("esperaba ErrCredencialesInvalidas al intentar loguearse en otro tenant, got: %v", err)
	}

	_, err = svc.AutenticarStaffPorSlug(context.Background(), "local-a", "mozo1", "ClaveFalsa99")
	if !errors.Is(err, auth.ErrCredencialesInvalidas) {
		t.Fatalf("esperaba ErrCredencialesInvalidas por clave incorrecta, got: %v", err)
	}
}

func TestValidarJWT_RechazaTokenConAudiencia(t *testing.T) {
	secreto := "secreto-de-prueba"
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, jwtlib.MapClaims{
		"usuario_id": "user-123",
		"tenant_id":  "tenant-456",
		"rol":        "admin",
		"aud":        "https://mcp.example.com/mcp",
		"exp":        time.Now().Add(time.Hour).Unix(),
	})
	firmado, err := token.SignedString([]byte(secreto))
	if err != nil {
		t.Fatalf("firmando token: %v", err)
	}

	if _, err := auth.ValidarJWT(firmado, secreto); err == nil {
		t.Fatal("se esperaba error para un token con aud")
	}
}
