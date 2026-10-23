package mcp

import (
	"encoding/json"
	"errors"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/evento"
)

var (
	ErrMesaNoEncontrada = errors.New("mesa no encontrada")
	ErrMomentoInvalido  = errors.New("momento inválido")
	ErrMomentoFuturo    = errors.New("momento en el futuro")
)

const (
	CodigoParametroInvalido = "parametro_invalido"
	CodigoSinDatos          = "sin_datos"
	CodigoMesaNoEncontrada  = "mesa_no_encontrada"

	// ventanaReconstruccion es cuánto se lee hacia atrás desde el momento consultado.
	// Una cuenta abierta hace más que esto aparece sin los pedidos anteriores a la ventana.
	ventanaReconstruccion = 24 * time.Hour
	// toleranciaFuturo absorbe el desfase de reloj entre el cliente y el servidor.
	toleranciaFuturo = time.Minute
	// maxRelativoHoras acota los valores relativos ("hace 99999 h") para no desbordar la duración.
	maxRelativoHoras = 24 * 365
)

// MesaRef identifica una mesa del tenant por su ID y su número visible.
type MesaRef struct {
	ID     string
	Numero int
}

// PedidoEnMomento es un pedido de la cuenta abierta tal como estaba en el momento consultado.
type PedidoEnMomento struct {
	PedidoID          string  `json:"pedido_id"`
	Estado            string  `json:"estado"`
	DesdeTS           string  `json:"desde_ts"`
	MinutosEnEstado   int     `json:"minutos_en_estado"`
	QuienUltimoCambio string  `json:"quien_ultimo_cambio"`
	Rol               *string `json:"rol"`
}

// MesaEnMomento es el estado reconstruido de una mesa a partir de sus hitos.
type MesaEnMomento struct {
	Mesa             int               `json:"mesa"`
	CuentaAbierta    bool              `json:"cuenta_abierta"`
	CuentaVersion    *int              `json:"cuenta_version"`
	Pedidos          []PedidoEnMomento `json:"pedidos"`
	CuentaSolicitada bool              `json:"cuenta_solicitada"`
	PagoHabilitado   bool              `json:"pago_habilitado"`
	Espera           *string           `json:"espera"`
	MinutosEnEspera  *int              `json:"minutos_en_espera"`
}

type SalidaEstadoMesas struct {
	Resumen        string          `json:"resumen"`
	SucursalID     string          `json:"sucursal_id"`
	SucursalNombre string          `json:"sucursal_nombre"`
	TS             string          `json:"ts"`
	Momento        string          `json:"momento"`
	HistorialDesde string          `json:"historial_desde"`
	Mesas          []MesaEnMomento `json:"mesas"`
}

type Hito struct {
	TS                   string   `json:"ts"`
	Hito                 string   `json:"hito"`
	PedidoID             *string  `json:"pedido_id"`
	ItemID               *string  `json:"item_id"`
	EstadoAnterior       *string  `json:"estado_anterior"`
	EstadoNuevo          *string  `json:"estado_nuevo"`
	Quien                string   `json:"quien"`
	Rol                  *string  `json:"rol"`
	ModoEntrega          *string  `json:"modo_entrega"`
	MinutosDesdeAnterior *float64 `json:"minutos_desde_anterior"`
}

// EtapasPedido son los minutos entre hitos consecutivos de un pedido; null si falta alguno.
type EtapasPedido struct {
	PedidoID                 string   `json:"pedido_id"`
	MinutosHastaConfirmacion *float64 `json:"minutos_hasta_confirmacion"`
	MinutosEsperaCocina      *float64 `json:"minutos_espera_cocina"`
	MinutosPreparacion       *float64 `json:"minutos_preparacion"`
	MinutosAEntrega          *float64 `json:"minutos_a_entrega"`
	ModoEntrega              *string  `json:"modo_entrega"`
}

type SalidaHistorialMesa struct {
	Resumen        string         `json:"resumen"`
	SucursalID     string         `json:"sucursal_id"`
	SucursalNombre string         `json:"sucursal_nombre"`
	TS             string         `json:"ts"`
	Mesa           int            `json:"mesa"`
	Desde          string         `json:"desde"`
	Hasta          string         `json:"hasta"`
	Hitos          []Hito         `json:"hitos"`
	Etapas         []EtapasPedido `json:"etapas"`
}

var reRelativo = regexp.MustCompile(`^(?:hace\s+)?(\d+)\s*([a-z]+)(?:\s+atras|\s+ago)?$`)

var unidadesRelativas = map[string]time.Duration{
	"m": time.Minute, "min": time.Minute, "mins": time.Minute, "minuto": time.Minute, "minutos": time.Minute,
	"minute": time.Minute, "minutes": time.Minute,
	"h": time.Hour, "hs": time.Hour, "hr": time.Hour, "hrs": time.Hour, "hora": time.Hour, "horas": time.Hour,
	"hour": time.Hour, "hours": time.Hour,
}

// parsearMomento interpreta un instante: ISO-8601 con zona ("2026-10-22T14:00:00-03:00") o relativo a
// `ahora` ("hace 60 min", "hace 2 horas", "60m", "2h", "90 minutes ago"). El vacío, "ahora" y "now"
// devuelven `ahora`. Un instante futuro (más allá de un minuto de tolerancia) es ErrMomentoFuturo.
func parsearMomento(raw string, ahora time.Time) (time.Time, error) {
	valor := strings.ToLower(strings.TrimSpace(raw))
	switch valor {
	case "", "ahora", "now":
		return ahora, nil
	}
	if m := reRelativo.FindStringSubmatch(strings.ReplaceAll(valor, "á", "a")); m != nil {
		unidad, ok := unidadesRelativas[m[2]]
		cantidad, err := strconv.Atoi(m[1])
		if !ok || err != nil || cantidad > maxRelativoHoras*int(time.Hour/unidad) {
			return time.Time{}, ErrMomentoInvalido
		}
		return ahora.Add(-time.Duration(cantidad) * unidad), nil
	}
	t, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(raw))
	if err != nil {
		return time.Time{}, ErrMomentoInvalido
	}
	if t.After(ahora.Add(toleranciaFuturo)) {
		return time.Time{}, ErrMomentoFuturo
	}
	if t.After(ahora) {
		return ahora, nil
	}
	return t, nil
}

func quienDe(e evento.Evento) string {
	if e.ActorNombre != "" {
		return e.ActorNombre
	}
	return e.ActorTipo
}

func ptrNoVacio(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func modoEntregaDe(e evento.Evento) *string {
	var detalle struct {
		ModoEntrega string `json:"modo_entrega"`
	}
	if len(e.Detalle) == 0 || json.Unmarshal(e.Detalle, &detalle) != nil {
		return nil
	}
	return ptrNoVacio(detalle.ModoEntrega)
}

func ordenarPorTS(eventos []evento.Evento) []evento.Evento {
	copia := append([]evento.Evento{}, eventos...)
	sort.SliceStable(copia, func(i, j int) bool { return copia[i].TS.Before(copia[j].TS) })
	return copia
}

// reconstruirMesas arma el estado de cada mesa en `momento` solo con los eventos hasta ese instante.
// Es determinista: depende únicamente de sus argumentos.
func reconstruirMesas(mesas []MesaRef, eventos []evento.Evento, momento time.Time) []MesaEnMomento {
	porMesa := map[string][]evento.Evento{}
	for _, e := range ordenarPorTS(eventos) {
		if !e.TS.After(momento) {
			porMesa[e.MesaID] = append(porMesa[e.MesaID], e)
		}
	}
	resultado := make([]MesaEnMomento, 0, len(mesas))
	for _, ref := range mesas {
		resultado = append(resultado, estadoDeMesa(ref.Numero, porMesa[ref.ID], momento))
	}
	sort.SliceStable(resultado, func(i, j int) bool { return resultado[i].Mesa < resultado[j].Mesa })
	return resultado
}

func estadoDeMesa(numero int, eventos []evento.Evento, momento time.Time) MesaEnMomento {
	m := MesaEnMomento{Mesa: numero, Pedidos: []PedidoEnMomento{}}

	// Un pago aprobado tardío puede llevar la versión de la cuenta siguiente: no define cuál es la vigente.
	version, hay := 0, false
	for _, e := range eventos {
		if e.Tipo != evento.TipoPagoAprobado && (!hay || e.CuentaVersion > version) {
			version, hay = e.CuentaVersion, true
		}
	}
	if !hay {
		return m
	}
	for _, e := range eventos {
		if e.Tipo == evento.TipoCuentaCerrada && e.CuentaVersion == version {
			return m
		}
	}
	m.CuentaAbierta, m.CuentaVersion = true, &version

	type pedidoAcum struct {
		PedidoEnMomento
		desde time.Time
	}
	var orden []string
	pedidos := map[string]*pedidoAcum{}
	loc := ubicacionNegocio()
	for _, e := range eventos {
		if e.CuentaVersion != version {
			continue
		}
		switch e.Tipo {
		case evento.TipoPedidoCreado, evento.TipoPedidoEstado:
			if e.PedidoID == nil {
				continue
			}
			p := pedidos[*e.PedidoID]
			if p == nil {
				p = &pedidoAcum{PedidoEnMomento: PedidoEnMomento{PedidoID: *e.PedidoID}}
				pedidos[*e.PedidoID] = p
				orden = append(orden, *e.PedidoID)
			}
			p.Estado, p.desde = e.EstadoNuevo, e.TS
			p.DesdeTS = formatoTS(e.TS.In(loc))
			p.QuienUltimoCambio, p.Rol = quienDe(e), ptrNoVacio(e.ActorRol)
		case evento.TipoCuentaSolicitada:
			m.CuentaSolicitada = true
		case evento.TipoPagoHabilitado:
			m.PagoHabilitado = e.EstadoNuevo == "habilitado"
		}
	}

	var cocina, listos int
	var cocinaDesde, listoDesde *time.Time
	for _, id := range orden {
		p := pedidos[id]
		p.MinutosEnEstado = *minutosDesde(momento, &p.desde)
		m.Pedidos = append(m.Pedidos, p.PedidoEnMomento)
		desde := p.desde
		switch p.Estado {
		case "recibido", "confirmado", "preparando":
			cocina++
			if cocinaDesde == nil || desde.Before(*cocinaDesde) {
				cocinaDesde = &desde
			}
		case "listo":
			listos++
			if listoDesde == nil || desde.Before(*listoDesde) {
				listoDesde = &desde
			}
		}
	}
	estado := MesaOcupada
	switch {
	case m.PagoHabilitado:
		estado = MesaPagoHabilitado
	case m.CuentaSolicitada:
		estado = MesaCuentaSolicitada
	}
	m.Espera, m.MinutosEnEspera = derivarEspera(estado, cocina, listos, minutosDesde(momento, cocinaDesde), minutosDesde(momento, listoDesde))
	return m
}

func minutosEntre(a, b *time.Time) *float64 {
	if a == nil || b == nil {
		return nil
	}
	v := math.Max(math.Round(b.Sub(*a).Minutes()*10)/10, 0)
	return &v
}

// construirHitos pasa los eventos a la línea de tiempo; minutos_desde_anterior es respecto del hito previo de la lista.
func construirHitos(eventos []evento.Evento) []Hito {
	loc := ubicacionNegocio()
	hitos := make([]Hito, 0, len(eventos))
	var previo *time.Time
	for _, e := range ordenarPorTS(eventos) {
		ts := e.TS
		hitos = append(hitos, Hito{
			TS: formatoTS(ts.In(loc)), Hito: e.Tipo, PedidoID: e.PedidoID, ItemID: e.PedidoItemID,
			EstadoAnterior: ptrNoVacio(e.EstadoAnterior), EstadoNuevo: ptrNoVacio(e.EstadoNuevo),
			Quien: quienDe(e), Rol: ptrNoVacio(e.ActorRol), ModoEntrega: modoEntregaDe(e),
			MinutosDesdeAnterior: minutosEntre(previo, &ts),
		})
		previo = &ts
	}
	return hitos
}

// calcularEtapas mide por pedido los minutos entre creación, confirmación, inicio de preparación, listo y entrega.
// Si un hito cayó fuera del rango consultado o todavía no ocurrió, las etapas que lo usan quedan en null.
func calcularEtapas(eventos []evento.Evento) []EtapasPedido {
	type tiempos struct {
		creado, confirmado, preparando, listo, entregado *time.Time
		modo                                             *string
	}
	var orden []string
	porPedido := map[string]*tiempos{}
	for _, e := range ordenarPorTS(eventos) {
		if e.PedidoID == nil || (e.Tipo != evento.TipoPedidoCreado && e.Tipo != evento.TipoPedidoEstado) {
			continue
		}
		t := porPedido[*e.PedidoID]
		if t == nil {
			t = &tiempos{}
			porPedido[*e.PedidoID] = t
			orden = append(orden, *e.PedidoID)
		}
		ts := e.TS
		var destino **time.Time
		switch {
		case e.Tipo == evento.TipoPedidoCreado:
			destino = &t.creado
		case e.EstadoNuevo == "confirmado":
			destino = &t.confirmado
		case e.EstadoNuevo == "preparando":
			destino = &t.preparando
		case e.EstadoNuevo == "listo":
			destino = &t.listo
		case e.EstadoNuevo == "entregado":
			destino = &t.entregado
			t.modo = modoEntregaDe(e)
		default:
			continue
		}
		if *destino == nil {
			*destino = &ts
		}
	}
	etapas := make([]EtapasPedido, 0, len(orden))
	for _, id := range orden {
		t := porPedido[id]
		etapas = append(etapas, EtapasPedido{
			PedidoID:                 id,
			MinutosHastaConfirmacion: minutosEntre(t.creado, t.confirmado),
			MinutosEsperaCocina:      minutosEntre(t.confirmado, t.preparando),
			MinutosPreparacion:       minutosEntre(t.preparando, t.listo),
			MinutosAEntrega:          minutosEntre(t.listo, t.entregado),
			ModoEntrega:              t.modo,
		})
	}
	return etapas
}
