package pedido

import (
	"context"
	"errors"
	"fmt"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/db"
	"github.com/jackc/pgx/v5"
)

type Store interface {
	Crear(ctx context.Context, input NuevoPedidoInput, sucursalID string, cuentaVersion int) (*Pedido, error)
	ListarActivos(ctx context.Context, sucursalID, tenantID string) ([]Pedido, error)
	ListarCuentaActualPorQR(ctx context.Context, qrToken string) ([]Pedido, error)
	CambiarEstado(ctx context.Context, id, tenantID, sucursalAlcance, nuevoEstado string) (*Pedido, error)
	CambiarEstadoItem(ctx context.Context, itemID, tenantID, sucursalAlcance, nuevoEstado string) (*Pedido, error)
	SucursalPerteneceATenant(ctx context.Context, sucursalID, tenantID string) (bool, error)
	ObtenerSucursalPorMesa(ctx context.Context, mesaID string) (string, int, error)
}

type pgStore struct{}

func NuevoStore() Store { return &pgStore{} }

func (s *pgStore) ObtenerSucursalPorMesa(ctx context.Context, mesaID string) (string, int, error) {
	var sucursalID string
	var estado string
	var cuentaSolicitada bool
	var cuentaVersion int
	err := db.Pool.QueryRow(ctx,
		`SELECT sucursal_id, estado, cuenta_solicitada, cuenta_version FROM mesas WHERE id = $1`, mesaID,
	).Scan(&sucursalID, &estado, &cuentaSolicitada, &cuentaVersion)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", 0, ErrNotFound
		}
		return "", 0, fmt.Errorf("error obteniendo sucursal por mesa: %w", err)
	}
	if estado != "activa" {
		return "", 0, ErrMesaCerrada
	}
	if cuentaSolicitada {
		return "", 0, ErrCuentaSolicitada
	}
	return sucursalID, cuentaVersion, nil
}

func (s *pgStore) Crear(ctx context.Context, input NuevoPedidoInput, sucursalID string, cuentaVersion int) (*Pedido, error) {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	p := &Pedido{}
	err = tx.QueryRow(ctx,
		`INSERT INTO pedidos (mesa_id, sucursal_id, cuenta_version, estado, origen, creado_por_usuario_id, creado_por_nombre)
		 SELECT $1, $2, $3, 'recibido', $4, NULLIF($5, '')::uuid,
		        COALESCE((SELECT nombre FROM usuarios WHERE id=NULLIF($5, '')::uuid), NULLIF($6, ''))
		 WHERE EXISTS (
			SELECT 1
			FROM mesas
			WHERE id = $1
			  AND sucursal_id = $2
			  AND estado = 'activa'
			  AND cuenta_solicitada = false
			  AND cuenta_version = $3
		 )
		 RETURNING id, mesa_id, sucursal_id, cuenta_version, estado, origen, creado_por_usuario_id, creado_por_nombre, created_at, updated_at`,
		input.MesaID, sucursalID, cuentaVersion, input.Origen, input.CreadoPorUsuarioID, input.CreadoPorNombre,
	).Scan(&p.ID, &p.MesaID, &p.SucursalID, &p.CuentaVersion, &p.Estado, &p.Origen, &p.CreadoPorUsuarioID, &p.CreadoPorNombre, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCuentaSolicitada
		}
		return nil, fmt.Errorf("error creando pedido: %w", err)
	}

	var tenantID string
	err = tx.QueryRow(ctx,
		`SELECT tenant_id FROM sucursales WHERE id = $1`, sucursalID,
	).Scan(&tenantID)
	if err != nil {
		return nil, fmt.Errorf("sucursal no encontrada: %w", ErrNotFound)
	}

	for _, item := range input.Items {
		var nombreArticulo string
		var precioUnitario float64
		err = tx.QueryRow(ctx,
			`SELECT nombre, precio FROM articulos WHERE id = $1 AND tenant_id = $2 AND activo = true`,
			item.ArticuloID, tenantID,
		).Scan(&nombreArticulo, &precioUnitario)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, fmt.Errorf("artículo %s no encontrado en la carta: %w", item.ArticuloID, ErrValidation)
			}
			return nil, fmt.Errorf("error obteniendo artículo: %w", err)
		}

		var itemVariantes []PedidoItemVariante
		if len(item.Variantes) > 0 {
			vRows, err := tx.Query(ctx,
				`SELECT v.id, v.nombre, v.precio_adicional
				 FROM variantes v
				 JOIN articulos a ON a.id = v.articulo_id
				 WHERE v.id = ANY($1) AND v.articulo_id = $2 AND a.tenant_id = $3`,
				item.Variantes, item.ArticuloID, tenantID,
			)
			if err != nil {
				return nil, fmt.Errorf("error validando variantes: %w", err)
			}

			for vRows.Next() {
				var piv PedidoItemVariante
				if err := vRows.Scan(&piv.VarianteID, &piv.Nombre, &piv.PrecioAdicional); err != nil {
					vRows.Close()
					return nil, err
				}
				itemVariantes = append(itemVariantes, piv)
			}
			errRows := vRows.Err()
			vRows.Close()
			if errRows != nil {
				return nil, errRows
			}
			if len(itemVariantes) != len(item.Variantes) {
				return nil, fmt.Errorf("variantes inválidas para artículo %s: %w", item.ArticuloID, ErrValidation)
			}

			for _, v := range itemVariantes {
				precioUnitario += v.PrecioAdicional
			}
		}

		var itemID string
		err = tx.QueryRow(ctx,
			`INSERT INTO pedido_items (
				pedido_id, articulo_id, cantidad, precio_unitario, notas, comensal_id, comensal_nombre
			)
			 VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, '')) RETURNING id`,
			p.ID, item.ArticuloID, item.Cantidad, precioUnitario, item.Notas,
			item.ComensalID, item.ComensalNombre,
		).Scan(&itemID)
		if err != nil {
			return nil, fmt.Errorf("error insertando item: %w", err)
		}

		for _, v := range itemVariantes {
			_, err = tx.Exec(ctx,
				`INSERT INTO pedido_item_variantes (pedido_item_id, variante_id) VALUES ($1, $2)`,
				itemID, v.VarianteID,
			)
			if err != nil {
				return nil, fmt.Errorf("error guardando variante del item: %w", err)
			}
		}

		p.Items = append(p.Items, PedidoItem{
			ID:             itemID,
			PedidoID:       p.ID,
			ArticuloID:     item.ArticuloID,
			NombreArticulo: nombreArticulo,
			Cantidad:       item.Cantidad,
			PrecioUnitario: precioUnitario,
			Notas:          item.Notas,
			ComensalID:     item.ComensalID,
			ComensalNombre: item.ComensalNombre,
			Estado:         "pendiente",
			Variantes:      itemVariantes,
		})
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *pgStore) ListarActivos(ctx context.Context, sucursalID, tenantID string) ([]Pedido, error) {
	return s.listarPedidos(ctx,
		`SELECT p.id, p.mesa_id, p.sucursal_id, p.cuenta_version, p.estado, p.origen, p.creado_por_usuario_id, p.creado_por_nombre, p.created_at, p.updated_at
		 FROM pedidos p
		 JOIN sucursales su ON su.id = p.sucursal_id
		 JOIN mesas m ON m.id = p.mesa_id
		 WHERE p.sucursal_id = $1
		   AND su.tenant_id = $2
		   AND p.cuenta_version = m.cuenta_version
		 ORDER BY p.created_at`, sucursalID, tenantID)
}

func (s *pgStore) ListarCuentaActualPorQR(ctx context.Context, qrToken string) ([]Pedido, error) {
	return s.listarPedidos(ctx,
		`SELECT p.id, p.mesa_id, p.sucursal_id, p.cuenta_version, p.estado, p.origen, p.creado_por_usuario_id, p.creado_por_nombre, p.created_at, p.updated_at
		 FROM pedidos p
		 JOIN mesas m ON m.id = p.mesa_id
		 WHERE m.qr_token = $1
		   AND p.cuenta_version = m.cuenta_version
		 ORDER BY p.created_at`, qrToken)
}

func (s *pgStore) listarPedidos(ctx context.Context, query string, args ...any) ([]Pedido, error) {
	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pedidos := make([]Pedido, 0)
	for rows.Next() {
		var p Pedido
		if err := rows.Scan(
			&p.ID,
			&p.MesaID,
			&p.SucursalID,
			&p.CuentaVersion,
			&p.Estado,
			&p.Origen,
			&p.CreadoPorUsuarioID,
			&p.CreadoPorNombre,
			&p.CreatedAt,
			&p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		pedidos = append(pedidos, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := s.cargarItemsDePedidos(ctx, pedidos); err != nil {
		return nil, err
	}
	return pedidos, nil
}

// cargarItemsDePedidos carga todos los ítems y variantes de un lote de pedidos en 2 consultas SQL (evitando N+1).
func (s *pgStore) cargarItemsDePedidos(ctx context.Context, pedidos []Pedido) error {
	if len(pedidos) == 0 {
		return nil
	}

	pedidoIDs := make([]string, len(pedidos))
	pedidoMap := make(map[string]*Pedido, len(pedidos))
	for i := range pedidos {
		pedidoIDs[i] = pedidos[i].ID
		pedidoMap[pedidos[i].ID] = &pedidos[i]
	}

	rows, err := db.Pool.Query(ctx,
		`SELECT pi.id, pi.pedido_id, pi.articulo_id, a.nombre,
		        pi.cantidad, pi.precio_unitario, COALESCE(pi.notas, ''),
		        COALESCE(pi.comensal_id::text, ''), COALESCE(pi.comensal_nombre, ''),
		        pi.estado
		 FROM pedido_items pi
		 JOIN articulos a ON a.id = pi.articulo_id
		 WHERE pi.pedido_id = ANY($1)
		 ORDER BY pi.id`, pedidoIDs)
	if err != nil {
		return err
	}
	defer rows.Close()

	var itemIDs []string
	for rows.Next() {
		var item PedidoItem
		if err := rows.Scan(
			&item.ID,
			&item.PedidoID,
			&item.ArticuloID,
			&item.NombreArticulo,
			&item.Cantidad,
			&item.PrecioUnitario,
			&item.Notas,
			&item.ComensalID,
			&item.ComensalNombre,
			&item.Estado,
		); err != nil {
			return err
		}
		if p, ok := pedidoMap[item.PedidoID]; ok {
			p.Items = append(p.Items, item)
			itemIDs = append(itemIDs, item.ID)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if len(itemIDs) == 0 {
		return nil
	}

	itemMap := make(map[string]*PedidoItem, len(itemIDs))
	for i := range pedidos {
		for j := range pedidos[i].Items {
			itemMap[pedidos[i].Items[j].ID] = &pedidos[i].Items[j]
		}
	}

	vRows, err := db.Pool.Query(ctx,
		`SELECT piv.pedido_item_id, v.id, v.nombre, v.precio_adicional
		 FROM pedido_item_variantes piv
		 JOIN variantes v ON v.id = piv.variante_id
		 WHERE piv.pedido_item_id = ANY($1)
		 ORDER BY v.grupo NULLS LAST, v.orden, v.nombre`,
		itemIDs,
	)
	if err != nil {
		return fmt.Errorf("error listando variantes de items: %w", err)
	}
	defer vRows.Close()

	for vRows.Next() {
		var (
			itemID  string
			pivItem PedidoItemVariante
		)
		if err := vRows.Scan(&itemID, &pivItem.VarianteID, &pivItem.Nombre, &pivItem.PrecioAdicional); err != nil {
			return err
		}
		if pi, ok := itemMap[itemID]; ok {
			pi.Variantes = append(pi.Variantes, pivItem)
		}
	}
	return vRows.Err()
}

func (s *pgStore) CambiarEstado(ctx context.Context, id, tenantID, sucursalAlcance, nuevoEstado string) (*Pedido, error) {
	p := &Pedido{}
	err := db.Pool.QueryRow(ctx,
		`UPDATE pedidos
		 SET estado = $1,
		     updated_at = now(),
		     listo_at = CASE
		       WHEN $1 = 'listo' THEN COALESCE(listo_at, now())
		       WHEN $1 IN ('recibido', 'preparando') THEN NULL
		       ELSE listo_at
		     END
		 WHERE id = $2
		   AND sucursal_id IN (SELECT id FROM sucursales WHERE tenant_id = $3)
		   AND ($4 = '' OR sucursal_id = $4::uuid)
		 RETURNING id, mesa_id, sucursal_id, cuenta_version, estado, origen, creado_por_usuario_id, creado_por_nombre, created_at, updated_at`,
		nuevoEstado, id, tenantID, sucursalAlcance,
	).Scan(&p.ID, &p.MesaID, &p.SucursalID, &p.CuentaVersion, &p.Estado, &p.Origen, &p.CreadoPorUsuarioID, &p.CreadoPorNombre, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("error cambiando estado del pedido: %w", err)
	}
	return p, nil
}

func (s *pgStore) CambiarEstadoItem(ctx context.Context, itemID, tenantID, sucursalAlcance, nuevoEstado string) (*Pedido, error) {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("error iniciando actualización de item: %w", err)
	}
	defer tx.Rollback(ctx)

	var pedidoID, estadoPedido string
	err = tx.QueryRow(ctx,
		`SELECT p.id, p.estado
		 FROM pedido_items pi
		 JOIN pedidos p ON p.id = pi.pedido_id
		 JOIN sucursales su ON su.id = p.sucursal_id
		 WHERE pi.id = $1 AND su.tenant_id = $2 AND ($3 = '' OR p.sucursal_id = $3::uuid)
		 FOR UPDATE OF p`,
		itemID, tenantID, sucursalAlcance,
	).Scan(&pedidoID, &estadoPedido)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("error obteniendo pedido del item: %w", err)
	}
	if estadoPedido == "cerrado" {
		return nil, ErrPedidoCerrado
	}

	resultado, err := tx.Exec(ctx,
		`UPDATE pedido_items SET estado = $1 WHERE id = $2`,
		nuevoEstado, itemID,
	)
	if err != nil {
		return nil, fmt.Errorf("error cambiando estado del item: %w", err)
	}
	if resultado.RowsAffected() == 0 {
		return nil, ErrNotFound
	}

	var total, listos, iniciados int
	err = tx.QueryRow(ctx,
		`SELECT COUNT(*),
		        COUNT(*) FILTER (WHERE estado = 'listo'),
		        COUNT(*) FILTER (WHERE estado IN ('preparando', 'listo'))
		 FROM pedido_items
		 WHERE pedido_id = $1`,
		pedidoID,
	).Scan(&total, &listos, &iniciados)
	if err != nil {
		return nil, fmt.Errorf("error calculando estado agregado del pedido: %w", err)
	}

	estadoAgregado := resolverEstadoPedido(total, listos, iniciados)
	_, err = tx.Exec(ctx,
		`UPDATE pedidos
		 SET estado = $1,
		     updated_at = now(),
		     listo_at = CASE
		       WHEN $1 = 'listo' THEN COALESCE(listo_at, now())
		       ELSE NULL
		     END
		 WHERE id = $2 AND estado IS DISTINCT FROM $1`,
		estadoAgregado, pedidoID,
	)
	if err != nil {
		return nil, fmt.Errorf("error actualizando estado agregado del pedido: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("error confirmando estado del item: %w", err)
	}

	return s.obtenerPedidoPorID(ctx, pedidoID, tenantID)
}

func resolverEstadoPedido(total, listos, iniciados int) string {
	if total > 0 && listos == total {
		return "listo"
	}
	if iniciados > 0 {
		return "preparando"
	}
	return "recibido"
}

func (s *pgStore) obtenerPedidoPorID(ctx context.Context, pedidoID, tenantID string) (*Pedido, error) {
	pedidos, err := s.listarPedidos(ctx,
		`SELECT p.id, p.mesa_id, p.sucursal_id, p.cuenta_version, p.estado, p.origen, p.creado_por_usuario_id, p.creado_por_nombre, p.created_at, p.updated_at
		 FROM pedidos p
		 JOIN sucursales su ON su.id = p.sucursal_id
		 WHERE p.id = $1 AND su.tenant_id = $2`,
		pedidoID, tenantID,
	)
	if err != nil {
		return nil, err
	}
	if len(pedidos) == 0 {
		return nil, ErrNotFound
	}
	return &pedidos[0], nil
}

func (s *pgStore) SucursalPerteneceATenant(ctx context.Context, sucursalID, tenantID string) (bool, error) {
	var pertenece bool
	err := db.Pool.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM sucursales WHERE id = $1 AND tenant_id = $2
		)`,
		sucursalID, tenantID,
	).Scan(&pertenece)
	if err != nil {
		return false, fmt.Errorf("error validando sucursal: %w", err)
	}
	return pertenece, nil
}
