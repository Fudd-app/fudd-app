package evento_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/db"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/evento"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/mesa"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/pedido"
)

func TestIntegration_FlujoPedidoConfirmadoYEntregado(t *testing.T) {
	prepararDB(t)
	ctx := context.Background()
	a := crearNegocio(t, "flujo")
	pedidoSvc := pedido.NuevoService(pedido.NuevoStore())

	mozo := &auth.Claims{UsuarioID: a.AdminID, TenantID: a.TenantID, Rol: auth.RolMozo}
	ctxMozo := auth.ContextConClaims(ctx, mozo)
	cocina := &auth.Claims{UsuarioID: a.AdminID, TenantID: a.TenantID, Rol: auth.RolCocinero}
	ctxCocina := auth.ContextConClaims(ctx, cocina)

	estadoEnDB := func(id string) string {
		t.Helper()
		var estado string
		if err := db.Pool.QueryRow(ctx, `SELECT estado FROM pedidos WHERE id = $1`, id).Scan(&estado); err != nil {
			t.Fatal(err)
		}
		return estado
	}

	p, err := pedidoSvc.Crear(ctx, pedido.NuevoPedidoInput{MesaID: a.MesaID, Items: itemsQR(a.ArticuloID, "Mateo", "Juani")})
	if err != nil {
		t.Fatalf("pedido por QR: %v", err)
	}
	if p.Estado != "recibido" {
		t.Fatalf("estado inicial: %q", p.Estado)
	}

	// Cocina no puede tocar un pedido sin confirmar.
	if _, err := pedidoSvc.CambiarEstadoItem(ctxCocina, p.Items[0].ID, a.TenantID, "", "preparando"); !errors.Is(err, pedido.ErrPedidoSinConfirmar) {
		t.Fatalf("KDS sobre recibido: got %v, want ErrPedidoSinConfirmar", err)
	}
	// Ni saltear la confirmación por el endpoint de pedido.
	if _, err := pedidoSvc.CambiarEstado(ctxMozo, mozo, p.ID, pedido.CambiarEstadoInput{Estado: "preparando"}); !errors.Is(err, pedido.ErrPedidoSinConfirmar) {
		t.Fatalf("recibido -> preparando: got %v", err)
	}
	// Cocina tampoco confirma.
	if _, err := pedidoSvc.CambiarEstado(ctxCocina, cocina, p.ID, pedido.CambiarEstadoInput{Estado: "confirmado"}); !errors.Is(err, pedido.ErrSinPermiso) {
		t.Fatalf("cocina confirmando: got %v", err)
	}
	if got := estadoEnDB(p.ID); got != "recibido" {
		t.Fatalf("los intentos rechazados no deben cambiar el estado: %q", got)
	}

	if _, err := pedidoSvc.CambiarEstado(ctxMozo, mozo, p.ID, pedido.CambiarEstadoInput{Estado: "confirmado"}); err != nil {
		t.Fatalf("confirmar: %v", err)
	}
	// Entregar antes de estar listo es inválido.
	if _, err := pedidoSvc.CambiarEstado(ctxMozo, mozo, p.ID, pedido.CambiarEstadoInput{Estado: "entregado", ModoEntrega: "mesa"}); !errors.Is(err, pedido.ErrTransicionInvalida) {
		t.Fatalf("confirmado -> entregado: got %v", err)
	}

	pr, err := pedidoSvc.CambiarEstadoItem(ctxCocina, p.Items[0].ID, a.TenantID, "", "preparando")
	if err != nil || pr.Estado != "preparando" {
		t.Fatalf("primer ítem: estado=%v err=%v", pr, err)
	}
	for _, it := range p.Items {
		if it.ID != p.Items[0].ID {
			if _, err := pedidoSvc.CambiarEstadoItem(ctxCocina, it.ID, a.TenantID, "", "listo"); err != nil {
				t.Fatalf("ítem %s: %v", it.ID, err)
			}
		}
	}
	pr, err = pedidoSvc.CambiarEstadoItem(ctxCocina, p.Items[0].ID, a.TenantID, "", "listo")
	if err != nil || pr.Estado != "listo" {
		t.Fatalf("todos listos: estado=%v err=%v", pr, err)
	}

	// Entrega por retiro: requiere modo_entrega.
	if _, err := pedidoSvc.CambiarEstado(ctxMozo, mozo, p.ID, pedido.CambiarEstadoInput{Estado: "entregado"}); !errors.Is(err, pedido.ErrModoEntregaInvalido) {
		t.Fatalf("entregado sin modo: got %v", err)
	}
	if _, err := pedidoSvc.CambiarEstado(ctxMozo, mozo, p.ID, pedido.CambiarEstadoInput{Estado: "entregado", ModoEntrega: "retiro"}); err != nil {
		t.Fatalf("entregar: %v", err)
	}
	if got := estadoEnDB(p.ID); got != "entregado" {
		t.Fatalf("estado tras entregar: %q", got)
	}
	// Un pedido entregado ya no vuelve a cocina ni retrocede.
	if _, err := pedidoSvc.CambiarEstadoItem(ctxCocina, p.Items[0].ID, a.TenantID, "", "preparando"); !errors.Is(err, pedido.ErrPedidoEntregado) {
		t.Fatalf("KDS sobre entregado: got %v", err)
	}
	if _, err := pedidoSvc.CambiarEstado(ctxMozo, mozo, p.ID, pedido.CambiarEstadoInput{Estado: "listo"}); !errors.Is(err, pedido.ErrTransicionInvalida) {
		t.Fatalf("entregado -> listo: got %v", err)
	}

	// El pedido entregado sigue en la cuenta actual hasta cerrarla.
	cuenta, err := pedidoSvc.ListarCuentaActualPorQR(ctx, a.QRToken)
	if err != nil || len(cuenta) != 1 || cuenta[0].Estado != "entregado" {
		t.Fatalf("cuenta actual: %+v err=%v", cuenta, err)
	}

	if _, err := mesa.NuevoService(mesa.NuevoStore()).CerrarCuenta(a.ctxAdmin(), a.MesaID, a.TenantID); err != nil {
		t.Fatalf("cerrar cuenta: %v", err)
	}
	if got := estadoEnDB(p.ID); got != "cerrado" {
		t.Fatalf("estado tras cerrar la cuenta: %q", got)
	}

	eventos, err := evento.EventosPorMesa(ctx, a.TenantID, a.SucursalID, a.MesaID, time.Time{}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	var transiciones [][2]string
	for _, e := range eventos {
		if e.Tipo != evento.TipoPedidoEstado {
			continue
		}
		transiciones = append(transiciones, [2]string{e.EstadoAnterior, e.EstadoNuevo})
		if e.ActorRol != auth.RolMozo && e.ActorRol != auth.RolCocinero || e.UsuarioID == nil || *e.UsuarioID != a.AdminID {
			t.Errorf("hito %s->%s sin actor: %+v", e.EstadoAnterior, e.EstadoNuevo, e)
		}
		var detalle map[string]any
		if err := json.Unmarshal(e.Detalle, &detalle); err != nil {
			t.Fatal(err)
		}
		if e.EstadoNuevo == "entregado" {
			if detalle["modo_entrega"] != "retiro" || e.ActorRol != auth.RolMozo {
				t.Errorf("hito entregado: detalle=%v rol=%q", detalle, e.ActorRol)
			}
		} else if _, ok := detalle["modo_entrega"]; ok {
			t.Errorf("modo_entrega solo va en entregado: %v", detalle)
		}
	}
	esperadas := [][2]string{
		{"recibido", "confirmado"}, {"confirmado", "preparando"}, {"preparando", "listo"}, {"listo", "entregado"},
	}
	if len(transiciones) != len(esperadas) {
		t.Fatalf("hitos de pedido: got %v, want %v", transiciones, esperadas)
	}
	for i := range esperadas {
		if transiciones[i] != esperadas[i] {
			t.Errorf("hito %d: got %v, want %v", i, transiciones[i], esperadas[i])
		}
	}
}

func TestIntegration_ElPedidoNoRetrocedeAlDeshacerUnItem(t *testing.T) {
	prepararDB(t)
	ctx := context.Background()
	a := crearNegocio(t, "noretro")
	pedidoSvc := pedido.NuevoService(pedido.NuevoStore())

	mozo := &auth.Claims{UsuarioID: a.AdminID, TenantID: a.TenantID, Rol: auth.RolMozo}
	ctxMozo := auth.ContextConClaims(ctx, mozo)
	cocina := &auth.Claims{UsuarioID: a.AdminID, TenantID: a.TenantID, Rol: auth.RolCocinero}
	ctxCocina := auth.ContextConClaims(ctx, cocina)

	p, err := pedidoSvc.Crear(ctx, pedido.NuevoPedidoInput{MesaID: a.MesaID, Items: itemsQR(a.ArticuloID, "Mateo", "Juani")})
	if err != nil {
		t.Fatalf("pedido por QR: %v", err)
	}
	for _, estado := range []string{"confirmado", "preparando", "listo"} {
		if _, err := pedidoSvc.CambiarEstado(ctxMozo, mozo, p.ID, pedido.CambiarEstadoInput{Estado: estado}); err != nil {
			t.Fatalf("mozo -> %s: %v", estado, err)
		}
	}

	leer := func() (estado string, listoAt *time.Time, hitos int) {
		t.Helper()
		if err := db.Pool.QueryRow(ctx, `SELECT estado, listo_at FROM pedidos WHERE id = $1`, p.ID).Scan(&estado, &listoAt); err != nil {
			t.Fatal(err)
		}
		if err := db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM eventos_operacion WHERE pedido_id = $1 AND tipo = 'pedido_estado'`, p.ID).Scan(&hitos); err != nil {
			t.Fatal(err)
		}
		return
	}
	_, listoAntes, hitosAntes := leer()

	// Ítems pendientes con el pedido "listo" (lo marcó el mozo), y luego ítem listo -> preparando -> pendiente.
	for _, estadoItem := range []string{"pendiente", "listo", "preparando", "pendiente"} {
		pr, err := pedidoSvc.CambiarEstadoItem(ctxCocina, p.Items[0].ID, a.TenantID, "", estadoItem)
		if err != nil {
			t.Fatalf("ítem -> %s: %v", estadoItem, err)
		}
		estado, listoAt, hitos := leer()
		if pr.Estado != "listo" || estado != "listo" {
			t.Fatalf("ítem -> %s: el pedido retrocedió (respuesta=%q db=%q)", estadoItem, pr.Estado, estado)
		}
		if listoAt == nil || listoAntes == nil || !listoAt.Equal(*listoAntes) {
			t.Fatalf("ítem -> %s: listo_at cambió: %v -> %v", estadoItem, listoAntes, listoAt)
		}
		if hitos != hitosAntes {
			t.Fatalf("ítem -> %s: no debe registrarse hito de pedido (%d -> %d)", estadoItem, hitosAntes, hitos)
		}
	}
	var hitosItem int
	if err := db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM eventos_operacion WHERE pedido_id = $1 AND tipo = 'item_estado'`, p.ID).Scan(&hitosItem); err != nil || hitosItem != 3 {
		t.Fatalf("hitos de ítem: got %d err=%v, want 3", hitosItem, err)
	}
}

func TestIntegration_PedidoAsistidoNaceConfirmado(t *testing.T) {
	prepararDB(t)
	ctx := context.Background()
	a := crearNegocio(t, "asistido")
	pedidoSvc := pedido.NuevoService(pedido.NuevoStore())

	mozo := &auth.Claims{UsuarioID: a.AdminID, TenantID: a.TenantID, Rol: auth.RolMozo, SucursalID: a.SucursalID}
	p, err := pedidoSvc.CrearAsistido(auth.ContextConClaims(ctx, mozo), mozo, pedido.PedidoAsistidoInput{
		MesaID: a.MesaID, AliasCliente: "Mesa 1",
		Items: []pedido.NuevoItemInput{{ArticuloID: a.ArticuloID, Cantidad: 1}},
	})
	if err != nil {
		t.Fatalf("pedido asistido: %v", err)
	}
	if p.Estado != "confirmado" {
		t.Fatalf("estado inicial: %q", p.Estado)
	}
	var estado string
	if err := db.Pool.QueryRow(ctx, `SELECT estado FROM pedidos WHERE id = $1`, p.ID).Scan(&estado); err != nil || estado != "confirmado" {
		t.Fatalf("estado en DB: %q err=%v", estado, err)
	}

	eventos, err := evento.EventosPorMesa(ctx, a.TenantID, a.SucursalID, a.MesaID, time.Time{}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	var vistos []string
	for _, e := range eventos {
		if e.Tipo != evento.TipoPedidoCreado && e.Tipo != evento.TipoPedidoEstado {
			continue
		}
		vistos = append(vistos, e.Tipo+":"+e.EstadoAnterior+">"+e.EstadoNuevo)
		if e.ActorRol != auth.RolMozo || e.UsuarioID == nil || *e.UsuarioID != a.AdminID {
			t.Errorf("hito %s sin el mozo como actor: %+v", e.Tipo, e)
		}
	}
	esperado := []string{"pedido_creado:>recibido", "pedido_estado:recibido>confirmado"}
	if len(vistos) != 2 || vistos[0] != esperado[0] || vistos[1] != esperado[1] {
		t.Fatalf("hitos: got %v, want %v", vistos, esperado)
	}
}
