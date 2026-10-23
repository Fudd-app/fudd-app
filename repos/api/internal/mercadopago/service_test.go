package mercadopago_test

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/evento"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/mercadopago"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/mesa"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/pedido"
)

type mockMPClient struct {
	prefResp *mercadopago.PreferenciaResponse
	pagoResp *mercadopago.PagoMercadoPago
	prefErr  error
	pagoErr  error
}

func (m *mockMPClient) CrearPreferencia(ctx context.Context, req mercadopago.PreferenciaRequest) (*mercadopago.PreferenciaResponse, error) {
	if m.prefErr != nil {
		return nil, m.prefErr
	}
	return m.prefResp, nil
}

func (m *mockMPClient) ObtenerPago(ctx context.Context, paymentID string) (*mercadopago.PagoMercadoPago, error) {
	if m.pagoErr != nil {
		return nil, m.pagoErr
	}
	return m.pagoResp, nil
}

type mockMPStore struct {
	guardadaPref  bool
	registrado    bool
	pagoExistente bool
	inactivo      bool
	tokenOverride string
}

func (m *mockMPStore) GuardarPreferencia(ctx context.Context, mesaID string, cuentaVersion int, prefID string, monto float64) error {
	m.guardadaPref = true
	return nil
}

func (m *mockMPStore) RegistrarPagoAprobado(ctx context.Context, mesaID string, cuentaVersion int, pagoID, prefID string, monto float64, detalles any) (*mercadopago.RegistroPago, error) {
	m.registrado = true
	return &mercadopago.RegistroPago{
		ID:            "reg-1",
		MesaID:        mesaID,
		CuentaVersion: cuentaVersion,
		PagoID:        pagoID,
		Monto:         monto,
		Moneda:        "ARS",
		Estado:        "aprobado",
		CreatedAt:     time.Now(),
	}, nil
}

func (m *mockMPStore) ExistePago(ctx context.Context, pagoID string) (bool, error) {
	return m.pagoExistente, nil
}

func (m *mockMPStore) ObtenerMesaIDYTenantPorPago(ctx context.Context, mesaID string) (string, string, int, error) {
	return "tenant-1", "suc-1", 3, nil
}

func (m *mockMPStore) ObtenerCredencialesMesa(ctx context.Context, mesaID string) (string, bool, error) {
	if m.inactivo {
		return "", false, nil
	}
	if m.tokenOverride != "" {
		return m.tokenOverride, true, nil
	}
	return "TEST-token", true, nil
}

type mockMesaStore struct {
	mesaPub   *mesa.MesaPublica
	cerrarCtx context.Context
}

func (m *mockMesaStore) Listar(ctx context.Context, tenantID string) ([]mesa.Mesa, error) {
	return nil, nil
}
func (m *mockMesaStore) Crear(ctx context.Context, tenantID string, input mesa.MesaInput, qrToken string) (*mesa.Mesa, error) {
	return nil, nil
}
func (m *mockMesaStore) Actualizar(ctx context.Context, id, tenantID string, u mesa.MesaUpdate) (*mesa.Mesa, error) {
	return nil, nil
}
func (m *mockMesaStore) HabilitarPago(ctx context.Context, id, tenantID string, habilitado bool) (*mesa.Mesa, error) {
	return &mesa.Mesa{ID: id, SucursalID: "suc-1", Estado: "activa", PagoHabilitado: habilitado}, nil
}
func (m *mockMesaStore) CerrarCuenta(ctx context.Context, id, tenantID string) (*mesa.Mesa, error) {
	m.cerrarCtx = ctx
	return &mesa.Mesa{ID: id, SucursalID: "suc-1", Estado: "activa", CuentaVersion: 2}, nil
}
func (m *mockMesaStore) Eliminar(ctx context.Context, id, tenantID string) error { return nil }
func (m *mockMesaStore) ObtenerPorQRToken(ctx context.Context, token string) (*mesa.MesaPublica, error) {
	if m.mesaPub == nil {
		return nil, mesa.ErrNotFound
	}
	return m.mesaPub, nil
}
func (m *mockMesaStore) SolicitarCuentaPorQRToken(ctx context.Context, token string) (*mesa.MesaPublica, error) {
	return m.mesaPub, nil
}

type mockPedidoStore struct {
	pedidos []pedido.Pedido
}

func (m *mockPedidoStore) Crear(ctx context.Context, input pedido.NuevoPedidoInput, sucursalID string, cuentaVersion int) (*pedido.Pedido, error) {
	return nil, nil
}
func (m *mockPedidoStore) ListarActivos(ctx context.Context, sucursalID, tenantID string) ([]pedido.Pedido, error) {
	return nil, nil
}
func (m *mockPedidoStore) ListarCuentaActualPorQR(ctx context.Context, qrToken string) ([]pedido.Pedido, error) {
	return m.pedidos, nil
}
func (m *mockPedidoStore) CambiarEstado(ctx context.Context, id, tenantID, sucursalAlcance, nuevoEstado string) (*pedido.Pedido, error) {
	return nil, nil
}
func (m *mockPedidoStore) CambiarEstadoItem(ctx context.Context, itemID, tenantID, sucursalAlcance, nuevoEstado string) (*pedido.Pedido, error) {
	return nil, nil
}
func (m *mockPedidoStore) SucursalPerteneceATenant(ctx context.Context, sucursalID, tenantID string) (bool, error) {
	return true, nil
}
func (m *mockPedidoStore) ObtenerPorID(ctx context.Context, id string) (*pedido.Pedido, error) {
	return nil, nil
}
func (m *mockPedidoStore) ObtenerSucursalPorMesa(ctx context.Context, mesaID string) (sucursalID string, cuentaVersion int, err error) {
	return "suc-1", 1, nil
}

func TestCrearPreferenciaMesa_Exitoso(t *testing.T) {
	mesaStore := &mockMesaStore{
		mesaPub: &mesa.MesaPublica{
			ID:               "mesa-1",
			Numero:           3,
			SucursalID:       "suc-1",
			TenantID:         "tenant-1",
			Estado:           "activa",
			CuentaSolicitada: true,
			PagoHabilitado:   true,
			CuentaVersion:    1,
		},
	}
	mesaSvc := mesa.NuevoService(mesaStore)

	pedidosStore := &mockPedidoStore{
		pedidos: []pedido.Pedido{
			{
				ID: "ped-1",
				Items: []pedido.PedidoItem{
					{
						NombreArticulo: "Burger Clásica",
						Cantidad:       2,
						PrecioUnitario: 5000,
					},
					{
						NombreArticulo: "Papas Fritas",
						Cantidad:       1,
						PrecioUnitario: 2500,
					},
				},
			},
		},
	}
	pedidosSvc := pedido.NuevoService(pedidosStore)

	mpClient := &mockMPClient{
		prefResp: &mercadopago.PreferenciaResponse{
			ID:               "pref-123",
			InitPoint:        "https://www.mercadopago.com.ar/checkout/v1/redirect?pref_id=pref-123",
			SandboxInitPoint: "https://sandbox.mercadopago.com.ar/checkout/v1/redirect?pref_id=pref-123",
		},
	}
	mpStore := &mockMPStore{}

	svc := mercadopago.NuevoService(mpClient, mpStore, mesaSvc, pedidosSvc, "http://localhost:3000", "http://localhost:8080")
	svc.SetClientFactory(func(token string) mercadopago.Client {
		return mpClient
	})

	pref, err := svc.CrearPreferenciaMesa(context.Background(), "token-valido")
	if err != nil {
		t.Fatalf("error inesperado creando preferencia: %v", err)
	}

	if pref.PreferenceID != "pref-123" {
		t.Errorf("esperaba pref-123, obtuve %s", pref.PreferenceID)
	}
	if pref.MontoTotal != 12500 {
		t.Errorf("esperaba monto total 12500, obtuve %f", pref.MontoTotal)
	}
	if pref.SandboxInitPoint == "" {
		t.Error("esperaba sandbox_init_point no vacío")
	}
	if !mpStore.guardadaPref {
		t.Error("esperaba que la preferencia se guarde en el store")
	}
}

func TestCrearPreferenciaMesa_PagoNoHabilitado(t *testing.T) {
	mesaStore := &mockMesaStore{
		mesaPub: &mesa.MesaPublica{
			ID:               "mesa-1",
			Numero:           3,
			Estado:           "activa",
			CuentaSolicitada: true,
			PagoHabilitado:   false,
			CuentaVersion:    1,
		},
	}
	mesaSvc := mesa.NuevoService(mesaStore)
	pedidosStore := &mockPedidoStore{
		pedidos: []pedido.Pedido{
			{
				Items: []pedido.PedidoItem{
					{NombreArticulo: "Burger", Cantidad: 1, PrecioUnitario: 5000},
				},
			},
		},
	}
	pedidosSvc := pedido.NuevoService(pedidosStore)

	svc := mercadopago.NuevoService(&mockMPClient{}, &mockMPStore{}, mesaSvc, pedidosSvc, "", "")

	_, err := svc.CrearPreferenciaMesa(context.Background(), "token-valido")
	if !errors.Is(err, mercadopago.ErrPagoNoHabilitado) {
		t.Fatalf("esperaba ErrPagoNoHabilitado, obtuve %v", err)
	}
}

func TestCrearPreferenciaMesa_SinConsumos(t *testing.T) {
	mesaStore := &mockMesaStore{
		mesaPub: &mesa.MesaPublica{
			ID:             "mesa-1",
			Numero:         3,
			Estado:         "activa",
			PagoHabilitado: true,
			CuentaVersion:  1,
		},
	}
	mesaSvc := mesa.NuevoService(mesaStore)
	pedidosStore := &mockPedidoStore{pedidos: []pedido.Pedido{}}
	pedidosSvc := pedido.NuevoService(pedidosStore)

	svc := mercadopago.NuevoService(&mockMPClient{}, &mockMPStore{}, mesaSvc, pedidosSvc, "", "")

	_, err := svc.CrearPreferenciaMesa(context.Background(), "token-valido")
	if !errors.Is(err, mercadopago.ErrSinConsumos) {
		t.Fatalf("esperaba ErrSinConsumos, obtuve %v", err)
	}
}

func TestCrearPreferenciaMesa_Inactivo(t *testing.T) {
	mesaStore := &mockMesaStore{
		mesaPub: &mesa.MesaPublica{
			ID:             "mesa-1",
			Numero:         3,
			Estado:         "activa",
			PagoHabilitado: true,
			CuentaVersion:  1,
		},
	}
	mesaSvc := mesa.NuevoService(mesaStore)
	pedidosStore := &mockPedidoStore{
		pedidos: []pedido.Pedido{
			{
				Items: []pedido.PedidoItem{
					{NombreArticulo: "Burger", Cantidad: 1, PrecioUnitario: 5000},
				},
			},
		},
	}
	pedidosSvc := pedido.NuevoService(pedidosStore)

	mpStore := &mockMPStore{inactivo: true}
	svc := mercadopago.NuevoService(&mockMPClient{}, mpStore, mesaSvc, pedidosSvc, "", "")

	_, err := svc.CrearPreferenciaMesa(context.Background(), "token-valido")
	if !errors.Is(err, mercadopago.ErrMercadoPagoInactivo) {
		t.Fatalf("esperaba ErrMercadoPagoInactivo, obtuve %v", err)
	}
}

func TestConfirmarPago_Exitoso(t *testing.T) {
	mesaStore := &mockMesaStore{
		mesaPub: &mesa.MesaPublica{
			ID:            "mesa-1",
			Numero:        3,
			SucursalID:    "suc-1",
			TenantID:      "tenant-1",
			Estado:        "activa",
			CuentaVersion: 1,
		},
	}
	mesaSvc := mesa.NuevoService(mesaStore)
	pedidosSvc := pedido.NuevoService(&mockPedidoStore{})

	mpClient := &mockMPClient{
		pagoResp: &mercadopago.PagoMercadoPago{
			ID:                998877,
			Status:            "approved",
			StatusDetail:      "accredited",
			ExternalReference: "mesa:mesa-1:v1:token:token-valido",
			TransactionAmount: 12500,
		},
	}
	mpStore := &mockMPStore{}

	svc := mercadopago.NuevoService(mpClient, mpStore, mesaSvc, pedidosSvc, "", "")
	svc.SetClientFactory(func(token string) mercadopago.Client {
		return mpClient
	})

	rp, err := svc.ConfirmarPago(context.Background(), "token-valido", "998877")
	if err != nil {
		t.Fatalf("error inesperado confirmando pago: %v", err)
	}

	if rp.Estado != "aprobado" {
		t.Errorf("esperaba estado aprobado, obtuve %s", rp.Estado)
	}
	if !mpStore.registrado {
		t.Error("esperaba que el pago quede registrado en la BD")
	}
}

func TestConfirmarPago_Rechazado(t *testing.T) {
	mesaStore := &mockMesaStore{
		mesaPub: &mesa.MesaPublica{
			ID:            "mesa-1",
			Numero:        3,
			Estado:        "activa",
			CuentaVersion: 1,
		},
	}
	mesaSvc := mesa.NuevoService(mesaStore)
	pedidosSvc := pedido.NuevoService(&mockPedidoStore{})

	mpClient := &mockMPClient{
		pagoResp: &mercadopago.PagoMercadoPago{
			ID:                998877,
			Status:            "rejected",
			StatusDetail:      "cc_rejected_insufficient_amount",
			ExternalReference: "mesa:mesa-1:v1:token:token-valido",
		},
	}
	mpStore := &mockMPStore{}

	svc := mercadopago.NuevoService(mpClient, mpStore, mesaSvc, pedidosSvc, "", "")
	svc.SetClientFactory(func(token string) mercadopago.Client {
		return mpClient
	})

	_, err := svc.ConfirmarPago(context.Background(), "token-valido", "998877")
	if !errors.Is(err, mercadopago.ErrPagoNoAprobado) {
		t.Fatalf("esperaba ErrPagoNoAprobado, obtuve %v", err)
	}
}

func TestProcesarWebhook_Aprobado(t *testing.T) {
	mesaStore := &mockMesaStore{
		mesaPub: &mesa.MesaPublica{
			ID:            "mesa-1",
			Numero:        3,
			SucursalID:    "suc-1",
			TenantID:      "tenant-1",
			Estado:        "activa",
			CuentaVersion: 1,
		},
	}
	mesaSvc := mesa.NuevoService(mesaStore)
	pedidosSvc := pedido.NuevoService(&mockPedidoStore{})

	mpClient := &mockMPClient{
		pagoResp: &mercadopago.PagoMercadoPago{
			ID:                112233,
			Status:            "approved",
			StatusDetail:      "accredited",
			ExternalReference: "mesa:mesa-1:v1:token:token-valido",
			TransactionAmount: 7500,
		},
	}
	mpStore := &mockMPStore{}

	svc := mercadopago.NuevoService(mpClient, mpStore, mesaSvc, pedidosSvc, "", "")
	query := url.Values{
		"data.id": []string{"112233"},
		"type":    []string{"payment"},
	}

	err := svc.ProcesarWebhook(context.Background(), query, "", "")
	if err != nil {
		t.Fatalf("error inesperado en webhook: %v", err)
	}

	if !mpStore.registrado {
		t.Error("esperaba que el pago quede registrado tras webhook")
	}
	if actor := evento.ActorDesdeContexto(mesaStore.cerrarCtx, nil); actor.Tipo != evento.ActorMercadoPago {
		t.Errorf("el cierre por webhook debe atribuirse a mercadopago: %+v", actor)
	}
}

func TestProcesarWebhook_IgnorarNoPayment(t *testing.T) {
	mesaStore := &mockMesaStore{}
	mesaSvc := mesa.NuevoService(mesaStore)
	pedidosSvc := pedido.NuevoService(&mockPedidoStore{})
	mpStore := &mockMPStore{}

	svc := mercadopago.NuevoService(&mockMPClient{}, mpStore, mesaSvc, pedidosSvc, "", "")
	query := url.Values{
		"data.id": []string{"112233"},
		"type":    []string{"merchant_order"},
	}

	err := svc.ProcesarWebhook(context.Background(), query, "", "")
	if err != nil {
		t.Fatalf("error inesperado en webhook con tipo no payment: %v", err)
	}

	if mpStore.registrado {
		t.Error("no esperaba que se registre el pago si el webhook no es de tipo payment")
	}
}
