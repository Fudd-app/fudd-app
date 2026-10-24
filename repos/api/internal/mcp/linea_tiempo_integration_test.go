package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/carta"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/db"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/evento"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/mesa"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/pedido"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/tenant"
)

type negocioLineaTiempo struct {
	TenantID, SucursalID, AdminID, AdminNombre, MesaID, QRToken, ArticuloID string
}

func conectarLineaTiempo(t *testing.T) {
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

// crearNegocioLineaTiempo arma un tenant con su sucursal, una mesa (número 1) y un artículo, usando los services reales.
func crearNegocioLineaTiempo(t *testing.T, etiqueta string) negocioLineaTiempo {
	t.Helper()
	ctx := context.Background()
	sufijo := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(1_000_000_000)

	creado, err := tenant.NuevoService(tenant.NuevoStore()).Crear(ctx, tenant.OnboardingInput{
		Nombre:      "Tiempo " + etiqueta,
		Slug:        fmt.Sprintf("lt-%s-%d", etiqueta, sufijo),
		EmailAdmin:  fmt.Sprintf("lt-%s-%d@test.com", etiqueta, sufijo),
		NombreAdmin: "Admin " + etiqueta,
		Password:    "MesaClick2026",
	})
	if err != nil {
		t.Fatalf("error creando tenant: %v", err)
	}
	t.Cleanup(func() {
		// pedidos referencia a la mesa con RESTRICT: se borran antes que el tenant.
		_, _ = db.Pool.Exec(ctx, `DELETE FROM pedidos WHERE sucursal_id IN (SELECT id FROM sucursales WHERE tenant_id = $1)`, creado.ID)
		_, _ = db.Pool.Exec(ctx, `DELETE FROM tenants WHERE id = $1`, creado.ID)
	})

	n := negocioLineaTiempo{TenantID: creado.ID, AdminNombre: "Admin " + etiqueta}
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

func (n negocioLineaTiempo) ctx(rol string) context.Context {
	return auth.ContextConClaims(context.Background(), &auth.Claims{UsuarioID: n.AdminID, TenantID: n.TenantID, Rol: rol})
}

func itemsLineaTiempo(articuloID string) []pedido.NuevoItemInput {
	return []pedido.NuevoItemInput{
		{ArticuloID: articuloID, Cantidad: 1, ComensalID: "47dc8c9e-fb98-44d7-80a1-b598addc1e8a", ComensalNombre: "Mateo"},
		{ArticuloID: articuloID, Cantidad: 1, ComensalID: "5ef10747-503a-4d72-a433-94978a2447e6", ComensalNombre: "Juani"},
	}
}

func TestIntegrationLineaDeTiempoDeUnaCuentaCompleta(t *testing.T) {
	conectarLineaTiempo(t)
	ctx := context.Background()
	a := crearNegocioLineaTiempo(t, "a")
	b := crearNegocioLineaTiempo(t, "b")
	pedidoSvc := pedido.NuevoService(pedido.NuevoStore())
	mesaSvc := mesa.NuevoService(mesa.NuevoStore())

	mozo := &auth.Claims{UsuarioID: a.AdminID, TenantID: a.TenantID, Rol: auth.RolMozo}
	ctxMozo := auth.ContextConClaims(ctx, mozo)
	ctxCocina := a.ctx(auth.RolCocinero)

	// Cada paso toma su marca de tiempo después de ejecutarse; la pausa asegura que los ts de Postgres no se pisen.
	var marcas = map[string]time.Time{}
	paso := func(nombre string, f func()) {
		t.Helper()
		time.Sleep(25 * time.Millisecond)
		f()
		marcas[nombre] = time.Now()
	}
	cambiar := func(id, estado, modo string) {
		t.Helper()
		if _, err := pedidoSvc.CambiarEstado(ctxMozo, mozo, id, pedido.CambiarEstadoInput{Estado: estado, ModoEntrega: modo}); err != nil {
			t.Fatalf("pedido -> %s: %v", estado, err)
		}
	}
	itemEstado := func(id, estado string) {
		t.Helper()
		if _, err := pedidoSvc.CambiarEstadoItem(ctxCocina, id, a.TenantID, "", estado); err != nil {
			t.Fatalf("ítem -> %s: %v", estado, err)
		}
	}

	var p *pedido.Pedido
	paso("recibido", func() {
		var err error
		if p, err = pedidoSvc.Crear(ctx, pedido.NuevoPedidoInput{MesaID: a.MesaID, Items: itemsLineaTiempo(a.ArticuloID)}); err != nil {
			t.Fatalf("pedido por QR: %v", err)
		}
	})
	paso("confirmado", func() { cambiar(p.ID, "confirmado", "") })
	paso("preparando", func() { itemEstado(p.Items[0].ID, "preparando") })
	paso("listo", func() {
		itemEstado(p.Items[1].ID, "listo")
		itemEstado(p.Items[0].ID, "listo")
	})
	paso("entregado", func() { cambiar(p.ID, "entregado", "retiro") })
	paso("cerrado", func() {
		if _, err := mesaSvc.CerrarCuenta(a.ctx(auth.RolDuenio), a.MesaID, a.TenantID); err != nil {
			t.Fatalf("cerrar cuenta: %v", err)
		}
	})
	// El tenant B también tiene actividad: A no debe verla.
	if _, err := pedidoSvc.Crear(ctx, pedido.NuevoPedidoInput{MesaID: b.MesaID, Items: itemsLineaTiempo(b.ArticuloID)}); err != nil {
		t.Fatalf("pedido de B: %v", err)
	}

	// Ahora lo controla el test: cada consulta se hace "desde" un instante concreto.
	var ahora time.Time
	t.Setenv("MCP_HABILITADO", "true")
	h := nuevoHandler(authDePrueba, Dependencias{
		Store: NuevoStore(), Metricas: &metricasFalsas{}, Acceso: permiteTodo(),
		Ahora: func() time.Time { return ahora },
	})

	t.Run("estado_mesas con el pedido en preparando (hace 60 min)", func(t *testing.T) {
		// Se mide 60 min después del paso y se pide "hace 60 min": el momento es justo el del paso, más 5 ms.
		ahora = marcas["preparando"].Add(5*time.Millisecond + 60*time.Minute)
		var out SalidaEstadoMesas
		salidaTool(t, llamarTool(t, h, a.TenantID, "estado_mesas", map[string]any{"momento": "hace 60 min", "mesa": 1}), &out)
		raw, _ := json.Marshal(out)
		t.Logf("estado_mesas: %s", raw)

		if len(out.Mesas) != 1 {
			t.Fatalf("mesas: %+v", out.Mesas)
		}
		m := out.Mesas[0]
		if !m.CuentaAbierta || m.CuentaVersion == nil || len(m.Pedidos) != 1 || m.CuentaSolicitada || m.PagoHabilitado {
			t.Fatalf("mesa: %+v", m)
		}
		ped := m.Pedidos[0]
		if ped.PedidoID != p.ID || ped.Estado != "preparando" || ped.QuienUltimoCambio != a.AdminNombre || ped.Rol == nil || *ped.Rol != auth.RolCocinero {
			t.Errorf("pedido: %+v", ped)
		}
		if m.Espera == nil || *m.Espera != EsperaComida {
			t.Errorf("espera: %+v", m.Espera)
		}
	})

	t.Run("estado_mesas en otros momentos", func(t *testing.T) {
		consulta := func(paso string) MesaEnMomento {
			t.Helper()
			ahora = marcas[paso].Add(5 * time.Millisecond)
			var out SalidaEstadoMesas
			salidaTool(t, llamarTool(t, h, a.TenantID, "estado_mesas", map[string]any{"mesa": 1}), &out)
			return out.Mesas[0]
		}
		if m := consulta("recibido"); !m.CuentaAbierta || m.Pedidos[0].Estado != "recibido" || m.Pedidos[0].QuienUltimoCambio != "Mateo" || m.Pedidos[0].Rol != nil {
			t.Errorf("recibido: %+v", m)
		}
		if m := consulta("listo"); m.Pedidos[0].Estado != "listo" || *m.Espera != EsperaEntrega {
			t.Errorf("listo: %+v", m)
		}
		m := consulta("entregado")
		if m.Pedidos[0].Estado != "entregado" || m.Pedidos[0].Rol == nil || *m.Pedidos[0].Rol != auth.RolMozo || m.Espera != nil {
			t.Errorf("entregado: %+v", m)
		}
		if m := consulta("cerrado"); m.CuentaAbierta || len(m.Pedidos) != 0 {
			t.Errorf("cerrado: %+v", m)
		}
	})

	t.Run("antes del primer hito responde sin_datos", func(t *testing.T) {
		ahora = marcas["recibido"].Add(5 * time.Minute)
		e := errorEstructurado(t, llamarTool(t, h, a.TenantID, "estado_mesas", map[string]any{"momento": "hace 2 horas"}))
		detalle, _ := e.Detalle.(map[string]any)
		if e.Error != CodigoSinDatos || detalle["historial_desde"] == nil {
			t.Errorf("%+v", e)
		}
	})

	t.Run("historial_mesa completo", func(t *testing.T) {
		ahora = time.Now()
		var out SalidaHistorialMesa
		salidaTool(t, llamarTool(t, h, a.TenantID, "historial_mesa", map[string]any{"mesa": 1}), &out)
		raw, _ := json.Marshal(out)
		t.Logf("historial_mesa: %s", raw)

		// Secuencia de hitos: apertura, pedido, confirmación, ítems/preparación, listo, entrega y cierre.
		var secuencia []string
		for _, hito := range out.Hitos {
			etiqueta := hito.Hito
			if hito.EstadoNuevo != nil && hito.Hito != evento.TipoItemEstado {
				etiqueta += ":" + *hito.EstadoNuevo
			}
			if hito.Hito != evento.TipoItemEstado {
				secuencia = append(secuencia, etiqueta)
			}
		}
		esperada := []string{"cuenta_abierta:recibido", "pedido_creado:recibido", "pedido_estado:confirmado", "pedido_estado:preparando",
			"pedido_estado:listo", "pedido_estado:entregado", "cuenta_cerrada"}
		if fmt.Sprint(secuencia) != fmt.Sprint(esperada) {
			t.Fatalf("hitos %v, quería %v", secuencia, esperada)
		}
		if out.Hitos[0].Quien != "Mateo" || out.Hitos[0].MinutosDesdeAnterior != nil {
			t.Errorf("primer hito: %+v", out.Hitos[0])
		}
		for i, hito := range out.Hitos {
			if i > 0 && (hito.MinutosDesdeAnterior == nil || *hito.MinutosDesdeAnterior < 0) {
				t.Errorf("minutos_desde_anterior del hito %d: %+v", i, hito.MinutosDesdeAnterior)
			}
			if hito.Hito == evento.TipoPedidoEstado && hito.EstadoNuevo != nil && *hito.EstadoNuevo == "entregado" {
				if hito.ModoEntrega == nil || *hito.ModoEntrega != "retiro" || hito.Quien != a.AdminNombre || hito.Rol == nil || *hito.Rol != auth.RolMozo {
					t.Errorf("hito entregado: %+v", hito)
				}
			}
		}
		if len(out.Etapas) != 1 {
			t.Fatalf("etapas: %+v", out.Etapas)
		}
		e := out.Etapas[0]
		if e.PedidoID != p.ID || e.MinutosHastaConfirmacion == nil || e.MinutosEsperaCocina == nil || e.MinutosPreparacion == nil ||
			e.MinutosAEntrega == nil || e.ModoEntrega == nil || *e.ModoEntrega != "retiro" {
			t.Errorf("etapas: %+v", e)
		}
		// Los tiempos reales entre pasos son de ~25 ms: la suma de etapas ronda cero minutos y nunca es negativa.
		for _, v := range []*float64{e.MinutosHastaConfirmacion, e.MinutosEsperaCocina, e.MinutosPreparacion, e.MinutosAEntrega} {
			if *v < 0 || *v > 1 {
				t.Errorf("etapa fuera de rango: %v", *v)
			}
		}
	})

	t.Run("historial_mesa de una mesa inexistente", func(t *testing.T) {
		ahora = time.Now()
		if e := errorEstructurado(t, llamarTool(t, h, a.TenantID, "historial_mesa", map[string]any{"mesa": 42})); e.Error != CodigoMesaNoEncontrada {
			t.Errorf("%+v", e)
		}
	})

	t.Run("aislamiento entre tenants", func(t *testing.T) {
		ahora = time.Now()
		// B ve su propia actividad y no la de A.
		var propia SalidaEstadoMesas
		salidaTool(t, llamarTool(t, h, b.TenantID, "estado_mesas", nil), &propia)
		if len(propia.Mesas) != 1 || !propia.Mesas[0].CuentaAbierta || propia.Mesas[0].Pedidos[0].PedidoID == p.ID {
			t.Errorf("B: %+v", propia.Mesas)
		}
		// A no puede consultar la sucursal de B ni leer eventos de su mesa.
		for _, tool := range []string{"estado_mesas", "historial_mesa"} {
			e := errorEstructurado(t, llamarTool(t, h, a.TenantID, tool, map[string]any{"sucursal_id": b.SucursalID, "mesa": 1}))
			if e.Error != CodigoSucursalNoEncontrada {
				t.Errorf("%s con sucursal ajena: %+v", tool, e)
			}
		}
		st := NuevoStore()
		if ev, err := st.EventosMesa(ctx, a.TenantID, a.SucursalID, b.MesaID, time.Time{}, time.Time{}); err != nil || len(ev) != 0 {
			t.Errorf("A leyó eventos de la mesa de B: %d %v", len(ev), err)
		}
		if ev, err := st.EventosSucursal(ctx, a.TenantID, b.SucursalID, time.Time{}, time.Time{}); err != nil || len(ev) != 0 {
			t.Errorf("A leyó eventos de la sucursal de B: %d %v", len(ev), err)
		}
		if primero, err := st.PrimerEvento(ctx, a.TenantID, b.SucursalID); err != nil || primero != nil {
			t.Errorf("A vio el primer evento de B: %v %v", primero, err)
		}
		if m, err := st.MesaPorNumero(ctx, a.TenantID, b.SucursalID, 1); err == nil {
			t.Errorf("A encontró la mesa de B: %+v", m)
		}
	})
}
