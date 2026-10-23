package auth_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
)

func TestEsSuperadmin(t *testing.T) {
	os.Setenv("SUPERADMIN_EMAILS", "pablo@fudd.com.ar, socio2@gmail.com, SOCIO3@OUTLOOK.COM ")
	defer os.Unsetenv("SUPERADMIN_EMAILS")

	casos := []struct {
		email    string
		esperado bool
	}{
		{"pablo@fudd.com.ar", true},
		{"PABLO@fudd.com.ar", true},
		{"socio2@gmail.com", true},
		{"socio3@outlook.com", true},
		{"otro@gmail.com", false},
		{"", false},
	}

	for _, c := range casos {
		resultado := auth.EsSuperadmin(c.email)
		if resultado != c.esperado {
			t.Errorf("EsSuperadmin(%q) = %v; se esperaba %v", c.email, resultado, c.esperado)
		}
	}
}

func TestRequerirSuperadmin(t *testing.T) {
	secreto := "secreto-superadmin-test"
	os.Setenv("JWT_SECRET", secreto)
	os.Setenv("SUPERADMIN_EMAILS", "admin@fudd.com.ar")
	defer func() {
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("SUPERADMIN_EMAILS")
	}()

	mock := &mockStore{
		obtenerUsuarioPorIDFn: func(ctx context.Context, id string) (*auth.UsuarioAuth, error) {
			if id == "u-super" {
				return &auth.UsuarioAuth{
					ID:             "u-super",
					TenantID:       "t-1",
					Email:          "admin@fudd.com.ar",
					Rol:            auth.RolDuenio,
					Estado:         "activo",
					SessionVersion: 1,
				}, nil
			}
			return &auth.UsuarioAuth{
				ID:             "u-mortal",
				TenantID:       "t-2",
				Email:          "mortal@negocio.com",
				Rol:            auth.RolDuenio,
				Estado:         "activo",
				SessionVersion: 1,
			}, nil
		},
	}

	superadminMiddleware := auth.RequerirSuperadmin(mock)
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	handler := superadminMiddleware(nextHandler)

	// 1. Sin token -> 401
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/platform/status", nil)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("se esperaba 401 sin token, obtenido: %d", rec.Code)
	}

	// 2. Token de usuario no superadmin -> 403
	tokenMortal, _ := auth.GenerarJWT(&auth.Claims{
		UsuarioID:      "u-mortal",
		TenantID:       "t-2",
		Rol:            auth.RolDuenio,
		SessionVersion: 1,
	}, secreto)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/platform/status", nil)
	req.Header.Set("Authorization", "Bearer "+tokenMortal)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("se esperaba 403 para usuario no superadmin, obtenido: %d", rec.Code)
	}

	// 3. Token de superadmin -> 200
	tokenSuper, _ := auth.GenerarJWT(&auth.Claims{
		UsuarioID:      "u-super",
		TenantID:       "t-1",
		Rol:            auth.RolDuenio,
		SessionVersion: 1,
	}, secreto)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/platform/status", nil)
	req.Header.Set("Authorization", "Bearer "+tokenSuper)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("se esperaba 200 para superadmin, obtenido: %d", rec.Code)
	}
}

func TestMiSesionEsSuperadmin(t *testing.T) {
	os.Setenv("SUPERADMIN_EMAILS", "admin@fudd.com.ar")
	defer os.Unsetenv("SUPERADMIN_EMAILS")

	mock := &mockStore{
		obtenerUsuarioPorIDFn: func(ctx context.Context, id string) (*auth.UsuarioAuth, error) {
			email := "mortal@negocio.com"
			if id == "u-super" {
				email = "admin@fudd.com.ar"
			}
			return &auth.UsuarioAuth{ID: id, TenantID: "t-1", Email: email, Rol: auth.RolDuenio, Estado: "activo"}, nil
		},
	}
	h := auth.NuevosHandlers(auth.NuevoServiceConGoogle(mock, nil, "", nil), false)

	casos := []struct {
		usuarioID string
		esperado  bool
	}{
		{"u-super", true},
		{"u-mortal", false},
	}
	for _, c := range casos {
		req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
		req = req.WithContext(auth.ContextConClaims(req.Context(), &auth.Claims{UsuarioID: c.usuarioID}))
		rec := httptest.NewRecorder()
		h.MiSesion(rec, req)

		var body map[string]any
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("respuesta inválida: %v", err)
		}
		if body["es_superadmin"] != c.esperado {
			t.Errorf("%s: es_superadmin = %v; se esperaba %v", c.usuarioID, body["es_superadmin"], c.esperado)
		}
	}
}
