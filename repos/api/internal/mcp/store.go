package mcp

import (
	"context"
	"errors"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/db"
)

// Store reúne las lecturas SQL del servidor MCP. Todas filtran por tenant.
type Store interface {
	SucursalDelTenant(ctx context.Context, tenantID, sucursalID string) (*SucursalRef, error)
	SucursalesDelTenant(ctx context.Context, tenantID string) ([]SucursalResumen, error)
	EstadoSalon(ctx context.Context, tenantID, sucursalID string, ahora time.Time) ([]MesaEstado, error)
	PedidosCocina(ctx context.Context, tenantID, sucursalID string, ahora time.Time) ([]PedidoCocina, error)
	DespachoPromedio(ctx context.Context, tenantID, sucursalID string, desde time.Time) (*float64, error)
	ComensalesAtendidos(ctx context.Context, tenantID, sucursalID string, desde, hasta time.Time) (int, error)
}

type pgStore struct{}

func NuevoStore() Store { return &pgStore{} }

var reUUID = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func (s *pgStore) SucursalDelTenant(ctx context.Context, tenantID, sucursalID string) (*SucursalRef, error) {
	if !reUUID.MatchString(sucursalID) || !reUUID.MatchString(tenantID) {
		return nil, ErrSucursalNoEncontrada
	}
	var ref SucursalRef
	err := db.Pool.QueryRow(ctx,
		`SELECT id, nombre FROM sucursales WHERE id = $1 AND tenant_id = $2`,
		sucursalID, tenantID).Scan(&ref.ID, &ref.Nombre)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSucursalNoEncontrada
	}
	if err != nil {
		return nil, err
	}
	return &ref, nil
}

func (s *pgStore) SucursalesDelTenant(ctx context.Context, tenantID string) ([]SucursalResumen, error) {
	if !reUUID.MatchString(tenantID) {
		return []SucursalResumen{}, nil
	}
	rows, err := db.Pool.Query(ctx, `
		SELECT su.id, su.nombre,
		       (SELECT COUNT(*) FROM mesas m WHERE m.sucursal_id = su.id AND m.estado = 'activa')::int
		FROM sucursales su
		WHERE su.tenant_id = $1
		ORDER BY su.nombre, su.id`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	resultado := []SucursalResumen{}
	for rows.Next() {
		var r SucursalResumen
		if err := rows.Scan(&r.ID, &r.Nombre, &r.MesasActivas); err != nil {
			return nil, err
		}
		resultado = append(resultado, r)
	}
	return resultado, rows.Err()
}

func (s *pgStore) EstadoSalon(ctx context.Context, tenantID, sucursalID string, ahora time.Time) ([]MesaEstado, error) {
	if !reUUID.MatchString(sucursalID) || !reUUID.MatchString(tenantID) {
		return []MesaEstado{}, nil
	}
	rows, err := db.Pool.Query(ctx, `
		WITH mesas_activas AS (
			SELECT m.id, m.numero, m.capacidad, m.cuenta_solicitada, m.pago_habilitado,
			       m.cuenta_version, sec.nombre AS sector
			FROM mesas m
			JOIN sucursales su ON su.id = m.sucursal_id
			LEFT JOIN sectores sec ON sec.id = m.sector_id AND sec.sucursal_id = m.sucursal_id
			WHERE su.tenant_id = $1 AND m.sucursal_id = $2 AND m.estado = 'activa'
		),
		por_pedido AS (
			SELECT m.id AS mesa_id,
			       COUNT(*) AS totales,
			       COUNT(*) FILTER (WHERE p.estado <> 'cerrado') AS activos,
			       MIN(p.created_at) AS primero
			FROM mesas_activas m
			JOIN pedidos p ON p.mesa_id = m.id AND p.cuenta_version = m.cuenta_version
			GROUP BY m.id
		),
		por_item AS (
			SELECT m.id AS mesa_id,
			       COUNT(DISTINCT pi.comensal_id) AS comensales,
			       COALESCE(BOOL_OR(pi.comensal_id IS NULL), FALSE) AS sin_identificar,
			       COALESCE(SUM(pi.cantidad * pi.precio_unitario), 0)::float8 AS total
			FROM mesas_activas m
			JOIN pedidos p ON p.mesa_id = m.id AND p.cuenta_version = m.cuenta_version
			JOIN pedido_items pi ON pi.pedido_id = p.id
			GROUP BY m.id
		)
		SELECT m.numero, COALESCE(m.sector, ''), m.capacidad, m.cuenta_solicitada, m.pago_habilitado,
		       COALESCE(pp.totales, 0)::int, COALESCE(pp.activos, 0)::int,
		       COALESCE(pi.comensales, 0)::int, COALESCE(pi.sin_identificar, FALSE),
		       COALESCE(pi.total, 0)::float8,
		       CASE WHEN pp.primero IS NULL THEN NULL
		            ELSE GREATEST(FLOOR(EXTRACT(EPOCH FROM ($3::timestamptz - pp.primero)) / 60), 0)::int END
		FROM mesas_activas m
		LEFT JOIN por_pedido pp ON pp.mesa_id = m.id
		LEFT JOIN por_item pi ON pi.mesa_id = m.id
		ORDER BY m.numero`, tenantID, sucursalID, ahora)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	mesas := []MesaEstado{}
	for rows.Next() {
		var m MesaEstado
		var d datosMesa
		if err := rows.Scan(&m.Numero, &m.Sector, &m.Capacidad, &d.CuentaSolicitada, &d.PagoHabilitado,
			&d.PedidosTotales, &d.PedidosActivos, &d.Comensales, &d.SinIdentificar,
			&m.TotalParcial, &d.MinutosPrimero); err != nil {
			return nil, err
		}
		m.Estado, m.Comensales, m.ComensalesSinIdentificar = derivarMesa(d)
		m.PedidosActivos = d.PedidosActivos
		m.MinutosDesdePrimerPedido = d.MinutosPrimero
		mesas = append(mesas, m)
	}
	return mesas, rows.Err()
}

func (s *pgStore) PedidosCocina(ctx context.Context, tenantID, sucursalID string, ahora time.Time) ([]PedidoCocina, error) {
	if !reUUID.MatchString(sucursalID) || !reUUID.MatchString(tenantID) {
		return []PedidoCocina{}, nil
	}
	rows, err := db.Pool.Query(ctx, `
		SELECT m.numero, p.estado,
		       GREATEST(FLOOR(EXTRACT(EPOCH FROM ($3::timestamptz - p.created_at)) / 60), 0)::int,
		       COUNT(pi.id) FILTER (WHERE pi.estado = 'pendiente')::int,
		       COUNT(pi.id) FILTER (WHERE pi.estado = 'preparando')::int,
		       COUNT(pi.id) FILTER (WHERE pi.estado = 'listo')::int
		FROM pedidos p
		JOIN sucursales su ON su.id = p.sucursal_id
		JOIN mesas m ON m.id = p.mesa_id AND p.cuenta_version = m.cuenta_version
		LEFT JOIN pedido_items pi ON pi.pedido_id = p.id
		WHERE su.tenant_id = $1 AND p.sucursal_id = $2
		  AND p.estado IN ('recibido', 'confirmado', 'preparando', 'listo')
		GROUP BY p.id, m.numero, p.estado, p.created_at
		ORDER BY p.created_at, p.id`, tenantID, sucursalID, ahora)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	pedidos := []PedidoCocina{}
	for rows.Next() {
		var p PedidoCocina
		if err := rows.Scan(&p.Mesa, &p.Estado, &p.Minutos, &p.ItemsPendientes, &p.ItemsPreparando, &p.ItemsListos); err != nil {
			return nil, err
		}
		pedidos = append(pedidos, p)
	}
	return pedidos, rows.Err()
}

func (s *pgStore) DespachoPromedio(ctx context.Context, tenantID, sucursalID string, desde time.Time) (*float64, error) {
	if !reUUID.MatchString(sucursalID) || !reUUID.MatchString(tenantID) {
		return nil, nil
	}
	var promedio *float64
	err := db.Pool.QueryRow(ctx, `
		SELECT ROUND(AVG(EXTRACT(EPOCH FROM (p.listo_at - p.created_at)) / 60)::numeric, 1)::float8
		FROM pedidos p
		JOIN sucursales su ON su.id = p.sucursal_id
		WHERE su.tenant_id = $1 AND p.sucursal_id = $2
		  AND p.listo_at IS NOT NULL AND p.listo_at >= $3
		  AND p.created_at >= $3 - interval '1 day'`,
		tenantID, sucursalID, desde).Scan(&promedio)
	if err != nil {
		return nil, err
	}
	return promedio, nil
}

func (s *pgStore) ComensalesAtendidos(ctx context.Context, tenantID, sucursalID string, desde, hasta time.Time) (int, error) {
	if !reUUID.MatchString(sucursalID) || !reUUID.MatchString(tenantID) {
		return 0, nil
	}
	var total int
	err := db.Pool.QueryRow(ctx, `
		WITH por_cuenta AS (
			SELECT GREATEST(COUNT(DISTINCT pi.comensal_id),
			                CASE WHEN BOOL_OR(pi.comensal_id IS NULL) THEN 1 ELSE 0 END) AS comensales
			FROM pedidos p
			JOIN sucursales su ON su.id = p.sucursal_id
			JOIN pedido_items pi ON pi.pedido_id = p.id
			WHERE su.tenant_id = $1 AND p.sucursal_id = $2
			  AND p.created_at >= $3 AND p.created_at < $4
			GROUP BY p.mesa_id, p.cuenta_version
		)
		SELECT COALESCE(SUM(comensales), 0)::int FROM por_cuenta`,
		tenantID, sucursalID, desde, hasta).Scan(&total)
	return total, err
}
