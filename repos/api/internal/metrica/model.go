package metrica

import "time"

type Periodo struct {
	Desde          time.Time `json:"desde"`
	Hasta          time.Time `json:"hasta"`
	ZonaHoraria    string    `json:"zona_horaria"`
	SucursalID     string    `json:"sucursal_id,omitempty"`
	ComparadoDesde time.Time `json:"comparado_desde"`
	ComparadoHasta time.Time `json:"comparado_hasta"`
}

type ResumenBase struct {
	FacturacionTotal              float64 `json:"facturacion_total"`
	TicketPromedio                float64 `json:"ticket_promedio"`
	PedidosTotales                int64   `json:"pedidos_totales"`
	PedidosCerrados               int64   `json:"pedidos_cerrados"`
	PedidosActivos                int64   `json:"pedidos_activos"`
	TiempoPromedioDespachoMinutos float64 `json:"tiempo_promedio_despacho_minutos"`
}

type PlatoEstrella struct {
	ArticuloID string  `json:"articulo_id"`
	Nombre     string  `json:"nombre"`
	Unidades   int64   `json:"unidades"`
	Monto      float64 `json:"monto"`
}

type MetricaDiaria struct {
	Fecha            string  `json:"fecha"`
	FacturacionTotal float64 `json:"facturacion_total"`
	PedidosTotales   int64   `json:"pedidos_totales"`
	PedidosCerrados  int64   `json:"pedidos_cerrados"`
}

type MetricaTurno struct {
	Turno            string  `json:"turno"`
	FacturacionTotal float64 `json:"facturacion_total"`
	PedidosTotales   int64   `json:"pedidos_totales"`
	PedidosCerrados  int64   `json:"pedidos_cerrados"`
}

type MetricaEstado struct {
	Estado   string `json:"estado"`
	Cantidad int64  `json:"cantidad"`
}

type MetricaCategoria struct {
	CategoriaID string  `json:"categoria_id"`
	Nombre      string  `json:"nombre"`
	Unidades    int64   `json:"unidades"`
	Monto       float64 `json:"monto"`
	Porcentaje  float64 `json:"porcentaje"`
}

type CeldaCalor struct {
	DiaSemana int     `json:"dia_semana"`
	Hora      int     `json:"hora"`
	Pedidos   int64   `json:"pedidos"`
	Monto     float64 `json:"monto"`
}

type Variaciones struct {
	FacturacionTotal              *float64 `json:"facturacion_total"`
	TicketPromedio                *float64 `json:"ticket_promedio"`
	PedidosTotales                *float64 `json:"pedidos_totales"`
	TiempoPromedioDespachoMinutos *float64 `json:"tiempo_promedio_despacho_minutos"`
}

type Resumen struct {
	Periodo         Periodo            `json:"periodo"`
	ResumenBase
	PlatoMasVendido *PlatoEstrella     `json:"plato_mas_vendido"`
	PlatosEstrella  []PlatoEstrella    `json:"platos_estrella"`
	Variaciones     Variaciones        `json:"variaciones"`
	PorDia          []MetricaDiaria    `json:"por_dia"`
	PorTurno        []MetricaTurno     `json:"por_turno"`
	PorEstado       []MetricaEstado    `json:"por_estado"`
	PorCategoria    []MetricaCategoria `json:"por_categoria"`
	MapaCalor       []CeldaCalor       `json:"mapa_calor"`
}
