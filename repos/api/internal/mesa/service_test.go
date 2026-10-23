package mesa_test

import (
	"context"
	"testing"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/evento"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/mesa"
)

type mockStore struct {
	mesas        []mesa.Mesa
	solicitarCtx context.Context
}

func (m *mockStore) Listar(ctx context.Context, tenantID string) ([]mesa.Mesa, error) {
	return m.mesas, nil
}
func (m *mockStore) Crear(ctx context.Context, tenantID string, input mesa.MesaInput, qrToken string) (*mesa.Mesa, error) {
	return &mesa.Mesa{ID: "m-1", Numero: input.Numero, QRToken: qrToken, Estado: "activa"}, nil
}
func (m *mockStore) Actualizar(ctx context.Context, id, tenantID string, u mesa.MesaUpdate) (*mesa.Mesa, error) {
	return &mesa.Mesa{ID: id}, nil
}
func (m *mockStore) HabilitarPago(ctx context.Context, id, tenantID string, habilitado bool) (*mesa.Mesa, error) {
	return &mesa.Mesa{ID: id, Estado: "activa", SucursalID: "suc-1", PagoHabilitado: habilitado}, nil
}
func (m *mockStore) CerrarCuenta(ctx context.Context, id, tenantID string) (*mesa.Mesa, error) {
	return &mesa.Mesa{ID: id, Estado: "activa", SucursalID: "suc-1", CuentaVersion: 2}, nil
}
func (m *mockStore) Eliminar(ctx context.Context, id, tenantID string) error { return nil }
func (m *mockStore) ObtenerPorQRToken(ctx context.Context, token string) (*mesa.MesaPublica, error) {
	return &mesa.MesaPublica{ID: "m-1"}, nil
}

func (m *mockStore) SolicitarCuentaPorQRToken(ctx context.Context, token string) (*mesa.MesaPublica, error) {
	m.solicitarCtx = ctx
	return &mesa.MesaPublica{ID: "m-1", SucursalID: "suc-1", CuentaSolicitada: true, CuentaVersion: 1}, nil
}

func TestCrear_QRTokenGenerado(t *testing.T) {
	svc := mesa.NuevoService(&mockStore{})
	m, err := svc.Crear(context.Background(), "tenant-1", mesa.MesaInput{
		SucursalID: "suc-1",
		Numero:     5,
		Capacidad:  4,
	})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if m.QRToken == "" {
		t.Error("QRToken vacío — debería generarse automáticamente")
	}
}

func TestCrear_NumeroInvalido(t *testing.T) {
	svc := mesa.NuevoService(&mockStore{})
	_, err := svc.Crear(context.Background(), "tenant-1", mesa.MesaInput{
		SucursalID: "suc-1",
		Numero:     0,
	})
	if err == nil {
		t.Fatal("esperaba error por número de mesa inválido")
	}
}

func TestHabilitarPago_Exitoso(t *testing.T) {
	svc := mesa.NuevoService(&mockStore{})
	m, err := svc.HabilitarPago(context.Background(), "m-1", "tenant-1", true)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if !m.PagoHabilitado {
		t.Error("esperaba PagoHabilitado en true")
	}
}

func TestHabilitarPago_Deshabilitar(t *testing.T) {
	svc := mesa.NuevoService(&mockStore{})
	m, err := svc.HabilitarPago(context.Background(), "m-1", "tenant-1", false)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if m.PagoHabilitado {
		t.Error("esperaba PagoHabilitado en false")
	}
}

func TestSolicitarCuenta_ActorEsComensal(t *testing.T) {
	store := &mockStore{}
	svc := mesa.NuevoService(store)
	if _, err := svc.SolicitarCuentaPorQRToken(context.Background(), "token"); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if actor := evento.ActorDesdeContexto(store.solicitarCtx, nil); actor.Tipo != evento.ActorComensal {
		t.Fatalf("actor incorrecto: %+v", actor)
	}
}
