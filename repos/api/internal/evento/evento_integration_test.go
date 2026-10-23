package evento_test

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/carta"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/db"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/evento"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/mercadopago"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/mesa"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/pedido"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/tenant"
)

type negocioPrueba struct {
	TenantID   string
	SucursalID string
	AdminID    string
	MesaID     string
	QRToken    string
	ArticuloID string
}

func (n negocioPrueba) ctxAdmin() context.Context {
	return auth.ContextConClaims(context.Background(), &auth.Claims{
		UsuarioID: n.AdminID, TenantID: n.TenantID, Rol: auth.RolDuenio,
	})
}

func crearNegocio(t *testing.T, etiqueta string) negocioPrueba {
	t.Helper()
	ctx := context.Background()
	sufijo := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(1_000_000_000)

	tenantSvc := tenant.NuevoService(tenant.NuevoStore())
	creado, err := tenantSvc.Crear(ctx, tenant.OnboardingInput{
		Nombre:      "Eventos " + etiqueta,
		Slug:        fmt.Sprintf("ev-%s-%d", etiqueta, sufijo),
		EmailAdmin:  fmt.Sprintf("ev-%s-%d@test.com", etiqueta, sufijo),
		NombreAdmin: "Admin " + etiqueta,
		Password:    "MesaClick2026",
	})
	if err != nil {
		t.Fatalf("error creando tenant: %v", err)
	}
	t.Cleanup(func() {
		// pedidos referencia a la mesa con RESTRICT: se borran antes que el tenant.
		if _, err := db.Pool.Exec(ctx, `DELETE FROM pedidos WHERE sucursal_id IN (SELECT id FROM sucursales WHERE tenant_id = $1)`, creado.ID); err != nil {
			t.Errorf("limpieza de pedidos: %v", err)
		}
		if _, err := db.Pool.Exec(ctx, "DELETE FROM tenants WHERE id = $1", creado.ID); err != nil {
			t.Errorf("limpieza del tenant: %v", err)
		}
	})

	n := negocioPrueba{TenantID: creado.ID}
	if err := db.Pool.QueryRow(ctx, `SELECT id FROM sucursales WHERE tenant_id = $1 LIMIT 1`, creado.ID).Scan(&n.SucursalID); err != nil {
		t.Fatalf("sucursal del tenant: %v", err)
	}
	if err := db.Pool.QueryRow(ctx, `SELECT id FROM usuarios WHERE tenant_id = $1 LIMIT 1`, creado.ID).Scan(&n.AdminID); err != nil {
		t.Fatalf("admin del tenant: %v", err)
	}

	m, err := mesa.NuevoService(mesa.NuevoStore()).Crear(ctx, creado.ID, mesa.MesaInput{SucursalID: n.SucursalID, Numero: 1, Capacidad: 4})
	if err != nil {
		t.Fatalf("error creando mesa: %v", err)
	}
	n.MesaID, n.QRToken = m.ID, m.QRToken

	cartaSvc := carta.NuevoService(carta.NuevoStore())
	cat, err := cartaSvc.CrearCategoria(ctx, creado.ID, carta.CategoriaInput{Nombre: "Bebidas"})
	if err != nil {
		t.Fatalf("error creando categoría: %v", err)
	}
	art, err := cartaSvc.CrearArticulo(ctx, creado.ID, carta.ArticuloInput{CategoriaID: cat.ID, Nombre: "Agua", Precio: 250})
	if err != nil {
		t.Fatalf("error creando artículo: %v", err)
	}
	n.ArticuloID = art.ID
	return n
}

func prepararDB(t *testing.T) {
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
}

func itemsQR(articuloID string, nombres ...string) []pedido.NuevoItemInput {
	ids := []string{"47dc8c9e-fb98-44d7-80a1-b598addc1e8a", "5ef10747-503a-4d72-a433-94978a2447e6"}
	items := make([]pedido.NuevoItemInput, len(nombres))
	for i, nombre := range nombres {
		items[i] = pedido.NuevoItemInput{ArticuloID: articuloID, Cantidad: 1, ComensalID: ids[i], ComensalNombre: nombre}
	}
	return items
}

type esperado struct {
	tipo, anterior, nuevo, actorTipo, actorNombre, actorRol string
	cuentaVersion                                           int
}

func TestIntegration_RegistroDeHitosDeUnaCuentaCompleta(t *testing.T) {
	prepararDB(t)
	ctx := context.Background()
	a := crearNegocio(t, "a")
	b := crearNegocio(t, "b")

	pedidoSvc := pedido.NuevoService(pedido.NuevoStore())
	mesaSvc := mesa.NuevoService(mesa.NuevoStore())
	admin := a.ctxAdmin()

	// Pedido por QR (abre la cuenta) y un segundo pedido (no debe volver a abrirla).
	p1, err := pedidoSvc.Crear(ctx, pedido.NuevoPedidoInput{MesaID: a.MesaID, Items: itemsQR(a.ArticuloID, "Mateo", "Juani")})
	if err != nil {
		t.Fatalf("pedido por QR: %v", err)
	}
	if _, err := pedidoSvc.Crear(ctx, pedido.NuevoPedidoInput{MesaID: a.MesaID, Items: itemsQR(a.ArticuloID, "")}); err != nil {
		t.Fatalf("segundo pedido por QR: %v", err)
	}

	// Cambio de estado del pedido y cocina por ítem, con el admin autenticado.
	if _, err := pedidoSvc.CambiarEstado(admin, p1.ID, a.TenantID, "", "preparando"); err != nil {
		t.Fatalf("cambiar estado: %v", err)
	}
	// Repetir el mismo estado no es una transición: no debe generar hito.
	if _, err := pedidoSvc.CambiarEstado(admin, p1.ID, a.TenantID, "", "preparando"); err != nil {
		t.Fatalf("cambiar estado repetido: %v", err)
	}
	for _, estado := range []struct{ item, estado string }{
		{p1.Items[0].ID, "preparando"}, {p1.Items[0].ID, "listo"}, {p1.Items[1].ID, "listo"},
	} {
		if _, err := pedidoSvc.CambiarEstadoItem(admin, estado.item, a.TenantID, "", estado.estado); err != nil {
			t.Fatalf("cambiar estado de ítem: %v", err)
		}
	}

	if _, err := mesaSvc.SolicitarCuentaPorQRToken(ctx, a.QRToken); err != nil {
		t.Fatalf("solicitar cuenta: %v", err)
	}
	if _, err := mesaSvc.HabilitarPago(admin, a.MesaID, a.TenantID, true); err != nil {
		t.Fatalf("habilitar pago: %v", err)
	}
	if _, err := mercadopago.NuevoStore().RegistrarPagoAprobado(ctx, a.MesaID, 1, "pago-ev-"+a.MesaID, "", 750, map[string]any{"status": "approved"}); err != nil {
		t.Fatalf("registrar pago: %v", err)
	}
	if _, err := mesaSvc.CerrarCuenta(admin, a.MesaID, a.TenantID); err != nil {
		t.Fatalf("cerrar cuenta: %v", err)
	}

	// La cuenta siguiente vuelve a abrirse con su propia versión.
	if _, err := pedidoSvc.Crear(ctx, pedido.NuevoPedidoInput{MesaID: a.MesaID, Items: itemsQR(a.ArticuloID, "Sofi")}); err != nil {
		t.Fatalf("pedido de la nueva cuenta: %v", err)
	}

	const nombreAdmin = "Admin a"
	esperados := []esperado{
		{"cuenta_abierta", "", "recibido", "comensal", "Mateo", "", 1},
		{"pedido_creado", "", "recibido", "comensal", "Mateo", "", 1},
		{"pedido_creado", "", "recibido", "comensal", "", "", 1},
		{"pedido_estado", "recibido", "preparando", "usuario", nombreAdmin, auth.RolDuenio, 1},
		{"item_estado", "pendiente", "preparando", "usuario", nombreAdmin, auth.RolDuenio, 1},
		{"item_estado", "preparando", "listo", "usuario", nombreAdmin, auth.RolDuenio, 1},
		{"item_estado", "pendiente", "listo", "usuario", nombreAdmin, auth.RolDuenio, 1},
		{"pedido_estado", "preparando", "listo", "usuario", nombreAdmin, auth.RolDuenio, 1},
		{"cuenta_solicitada", "abierta", "solicitada", "comensal", "", "", 1},
		{"pago_habilitado", "deshabilitado", "habilitado", "usuario", nombreAdmin, auth.RolDuenio, 1},
		{"pago_aprobado", "", "aprobado", "mercadopago", "", "", 1},
		{"cuenta_cerrada", "", "", "usuario", nombreAdmin, auth.RolDuenio, 1},
		{"cuenta_abierta", "", "recibido", "comensal", "Sofi", "", 2},
		{"pedido_creado", "", "recibido", "comensal", "Sofi", "", 2},
	}

	eventos, err := evento.EventosPorMesa(ctx, a.TenantID, a.SucursalID, a.MesaID, time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("EventosPorMesa: %v", err)
	}
	if len(eventos) != len(esperados) {
		for _, e := range eventos {
			t.Logf("evento: %s %s->%s %s/%s v%d", e.Tipo, e.EstadoAnterior, e.EstadoNuevo, e.ActorTipo, e.ActorNombre, e.CuentaVersion)
		}
		t.Fatalf("cantidad de eventos: got %d, want %d", len(eventos), len(esperados))
	}
	for i, want := range esperados {
		got := eventos[i]
		if got.Tipo != want.tipo || got.EstadoAnterior != want.anterior || got.EstadoNuevo != want.nuevo ||
			got.ActorTipo != want.actorTipo || got.ActorNombre != want.actorNombre || got.ActorRol != want.actorRol ||
			got.CuentaVersion != want.cuentaVersion {
			t.Errorf("evento %d: got %s %q->%q %s/%q/%q v%d, want %+v", i, got.Tipo, got.EstadoAnterior, got.EstadoNuevo,
				got.ActorTipo, got.ActorNombre, got.ActorRol, got.CuentaVersion, want)
		}
		if want.actorTipo == "usuario" && (got.UsuarioID == nil || *got.UsuarioID != a.AdminID) {
			t.Errorf("evento %d: usuario_id incorrecto: %v", i, got.UsuarioID)
		}
		if got.TS.IsZero() || (i > 0 && got.TS.Before(eventos[i-1].TS)) {
			t.Errorf("evento %d: ts fuera de orden", i)
		}
	}
	if eventos[4].PedidoItemID == nil || *eventos[4].PedidoItemID != p1.Items[0].ID {
		t.Errorf("el hito de ítem debe referenciar el ítem: %v", eventos[4].PedidoItemID)
	}

	// Lectura por sucursal y ventana temporal.
	porSucursal, err := evento.EventosPorSucursal(ctx, a.TenantID, a.SucursalID, time.Time{}, time.Time{})
	if err != nil || len(porSucursal) != len(esperados) {
		t.Fatalf("EventosPorSucursal: len=%d err=%v", len(porSucursal), err)
	}
	futuro, err := evento.EventosPorSucursal(ctx, a.TenantID, a.SucursalID, time.Now().Add(time.Hour), time.Time{})
	if err != nil || len(futuro) != 0 {
		t.Fatalf("la ventana futura debe estar vacía: len=%d err=%v", len(futuro), err)
	}

	// Aislamiento por tenant: B no lee los eventos de A ni con los ids de A.
	if ajenos, err := evento.EventosPorMesa(ctx, b.TenantID, a.SucursalID, a.MesaID, time.Time{}, time.Time{}); err != nil || len(ajenos) != 0 {
		t.Fatalf("el tenant B no debe leer eventos de la mesa de A: len=%d err=%v", len(ajenos), err)
	}
	if ajenos, err := evento.EventosPorSucursal(ctx, b.TenantID, a.SucursalID, time.Time{}, time.Time{}); err != nil || len(ajenos) != 0 {
		t.Fatalf("el tenant B no debe leer eventos de la sucursal de A: len=%d err=%v", len(ajenos), err)
	}
	if propios, err := evento.EventosPorSucursal(ctx, b.TenantID, b.SucursalID, time.Time{}, time.Time{}); err != nil || len(propios) != 0 {
		t.Fatalf("B no registró nada todavía: len=%d err=%v", len(propios), err)
	}
}

func TestIntegration_FalloDelEventoRevierteLaOperacion(t *testing.T) {
	prepararDB(t)
	ctx := context.Background()
	a := crearNegocio(t, "rb")
	pedidoSvc := pedido.NuevoService(pedido.NuevoStore())

	p, err := pedidoSvc.Crear(ctx, pedido.NuevoPedidoInput{MesaID: a.MesaID, Items: itemsQR(a.ArticuloID, "Mateo")})
	if err != nil {
		t.Fatalf("pedido por QR: %v", err)
	}

	// Un actor_tipo inválido hace fallar el insert del evento (CHECK de la tabla).
	roto := evento.ContextoConActor(ctx, evento.Actor{Tipo: "inexistente"})
	if _, err := pedidoSvc.CambiarEstado(roto, p.ID, a.TenantID, "", "preparando"); err == nil {
		t.Fatal("se esperaba error: el evento no pudo registrarse")
	}
	var estado string
	if err := db.Pool.QueryRow(ctx, `SELECT estado FROM pedidos WHERE id = $1`, p.ID).Scan(&estado); err != nil {
		t.Fatal(err)
	}
	if estado != "recibido" {
		t.Fatalf("el cambio debió revertirse: estado=%q", estado)
	}
}

func TestIntegration_FalloDelEventoRevierteLaCreacionDelPedido(t *testing.T) {
	prepararDB(t)
	ctx := context.Background()
	a := crearNegocio(t, "rbc")

	roto := evento.ContextoConActor(ctx, evento.Actor{Tipo: "inexistente"})
	input := pedido.NuevoPedidoInput{MesaID: a.MesaID, Origen: "comensal", Items: itemsQR(a.ArticuloID, "Mateo")}
	// Se usa el store: el servicio fija siempre el actor comensal.
	if _, err := pedido.NuevoStore().Crear(roto, input, a.SucursalID, 1); err == nil {
		t.Fatal("se esperaba error: el evento no pudo registrarse")
	}
	var pedidos, items int
	if err := db.Pool.QueryRow(ctx, `SELECT count(*) FROM pedidos WHERE mesa_id = $1`, a.MesaID).Scan(&pedidos); err != nil {
		t.Fatal(err)
	}
	if err := db.Pool.QueryRow(ctx, `SELECT count(*) FROM pedido_items pi JOIN pedidos p ON p.id = pi.pedido_id WHERE p.mesa_id = $1`, a.MesaID).Scan(&items); err != nil {
		t.Fatal(err)
	}
	if pedidos != 0 || items != 0 {
		t.Fatalf("la creación debió revertirse: pedidos=%d items=%d", pedidos, items)
	}
}

func TestIntegration_FalloDelEventoRevierteElCierreDeCuenta(t *testing.T) {
	prepararDB(t)
	ctx := context.Background()
	a := crearNegocio(t, "rbx")

	if _, err := pedido.NuevoService(pedido.NuevoStore()).Crear(ctx, pedido.NuevoPedidoInput{MesaID: a.MesaID, Items: itemsQR(a.ArticuloID, "Mateo")}); err != nil {
		t.Fatalf("pedido por QR: %v", err)
	}

	roto := evento.ContextoConActor(ctx, evento.Actor{Tipo: "inexistente"})
	if _, err := mesa.NuevoService(mesa.NuevoStore()).CerrarCuenta(roto, a.MesaID, a.TenantID); err == nil {
		t.Fatal("se esperaba error: el evento no pudo registrarse")
	}
	var version int
	if err := db.Pool.QueryRow(ctx, `SELECT cuenta_version FROM mesas WHERE id = $1`, a.MesaID).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 1 {
		t.Fatalf("cuenta_version debió quedar en 1: %d", version)
	}
	var abiertos int
	if err := db.Pool.QueryRow(ctx, `SELECT count(*) FROM pedidos WHERE mesa_id = $1 AND estado != 'cerrado'`, a.MesaID).Scan(&abiertos); err != nil {
		t.Fatal(err)
	}
	if abiertos != 1 {
		t.Fatalf("el pedido no debió cerrarse: abiertos=%d", abiertos)
	}
}
