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
