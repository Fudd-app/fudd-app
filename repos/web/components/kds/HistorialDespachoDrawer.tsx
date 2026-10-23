// repos/web/components/kds/HistorialDespachoDrawer.tsx
"use client";
import React from "react";
import { PedidoAPI } from "@/lib/api";

export interface HistorialDespachoDrawerProps {
  isOpen: boolean;
  onClose: () => void;
  pedidosDespachados: PedidoAPI[];
  mesasMap: Record<string, { numero: number; sector?: string }>;
}

export function HistorialDespachoDrawer({
  isOpen,
  onClose,
  pedidosDespachados,
  mesasMap,
}: HistorialDespachoDrawerProps) {
  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex justify-end">
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-black/45 backdrop-blur-sm transition-opacity"
        onClick={onClose}
        aria-hidden="true"
      />

      {/* Drawer Panel */}
      <aside className="animate-in slide-in-from-right relative z-10 flex w-full max-w-md flex-col border-l border-concrete bg-canvas-white text-ash-graphite shadow-2xl duration-200">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-concrete bg-canvas-white px-20 py-16">
          <div>
            <h2 className="flex items-center gap-8 text-18 font-black tracking-tight text-ash-graphite">
              <span className="material-symbols-outlined text-24">history</span>
              Historial de despachos
            </h2>
            <p className="mt-3 text-12 text-sage-green">
              Últimas comandas despachadas / retiradas de cocina
            </p>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="flex h-44 w-44 cursor-pointer items-center justify-center rounded-lg border border-concrete bg-canvas-white text-ash-graphite transition-colors hover:border-stone hover:bg-ghost-fog"
            title="Cerrar panel"
            aria-label="Cerrar historial"
          >
            <span className="material-symbols-outlined text-20">close</span>
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-16 space-y-12">
          {pedidosDespachados.length === 0 ? (
            <div className="flex h-64 flex-col items-center justify-center text-center text-sage-green">
              <span className="material-symbols-outlined mb-8 text-36 text-[#4D8EDB]">skillet</span>
              <p className="text-15 font-semibold text-ash-graphite">Sin comandas despachadas aún</p>
              <p className="mt-3 max-w-xs text-12 text-sage-green">
                A medida que se entreguen los pedidos, aparecerán aquí.
              </p>
            </div>
          ) : (
            pedidosDespachados.map((pedido) => {
              const mesaInfo = mesasMap[pedido.mesa_id];
              const numeroMesa = mesaInfo?.numero ?? "?";
              const sector = mesaInfo?.sector;
              const hora = new Date(pedido.updated_at || pedido.created_at).toLocaleTimeString([], {
                hour: "2-digit",
                minute: "2-digit",
              });

              return (
                <div
                  key={pedido.id}
                  className="space-y-10 rounded-xl border border-concrete bg-[#F7F8F8] p-14 transition-colors hover:border-stone"
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-8">
                      <span className="text-16 font-black text-[#087657]">
                        MESA {numeroMesa}
                      </span>
                      {sector && (
                        <span className="text-11 text-sage-green">
                          ({sector})
                        </span>
                      )}
                      <span className="rounded bg-ghost-fog px-6 py-2 font-mono text-10 text-sage-green">
                        #{pedido.id.slice(0, 6)}
                      </span>
                    </div>
                    <span className="flex items-center gap-4 text-11 font-mono text-sage-green">
                      <span className="material-symbols-outlined text-14">schedule</span>
                      {hora}
                    </span>
                  </div>

                  {/* Resumen de ítems */}
                  <div className="space-y-4 border-t border-concrete pt-8">
                    {pedido.items && pedido.items.length > 0 ? (
                      pedido.items.map((item, idx) => (
                        <div
                          key={item.id || idx}
                          className="flex items-center justify-between text-12 text-ash-graphite"
                        >
                          <span className="truncate pr-8">
                            <strong className="font-mono text-ash-graphite">{item.cantidad}×</strong>{" "}
                            {item.nombre_articulo || "Artículo"}
                          </span>
                          <span className="shrink-0 font-mono text-10 font-semibold text-[#087657]">
                            ✓ Listo
                          </span>
                        </div>
                      ))
                    ) : (
                      <span className="text-12 italic text-sage-green">Comanda sin ítems</span>
                    )}
                  </div>
                </div>
              );
            })
          )}
        </div>

        {/* Footer */}
        <div className="border-t border-concrete bg-[#F7F8F8] p-16 text-center">
          <p className="text-11 text-sage-green">
            KDS fudd · Pantalla dedicada de cocina
          </p>
        </div>
      </aside>
    </div>
  );
}
