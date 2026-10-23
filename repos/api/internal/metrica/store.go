package metrica

import (
	"context"
	"fmt"
	"time"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/db"
)

type Store interface {
	SucursalPerteneceATenant(ctx context.Context, sucursalID, tenantID string) (bool, error)
	ObtenerResumenBase(ctx context.Context, tenantID, sucursalID string, desde, hasta time.Time) (ResumenBase, error)
	ListarPlatosEstrella(ctx context.Context, tenantID, sucursalID string, desde, hasta time.Time, limite int) ([]PlatoEstrella, error)
	ListarPorDia(ctx context.Context, tenantID, sucursalID string, desde, hasta time.Time, zonaHoraria string) ([]MetricaDiaria, error)
	ListarPorTurno(ctx context.Context, tenantID, sucursalID string, desde, hasta time.Time, zonaHoraria string) ([]MetricaTurno, error)
	ListarPorEstado(ctx context.Context, tenantID, sucursalID string, desde, hasta time.Time) ([]MetricaEstado, error)
}

type pgStore struct{}

func NuevoStore() Store { return &pgStore{} }

func (s *pgStore) SucursalPerteneceATenant(ctx context.Context, sucursalID, tenantID string) (bool, error) {
	var pertenece bool
	err := db.Pool.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1
			FROM sucursales
			WHERE id = $1 AND tenant_id = $2
		)`,
		sucursalID, tenantID,
	).Scan(&pertenece)
	if err != nil {
		return false, fmt.Errorf("error validando sucursal para métricas: %w", err)
	}
	return pertenece, nil
}

func (s *pgStore) ObtenerResumenBase(
	ctx context.Context,
	tenantID, sucursalID string,
	desde, hasta time.Time,
) (ResumenBase, error) {
	const query = `
		WITH pedidos_periodo AS (
			SELECT p.id, p.estado, p.created_at, p.updated_at, p.listo_at
			FROM pedidos p
			JOIN sucursales su ON su.id = p.sucursal_id
			WHERE su.tenant_id = $1
			  AND (NULLIF($2, '')::uuid IS NULL OR p.sucursal_id = NULLIF($2, '')::uuid)
			  AND p.created_at >= $3
			  AND p.created_at < $4
		),
		totales_cerrados AS (
			SELECT pp.id,
			       COALESCE(SUM(pi.cantidad * pi.precio_unitario), 0)::double precision AS total
			FROM pedidos_periodo pp
			JOIN pedido_items pi ON pi.pedido_id = pp.id
			WHERE pp.estado = 'cerrado'
			GROUP BY pp.id
		)
		SELECT
			COALESCE((SELECT SUM(total) FROM totales_cerrados), 0)::double precision,
			COALESCE((SELECT AVG(total) FROM totales_cerrados), 0)::double precision,
			(SELECT COUNT(*) FROM pedidos_periodo),
			(SELECT COUNT(*) FROM pedidos_periodo WHERE estado = 'cerrado'),
			(SELECT COUNT(*) FROM pedidos_periodo WHERE estado <> 'cerrado'),
			COALESCE((
				SELECT AVG(EXTRACT(EPOCH FROM (COALESCE(listo_at, updated_at) - created_at)) / 60.0)
				FROM pedidos_periodo
				WHERE estado = 'cerrado'
				  AND COALESCE(listo_at, updated_at) >= created_at
			), 0)::double precision`

	var resumen ResumenBase
	err := db.Pool.QueryRow(ctx, query, tenantID, sucursalID, desde, hasta).Scan(
		&resumen.FacturacionTotal,
		&resumen.TicketPromedio,
		&resumen.PedidosTotales,
		&resumen.PedidosCerrados,
		&resumen.PedidosActivos,
		&resumen.TiempoPromedioDespachoMinutos,
	)
	if err != nil {
		return ResumenBase{}, fmt.Errorf("error calculando resumen de métricas: %w", err)
	}
	return resumen, nil
}

func (s *pgStore) ListarPlatosEstrella(
	ctx context.Context,
	tenantID, sucursalID string,
	desde, hasta time.Time,
	limite int,
) ([]PlatoEstrella, error) {
	const query = `
		SELECT a.id, a.nombre,
		       SUM(pi.cantidad)::bigint AS unidades,
		       SUM(pi.cantidad * pi.precio_unitario)::double precision AS monto
		FROM pedidos p
		JOIN sucursales su ON su.id = p.sucursal_id
		JOIN pedido_items pi ON pi.pedido_id = p.id
		JOIN articulos a ON a.id = pi.articulo_id
		WHERE su.tenant_id = $1
		  AND (NULLIF($2, '')::uuid IS NULL OR p.sucursal_id = NULLIF($2, '')::uuid)
		  AND p.created_at >= $3
		  AND p.created_at < $4
		  AND p.estado = 'cerrado'
		GROUP BY a.id, a.nombre
		ORDER BY unidades DESC, monto DESC, a.nombre
		LIMIT $5`

	rows, err := db.Pool.Query(ctx, query, tenantID, sucursalID, desde, hasta, limite)
	if err != nil {
		return nil, fmt.Errorf("error listando platos estrella: %w", err)
	}
	defer rows.Close()

	platos := make([]PlatoEstrella, 0, limite)
	for rows.Next() {
		var plato PlatoEstrella
		if err := rows.Scan(&plato.ArticuloID, &plato.Nombre, &plato.Unidades, &plato.Monto); err != nil {
			return nil, fmt.Errorf("error leyendo plato estrella: %w", err)
		}
		platos = append(platos, plato)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error recorriendo platos estrella: %w", err)
	}
	return platos, nil
}

func (s *pgStore) ListarPorDia(
	ctx context.Context,
	tenantID, sucursalID string,
	desde, hasta time.Time,
	zonaHoraria string,
) ([]MetricaDiaria, error) {
	const query = `
		SELECT (p.created_at AT TIME ZONE $5)::date AS fecha,
		       COALESCE(SUM(pi.cantidad * pi.precio_unitario) FILTER (WHERE p.estado = 'cerrado'), 0)::double precision,
		       COUNT(DISTINCT p.id)::bigint,
		       COUNT(DISTINCT p.id) FILTER (WHERE p.estado = 'cerrado')::bigint
		FROM pedidos p
		JOIN sucursales su ON su.id = p.sucursal_id
		LEFT JOIN pedido_items pi ON pi.pedido_id = p.id
		WHERE su.tenant_id = $1
		  AND (NULLIF($2, '')::uuid IS NULL OR p.sucursal_id = NULLIF($2, '')::uuid)
		  AND p.created_at >= $3
		  AND p.created_at < $4
		GROUP BY fecha
		ORDER BY fecha`

	rows, err := db.Pool.Query(ctx, query, tenantID, sucursalID, desde, hasta, zonaHoraria)
	if err != nil {
		return nil, fmt.Errorf("error agrupando métricas por día: %w", err)
	}
	defer rows.Close()

	resultado := make([]MetricaDiaria, 0)
	for rows.Next() {
		var (
			fecha time.Time
			item  MetricaDiaria
		)
		if err := rows.Scan(&fecha, &item.FacturacionTotal, &item.PedidosTotales, &item.PedidosCerrados); err != nil {
			return nil, fmt.Errorf("error leyendo métricas diarias: %w", err)
		}
		item.Fecha = fecha.Format("2006-01-02")
		resultado = append(resultado, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error recorriendo métricas diarias: %w", err)
	}
	return resultado, nil
}

func (s *pgStore) ListarPorTurno(
	ctx context.Context,
	tenantID, sucursalID string,
	desde, hasta time.Time,
	zonaHoraria string,
) ([]MetricaTurno, error) {
	const query = `
		WITH clasificados AS (
			SELECT p.id, p.estado, pi.cantidad, pi.precio_unitario,
			       CASE
				   WHEN EXTRACT(HOUR FROM p.created_at AT TIME ZONE $5) < 6 THEN 'Madrugada'
				   WHEN EXTRACT(HOUR FROM p.created_at AT TIME ZONE $5) < 12 THEN 'Mañana'
				   WHEN EXTRACT(HOUR FROM p.created_at AT TIME ZONE $5) < 16 THEN 'Mediodía'
				   WHEN EXTRACT(HOUR FROM p.created_at AT TIME ZONE $5) < 20 THEN 'Tarde'
				   ELSE 'Noche'
			       END AS turno,
			       CASE
				   WHEN EXTRACT(HOUR FROM p.created_at AT TIME ZONE $5) < 6 THEN 1
				   WHEN EXTRACT(HOUR FROM p.created_at AT TIME ZONE $5) < 12 THEN 2
				   WHEN EXTRACT(HOUR FROM p.created_at AT TIME ZONE $5) < 16 THEN 3
				   WHEN EXTRACT(HOUR FROM p.created_at AT TIME ZONE $5) < 20 THEN 4
				   ELSE 5
			       END AS orden
			FROM pedidos p
			JOIN sucursales su ON su.id = p.sucursal_id
			LEFT JOIN pedido_items pi ON pi.pedido_id = p.id
			WHERE su.tenant_id = $1
			  AND (NULLIF($2, '')::uuid IS NULL OR p.sucursal_id = NULLIF($2, '')::uuid)
			  AND p.created_at >= $3
			  AND p.created_at < $4
		)
		SELECT turno,
		       COALESCE(SUM(cantidad * precio_unitario) FILTER (WHERE estado = 'cerrado'), 0)::double precision,
		       COUNT(DISTINCT id)::bigint,
		       COUNT(DISTINCT id) FILTER (WHERE estado = 'cerrado')::bigint
		FROM clasificados
		GROUP BY turno
		ORDER BY MIN(orden)`

	rows, err := db.Pool.Query(ctx, query, tenantID, sucursalID, desde, hasta, zonaHoraria)
	if err != nil {
		return nil, fmt.Errorf("error agrupando métricas por turno: %w", err)
	}
	defer rows.Close()

	resultado := make([]MetricaTurno, 0)
	for rows.Next() {
		var item MetricaTurno
		if err := rows.Scan(&item.Turno, &item.FacturacionTotal, &item.PedidosTotales, &item.PedidosCerrados); err != nil {
			return nil, fmt.Errorf("error leyendo métricas por turno: %w", err)
		}
		resultado = append(resultado, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error recorriendo métricas por turno: %w", err)
	}
	return resultado, nil
}

func (s *pgStore) ListarPorEstado(
	ctx context.Context,
	tenantID, sucursalID string,
	desde, hasta time.Time,
) ([]MetricaEstado, error) {
	const query = `
		SELECT p.estado, COUNT(*)::bigint
		FROM pedidos p
		JOIN sucursales su ON su.id = p.sucursal_id
		WHERE su.tenant_id = $1
		  AND (NULLIF($2, '')::uuid IS NULL OR p.sucursal_id = NULLIF($2, '')::uuid)
		  AND p.created_at >= $3
		  AND p.created_at < $4
		GROUP BY p.estado
		ORDER BY CASE p.estado
			WHEN 'recibido' THEN 1
			WHEN 'confirmado' THEN 2
			WHEN 'preparando' THEN 3
			WHEN 'listo' THEN 4
			WHEN 'entregado' THEN 5
			WHEN 'cerrado' THEN 6
			ELSE 7
		END`

	rows, err := db.Pool.Query(ctx, query, tenantID, sucursalID, desde, hasta)
	if err != nil {
		return nil, fmt.Errorf("error agrupando métricas por estado: %w", err)
	}
	defer rows.Close()

	resultado := make([]MetricaEstado, 0)
	for rows.Next() {
		var item MetricaEstado
		if err := rows.Scan(&item.Estado, &item.Cantidad); err != nil {
			return nil, fmt.Errorf("error leyendo métricas por estado: %w", err)
		}
		resultado = append(resultado, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error recorriendo métricas por estado: %w", err)
	}
	return resultado, nil
}
