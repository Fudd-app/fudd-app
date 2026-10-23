// repos/web/components/kds/ComandaCard.tsx
"use client";
import React, { useState, useEffect } from "react";
import { PedidoAPI, PedidoItemAPI } from "@/lib/api";

export interface ComandaCardProps {
  pedidos: PedidoAPI[];
  numeroMesa: number;
  nombreSector?: string;
  onCambiarEstadoItem: (itemId: string, nuevoEstado: "pendiente" | "preparando" | "listo") => Promise<void>;
  onComandasCompletasLista: (pedidoIds: string[]) => Promise<void>;
  onImprimirTicket?: (pedidos: PedidoAPI[], numeroMesa: number, nombreSector?: string) => void;
  ticketImpreso?: boolean;
  errorImpresion?: boolean;
}

export function ComandaCard({
  pedidos,
  numeroMesa,
  nombreSector,
  onCambiarEstadoItem,
  onComandasCompletasLista,
  onImprimirTicket,
  ticketImpreso = false,
  errorImpresion = false,
}: ComandaCardProps) {
  const [minutos, setMinutos] = useState<number>(0);
  const [segundosTexto, setSegundosTexto] = useState<string>("00:00");
  const [cargandoAccion, setCargandoAccion] = useState<boolean>(false);
  const [cargandoItemId, setCargandoItemId] = useState<string | null>(null);
  const pedidoPrincipal = pedidos[0]!;
  const fechaPrimerPedido = pedidoPrincipal.created_at;

  // El grupo conserva como prioridad el horario del primer pedido recibido.
  useEffect(() => {
    function calcularTiempo() {
      const fechaCreacion = new Date(fechaPrimerPedido).getTime();
      const ahora = Date.now();
      const difSegundos = Math.max(0, Math.floor((ahora - fechaCreacion) / 1000));
      const mins = Math.floor(difSegundos / 60);
      const secs = difSegundos % 60;

      setMinutos(mins);
      setSegundosTexto(`${String(mins).padStart(2, "0")}:${String(secs).padStart(2, "0")}`);
    }

    calcularTiempo();
    const interval = setInterval(calcularTiempo, 1000);
    return () => clearInterval(interval);
  }, [fechaPrimerPedido]);

  // Código cromático US-65: Verde (<10m), Amarillo (10-20m), Rojo (>20m)
  const colorTemporizador =
    minutos < 10
      ? {
          badge: "bg-[#EAF8F2] text-[#087657] border-[#B7EAD8]",
          cardBorder: "border-[#B7EAD8] hover:border-[#14A77B]",
          dot: "bg-[#14A77B]",
        }
      : minutos < 20
      ? {
          badge: "bg-[#FFF4DB] text-[#9A5700] border-[#F3D4A3]",
          cardBorder: "border-[#F3D4A3] hover:border-[#F2A51A]",
          dot: "bg-[#F2A51A]",
        }
      : {
          badge: "animate-pulse bg-red-50 text-alert-red border-alert-red/30",
          cardBorder: "border-alert-red/60 hover:border-alert-red shadow-alert-red/10 shadow-lg",
          dot: "bg-alert-red",
        };

  const items = pedidos.flatMap((pedido) => pedido.items || []);
  const todosListos = items.length > 0 && items.every((i) => i.estado === "listo");
  const pedidosListos = pedidos.every((pedido) => pedido.estado === "listo");
  const pedidoIds = pedidos.map((pedido) => pedido.id);

  // Manejo de tap táctil por ítem (US-66)
  async function handleItemTap(item: PedidoItemAPI) {
    if (cargandoItemId || cargandoAccion) return;
    const estadoActual = item.estado || "pendiente";

    // Rotación táctil de estados: pendiente -> preparando -> listo -> pendiente
    let nuevoEstado: "pendiente" | "preparando" | "listo" = "preparando";
    if (estadoActual === "pendiente") nuevoEstado = "preparando";
    else if (estadoActual === "preparando") nuevoEstado = "listo";
    else if (estadoActual === "listo") nuevoEstado = "pendiente";

    try {
      setCargandoItemId(item.id);
      await onCambiarEstadoItem(item.id, nuevoEstado);
    } finally {
      setCargandoItemId(null);
    }
  }

  async function handleComandaCompleta() {
    if (cargandoAccion) return;
    try {
      setCargandoAccion(true);
      await onComandasCompletasLista(pedidoIds);
    } finally {
      setCargandoAccion(false);
    }
  }

  return (
    <article
      className={`flex select-none flex-col overflow-hidden rounded-xl border-2 bg-canvas-white shadow-sm transition-all duration-200 ${colorTemporizador.cardBorder} ${
        todosListos ? "ring-2 ring-[#14A77B]/20" : ""
      }`}
    >
      {/* HEADER DE COMANDA */}
      <div className="flex items-center justify-between gap-8 border-b border-ghost-fog bg-canvas-white px-12 py-8">
        <div className="flex min-w-0 items-center gap-8">
          <div className="flex min-w-0 items-center gap-5">
            <span className="shrink-0 text-18 font-black leading-none tracking-tight text-ash-graphite">
              MESA {numeroMesa}
            </span>
            {nombreSector && (
              <span className="truncate text-10 font-semibold text-sage-green">
                · {nombreSector}
              </span>
            )}
          </div>
          <span className="rounded-md bg-ghost-fog px-6 py-3 text-10 font-mono font-semibold text-sage-green">
            {pedidos.length > 1 ? `${pedidos.length} pedidos` : `#${pedidoPrincipal.id.slice(0, 6)}`}
          </span>
        </div>

        {/* TEMPORIZADOR CROMÁTICO */}
        <div
          className={`flex shrink-0 items-center gap-5 rounded-lg border px-8 py-5 font-mono text-13 font-black ${colorTemporizador.badge}`}
          title="Tiempo transcurrido desde el primer pedido del grupo"
        >
          <span className={`inline-block h-8 w-8 rounded-full ${colorTemporizador.dot}`} />
          <span>{segundosTexto}</span>
        </div>
      </div>

      {/* LISTA DE ÍTEMS CON TAP TÁCTIL */}
      <div className="max-h-[360px] flex-1 space-y-6 overflow-y-auto p-8">
        {items.map((item) => {
          const est = item.estado || "pendiente";
          const estaListo = est === "listo";
          const estaPreparando = est === "preparando";
          const esItemCargando = cargandoItemId === item.id;

          return (
            <div
              key={item.id}
              onClick={() => handleItemTap(item)}
              role="button"
              tabIndex={0}
              onKeyDown={(e) => {
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault();
                  handleItemTap(item);
                }
              }}
              className={`group flex min-h-56 cursor-pointer items-start gap-8 rounded-lg border p-8 transition-all active:scale-[0.98] ${
                estaListo
                  ? "border-[#B7EAD8] bg-[#EAF8F2] text-sage-green"
                  : estaPreparando
                  ? "border-[#C9DCF7] bg-[#EFF6FF] text-ash-graphite"
                  : "border-concrete bg-canvas-white text-ash-graphite hover:border-stone hover:bg-ghost-fog/50"
              }`}
            >
              {/* CHECK / INDICADOR DE ESTADO */}
              <div
                className={`flex h-28 w-28 shrink-0 items-center justify-center rounded-md border text-13 font-black transition-colors ${
                  estaListo
                    ? "border-[#14A77B] bg-[#14A77B] text-white"
                    : estaPreparando
                    ? "border-[#4D8EDB] bg-[#EAF3FF] text-[#285F9F]"
                    : "border-stone bg-canvas-white text-sage-green group-hover:border-ash-graphite"
                }`}
              >
                <span className="material-symbols-outlined text-18">
                  {estaListo ? "check" : estaPreparando ? "skillet" : "radio_button_unchecked"}
                </span>
              </div>

              {/* DETALLE DEL ÍTEM */}
              <div className="flex-1 min-w-0">
                <div className="flex items-baseline justify-between gap-6">
                  <div className="flex items-baseline gap-6">
                    <span className="font-mono text-15 font-black text-ash-graphite">
                      {item.cantidad}×
                    </span>
                    <span
                      className={`text-14 font-bold leading-snug ${
                        estaListo ? "line-through text-sage-green" : "text-ash-graphite"
                      }`}
                    >
                      {item.nombre_articulo || "Artículo"}
                    </span>
                  </div>

                  <span
                    className={`shrink-0 rounded px-6 py-1 text-10 font-bold uppercase tracking-wider ${
                      estaListo
                        ? "bg-[#DDF4EA] text-[#087657]"
                        : estaPreparando
                        ? "bg-[#E2EEFC] text-[#285F9F]"
                        : "bg-ghost-fog text-sage-green"
                    }`}
                  >
                    {esItemCargando ? "..." : est}
                  </span>
                </div>

                {/* VARIANTES / OPCIONES */}
                {item.variantes && item.variantes.length > 0 && (
                  <div className="mt-2 flex flex-wrap gap-4">
                    {item.variantes.map((v) => (
                      <span
                        key={v.variante_id}
                        className="rounded-md border border-concrete bg-canvas-white px-6 py-2 text-10 font-semibold text-ash-graphite"
                      >
                        + {v.nombre}
                      </span>
                    ))}
                  </div>
                )}

                {/* NOTAS DEL COMENSAL / ALERTA DE COCINA */}
                {item.notas && item.notas.trim() !== "" && (
                  <div className="mt-4 flex items-center gap-4 rounded-md border border-[#F3D4A3] bg-[#FFF4DB] px-7 py-4 text-11 font-semibold text-[#7A4700]">
                    <span className="material-symbols-outlined text-16">warning</span>
                    <span className="font-black">Nota:</span>
                    <span>{item.notas}</span>
                  </div>
                )}

              </div>
            </div>
          );
        })}
      </div>

      {/* FOOTER CON BOTONES TÁCTILES RÁPIDOS (US-66 & US-67) */}
      <div className="flex gap-8 border-t border-ghost-fog bg-[#F7F8F8] p-8">
        {onImprimirTicket && (
          <button
            type="button"
            onClick={() => onImprimirTicket(pedidos, numeroMesa, nombreSector)}
            className={`flex min-h-48 shrink-0 cursor-pointer items-center justify-center gap-5 rounded-lg border px-10 text-12 font-bold transition-all active:scale-[0.98] ${
              errorImpresion
                ? "border-alert-red/50 bg-red-50 text-alert-red hover:bg-red-100"
                : ticketImpreso
                ? "border-concrete bg-canvas-white text-sage-green hover:border-stone hover:text-ash-graphite"
                : "border-concrete bg-canvas-white text-ash-graphite hover:border-stone hover:bg-ghost-fog"
            }`}
            title={
              errorImpresion
                ? "Fallo al imprimir — clic para reintentar"
                : ticketImpreso
                ? "Reimprimir ticket térmico (58/80 mm)"
                : "Imprimir ticket térmico (58/80 mm)"
            }
          >
            <span className="material-symbols-outlined text-18">
              {errorImpresion ? "print_error" : "print"}
            </span>
            <span>
              {errorImpresion
                ? "Reintentar"
                : ticketImpreso
                ? "Reimprimir"
                : "Imprimir ticket"}
            </span>
          </button>
        )}
        {!todosListos && !pedidosListos ? (
          <button
            type="button"
            onClick={handleComandaCompleta}
            disabled={cargandoAccion}
            className="flex min-h-48 flex-1 cursor-pointer items-center justify-center gap-7 rounded-lg bg-[#285F9F] px-10 text-13 font-black text-white shadow-sm transition-all hover:bg-[#1F548E] active:scale-[0.98] disabled:opacity-50"
          >
            {cargandoAccion ? (
              <span>Actualizando...</span>
            ) : (
              <>
                <span className="material-symbols-outlined text-18">done_all</span>
                <span>{pedidos.length > 1 ? "Marcar grupo listo" : "Marcar comanda lista"}</span>
              </>
            )}
          </button>
        ) : (
          // Cocina no entrega: el mozo marca "entregado" y recién ahí sale del KDS.
          <div
            className="animate-pulse-slow flex min-h-48 flex-1 items-center justify-center gap-7 rounded-lg bg-[#14A77B] px-10 text-13 font-black text-white shadow-md"
            role="status"
          >
            <span className="material-symbols-outlined text-18">notifications</span>
            <span>Listo para retirar</span>
          </div>
        )}
      </div>
    </article>
  );
}
