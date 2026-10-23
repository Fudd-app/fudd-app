package mcp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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

func mensajePlanSinMCP(appURL string) string {
	return "Tu plan actual no incluye la conexión con asistentes de IA (es exclusiva del Plan Pro). " +
		"Podés activarlo desde Configuración → Planes en " + appURL + "/dashboard."
}

// errNegocio es un error esperable que el modelo puede leer y corregir.
type errNegocio string

func (e errNegocio) Error() string { return string(e) }

type herramientas struct {
	deps      Dependencias
	claims    *auth.Claims
	limitador *ratelimit.Limitador
}

type entradaSucursal struct {
	SucursalID string `json:"sucursal_id,omitempty" jsonschema:"ID de la sucursal; opcional si el negocio tiene una sola"`
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
	sdk.AddTool(s, def("listar_sucursales", "Listar sucursales", "Lista las sucursales del negocio con su cantidad de mesas activas."), h.listarSucursales)
	sdk.AddTool(s, def("estado_salon", "Estado del salón", "Estado actual de cada mesa activa: libre, ocupada, cuenta solicitada o pago habilitado, con comensales, pedidos activos, tiempo y total parcial."), h.estadoSalon)
	sdk.AddTool(s, def("estado_cocina", "Estado de cocina", "Pedidos en cocina (recibido, preparando, listo) con su demora y el despacho promedio de la última hora: tiempo promedio desde que entra un pedido hasta que cocina lo marca listo, en la última hora."), h.estadoCocina)
	sdk.AddTool(s, def("resumen_turno", "Resumen del turno", "Resumen de hoy hasta ahora: pedidos, comensales atendidos, facturación, ticket promedio y despacho promedio (despacho_promedio_min es el promedio del día sobre pedidos cerrados)."), h.resumenTurno)
}

func ubicacionNegocio() *time.Location {
	ubicacion, err := time.LoadLocation(zonaHorariaNegocio)
	if err != nil {
		return time.FixedZone(zonaHorariaNegocio, -3*60*60)
	}
	return ubicacion
}

// ejecutar registra la invocación y traduce errores: los de negocio se muestran
// tal cual y los internos se ocultan.
func ejecutar[Out any](ctx context.Context, h *herramientas, nombre string, f func() (Out, string, error)) (*sdk.CallToolResult, Out, error) {
	inicio := time.Now()
	if h.limitador != nil && !h.limitador.Permitir(h.claims.UsuarioID) {
		var cero Out
		return nil, cero, errNegocio(mensajeRateLimit)
	}
	if h.deps.Acceso == nil {
		var cero Out
		slog.ErrorContext(ctx, "mcp sin verificador de plan configurado", "tool", nombre)
		return nil, cero, errors.New(mensajeInterno)
	}
	incluido, err := h.deps.Acceso.AccesoMCP(ctx, h.claims.TenantID)
	if err != nil {
		var cero Out
		slog.ErrorContext(ctx, "error verificando plan para mcp", "tool", nombre, "tenant_id", h.claims.TenantID, "error", err)
		return nil, cero, errors.New(mensajeInterno)
	}
	if !incluido {
		var cero Out
		return nil, cero, errNegocio(mensajePlanSinMCP(h.deps.AppURL))
	}
	salida, resumen, err := f()
	slog.InfoContext(ctx, "mcp tool",
		"tool", nombre, "usuario_id", h.claims.UsuarioID, "tenant_id", h.claims.TenantID,
		"duracion_ms", time.Since(inicio).Milliseconds(), "error", err != nil)
	if err != nil {
		var negocio errNegocio
		if !errors.As(err, &negocio) {
			slog.ErrorContext(ctx, "error en tool mcp", "tool", nombre, "error", err)
			err = errors.New(mensajeInterno)
		}
		var cero Out
		return nil, cero, err
	}
	return &sdk.CallToolResult{Content: []sdk.Content{&sdk.TextContent{Text: resumen}}}, salida, nil
}

// resolverSucursal valida pertenencia al tenant o infiere la única sucursal.
func (h *herramientas) resolverSucursal(ctx context.Context, tenantID, sucursalID string) (*SucursalRef, error) {
	sucursalID = strings.TrimSpace(sucursalID)
	if sucursalID != "" {
		ref, err := h.deps.Store.SucursalDelTenant(ctx, tenantID, sucursalID)
		if errors.Is(err, ErrSucursalNoEncontrada) {
			return nil, errNegocio("Sucursal no encontrada")
		}
		return ref, err
	}
	sucursales, err := h.deps.Store.SucursalesDelTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	switch len(sucursales) {
	case 0:
		return nil, errNegocio("El negocio todavía no tiene sucursales.")
	case 1:
		return &SucursalRef{ID: sucursales[0].ID, Nombre: sucursales[0].Nombre}, nil
	}
	items := make([]string, len(sucursales))
	for i, s := range sucursales {
		items[i] = fmt.Sprintf("%s (%s)", s.Nombre, s.ID)
	}
	return nil, errNegocio("El negocio tiene varias sucursales: " + strings.Join(items, ", ") + ". Indicá sucursal_id.")
}

func (h *herramientas) listarSucursales(ctx context.Context, _ *sdk.CallToolRequest, _ sinEntrada) (*sdk.CallToolResult, SalidaSucursales, error) {
	return ejecutar(ctx, h, "listar_sucursales", func() (SalidaSucursales, string, error) {
		lista, err := h.deps.Store.SucursalesDelTenant(ctx, h.claims.TenantID)
		if err != nil {
			return SalidaSucursales{}, "", err
		}
		if lista == nil {
			lista = []SucursalResumen{}
		}
		return SalidaSucursales{Sucursales: lista}, fmt.Sprintf("%d sucursal(es) en el negocio.", len(lista)), nil
	})
}

func (h *herramientas) estadoSalon(ctx context.Context, _ *sdk.CallToolRequest, in entradaSucursal) (*sdk.CallToolResult, SalidaSalon, error) {
	return ejecutar(ctx, h, "estado_salon", func() (SalidaSalon, string, error) {
		suc, err := h.resolverSucursal(ctx, h.claims.TenantID, in.SucursalID)
		if err != nil {
			return SalidaSalon{}, "", err
		}
		mesas, err := h.deps.Store.EstadoSalon(ctx, h.claims.TenantID, suc.ID, h.deps.Ahora())
		if err != nil {
			return SalidaSalon{}, "", err
		}
		if mesas == nil {
			mesas = []MesaEstado{}
		}
		totales := TotalesSalon{Mesas: len(mesas)}
		for _, m := range mesas {
			if m.Estado != MesaLibre {
				totales.Ocupadas++
			}
			totales.Comensales += m.Comensales
		}
		resumen := fmt.Sprintf("%s: %d de %d mesas ocupadas, %d comensales.", suc.Nombre, totales.Ocupadas, totales.Mesas, totales.Comensales)
		return SalidaSalon{Sucursal: *suc, Mesas: mesas, Totales: totales}, resumen, nil
	})
}

func (h *herramientas) estadoCocina(ctx context.Context, _ *sdk.CallToolRequest, in entradaSucursal) (*sdk.CallToolResult, SalidaCocina, error) {
	return ejecutar(ctx, h, "estado_cocina", func() (SalidaCocina, string, error) {
		suc, err := h.resolverSucursal(ctx, h.claims.TenantID, in.SucursalID)
		if err != nil {
			return SalidaCocina{}, "", err
		}
		ahora := h.deps.Ahora()
		pedidos, err := h.deps.Store.PedidosCocina(ctx, h.claims.TenantID, suc.ID, ahora)
		if err != nil {
			return SalidaCocina{}, "", err
		}
		if pedidos == nil {
			pedidos = []PedidoCocina{}
		}
		ultimaHora, err := h.deps.Store.DespachoPromedio(ctx, h.claims.TenantID, suc.ID, ahora.Add(-time.Hour))
		if err != nil {
			return SalidaCocina{}, "", err
		}

		resumen := ResumenCocina{DespachoPromedioUltimaHora: ultimaHora}
		for _, p := range pedidos {
			switch p.Estado {
			case "recibido":
				resumen.Recibidos++
			case "preparando":
				resumen.Preparando++
			case "listo":
				resumen.Listos++
			}
			if (p.Estado == "recibido" || p.Estado == "preparando") &&
				(resumen.PedidoMasDemorado == nil || p.Minutos > resumen.PedidoMasDemorado.Minutos) {
				resumen.PedidoMasDemorado = &PedidoDemorado{Mesa: p.Mesa, Estado: p.Estado, Minutos: p.Minutos}
			}
		}
		texto := fmt.Sprintf("%s: %d recibidos, %d preparando, %d listos.", suc.Nombre, resumen.Recibidos, resumen.Preparando, resumen.Listos)
		if d := resumen.PedidoMasDemorado; d != nil {
			texto += fmt.Sprintf(" Más demorado: mesa %d (%s) hace %d min.", d.Mesa, d.Estado, d.Minutos)
		}
		return SalidaCocina{Sucursal: *suc, Pedidos: pedidos, Resumen: resumen}, texto, nil
	})
}

func (h *herramientas) resumenTurno(ctx context.Context, _ *sdk.CallToolRequest, in entradaSucursal) (*sdk.CallToolResult, SalidaTurno, error) {
	return ejecutar(ctx, h, "resumen_turno", func() (SalidaTurno, string, error) {
		suc, err := h.resolverSucursal(ctx, h.claims.TenantID, in.SucursalID)
		if err != nil {
			return SalidaTurno{}, "", err
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
			return SalidaTurno{}, "", err
		}
		comensales, err := h.deps.Store.ComensalesAtendidos(ctx, h.claims.TenantID, suc.ID, desde, hasta)
		if err != nil {
			return SalidaTurno{}, "", err
		}
		var despacho *float64
		if resumen.TiempoPromedioDespachoMinutos > 0 {
			v := resumen.TiempoPromedioDespachoMinutos
			despacho = &v
		}
		salida := SalidaTurno{
			Sucursal:            *suc,
			Desde:               desde,
			Hasta:               hasta,
			ZonaHoraria:         zonaHorariaNegocio,
			Pedidos:             resumen.PedidosTotales,
			PedidosCerrados:     resumen.PedidosCerrados,
			ComensalesAtendidos: comensales,
			DespachoPromedioMin: despacho,
			Facturacion:         resumen.FacturacionTotal,
			TicketPromedio:      resumen.TicketPromedio,
		}
		texto := fmt.Sprintf("%s, hoy: %d pedidos (%d cerrados), %d comensales, facturación $%.2f, ticket promedio $%.2f.",
			suc.Nombre, salida.Pedidos, salida.PedidosCerrados, comensales, salida.Facturacion, salida.TicketPromedio)
		return salida, texto, nil
	})
}
