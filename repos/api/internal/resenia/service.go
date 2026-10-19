package resenia

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/notificacion"
)

type Service struct {
	store Store
}

func NuevoService(store Store) *Service {
	return &Service{store: store}
}

func (svc *Service) CrearResena(ctx context.Context, qrToken string, input CrearResenaInput) (*Resena, error) {
	qrToken = strings.TrimSpace(qrToken)
	if qrToken == "" {
		return nil, fmt.Errorf("qr_token requerido: %w", ErrValidation)
	}

	if input.Estrellas < 1 || input.Estrellas > 5 {
		return nil, fmt.Errorf("las estrellas deben ser entre 1 y 5: %w", ErrValidation)
	}

	var comentario *string
	if input.Comentario != nil {
		limpio := strings.TrimSpace(*input.Comentario)
		if utf8.RuneCountInString(limpio) > 500 {
			return nil, fmt.Errorf("el comentario no puede superar los 500 caracteres: %w", ErrValidation)
		}
		if limpio != "" {
			comentario = &limpio
		}
	}

	mesaCtx, err := svc.store.ObtenerContextoMesaPorQR(ctx, qrToken)
	if err != nil {
		return nil, err
	}

	if mesaCtx.Estado != "activa" {
		return nil, ErrMesaInactiva
	}

	tienePedidos, err := svc.store.VerificarPedidosEnCuenta(ctx, mesaCtx.MesaID, mesaCtx.CuentaVersion)
	if err != nil {
		return nil, err
	}
	if !tienePedidos {
		return nil, ErrSinPedidos
	}

	yaExiste, err := svc.store.VerificarResenaExiste(ctx, mesaCtx.MesaID, mesaCtx.CuentaVersion)
	if err != nil {
		return nil, err
	}
	if yaExiste {
		return nil, ErrResenaDuplicada
	}

	resena, err := svc.store.CrearResena(
		ctx,
		mesaCtx.MesaID,
		mesaCtx.SucursalID,
		mesaCtx.TenantID,
		mesaCtx.CuentaVersion,
		input.Estrellas,
		comentario,
	)
	if err != nil {
		return nil, err
	}

	// Notificar en tiempo real al dashboard de la sucursal
	payload := ResenaNotificacionPayload{
		ID:         resena.ID,
		Estrellas:  resena.Estrellas,
		MesaNumero: mesaCtx.MesaNumero,
		SucursalID: mesaCtx.SucursalID,
		Comentario: resena.Comentario,
		CreatedAt:  resena.CreatedAt,
	}
	notificacion.Instancia.Publicar(fmt.Sprintf("sucursal:%s", mesaCtx.SucursalID), "resena_creada", payload)

	return resena, nil
}

func (svc *Service) RegistrarClickGoogle(ctx context.Context, qrToken string) error {
	qrToken = strings.TrimSpace(qrToken)
	if qrToken == "" {
		return fmt.Errorf("qr_token requerido: %w", ErrValidation)
	}

	return svc.store.RegistrarClickGoogle(ctx, qrToken)
}

func (svc *Service) ObtenerResumen(ctx context.Context, tenantID, sucursalID, desdeStr, hastaStr string) (*ResumenResenas, error) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return nil, fmt.Errorf("tenant_id requerido: %w", ErrValidation)
	}

	sucursalID = strings.TrimSpace(sucursalID)
	if sucursalID != "" {
		pertenece, err := svc.store.SucursalPerteneceATenant(ctx, sucursalID, tenantID)
		if err != nil {
			return nil, err
		}
		if !pertenece {
			return nil, ErrSucursalInvalida
		}
	}

	var desde *time.Time
	if strings.TrimSpace(desdeStr) != "" {
		t, err := parsearFecha(strings.TrimSpace(desdeStr), false)
		if err != nil {
			return nil, fmt.Errorf("fecha desde inválida: %w", ErrValidation)
		}
		desde = &t
	}

	var hasta *time.Time
	if strings.TrimSpace(hastaStr) != "" {
		t, err := parsearFecha(strings.TrimSpace(hastaStr), true)
		if err != nil {
			return nil, fmt.Errorf("fecha hasta inválida: %w", ErrValidation)
		}
		hasta = &t
	}

	return svc.store.ObtenerResumen(ctx, tenantID, sucursalID, desde, hasta)
}

func parsearFecha(valor string, finDelDia bool) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, valor); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.DateOnly, valor); err == nil {
		if finDelDia {
			return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location()), nil
		}
		return t, nil
	}
	return time.Time{}, fmt.Errorf("formato no reconocido: %s", valor)
}
