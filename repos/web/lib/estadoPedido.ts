import type { EstadoPedidoAPI } from "@/lib/api";

// El estado de un pedido solo avanza: una respuesta tardía del PATCH no puede
// pisar un estado más nuevo que ya trajo el SSE.
export const RANGO_ESTADO: Record<EstadoPedidoAPI, number> = {
  recibido: 0,
  confirmado: 1,
  preparando: 2,
  listo: 3,
  entregado: 4,
  cerrado: 5,
};

export function avanza(actual: EstadoPedidoAPI | undefined, nuevo: EstadoPedidoAPI): boolean {
  if (!actual) return true;
  return RANGO_ESTADO[nuevo] >= RANGO_ESTADO[actual];
}
