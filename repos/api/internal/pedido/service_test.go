package pedido_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/evento"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/notificacion"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/pedido"
)

type mockStore struct {
	crearInput             pedido.NuevoPedidoInput
	crearCtx               context.Context
	estadoItemRecibido     string
	pedidoTrasCambioDeItem *pedido.Pedido
	sucursalPertenece      bool
	// estadoActual simula el estado persistido del pedido (por defecto "recibido").
	estadoActual    string
	estadoPedido    string
	modoEntrega     string
	cambiosDeEstado int
	errItem         error
}

func (m *mockStore) Crear(ctx context.Context, input pedido.NuevoPedidoInput, sucursalID string, cuentaVersion int) (*pedido.Pedido, error) {
	m.crearInput = input
	m.crearCtx = ctx
	estado := input.EstadoInicial
	if estado == "" {
		estado = "recibido"
	}
	return &pedido.Pedido{ID: "p-1", MesaID: input.MesaID, SucursalID: sucursalID, CuentaVersion: cuentaVersion, Estado: estado}, nil
}
func (m *mockStore) ListarActivos(ctx context.Context, sucursalID, tenantID string) ([]pedido.Pedido, error) {
	return []pedido.Pedido{}, nil
}
func (m *mockStore) ListarCuentaActualPorQR(ctx context.Context, qrToken string) ([]pedido.Pedido, error) {
	return []pedido.Pedido{}, nil
}
func (m *mockStore) CambiarEstado(ctx context.Context, id, tenantID, sucursalAlcance, nuevoEstado, modoEntrega string) (*pedido.Pedido, error) {
	anterior := m.estadoActual
	if anterior == "" {
		anterior = "recibido"
	}
	if err := pedido.ValidarTransicion(anterior, nuevoEstado); err != nil {
		return nil, err
	}
	m.cambiosDeEstado++
	m.estadoPedido, m.modoEntrega = nuevoEstado, modoEntrega
	return &pedido.Pedido{ID: id, MesaID: "mesa-1", SucursalID: "suc-1", Estado: nuevoEstado}, nil
}
func (m *mockStore) CambiarEstadoItem(ctx context.Context, itemID, tenantID, sucursalAlcance, nuevoEstado string) (*pedido.Pedido, error) {
	if m.errItem != nil {
		return nil, m.errItem
	}
	m.estadoItemRecibido = nuevoEstado
	if m.pedidoTrasCambioDeItem != nil {
		return m.pedidoTrasCambioDeItem, nil
	}
	return &pedido.Pedido{
		ID:         "p-1",
		MesaID:     "mesa-1",
		SucursalID: "suc-1",
		Estado:     "preparando",
		Items: []pedido.PedidoItem{{
			ID:     itemID,
			Estado: nuevoEstado,
		}},
	}, nil
}
func (m *mockStore) SucursalPerteneceATenant(ctx context.Context, sucursalID, tenantID string) (bool, error) {
	return m.sucursalPertenece, nil
}
func (m *mockStore) ObtenerSucursalPorMesa(ctx context.Context, mesaID string) (string, int, error) {
	return "suc-1", 3, nil
}

func TestCrear_AsignaComensalAlItem(t *testing.T) {
	store := &mockStore{}
	svc := pedido.NuevoService(store)

	p, err := svc.Crear(context.Background(), pedido.NuevoPedidoInput{
		MesaID: "mesa-1",
		Items: []pedido.NuevoItemInput{{
			ArticuloID:     "articulo-1",
			Cantidad:       1,
			ComensalID:     "47dc8c9e-fb98-44d7-80a1-b598addc1e8a",
			ComensalNombre: "  Mateo  ",
		}},
	})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if p.CuentaVersion != 3 {
		t.Fatalf("cuenta_version incorrecta: got %d, want 3", p.CuentaVersion)
	}
	if got := store.crearInput.Items[0].ComensalNombre; got != "Mateo" {
		t.Errorf("nombre no normalizado: got %q, want %q", got, "Mateo")
	}
}

func TestCrear_ComensalInvalido_Error(t *testing.T) {
	svc := pedido.NuevoService(&mockStore{})
	_, err := svc.Crear(context.Background(), pedido.NuevoPedidoInput{
		MesaID: "mesa-1",
		Items: []pedido.NuevoItemInput{{
			ArticuloID:     "articulo-1",
			Cantidad:       1,
			ComensalID:     "no-es-uuid",
			ComensalNombre: "Mateo",
		}},
	})
	if err == nil {
		t.Fatal("esperaba error por comensal_id inválido")
	}
}

func TestCrear_SinAlias_ConservaIdentidadAnonima(t *testing.T) {
	store := &mockStore{}
	svc := pedido.NuevoService(store)

	_, err := svc.Crear(context.Background(), pedido.NuevoPedidoInput{
		MesaID: "mesa-1",
		Items: []pedido.NuevoItemInput{{
			ArticuloID:     "articulo-1",
			Cantidad:       1,
			ComensalID:     "47dc8c9e-fb98-44d7-80a1-b598addc1e8a",
			ComensalNombre: "",
		}},
	})
	if err != nil {
		t.Fatalf("un comensal sin alias debe poder pedir: %v", err)
	}
	if got := store.crearInput.Items[0].ComensalID; got == "" {
		t.Fatal("el pedido sin alias debe conservar el UUID del comensal")
	}
	if got := store.crearInput.Items[0].ComensalNombre; got != "" {
		t.Fatalf("el alias vacío debe conservarse vacío: got %q", got)
	}
}

func TestCrear_DosComensalesMismoNombre_ConservaIDsDistintos(t *testing.T) {
	store := &mockStore{}
	svc := pedido.NuevoService(store)

	_, err := svc.Crear(context.Background(), pedido.NuevoPedidoInput{
		MesaID: "mesa-1",
		Items: []pedido.NuevoItemInput{
			{
				ArticuloID:     "articulo-1",
				Cantidad:       1,
				ComensalID:     "47dc8c9e-fb98-44d7-80a1-b598addc1e8a",
				ComensalNombre: "Mateo",
			},
			{
				ArticuloID:     "articulo-2",
				Cantidad:       1,
				ComensalID:     "5ef10747-503a-4d72-a433-94978a2447e6",
				ComensalNombre: "Mateo",
			},
		},
	})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	primero := store.crearInput.Items[0]
	segundo := store.crearInput.Items[1]
	if primero.ComensalNombre != segundo.ComensalNombre {
		t.Fatal("el escenario requiere dos nombres iguales")
	}
	if primero.ComensalID == segundo.ComensalID {
		t.Fatal("dos comensales homónimos no deben compartir identidad")
	}
}

func actorRol(rol string) *auth.Claims {
	return &auth.Claims{UsuarioID: "u-1", TenantID: "tenant-1", Rol: rol, SucursalID: "suc-1"}
}

// modoPara devuelve el modo_entrega que corresponde enviar para el estado destino.
func modoPara(estado string) string {
	if estado == "entregado" {
		return "mesa"
	}
	return ""
}

func TestCambiarEstado_EstadoInvalido(t *testing.T) {
	svc := pedido.NuevoService(&mockStore{})
	_, err := svc.CambiarEstado(context.Background(), actorRol(auth.RolDuenio), "p-1", pedido.CambiarEstadoInput{Estado: "invalido"})
	if !errors.Is(err, pedido.ErrValidation) {
		t.Fatalf("esperaba ErrValidation por estado inválido, got %v", err)
	}
}

func TestCambiarEstado_EstadosValidos(t *testing.T) {
	for _, estado := range pedido.EstadosValidos {
		if estado == "recibido" {
			continue // no se puede volver a recibido
		}
		svc := pedido.NuevoService(&mockStore{estadoActual: "listo"})
		_, err := svc.CambiarEstado(context.Background(), actorRol(auth.RolDuenio), "p-1",
			pedido.CambiarEstadoInput{Estado: estado, ModoEntrega: modoPara(estado)})
		if estado == "listo" {
			// Repetir el estado actual es un no-op válido.
			if err != nil {
				t.Errorf("repetir %q debería aceptarse: %v", estado, err)
			}
			continue
		}
		if estado == "confirmado" || estado == "preparando" {
			if !errors.Is(err, pedido.ErrTransicionInvalida) {
				t.Errorf("retroceder de listo a %q debería ser inválido, got %v", estado, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("estado %q debería ser válido pero dio error: %v", estado, err)
		}
	}
}

func TestValidarTransicion_Matriz(t *testing.T) {
	flujo := []string{"recibido", "confirmado", "preparando", "listo", "entregado", "cerrado"}
	for i, desde := range flujo {
		for j, hasta := range flujo {
			err := pedido.ValidarTransicion(desde, hasta)
			var esperado error
			switch {
			case i == j:
				esperado = nil
			case desde == "cerrado" || hasta == "recibido":
				esperado = pedido.ErrTransicionInvalida
			case hasta == "cerrado":
				esperado = nil
			case j < i:
				esperado = pedido.ErrTransicionInvalida
			case hasta == "entregado" && desde != "listo":
				esperado = pedido.ErrTransicionInvalida
			case desde == "recibido" && (hasta == "preparando" || hasta == "listo"):
				esperado = pedido.ErrPedidoSinConfirmar
			}
			if esperado == nil && err != nil || esperado != nil && !errors.Is(err, esperado) {
				t.Errorf("%s -> %s: got %v, want %v", desde, hasta, err, esperado)
			}
		}
	}
}

func TestCambiarEstado_PermisosPorRol(t *testing.T) {
	tests := []struct {
		rol, desde, hasta string
		permitido         bool
	}{
		{auth.RolMozo, "recibido", "confirmado", true},
		{auth.RolEncargado, "recibido", "confirmado", true},
		{auth.RolDuenio, "recibido", "confirmado", true},
		{auth.RolCocinero, "recibido", "confirmado", false},
		{auth.RolCocinero, "confirmado", "preparando", true},
		{auth.RolCocinero, "preparando", "listo", true},
		{auth.RolEncargado, "confirmado", "preparando", true},
		{auth.RolDuenio, "preparando", "listo", true},
		// Compatibilidad: el mozo ya podía setear preparando y listo.
		{auth.RolMozo, "confirmado", "preparando", true},
		{auth.RolMozo, "preparando", "listo", true},
		{auth.RolMozo, "listo", "entregado", true},
		{auth.RolEncargado, "listo", "entregado", true},
		{auth.RolDuenio, "listo", "entregado", true},
		{auth.RolCocinero, "listo", "entregado", false},
		{auth.RolCocinero, "listo", "cerrado", false},
		{auth.RolMozo, "listo", "cerrado", true},
		{"", "recibido", "confirmado", false},
	}
	for _, tt := range tests {
		t.Run(tt.rol+"/"+tt.desde+"->"+tt.hasta, func(t *testing.T) {
			store := &mockStore{estadoActual: tt.desde}
			svc := pedido.NuevoService(store)
			_, err := svc.CambiarEstado(context.Background(), actorRol(tt.rol), "p-1",
				pedido.CambiarEstadoInput{Estado: tt.hasta, ModoEntrega: modoPara(tt.hasta)})
			if tt.permitido && err != nil {
				t.Fatalf("debería permitirse: %v", err)
			}
			if !tt.permitido {
				if !errors.Is(err, pedido.ErrSinPermiso) {
					t.Fatalf("esperaba ErrSinPermiso, got %v", err)
				}
				if store.cambiosDeEstado != 0 {
					t.Fatal("un rol sin permiso no debe llegar a persistencia")
				}
			}
		})
	}
}

func TestCambiarEstado_SinActorEsRechazado(t *testing.T) {
	svc := pedido.NuevoService(&mockStore{})
	_, err := svc.CambiarEstado(context.Background(), nil, "p-1", pedido.CambiarEstadoInput{Estado: "confirmado"})
	if !errors.Is(err, pedido.ErrSinPermiso) {
		t.Fatalf("esperaba ErrSinPermiso, got %v", err)
	}
}

func TestCambiarEstado_EntregadoExigeModoEntrega(t *testing.T) {
	for _, modo := range []string{"", "  ", "domicilio", "MESA"} {
		store := &mockStore{estadoActual: "listo"}
		svc := pedido.NuevoService(store)
		_, err := svc.CambiarEstado(context.Background(), actorRol(auth.RolMozo), "p-1",
			pedido.CambiarEstadoInput{Estado: "entregado", ModoEntrega: modo})
		if !errors.Is(err, pedido.ErrModoEntregaInvalido) {
			t.Errorf("modo %q: esperaba ErrModoEntregaInvalido, got %v", modo, err)
		}
		if store.cambiosDeEstado != 0 {
			t.Errorf("modo %q: no debe persistirse", modo)
		}
	}
	for _, modo := range []string{"mesa", "retiro"} {
		store := &mockStore{estadoActual: "listo"}
		svc := pedido.NuevoService(store)
		p, err := svc.CambiarEstado(context.Background(), actorRol(auth.RolMozo), "p-1",
			pedido.CambiarEstadoInput{Estado: "entregado", ModoEntrega: modo})
		if err != nil || p.Estado != "entregado" || store.modoEntrega != modo {
			t.Errorf("modo %q: err=%v estado=%v modo persistido=%q", modo, err, p, store.modoEntrega)
		}
	}
}

func TestCambiarEstado_EntregadoSoloDesdeListo(t *testing.T) {
	for _, desde := range []string{"recibido", "confirmado", "preparando"} {
		svc := pedido.NuevoService(&mockStore{estadoActual: desde})
		_, err := svc.CambiarEstado(context.Background(), actorRol(auth.RolDuenio), "p-1",
			pedido.CambiarEstadoInput{Estado: "entregado", ModoEntrega: "mesa"})
		if !errors.Is(err, pedido.ErrTransicionInvalida) {
			t.Errorf("%s -> entregado: esperaba ErrTransicionInvalida, got %v", desde, err)
		}
	}
}

func TestCambiarEstado_PublicaSSEParaConfirmadoYEntregado(t *testing.T) {
	casos := []struct{ desde, hasta string }{{"recibido", "confirmado"}, {"listo", "entregado"}}
	for _, c := range casos {
		svc := pedido.NuevoService(&mockStore{estadoActual: c.desde})
		var canales []<-chan notificacion.Evento
		for _, canal := range []string{"pedido:p-1", "sucursal:suc-1", "mesa:mesa-1", "kds:suc-1"} {
			ch, desuscribir := notificacion.Instancia.Suscribir(canal)
			defer desuscribir()
			canales = append(canales, ch)
		}
		if _, err := svc.CambiarEstado(context.Background(), actorRol(auth.RolMozo), "p-1",
			pedido.CambiarEstadoInput{Estado: c.hasta, ModoEntrega: modoPara(c.hasta)}); err != nil {
			t.Fatalf("%s: %v", c.hasta, err)
		}
		for i, ch := range canales {
			select {
			case ev := <-ch:
				if ev.Nombre != "pedido_actualizado" || !strings.Contains(string(ev.Data), `"estado":"`+c.hasta+`"`) {
					t.Errorf("%s canal %d: evento incorrecto %q %s", c.hasta, i, ev.Nombre, ev.Data)
				}
			case <-time.After(time.Second):
				t.Errorf("%s: canal %d sin evento SSE", c.hasta, i)
			}
		}
	}
}

func TestCambiarEstado_ModoEntregaSoloEnEntregado(t *testing.T) {
	store := &mockStore{estadoActual: "recibido"}
	svc := pedido.NuevoService(store)
	_, err := svc.CambiarEstado(context.Background(), actorRol(auth.RolMozo), "p-1",
		pedido.CambiarEstadoInput{Estado: "confirmado", ModoEntrega: "mesa"})
	if !errors.Is(err, pedido.ErrModoEntregaInvalido) {
		t.Fatalf("esperaba ErrModoEntregaInvalido, got %v", err)
	}
	if store.cambiosDeEstado != 0 {
		t.Fatal("no debe persistirse")
	}
}

func recibeEventoKDS(canal <-chan notificacion.Evento) bool {
	select {
	case <-canal:
		return true
	case <-time.After(100 * time.Millisecond):
		return false
	}
}

func TestCrear_PorQRNoPublicaEnKDS(t *testing.T) {
	svc := pedido.NuevoService(&mockStore{})
	kds, desKDS := notificacion.Instancia.Suscribir("kds:suc-1")
	defer desKDS()
	sucursal, desSuc := notificacion.Instancia.Suscribir("sucursal:suc-1")
	defer desSuc()

	_, err := svc.Crear(context.Background(), pedido.NuevoPedidoInput{
		MesaID: "mesa-1",
		Items:  []pedido.NuevoItemInput{{ArticuloID: "a-1", Cantidad: 1, ComensalID: "47dc8c9e-fb98-44d7-80a1-b598addc1e8a"}},
	})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if !recibeEventoKDS(sucursal) {
		t.Error("el salón debe enterarse del pedido nuevo")
	}
	if recibeEventoKDS(kds) {
		t.Error("el KDS no debe recibir pedidos sin confirmar")
	}
}

func TestCrearAsistido_NaceConfirmadoYPublicaEnKDS(t *testing.T) {
	store := &mockStore{}
	svc := pedido.NuevoService(store)
	kds, desKDS := notificacion.Instancia.Suscribir("kds:suc-1")
	defer desKDS()

	p, err := svc.CrearAsistido(context.Background(), &auth.Claims{
		UsuarioID: "usuario-1", TenantID: "tenant-1", Rol: auth.RolMozo, SucursalID: "suc-1",
	}, pedido.PedidoAsistidoInput{
		MesaID: "mesa-1",
		Items:  []pedido.NuevoItemInput{{ArticuloID: "art-1", Cantidad: 1}},
	})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if store.crearInput.EstadoInicial != "confirmado" || p.Estado != "confirmado" {
		t.Fatalf("el pedido del mozo debe nacer confirmado: input=%q pedido=%q", store.crearInput.EstadoInicial, p.Estado)
	}
	if !recibeEventoKDS(kds) {
		t.Error("un pedido confirmado debe llegar al KDS")
	}
}

func TestCambiarEstadoItem_PedidoSinConfirmarPropagaError(t *testing.T) {
	svc := pedido.NuevoService(&mockStore{errItem: pedido.ErrPedidoSinConfirmar})
	_, err := svc.CambiarEstadoItem(context.Background(), "item-1", "tenant-1", "", "preparando")
	if !errors.Is(err, pedido.ErrPedidoSinConfirmar) {
		t.Fatalf("esperaba ErrPedidoSinConfirmar, got %v", err)
	}
}

func TestCrear_SinItems_Error(t *testing.T) {
	svc := pedido.NuevoService(&mockStore{})
	_, err := svc.Crear(context.Background(), pedido.NuevoPedidoInput{
		MesaID: "mesa-1",
		Items:  []pedido.NuevoItemInput{},
	})
	if err == nil {
		t.Fatal("esperaba error por pedido sin items")
	}
}

func TestCrear_VarianteVacia_Error(t *testing.T) {
	svc := pedido.NuevoService(&mockStore{})
	_, err := svc.Crear(context.Background(), pedido.NuevoPedidoInput{
		MesaID: "mesa-1",
		Items: []pedido.NuevoItemInput{
			{
				ArticuloID:     "art-1",
				Cantidad:       1,
				ComensalID:     "47dc8c9e-fb98-44d7-80a1-b598addc1e8a",
				ComensalNombre: "Mateo",
				Variantes:      []string{"  "},
			},
		},
	})
	if err == nil {
		t.Fatal("esperaba error por variante_id vacío")
	}
}

func TestCrear_ConVariantes_Exito(t *testing.T) {
	store := &mockStore{}
	svc := pedido.NuevoService(store)
	_, err := svc.Crear(context.Background(), pedido.NuevoPedidoInput{
		MesaID: "mesa-1",
		Items: []pedido.NuevoItemInput{
			{
				ArticuloID:     "art-1",
				Cantidad:       2,
				ComensalID:     "47dc8c9e-fb98-44d7-80a1-b598addc1e8a",
				ComensalNombre: "Mateo",
				Variantes:      []string{"var-1", "var-2"},
			},
		},
	})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if len(store.crearInput.Items[0].Variantes) != 2 {
		t.Fatalf("esperaba 2 variantes, obtuve %d", len(store.crearInput.Items[0].Variantes))
	}
}

func TestCrearAsistido_RegistraOrigenCreadorYAlias(t *testing.T) {
	store := &mockStore{}
	svc := pedido.NuevoService(store)

	_, err := svc.CrearAsistido(context.Background(), &auth.Claims{
		UsuarioID:  "usuario-1",
		TenantID:   "tenant-1",
		Rol:        auth.RolMozo,
		SucursalID: "suc-1",
	}, pedido.PedidoAsistidoInput{
		MesaID:       "mesa-1",
		AliasCliente: "  Mesa cumple  ",
		Items: []pedido.NuevoItemInput{{
			ArticuloID: "art-1",
			Cantidad:   1,
		}},
	})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if store.crearInput.Origen != "mozo" {
		t.Fatalf("origen incorrecto: got %q, want mozo", store.crearInput.Origen)
	}
	if store.crearInput.CreadoPorUsuarioID != "usuario-1" {
		t.Fatalf("creador incorrecto: got %q", store.crearInput.CreadoPorUsuarioID)
	}
	item := store.crearInput.Items[0]
	if item.ComensalNombre != "Mesa cumple" {
		t.Fatalf("alias no normalizado: got %q", item.ComensalNombre)
	}
	if len(item.ComensalID) != 36 {
		t.Fatalf("se esperaba identidad UUID para el comensal, got %q", item.ComensalID)
	}
}

func TestCrearAsistido_RechazaMesaDeOtraSucursal(t *testing.T) {
	store := &mockStore{}
	svc := pedido.NuevoService(store)

	_, err := svc.CrearAsistido(context.Background(), &auth.Claims{
		UsuarioID:  "usuario-1",
		TenantID:   "tenant-1",
		Rol:        auth.RolMozo,
		SucursalID: "suc-2",
	}, pedido.PedidoAsistidoInput{
		MesaID: "mesa-1",
		Items: []pedido.NuevoItemInput{{
			ArticuloID: "art-1",
			Cantidad:   1,
		}},
	})
	if err == nil {
		t.Fatal("se esperaba rechazo para una mesa de otra sucursal")
	}
	if store.crearInput.MesaID != "" {
		t.Fatal("el pedido no debe llegar a persistencia cuando la sucursal no coincide")
	}
}

func TestCambiarEstadoItem_EstadoInvalido(t *testing.T) {
	svc := pedido.NuevoService(&mockStore{})
	_, err := svc.CambiarEstadoItem(context.Background(), "item-1", "tenant-1", "", "entregado")
	if err == nil {
		t.Fatal("esperaba error por estado de item inválido")
	}
}

func TestCambiarEstadoItem_EstadosValidos(t *testing.T) {
	store := &mockStore{}
	svc := pedido.NuevoService(store)

	for _, estado := range pedido.EstadosItemValidos {
		_, err := svc.CambiarEstadoItem(context.Background(), "item-1", "tenant-1", "", estado)
		if err != nil {
			t.Errorf("estado %q debería ser válido pero dio error: %v", estado, err)
		}
		if store.estadoItemRecibido != estado {
			t.Errorf("el store recibió %q, se esperaba %q", store.estadoItemRecibido, estado)
		}
	}
}

func TestCambiarEstadoItem_PublicaEventoEnCanalKDS(t *testing.T) {
	store := &mockStore{
		pedidoTrasCambioDeItem: &pedido.Pedido{
			ID:         "pedido-1",
			MesaID:     "mesa-1",
			SucursalID: "suc-kds",
			Estado:     "listo",
			Items:      []pedido.PedidoItem{{ID: "item-1", Estado: "listo"}},
		},
	}
	svc := pedido.NuevoService(store)
	canal, desuscribir := notificacion.Instancia.Suscribir("kds:suc-kds")
	defer desuscribir()

	_, err := svc.CambiarEstadoItem(context.Background(), "item-1", "tenant-1", "", "listo")
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	select {
	case evento := <-canal:
		if evento.Nombre != "pedido_item_actualizado" {
			t.Fatalf("evento incorrecto: got %q, want %q", evento.Nombre, "pedido_item_actualizado")
		}
	case <-time.After(time.Second):
		t.Fatal("no se publicó la actualización en el canal KDS")
	}
}

func TestValidarAccesoKDS_RechazaSucursalAjena(t *testing.T) {
	svc := pedido.NuevoService(&mockStore{sucursalPertenece: false})
	if err := svc.ValidarAccesoKDS(context.Background(), "suc-otra", "tenant-1"); err == nil {
		t.Fatal("esperaba error para una sucursal de otro tenant")
	}
}

func TestValidarAccesoKDS_AceptaSucursalPropia(t *testing.T) {
	svc := pedido.NuevoService(&mockStore{sucursalPertenece: true})
	if err := svc.ValidarAccesoKDS(context.Background(), "suc-1", "tenant-1"); err != nil {
		t.Fatalf("la sucursal del tenant debería aceptarse: %v", err)
	}
}

func TestCrear_ActorEsComensalConNombre(t *testing.T) {
	store := &mockStore{}
	svc := pedido.NuevoService(store)

	_, err := svc.Crear(context.Background(), pedido.NuevoPedidoInput{
		MesaID: "mesa-1",
		Items: []pedido.NuevoItemInput{
			{ArticuloID: "a-1", Cantidad: 1, ComensalID: "47dc8c9e-fb98-44d7-80a1-b598addc1e8a"},
			{ArticuloID: "a-2", Cantidad: 1, ComensalID: "5ef10747-503a-4d72-a433-94978a2447e6", ComensalNombre: " Juani "},
		},
	})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	actor := evento.ActorDesdeContexto(store.crearCtx, nil)
	if actor.Tipo != evento.ActorComensal || actor.Nombre != "Juani" || actor.UsuarioID != "" {
		t.Fatalf("actor incorrecto: %+v", actor)
	}
}

func TestCrearAsistido_ActorEsUsuarioConRol(t *testing.T) {
	store := &mockStore{}
	svc := pedido.NuevoService(store)

	_, err := svc.CrearAsistido(context.Background(), &auth.Claims{
		UsuarioID: "usuario-1", TenantID: "tenant-1", Rol: auth.RolMozo, SucursalID: "suc-1",
	}, pedido.PedidoAsistidoInput{
		MesaID: "mesa-1",
		Items:  []pedido.NuevoItemInput{{ArticuloID: "art-1", Cantidad: 1}},
	})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	actor := evento.ActorDesdeContexto(store.crearCtx, nil)
	if actor.Tipo != evento.ActorUsuario || actor.UsuarioID != "usuario-1" || actor.Rol != auth.RolMozo {
		t.Fatalf("actor incorrecto: %+v", actor)
	}
}
