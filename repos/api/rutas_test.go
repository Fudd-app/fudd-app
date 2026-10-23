package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEsEntornoDesarrolloLocal(t *testing.T) {
	tests := []struct {
		nombre   string
		entorno  string
		appURL   string
		esperado bool
	}{
		{nombre: "sin entorno en localhost", entorno: "", appURL: "http://localhost:3000", esperado: true},
		{nombre: "sin entorno en loopback", entorno: "", appURL: "http://127.0.0.1:3000", esperado: true},
		{nombre: "sin entorno ni URL", entorno: "", esperado: false},
		{nombre: "sin entorno en dominio público", entorno: "", appURL: "https://mesaclick.app", esperado: false},
		{nombre: "development", entorno: "development", esperado: true},
		{nombre: "local normalizado", entorno: " LOCAL ", esperado: true},
		{nombre: "qa", entorno: "qa", esperado: false},
		{nombre: "staging", entorno: "staging", esperado: false},
		{nombre: "produccion", entorno: "production", esperado: false},
	}

	for _, tt := range tests {
		t.Run(tt.nombre, func(t *testing.T) {
			if obtenido := esEntornoDesarrolloLocal(tt.entorno, tt.appURL); obtenido != tt.esperado {
				t.Fatalf(
					"esEntornoDesarrolloLocal(%q, %q) = %v; esperado %v",
					tt.entorno,
					tt.appURL,
					obtenido,
					tt.esperado,
				)
			}
		})
	}
}

func TestCORSCredencialesSoloParaOrigenPermitido(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_URL", "https://mesa-click-web.onrender.com")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://qa.mesaclick.com")
	handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	permitido := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	permitido.Header.Set("Origin", "https://mesa-click-web.onrender.com")
	respuestaPermitida := httptest.NewRecorder()
	handler.ServeHTTP(respuestaPermitida, permitido)
	if respuestaPermitida.Header().Get("Access-Control-Allow-Origin") != "https://mesa-click-web.onrender.com" ||
		respuestaPermitida.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatal("el frontend configurado debe poder enviar la cookie de sesión")
	}

	denegado := httptest.NewRequest(http.MethodOptions, "/auth/me", nil)
	denegado.Header.Set("Origin", "https://sitio-ajeno.example")
	respuestaDenegada := httptest.NewRecorder()
	handler.ServeHTTP(respuestaDenegada, denegado)
	if respuestaDenegada.Code != http.StatusForbidden || respuestaDenegada.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("un origen ajeno no debe recibir CORS con credenciales: status=%d", respuestaDenegada.Code)
	}
}

func nuevoCORSDePrueba(t *testing.T) http.Handler {
	t.Helper()
	t.Setenv("APP_URL", "https://app.example.com")
	t.Setenv("APP_ENV", "production")
	return corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
}

func TestCORS_RutasOAuthPublicas(t *testing.T) {
	manejador := nuevoCORSDePrueba(t)

	for _, ruta := range []string{"/oauth/token", "/oauth/register", "/oauth/revoke", "/.well-known/oauth-authorization-server"} {
		req := httptest.NewRequest(http.MethodOptions, ruta, nil)
		req.Header.Set("Origin", "https://claude.ai")
		rec := httptest.NewRecorder()
		manejador.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Errorf("%s OPTIONS: got %d, want 204", ruta, rec.Code)
		}
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
			t.Errorf("%s Allow-Origin: got %q, want *", ruta, got)
		}
		if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "" {
			t.Errorf("%s no debe enviar Allow-Credentials, got %q", ruta, got)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/oauth/token", nil)
	req.Header.Set("Origin", "https://claude.ai")
	rec := httptest.NewRecorder()
	manejador.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("POST /oauth/token: got %d / %q", rec.Code, rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORS_RutasNoPublicasMantienenPolitica(t *testing.T) {
	manejador := nuevoCORSDePrueba(t)

	for _, ruta := range []string{"/mesas", "/oauth/authorize"} {
		req := httptest.NewRequest(http.MethodOptions, ruta, nil)
		req.Header.Set("Origin", "https://claude.ai")
		rec := httptest.NewRecorder()
		manejador.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("origen ajeno en %s: got %d, want 403", ruta, rec.Code)
		}
	}

	req := httptest.NewRequest(http.MethodOptions, "/mesas", nil)
	req.Header.Set("Origin", "https://app.example.com")
	rec := httptest.NewRecorder()
	manejador.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent ||
		rec.Header().Get("Access-Control-Allow-Origin") != "https://app.example.com" ||
		rec.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Errorf("origen propio en /mesas: got %d, headers %v", rec.Code, rec.Header())
	}
}
