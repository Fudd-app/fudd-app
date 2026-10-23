package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var ErrSucursalNoEncontrada = errors.New("sucursal no encontrada")

const (
	MesaLibre            = "libre"
	MesaOcupada          = "ocupada"
	MesaCuentaSolicitada = "cuenta_solicitada"
	MesaPagoHabilitado   = "pago_habilitado"

	EsperaComida  = "comida"
	EsperaEntrega = "entrega"
	EsperaCobro   = "cobro"

	// TomadorQR es el valor de tomado_por para pedidos cargados por el comensal.
	TomadorQR = "QR"

	monedaARS = "ARS"
)

// Códigos de los errores estructurados que ve el modelo.
const (
	CodigoSucursalRequerida    = "sucursal_requerida"
	CodigoSucursalNoEncontrada = "sucursal_no_encontrada"
	CodigoPlanLimitReached     = "PLAN_LIMIT_REACHED"
	CodigoRateLimited          = "rate_limited"
	CodigoErrorInterno         = "error_interno"
)

// errTool es un error que el modelo puede leer: su texto es el JSON
// {"error","mensaje","detalle"}; viaja solo como texto, sin structuredContent.
type errTool struct {
	Codigo  string
	Mensaje string
	Detalle any
}

func (e *errTool) Error() string {
	cuerpo, _ := json.Marshal(struct {
		Error   string `json:"error"`
		Mensaje string `json:"mensaje"`
		Detalle any    `json:"detalle,omitempty"`
	}{e.Codigo, e.Mensaje, e.Detalle})
	return string(cuerpo)
}

type SucursalRef struct {
	ID     string `json:"id"`
	Nombre string `json:"nombre"`
}

// SucursalResumen es lo que el store devuelve de cada sucursal; Horarios es el JSONB crudo.
type SucursalResumen struct {
	ID           string          `json:"id"`
	Nombre       string          `json:"nombre"`
	MesasTotales int             `json:"mesas_totales"`
	MesasActivas int             `json:"mesas_activas"`
	Horarios     json.RawMessage `json:"-"`
}

type MesaEstado struct {
	Numero                   int      `json:"mesa"`
	Sector                   string   `json:"sector"`
	Capacidad                int      `json:"capacidad"`
	Estado                   string   `json:"estado"`
	Comensales               int      `json:"comensales"`
	ComensalesSinIdentificar bool     `json:"comensales_sin_identificar"`
	PedidosActivos           int      `json:"pedidos_activos"`
	TotalParcial             float64  `json:"total_parcial"`
	MinutosDesdePrimerPedido *int     `json:"minutos_desde_primer_pedido"`
	MinutosDesdeUltimoPedido *int     `json:"minutos_desde_ultimo_pedido"`
	Espera                   *string  `json:"espera"`
	MinutosEnEspera          *int     `json:"minutos_en_espera"`
	TomadoPor                []string `json:"tomado_por"`
}

type ItemsPedido struct {
	Pendientes int `json:"pendientes"`
	Preparando int `json:"preparando"`
	Listos     int `json:"listos"`
}

type PedidoCocina struct {
	PedidoID            string      `json:"pedido_id"`
	Mesa                int         `json:"mesa"`
	Estado              string      `json:"estado"`
	MinutosDesdeIngreso int         `json:"minutos_desde_ingreso"`
	Items               ItemsPedido `json:"items"`
}

// UltimaHora agrega la actividad de cocina de los últimos 60 minutos.
type UltimaHora struct {
	Ingresados60Min     int      `json:"ingresados_60min"`
	Listos60Min         int      `json:"listos_60min"`
	Balance60Min        int      `json:"balance_60min"`
	DespachoPromedioMin *float64 `json:"despacho_promedio_min"`
	DespachoP90Min      *float64 `json:"despacho_p90_min"`
}

type HoraTurno struct {
	Hora                string   `json:"hora"`
	Pedidos             int      `json:"pedidos"`
	Comensales          int      `json:"comensales"`
	Facturacion         float64  `json:"facturacion"`
	DespachoPromedioMin *float64 `json:"despacho_promedio_min"`
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

// derivarEspera dice qué espera una mesa. cobro no tiene timestamp: minutos nil.
func derivarEspera(estado string, enCocina, listos int, minutosCocina, minutosListo *int) (*string, *int) {
	var espera string
	switch {
	case estado == MesaCuentaSolicitada || estado == MesaPagoHabilitado:
		espera, minutosCocina = EsperaCobro, nil
	case enCocina > 0:
		espera = EsperaComida
	case listos > 0:
		espera, minutosCocina = EsperaEntrega, minutosListo
	default:
		return nil, nil
	}
	return &espera, minutosCocina
}

// minutosDesde devuelve los minutos enteros transcurridos, o nil si no hay timestamp.
func minutosDesde(ahora time.Time, desde *time.Time) *int {
	if desde == nil {
		return nil
	}
	m := int(ahora.Sub(*desde) / time.Minute)
	if m < 0 {
		m = 0
	}
	return &m
}

// cantidad arma "1 mesa" / "3 mesas".
func cantidad(n int, singular, plural string) string {
	if n == 1 {
		return "1 " + singular
	}
	return strconv.Itoa(n) + " " + plural
}

// Formato de sucursales.horarios (ver ConfiguracionSection.tsx):
// {"lunes":{"abierto":true,"tramos":[{"apertura":"08:00","cierre":"00:00"}]}, ...}
var diasHorarios = [7]string{"domingo", "lunes", "martes", "miercoles", "jueves", "viernes", "sabado"}

type tramoHorario struct {
	Apertura string `json:"apertura"`
	Cierre   string `json:"cierre"`
}

type diaHorario struct {
	Abierto bool           `json:"abierto"`
	Tramos  []tramoHorario `json:"tramos"`
}

func minutosDelDia(hhmm string) (int, bool) {
	h, m, ok := strings.Cut(strings.TrimSpace(hhmm), ":")
	if !ok {
		return 0, false
	}
	hora, err1 := strconv.Atoi(h)
	min, err2 := strconv.Atoi(m)
	if err1 != nil || err2 != nil || hora < 0 || hora > 24 || min < 0 || min > 59 {
		return 0, false
	}
	return hora*60 + min, true
}

// abiertaAhora dice si la sucursal está abierta en `ahora` (ya en la zona del negocio).
// Devuelve nil si no hay horarios interpretables. Un tramo con cierre <= apertura
// (ej. 20:00-02:00 u 08:00-00:00) cruza la medianoche: su cola cuenta en el día siguiente.
func abiertaAhora(horarios []byte, ahora time.Time) *bool {
	if len(horarios) == 0 {
		return nil
	}
	var dias map[string]diaHorario
	if json.Unmarshal(horarios, &dias) != nil {
		return nil
	}
	conocido := false
	for _, d := range diasHorarios {
		if _, ok := dias[d]; ok {
			conocido = true
		}
	}
	if !conocido {
		return nil
	}

	actual := ahora.Hour()*60 + ahora.Minute()
	abierta := false
	hoy := dias[diasHorarios[ahora.Weekday()]]
	ayer := dias[diasHorarios[(int(ahora.Weekday())+6)%7]]
	for _, t := range hoy.Tramos {
		a, okA := minutosDelDia(t.Apertura)
		c, okC := minutosDelDia(t.Cierre)
		if !hoy.Abierto || !okA || !okC {
			continue
		}
		if (c > a && actual >= a && actual < c) || (c <= a && actual >= a) {
			abierta = true
		}
	}
	for _, t := range ayer.Tramos {
		a, okA := minutosDelDia(t.Apertura)
		c, okC := minutosDelDia(t.Cierre)
		if !ayer.Abierto || !okA || !okC {
			continue
		}
		if c <= a && actual < c {
			abierta = true
		}
	}
	return &abierta
}

func formatoTS(t time.Time) string { return t.Format(time.RFC3339) }

func formatoMonto(v float64) string { return fmt.Sprintf("$%.2f %s", v, monedaARS) }

// Salidas de las tools. Todas llevan `resumen` (una línea) y `ts`; el TextContent es este mismo JSON.

type SucursalItem struct {
	ID           string `json:"id"`
	Nombre       string `json:"nombre"`
	MesasTotales int    `json:"mesas_totales"`
	MesasActivas int    `json:"mesas_activas"`
	AbiertaAhora *bool  `json:"abierta_ahora"`
}

type SalidaSucursales struct {
	Resumen    string         `json:"resumen"`
	TS         string         `json:"ts"`
	Sucursales []SucursalItem `json:"sucursales"`
}

type Esperando struct {
	Comida  int `json:"comida"`
	Entrega int `json:"entrega"`
	Cobro   int `json:"cobro"`
}

type TotalesSalon struct {
	MesasOcupadas  int       `json:"mesas_ocupadas"`
	MesasTotales   int       `json:"mesas_totales"`
	Comensales     int       `json:"comensales"`
	MesasEsperando Esperando `json:"mesas_esperando"`
}

type TomadorSalon struct {
	Tomador        string    `json:"tomador"`
	Mesas          int       `json:"mesas"`
	Comensales     int       `json:"comensales"`
	MesasEsperando Esperando `json:"mesas_esperando"`
}

type SalidaSalon struct {
	Resumen        string         `json:"resumen"`
	SucursalID     string         `json:"sucursal_id"`
	SucursalNombre string         `json:"sucursal_nombre"`
	TS             string         `json:"ts"`
	Moneda         string         `json:"moneda"`
	Totales        TotalesSalon   `json:"totales"`
	PorTomador     []TomadorSalon `json:"por_tomador"`
	Mesas          []MesaEstado   `json:"mesas"`
}

type ColaCocina struct {
	Recibido   int `json:"recibido"`
	Preparando int `json:"preparando"`
	Listo      int `json:"listo"`
}

type SalidaCocina struct {
	Resumen        string         `json:"resumen"`
	SucursalID     string         `json:"sucursal_id"`
	SucursalNombre string         `json:"sucursal_nombre"`
	TS             string         `json:"ts"`
	Cola           ColaCocina     `json:"cola"`
	Pedidos        []PedidoCocina `json:"pedidos"`
	UltimaHora     UltimaHora     `json:"ultima_hora"`
}

type SalidaTurno struct {
	Resumen             string      `json:"resumen"`
	SucursalID          string      `json:"sucursal_id"`
	SucursalNombre      string      `json:"sucursal_nombre"`
	TS                  string      `json:"ts"`
	Moneda              string      `json:"moneda"`
	Desde               string      `json:"desde"`
	ZonaHoraria         string      `json:"zona_horaria"`
	Pedidos             int64       `json:"pedidos"`
	PedidosCerrados     int64       `json:"pedidos_cerrados"`
	Comensales          int         `json:"comensales"`
	Facturacion         float64     `json:"facturacion"`
	TicketPromedio      *float64    `json:"ticket_promedio"`
	DespachoPromedioMin *float64    `json:"despacho_promedio_min"`
	PorHora             []HoraTurno `json:"por_hora"`
}
