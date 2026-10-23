package oauth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
)

func TestRequerirTokenMCP(t *testing.T) {
	e := nuevoEntorno(t)
	resp, _ := e.conectar(t)

	var claims *auth.Claims
	protegido := RequerirTokenMCP(e.svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims = auth.ClaimsFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))
	llamar := func(mod func(*http.Request)) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/mcp", nil)
		if mod != nil {
			mod(r)
		}
		rec := httptest.NewRecorder()
		protegido.ServeHTTP(rec, r)
		return rec
	}
	metadata := `Bearer resource_metadata="https://mcp.test/.well-known/oauth-protected-resource/mcp"`

	rec := llamar(nil)
	if rec.Code != 401 || rec.Header().Get("WWW-Authenticate") != metadata {
		t.Errorf("sin header: %d %q", rec.Code, rec.Header().Get("WWW-Authenticate"))
	}

	sesion, _ := auth.GenerarJWT(claimsDuenio(), "secreto-de-test")
	rec = llamar(func(r *http.Request) { r.AddCookie(&http.Cookie{Name: "session", Value: sesion}) })
	if rec.Code != 401 {
		t.Errorf("cookie de sesión sin Bearer: %d", rec.Code)
	}

	rec = llamar(func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+sesion) })
	if rec.Code != 401 || !strings.Contains(rec.Header().Get("WWW-Authenticate"), `error="invalid_token"`) {
		t.Errorf("token de sesión como Bearer: %d %q", rec.Code, rec.Header().Get("WWW-Authenticate"))
	}

	rec = llamar(func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+resp.AccessToken) })
	if rec.Code != 200 || claims == nil || claims.UsuarioID != "u-duenio" || claims.TenantID != "t-1" {
		t.Errorf("token MCP válido: %d %+v", rec.Code, claims)
	}

	rec = llamar(func(r *http.Request) { r.Header.Set("Authorization", "bearer "+resp.AccessToken) })
	if rec.Code != 200 {
		t.Errorf("esquema bearer en minúsculas: %d", rec.Code)
	}

	e.usuarios.fallar(errors.New("db caída"))
	rec = llamar(func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+resp.AccessToken) })
	if rec.Code != 500 || rec.Header().Get("WWW-Authenticate") != "" || !strings.Contains(rec.Body.String(), `"server_error"`) {
		t.Errorf("error de infraestructura: %d %q %s", rec.Code, rec.Header().Get("WWW-Authenticate"), rec.Body)
	}
}
