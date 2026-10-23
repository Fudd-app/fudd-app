package mcp

import (
	"errors"
	"time"
)

var ErrSucursalNoEncontrada = errors.New("sucursal no encontrada")

const (
	MesaLibre            = "libre"
	MesaOcupada          = "ocupada"
	MesaCuentaSolicitada = "cuenta_solicitada"
	MesaPagoHabilitado   = "pago_habilitado"
)

type SucursalRef struct {
	ID     string `json:"id"`
	Nombre string `json:"nombre"`
}

type SucursalResumen struct {
	ID           string `json:"id"`
	Nombre       string `json:"nombre"`
	MesasActivas int    `json:"mesas_activas"`
}

type MesaEstado struct {
	Numero                   int     `json:"numero"`
	Sector                   string  `json:"sector"`
	Capacidad                int     `json:"capacidad"`
	Estado                   string  `json:"estado"`
	Comensales               int     `json:"comensales"`
	ComensalesSinIdentificar bool    `json:"comensales_sin_identificar"`
	PedidosActivos           int     `json:"pedidos_activos"`
	MinutosDesdePrimerPedido *int    `json:"minutos_desde_primer_pedido"`
	TotalParcial             float64 `json:"total_parcial"`
}

type PedidoCocina struct {
	Mesa            int    `json:"mesa"`
	Estado          string `json:"estado"`
	Minutos         int    `json:"minutos"`
	ItemsPendientes int    `json:"items_pendientes"`
	ItemsPreparando int    `json:"items_preparando"`
	ItemsListos     int    `json:"items_listos"`
}

// datosMesa son los agregados crudos de la cuenta actual de una mesa.
type datosMesa struct {
	CuentaSolicitada bool
	PagoHabilitado   bool
	PedidosTotales   int
	PedidosActivos   int
	Comensales       int
	SinIdentificar   bool
	MinutosPrimero   *int
}

// derivarMesa aplica las reglas de estado y comensales del salón.
func derivarMesa(d datosMesa) (estado string, comensales int, sinIdentificar bool) {
	estado = MesaLibre
	if d.PedidosTotales > 0 {
		switch {
		case d.PagoHabilitado:
			estado = MesaPagoHabilitado
		case d.CuentaSolicitada:
			estado = MesaCuentaSolicitada
		case d.PedidosActivos > 0:
			estado = MesaOcupada
		}
	}
	comensales = d.Comensales
	if d.SinIdentificar {
		sinIdentificar = true
		if comensales < 1 {
			comensales = 1
		}
	}
	return estado, comensales, sinIdentificar
}

// Salidas de las tools.

type SalidaSucursales struct {
	Sucursales []SucursalResumen `json:"sucursales"`
}

type TotalesSalon struct {
	Mesas      int `json:"mesas"`
	Ocupadas   int `json:"ocupadas"`
	Comensales int `json:"comensales"`
}

type SalidaSalon struct {
	Sucursal SucursalRef  `json:"sucursal"`
	Mesas    []MesaEstado `json:"mesas"`
	Totales  TotalesSalon `json:"totales"`
}

type PedidoDemorado struct {
	Mesa    int    `json:"mesa"`
	Estado  string `json:"estado"`
	Minutos int    `json:"minutos"`
}

type ResumenCocina struct {
	Recibidos                  int             `json:"recibidos"`
	Confirmados                int             `json:"confirmados"`
	Preparando                 int             `json:"preparando"`
	Listos                     int             `json:"listos"`
	PedidoMasDemorado          *PedidoDemorado `json:"pedido_mas_demorado"`
	DespachoPromedioUltimaHora *float64        `json:"despacho_promedio_ultima_hora_min"`
}

type SalidaCocina struct {
	Sucursal SucursalRef    `json:"sucursal"`
	Pedidos  []PedidoCocina `json:"pedidos"`
	Resumen  ResumenCocina  `json:"resumen"`
}

type SalidaTurno struct {
	Sucursal            SucursalRef `json:"sucursal"`
	Desde               time.Time   `json:"desde"`
	Hasta               time.Time   `json:"hasta"`
	ZonaHoraria         string      `json:"zona_horaria"`
	Pedidos             int64       `json:"pedidos"`
	PedidosCerrados     int64       `json:"pedidos_cerrados"`
	ComensalesAtendidos int         `json:"comensales_atendidos"`
	DespachoPromedioMin *float64    `json:"despacho_promedio_min"`
	Facturacion         float64     `json:"facturacion"`
	TicketPromedio      float64     `json:"ticket_promedio"`
}
