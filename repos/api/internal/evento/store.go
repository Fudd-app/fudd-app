package evento

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/db"
)

// Execer es lo mínimo que comparten pgx.Tx y pgxpool.Pool para insertar.
type Execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// Registrar inserta el hito usando la transacción del cambio que lo origina.
// Si falla, el llamador debe abortar la operación completa.
func Registrar(ctx context.Context, ex Execer, n Nuevo) error {
	detalle := []byte("{}")
	if len(n.Detalle) > 0 {
		b, err := json.Marshal(n.Detalle)
		if err != nil {
			return fmt.Errorf("error serializando detalle del evento: %w", err)
		}
		detalle = b
	}
	// ts usa clock_timestamp() y no now(): dentro de una misma transacción now()
	// es constante y los hitos de un mismo cambio perderían su orden.
	// Si no hay nombre, se toma del usuario; si ya no existe, queda el rol.
	_, err := ex.Exec(ctx,
		`INSERT INTO eventos_operacion (
			sucursal_id, mesa_id, cuenta_version, pedido_id, pedido_item_id,
			tipo, estado_anterior, estado_nuevo,
			actor_tipo, usuario_id, actor_nombre, actor_rol, detalle, ts
		 ) VALUES (
			$1, $2, $3, NULLIF($4, '')::uuid, NULLIF($5, '')::uuid,
			$6, NULLIF($7, ''), NULLIF($8, ''),
			$9, NULLIF($10, '')::uuid,
			COALESCE(NULLIF($11, ''), (SELECT nombre FROM usuarios WHERE id = NULLIF($10, '')::uuid AND tenant_id = (SELECT tenant_id FROM sucursales WHERE id = $1)), NULLIF($12, '')),
			NULLIF($12, ''), $13, clock_timestamp()
		 )`,
		n.SucursalID, n.MesaID, n.CuentaVersion, n.PedidoID, n.PedidoItemID,
		n.Tipo, n.EstadoAnterior, n.EstadoNuevo,
		n.Actor.Tipo, n.Actor.UsuarioID, n.Actor.Nombre, n.Actor.Rol, detalle,
	)
	if err != nil {
		return fmt.Errorf("error registrando evento %s: %w", n.Tipo, err)
	}
	return nil
}

const columnasEvento = `e.id, e.sucursal_id, e.mesa_id, e.cuenta_version, e.pedido_id, e.pedido_item_id,
	e.tipo, COALESCE(e.estado_anterior, ''), COALESCE(e.estado_nuevo, ''),
	e.actor_tipo, e.usuario_id, COALESCE(e.actor_nombre, ''), COALESCE(e.actor_rol, ''), e.detalle, e.ts`

// EventosPorMesa devuelve los hitos de una mesa del tenant, ordenados cronológicamente.
// desde/hasta en cero significan sin límite; hasta es exclusivo.
func EventosPorMesa(ctx context.Context, tenantID, sucursalID, mesaID string, desde, hasta time.Time) ([]Evento, error) {
	return consultar(ctx,
		`SELECT `+columnasEvento+`
		 FROM eventos_operacion e
		 JOIN sucursales su ON su.id = e.sucursal_id
		 WHERE su.tenant_id = $1 AND e.sucursal_id = $2 AND e.mesa_id = $3
		   AND ($4::timestamptz IS NULL OR e.ts >= $4)
		   AND ($5::timestamptz IS NULL OR e.ts < $5)
		 ORDER BY e.ts, e.id`,
		tenantID, sucursalID, mesaID, limite(desde), limite(hasta))
}

// EventosPorSucursal devuelve los hitos de una sucursal del tenant, ordenados cronológicamente.
func EventosPorSucursal(ctx context.Context, tenantID, sucursalID string, desde, hasta time.Time) ([]Evento, error) {
	return consultar(ctx,
		`SELECT `+columnasEvento+`
		 FROM eventos_operacion e
		 JOIN sucursales su ON su.id = e.sucursal_id
		 WHERE su.tenant_id = $1 AND e.sucursal_id = $2
		   AND ($3::timestamptz IS NULL OR e.ts >= $3)
		   AND ($4::timestamptz IS NULL OR e.ts < $4)
		 ORDER BY e.ts, e.id`,
		tenantID, sucursalID, limite(desde), limite(hasta))
}

func limite(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func consultar(ctx context.Context, query string, args ...any) ([]Evento, error) {
	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("error consultando eventos: %w", err)
	}
	defer rows.Close()

	eventos := make([]Evento, 0)
	for rows.Next() {
		var e Evento
		if err := rows.Scan(&e.ID, &e.SucursalID, &e.MesaID, &e.CuentaVersion, &e.PedidoID, &e.PedidoItemID,
			&e.Tipo, &e.EstadoAnterior, &e.EstadoNuevo,
			&e.ActorTipo, &e.UsuarioID, &e.ActorNombre, &e.ActorRol, &e.Detalle, &e.TS); err != nil {
			return nil, err
		}
		eventos = append(eventos, e)
	}
	return eventos, rows.Err()
}
