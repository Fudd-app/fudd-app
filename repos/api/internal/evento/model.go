package evento

import (
	"encoding/json"
	"time"
)

// Tipos de hito operativo.
const (
	TipoCuentaAbierta    = "cuenta_abierta"
	TipoPedidoCreado     = "pedido_creado"
	TipoPedidoEstado     = "pedido_estado"
	TipoItemEstado       = "item_estado"
	TipoCuentaSolicitada = "cuenta_solicitada"
	TipoPagoHabilitado   = "pago_habilitado"
	TipoPagoAprobado     = "pago_aprobado"
	TipoCuentaCerrada    = "cuenta_cerrada"
)

// Tipos de actor.
const (
	ActorUsuario     = "usuario"
	ActorComensal    = "comensal"
	ActorSistema     = "sistema"
	ActorMercadoPago = "mercadopago"
)

// Actor identifica quién originó un hito.
type Actor struct {
	Tipo      string `json:"tipo"`
	UsuarioID string `json:"usuario_id,omitempty"`
	Nombre    string `json:"nombre,omitempty"`
	Rol       string `json:"rol,omitempty"`
}

// Evento es un hito inmutable de una mesa y, opcionalmente, de un pedido o ítem.
type Evento struct {
	ID             string          `json:"id"`
	SucursalID     string          `json:"sucursal_id"`
	MesaID         string          `json:"mesa_id"`
	CuentaVersion  int             `json:"cuenta_version"`
	PedidoID       *string         `json:"pedido_id,omitempty"`
	PedidoItemID   *string         `json:"pedido_item_id,omitempty"`
	Tipo           string          `json:"tipo"`
	EstadoAnterior string          `json:"estado_anterior,omitempty"`
	EstadoNuevo    string          `json:"estado_nuevo,omitempty"`
	ActorTipo      string          `json:"actor_tipo"`
	UsuarioID      *string         `json:"usuario_id,omitempty"`
	ActorNombre    string          `json:"actor_nombre,omitempty"`
	ActorRol       string          `json:"actor_rol,omitempty"`
	Detalle        json.RawMessage `json:"detalle"`
	TS             time.Time       `json:"ts"`
}

// Nuevo son los datos para registrar un hito. Detalle es opcional y se serializa a JSON.
type Nuevo struct {
	SucursalID     string
	MesaID         string
	CuentaVersion  int
	PedidoID       string
	PedidoItemID   string
	Tipo           string
	EstadoAnterior string
	EstadoNuevo    string
	Actor          Actor
	Detalle        map[string]any
}
