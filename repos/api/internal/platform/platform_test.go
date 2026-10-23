package platform_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/platform"
)

type mockPlatformStore struct {
	obtenerMetricasGlobalesFn func(ctx context.Context) (*platform.StatusPlataforma, error)
	listarTenantsFn           func(ctx context.Context) ([]platform.TenantResumen, error)
	obtenerDetalleTenantFn    func(ctx context.Context, tenantID string) (*platform.TenantResumen, []platform.UsuarioPresencia, []platform.Movimiento, error)
}

func (m *mockPlatformStore) ObtenerMetricasGlobales(ctx context.Context) (*platform.StatusPlataforma, error) {
	if m.obtenerMetricasGlobalesFn != nil {
		return m.obtenerMetricasGlobalesFn(ctx)
	}
	return &platform.StatusPlataforma{
		Servicio:        "mesa-click-api",
		Estado:          "ok",
		Database:        "ok",
		TotalTenants:    3,
		TotalSucursales: 5,
		TotalMesas:      25,
		TotalUsuarios:   10,
		PedidosHoy:      42,
		TotalVentasHoy:  125000.50,
	}, nil
}

func (m *mockPlatformStore) ListarTenantsConsolidado(ctx context.Context) ([]platform.TenantResumen, error) {
	if m.listarTenantsFn != nil {
		return m.listarTenantsFn(ctx)
	}
	return []platform.TenantResumen{
		{
			ID:              "t-1",
			Nombre:          "Bar Central",
			Slug:            "bar-central",
			Plan:            "pro",
			TotalSucursales: 2,
			TotalMesas:      15,
			MesasActivas:    10,
			TotalUsuarios:   4,
			UsuariosPorRol:  map[string]int{"admin": 1, "mozo": 3},
			TotalPedidos:    120,
			TotalVentas:     450000.0,
		},
	}, nil
}

func (m *mockPlatformStore) ObtenerDetalleTenant(ctx context.Context, tenantID string) (*platform.TenantResumen, []platform.UsuarioPresencia, []platform.Movimiento, error) {
	if m.obtenerDetalleTenantFn != nil {
		return m.obtenerDetalleTenantFn(ctx, tenantID)
	}
	ahora := time.Now()
	hace10Min := ahora.Add(-10 * time.Minute)
	hace1Dia := ahora.Add(-24 * time.Hour)

	tr := &platform.TenantResumen{
		ID:     tenantID,
		Nombre: "Bar Central",
		Slug:   "bar-central",
		Plan:   "pro",
	}

	usuarios := []platform.UsuarioPresencia{
		{
			ID:            "u-1",
			TenantID:      tenantID,
			Nombre:        "Pablo",
			Email:         "pablo@bar.com",
			Rol:           "admin",
			Estado:        "activo",
			UltimoLoginAt: &hace10Min,
		},
		{
			ID:            "u-2",
			TenantID:      tenantID,
			Nombre:        "Mozo Juan",
			Email:         "juan@bar.com",
			Rol:           "mozo",
			Estado:        "activo",
			UltimoLoginAt: &hace1Dia,
		},
	}

	movimientos := []platform.Movimiento{
		{
			ID:          "mov-1",
			Tipo:        "pedido",
			Descripcion: "Pedido #abc12345 en Mesa 3",
			Actor:       "Comensal",
			Fecha:       ahora.Add(-5 * time.Minute),
		},
	}

	return tr, usuarios, movimientos, nil
}

func TestTrackerPresencia(t *testing.T) {
	tr := platform.NuevoTracker()

	// 1. Sin registrar y sin login previo -> desconectado
	estado, act := tr.EstadoPresencia("u-nadie", nil)
	if estado != "desconectado" || act != nil {
		t.Errorf("esperaba desconectado y act nil, obtenido: %s, %v", estado, act)
	}

	// 2. Actividad registrada hace instantes -> online
	tr.RegistrarActividad("u-online")
	estado, act = tr.EstadoPresencia("u-online", nil)
	if estado != "online" || act == nil {
		t.Errorf("esperaba online, obtenido: %s", estado)
	}

	if online := tr.TotalUsuariosOnline(); online != 1 {
		t.Errorf("esperaba 1 online, obtenido: %d", online)
	}
}

func TestPlatformHandlers(t *testing.T) {
	mock := &mockPlatformStore{}
	tracker := platform.NuevoTracker()
	tracker.RegistrarActividad("u-1") // Actividad reciente para u-1

	svc := platform.NuevoService(mock, tracker)
	h := platform.NuevosHandlers(svc)

	// 1. GET /api/platform/status
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/platform/status", nil)
	h.Status(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Status handler falló: code=%d", rec.Code)
	}
	var st platform.StatusPlataforma
	if err := json.NewDecoder(rec.Body).Decode(&st); err != nil {
		t.Fatalf("error decodificando status: %v", err)
	}
	if st.TotalTenants != 3 || st.TotalMesas != 25 || st.UsuariosOnline != 1 {
		t.Errorf("datos inesperados en status: %+v", st)
	}

	// 2. GET /api/platform/tenants
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/platform/tenants", nil)
	h.ListarTenants(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("ListarTenants falló: code=%d", rec.Code)
	}
	var tenants []platform.TenantResumen
	if err := json.NewDecoder(rec.Body).Decode(&tenants); err != nil {
		t.Fatalf("error decodificando tenants: %v", err)
	}
	if len(tenants) != 1 || tenants[0].Slug != "bar-central" {
		t.Errorf("tenants inesperados: %+v", tenants)
	}

	// 3. GET /api/platform/tenants/{id}/actividad
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/platform/tenants/t-1/actividad", nil)
	req.SetPathValue("id", "t-1")
	h.DetalleActividad(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("DetalleActividad falló: code=%d", rec.Code)
	}
	var det platform.TenantDetalleActividad
	if err := json.NewDecoder(rec.Body).Decode(&det); err != nil {
		t.Fatalf("error decodificando detalle: %v", err)
	}
	if len(det.Usuarios) != 2 || len(det.UltimosMovimientos) != 1 {
		t.Errorf("usuarios o movimientos inesperados: %+v", det)
	}
	// u-1 tiene actividad en tracker -> debe ser "online"
	if det.Usuarios[0].EstadoPresencia != "online" {
		t.Errorf("esperaba u-1 online, obtenido: %s", det.Usuarios[0].EstadoPresencia)
	}
	// u-2 sin actividad en tracker pero login hace 1 dia -> debe ser "desconectado"
	if det.Usuarios[1].EstadoPresencia != "desconectado" {
		t.Errorf("esperaba u-2 desconectado, obtenido: %s", det.Usuarios[1].EstadoPresencia)
	}
}
