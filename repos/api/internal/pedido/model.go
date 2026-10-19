package pedido

import (
	"errors"
	"time"
)

var (
	ErrNotFound         = errors.New("pedido no encontrado")
	ErrValidation       = errors.New("validación fallida")
	ErrMesaCerrada      = errors.New("mesa cerrada")
	ErrCuentaSolicitada = errors.New("cuenta solicitada")
	ErrPedidoCerrado    = errors.New("pedido cerrado")
)

var EstadosValidos = []string{"recibido", "preparando", "listo", "cerrado"}
var EstadosItemValidos = []string{"pendiente", "preparando", "listo"}

type Pedido struct {
	ID                 string       `json:"id"`
	MesaID             string       `json:"mesa_id"`
	SucursalID         string       `json:"sucursal_id"`
	CuentaVersion      int          `json:"cuenta_version"`
	Estado             string       `json:"estado"`
	Origen             string       `json:"origen"`
	CreadoPorUsuarioID *string      `json:"creado_por_usuario_id,omitempty"`
	CreadoPorNombre    *string      `json:"creado_por_nombre,omitempty"`
	Items              []PedidoItem `json:"items,omitempty"`
	CreatedAt          time.Time    `json:"created_at"`
	UpdatedAt          time.Time    `json:"updated_at"`
}

type PedidoItemVariante struct {
	VarianteID      string  `json:"variante_id"`
	Nombre          string  `json:"nombre"`
	PrecioAdicional float64 `json:"precio_adicional"`
}

type PedidoItem struct {
	ID             string               `json:"id"`
	PedidoID       string               `json:"pedido_id"`
	ArticuloID     string               `json:"articulo_id"`
	NombreArticulo string               `json:"nombre_articulo,omitempty"`
	Cantidad       int                  `json:"cantidad"`
	PrecioUnitario float64              `json:"precio_unitario"`
	Notas          string               `json:"notas,omitempty"`
	ComensalID     string               `json:"comensal_id,omitempty"`
	ComensalNombre string               `json:"comensal_nombre,omitempty"`
	Estado         string               `json:"estado"`
	Variantes      []PedidoItemVariante `json:"variantes,omitempty"`
}

type NuevoPedidoInput struct {
	MesaID             string           `json:"mesa_id"`
	Items              []NuevoItemInput `json:"items"`
	Origen             string           `json:"-"`
	CreadoPorUsuarioID string           `json:"-"`
	CreadoPorNombre    string           `json:"-"`
}

type PedidoAsistidoInput struct {
	MesaID       string           `json:"mesa_id"`
	AliasCliente string           `json:"alias_cliente,omitempty"`
	Items        []NuevoItemInput `json:"items"`
}

type NuevoItemInput struct {
	ArticuloID     string   `json:"articulo_id"`
	Cantidad       int      `json:"cantidad"`
	Notas          string   `json:"notas"`
	ComensalID     string   `json:"comensal_id"`
	ComensalNombre string   `json:"comensal_nombre"`
	Variantes      []string `json:"variantes,omitempty"`
}

type CambiarEstadoInput struct {
	Estado string `json:"estado"`
}

type CambiarEstadoItemInput struct {
	Estado string `json:"estado"`
}
