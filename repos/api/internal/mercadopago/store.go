package mercadopago

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/db"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/evento"
)

type Store interface {
	GuardarPreferencia(ctx context.Context, mesaID string, cuentaVersion int, preferenciaID string, monto float64) error
	RegistrarPagoAprobado(ctx context.Context, mesaID string, cuentaVersion int, pagoID, preferenciaID string, monto float64, detalles any) (*RegistroPago, error)
	ExistePago(ctx context.Context, pagoID string) (bool, error)
	ObtenerMesaIDYTenantPorPago(ctx context.Context, mesaID string) (tenantID string, sucursalID string, numero int, err error)
	ObtenerCredencialesMesa(ctx context.Context, mesaID string) (token string, activo bool, err error)
}

type pgStore struct{}

func NuevoStore() Store {
	return &pgStore{}
}

func (s *pgStore) GuardarPreferencia(ctx context.Context, mesaID string, cuentaVersion int, preferenciaID string, monto float64) error {
	query := `
		INSERT INTO pagos (mesa_id, cuenta_version, proveedor, preferencia_id, monto, moneda, estado, created_at, updated_at)
		VALUES ($1, $2, 'mercadopago', $3, $4, 'ARS', 'pendiente', now(), now())
	`
	_, err := db.Pool.Exec(ctx, query, mesaID, cuentaVersion, preferenciaID, monto)
	if err != nil {
		return fmt.Errorf("error guardando preferencia de pago: %w", err)
	}
	return nil
}

func (s *pgStore) RegistrarPagoAprobado(ctx context.Context, mesaID string, cuentaVersion int, pagoID, preferenciaID string, monto float64, detalles any) (*RegistroPago, error) {
	detallesJSON, err := json.Marshal(detalles)
	if err != nil {
		detallesJSON = []byte("{}")
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("error iniciando registro de pago: %w", err)
	}
	defer tx.Rollback(ctx)

	var rp RegistroPago
	err = tx.QueryRow(ctx, `
		INSERT INTO pagos (mesa_id, cuenta_version, proveedor, preferencia_id, pago_id, monto, moneda, estado, detalles, created_at, updated_at)
		VALUES ($1, $2, 'mercadopago', $3, $4, $5, 'ARS', 'aprobado', $6, now(), now())
		RETURNING id, mesa_id, cuenta_version, proveedor, COALESCE(preferencia_id, ''), COALESCE(pago_id, ''), monto, moneda, estado, created_at
	`, mesaID, cuentaVersion, preferenciaID, pagoID, monto, detallesJSON).Scan(
		&rp.ID, &rp.MesaID, &rp.CuentaVersion, &rp.Proveedor, &rp.PreferenciaID, &rp.PagoID, &rp.Monto, &rp.Moneda, &rp.Estado, &rp.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("error registrando pago aprobado: %w", err)
	}

	var sucursalID string
	if err := tx.QueryRow(ctx, `SELECT sucursal_id FROM mesas WHERE id = $1`, mesaID).Scan(&sucursalID); err != nil {
		return nil, fmt.Errorf("error obteniendo sucursal del pago: %w", err)
	}
	if err := evento.Registrar(ctx, tx, evento.Nuevo{
		SucursalID: sucursalID, MesaID: mesaID, CuentaVersion: cuentaVersion,
		Tipo: evento.TipoPagoAprobado, EstadoNuevo: "aprobado", Actor: evento.MercadoPago(),
		Detalle: map[string]any{"pago_id": pagoID, "monto": monto},
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("error confirmando registro de pago: %w", err)
	}
	return &rp, nil
}

func (s *pgStore) ExistePago(ctx context.Context, pagoID string) (bool, error) {
	var count int
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(1) FROM pagos WHERE pago_id = $1 AND estado = 'aprobado'
	`, pagoID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *pgStore) ObtenerMesaIDYTenantPorPago(ctx context.Context, mesaID string) (string, string, int, error) {
	var tenantID, sucursalID string
	var numero int
	err := db.Pool.QueryRow(ctx, `
		SELECT s.tenant_id, m.sucursal_id, m.numero
		FROM mesas m
		JOIN sucursales s ON s.id = m.sucursal_id
		WHERE m.id = $1
	`, mesaID).Scan(&tenantID, &sucursalID, &numero)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", 0, ErrMesaNoEncontrada
		}
		return "", "", 0, err
	}
	return tenantID, sucursalID, numero, nil
}

func (s *pgStore) ObtenerCredencialesMesa(ctx context.Context, mesaID string) (string, bool, error) {
	var token string
	var activo bool
	err := db.Pool.QueryRow(ctx, `
		SELECT 
			COALESCE(NULLIF(TRIM(su.mp_access_token), ''), NULLIF(TRIM(t.mp_access_token), ''), ''),
			COALESCE(su.mp_activo, t.mp_activo, true)
		FROM mesas m
		JOIN sucursales su ON su.id = m.sucursal_id
		JOIN tenants t ON t.id = su.tenant_id
		WHERE m.id = $1
	`, mesaID).Scan(&token, &activo)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", false, ErrMesaNoEncontrada
		}
		return "", false, err
	}
	return token, activo, nil
}
