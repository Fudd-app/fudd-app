package platform

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/db"
	"github.com/jackc/pgx/v5"
)

type Store interface {
	ObtenerMetricasGlobales(ctx context.Context) (*StatusPlataforma, error)
	ListarTenantsConsolidado(ctx context.Context) ([]TenantResumen, error)
	ObtenerDetalleTenant(ctx context.Context, tenantID string) (*TenantResumen, []UsuarioPresencia, []Movimiento, error)
}

type pgStore struct{}

func NuevoStore() Store {
	return &pgStore{}
}

func (s *pgStore) ObtenerMetricasGlobales(ctx context.Context) (*StatusPlataforma, error) {
	st := &StatusPlataforma{
		Servicio: "mesa-click-api",
		Estado:   "ok",
		Database: "ok",
	}

	if db.Pool == nil {
		st.Database = "desconectado"
		st.Estado = "degraded"
		return st, nil
	}

	if err := db.Pool.Ping(ctx); err != nil {
		st.Database = "error: " + err.Error()
		st.Estado = "error"
		return st, nil
	}

	err := db.Pool.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM tenants),
			(SELECT COUNT(*) FROM sucursales),
			(SELECT COUNT(*) FROM mesas),
			(SELECT COUNT(*) FROM usuarios WHERE deleted_at IS NULL),
			(SELECT COUNT(*) FROM pedidos WHERE created_at >= CURRENT_DATE),
			(SELECT COALESCE(SUM(total), 0) FROM pedidos WHERE created_at >= CURRENT_DATE)
	`).Scan(
		&st.TotalTenants,
		&st.TotalSucursales,
		&st.TotalMesas,
		&st.TotalUsuarios,
		&st.PedidosHoy,
		&st.TotalVentasHoy,
	)
	if err != nil {
		return nil, fmt.Errorf("error consultando metricas globales: %w", err)
	}

	return st, nil
}

func (s *pgStore) ListarTenantsConsolidado(ctx context.Context) ([]TenantResumen, error) {
	if db.Pool == nil {
		return nil, errors.New("base de datos desconectada")
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT
			t.id, t.nombre, COALESCE(t.nombre_fantasia, ''), t.slug, COALESCE(t.rubro, ''),
			COALESCE(t.plan, 'free'), t.trial_hasta, t.created_at,
			(SELECT COUNT(*) FROM sucursales s WHERE s.tenant_id = t.id) AS total_sucursales,
			(SELECT COUNT(*) FROM mesas m JOIN sucursales s ON s.id = m.sucursal_id WHERE s.tenant_id = t.id) AS total_mesas,
			(SELECT COUNT(*) FROM mesas m JOIN sucursales s ON s.id = m.sucursal_id WHERE s.tenant_id = t.id AND m.activa = true) AS mesas_activas,
			(SELECT COUNT(*) FROM usuarios u WHERE u.tenant_id = t.id AND u.deleted_at IS NULL) AS total_usuarios,
			(SELECT COUNT(*) FROM pedidos p WHERE p.tenant_id = t.id) AS total_pedidos,
			(SELECT COALESCE(SUM(p.total), 0) FROM pedidos p WHERE p.tenant_id = t.id) AS total_ventas,
			(SELECT MAX(p.created_at) FROM pedidos p WHERE p.tenant_id = t.id) AS ultimo_pedido_at
		FROM tenants t
		ORDER BY t.created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("error listando tenants: %w", err)
	}
	defer rows.Close()

	tenantsMap := make(map[string]*TenantResumen)
	var resultado []TenantResumen

	for rows.Next() {
		var tr TenantResumen
		if err := rows.Scan(
			&tr.ID, &tr.Nombre, &tr.NombreFantasia, &tr.Slug, &tr.Rubro,
			&tr.Plan, &tr.TrialHasta, &tr.CreatedAt,
			&tr.TotalSucursales, &tr.TotalMesas, &tr.MesasActivas,
			&tr.TotalUsuarios, &tr.TotalPedidos, &tr.TotalVentas,
			&tr.UltimoPedidoAt,
		); err != nil {
			return nil, fmt.Errorf("error leyendo fila de tenant: %w", err)
		}
		tr.UsuariosPorRol = make(map[string]int)
		resultado = append(resultado, tr)
	}

	for i := range resultado {
		tenantsMap[resultado[i].ID] = &resultado[i]
	}

	// Roles de usuarios agrupados
	uRows, err := db.Pool.Query(ctx, `
		SELECT tenant_id, rol, COUNT(*)
		FROM usuarios
		WHERE deleted_at IS NULL
		GROUP BY tenant_id, rol
	`)
	if err == nil {
		defer uRows.Close()
		for uRows.Next() {
			var tid, rol string
			var cant int
			if err := uRows.Scan(&tid, &rol, &cant); err == nil {
				if tr, ok := tenantsMap[tid]; ok {
					tr.UsuariosPorRol[rol] = cant
				}
			}
		}
	}

	return resultado, nil
}

func (s *pgStore) ObtenerDetalleTenant(ctx context.Context, tenantID string) (*TenantResumen, []UsuarioPresencia, []Movimiento, error) {
	if db.Pool == nil {
		return nil, nil, nil, errors.New("base de datos desconectada")
	}

	var tr TenantResumen
	tr.UsuariosPorRol = make(map[string]int)

	err := db.Pool.QueryRow(ctx, `
		SELECT
			t.id, t.nombre, COALESCE(t.nombre_fantasia, ''), t.slug, COALESCE(t.rubro, ''),
			COALESCE(t.plan, 'free'), t.trial_hasta, t.created_at,
			(SELECT COUNT(*) FROM sucursales s WHERE s.tenant_id = t.id) AS total_sucursales,
			(SELECT COUNT(*) FROM mesas m JOIN sucursales s ON s.id = m.sucursal_id WHERE s.tenant_id = t.id) AS total_mesas,
			(SELECT COUNT(*) FROM mesas m JOIN sucursales s ON s.id = m.sucursal_id WHERE s.tenant_id = t.id AND m.activa = true) AS mesas_activas,
			(SELECT COUNT(*) FROM usuarios u WHERE u.tenant_id = t.id AND u.deleted_at IS NULL) AS total_usuarios,
			(SELECT COUNT(*) FROM pedidos p WHERE p.tenant_id = t.id) AS total_pedidos,
			(SELECT COALESCE(SUM(p.total), 0) FROM pedidos p WHERE p.tenant_id = t.id) AS total_ventas,
			(SELECT MAX(p.created_at) FROM pedidos p WHERE p.tenant_id = t.id) AS ultimo_pedido_at
		FROM tenants t
		WHERE t.id = $1
	`, tenantID).Scan(
		&tr.ID, &tr.Nombre, &tr.NombreFantasia, &tr.Slug, &tr.Rubro,
		&tr.Plan, &tr.TrialHasta, &tr.CreatedAt,
		&tr.TotalSucursales, &tr.TotalMesas, &tr.MesasActivas,
		&tr.TotalUsuarios, &tr.TotalPedidos, &tr.TotalVentas,
		&tr.UltimoPedidoAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, nil, errors.New("tenant no encontrado")
		}
		return nil, nil, nil, fmt.Errorf("error consultando tenant: %w", err)
	}

	// Usuarios del tenant
	uRows, err := db.Pool.Query(ctx, `
		SELECT u.id, u.tenant_id, u.sucursal_id, s.nombre, COALESCE(u.nombre, ''), COALESCE(u.email, ''),
		       u.username, u.rol, COALESCE(u.estado, 'activo'), u.ultimo_login_at, u.created_at
		FROM usuarios u
		LEFT JOIN sucursales s ON s.id = u.sucursal_id
		WHERE u.tenant_id = $1 AND u.deleted_at IS NULL
		ORDER BY u.created_at ASC
	`, tenantID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error consultando usuarios del tenant: %w", err)
	}
	defer uRows.Close()

	var usuarios []UsuarioPresencia
	for uRows.Next() {
		var u UsuarioPresencia
		if err := uRows.Scan(
			&u.ID, &u.TenantID, &u.SucursalID, &u.SucursalNombre,
			&u.Nombre, &u.Email, &u.Username, &u.Rol, &u.Estado,
			&u.UltimoLoginAt, &u.CreatedAt,
		); err != nil {
			return nil, nil, nil, fmt.Errorf("error leyendo usuario: %w", err)
		}
		tr.UsuariosPorRol[u.Rol]++
		usuarios = append(usuarios, u)
	}

	// Movimientos recientes (pedidos e invitaciones)
	var movimientos []Movimiento

	pRows, err := db.Pool.Query(ctx, `
		SELECT p.id, p.total, p.estado, p.origen, COALESCE(p.creado_por_nombre, 'Comensal'),
		       COALESCE(m.numero, 0), p.created_at
		FROM pedidos p
		LEFT JOIN mesas m ON m.id = p.mesa_id
		WHERE p.tenant_id = $1
		ORDER BY p.created_at DESC
		LIMIT 30
	`, tenantID)
	if err == nil {
		defer pRows.Close()
		for pRows.Next() {
			var pid, estado, origen, actor string
			var total float64
			var numMesa int
			var fecha time.Time
			if err := pRows.Scan(&pid, &total, &estado, &origen, &actor, &numMesa, &fecha); err == nil {
				desc := fmt.Sprintf("Pedido #%s en Mesa %d ($%.2f - %s)", pid[:8], numMesa, total, estado)
				movimientos = append(movimientos, Movimiento{
					ID:          pid,
					Tipo:        "pedido",
					Descripcion: desc,
					Actor:       actor,
					Fecha:       fecha,
					Detalle: map[string]any{
						"total":  total,
						"estado": estado,
						"origen": origen,
						"mesa":   numMesa,
					},
				})
			}
		}
	}

	iRows, err := db.Pool.Query(ctx, `
		SELECT i.id, i.email, i.rol, i.estado, i.invitado_por_nombre, i.created_at
		FROM invitaciones_equipo i
		WHERE i.tenant_id = $1
		ORDER BY i.created_at DESC
		LIMIT 15
	`, tenantID)
	if err == nil {
		defer iRows.Close()
		for iRows.Next() {
			var iid, email, rol, estado, actor string
			var fecha time.Time
			if err := iRows.Scan(&iid, &email, &rol, &estado, &actor, &fecha); err == nil {
				desc := fmt.Sprintf("Invitación de %s para %s (%s)", rol, email, estado)
				movimientos = append(movimientos, Movimiento{
					ID:          iid,
					Tipo:        "invitacion",
					Descripcion: desc,
					Actor:       actor,
					Fecha:       fecha,
					Detalle: map[string]any{
						"email":  email,
						"rol":    rol,
						"estado": estado,
					},
				})
			}
		}
	}

	sort.Slice(movimientos, func(i, j int) bool {
		return movimientos[i].Fecha.After(movimientos[j].Fecha)
	})

	return &tr, usuarios, movimientos, nil
}
