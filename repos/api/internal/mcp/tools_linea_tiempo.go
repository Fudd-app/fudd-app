package mcp

import (
	"context"
	"errors"
	"strconv"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	descEstadoMesas = "Reconstruye cómo estaban las mesas de una sucursal en un momento del pasado (o ahora). " +
		"Sirve para preguntas como \"¿cómo estaban las mesas hace una hora?\" o \"¿qué mesas esperaban comida a las 21:00?\". " +
		"`momento` acepta ISO-8601 con zona (\"2026-10-22T21:00:00-03:00\") o relativo (\"hace 60 min\", \"hace 2 horas\", \"60m\", \"2h\"); sin `momento` es ahora; no admite futuro. " +
		"`mesa` (opcional) es el número de una mesa para ver solo esa. Lista todas las mesas activas. " +
		"Por mesa: `cuenta_abierta` y `cuenta_version` (null si no había cuenta abierta), `pedidos` de esa cuenta con su `estado` en ese momento " +
		"(recibido, confirmado, preparando, listo, entregado), `desde_ts`, `minutos_en_estado`, `quien_ultimo_cambio` y `rol` de quien hizo el último cambio, " +
		"`cuenta_solicitada`, `pago_habilitado`, y `espera` con la misma regla que estado_salon: `cobro` si pidió la cuenta o tiene el pago habilitado; " +
		"si no, `comida` (pedidos recibidos, confirmados o en preparación); si no, `entrega` (hay pedidos listos); si no, null. " +
		"`minutos_en_espera` es null para `cobro`. " +
		"Si `momento` es anterior al primer hito registrado responde el error `sin_datos` con `detalle.historial_desde`: solo hay historial desde el despliegue del registro de hitos. " +
		"Se reconstruye con hasta 24 horas de hitos previos al momento: una cuenta abierta hace más tiempo puede aparecer sin sus pedidos más viejos. " +
		"Lo que NO se sabe: `entregado` registra cuándo se marcó, no necesariamente cuándo se entregó físicamente; " +
		"no hay registro de cuándo se pidió la cuenta en minutos ni de comensales en ese momento."

	descHistorialMesa = "Línea de tiempo de una mesa: cada hito registrado (cuenta abierta, pedido creado, cambios de estado del pedido y de ítems, cuenta solicitada, pago habilitado y aprobado, cuenta cerrada). " +
		"Sirve para preguntas como \"¿quién tomó el pedido de la mesa 5 en cocina y cuánto tardó?\" o \"¿dónde se demoró el pedido de la mesa 3?\". " +
		"`mesa` es el número de la mesa (obligatorio). `desde` y `hasta` aceptan ISO-8601 con zona o relativo (\"hace 2 horas\", \"90m\"); " +
		"por defecto `hasta` es ahora y `desde` son 12 horas antes de `hasta`. " +
		"`hitos` va ordenado por `ts` con `hito` (tipo), `pedido_id`, `item_id`, `estado_anterior`, `estado_nuevo`, `quien` y `rol` (quien hizo el cambio: usuario, \"comensal\" por QR o sistema), " +
		"`modo_entrega` (solo al entregar) y `minutos_desde_anterior` respecto del hito previo de la lista (null en el primero). " +
		"`etapas` resume cada pedido en minutos: `minutos_hasta_confirmacion` (recibido a confirmado), `minutos_espera_cocina` (confirmado hasta que cocina empieza a preparar), " +
		"`minutos_preparacion` (preparando a listo), `minutos_a_entrega` (listo a entregado) y `modo_entrega`; las etapas que faltan son null. " +
		"Lo que NO se sabe: `entregado` registra el momento en que se marcó, no necesariamente cuándo se entregó físicamente; " +
		"hay historial solo desde el despliegue del registro de hitos, y no hay hitos de antes."

	ejemplosMomento = "\"hace 60 min\", \"hace 2 horas\", \"60m\", \"2h\" o \"2026-10-22T21:00:00-03:00\""
)

type entradaEstadoMesas struct {
	SucursalID string `json:"sucursal_id,omitempty" jsonschema:"ID de la sucursal; opcional si el negocio tiene una sola"`
	Momento    string `json:"momento,omitempty" jsonschema:"instante a reconstruir: ISO-8601 con zona o relativo (hace 60 min, 2h); por defecto ahora"`
	Mesa       *int   `json:"mesa,omitempty" jsonschema:"número de mesa; si se omite se listan todas las mesas activas"`
}

type entradaHistorialMesa struct {
	SucursalID string `json:"sucursal_id,omitempty" jsonschema:"ID de la sucursal; opcional si el negocio tiene una sola"`
	Mesa       int    `json:"mesa" jsonschema:"número de la mesa"`
	Desde      string `json:"desde,omitempty" jsonschema:"inicio del rango: ISO-8601 con zona o relativo (hace 2 horas); por defecto 12 horas antes de hasta"`
	Hasta      string `json:"hasta,omitempty" jsonschema:"fin del rango: ISO-8601 con zona o relativo; por defecto ahora"`
}

func errParametroInvalido(parametro, valor string, causa error) error {
	mensaje := "No pude interpretar `" + parametro + "`. Usá " + ejemplosMomento + "."
	if errors.Is(causa, ErrMomentoFuturo) {
		mensaje = "`" + parametro + "` no puede ser futuro: solo hay historial hasta ahora."
	}
	return &errTool{
		Codigo: CodigoParametroInvalido, Mensaje: mensaje,
		Detalle: map[string]any{"parametro": parametro, "valor": valor, "ejemplos": []string{"hace 60 min", "hace 2 horas", "60m", "2h", "2026-10-22T21:00:00-03:00"}},
	}
}

func errMesaNoEncontrada(numero int) error {
	return &errTool{
		Codigo: CodigoMesaNoEncontrada, Mensaje: "No existe la mesa " + strconv.Itoa(numero) + " en esta sucursal.",
		Detalle: map[string]any{"mesa": numero},
	}
}

func (h *herramientas) estadoMesas(ctx context.Context, _ *sdk.CallToolRequest, in entradaEstadoMesas) (*sdk.CallToolResult, SalidaEstadoMesas, error) {
	return ejecutar(ctx, h, "estado_mesas", func() (SalidaEstadoMesas, error) {
		suc, err := h.resolverSucursal(ctx, h.claims.TenantID, in.SucursalID)
		if err != nil {
			return SalidaEstadoMesas{}, err
		}
		loc := ubicacionNegocio()
		ahora := h.deps.Ahora()
		momento, err := parsearMomento(in.Momento, ahora)
		if err != nil {
			return SalidaEstadoMesas{}, errParametroInvalido("momento", in.Momento, err)
		}
		primero, err := h.deps.Store.PrimerEvento(ctx, h.claims.TenantID, suc.ID)
		if err != nil {
			return SalidaEstadoMesas{}, err
		}
		if primero == nil || momento.Before(*primero) {
			var desde any
			mensaje := "Todavía no hay hitos registrados en esta sucursal."
			if primero != nil {
				desde = formatoTS(primero.In(loc))
				mensaje = "No hay historial anterior a " + formatoTS(primero.In(loc)) + "."
			}
			return SalidaEstadoMesas{}, &errTool{
				Codigo: CodigoSinDatos, Mensaje: mensaje + " El registro de hitos empezó recién en esa fecha.",
				Detalle: map[string]any{"historial_desde": desde, "momento": formatoTS(momento.In(loc))},
			}
		}
		mesas, err := h.deps.Store.MesasActivas(ctx, h.claims.TenantID, suc.ID)
		if err != nil {
			return SalidaEstadoMesas{}, err
		}
		if in.Mesa != nil {
			filtradas := []MesaRef{}
			for _, m := range mesas {
				if m.Numero == *in.Mesa {
					filtradas = append(filtradas, m)
				}
			}
			if len(filtradas) == 0 {
				return SalidaEstadoMesas{}, errMesaNoEncontrada(*in.Mesa)
			}
			mesas = filtradas
		}
		// hasta es exclusivo en la lectura: se suma la resolución de Postgres para incluir el propio instante.
		eventos, err := h.deps.Store.EventosSucursal(ctx, h.claims.TenantID, suc.ID,
			momento.Add(-ventanaReconstruccion), momento.Add(time.Microsecond))
		if err != nil {
			return SalidaEstadoMesas{}, err
		}
		estados := reconstruirMesas(mesas, eventos, momento)

		var abiertas int
		var esperando Esperando
		for _, m := range estados {
			if m.CuentaAbierta {
				abiertas++
			}
			sumarEspera(&esperando, m.Espera)
		}
		resumen := suc.Nombre + " el " + formatoTS(momento.In(loc)) + ": " + strconv.Itoa(abiertas) + " de " +
			cantidad(len(estados), "mesa", "mesas") + " con cuenta abierta. Mesas esperando: comida " +
			strconv.Itoa(esperando.Comida) + ", entrega " + strconv.Itoa(esperando.Entrega) + ", cobro " + strconv.Itoa(esperando.Cobro) + "."
		return SalidaEstadoMesas{
			Resumen: resumen, SucursalID: suc.ID, SucursalNombre: suc.Nombre, TS: formatoTS(ahora.In(loc)),
			Momento: formatoTS(momento.In(loc)), HistorialDesde: formatoTS(primero.In(loc)), Mesas: estados,
		}, nil
	})
}

func (h *herramientas) historialMesa(ctx context.Context, _ *sdk.CallToolRequest, in entradaHistorialMesa) (*sdk.CallToolResult, SalidaHistorialMesa, error) {
	return ejecutar(ctx, h, "historial_mesa", func() (SalidaHistorialMesa, error) {
		suc, err := h.resolverSucursal(ctx, h.claims.TenantID, in.SucursalID)
		if err != nil {
			return SalidaHistorialMesa{}, err
		}
		loc := ubicacionNegocio()
		ahora := h.deps.Ahora()
		hasta, err := parsearMomento(in.Hasta, ahora)
		if err != nil {
			return SalidaHistorialMesa{}, errParametroInvalido("hasta", in.Hasta, err)
		}
		desde := hasta.Add(-12 * time.Hour)
		if in.Desde != "" {
			if desde, err = parsearMomento(in.Desde, ahora); err != nil {
				return SalidaHistorialMesa{}, errParametroInvalido("desde", in.Desde, err)
			}
		}
		if desde.After(hasta) {
			return SalidaHistorialMesa{}, &errTool{
				Codigo: CodigoParametroInvalido, Mensaje: "`desde` es posterior a `hasta`.",
				Detalle: map[string]any{"parametro": "desde", "valor": in.Desde, "ejemplos": []string{"hace 2 horas", "90m"}},
			}
		}
		mesa, err := h.deps.Store.MesaPorNumero(ctx, h.claims.TenantID, suc.ID, in.Mesa)
		if errors.Is(err, ErrMesaNoEncontrada) {
			return SalidaHistorialMesa{}, errMesaNoEncontrada(in.Mesa)
		}
		if err != nil {
			return SalidaHistorialMesa{}, err
		}
		eventos, err := h.deps.Store.EventosMesa(ctx, h.claims.TenantID, suc.ID, mesa.ID, desde, hasta.Add(time.Microsecond))
		if err != nil {
			return SalidaHistorialMesa{}, err
		}
		hitos, etapas := construirHitos(eventos), calcularEtapas(eventos)
		resumen := suc.Nombre + ", mesa " + strconv.Itoa(mesa.Numero) + ": " + cantidad(len(hitos), "hito", "hitos") +
			" y " + cantidad(len(etapas), "pedido", "pedidos") + " entre " + formatoTS(desde.In(loc)) + " y " + formatoTS(hasta.In(loc)) + "."
		return SalidaHistorialMesa{
			Resumen: resumen, SucursalID: suc.ID, SucursalNombre: suc.Nombre, TS: formatoTS(ahora.In(loc)),
			Mesa: mesa.Numero, Desde: formatoTS(desde.In(loc)), Hasta: formatoTS(hasta.In(loc)), Hitos: hitos, Etapas: etapas,
		}, nil
	})
}
