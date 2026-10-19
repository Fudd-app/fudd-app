package sucursal

import (
	"context"
	"fmt"
)

type Service struct {
	store Store
}

func NuevoService(s Store) *Service { return &Service{store: s} }

func (svc *Service) Crear(ctx context.Context, tenantID string, input SucursalInput) (*Sucursal, error) {
	if input.Nombre == "" {
		return nil, fmt.Errorf("%w: nombre requerido", ErrValidation)
	}
	if input.Horarios == nil {
		input.Horarios = make(map[string]any)
	}
	return svc.store.Crear(ctx, tenantID, input)
}

func (svc *Service) ObtenerPorID(ctx context.Context, id string, tenantID string) (*Sucursal, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: id de sucursal requerido", ErrValidation)
	}
	return svc.store.ObtenerPorID(ctx, id, tenantID)
}

func (svc *Service) Listar(ctx context.Context, tenantID string) ([]Sucursal, error) {
	return svc.store.Listar(ctx, tenantID)
}

func (svc *Service) Actualizar(ctx context.Context, id string, tenantID string, input SucursalInput) (*Sucursal, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: id de sucursal requerido", ErrValidation)
	}
	if input.Nombre == "" {
		return nil, fmt.Errorf("%w: nombre requerido", ErrValidation)
	}
	if input.Horarios == nil {
		input.Horarios = make(map[string]any)
	}
	return svc.store.Actualizar(ctx, id, tenantID, input)
}

func (svc *Service) Eliminar(ctx context.Context, id string, tenantID string) error {
	if id == "" {
		return fmt.Errorf("%w: id de sucursal requerido", ErrValidation)
	}
	return svc.store.Eliminar(ctx, id, tenantID)
}

func (svc *Service) CrearSector(ctx context.Context, sucursalID string, tenantID string, input SectorInput) (*Sector, error) {
	if sucursalID == "" {
		return nil, fmt.Errorf("%w: sucursal_id requerido", ErrValidation)
	}
	if input.Nombre == "" {
		return nil, fmt.Errorf("%w: nombre de sector requerido", ErrValidation)
	}
	return svc.store.CrearSector(ctx, sucursalID, tenantID, input)
}

func (svc *Service) ListarSectores(ctx context.Context, sucursalID string, tenantID string) ([]Sector, error) {
	if sucursalID == "" {
		return nil, fmt.Errorf("%w: sucursal_id requerido", ErrValidation)
	}
	return svc.store.ListarSectores(ctx, sucursalID, tenantID)
}

func (svc *Service) EliminarSector(ctx context.Context, id, tenantID, sucursalAlcance string) error {
	if id == "" {
		return fmt.Errorf("%w: id de sector requerido", ErrValidation)
	}
	return svc.store.EliminarSector(ctx, id, tenantID, sucursalAlcance)
}
