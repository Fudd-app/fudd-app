package mcp

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/db"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/evento"
)

// Las lecturas de eventos delegan en internal/evento, que ya filtra por tenant.

func (s *pgStore) EventosSucursal(ctx context.Context, tenantID, sucursalID string, desde, hasta time.Time) ([]evento.Evento, error) {
	if !reUUID.MatchString(sucursalID) || !reUUID.MatchString(tenantID) {
		return []evento.Evento{}, nil
	}
	return evento.EventosPorSucursal(ctx, tenantID, sucursalID, desde, hasta)
}

func (s *pgStore) EventosMesa(ctx context.Context, tenantID, sucursalID, mesaID string, desde, hasta time.Time) ([]evento.Evento, error) {
	if !reUUID.MatchString(sucursalID) || !reUUID.MatchString(tenantID) || !reUUID.MatchString(mesaID) {
		return []evento.Evento{}, nil
	}
	return evento.EventosPorMesa(ctx, tenantID, sucursalID, mesaID, desde, hasta)
}

// PrimerEvento devuelve el ts del hito más antiguo de la sucursal, o nil si todavía no hay ninguno.
func (s *pgStore) PrimerEvento(ctx context.Context, tenantID, sucursalID string) (*time.Time, error) {
	if !reUUID.MatchString(sucursalID) || !reUUID.MatchString(tenantID) {
		return nil, nil
	}
	var primero *time.Time
	err := db.Pool.QueryRow(ctx, `
		SELECT MIN(e.ts)
		FROM eventos_operacion e
		JOIN sucursales su ON su.id = e.sucursal_id
		WHERE su.tenant_id = $1 AND e.sucursal_id = $2`, tenantID, sucursalID).Scan(&primero)
	return primero, err
}

// MesasSucursal devuelve todas las mesas de la sucursal, activas o no: para reconstruir el pasado
// importa también una mesa que hoy está dada de baja.
func (s *pgStore) MesasSucursal(ctx context.Context, tenantID, sucursalID string) ([]MesaRef, error) {
	if !reUUID.MatchString(sucursalID) || !reUUID.MatchString(tenantID) {
		return []MesaRef{}, nil
	}
	rows, err := db.Pool.Query(ctx, `
		SELECT m.id, m.numero, m.estado = 'activa'
		FROM mesas m
		JOIN sucursales su ON su.id = m.sucursal_id
		WHERE su.tenant_id = $1 AND m.sucursal_id = $2
		ORDER BY m.numero`, tenantID, sucursalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	mesas := []MesaRef{}
	for rows.Next() {
		var m MesaRef
		if err := rows.Scan(&m.ID, &m.Numero, &m.Activa); err != nil {
			return nil, err
		}
		mesas = append(mesas, m)
	}
	return mesas, rows.Err()
}

// MesaPorNumero busca la mesa por su número sin importar si está activa: el historial de una mesa dada de baja sigue siendo válido.
func (s *pgStore) MesaPorNumero(ctx context.Context, tenantID, sucursalID string, numero int) (*MesaRef, error) {
	if !reUUID.MatchString(sucursalID) || !reUUID.MatchString(tenantID) {
		return nil, ErrMesaNoEncontrada
	}
	var m MesaRef
	err := db.Pool.QueryRow(ctx, `
		SELECT m.id, m.numero
		FROM mesas m
		JOIN sucursales su ON su.id = m.sucursal_id
		WHERE su.tenant_id = $1 AND m.sucursal_id = $2 AND m.numero = $3`, tenantID, sucursalID, numero).Scan(&m.ID, &m.Numero)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrMesaNoEncontrada
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}
