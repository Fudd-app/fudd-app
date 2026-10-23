package mcp

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/db"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/metrica"
)

type fixtureTenant struct {
	TenantID   string
	SucursalID string
}

type fixture struct {
	A, B  fixtureTenant
	Ahora time.Time
}

func prepararIntegracion(t *testing.T) fixture {
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
	ahora := time.Now().UTC().Truncate(time.Second)
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := db.Pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("fixture: %v\n%s", err, sql)
		}
	}
	scan := func(sql string, args ...any) string {
		t.Helper()
		var id string
		if err := db.Pool.QueryRow(ctx, sql, args...).Scan(&id); err != nil {
			t.Fatalf("fixture: %v\n%s", err, sql)
		}
		return id
	}

	crearBase := func(etiqueta string) (fixtureTenant, string) {
		tenant := scan(`INSERT INTO tenants (nombre, nombre_fantasia, rubro, slug) VALUES ($1, $1, 'bar', $2) RETURNING id`,
			"MCP "+etiqueta, fmt.Sprintf("mcp-%s-%d", etiqueta, sufijo))
		suc := scan(`INSERT INTO sucursales (tenant_id, nombre) VALUES ($1, $2) RETURNING id`, tenant, "Sucursal "+etiqueta)
		t.Cleanup(func() {
			_, _ = db.Pool.Exec(ctx, `DELETE FROM pedidos WHERE sucursal_id IN (SELECT id FROM sucursales WHERE tenant_id = $1)`, tenant)
			_, _ = db.Pool.Exec(ctx, `DELETE FROM tenants WHERE id = $1`, tenant)
		})
		cat := scan(`INSERT INTO categorias (tenant_id, nombre) VALUES ($1, 'Varios') RETURNING id`, tenant)
		art := scan(`INSERT INTO articulos (tenant_id, categoria_id, nombre, precio) VALUES ($1, $2, 'Plato', 10) RETURNING id`, tenant, cat)
		return fixtureTenant{TenantID: tenant, SucursalID: suc}, art
	}
	pedido := func(mesa, suc string, version int, estado string, creado time.Time, listo *time.Time) string {
		return scan(`INSERT INTO pedidos (mesa_id, sucursal_id, estado, cuenta_version, created_at, updated_at, listo_at)
			VALUES ($1, $2, $3, $4, $5, $5, $6) RETURNING id`, mesa, suc, estado, version, creado, listo)
	}
	item := func(pedidoID, art string, cantidad int, precio float64, estado string, comensal *string) {
		nombre := (*string)(nil)
		if comensal != nil {
			n := "Comensal"
			nombre = &n
		}
		exec(`INSERT INTO pedido_items (pedido_id, articulo_id, cantidad, precio_unitario, estado, comensal_id, comensal_nombre)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`, pedidoID, art, cantidad, precio, estado, comensal, nombre)
	}
	uuid := func() *string { id := scan(`SELECT gen_random_uuid()::text`); return &id }
	minutos := func(n int) time.Time { return ahora.Add(-time.Duration(n) * time.Minute) }

	// Tenant A: mesa 1 con cuenta nueva (versión 2) y un pedido de la cuenta anterior,
	// mesa 2 con pago habilitado, mesa 3 inactiva.
	a, artA := crearBase("A")
	sector := scan(`INSERT INTO sectores (sucursal_id, nombre) VALUES ($1, 'Salón') RETURNING id`, a.SucursalID)
	mesa1 := scan(`INSERT INTO mesas (sucursal_id, sector_id, numero, capacidad, cuenta_version) VALUES ($1, $2, 1, 4, 2) RETURNING id`, a.SucursalID, sector)
	mesa2 := scan(`INSERT INTO mesas (sucursal_id, numero, capacidad, pago_habilitado) VALUES ($1, 2, 2, TRUE) RETURNING id`, a.SucursalID)
	exec(`INSERT INTO mesas (sucursal_id, numero, capacidad, estado) VALUES ($1, 3, 2, 'inactiva')`, a.SucursalID)
	c1, c2, c9 := uuid(), uuid(), uuid()

	viejo := pedido(mesa1, a.SucursalID, 1, "listo", minutos(300), nil)
	item(viejo, artA, 5, 100, "listo", c9)
	p1 := pedido(mesa1, a.SucursalID, 2, "recibido", minutos(30), nil)
	item(p1, artA, 2, 10, "pendiente", c1)
	item(p1, artA, 1, 5, "preparando", c2)
	listo2 := minutos(5)
	p2 := pedido(mesa1, a.SucursalID, 2, "listo", minutos(10), &listo2)
	item(p2, artA, 1, 20, "listo", c1)
	item(p2, artA, 1, 1, "listo", nil)
	listo3 := minutos(70)
	p3 := pedido(mesa2, a.SucursalID, 1, "cerrado", minutos(90), &listo3)
	item(p3, artA, 3, 10, "listo", nil)

	// Tenant B: datos que nunca deben aparecer en las consultas de A.
	b, artB := crearBase("B")
	mesaB1 := scan(`INSERT INTO mesas (sucursal_id, numero, capacidad) VALUES ($1, 1, 4) RETURNING id`, b.SucursalID)
	exec(`INSERT INTO mesas (sucursal_id, numero, capacidad) VALUES ($1, 2, 4)`, b.SucursalID)
	pb1 := pedido(mesaB1, b.SucursalID, 1, "recibido", minutos(5), nil)
	item(pb1, artB, 4, 7, "pendiente", uuid())
	listoB := minutos(1)
	pb2 := pedido(mesaB1, b.SucursalID, 1, "listo", minutos(31), &listoB)
	item(pb2, artB, 1, 3, "listo", uuid())

	return fixture{A: a, B: b, Ahora: ahora}
}

func TestIntegrationSucursales(t *testing.T) {
	f := prepararIntegracion(t)
	ctx := context.Background()
	s := NuevoStore()

	ref, err := s.SucursalDelTenant(ctx, f.A.TenantID, f.A.SucursalID)
	if err != nil || ref.ID != f.A.SucursalID || ref.Nombre != "Sucursal A" {
		t.Errorf("sucursal propia: %+v %v", ref, err)
	}
	if _, err := s.SucursalDelTenant(ctx, f.A.TenantID, f.B.SucursalID); !errors.Is(err, ErrSucursalNoEncontrada) {
		t.Errorf("sucursal ajena: %v", err)
	}
	if _, err := s.SucursalDelTenant(ctx, f.A.TenantID, "no-es-uuid"); !errors.Is(err, ErrSucursalNoEncontrada) {
		t.Errorf("id inválido: %v", err)
	}

	lista, err := s.SucursalesDelTenant(ctx, f.A.TenantID)
	if err != nil || len(lista) != 1 || lista[0].ID != f.A.SucursalID || lista[0].MesasActivas != 2 {
		t.Errorf("lista A (mesa inactiva no cuenta): %+v %v", lista, err)
	}
	lista, _ = s.SucursalesDelTenant(ctx, f.B.TenantID)
	if len(lista) != 1 || lista[0].ID != f.B.SucursalID || lista[0].MesasActivas != 2 {
		t.Errorf("lista B: %+v", lista)
	}
}

func TestIntegrationEstadoSalon(t *testing.T) {
	f := prepararIntegracion(t)
	ctx := context.Background()
	s := NuevoStore()

	if ajenas, err := s.EstadoSalon(ctx, f.A.TenantID, f.B.SucursalID, f.Ahora); err != nil || len(ajenas) != 0 {
		t.Errorf("tenant A con sucursal de B: %+v %v", ajenas, err)
	}

	mesas, err := s.EstadoSalon(ctx, f.A.TenantID, f.A.SucursalID, f.Ahora)
	if err != nil {
		t.Fatal(err)
	}
	if len(mesas) != 2 {
		t.Fatalf("esperaba 2 mesas activas, obtuve %+v", mesas)
	}
	m1, m2 := mesas[0], mesas[1]
	if m1.Numero != 1 || m1.Sector != "Salón" || m1.Capacidad != 4 || m1.Estado != MesaOcupada ||
		m1.Comensales != 2 || !m1.ComensalesSinIdentificar || m1.PedidosActivos != 2 ||
		m1.TotalParcial != 46 || m1.MinutosDesdePrimerPedido == nil || *m1.MinutosDesdePrimerPedido != 30 {
		t.Errorf("mesa 1 (la cuenta anterior no cuenta): %+v", m1)
	}
	if m2.Numero != 2 || m2.Sector != "" || m2.Estado != MesaPagoHabilitado || m2.Comensales != 1 ||
		!m2.ComensalesSinIdentificar || m2.PedidosActivos != 0 || m2.TotalParcial != 30 ||
		m2.MinutosDesdePrimerPedido == nil || *m2.MinutosDesdePrimerPedido != 90 {
		t.Errorf("mesa 2: %+v", m2)
	}

	mesasB, err := s.EstadoSalon(ctx, f.B.TenantID, f.B.SucursalID, f.Ahora)
	if err != nil || len(mesasB) != 2 || mesasB[0].Estado != MesaOcupada || mesasB[0].TotalParcial != 31 ||
		mesasB[0].Comensales != 2 || mesasB[1].Estado != MesaLibre || mesasB[1].Comensales != 0 ||
		mesasB[1].MinutosDesdePrimerPedido != nil || mesasB[1].TotalParcial != 0 {
		t.Errorf("mesas de B: %+v %v", mesasB, err)
	}
}

func TestIntegrationPedidosCocina(t *testing.T) {
	f := prepararIntegracion(t)
	ctx := context.Background()
	s := NuevoStore()

	if ajenos, err := s.PedidosCocina(ctx, f.A.TenantID, f.B.SucursalID, f.Ahora); err != nil || len(ajenos) != 0 {
		t.Errorf("tenant A con sucursal de B: %+v %v", ajenos, err)
	}

	pedidos, err := s.PedidosCocina(ctx, f.A.TenantID, f.A.SucursalID, f.Ahora)
	if err != nil {
		t.Fatal(err)
	}
	esperados := []PedidoCocina{
		{Mesa: 1, Estado: "recibido", Minutos: 30, ItemsPendientes: 1, ItemsPreparando: 1},
		{Mesa: 1, Estado: "listo", Minutos: 10, ItemsListos: 2},
	}
	if len(pedidos) != len(esperados) {
		t.Fatalf("esperaba %d pedidos (sin cerrados ni cuentas anteriores), obtuve %+v", len(esperados), pedidos)
	}
	for i, e := range esperados {
		if pedidos[i] != e {
			t.Errorf("pedido %d: %+v, esperaba %+v", i, pedidos[i], e)
		}
	}
}

func TestIntegrationDespachoPromedio(t *testing.T) {
	f := prepararIntegracion(t)
	ctx := context.Background()
	s := NuevoStore()

	if v, err := s.DespachoPromedio(ctx, f.A.TenantID, f.B.SucursalID, f.Ahora.Add(-6*time.Hour)); err != nil || v != nil {
		t.Errorf("tenant A con sucursal de B: %v %v", v, err)
	}
	// P2 (5 min de despacho, listo hace 5) y P3 (20 min, listo hace 70).
	if v, err := s.DespachoPromedio(ctx, f.A.TenantID, f.A.SucursalID, f.Ahora.Add(-2*time.Hour)); err != nil || v == nil || *v != 12.5 {
		t.Errorf("últimas 2 h: %v %v", v, err)
	}
	if v, err := s.DespachoPromedio(ctx, f.A.TenantID, f.A.SucursalID, f.Ahora.Add(-time.Hour)); err != nil || v == nil || *v != 5 {
		t.Errorf("última hora: %v %v", v, err)
	}
	if v, err := s.DespachoPromedio(ctx, f.A.TenantID, f.A.SucursalID, f.Ahora); err != nil || v != nil {
		t.Errorf("sin datos debería ser nil: %v %v", v, err)
	}
	if v, err := s.DespachoPromedio(ctx, f.B.TenantID, f.B.SucursalID, f.Ahora.Add(-time.Hour)); err != nil || v == nil || *v != 30 {
		t.Errorf("tenant B: %v %v", v, err)
	}
}

func TestIntegrationComensalesAtendidos(t *testing.T) {
	f := prepararIntegracion(t)
	ctx := context.Background()
	s := NuevoStore()
	hasta := f.Ahora.Add(time.Minute)

	if n, err := s.ComensalesAtendidos(ctx, f.A.TenantID, f.B.SucursalID, f.Ahora.Add(-3*time.Hour), hasta); err != nil || n != 0 {
		t.Errorf("tenant A con sucursal de B: %d %v", n, err)
	}
	// Misma regla que estado_salon, por cuenta: mesa 1 tiene c1, c2 y un ítem sin
	// comensal (el mínimo de 1 no suma sobre los identificados: 2) y mesa 2 solo
	// ítems sin comensal (1).
	if n, err := s.ComensalesAtendidos(ctx, f.A.TenantID, f.A.SucursalID, f.Ahora.Add(-3*time.Hour), hasta); err != nil || n != 3 {
		t.Errorf("A desde hace 3 h: %d %v", n, err)
	}
	// Se suma la cuenta anterior de la mesa 1 (c9).
	if n, err := s.ComensalesAtendidos(ctx, f.A.TenantID, f.A.SucursalID, f.Ahora.Add(-6*time.Hour), hasta); err != nil || n != 4 {
		t.Errorf("A desde hace 6 h: %d %v", n, err)
	}
	if n, err := s.ComensalesAtendidos(ctx, f.B.TenantID, f.B.SucursalID, f.Ahora.Add(-3*time.Hour), hasta); err != nil || n != 2 {
		t.Errorf("B: %d %v", n, err)
	}
}

// Recorre las 4 tools por HTTP con el store real y claims de cada tenant: nunca
// aparece un dato del otro, ni pasando su sucursal_id.
func TestIntegration_ToolsAisladasPorTenant(t *testing.T) {
	f := prepararIntegracion(t)
	t.Setenv("MCP_HABILITADO", "true")
	h := nuevoHandler(authDePrueba, Dependencias{
		Store:    NuevoStore(),
		Metricas: metrica.NuevoService(metrica.NuevoStore()),
		Ahora:    func() time.Time { return f.Ahora },
	})

	// Valor distintivo de A en la salida estructurada de cada tool, y de B para la ausencia.
	distintivos := map[string][2]string{
		"listar_sucursales": {"Sucursal A", "Sucursal B"},
		"estado_salon":      {`"total_parcial":46`, `"total_parcial":31`},
		"estado_cocina":     {`"minutos":30`, `"minutos":31`},
		"resumen_turno":     {"Sucursal A", "Sucursal B"},
	}
	for tool, marcas := range distintivos {
		t.Run(tool, func(t *testing.T) {
			resp := llamarTool(t, h, f.A.TenantID, tool, nil)
			crudo := string(resp.Result.StructuredContent)
			if resp.Result.IsError || !strings.Contains(crudo, marcas[0]) || strings.Contains(crudo, marcas[1]) ||
				strings.Contains(crudo, f.B.SucursalID) {
				t.Errorf("A sin sucursal_id: %+v %s", resp.Result, crudo)
			}

			if tool == "listar_sucursales" {
				// No recibe argumentos: lista solo las del tenant.
				if !strings.Contains(crudo, f.A.SucursalID) {
					t.Errorf("listar_sucursales de A: %s", crudo)
				}
				return
			}
			resp = llamarTool(t, h, f.A.TenantID, tool, map[string]any{"sucursal_id": f.B.SucursalID})
			if !resp.Result.IsError || len(resp.Result.Content) == 0 || resp.Result.Content[0].Text != "Sucursal no encontrada" {
				t.Errorf("A con sucursal de B: %+v", resp.Result)
			}
		})
	}
}
