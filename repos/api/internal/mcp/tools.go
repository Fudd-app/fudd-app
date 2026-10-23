package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/ratelimit"
)

const (
	// Mismo criterio que internal/metrica: allí la zona no es exportable.
	zonaHorariaNegocio = "America/Argentina/Buenos_Aires"
	mensajeInterno     = "Error interno, reintentá en unos segundos"
	mensajeRateLimit   = "Demasiadas consultas en el último minuto. Esperá unos segundos y volvé a intentar."
)

const (
	descListarSucursales = "Lista las sucursales del negocio. Usala primero cuando el usuario tiene varias sucursales o no aclara cuál: " +
		"el campo `id` es el `sucursal_id` que piden las demás tools. " +
		"Cada sucursal trae `mesas_totales` (incluye mesas inactivas), `mesas_activas` (las que se usan hoy) y " +
		"`abierta_ahora`: true/false según el horario cargado en la configuración de la sucursal, en la zona horaria del negocio " +
		"(contempla tramos que cruzan la medianoche); es null si la sucursal no tiene horarios cargados. " +
		"No informa personal en turno ni estado operativo: para eso usá estado_salon y estado_cocina."

	descEstadoSalon = "Foto en vivo del salón de una sucursal: una entrada por mesa activa. " +
		"`estado` es libre, ocupada, cuenta_solicitada o pago_habilitado. " +
		"`totales` resume mesas ocupadas, mesas totales, comensales y `mesas_esperando` (cuántas mesas esperan comida, entrega o cobro). " +
		"Por mesa: `comensales` (si `comensales_sin_identificar` es true hay al menos uno que no se identificó), `pedidos_activos` (no cerrados), " +
		"`total_parcial` en ARS de la cuenta vigente, `minutos_desde_primer_pedido` y `minutos_desde_ultimo_pedido` (null si no hay pedidos), " +
		"`espera` y `minutos_en_espera`, y `tomado_por` (nombres de quienes cargaron los pedidos de la cuenta, o \"QR\" si los pidió el comensal). " +
		"`espera` indica en qué etapa está trabada la mesa: `cobro` si pidió la cuenta o tiene el pago habilitado; si no, `comida` si hay pedidos " +
		"recibidos, confirmados o en preparación (minutos desde el pedido pendiente más antiguo); si no, `entrega` si cocina ya terminó pedidos y falta marcar la entrega (minutos desde el " +
		"listo más antiguo); si no, null (los pedidos entregados no generan espera). Para `cobro` `minutos_en_espera` es null porque no se registra cuándo se pidió la cuenta. " +
		"`por_tomador` agrupa mesas, comensales y esperas por quien cargó pedidos; una mesa con varios tomadores cuenta en cada uno. " +
		"Por defecto solo lista las mesas con actividad; con `incluir_libres` también las libres. " +
		"Lo que NO se sabe: no hay registro de personal en turno ni de qué mozo atiende cada mesa más allá de quién cargó pedidos; " +
		"`entrega` significa que cocina terminó el pedido, no hay confirmación de que se haya entregado; una mesa sentada que todavía no pidió aparece como libre."

	descEstadoCocina = "Estado de la cocina de una sucursal. `cola` cuenta los pedidos de las cuentas vigentes en recibido (pendiente de confirmar recepción), confirmado (esperan que cocina lo tome), preparando y listo (cocina terminó, falta marcar la entrega; los entregados ya no figuran). " +
		"`pedidos` va ordenado del más antiguo al más nuevo (`minutos_desde_ingreso` descendente), con mesa, estado y `items` (cantidad de ítems pendientes, preparando y listos). " +
		"Para ver el cuello de botella en cocina mirá los primeros pedidos en recibido, confirmado o preparando. " +
		"`ultima_hora` describe los últimos 60 minutos: `ingresados_60min`, `listos_60min`, `balance_60min` (ingresados menos listos: positivo significa que la cola crece) y " +
		"el tiempo de despacho (desde que entra el pedido hasta que cocina lo marca listo) como promedio y p90 en minutos, calculado sobre los pedidos que pasaron a listo en esa hora; " +
		"son null si ninguno pasó a listo. " +
		"Lo que NO se sabe: no hay estaciones ni cocineros, ni historial de cambios de estado, ni referencia histórica para decidir qué es una demora normal; " +
		"un pedido listo no implica que se haya entregado: el mozo debe marcarlo entregado."

	descResumenTurno = "Resumen de lo que va de hoy (desde las 00:00 hasta ahora, en la zona horaria del negocio; no contempla jornadas que cruzan la medianoche). " +
		"`pedidos` son todos los creados hoy y `pedidos_cerrados` los ya cobrados/cerrados. " +
		"`facturacion` en ARS suma solo pedidos cerrados; `ticket_promedio` es facturación por pedido cerrado (null si no hay cerrados). " +
		"`comensales` son los atendidos hoy (por cuenta de mesa). `despacho_promedio_min` es el tiempo medio hasta que cocina marcó listo los pedidos cerrados (null sin datos). " +
		"`por_hora` desglosa por hora de creación del pedido: pedidos, comensales, facturación y despacho; las horas sin pedidos no aparecen, y una cuenta que se extiende por varias horas suma comensales en cada una. " +
		"Lo que NO se sabe: no hay comparación con otros días ni personal en turno."
)

func mensajePlanSinMCP(appURL string) string {
	return "Tu plan actual no incluye la conexión con asistentes de IA (es exclusiva del Plan Pro). " +
		"Podés activarlo desde Configuración → Planes en " + appURL + "/dashboard."
}

func errInterno() error {
	return &errTool{Codigo: CodigoErrorInterno, Mensaje: mensajeInterno}
}

type herramientas struct {
	deps      Dependencias
	claims    *auth.Claims
	limitador *ratelimit.Limitador
}

type entradaSucursal struct {
	SucursalID string `json:"sucursal_id,omitempty" jsonschema:"ID de la sucursal; opcional si el negocio tiene una sola"`
}

type entradaSalon struct {
	SucursalID    string `json:"sucursal_id,omitempty" jsonschema:"ID de la sucursal; opcional si el negocio tiene una sola"`
	IncluirLibres bool   `json:"incluir_libres,omitempty" jsonschema:"incluir también las mesas libres (por defecto solo las que tienen actividad)"`
}

type sinEntrada struct{}

func (h *herramientas) registrar(s *sdk.Server) {
	def := func(nombre, titulo, descripcion string) *sdk.Tool {
		abierto := false
		return &sdk.Tool{
			Name:        nombre,
			Description: descripcion,
			Annotations: &sdk.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &abierto, Title: titulo},
		}
	}
	sdk.AddTool(s, def("listar_sucursales", "Listar sucursales", descListarSucursales), h.listarSucursales)
	sdk.AddTool(s, def("estado_salon", "Estado del salón", descEstadoSalon), h.estadoSalon)
	sdk.AddTool(s, def("estado_cocina", "Estado de cocina", descEstadoCocina), h.estadoCocina)
	sdk.AddTool(s, def("resumen_turno", "Resumen del turno", descResumenTurno), h.resumenTurno)
}

func ubicacionNegocio() *time.Location {
	ubicacion, err := time.LoadLocation(zonaHorariaNegocio)
	if err != nil {
		return time.FixedZone(zonaHorariaNegocio, -3*60*60)
	}
	return ubicacion
}

// ejecutar registra la invocación y traduce errores: los de negocio se muestran
// tal cual (JSON estructurado) y los internos se ocultan. En éxito el TextContent
// es el mismo JSON que el SDK pone en structuredContent.
func ejecutar[Out any](ctx context.Context, h *herramientas, nombre string, f func() (Out, error)) (*sdk.CallToolResult, Out, error) {
	var cero Out
	inicio := time.Now()
	if h.limitador != nil && !h.limitador.Permitir(h.claims.UsuarioID) {
		return nil, cero, &errTool{Codigo: CodigoRateLimited, Mensaje: mensajeRateLimit}
	}
	if h.deps.Acceso == nil {
		slog.ErrorContext(ctx, "mcp sin verificador de plan configurado", "tool", nombre)
		return nil, cero, errInterno()
	}
	incluido, err := h.deps.Acceso.AccesoMCP(ctx, h.claims.TenantID)
	if err != nil {
		slog.ErrorContext(ctx, "error verificando plan para mcp", "tool", nombre, "tenant_id", h.claims.TenantID, "error", err)
		return nil, cero, errInterno()
	}
	if !incluido {
		return nil, cero, &errTool{
			Codigo:  CodigoPlanLimitReached,
			Mensaje: mensajePlanSinMCP(h.deps.AppURL),
			Detalle: map[string]string{"recurso": "mcp", "upgrade_url": h.deps.AppURL + "/dashboard"},
		}
	}
	salida, err := f()
	slog.InfoContext(ctx, "mcp tool",
		"tool", nombre, "usuario_id", h.claims.UsuarioID, "tenant_id", h.claims.TenantID,
		"duracion_ms", time.Since(inicio).Milliseconds(), "error", err != nil)
	if err != nil {
		var negocio *errTool
		if !errors.As(err, &negocio) {
			slog.ErrorContext(ctx, "error en tool mcp", "tool", nombre, "error", err)
			err = errInterno()
		}
		return nil, cero, err
	}
	texto, err := json.Marshal(salida)
	if err != nil {
		slog.ErrorContext(ctx, "error serializando salida de tool mcp", "tool", nombre, "error", err)
		return nil, cero, errInterno()
	}
	return &sdk.CallToolResult{Content: []sdk.Content{&sdk.TextContent{Text: string(texto)}}}, salida, nil
}

func refsSucursales(lista []SucursalResumen) []SucursalRef {
	refs := make([]SucursalRef, len(lista))
	for i, s := range lista {
		refs[i] = SucursalRef{ID: s.ID, Nombre: s.Nombre}
	}
	return refs
}

// resolverSucursal valida pertenencia al tenant o infiere la única sucursal.
// Los errores llevan en detalle.sucursales los IDs y nombres válidos.
func (h *herramientas) resolverSucursal(ctx context.Context, tenantID, sucursalID string) (*SucursalRef, error) {
	// validas se consulta solo cuando hace falta: sin sucursal_id o para armar el error.
	validas := func() (map[string]any, []SucursalResumen, error) {
		sucursales, err := h.deps.Store.SucursalesDelTenant(ctx, tenantID)
		if err != nil {
			return nil, nil, err
		}
		return map[string]any{"sucursales": refsSucursales(sucursales)}, sucursales, nil
	}

	if sucursalID = strings.TrimSpace(sucursalID); sucursalID != "" {
		ref, err := h.deps.Store.SucursalDelTenant(ctx, tenantID, sucursalID)
		if errors.Is(err, ErrSucursalNoEncontrada) {
			detalle, _, err := validas()
			if err != nil {
				return nil, err
			}
			return nil, &errTool{Codigo: CodigoSucursalNoEncontrada, Mensaje: "Sucursal no encontrada", Detalle: detalle}
		}
		return ref, err
	}
	detalle, sucursales, err := validas()
	if err != nil {
		return nil, err
	}
	switch len(sucursales) {
	case 0:
		return nil, &errTool{Codigo: CodigoSucursalNoEncontrada, Mensaje: "El negocio todavía no tiene sucursales.", Detalle: detalle}
	case 1:
		return &SucursalRef{ID: sucursales[0].ID, Nombre: sucursales[0].Nombre}, nil
	}
	return nil, &errTool{
		Codigo:  CodigoSucursalRequerida,
		Mensaje: "El negocio tiene varias sucursales. Indicá sucursal_id con uno de los ids de detalle.sucursales.",
		Detalle: detalle,
	}
}

func (h *herramientas) listarSucursales(ctx context.Context, _ *sdk.CallToolRequest, _ sinEntrada) (*sdk.CallToolResult, SalidaSucursales, error) {
	return ejecutar(ctx, h, "listar_sucursales", func() (SalidaSucursales, error) {
		lista, err := h.deps.Store.SucursalesDelTenant(ctx, h.claims.TenantID)
		if err != nil {
			return SalidaSucursales{}, err
		}
		ahora := h.deps.Ahora().In(ubicacionNegocio())
		items := make([]SucursalItem, len(lista))
		for i, s := range lista {
			items[i] = SucursalItem{
				ID: s.ID, Nombre: s.Nombre, MesasTotales: s.MesasTotales, MesasActivas: s.MesasActivas,
				AbiertaAhora: abiertaAhora(s.Horarios, ahora),
			}
		}
		return SalidaSucursales{
			Resumen:    cantidad(len(items), "sucursal", "sucursales") + " en el negocio.",
			TS:         formatoTS(ahora),
			Sucursales: items,
		}, nil
	})
}

func (h *herramientas) estadoSalon(ctx context.Context, _ *sdk.CallToolRequest, in entradaSalon) (*sdk.CallToolResult, SalidaSalon, error) {
	return ejecutar(ctx, h, "estado_salon", func() (SalidaSalon, error) {
		suc, err := h.resolverSucursal(ctx, h.claims.TenantID, in.SucursalID)
		if err != nil {
			return SalidaSalon{}, err
		}
		ahora := h.deps.Ahora()
		mesas, err := h.deps.Store.EstadoSalon(ctx, h.claims.TenantID, suc.ID, ahora)
		if err != nil {
			return SalidaSalon{}, err
		}
		totales, porTomador, visibles := resumirSalon(mesas, in.IncluirLibres)
		resumen := suc.Nombre + ": " + strconv.Itoa(totales.MesasOcupadas) + " de " +
			cantidad(totales.MesasTotales, "mesa", "mesas") + " " + pluralOcupadas(totales.MesasTotales) + ", " +
			cantidad(totales.Comensales, "comensal", "comensales") + ". Mesas esperando: comida " +
			strconv.Itoa(totales.MesasEsperando.Comida) + ", entrega " + strconv.Itoa(totales.MesasEsperando.Entrega) +
			", cobro " + strconv.Itoa(totales.MesasEsperando.Cobro) + "."
		return SalidaSalon{
			Resumen: resumen, SucursalID: suc.ID, SucursalNombre: suc.Nombre, TS: formatoTS(ahora.In(ubicacionNegocio())),
			Moneda: monedaARS, Totales: totales, PorTomador: porTomador, Mesas: visibles,
		}, nil
	})
}

func pluralOcupadas(total int) string {
	if total == 1 {
		return "ocupada"
	}
	return "ocupadas"
}

// resumirSalon calcula los totales y el desglose por tomador; sin incluirLibres descarta las mesas libres de la lista.
func resumirSalon(mesas []MesaEstado, incluirLibres bool) (TotalesSalon, []TomadorSalon, []MesaEstado) {
	totales := TotalesSalon{MesasTotales: len(mesas)}
	visibles := []MesaEstado{}
	porNombre := map[string]*TomadorSalon{}
	for _, m := range mesas {
		if m.Estado != MesaLibre {
			totales.MesasOcupadas++
		}
		totales.Comensales += m.Comensales
		sumarEspera(&totales.MesasEsperando, m.Espera)
		if m.Estado == MesaLibre {
			if incluirLibres {
				visibles = append(visibles, m)
			}
			continue
		}
		visibles = append(visibles, m)
		for _, nombre := range m.TomadoPor {
			t := porNombre[nombre]
			if t == nil {
				t = &TomadorSalon{Tomador: nombre}
				porNombre[nombre] = t
			}
			t.Mesas++
			t.Comensales += m.Comensales
			sumarEspera(&t.MesasEsperando, m.Espera)
		}
	}
	porTomador := make([]TomadorSalon, 0, len(porNombre))
	for _, t := range porNombre {
		porTomador = append(porTomador, *t)
	}
	sort.Slice(porTomador, func(i, j int) bool {
		if porTomador[i].Mesas != porTomador[j].Mesas {
			return porTomador[i].Mesas > porTomador[j].Mesas
		}
		return porTomador[i].Tomador < porTomador[j].Tomador
	})
	return totales, porTomador, visibles
}

func sumarEspera(e *Esperando, espera *string) {
	if espera == nil {
		return
	}
	switch *espera {
	case EsperaComida:
		e.Comida++
	case EsperaEntrega:
		e.Entrega++
	case EsperaCobro:
		e.Cobro++
	}
}

func (h *herramientas) estadoCocina(ctx context.Context, _ *sdk.CallToolRequest, in entradaSucursal) (*sdk.CallToolResult, SalidaCocina, error) {
	return ejecutar(ctx, h, "estado_cocina", func() (SalidaCocina, error) {
		suc, err := h.resolverSucursal(ctx, h.claims.TenantID, in.SucursalID)
		if err != nil {
			return SalidaCocina{}, err
		}
		ahora := h.deps.Ahora()
		pedidos, err := h.deps.Store.PedidosCocina(ctx, h.claims.TenantID, suc.ID, ahora)
		if err != nil {
			return SalidaCocina{}, err
		}
		ultimaHora, err := h.deps.Store.UltimaHoraCocina(ctx, h.claims.TenantID, suc.ID, ahora)
		if err != nil {
			return SalidaCocina{}, err
		}
		ultimaHora.Balance60Min = ultimaHora.Ingresados60Min - ultimaHora.Listos60Min

		pedidos = append([]PedidoCocina{}, pedidos...)
		sort.SliceStable(pedidos, func(i, j int) bool { return pedidos[i].MinutosDesdeIngreso > pedidos[j].MinutosDesdeIngreso })
		var cola ColaCocina
		var masDemorado *PedidoCocina
		for i, p := range pedidos {
			switch p.Estado {
			case "recibido":
				cola.Recibido++
			case "confirmado":
				cola.Confirmado++
			case "preparando":
				cola.Preparando++
			case "listo":
				cola.Listo++
			}
			if masDemorado == nil && (p.Estado == "recibido" || p.Estado == "confirmado" || p.Estado == "preparando") {
				masDemorado = &pedidos[i]
			}
		}
		resumen := suc.Nombre + ": en cocina " + cantidad(cola.Recibido, "pedido recibido", "pedidos recibidos") + ", " +
			strconv.Itoa(cola.Preparando) + " preparando, " + cantidad(cola.Listo, "listo", "listos") + "."
		if cola.Confirmado > 0 {
			resumen += " " + cantidad(cola.Confirmado, "confirmado esperando a cocina", "confirmados esperando a cocina") + "."
		}
		if masDemorado != nil {
			resumen += " Más demorado: mesa " + strconv.Itoa(masDemorado.Mesa) + " (" + masDemorado.Estado + ") hace " +
				cantidad(masDemorado.MinutosDesdeIngreso, "minuto", "minutos") + "."
		}
		return SalidaCocina{
			Resumen: resumen, SucursalID: suc.ID, SucursalNombre: suc.Nombre, TS: formatoTS(ahora.In(ubicacionNegocio())),
			Cola: cola, Pedidos: pedidos, UltimaHora: ultimaHora,
		}, nil
	})
}

func (h *herramientas) resumenTurno(ctx context.Context, _ *sdk.CallToolRequest, in entradaSucursal) (*sdk.CallToolResult, SalidaTurno, error) {
	return ejecutar(ctx, h, "resumen_turno", func() (SalidaTurno, error) {
		suc, err := h.resolverSucursal(ctx, h.claims.TenantID, in.SucursalID)
		if err != nil {
			return SalidaTurno{}, err
		}
		ahora := h.deps.Ahora().In(ubicacionNegocio())
		desde := time.Date(ahora.Year(), ahora.Month(), ahora.Day(), 0, 0, 0, 0, ahora.Location())
		hasta := ahora
		if !hasta.After(desde) {
			hasta = desde.Add(time.Second)
		}
		resumen, err := h.deps.Metricas.ObtenerResumen(ctx, h.claims.TenantID, suc.ID,
			desde.Format(time.RFC3339Nano), hasta.Format(time.RFC3339Nano))
		if err != nil {
			return SalidaTurno{}, err
		}
		comensales, err := h.deps.Store.ComensalesAtendidos(ctx, h.claims.TenantID, suc.ID, desde, hasta)
		if err != nil {
			return SalidaTurno{}, err
		}
		porHora, err := h.deps.Store.ResumenPorHora(ctx, h.claims.TenantID, suc.ID, desde, hasta, zonaHorariaNegocio)
		if err != nil {
			return SalidaTurno{}, err
		}
		if porHora == nil {
			porHora = []HoraTurno{}
		}
		var despacho, ticket *float64
		if resumen.TiempoPromedioDespachoMinutos > 0 {
			v := resumen.TiempoPromedioDespachoMinutos
			despacho = &v
		}
		if resumen.PedidosCerrados > 0 {
			v := resumen.TicketPromedio
			ticket = &v
		}
		texto := suc.Nombre + ", hoy: " + cantidad(int(resumen.PedidosTotales), "pedido", "pedidos") + " (" +
			cantidad(int(resumen.PedidosCerrados), "cerrado", "cerrados") + "), " +
			cantidad(comensales, "comensal", "comensales") + ", facturación " + formatoMonto(resumen.FacturacionTotal)
		if ticket != nil {
			texto += ", ticket promedio " + formatoMonto(*ticket)
		} else {
			texto += ", sin pedidos cerrados todavía"
		}
		texto += "."
		return SalidaTurno{
			Resumen: texto, SucursalID: suc.ID, SucursalNombre: suc.Nombre, TS: formatoTS(ahora), Moneda: monedaARS,
			Desde: formatoTS(desde), ZonaHoraria: zonaHorariaNegocio,
			Pedidos: resumen.PedidosTotales, PedidosCerrados: resumen.PedidosCerrados, Comensales: comensales,
			Facturacion: resumen.FacturacionTotal, TicketPromedio: ticket, DespachoPromedioMin: despacho, PorHora: porHora,
		}, nil
	})
}
