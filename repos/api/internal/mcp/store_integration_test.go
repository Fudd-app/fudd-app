package mcp

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"reflect"
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
	tomadoPor := func(pedidoID, nombre string) {
		exec(`UPDATE pedidos SET origen = 'mozo', creado_por_nombre = $2 WHERE id = $1`, pedidoID, nombre)
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
	mesa4 := scan(`INSERT INTO mesas (sucursal_id, numero, capacidad) VALUES ($1, 4, 2) RETURNING id`, a.SucursalID)
	mesa5 := scan(`INSERT INTO mesas (sucursal_id, numero, capacidad, cuenta_solicitada) VALUES ($1, 5, 2, TRUE) RETURNING id`, a.SucursalID)
	c1, c2, c3, c4, c9 := uuid(), uuid(), uuid(), uuid(), uuid()

	viejo := pedido(mesa1, a.SucursalID, 1, "listo", minutos(300), nil)
	item(viejo, artA, 5, 100, "listo", c9)
	p1 := pedido(mesa1, a.SucursalID, 2, "recibido", minutos(30), nil)
	item(p1, artA, 2, 10, "pendiente", c1)
	item(p1, artA, 1, 5, "preparando", c2)
	tomadoPor(p1, "Ana")
	listo2 := minutos(5)
	p2 := pedido(mesa1, a.SucursalID, 2, "listo", minutos(10), &listo2)
	item(p2, artA, 1, 20, "listo", c1)
	item(p2, artA, 1, 1, "listo", nil)
	listo3 := minutos(70)
	p3 := pedido(mesa2, a.SucursalID, 1, "cerrado", minutos(90), &listo3)
	item(p3, artA, 3, 10, "listo", nil)

	// Mesa 4: solo un pedido listo cargado por otro mozo (espera entrega). Mesa 5: cuenta solicitada con un pedido en cocina.
	listo4 := minutos(12)
	p4 := pedido(mesa4, a.SucursalID, 1, "listo", minutos(25), &listo4)
	item(p4, artA, 2, 10, "listo", c3)
	tomadoPor(p4, "Beto")
	p5 := pedido(mesa5, a.SucursalID, 1, "recibido", minutos(8), nil)
	item(p5, artA, 1, 7, "pendiente", c4)

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
	if err != nil || len(lista) != 1 || lista[0].ID != f.A.SucursalID || lista[0].MesasTotales != 5 || lista[0].MesasActivas != 4 {
		t.Errorf("lista A (mesa inactiva no es activa): %+v %v", lista, err)
	}
	lista, _ = s.SucursalesDelTenant(ctx, f.B.TenantID)
	if len(lista) != 1 || lista[0].ID != f.B.SucursalID || lista[0].MesasTotales != 2 || lista[0].MesasActivas != 2 {
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
	if len(mesas) != 4 {
		t.Fatalf("esperaba 4 mesas activas, obtuve %+v", mesas)
	}
	m1, m2, m4, m5 := mesas[0], mesas[1], mesas[2], mesas[3]
	if m1.Numero != 1 || m1.Sector != "Salón" || m1.Capacidad != 4 || m1.Estado != MesaOcupada ||
		m1.Comensales != 2 || !m1.ComensalesSinIdentificar || m1.PedidosActivos != 2 ||
		m1.TotalParcial != 46 || m1.MinutosDesdePrimerPedido == nil || *m1.MinutosDesdePrimerPedido != 30 ||
		m1.MinutosDesdeUltimoPedido == nil || *m1.MinutosDesdeUltimoPedido != 10 {
		t.Errorf("mesa 1 (la cuenta anterior no cuenta): %+v", m1)
	}
	// Hay pedidos en cocina: espera comida desde el más antiguo (30 min), aunque haya otro listo.
	if m1.Espera == nil || *m1.Espera != EsperaComida || m1.MinutosEnEspera == nil || *m1.MinutosEnEspera != 30 ||
		!reflect.DeepEqual(m1.TomadoPor, []string{"Ana", TomadorQR}) {
		t.Errorf("mesa 1 espera/tomado_por: %+v", m1)
	}
	if m2.Numero != 2 || m2.Sector != "" || m2.Estado != MesaPagoHabilitado || m2.Comensales != 1 ||
		!m2.ComensalesSinIdentificar || m2.PedidosActivos != 0 || m2.TotalParcial != 30 ||
		m2.MinutosDesdePrimerPedido == nil || *m2.MinutosDesdePrimerPedido != 90 {
		t.Errorf("mesa 2: %+v", m2)
	}
	if m2.Espera == nil || *m2.Espera != EsperaCobro || m2.MinutosEnEspera != nil || !reflect.DeepEqual(m2.TomadoPor, []string{TomadorQR}) {
		t.Errorf("mesa 2 espera cobro sin minutos: %+v", m2)
	}
	if m4.Numero != 4 || m4.Estado != MesaOcupada || m4.Espera == nil || *m4.Espera != EsperaEntrega ||
		m4.MinutosEnEspera == nil || *m4.MinutosEnEspera != 12 || !reflect.DeepEqual(m4.TomadoPor, []string{"Beto"}) {
		t.Errorf("mesa 4 (entrega desde el listo_at): %+v", m4)
	}
	if m5.Numero != 5 || m5.Estado != MesaCuentaSolicitada || m5.Espera == nil || *m5.Espera != EsperaCobro || m5.MinutosEnEspera != nil {
		t.Errorf("mesa 5 (cobro gana a comida): %+v", m5)
	}

	mesasB, err := s.EstadoSalon(ctx, f.B.TenantID, f.B.SucursalID, f.Ahora)
	if err != nil || len(mesasB) != 2 || mesasB[0].Estado != MesaOcupada || mesasB[0].TotalParcial != 31 ||
		mesasB[0].Comensales != 2 || mesasB[1].Estado != MesaLibre || mesasB[1].Comensales != 0 ||
		mesasB[1].MinutosDesdePrimerPedido != nil || mesasB[1].MinutosDesdeUltimoPedido != nil || mesasB[1].TotalParcial != 0 ||
		mesasB[1].Espera != nil || mesasB[1].MinutosEnEspera != nil || len(mesasB[1].TomadoPor) != 0 {
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
		{Mesa: 1, Estado: "recibido", MinutosDesdeIngreso: 30, Items: ItemsPedido{Pendientes: 1, Preparando: 1}},
		{Mesa: 4, Estado: "listo", MinutosDesdeIngreso: 25, Items: ItemsPedido{Listos: 1}},
		{Mesa: 1, Estado: "listo", MinutosDesdeIngreso: 10, Items: ItemsPedido{Listos: 2}},
		{Mesa: 5, Estado: "recibido", MinutosDesdeIngreso: 8, Items: ItemsPedido{Pendientes: 1}},
	}
	if len(pedidos) != len(esperados) {
		t.Fatalf("esperaba %d pedidos (sin cerrados ni cuentas anteriores), obtuve %+v", len(esperados), pedidos)
	}
	for i, e := range esperados {
		if pedidos[i].PedidoID == "" {
			t.Errorf("pedido %d sin pedido_id", i)
		}
		pedidos[i].PedidoID = ""
		if pedidos[i] != e {
			t.Errorf("pedido %d: %+v, esperaba %+v", i, pedidos[i], e)
		}
	}
}

func TestIntegrationUltimaHoraCocina(t *testing.T) {
	f := prepararIntegracion(t)
	ctx := context.Background()
	s := NuevoStore()

	if u, err := s.UltimaHoraCocina(ctx, f.A.TenantID, f.B.SucursalID, f.Ahora); err != nil || u != (UltimaHora{}) {
		t.Errorf("tenant A con sucursal de B: %+v %v", u, err)
	}
	// Ingresaron P1, P2, P4 y P5 (P3 hace 90 min no). Pasaron a listo P2 (despacho 5 min) y P4 (13 min);
	// P3 listo hace 70 min queda fuera. p90 = 5 + 0,9 * (13 - 5).
	u, err := s.UltimaHoraCocina(ctx, f.A.TenantID, f.A.SucursalID, f.Ahora)
	if err != nil || u.Ingresados60Min != 4 || u.Listos60Min != 2 || u.Balance60Min != 2 ||
		u.DespachoPromedioMin == nil || *u.DespachoPromedioMin != 9 || u.DespachoP90Min == nil || *u.DespachoP90Min != 12.2 {
		t.Errorf("última hora de A: %+v %v", u, err)
	}
	// Dos horas después no queda nada: los despachos son null, no 0.
	u, err = s.UltimaHoraCocina(ctx, f.A.TenantID, f.A.SucursalID, f.Ahora.Add(2*time.Hour))
	if err != nil || u.Ingresados60Min != 0 || u.Listos60Min != 0 || u.DespachoPromedioMin != nil || u.DespachoP90Min != nil {
		t.Errorf("sin datos: %+v %v", u, err)
	}
	u, err = s.UltimaHoraCocina(ctx, f.B.TenantID, f.B.SucursalID, f.Ahora)
	if err != nil || u.Ingresados60Min != 2 || u.Listos60Min != 1 || u.Balance60Min != 1 ||
		u.DespachoPromedioMin == nil || *u.DespachoPromedioMin != 30 || u.DespachoP90Min == nil || *u.DespachoP90Min != 30 {
		t.Errorf("tenant B: %+v %v", u, err)
	}
}

func TestIntegrationResumenPorHora(t *testing.T) {
	f := prepararIntegracion(t)
	ctx := context.Background()
	s := NuevoStore()
	desde, hasta := f.Ahora.Add(-3*time.Hour), f.Ahora.Add(time.Minute)

	if ajenas, err := s.ResumenPorHora(ctx, f.A.TenantID, f.B.SucursalID, desde, hasta, zonaHorariaNegocio); err != nil || len(ajenas) != 0 {
		t.Errorf("tenant A con sucursal de B: %+v %v", ajenas, err)
	}
	horas, err := s.ResumenPorHora(ctx, f.A.TenantID, f.A.SucursalID, desde, hasta, zonaHorariaNegocio)
	if err != nil || len(horas) == 0 {
		t.Fatalf("horas de A: %+v %v", horas, err)
	}
	var pedidos, comensales int
	var facturacion float64
	var despachos []float64
	for _, h := range horas {
		if len(h.Hora) != 5 || h.Hora[2] != ':' || !strings.HasSuffix(h.Hora, ":00") {
			t.Errorf("formato de hora: %q", h.Hora)
		}
		pedidos += h.Pedidos
		comensales += h.Comensales
		facturacion += h.Facturacion
		if h.DespachoPromedioMin != nil {
			despachos = append(despachos, *h.DespachoPromedioMin)
		}
	}
	// P1, P2, P3, P4 y P5; solo P3 está cerrado (3 x 10, despacho de 20 min).
	if pedidos != 5 || facturacion != 30 || comensales < 5 || len(despachos) != 1 || despachos[0] != 20 {
		t.Errorf("A: pedidos=%d comensales=%d facturación=%v despachos=%v (%+v)", pedidos, comensales, facturacion, despachos, horas)
	}
	horasB, err := s.ResumenPorHora(ctx, f.B.TenantID, f.B.SucursalID, desde, hasta, zonaHorariaNegocio)
	if err != nil || len(horasB) == 0 || len(horasB) > 2 {
		t.Errorf("horas de B: %+v %v", horasB, err)
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
	// ítems sin comensal (1), más una cuenta de 1 comensal en cada una de las mesas 4 y 5.
	if n, err := s.ComensalesAtendidos(ctx, f.A.TenantID, f.A.SucursalID, f.Ahora.Add(-3*time.Hour), hasta); err != nil || n != 5 {
		t.Errorf("A desde hace 3 h: %d %v", n, err)
	}
	// Se suma la cuenta anterior de la mesa 1 (c9).
	if n, err := s.ComensalesAtendidos(ctx, f.A.TenantID, f.A.SucursalID, f.Ahora.Add(-6*time.Hour), hasta); err != nil || n != 6 {
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
		Acceso:   permiteTodo(),
		Ahora:    func() time.Time { return f.Ahora },
	})

	// Valor distintivo de A en la salida estructurada de cada tool, y de B para la ausencia.
	distintivos := map[string][2]string{
		"listar_sucursales": {"Sucursal A", "Sucursal B"},
		"estado_salon":      {`"total_parcial":46`, `"total_parcial":31`},
		"estado_cocina":     {`"minutos_desde_ingreso":30`, `"minutos_desde_ingreso":31`},
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
			if err := errorEstructurado(t, resp); err.Error != CodigoSucursalNoEncontrada || strings.Contains(fmt.Sprint(err.Detalle), f.B.SucursalID) {
				t.Errorf("A con sucursal de B: %+v", err)
			}
		})
	}
}

// Escenario de la checklist de #82 de punta a punta: store real, HTTP real y las tres esperas posibles.
func TestIntegration_EscenarioInterpretable(t *testing.T) {
	f := prepararIntegracion(t)
	t.Setenv("MCP_HABILITADO", "true")
	h := nuevoHandler(authDePrueba, Dependencias{
		Store:    NuevoStore(),
		Metricas: metrica.NuevoService(metrica.NuevoStore()),
		Acceso:   permiteTodo(),
		Ahora:    func() time.Time { return f.Ahora },
	})

	var salon SalidaSalon
	resp := llamarTool(t, h, f.A.TenantID, "estado_salon", map[string]any{"incluir_libres": true})
	salidaTool(t, resp, &salon)
	t.Logf("estado_salon: %s", resp.Result.Content[0].Text)
	if salon.SucursalID != f.A.SucursalID || salon.SucursalNombre != "Sucursal A" || salon.Moneda != "ARS" || len(salon.Mesas) != 4 {
		t.Errorf("cabecera/mesas: %+v", salon)
	}
	esperado := TotalesSalon{MesasOcupadas: 4, MesasTotales: 4, Comensales: 5, MesasEsperando: Esperando{Comida: 1, Entrega: 1, Cobro: 2}}
	if salon.Totales != esperado {
		t.Errorf("totales: %+v, esperaba %+v", salon.Totales, esperado)
	}
	porTomador := map[string]TomadorSalon{}
	for _, tom := range salon.PorTomador {
		porTomador[tom.Tomador] = tom
	}
	if len(porTomador) != 3 || porTomador["Ana"].Mesas != 1 || porTomador["Beto"].Mesas != 1 || porTomador[TomadorQR].Mesas != 3 ||
		porTomador["Beto"].MesasEsperando.Entrega != 1 || porTomador["Ana"].MesasEsperando.Comida != 1 {
		t.Errorf("por_tomador (dos tomadores distintos y QR): %+v", salon.PorTomador)
	}
	if !strings.HasPrefix(salon.Resumen, "Sucursal A: 4 de 4 mesas ocupadas, 5 comensales.") {
		t.Errorf("resumen: %q", salon.Resumen)
	}

	var cocina SalidaCocina
	resp = llamarTool(t, h, f.A.TenantID, "estado_cocina", nil)
	salidaTool(t, resp, &cocina)
	t.Logf("estado_cocina: %s", resp.Result.Content[0].Text)
	if cocina.Cola != (ColaCocina{Recibido: 2, Listo: 2}) || len(cocina.Pedidos) != 4 ||
		cocina.Pedidos[0].MinutosDesdeIngreso != 30 || cocina.Pedidos[3].MinutosDesdeIngreso != 8 {
		t.Errorf("cola/pedidos: %+v", cocina)
	}
	if cocina.UltimaHora.DespachoPromedioMin == nil || *cocina.UltimaHora.DespachoPromedioMin != 9 {
		t.Errorf("pedido listo en la última hora: despacho no debe ser null: %+v", cocina.UltimaHora)
	}
	if !strings.HasPrefix(cocina.Resumen, "Sucursal A: en cocina 2 pedidos recibidos, 0 preparando, 2 listos.") {
		t.Errorf("resumen: %q", cocina.Resumen)
	}

	// Singular: B tiene un pedido recibido y uno listo.
	var cocinaB SalidaCocina
	salidaTool(t, llamarTool(t, h, f.B.TenantID, "estado_cocina", nil), &cocinaB)
	if cocinaB.Resumen != "Sucursal B: en cocina 1 pedido recibido, 0 preparando, 1 listo. Más demorado: mesa 1 (recibido) hace 5 minutos." {
		t.Errorf("plural/singular: %q", cocinaB.Resumen)
	}

	// El pedido cerrado (P3) fue hace 90 min: solo cuenta como "hoy" pasada la 01:30 del negocio.
	if ahora := f.Ahora.In(ubicacionNegocio()); ahora.Hour() < 2 {
		t.Skipf("resumen_turno: a las %s el pedido cerrado cae en el día anterior", ahora.Format("15:04"))
	}
	var turno SalidaTurno
	resp = llamarTool(t, h, f.A.TenantID, "resumen_turno", nil)
	salidaTool(t, resp, &turno)
	t.Logf("resumen_turno: %s", resp.Result.Content[0].Text)
	if turno.PedidosCerrados != 1 || turno.Facturacion != 30 || turno.TicketPromedio == nil || *turno.TicketPromedio != 30 ||
		turno.DespachoPromedioMin == nil || *turno.DespachoPromedioMin != 20 || turno.Moneda != "ARS" {
		t.Errorf("turno con pedido cerrado: %+v", turno)
	}
	porHora := 0
	for _, hora := range turno.PorHora {
		porHora += hora.Pedidos
	}
	if int64(porHora) != turno.Pedidos {
		t.Errorf("por_hora suma %d pedidos y el turno %d", porHora, turno.Pedidos)
	}
	var turnoB SalidaTurno
	salidaTool(t, llamarTool(t, h, f.B.TenantID, "resumen_turno", nil), &turnoB)
	if turnoB.PedidosCerrados != 0 || turnoB.TicketPromedio != nil || turnoB.DespachoPromedioMin != nil {
		t.Errorf("sin cerrados el ticket y el despacho son null: %+v", turnoB)
	}

	// sucursal_id inválido: el error lista los ids propios, nunca los ajenos.
	for _, invalido := range []string{f.B.SucursalID, "no-es-uuid"} {
		err := errorEstructurado(t, llamarTool(t, h, f.A.TenantID, "estado_salon", map[string]any{"sucursal_id": invalido}))
		if err.Error != CodigoSucursalNoEncontrada || !strings.Contains(fmt.Sprint(err.Detalle), f.A.SucursalID) ||
			strings.Contains(fmt.Sprint(err.Detalle), f.B.SucursalID) {
			t.Errorf("sucursal_id %q: %+v", invalido, err)
		}
	}

	var lista SalidaSucursales
	salidaTool(t, llamarTool(t, h, f.A.TenantID, "listar_sucursales", nil), &lista)
	if len(lista.Sucursales) != 1 || lista.Sucursales[0].MesasTotales != 5 || lista.Sucursales[0].MesasActivas != 4 ||
		lista.Sucursales[0].AbiertaAhora != nil || lista.Resumen != "1 sucursal en el negocio." {
		t.Errorf("sin horarios cargados abierta_ahora es null: %+v", lista)
	}
}
