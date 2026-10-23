package mesa

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/evento"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/notificacion"
)

type Service struct {
	store Store
}

func NuevoService(s Store) *Service { return &Service{store: s} }

func (svc *Service) Listar(ctx context.Context, tenantID string) ([]Mesa, error) {
	return svc.store.Listar(ctx, tenantID)
}

func (svc *Service) ListarSucursal(ctx context.Context, tenantID, sucursalID string) ([]Mesa, error) {
	mesas, err := svc.store.Listar(ctx, tenantID)
	if err != nil || sucursalID == "" {
		return mesas, err
	}
	filtradas := make([]Mesa, 0)
	for _, m := range mesas {
		if m.SucursalID == sucursalID {
			filtradas = append(filtradas, m)
		}
	}
	return filtradas, nil
}

func (svc *Service) ValidarAlcance(ctx context.Context, mesaID, tenantID, sucursalID string) error {
	if sucursalID == "" {
		return nil
	}
	store, ok := svc.store.(ScopedStore)
	if !ok {
		return nil
	}
	pertenece, err := store.MesaPerteneceSucursal(ctx, mesaID, tenantID, sucursalID)
	if err != nil {
		return err
	}
	if !pertenece {
		return ErrNotFound
	}
	return nil
}

func (svc *Service) Crear(ctx context.Context, tenantID string, input MesaInput) (*Mesa, error) {
	if input.Numero <= 0 {
		return nil, fmt.Errorf("número de mesa debe ser mayor a 0: %w", ErrValidation)
	}
	if input.SucursalID == "" {
		return nil, fmt.Errorf("sucursal_id requerido: %w", ErrValidation)
	}
	qrToken, err := generarQRToken()
	if err != nil {
		return nil, fmt.Errorf("error generando QR token: %w", err)
	}
	return svc.store.Crear(ctx, tenantID, input, qrToken)
}

func (svc *Service) Actualizar(ctx context.Context, id, tenantID string, u MesaUpdate) (*Mesa, error) {
	if u.Estado != nil && *u.Estado != "activa" && *u.Estado != "inactiva" {
		return nil, fmt.Errorf("estado inválido: debe ser 'activa' o 'inactiva': %w", ErrValidation)
	}
	return svc.store.Actualizar(ctx, id, tenantID, u)
}

func (svc *Service) HabilitarPago(ctx context.Context, id, tenantID string, habilitado bool) (*Mesa, error) {
	m, err := svc.store.HabilitarPago(ctx, id, tenantID, habilitado)
	if err != nil {
		return nil, err
	}
	evento := "pago_habilitado"
	if !habilitado {
		evento = "pago_deshabilitado"
	}
	notificacion.Instancia.Publicar(fmt.Sprintf("sucursal:%s", m.SucursalID), evento, m)
	notificacion.Instancia.Publicar(fmt.Sprintf("mesa:%s", m.ID), evento, m)
	return m, nil
}

func (svc *Service) CerrarCuenta(ctx context.Context, id, tenantID string) (*Mesa, error) {
	m, err := svc.store.CerrarCuenta(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}
	notificacion.Instancia.Publicar(fmt.Sprintf("sucursal:%s", m.SucursalID), "cuenta_cerrada", m)
	notificacion.Instancia.Publicar(fmt.Sprintf("mesa:%s", m.ID), "cuenta_cerrada", m)
	return m, nil
}

func (svc *Service) Eliminar(ctx context.Context, id, tenantID string) error {
	return svc.store.Eliminar(ctx, id, tenantID)
}

func (svc *Service) ObtenerPorQRToken(ctx context.Context, token string) (*MesaPublica, error) {
	return svc.store.ObtenerPorQRToken(ctx, token)
}

func (svc *Service) SolicitarCuentaPorQRToken(ctx context.Context, token string) (*MesaPublica, error) {
	ctx = evento.ContextoConActor(ctx, evento.Comensal(""))
	mp, err := svc.store.SolicitarCuentaPorQRToken(ctx, token)
	if err != nil {
		return nil, err
	}
	notificacion.Instancia.Publicar(fmt.Sprintf("sucursal:%s", mp.SucursalID), "cuenta_solicitada", mp)
	notificacion.Instancia.Publicar(fmt.Sprintf("mesa:%s", mp.ID), "cuenta_solicitada", mp)
	return mp, nil
}

func generarQRToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
