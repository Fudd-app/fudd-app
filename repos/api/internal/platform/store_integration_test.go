package platform

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/db"
)

// prepararIntegracion conecta a la DB real, migra y crea un fixture mínimo:
// tenant, sucursal, 2 mesas (una inactiva), usuario y un pedido con 2 ítems.
func prepararIntegracion(t *testing.T) (tenantID string, usuarioID string) {
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
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.Pool.Exec(ctx, query, args...); err != nil {
			t.Fatalf("ejecutando fixture (%s): %v", query, err)
		}
	}
	scan := func(dest *string, query string, args ...any) {
		t.Helper()
		if err := db.Pool.QueryRow(ctx, query, args...).Scan(dest); err != nil {
			t.Fatalf("ejecutando fixture (%s): %v", query, err)
		}
	}

	scan(&tenantID,
		`INSERT INTO tenants (nombre, nombre_fantasia, rubro, slug) VALUES ($1, $1, 'bar', $2) RETURNING id`,
		"Test Platform", fmt.Sprintf("test-platform-%d", sufijo))
	t.Cleanup(func() { _, _ = db.Pool.Exec(ctx, "DELETE FROM tenants WHERE id = $1", tenantID) })

	var sucursalID, mesaActivaID, categoriaID, articuloID, pedidoID string
	scan(&sucursalID, `INSERT INTO sucursales (tenant_id, nombre) VALUES ($1, 'Central') RETURNING id`, tenantID)
	scan(&mesaActivaID, `INSERT INTO mesas (sucursal_id, numero, estado) VALUES ($1, 1, 'activa') RETURNING id`, sucursalID)
	exec(`INSERT INTO mesas (sucursal_id, numero, estado) VALUES ($1, 2, 'inactiva')`, sucursalID)
	scan(&usuarioID,
		`INSERT INTO usuarios (tenant_id, email, nombre, rol) VALUES ($1, $2, 'Admin Platform', 'admin') RETURNING id`,
		tenantID, fmt.Sprintf("platform-%d@test.com", sufijo))
	scan(&categoriaID, `INSERT INTO categorias (tenant_id, nombre) VALUES ($1, 'Bebidas') RETURNING id`, tenantID)
	scan(&articuloID,
		`INSERT INTO articulos (tenant_id, categoria_id, nombre, precio) VALUES ($1, $2, 'Cerveza', 1000) RETURNING id`,
		tenantID, categoriaID)
	scan(&pedidoID, `INSERT INTO pedidos (mesa_id, sucursal_id) VALUES ($1, $2) RETURNING id`, mesaActivaID, sucursalID)
	// 2 x 1000 + 1 x 500 = 2500
	exec(`INSERT INTO pedido_items (pedido_id, articulo_id, cantidad, precio_unitario) VALUES ($1, $2, 2, 1000), ($1, $2, 1, 500)`,
		pedidoID, articuloID)
	return tenantID, usuarioID
}

func TestIntegration_Platform_Store(t *testing.T) {
	tenantID, usuarioID := prepararIntegracion(t)
	ctx := context.Background()
	store := NuevoStore()

	// Las métricas globales abarcan toda la DB: solo se verifica que no falle y que incluya el fixture.
	st, err := store.ObtenerMetricasGlobales(ctx)
	if err != nil {
		t.Fatalf("ObtenerMetricasGlobales: %v", err)
	}
	if st.TotalTenants < 1 || st.TotalMesas < 2 || st.TotalUsuarios < 1 || st.PedidosHoy < 1 || st.TotalVentasHoy < 2500 {
		t.Errorf("métricas globales no incluyen el fixture: %+v", st)
	}

	lista, err := store.ListarTenantsConsolidado(ctx)
	if err != nil {
		t.Fatalf("ListarTenantsConsolidado: %v", err)
	}
	var tr *TenantResumen
	for i := range lista {
		if lista[i].ID == tenantID {
			tr = &lista[i]
		}
	}
	if tr == nil {
		t.Fatalf("el tenant del fixture no está en el listado")
	}
	verificarResumen(t, tr)
	if tr.UsuariosPorRol["admin"] != 1 {
		t.Errorf("UsuariosPorRol[admin] = %d, quería 1", tr.UsuariosPorRol["admin"])
	}

	detalle, usuarios, movimientos, err := store.ObtenerDetalleTenant(ctx, tenantID)
	if err != nil {
		t.Fatalf("ObtenerDetalleTenant: %v", err)
	}
	verificarResumen(t, detalle)
	if len(usuarios) != 1 || usuarios[0].ID != usuarioID {
		t.Errorf("usuarios = %+v, quería solo el usuario del fixture", usuarios)
	}
	if len(movimientos) != 1 || movimientos[0].Tipo != "pedido" {
		t.Errorf("movimientos = %+v, quería 1 pedido", movimientos)
	}

	// Mismo recorrido a través del servicio (status + actividad).
	svc := NuevoService(store, NuevoTracker())
	if _, err := svc.ObtenerStatus(ctx); err != nil {
		t.Fatalf("ObtenerStatus: %v", err)
	}
	act, err := svc.ObtenerActividadTenant(ctx, tenantID)
	if err != nil || act.Tenant.MesasActivas != 1 {
		t.Fatalf("ObtenerActividadTenant: %+v, %v", act, err)
	}
}

func verificarResumen(t *testing.T, tr *TenantResumen) {
	t.Helper()
	if tr.TotalSucursales != 1 || tr.TotalMesas != 2 || tr.MesasActivas != 1 {
		t.Errorf("sucursales/mesas = %d/%d/%d, quería 1/2/1", tr.TotalSucursales, tr.TotalMesas, tr.MesasActivas)
	}
	if tr.TotalUsuarios != 1 || tr.TotalPedidos != 1 || tr.TotalVentas != 2500 {
		t.Errorf("usuarios/pedidos/ventas = %d/%d/%v, quería 1/1/2500", tr.TotalUsuarios, tr.TotalPedidos, tr.TotalVentas)
	}
	if tr.UltimoPedidoAt == nil {
		t.Errorf("UltimoPedidoAt = nil, quería fecha del pedido")
	}
}
