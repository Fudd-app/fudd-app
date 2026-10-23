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
	UltimaHoraCocina(ctx context.Context, tenantID, sucursalID string, ahora time.Time) (UltimaHora, error)
	ComensalesAtendidos(ctx context.Context, tenantID, sucursalID string, desde, hasta time.Time) (int, error)
	ResumenPorHora(ctx context.Context, tenantID, sucursalID string, desde, hasta time.Time, zona string) ([]HoraTurno, error)
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
		       (SELECT COUNT(*) FROM mesas m WHERE m.sucursal_id = su.id)::int,
		       (SELECT COUNT(*) FROM mesas m WHERE m.sucursal_id = su.id AND m.estado = 'activa')::int,
		       su.horarios
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
		if err := rows.Scan(&r.ID, &r.Nombre, &r.MesasTotales, &r.MesasActivas, &r.Horarios); err != nil {
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
			       MIN(p.created_at) AS primero,
			       MAX(p.created_at) AS ultimo,
			       COUNT(*) FILTER (WHERE p.estado IN ('recibido', 'confirmado', 'preparando')) AS en_cocina,
			       COUNT(*) FILTER (WHERE p.estado = 'listo') AS listos,
			       MIN(p.created_at) FILTER (WHERE p.estado IN ('recibido', 'confirmado', 'preparando')) AS cocina_desde,
			       MIN(p.listo_at) FILTER (WHERE p.estado = 'listo') AS listo_desde,
			       ARRAY_AGG(DISTINCT CASE WHEN p.origen = 'mozo'
			                               THEN COALESCE(NULLIF(p.creado_por_nombre, ''), 'Mozo')
			                               ELSE 'QR' END) AS tomadores
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
		       pp.primero, pp.ultimo,
		       COALESCE(pp.en_cocina, 0)::int, COALESCE(pp.listos, 0)::int, pp.cocina_desde, pp.listo_desde,
		       COALESCE(pp.tomadores, '{}'::text[])
		FROM mesas_activas m
		LEFT JOIN por_pedido pp ON pp.mesa_id = m.id
		LEFT JOIN por_item pi ON pi.mesa_id = m.id
		ORDER BY m.numero`, tenantID, sucursalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	mesas := []MesaEstado{}
	for rows.Next() {
		var m MesaEstado
		var d datosMesa
		var primero, ultimo, cocinaDesde, listoDesde *time.Time
		var enCocina, listos int
		if err := rows.Scan(&m.Numero, &m.Sector, &m.Capacidad, &d.CuentaSolicitada, &d.PagoHabilitado,
			&d.PedidosTotales, &d.PedidosActivos, &d.Comensales, &d.SinIdentificar,
			&m.TotalParcial, &primero, &ultimo, &enCocina, &listos, &cocinaDesde, &listoDesde, &m.TomadoPor); err != nil {
			return nil, err
		}
		m.Estado, m.Comensales, m.ComensalesSinIdentificar = derivarMesa(d)
		m.PedidosActivos = d.PedidosActivos
		m.MinutosDesdePrimerPedido = minutosDesde(ahora, primero)
		m.MinutosDesdeUltimoPedido = minutosDesde(ahora, ultimo)
		m.Espera, m.MinutosEnEspera = derivarEspera(m.Estado, enCocina, listos,
			minutosDesde(ahora, cocinaDesde), minutosDesde(ahora, listoDesde))
		if m.TomadoPor == nil {
			m.TomadoPor = []string{}
		}
		mesas = append(mesas, m)
	}
	return mesas, rows.Err()
}

func (s *pgStore) PedidosCocina(ctx context.Context, tenantID, sucursalID string, ahora time.Time) ([]PedidoCocina, error) {
	if !reUUID.MatchString(sucursalID) || !reUUID.MatchString(tenantID) {
		return []PedidoCocina{}, nil
	}
	rows, err := db.Pool.Query(ctx, `
		SELECT p.id, m.numero, p.estado,
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
		if err := rows.Scan(&p.PedidoID, &p.Mesa, &p.Estado, &p.MinutosDesdeIngreso, &p.Items.Pendientes, &p.Items.Preparando, &p.Items.Listos); err != nil {
			return nil, err
		}
		pedidos = append(pedidos, p)
	}
	return pedidos, rows.Err()
}

// UltimaHoraCocina mide los 60 minutos previos a `ahora`: pedidos ingresados, pedidos que pasaron a
// listo y su tiempo de despacho (promedio y p90) sobre estos últimos.
func (s *pgStore) UltimaHoraCocina(ctx context.Context, tenantID, sucursalID string, ahora time.Time) (UltimaHora, error) {
	var u UltimaHora
	if !reUUID.MatchString(sucursalID) || !reUUID.MatchString(tenantID) {
		return u, nil
	}
	desde := ahora.Add(-time.Hour)
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FILTER (WHERE p.created_at >= $3)::int,
		       COUNT(*) FILTER (WHERE p.listo_at >= $3)::int,
		       ROUND((AVG(EXTRACT(EPOCH FROM (p.listo_at - p.created_at)) / 60)
		              FILTER (WHERE p.listo_at >= $3))::numeric, 1)::float8,
		       ROUND((percentile_cont(0.9) WITHIN GROUP (ORDER BY EXTRACT(EPOCH FROM (p.listo_at - p.created_at)) / 60)
		              FILTER (WHERE p.listo_at >= $3))::numeric, 1)::float8
		FROM pedidos p
		JOIN sucursales su ON su.id = p.sucursal_id
		WHERE su.tenant_id = $1 AND p.sucursal_id = $2
		  AND p.created_at >= $3 - interval '1 day'
		  AND (p.created_at >= $3 OR p.listo_at >= $3)`,
		tenantID, sucursalID, desde).Scan(&u.Ingresados60Min, &u.Listos60Min, &u.DespachoPromedioMin, &u.DespachoP90Min)
	u.Balance60Min = u.Ingresados60Min - u.Listos60Min
	return u, err
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

// ResumenPorHora agrupa los pedidos creados en [desde, hasta) por hora calendario de la zona del negocio.
// Facturación y despacho consideran solo pedidos cerrados, igual que internal/metrica.
func (s *pgStore) ResumenPorHora(ctx context.Context, tenantID, sucursalID string, desde, hasta time.Time, zona string) ([]HoraTurno, error) {
	if !reUUID.MatchString(sucursalID) || !reUUID.MatchString(tenantID) {
		return []HoraTurno{}, nil
	}
	rows, err := db.Pool.Query(ctx, `
		WITH base AS (
			SELECT p.id, p.mesa_id, p.cuenta_version, p.estado, p.created_at, p.updated_at, p.listo_at,
			       date_trunc('hour', p.created_at AT TIME ZONE $5::text) AS hora
			FROM pedidos p
			JOIN sucursales su ON su.id = p.sucursal_id
			WHERE su.tenant_id = $1 AND p.sucursal_id = $2
			  AND p.created_at >= $3 AND p.created_at < $4
		),
		totales AS (
			SELECT b.id, SUM(pi.cantidad * pi.precio_unitario) AS total
			FROM base b
			JOIN pedido_items pi ON pi.pedido_id = b.id
			WHERE b.estado = 'cerrado'
			GROUP BY b.id
		),
		por_cuenta AS (
			SELECT b.hora,
			       GREATEST(COUNT(DISTINCT pi.comensal_id),
			                CASE WHEN BOOL_OR(pi.comensal_id IS NULL) THEN 1 ELSE 0 END) AS comensales
			FROM base b
			JOIN pedido_items pi ON pi.pedido_id = b.id
			GROUP BY b.hora, b.mesa_id, b.cuenta_version
		),
		comensales AS (
			SELECT hora, SUM(comensales) AS comensales FROM por_cuenta GROUP BY hora
		),
		pedidos_hora AS (
			SELECT b.hora,
			       COUNT(*) AS pedidos,
			       COALESCE(SUM(t.total), 0)::float8 AS facturacion,
			       ROUND((AVG(EXTRACT(EPOCH FROM (COALESCE(b.listo_at, b.updated_at) - b.created_at)) / 60)
			              FILTER (WHERE b.estado = 'cerrado' AND COALESCE(b.listo_at, b.updated_at) >= b.created_at))::numeric, 1)::float8 AS despacho
			FROM base b
			LEFT JOIN totales t ON t.id = b.id
			GROUP BY b.hora
		)
		SELECT to_char(ph.hora, 'HH24:MI'), ph.pedidos::int, COALESCE(c.comensales, 0)::int, ph.facturacion, ph.despacho
		FROM pedidos_hora ph
		LEFT JOIN comensales c ON c.hora = ph.hora
		ORDER BY ph.hora`, tenantID, sucursalID, desde, hasta, zona)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	horas := []HoraTurno{}
	for rows.Next() {
		var h HoraTurno
		if err := rows.Scan(&h.Hora, &h.Pedidos, &h.Comensales, &h.Facturacion, &h.DespachoPromedioMin); err != nil {
			return nil, err
		}
		horas = append(horas, h)
	}
	return horas, rows.Err()
}
