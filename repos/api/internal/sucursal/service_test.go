package sucursal_test

import (
	"context"
	"errors"
	"testing"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/sucursal"
)

type mockStore struct {
	crearFn          func(ctx context.Context, tenantID string, input sucursal.SucursalInput) (*sucursal.Sucursal, error)
	obtenerPorIDFn   func(ctx context.Context, id string, tenantID string) (*sucursal.Sucursal, error)
	listarFn         func(ctx context.Context, tenantID string) ([]sucursal.Sucursal, error)
	actualizarFn     func(ctx context.Context, id string, tenantID string, input sucursal.SucursalInput) (*sucursal.Sucursal, error)
	eliminarFn       func(ctx context.Context, id string, tenantID string) error
	crearSectorFn    func(ctx context.Context, sucursalID string, tenantID string, input sucursal.SectorInput) (*sucursal.Sector, error)
	listarSectoresFn func(ctx context.Context, sucursalID string, tenantID string) ([]sucursal.Sector, error)
	eliminarSectorFn func(ctx context.Context, id, tenantID, sucursalAlcance string) error
}

func (m *mockStore) Crear(ctx context.Context, tenantID string, input sucursal.SucursalInput) (*sucursal.Sucursal, error) {
	return m.crearFn(ctx, tenantID, input)
}
func (m *mockStore) ObtenerPorID(ctx context.Context, id string, tenantID string) (*sucursal.Sucursal, error) {
	return m.obtenerPorIDFn(ctx, id, tenantID)
}
func (m *mockStore) Listar(ctx context.Context, tenantID string) ([]sucursal.Sucursal, error) {
	return m.listarFn(ctx, tenantID)
}
func (m *mockStore) Actualizar(ctx context.Context, id string, tenantID string, input sucursal.SucursalInput) (*sucursal.Sucursal, error) {
	return m.actualizarFn(ctx, id, tenantID, input)
}
func (m *mockStore) Eliminar(ctx context.Context, id string, tenantID string) error {
	return m.eliminarFn(ctx, id, tenantID)
}
func (m *mockStore) CrearSector(ctx context.Context, sucursalID string, tenantID string, input sucursal.SectorInput) (*sucursal.Sector, error) {
	return m.crearSectorFn(ctx, sucursalID, tenantID, input)
}
func (m *mockStore) ListarSectores(ctx context.Context, sucursalID string, tenantID string) ([]sucursal.Sector, error) {
	return m.listarSectoresFn(ctx, sucursalID, tenantID)
}
func (m *mockStore) EliminarSector(ctx context.Context, id, tenantID, sucursalAlcance string) error {
	return m.eliminarSectorFn(ctx, id, tenantID, sucursalAlcance)
}

func TestCrearSucursal_Exitoso(t *testing.T) {
	store := &mockStore{
		crearFn: func(ctx context.Context, tenantID string, input sucursal.SucursalInput) (*sucursal.Sucursal, error) {
			return &sucursal.Sucursal{ID: "suc-1", TenantID: tenantID, Nombre: input.Nombre}, nil
		},
	}
	svc := sucursal.NuevoService(store)
	result, err := svc.Crear(context.Background(), "tenant-123", sucursal.SucursalInput{
		Nombre: "Sucursal Palermo",
	})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if result.Nombre != "Sucursal Palermo" {
		t.Errorf("got %q, want %q", result.Nombre, "Sucursal Palermo")
	}
}

func TestCrearSucursal_NombreVacio_Error(t *testing.T) {
	svc := sucursal.NuevoService(&mockStore{})
	_, err := svc.Crear(context.Background(), "tenant-123", sucursal.SucursalInput{})
	if err == nil {
		t.Fatal("se esperaba un error debido a nombre vacío")
	}
	if !errors.Is(err, sucursal.ErrValidation) {
		t.Errorf("se esperaba error de tipo ErrValidation, obtenido: %v", err)
	}
}

func TestCrearSector_Exitoso(t *testing.T) {
	store := &mockStore{
		crearSectorFn: func(ctx context.Context, sucursalID string, tenantID string, input sucursal.SectorInput) (*sucursal.Sector, error) {
			return &sucursal.Sector{ID: "sec-1", SucursalID: sucursalID, Nombre: input.Nombre}, nil
		},
	}
	svc := sucursal.NuevoService(store)
	result, err := svc.CrearSector(context.Background(), "suc-1", "tenant-123", sucursal.SectorInput{
		Nombre: "Terraza",
	})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if result.Nombre != "Terraza" {
		t.Errorf("got %q, want %q", result.Nombre, "Terraza")
	}
}

func TestCrearSector_NombreVacio_Error(t *testing.T) {
	svc := sucursal.NuevoService(&mockStore{})
	_, err := svc.CrearSector(context.Background(), "suc-1", "tenant-123", sucursal.SectorInput{})
	if err == nil {
		t.Fatal("se esperaba un error debido a nombre de sector vacío")
	}
	if !errors.Is(err, sucursal.ErrValidation) {
		t.Errorf("se esperaba error de tipo ErrValidation, obtenido: %v", err)
	}
}
