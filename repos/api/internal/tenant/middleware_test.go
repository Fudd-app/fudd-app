package tenant_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/tenant"
)

func TestRequerirCuota(t *testing.T) {
	pasado := time.Now().Add(-24 * time.Hour)
	futuro := time.Now().Add(24 * time.Hour)

	nextOK := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	})

	t.Run("Sin claims devuelve 401 fail-closed", func(t *testing.T) {
		store := &mockStore{}
		mw := tenant.RequerirCuota(store, tenant.RecursoMesas)

		req := httptest.NewRequest("POST", "/mesas", nil)
		rec := httptest.NewRecorder()

		mw(nextOK).ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status: obtenido %d, esperado %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("Error de base de datos devuelve 500 fail-closed", func(t *testing.T) {
		store := &mockStore{
			obtenerEstadoCuotasFn: func(ctx context.Context, tenantID string) (*tenant.EstadoCuotas, error) {
				return nil, errors.New("db error")
			},
		}
		mw := tenant.RequerirCuota(store, tenant.RecursoMesas)

		req := httptest.NewRequest("POST", "/mesas", nil)
		req = req.WithContext(auth.ContextConClaims(req.Context(), &auth.Claims{TenantID: "t-1"}))
		rec := httptest.NewRecorder()

		mw(nextOK).ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("status: obtenido %d, esperado %d", rec.Code, http.StatusInternalServerError)
		}
	})

	t.Run("Plan Free al límite de mesas devuelve 403 estructurado", func(t *testing.T) {
		store := &mockStore{
			obtenerEstadoCuotasFn: func(ctx context.Context, tenantID string) (*tenant.EstadoCuotas, error) {
				return &tenant.EstadoCuotas{
					Plan: tenant.PlanFree,
					Uso: map[tenant.Recurso]int{
						tenant.RecursoMesas: 10,
					},
				}, nil
			},
		}
		mw := tenant.RequerirCuota(store, tenant.RecursoMesas)

		req := httptest.NewRequest("POST", "/mesas", nil)
		req = req.WithContext(auth.ContextConClaims(req.Context(), &auth.Claims{TenantID: "t-1"}))
		rec := httptest.NewRecorder()

		mw(nextOK).ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("status: obtenido %d, esperado %d", rec.Code, http.StatusForbidden)
		}

		var payload struct {
			Error   string `json:"error"`
			Codigo  string `json:"codigo"`
			Detalle struct {
				Recurso string `json:"recurso"`
				Nombre  string `json:"nombre"`
				Usado   int    `json:"usado"`
				Limite  int    `json:"limite"`
				Plan    string `json:"plan"`
				Accion  string `json:"accion"`
			} `json:"detalle"`
		}

		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("error parseando JSON: %v", err)
		}

		if payload.Codigo != "PLAN_LIMIT_REACHED" {
			t.Errorf("codigo: obtenido %q, esperado 'PLAN_LIMIT_REACHED'", payload.Codigo)
		}
		if payload.Detalle.Recurso != "mesas" {
			t.Errorf("detalle.recurso: obtenido %q, esperado 'mesas'", payload.Detalle.Recurso)
		}
		if payload.Detalle.Usado != 10 || payload.Detalle.Limite != 10 {
			t.Errorf("detalle usado/limite inesperado: %+v", payload.Detalle)
		}
		if payload.Error != "Alcanzaste el límite de 10 mesas de tu plan Free." {
			t.Errorf("mensaje de error inesperado: %q", payload.Error)
		}
	})

	t.Run("Plan Free con margen permite la operación", func(t *testing.T) {
		store := &mockStore{
			obtenerEstadoCuotasFn: func(ctx context.Context, tenantID string) (*tenant.EstadoCuotas, error) {
				return &tenant.EstadoCuotas{
					Plan: tenant.PlanFree,
					Uso: map[tenant.Recurso]int{
						tenant.RecursoMesas: 8,
					},
				}, nil
			},
		}
		mw := tenant.RequerirCuota(store, tenant.RecursoMesas)

		req := httptest.NewRequest("POST", "/mesas", nil)
		req = req.WithContext(auth.ContextConClaims(req.Context(), &auth.Claims{TenantID: "t-1"}))
		rec := httptest.NewRecorder()

		mw(nextOK).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status: obtenido %d, esperado %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("Plan Pro vencido degrada a Free y bloquea al límite", func(t *testing.T) {
		store := &mockStore{
			obtenerEstadoCuotasFn: func(ctx context.Context, tenantID string) (*tenant.EstadoCuotas, error) {
				return &tenant.EstadoCuotas{
					Plan:      tenant.PlanPro,
					PlanHasta: &pasado,
					Uso: map[tenant.Recurso]int{
						tenant.RecursoMesas: 10,
					},
				}, nil
			},
		}
		mw := tenant.RequerirCuota(store, tenant.RecursoMesas)

		req := httptest.NewRequest("POST", "/mesas", nil)
		req = req.WithContext(auth.ContextConClaims(req.Context(), &auth.Claims{TenantID: "t-1"}))
		rec := httptest.NewRecorder()

		mw(nextOK).ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("status: obtenido %d, esperado %d", rec.Code, http.StatusForbidden)
		}
	})

	t.Run("Carga masiva bloqueada en Free", func(t *testing.T) {
		store := &mockStore{
			obtenerEstadoCuotasFn: func(ctx context.Context, tenantID string) (*tenant.EstadoCuotas, error) {
				return &tenant.EstadoCuotas{
					Plan: tenant.PlanFree,
				}, nil
			},
		}
		mw := tenant.RequerirCuota(store, tenant.RecursoCargaMasiva)

		req := httptest.NewRequest("POST", "/carta/importar", nil)
		req = req.WithContext(auth.ContextConClaims(req.Context(), &auth.Claims{TenantID: "t-1"}))
		rec := httptest.NewRecorder()

		mw(nextOK).ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("status: obtenido %d, esperado %d", rec.Code, http.StatusForbidden)
		}
	})

	t.Run("Carga masiva permitida en Pro activo", func(t *testing.T) {
		store := &mockStore{
			obtenerEstadoCuotasFn: func(ctx context.Context, tenantID string) (*tenant.EstadoCuotas, error) {
				return &tenant.EstadoCuotas{
					Plan:      tenant.PlanPro,
					PlanHasta: &futuro,
				}, nil
			},
		}
		mw := tenant.RequerirCuota(store, tenant.RecursoCargaMasiva)

		req := httptest.NewRequest("POST", "/carta/importar", nil)
		req = req.WithContext(auth.ContextConClaims(req.Context(), &auth.Claims{TenantID: "t-1"}))
		rec := httptest.NewRecorder()

		mw(nextOK).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status: obtenido %d, esperado %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("Plan Free sin personalización devuelve mensaje propio, no el de carga masiva", func(t *testing.T) {
		store := &mockStore{
			obtenerEstadoCuotasFn: func(ctx context.Context, tenantID string) (*tenant.EstadoCuotas, error) {
				return &tenant.EstadoCuotas{Plan: tenant.PlanFree}, nil
			},
		}
		mw := tenant.RequerirCuota(store, tenant.RecursoPersonalizacion)

		req := httptest.NewRequest("PATCH", "/carta/categorias/c-1/icono", nil)
		req = req.WithContext(auth.ContextConClaims(req.Context(), &auth.Claims{TenantID: "t-1"}))
		rec := httptest.NewRecorder()

		mw(nextOK).ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("status: obtenido %d, esperado %d", rec.Code, http.StatusForbidden)
		}
		var payload struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
			t.Fatalf("decodificando respuesta: %v", err)
		}
		if strings.Contains(payload.Error, "carga masiva") {
			t.Errorf("el mensaje no debe mencionar carga masiva: %q", payload.Error)
		}
		if !strings.Contains(payload.Error, "personalización visual") {
			t.Errorf("el mensaje debe mencionar personalización visual: %q", payload.Error)
		}
	})

	t.Run("Plan Free en carga masiva mantiene el mensaje original", func(t *testing.T) {
		store := &mockStore{
			obtenerEstadoCuotasFn: func(ctx context.Context, tenantID string) (*tenant.EstadoCuotas, error) {
				return &tenant.EstadoCuotas{Plan: tenant.PlanFree}, nil
			},
		}
		mw := tenant.RequerirCuota(store, tenant.RecursoCargaMasiva)

		req := httptest.NewRequest("POST", "/carta/importar", nil)
		req = req.WithContext(auth.ContextConClaims(req.Context(), &auth.Claims{TenantID: "t-1"}))
		rec := httptest.NewRecorder()

		mw(nextOK).ServeHTTP(rec, req)

		var payload struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
			t.Fatalf("decodificando respuesta: %v", err)
		}
		esperado := "La carga masiva es una funcionalidad exclusiva del plan Pro."
		if payload.Error != esperado {
			t.Errorf("error: obtenido %q, esperado %q", payload.Error, esperado)
		}
	})
}
