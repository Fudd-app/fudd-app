package oauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
)

func nuevoHandlers(t *testing.T) (*entorno, *Handlers) {
	t.Helper()
	e := nuevoEntorno(t)
	return e, NuevosHandlers(e.svc)
}

func decodificar(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatalf("body no es JSON: %q", rec.Body.String())
	}
	return m
}

func peticionForm(metodo, ruta string, form url.Values) *http.Request {
	r := httptest.NewRequest(metodo, ruta, strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return r
}

func TestHandlers_Metadata(t *testing.T) {
	_, h := nuevoHandlers(t)

	rec := httptest.NewRecorder()
	h.MetadataRecurso(rec, httptest.NewRequest("GET", "/.well-known/oauth-protected-resource", nil))
	m := decodificar(t, rec)
	if rec.Code != 200 || m["resource"] != "https://mcp.test/mcp" || m["authorization_servers"].([]any)[0] != "https://mcp.test" {
		t.Errorf("recurso: %d %v", rec.Code, m)
	}

	rec = httptest.NewRecorder()
	h.MetadataServidor(rec, httptest.NewRequest("GET", "/.well-known/oauth-authorization-server", nil))
	m = decodificar(t, rec)
	if m["issuer"] != "https://mcp.test" || m["token_endpoint"] != "https://mcp.test/oauth/token" ||
		m["registration_endpoint"] != "https://mcp.test/oauth/register" ||
		m["code_challenge_methods_supported"].([]any)[0] != "S256" ||
		m["authorization_response_iss_parameter_supported"] != true {
		t.Errorf("servidor: %v", m)
	}
}

func TestHandlers_Registrar(t *testing.T) {
	_, h := nuevoHandlers(t)
	post := func(body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		h.Registrar(rec, httptest.NewRequest("POST", "/oauth/register", strings.NewReader(body)))
		return rec
	}

	rec := post(`{"client_name":"Claude","redirect_uris":["https://claude.ai/cb"],"extra":1}`)
	m := decodificar(t, rec)
	if rec.Code != 201 || !strings.HasPrefix(m["client_id"].(string), "mcp_") || m["token_endpoint_auth_method"] != "none" {
		t.Errorf("ok: %d %v", rec.Code, m)
	}

	rec = post(`{"redirect_uris":["http://ejemplo.com/cb"]}`)
	if rec.Code != 400 || decodificar(t, rec)["error"] != "invalid_redirect_uri" {
		t.Errorf("redirect inválido: %d %s", rec.Code, rec.Body)
	}
	rec = post(`{"redirect_uris":["https://a.com/cb"],"token_endpoint_auth_method":"client_secret_basic"}`)
	if rec.Code != 400 || decodificar(t, rec)["error"] != "invalid_client_metadata" {
		t.Errorf("auth method: %d %s", rec.Code, rec.Body)
	}
	rec = post(`{"client_name":"` + strings.Repeat("a", maxBodyBytes) + `","redirect_uris":["https://a.com/cb"]}`)
	if rec.Code != 400 || decodificar(t, rec)["error"] != "invalid_client_metadata" {
		t.Errorf("body excedido: %d %s", rec.Code, rec.Body)
	}
}

func TestHandlers_Registrar_RateLimit(t *testing.T) {
	_, h := nuevoHandlers(t)
	var ultimo *httptest.ResponseRecorder
	for i := 0; i < 11; i++ {
		ultimo = httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/oauth/register", strings.NewReader(`{"redirect_uris":["https://a.com/cb"]}`))
		r.RemoteAddr = "1.2.3.4:5000"
		h.Registrar(ultimo, r)
		if i < 10 && ultimo.Code != 201 {
			t.Fatalf("solicitud %d: %d", i+1, ultimo.Code)
		}
	}
	if ultimo.Code != 429 || ultimo.Header().Get("Retry-After") != "60" || decodificar(t, ultimo)["error"] != "rate_limited" {
		t.Errorf("esperaba 429: %d %s", ultimo.Code, ultimo.Body)
	}
}

func TestHandlers_Autorizar(t *testing.T) {
	e, h := nuevoHandlers(t)
	c := e.registrar(t)
	p := e.paramsOK(c.ClientID)
	consulta := func(mod func(url.Values)) *httptest.ResponseRecorder {
		q := url.Values{
			"client_id": {p.ClientID}, "redirect_uri": {p.RedirectURI}, "response_type": {"code"},
			"code_challenge": {p.CodeChallenge}, "code_challenge_method": {"S256"}, "state": {"xyz"},
			"scope": {ScopeLectura}, "resource": {p.Resource},
		}
		if mod != nil {
			mod(q)
		}
		rec := httptest.NewRecorder()
		h.Autorizar(rec, httptest.NewRequest("GET", "/oauth/authorize?"+q.Encode(), nil))
		return rec
	}

	rec := consulta(func(q url.Values) { q.Set("client_id", "nope") })
	if rec.Code != 400 || !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/html") {
		t.Errorf("cliente inválido: %d %s", rec.Code, rec.Header().Get("Content-Type"))
	}
	rec = consulta(func(q url.Values) { q.Set("redirect_uri", "https://otro.com/cb") })
	if rec.Code != 400 {
		t.Errorf("redirect inválido: %d", rec.Code)
	}
	rec = consulta(func(q url.Values) { q.Set("code_challenge_method", "plain") })
	if loc := rec.Header().Get("Location"); rec.Code != 302 || !strings.Contains(loc, "error=invalid_request") || !strings.Contains(loc, "state=xyz") {
		t.Errorf("invalid_request: %d %s", rec.Code, loc)
	}
	rec = consulta(func(q url.Values) { q.Set("scope", "admin") })
	if loc := rec.Header().Get("Location"); rec.Code != 302 || !strings.Contains(loc, "error=invalid_scope") {
		t.Errorf("invalid_scope: %d %s", rec.Code, loc)
	}
	rec = consulta(nil)
	loc := rec.Header().Get("Location")
	if rec.Code != 302 || !strings.HasPrefix(loc, "https://app.test/conectar?solicitud=") {
		t.Fatalf("ok: %d %s", rec.Code, loc)
	}
	id := strings.TrimPrefix(loc, "https://app.test/conectar?solicitud=")
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("esperaba 1 cookie, hay %d", len(cookies))
	}
	ck := cookies[0]
	if ck.Name != "mcp_sol_"+id || ck.Value == "" || ck.Path != "/" || ck.Domain != "" || !ck.HttpOnly ||
		ck.SameSite != http.SameSiteLaxMode || ck.MaxAge != 600 || ck.Secure {
		t.Errorf("cookie: %+v", ck)
	}

	r := httptest.NewRequest("GET", "/oauth/authorize?"+url.Values{
		"client_id": {p.ClientID}, "redirect_uri": {p.RedirectURI}, "response_type": {"code"},
		"code_challenge": {p.CodeChallenge}, "code_challenge_method": {"S256"}, "resource": {p.Resource},
	}.Encode(), nil)
	r.Header.Set("X-Forwarded-Proto", "https")
	rec = httptest.NewRecorder()
	h.Autorizar(rec, r)
	if c := rec.Result().Cookies(); len(c) != 1 || !c[0].Secure {
		t.Errorf("detrás de proxy https la cookie debe ser Secure: %+v", c)
	}
}

func TestHandlers_Autorizar_CookieDominio(t *testing.T) {
	autorizar := func(t *testing.T, dominio string) *http.Cookie {
		t.Helper()
		e, h := nuevoHandlers(t)
		h.svc.cfg.DominioCookie = dominio
		c := e.registrar(t)
		p := e.paramsOK(c.ClientID)
		q := url.Values{
			"client_id": {p.ClientID}, "redirect_uri": {p.RedirectURI}, "response_type": {"code"},
			"code_challenge": {p.CodeChallenge}, "code_challenge_method": {"S256"},
			"scope": {ScopeLectura}, "resource": {p.Resource},
		}
		rec := httptest.NewRecorder()
		h.Autorizar(rec, httptest.NewRequest("GET", "/oauth/authorize?"+q.Encode(), nil))
		cookies := rec.Result().Cookies()
		if len(cookies) != 1 {
			t.Fatalf("esperaba 1 cookie, hay %d", len(cookies))
		}
		return cookies[0]
	}

	if ck := autorizar(t, "fudd.com.ar"); ck.Domain != "fudd.com.ar" || ck.Path != "/" {
		t.Errorf("con dominio: %+v", ck)
	}
	if ck := autorizar(t, ""); ck.Domain != "" || ck.Path != "/" {
		t.Errorf("sin dominio: %+v", ck)
	}
	t.Setenv("APP_ENV", "Production")
	if ck := autorizar(t, ""); !ck.Secure {
		t.Errorf("con APP_ENV=production la cookie debe ser Secure: %+v", ck)
	}
}

func TestConfigDesdeEntorno_DominioCookie(t *testing.T) {
	t.Setenv("JWT_SECRET", "s")
	t.Setenv("MCP_COOKIE_DOMAIN", " .fudd.com.ar ")
	cfg, err := ConfigDesdeEntorno()
	if err != nil || cfg.DominioCookie != "fudd.com.ar" {
		t.Errorf("dominio: %q %v", cfg.DominioCookie, err)
	}
}

func TestHandlers_Autorizar_RateLimit(t *testing.T) {
	_, h := nuevoHandlers(t)
	var ultimo *httptest.ResponseRecorder
	for i := 0; i < 31; i++ {
		ultimo = httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/oauth/authorize?client_id=nope", nil)
		r.RemoteAddr = "1.2.3.4:5000"
		h.Autorizar(ultimo, r)
		if i < 30 && ultimo.Code != 400 {
			t.Fatalf("solicitud %d: %d", i+1, ultimo.Code)
		}
	}
	if ultimo.Code != 429 || ultimo.Header().Get("Location") != "" {
		t.Errorf("esperaba 429 sin redirección: %d", ultimo.Code)
	}
}

func TestHandlers_Token(t *testing.T) {
	e, h := nuevoHandlers(t)
	c := e.registrar(t)
	post := func(f url.Values) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		h.Token(rec, peticionForm("POST", "/oauth/token", f))
		if rec.Header().Get("Cache-Control") != "no-store" {
			t.Error("falta Cache-Control: no-store")
		}
		return rec
	}

	if rec := post(url.Values{"grant_type": {"authorization_code"}}); rec.Code != 400 || decodificar(t, rec)["error"] != "invalid_request" {
		t.Errorf("faltan params: %d %s", rec.Code, rec.Body)
	}
	r := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(`{"grant_type":"password"}`))
	r.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Token(rec, r)
	if rec.Code != 400 || decodificar(t, rec)["error"] != "invalid_request" {
		t.Errorf("content-type: %d %s", rec.Code, rec.Body)
	}
	r = httptest.NewRequest("POST", "/oauth/token", strings.NewReader("grant_type=password"))
	r.Header.Set("Content-Type", "Application/X-WWW-Form-Urlencoded; charset=utf-8")
	rec = httptest.NewRecorder()
	h.Token(rec, r)
	if rec.Code != 400 || decodificar(t, rec)["error"] != "unsupported_grant_type" {
		t.Errorf("content-type con parámetros: %d %s", rec.Code, rec.Body)
	}
	if rec := post(url.Values{"grant_type": {"authorization_code"}, "code": {strings.Repeat("a", maxBodyBytes)}}); rec.Code != 400 || decodificar(t, rec)["error"] != "invalid_request" {
		t.Errorf("body excedido: %d %s", rec.Code, rec.Body)
	}
	if rec := post(url.Values{"grant_type": {"password"}}); rec.Code != 400 || decodificar(t, rec)["error"] != "unsupported_grant_type" {
		t.Errorf("grant: %d %s", rec.Code, rec.Body)
	}
	malo := url.Values{"grant_type": {"authorization_code"}, "code": {"x"}, "code_verifier": {verifierTest}, "client_id": {c.ClientID}, "redirect_uri": {redirectTest}}
	if rec := post(malo); rec.Code != 400 || decodificar(t, rec)["error"] != "invalid_grant" {
		t.Errorf("invalid_grant: %d %s", rec.Code, rec.Body)
	}

	code := e.aprobada(t, c.ClientID)
	ok := url.Values{"grant_type": {"authorization_code"}, "code": {code}, "code_verifier": {verifierTest}, "client_id": {c.ClientID}, "redirect_uri": {redirectTest}}
	rec = post(ok)
	m := decodificar(t, rec)
	if rec.Code != 200 || m["access_token"] == "" || m["refresh_token"] == "" || m["token_type"] != "Bearer" {
		t.Fatalf("ok: %d %s", rec.Code, rec.Body)
	}

	rec = post(url.Values{"grant_type": {"refresh_token"}, "refresh_token": {m["refresh_token"].(string)}, "client_id": {c.ClientID}})
	if rec.Code != 200 {
		t.Errorf("refresh: %d %s", rec.Code, rec.Body)
	}
}

func TestHandlers_Revocar(t *testing.T) {
	_, h := nuevoHandlers(t)
	rec := httptest.NewRecorder()
	h.Revocar(rec, peticionForm("POST", "/oauth/revoke", url.Values{"token": {"cualquiera"}}))
	if rec.Code != 200 {
		t.Errorf("revocar: %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.Revocar(rec, peticionForm("POST", "/oauth/revoke", url.Values{}))
	if rec.Code != 400 || decodificar(t, rec)["error"] != "invalid_request" {
		t.Errorf("sin token: %d %s", rec.Code, rec.Body)
	}
}

func TestHandlers_AprobarRolMozo(t *testing.T) {
	e, h := nuevoHandlers(t)
	c := e.registrar(t)
	id, _, _ := e.svc.IniciarAutorizacion(context.Background(), e.paramsOK(c.ClientID))

	r := httptest.NewRequest("POST", "/oauth/solicitudes/"+id+"/aprobar", nil)
	r.SetPathValue("id", id)
	r = r.WithContext(auth.ContextConClaims(r.Context(), &auth.Claims{UsuarioID: "u-mozo", TenantID: "t-1", Rol: auth.RolMozo}))
	rec := httptest.NewRecorder()
	h.Aprobar(rec, r)
	if rec.Code != 403 || decodificar(t, rec)["codigo"] != "MCP_SOLO_DUENIO" {
		t.Errorf("mozo: %d %s", rec.Code, rec.Body)
	}

	r = httptest.NewRequest("POST", "/oauth/solicitudes/x/aprobar", nil)
	r.SetPathValue("id", "inexistente")
	r = r.WithContext(auth.ContextConClaims(r.Context(), claimsDuenio()))
	rec = httptest.NewRecorder()
	h.Aprobar(rec, r)
	if rec.Code != 404 || decodificar(t, rec)["codigo"] != "SOLICITUD_INVALIDA" {
		t.Errorf("404: %d %s", rec.Code, rec.Body)
	}

	// Sin sesión en contexto.
	rec = httptest.NewRecorder()
	h.Aprobar(rec, httptest.NewRequest("POST", "/", nil))
	if rec.Code != 401 {
		t.Errorf("sin claims: %d", rec.Code)
	}
}

func TestHandlers_Conexiones(t *testing.T) {
	e, h := nuevoHandlers(t)
	conCtx := func(r *http.Request) *http.Request {
		return r.WithContext(auth.ContextConClaims(r.Context(), claimsDuenio()))
	}

	rec := httptest.NewRecorder()
	h.ListarConexiones(rec, conCtx(httptest.NewRequest("GET", "/oauth/conexiones", nil)))
	if rec.Code != 200 || strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Errorf("lista vacía: %d %s", rec.Code, rec.Body)
	}

	e.conectar(t)
	lista, _ := e.svc.ListarConexiones(context.Background(), claimsDuenio())

	r := conCtx(httptest.NewRequest("DELETE", "/oauth/conexiones/"+lista[0].ID, nil))
	r.SetPathValue("id", lista[0].ID)
	rec = httptest.NewRecorder()
	h.RevocarConexion(rec, r)
	if rec.Code != 204 {
		t.Errorf("revocar: %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.RevocarConexion(rec, r)
	if rec.Code != 404 {
		t.Errorf("revocar de nuevo: %d", rec.Code)
	}
}

func TestHandlers_SolicitudLigadaAlNavegador(t *testing.T) {
	e, h := nuevoHandlers(t)
	c := e.registrar(t)
	id, nonce, _ := e.svc.IniciarAutorizacion(context.Background(), e.paramsOK(c.ClientID))
	otroID, otroNonce, _ := e.svc.IniciarAutorizacion(context.Background(), e.paramsOK(c.ClientID))

	llamar := func(f http.HandlerFunc, cookie *http.Cookie) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/oauth/solicitudes/"+id, nil)
		r.SetPathValue("id", id)
		if cookie != nil {
			r.AddCookie(cookie)
		}
		r = r.WithContext(auth.ContextConClaims(r.Context(), claimsDuenio()))
		rec := httptest.NewRecorder()
		f(rec, r)
		return rec
	}

	for nombre, ck := range map[string]*http.Cookie{
		"sin cookie":     nil,
		"cookie de otra": {Name: "mcp_sol_" + otroID, Value: otroNonce},
		"nonce de otra":  {Name: "mcp_sol_" + id, Value: otroNonce},
	} {
		rec := llamar(h.Aprobar, ck)
		if rec.Code != 403 || decodificar(t, rec)["codigo"] != "SOLICITUD_OTRO_NAVEGADOR" {
			t.Errorf("%s: %d %s", nombre, rec.Code, rec.Body)
		}
	}

	rec := llamar(h.VerSolicitud, nil)
	if m := decodificar(t, rec); rec.Code != 200 || m["mismo_navegador"] != false {
		t.Errorf("ver sin cookie: %d %s", rec.Code, rec.Body)
	}
	ok := &http.Cookie{Name: "mcp_sol_" + id, Value: nonce}
	rec = llamar(h.VerSolicitud, ok)
	if m := decodificar(t, rec); rec.Code != 200 || m["mismo_navegador"] != true {
		t.Errorf("ver con cookie: %d %s", rec.Code, rec.Body)
	}
	rec = llamar(h.Aprobar, ok)
	if rec.Code != 200 || decodificar(t, rec)["redirect_url"] == "" {
		t.Errorf("aprobar con cookie: %d %s", rec.Code, rec.Body)
	}
}
