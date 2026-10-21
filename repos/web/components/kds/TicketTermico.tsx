// repos/web/components/kds/TicketTermico.tsx
"use client";
import React, { useState, useEffect } from "react";
import { PedidoAPI } from "@/lib/api";
import Logo from "@/components/brand/Logo";

export type AnchoTicketTermico = "58mm" | "80mm";

export interface ItemTicketTermico {
  id: string;
  nombre: string;
  cantidad: number;
  precioUnitario?: number;
  notas?: string;
  comensalNombre?: string;
  variantes?: Array<{ nombre: string; precio_adicional?: number }>;
}

export interface DatosTicketTermico {
  id: string;
  numeroPedido: string;
  mesa: number;
  sector?: string;
  sucursalNombre?: string;
  fechaHora: string;
  items: ItemTicketTermico[];
  mostrarPrecios?: boolean;
  esReimpresion?: boolean;
}

const STORAGE_KEY_ANCHO = "mc_ticket_termico_ancho";
const STORAGE_KEY_AUTOPRINT = "mc_ticket_termico_autoprint";

export function getAnchoTicketTermico(): AnchoTicketTermico {
  if (typeof window === "undefined") return "80mm";
  const val = localStorage.getItem(STORAGE_KEY_ANCHO);
  return val === "58mm" ? "58mm" : "80mm";
}

export function setAnchoTicketTermico(ancho: AnchoTicketTermico): void {
  if (typeof window === "undefined") return;
  localStorage.setItem(STORAGE_KEY_ANCHO, ancho);
}

export function isAutoPrintEnabled(): boolean {
  if (typeof window === "undefined") return false;
  return localStorage.getItem(STORAGE_KEY_AUTOPRINT) === "true";
}

export function setAutoPrintEnabled(activo: boolean): void {
  if (typeof window === "undefined") return;
  localStorage.setItem(STORAGE_KEY_AUTOPRINT, activo ? "true" : "false");
}

export function construirDatosTicketDesdePedidos(
  pedidos: PedidoAPI[],
  numeroMesa: number,
  options?: {
    sector?: string;
    sucursalNombre?: string;
    mostrarPrecios?: boolean;
    esReimpresion?: boolean;
  }
): DatosTicketTermico {
  const principal = pedidos[0];
  const fechaBase = principal?.created_at ? new Date(principal.created_at) : new Date();
  const fechaHora = fechaBase.toLocaleString("es-AR", {
    day: "2-digit",
    month: "2-digit",
    year: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });

  const numeroPedido =
    pedidos.length > 1
      ? pedidos.map((p) => `#${p.id.slice(0, 6).toUpperCase()}`).join(" / ")
      : `#${(principal?.id || "000000").slice(0, 6).toUpperCase()}`;

  const items: ItemTicketTermico[] = pedidos.flatMap((p) =>
    (p.items || []).map((it) => ({
      id: it.id || `${p.id}-${it.articulo_id}`,
      nombre: it.nombre_articulo?.trim() || "Artículo",
      cantidad: it.cantidad,
      precioUnitario: it.precio_unitario,
      notas: it.notas?.trim() || undefined,
      comensalNombre: it.comensal_nombre?.trim() || undefined,
      variantes: (it.variantes || []).map((v) => ({
        nombre: v.nombre,
        precio_adicional: v.precio_adicional,
      })),
    }))
  );

  return {
    id: principal?.id || `ticket-${Date.now()}`,
    numeroPedido,
    mesa: numeroMesa,
    sector: options?.sector,
    sucursalNombre: options?.sucursalNombre,
    fechaHora,
    items,
    mostrarPrecios: options?.mostrarPrecios ?? false,
    esReimpresion: options?.esReimpresion ?? false,
  };
}

/**
 * Vista de comanda térmica monoespaciada de alto contraste (58mm y 80mm).
 */
export function TicketTermico({
  datos,
  ancho = "80mm",
}: {
  datos: DatosTicketTermico;
  ancho?: AnchoTicketTermico;
}) {
  const es58 = ancho === "58mm";
  const totalUnidades = datos.items.reduce((acc, it) => acc + it.cantidad, 0);
  const totalMonto = datos.items.reduce(
    (acc, it) => acc + (it.precioUnitario || 0) * it.cantidad,
    0
  );

  return (
    <div
      className={`mc-ticket-termico mx-auto bg-white font-mono text-black selection:bg-black selection:text-white ${
        es58 ? "w-[58mm] max-w-[58mm] p-[2.5mm] text-[11px]" : "w-[80mm] max-w-[80mm] p-[4mm] text-[12.5px]"
      }`}
      style={{
        lineHeight: 1.35,
        color: "#000000",
        backgroundColor: "#FFFFFF",
      }}
    >
      {/* CABECERA COMANDA */}
      <div className="border-b-2 border-dashed border-black pb-2 text-center">
        <div className="mb-1.5 flex justify-center">
          <Logo mark="full" ink="#2A1D17" lens="#FFFFFF" className="h-[24px] w-auto" />
        </div>
        <p className="text-[10px] font-bold tracking-widest uppercase">
          {datos.mostrarPrecios ? "TICKET DE MESA / PRE-CUENTA" : "COMANDA DE COCINA"}
        </p>
        {datos.sucursalNombre && (
          <p className="mt-0.5 text-[10px] font-semibold uppercase">{datos.sucursalNombre}</p>
        )}
        {datos.esReimpresion && (
          <p className="mt-1 inline-block border border-black px-1.5 py-0.5 text-[9px] font-black uppercase">
            *** REIMPRESIÓN ***
          </p>
        )}
        <div className={`mt-1.5 font-black tracking-tight ${es58 ? "text-[18px]" : "text-[22px]"}`}>
          MESA {datos.mesa}
        </div>
        {datos.sector && (
          <p className="text-[11px] font-bold uppercase">SECTOR: {datos.sector}</p>
        )}
      </div>

      {/* METADATOS DEL PEDIDO */}
      <div className="border-b border-dashed border-black py-1.5 text-[10.5px]">
        <div className="flex justify-between gap-2">
          <span className="font-bold">PEDIDO:</span>
          <span className="font-black">{datos.numeroPedido}</span>
        </div>
        <div className="flex justify-between gap-2">
          <span className="font-bold">FECHA/HORA:</span>
          <span>{datos.fechaHora}</span>
        </div>
      </div>

      {/* ENCABEZADO DE COLUMNAS */}
      <div className="flex justify-between border-b-2 border-black py-1 text-[10px] font-black uppercase">
        <span>CANT. / DETALLE</span>
        {datos.mostrarPrecios && <span>SUBTOTAL</span>}
      </div>

      {/* DETALLE DE ÍTEMS */}
      <div className="divide-y divide-dashed divide-black/60 py-1">
        {datos.items.map((item) => {
          const subtotal = (item.precioUnitario || 0) * item.cantidad;
          return (
            <div key={item.id} className="py-1.5">
              <div className="flex items-start justify-between gap-2">
                <div className="min-w-0 flex-1">
                  <span className={`font-black ${es58 ? "text-[12px]" : "text-[13.5px]"}`}>
                    {item.cantidad}x{" "}
                  </span>
                  <span className={`font-black uppercase ${es58 ? "text-[12px]" : "text-[13.5px]"}`}>
                    {item.nombre}
                  </span>
                </div>
                {datos.mostrarPrecios && item.precioUnitario !== undefined && (
                  <span className="shrink-0 font-bold">
                    ${subtotal.toLocaleString("es-AR")}
                  </span>
                )}
              </div>

              {/* MODIFICADORES / VARIANTES DESTACADAS */}
              {item.variantes && item.variantes.length > 0 && (
                <div className="mt-0.5 pl-3">
                  {item.variantes.map((v, idx) => (
                    <p key={idx} className="text-[10.5px] font-bold uppercase">
                      + {v.nombre}
                    </p>
                  ))}
                </div>
              )}

              {/* COMENSAL */}
              {item.comensalNombre && (
                <p className="mt-0.5 pl-3 text-[10px] font-semibold">
                  Comensal: {item.comensalNombre}
                </p>
              )}

              {/* NOTAS DESTACADAS EN ALTO CONTRASTE */}
              {item.notas && (
                <div className="mt-1 border border-black px-1.5 py-0.5 text-[10.5px] font-black uppercase">
                  *** NOTA: {item.notas} ***
                </div>
              )}
            </div>
          );
        })}
      </div>

      {/* PIE DEL TICKET */}
      <div className="mt-1 border-t-2 border-black pt-1.5">
        <div className="flex justify-between text-[11px] font-black uppercase">
          <span>TOTAL ÍTEMS:</span>
          <span>{totalUnidades} UNID.</span>
        </div>
        {datos.mostrarPrecios && (
          <div className="mt-1 flex justify-between text-[14px] font-black uppercase">
            <span>TOTAL:</span>
            <span>${totalMonto.toLocaleString("es-AR")}</span>
          </div>
        )}
      </div>

      <div className="mt-3 border-t border-dashed border-black pt-2 text-center text-[9.5px] font-semibold">
        <p>fudd · COMANDA TÉRMICA ({ancho})</p>
      </div>
    </div>
  );
}

/**
 * Ejecuta la impresión térmica del navegador y retorna true si el disparo fue exitoso.
 */
export function dispararImpresionNavegador(): boolean {
  if (typeof window === "undefined" || typeof window.print !== "function") {
    return false;
  }
  try {
    document.body.classList.add("mc-printing-ticket");
    window.print();
    setTimeout(() => {
      document.body.classList.remove("mc-printing-ticket");
    }, 500);
    return true;
  } catch (err) {
    console.error("Error disparando impresión térmica:", err);
    document.body.classList.remove("mc-printing-ticket");
    return false;
  }
}

export interface TicketTermicoModalProps {
  isOpen: boolean;
  datos: DatosTicketTermico | null;
  ancho: AnchoTicketTermico;
  onCambiarAncho: (ancho: AnchoTicketTermico) => void;
  autoPrint: boolean;
  onCambiarAutoPrint: (activo: boolean) => void;
  errorImpresion?: string | null;
  onImprimir: () => void;
  onClose: () => void;
}

export function TicketTermicoModal({
  isOpen,
  datos,
  ancho,
  onCambiarAncho,
  autoPrint,
  onCambiarAutoPrint,
  errorImpresion,
  onImprimir,
  onClose,
}: TicketTermicoModalProps) {
  const [mostrarGuia, setMostrarGuia] = useState(false);

  useEffect(() => {
    if (!isOpen) return;
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape") onClose();
    }
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [isOpen, onClose]);

  if (!isOpen || !datos) return null;

  return (
    <div
      className="no-print fixed inset-0 z-50 flex items-center justify-center bg-black/75 p-12 sm:p-24 font-inter"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-label="Impresión de comanda térmica"
        className="flex max-h-[calc(100vh-32px)] w-full max-w-xl flex-col overflow-hidden rounded-xl border border-concrete bg-canvas-white shadow-2xl"
      >
        {/* HEADER */}
        <div className="flex items-center justify-between border-b border-concrete px-16 py-12 sm:px-20">
          <div className="flex items-center gap-10">
            <span className="flex h-36 w-36 items-center justify-center rounded-lg bg-plain-green text-canvas-white">
              <span className="material-symbols-outlined text-20">print</span>
            </span>
            <div>
              <h2 className="text-16 font-black text-ash-graphite">
                Comanda Térmica · Mesa {datos.mesa}
              </h2>
              <p className="text-11 text-sage-green">
                Formato ESC/POS compatible con impresoras de 58 mm y 80 mm
              </p>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="flex h-36 w-36 cursor-pointer items-center justify-center rounded-lg border border-concrete text-ash-graphite hover:bg-ghost-fog"
            aria-label="Cerrar modal de impresión"
          >
            <span className="material-symbols-outlined text-18">close</span>
          </button>
        </div>

        {/* CONTROLES DE FORMATO Y AUTO-IMPRESIÓN */}
        <div className="flex flex-wrap items-center justify-between gap-12 border-b border-ghost-fog bg-ghost-fog/60 px-16 py-10 sm:px-20">
          <div className="flex items-center gap-8">
            <span className="text-12 font-semibold text-ash-graphite">Ancho de papel:</span>
            <div className="inline-flex rounded-lg border border-concrete bg-canvas-white p-2">
              <button
                type="button"
                onClick={() => onCambiarAncho("58mm")}
                className={`cursor-pointer rounded-md px-10 py-4 text-12 font-bold transition-colors ${
                  ancho === "58mm"
                    ? "bg-plain-green text-canvas-white"
                    : "text-sage-green hover:text-ash-graphite"
                }`}
              >
                58 mm
              </button>
              <button
                type="button"
                onClick={() => onCambiarAncho("80mm")}
                className={`cursor-pointer rounded-md px-10 py-4 text-12 font-bold transition-colors ${
                  ancho === "80mm"
                    ? "bg-plain-green text-canvas-white"
                    : "text-sage-green hover:text-ash-graphite"
                }`}
              >
                80 mm
              </button>
            </div>
          </div>

          <label className="flex cursor-pointer items-center gap-8 text-12 font-semibold text-ash-graphite">
            <input
              type="checkbox"
              checked={autoPrint}
              onChange={(e) => onCambiarAutoPrint(e.target.checked)}
              className="h-16 w-16 accent-plain-green"
            />
            <span>Auto-imprimir nuevos pedidos</span>
          </label>
        </div>

        {/* ALERTA DE FALLO Y REINTENTO MANUAL */}
        {errorImpresion && (
          <div className="mx-16 mt-12 flex items-center justify-between gap-12 rounded-lg border border-alert-red/40 bg-red-50 px-12 py-10 text-12 text-alert-red sm:mx-20">
            <div className="flex items-center gap-8">
              <span className="material-symbols-outlined text-18">error</span>
              <span className="font-semibold">{errorImpresion}</span>
            </div>
            <button
              type="button"
              onClick={onImprimir}
              className="shrink-0 cursor-pointer rounded-md bg-alert-red px-10 py-6 text-11 font-bold text-white hover:opacity-90"
            >
              Reintentar manual
            </button>
          </div>
        )}

        {/* VISTA PREVIA DEL TICKET */}
        <div className="flex-1 overflow-y-auto bg-[#E5E7EB] p-16 sm:p-20">
          <div className="mx-auto w-fit rounded-sm shadow-lg ring-1 ring-black/15">
            <TicketTermico datos={datos} ancho={ancho} />
          </div>

          {/* DOCUMENTACIÓN INTEGRACIÓN IMPRESORAS USB / RED */}
          <div className="mt-16 rounded-lg border border-concrete bg-canvas-white p-12">
            <button
              type="button"
              onClick={() => setMostrarGuia((prev) => !prev)}
              className="flex w-full cursor-pointer items-center justify-between text-left text-12 font-bold text-ash-graphite"
            >
              <span className="flex items-center gap-6">
                <span className="material-symbols-outlined text-16 text-sage-green">help_outline</span>
                Guía de conexión con impresoras térmicas USB / Red (ESC/POS)
              </span>
              <span className="material-symbols-outlined text-18">
                {mostrarGuia ? "expand_less" : "expand_more"}
              </span>
            </button>
            {mostrarGuia && (
              <ul className="mt-8 space-y-4 border-t border-ghost-fog pt-8 text-11 leading-relaxed text-sage-green">
                <li>
                  • <strong>Impresora USB o Red (LAN/Wi-Fi):</strong> instalá la comandera térmica (Epson TM-T20, Hasar, Posiflex, Xprinter 58/80mm) como impresora predeterminada del sistema operativo.
                </li>
                <li>
                  • <strong>Ajustes del diálogo de impresión:</strong> desmarcá <em>&ldquo;Encabezados y pies de página&rdquo;</em>, seleccioná <em>Márgenes: Ninguno</em> y <em>Escala: 100%</em>.
                </li>
                <li>
                  • <strong>Impresión directa sin confirmación (Kiosk Printing):</strong> en Chrome/Edge de cocina iniciá el navegador con el flag <code className="rounded bg-ghost-fog px-4 py-1 font-mono text-ash-graphite">--kiosk-printing</code> y activá el switch <strong>Auto-imprimir nuevos pedidos</strong>.
                </li>
              </ul>
            )}
          </div>
        </div>

        {/* FOOTER DE ACCIONES */}
        <div className="flex items-center justify-end gap-10 border-t border-concrete bg-canvas-white px-16 py-12 sm:px-20">
          <button
            type="button"
            onClick={onClose}
            className="min-h-44 cursor-pointer rounded-lg border border-concrete px-14 py-8 text-12 font-bold text-ash-graphite hover:bg-ghost-fog"
          >
            Cerrar
          </button>
          <button
            type="button"
            onClick={onImprimir}
            className="flex min-h-44 cursor-pointer items-center gap-6 rounded-lg bg-plain-green px-18 py-8 text-13 font-black text-canvas-white shadow-sm hover:bg-plain-green-muted"
          >
            <span className="material-symbols-outlined text-18">print</span>
            <span>{datos.esReimpresion ? "Reimprimir Ticket" : "Imprimir Ticket"}</span>
          </button>
        </div>
      </div>
    </div>
  );
}

/**
 * Contenedor invisible en pantalla y visible únicamente en @media print.
 */
export function TicketTermicoPrintPortal({
  datos,
  ancho,
}: {
  datos: DatosTicketTermico | null;
  ancho: AnchoTicketTermico;
}) {
  if (!datos) return null;

  return (
    <>
      <style>{`
        @media screen {
          #mc-ticket-termico-print-root {
            display: none !important;
          }
        }
        @media print {
          @page {
            size: ${ancho} auto;
            margin: 0mm;
          }
          body.mc-printing-ticket > *:not(#mc-ticket-termico-print-root) {
            display: none !important;
          }
          body.mc-printing-ticket .no-print {
            display: none !important;
          }
          #mc-ticket-termico-print-root {
            display: block !important;
            position: absolute !important;
            left: 0 !important;
            top: 0 !important;
            width: ${ancho} !important;
            margin: 0 !important;
            padding: 0 !important;
            background: #FFFFFF !important;
            color: #000000 !important;
            z-index: 999999 !important;
          }
        }
      `}</style>
      <div id="mc-ticket-termico-print-root" aria-hidden="true">
        <TicketTermico datos={datos} ancho={ancho} />
      </div>
    </>
  );
}
