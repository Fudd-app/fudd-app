package pedido_test

import (
	"context"
	"testing"
	"time"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/notificacion"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/pedido"
)

type mockStore struct {
	crearInput             pedido.NuevoPedidoInput
	estadoItemRecibido     string
	pedidoTrasCambioDeItem *pedido.Pedido
	sucursalPertenece      bool
}

func (m *mockStore) Crear(ctx context.Context, input pedido.NuevoPedidoInput, sucursalID string, cuentaVersion int) (*pedido.Pedido, error) {
	m.crearInput = input
	return &pedido.Pedido{ID: "p-1", MesaID: input.MesaID, CuentaVersion: cuentaVersion, Estado: "recibido"}, nil
}
func (m *mockStore) ListarActivos(ctx context.Context, sucursalID, tenantID string) ([]pedido.Pedido, error) {
	return []pedido.Pedido{}, nil
}
func (m *mockStore) ListarCuentaActualPorQR(ctx context.Context, qrToken string) ([]pedido.Pedido, error) {
	return []pedido.Pedido{}, nil
}
func (m *mockStore) CambiarEstado(ctx context.Context, id, tenantID, sucursalAlcance, nuevoEstado string) (*pedido.Pedido, error) {
	return &pedido.Pedido{ID: id, Estado: nuevoEstado}, nil
}
func (m *mockStore) CambiarEstadoItem(ctx context.Context, itemID, tenantID, sucursalAlcance, nuevoEstado string) (*pedido.Pedido, error) {
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

func TestCambiarEstado_EstadoInvalido(t *testing.T) {
	svc := pedido.NuevoService(&mockStore{})
	_, err := svc.CambiarEstado(context.Background(), "p-1", "tenant-1", "", "invalido")
	if err == nil {
		t.Fatal("esperaba error por estado inválido")
	}
}

func TestCambiarEstado_EstadosValidos(t *testing.T) {
	svc := pedido.NuevoService(&mockStore{})
	for _, estado := range pedido.EstadosValidos {
		_, err := svc.CambiarEstado(context.Background(), "p-1", "tenant-1", "", estado)
		if err != nil {
			t.Errorf("estado %q debería ser válido pero dio error: %v", estado, err)
		}
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
