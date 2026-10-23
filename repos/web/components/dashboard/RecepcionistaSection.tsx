"use client";
import { useState, useEffect, useRef, useCallback, useMemo } from "react";
import { avanza } from "@/lib/estadoPedido";
import { api, codigoError, getErrorMessage, EstadoPedidoAPI, ModoEntregaPedido, PedidoAPI, Sucursal, MesaAPI } from "@/lib/api";
import { agruparPorComensal } from "@/lib/desgloseCuenta";
import { EmptyState, Skeleton, useToast, useConfirm } from "@/components/ui";
import { playOrderReadySound } from "@/components/kds/AudioAlerts";
import {
  AnchoTicketTermico,
  DatosTicketTermico,
  TicketTermicoModal,
  TicketTermicoPrintPortal,
  dispararImpresionNavegador,
  getAnchoTicketTermico,
  isAutoPrintEnabled,
  setAnchoTicketTermico,
  setAutoPrintEnabled,
} from "@/components/kds/TicketTermico";
import PedidoAsistidoModal from "./PedidoAsistidoModal";

export interface PedidoVista {
  id: string;
  mesa: number;
  mesaId?: string;
  sucursalId?: string;
  timestamp: string;
  creadoEn?: string;
  estado: 'recibido' | 'confirmado' | 'preparando' | 'listo' | 'entregado';
  items: Array<{
    id: string;
    nombre: string;
    cantidad: number;
    precio: number;
    nota?: string;
    comensalId?: string;
    comensalNombre?: string;
    variantes?: Array<{ variante_id: string; nombre: string; precio_adicional: number }>;
  }>;
  cuentaSolicitada?: boolean;
  finalizado?: boolean;
}

interface MesaGrupo {
  key: string;
  mesa: number;
  mesaId?: string;
  pedidos: PedidoVista[];
  cuentaSolicitada: boolean;
  pagoHabilitado: boolean;
}

const ESTADO_LABELS: Record<PedidoVista['estado'], string> = {
  recibido: 'Por confirmar',
  confirmado: 'Confirmado · en cola de cocina',
  preparando: 'En preparación',
  listo: 'Listo para retirar',
  entregado: 'Entregado',
};

const ESTADO_STYLES: Record<PedidoVista['estado'], string> = {
  recibido: 'border-[#F7DCA8] bg-[#FFF4DB] text-[#9A5700]',
  confirmado: 'bg-ghost-fog text-ash-graphite border-concrete',
  preparando: 'border-[#BFDBFE] bg-[#EFF6FF] text-[#1D4ED8]',
  listo: 'bg-success/15 text-[#087645] border-success/30',
  entregado: 'bg-ghost-fog text-sage-green border-concrete',
};

// "cerrado" llega de la API solo para pedidos ya cobrados; se muestra como entregado y finalizado.
function estadoVista(estado: EstadoPedidoAPI): Pick<PedidoVista, 'estado' | 'finalizado'> {
  if (estado === 'cerrado') return { estado: 'entregado', finalizado: true };
  return { estado, finalizado: false };
}

// Estado de la API equivalente al de una vista (los finalizados llegan como "cerrado").
function estadoApiDeVista(p: Pick<PedidoVista, 'estado' | 'finalizado'>): EstadoPedidoAPI {
  return p.finalizado ? 'cerrado' : p.estado;
}

function mensajeErrorPedido(err: unknown): string {
  switch (codigoError(err)) {
    case 'PEDIDO_SIN_CONFIRMAR':
      return 'El pedido todavía no fue confirmado. Confirmalo antes de enviarlo a cocina.';
    case 'TRANSICION_INVALIDA':
      return 'El pedido ya cambió de estado. Revisá la vista y reintentá.';
    case 'SIN_PERMISO':
      return 'Tu rol no tiene permiso para realizar esta acción.';
    case 'MODO_ENTREGA_INVALIDO':
      return 'Indicá si se entregó en la mesa o lo retiró el cliente.';
    default:
      return getErrorMessage(err, 'No se pudo actualizar el pedido. Intentá nuevamente.');
  }
}

function minutosDesde(iso: string | undefined, ahora: number): number | null {
  if (!iso) return null;
  const t = Date.parse(iso);
  if (Number.isNaN(t)) return null;
  return Math.max(0, Math.floor((ahora - t) / 60000));
}

function textoAntiguedad(minutos: number | null): string {
  if (minutos === null) return '';
  return minutos < 1 ? 'Recién llegó' : `Hace ${minutos} min`;
}

type MesaCardTone = 'account' | 'ready' | 'progress';

const MESA_CARD_TONES: Record<MesaCardTone, {
  border: string;
  accent: string;
  badge: string;
  dot: string;
  action: string;
}> = {
  account: {
    border: 'border-[#F3D4A3]',
    accent: 'border-l-[#F2A51A]',
    badge: 'border-[#F7DCA8] bg-[#FFF4DB] text-[#9A5700]',
    dot: 'bg-[#F2A51A]',
    action: 'border-[#F3D4A3] bg-[#FFF9EF] text-[#8A4D00] hover:bg-[#FFF2D9]',
  },
  ready: {
    border: 'border-[#A7E4CC]',
    accent: 'border-l-[#14A77B]',
    badge: 'border-[#B7EAD8] bg-[#EAF8F2] text-[#087657]',
    dot: 'bg-[#14A77B]',
    action: 'border-[#B7EAD8] bg-[#F0FBF7] text-[#087657] hover:bg-[#E2F7EF]',
  },
  progress: {
    border: 'border-[#C9DCF7]',
    accent: 'border-l-[#4D8EDB]',
    badge: 'border-[#CFE1FA] bg-[#EFF6FF] text-[#285F9F]',
    dot: 'bg-[#4D8EDB]',
    action: 'border-[#D6E5F8] bg-[#F7FAFF] text-[#285F9F] hover:bg-[#EDF5FF]',
  },
};

function calcularEstadoMesa(pedidos: PedidoVista[]): PedidoVista['estado'] {
  const vigentes = pedidos.filter(pedido => pedido.estado !== 'entregado');
  if (pedidos.length > 0 && vigentes.length === 0) return 'entregado';
  if (vigentes.every(pedido => pedido.estado === 'listo')) return 'listo';
  if (vigentes.some(pedido => pedido.estado === 'preparando')) return 'preparando';
  if (vigentes.some(pedido => pedido.estado === 'recibido')) return 'recibido';
  return 'confirmado';
}

const BTN_ENTREGA_BASE = 'flex min-h-44 flex-1 cursor-pointer items-center justify-center gap-6 rounded-lg px-12 py-8 text-center text-12 font-semibold transition-all active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-50 sm:flex-initial';
const BTN_CONFIRMAR = 'flex min-h-44 flex-1 cursor-pointer items-center justify-center gap-6 rounded-lg bg-plain-green px-16 py-8 text-center text-12 font-bold text-canvas-white transition-all hover:bg-plain-green-muted active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-50 sm:flex-initial';

function PedidoAccionCard({
  pedido,
  ahora,
  ocupado,
  onConfirmar,
  onEntregar,
  onVerMesa,
}: {
  pedido: PedidoVista;
  ahora: number;
  ocupado: boolean;
  onConfirmar: (id: string) => void;
  onEntregar: (id: string, modo: ModoEntregaPedido) => void;
  onVerMesa: () => void;
}) {
  const porConfirmar = pedido.estado === 'recibido';
  const minutos = minutosDesde(pedido.creadoEn, ahora);
  const unidades = pedido.items.reduce((sum, item) => sum + item.cantidad, 0);

  return (
    <article
      className={`overflow-hidden rounded-xl border border-l-[3px] bg-canvas-white shadow-[0_1px_2px_rgba(15,23,42,0.04)] ${
        porConfirmar ? 'border-[#F3D4A3] border-l-[#F2A51A]' : 'border-[#A7E4CC] border-l-[#14A77B]'
      }`}
    >
      <div className="px-16 py-12">
        <div className="flex items-start justify-between gap-12">
          <h3 className="text-16 font-bold tracking-[-0.01em] text-ash-graphite">Mesa {pedido.mesa}</h3>
          <span className={`shrink-0 rounded-full border px-8 py-4 text-10 font-semibold ${ESTADO_STYLES[pedido.estado]}`}>
            {ESTADO_LABELS[pedido.estado]}
          </span>
        </div>
        <p className="mt-4 text-11 text-sage-green">
          {porConfirmar && minutos !== null && (
            <span className={`font-semibold ${minutos >= 5 ? 'text-alert-red' : 'text-[#9A5700]'}`}>{textoAntiguedad(minutos)} · </span>
          )}
          {pedido.timestamp} · {unidades} {unidades === 1 ? 'producto' : 'productos'}
        </p>
        <ul className="mt-8 space-y-2 text-12 text-ash-graphite">
          {pedido.items.slice(0, 4).map(item => (
            <li key={item.id}>{item.cantidad}× {item.nombre}</li>
          ))}
          {pedido.items.length > 4 && <li className="text-sage-green">y {pedido.items.length - 4} más…</li>}
        </ul>
      </div>
      <div className="flex flex-wrap gap-8 border-t border-ghost-fog px-10 py-8">
        {porConfirmar ? (
          <button type="button" onClick={() => onConfirmar(pedido.id)} disabled={ocupado} className={BTN_CONFIRMAR}>
            <span className="material-symbols-outlined text-17" aria-hidden="true">check_circle</span>
            <span>{ocupado ? 'Confirmando...' : 'Confirmar'}</span>
          </button>
        ) : (
          <>
            <button
              type="button"
              onClick={() => onEntregar(pedido.id, 'mesa')}
              disabled={ocupado}
              className={`${BTN_ENTREGA_BASE} bg-emerald-600 text-canvas-white hover:bg-emerald-500`}
            >
              <span>Entregado en mesa</span>
            </button>
            <button
              type="button"
              onClick={() => onEntregar(pedido.id, 'retiro')}
              disabled={ocupado}
              className={`${BTN_ENTREGA_BASE} border border-concrete bg-canvas-white text-ash-graphite hover:bg-ghost-fog`}
            >
              <span>Retirado por el cliente</span>
            </button>
          </>
        )}
        <button
          type="button"
          onClick={onVerMesa}
          className="flex min-h-44 w-full cursor-pointer items-center justify-center gap-6 rounded-lg px-10 text-12 font-semibold text-sage-green hover:text-ash-graphite"
        >
          Ver detalle de la mesa
        </button>
      </div>
    </article>
  );
}

function PedidoDetalle({
  pedido,
  onAvanzar,
  onConfirmar,
  onEntregar,
  ocupado,
  onImprimirPedido,
  etiquetaPorComensal,
}: {
  pedido: PedidoVista;
  onAvanzar: (id: string) => void;
  onConfirmar: (id: string) => void;
  onEntregar: (id: string, modo: ModoEntregaPedido) => void;
  ocupado: boolean;
  onImprimirPedido: (pedido: PedidoVista) => void;
  etiquetaPorComensal: Map<string, string>;
}) {
  const total = pedido.items.reduce((sum, i) => sum + i.precio * i.cantidad, 0);

  return (
    <div className="border-t border-ghost-fog px-16 py-12 sm:px-20">
      <div className="flex flex-wrap items-center justify-between gap-8">
        <div className="flex items-center gap-8">
          <span className="text-13 font-semibold text-ash-graphite">Pedido</span>
          <span className="text-11 font-mono text-sage-green">{pedido.timestamp}</span>
        </div>
        <span className={`shrink-0 rounded-full border px-10 py-3 text-11 font-semibold ${ESTADO_STYLES[pedido.estado]}`}>
          {ESTADO_LABELS[pedido.estado]}
        </span>
      </div>

      <div className="mt-12 space-y-8">
        {pedido.items.map(item => (
          <div key={item.id} className="flex items-start justify-between gap-12 text-13">
            <div className="min-w-0">
              <p className="text-ash-graphite font-medium">{item.cantidad}× {item.nombre}</p>
              {item.variantes && item.variantes.length > 0 && (
                <p className="mt-2 text-11 text-plain-green-muted font-medium">
                  {item.variantes.map(v => v.nombre).join(", ")}
                </p>
              )}
              <span className="mt-3 inline-block rounded-full bg-vanilla-cream px-7 py-2 text-10 font-medium text-plain-green">
                {etiquetaPorComensal.get(item.comensalId?.trim() || 'mesa-sin-identificar') || item.comensalNombre || 'Mesa'}
              </span>
              {item.nota && <p className="mt-2 text-11 text-sage-green">Nota: {item.nota}</p>}
            </div>
            <span className="shrink-0 font-mono text-sage-green">${(item.precio * item.cantidad).toLocaleString()}</span>
          </div>
        ))}
      </div>

      <div className="mt-12 flex items-center justify-between border-t border-ghost-fog pt-8 text-13 font-medium">
        <span className="font-semibold text-ash-graphite">Total del pedido</span>
        <span className="font-mono font-semibold text-ash-graphite">${total.toLocaleString()}</span>
      </div>

      <div className="mt-12 flex flex-wrap items-center gap-8">
        <button
          type="button"
          onClick={() => onImprimirPedido(pedido)}
          className="flex min-h-44 cursor-pointer items-center justify-center gap-6 rounded-lg border border-concrete bg-canvas-white px-12 py-8 text-12 font-semibold text-ash-graphite transition-all hover:border-stone hover:bg-ghost-fog active:scale-[0.98]"
          title="Imprimir ticket térmico de este pedido (58/80 mm)"
        >
          <span className="material-symbols-outlined text-16">print</span>
          <span>Imprimir ticket</span>
        </button>
        {pedido.estado === 'recibido' && (
          <button type="button" onClick={() => onConfirmar(pedido.id)} disabled={ocupado} className={BTN_CONFIRMAR}>
            <span className="material-symbols-outlined text-17" aria-hidden="true">check_circle</span>
            <span>{ocupado ? 'Confirmando...' : 'Confirmar'}</span>
          </button>
        )}
        {(pedido.estado === 'confirmado' || pedido.estado === 'preparando') && (
          <button
            onClick={() => onAvanzar(pedido.id)}
            disabled={ocupado}
            className="flex min-h-44 flex-1 items-center justify-center rounded-lg bg-plain-green px-16 py-8 text-center text-12 font-semibold text-canvas-white transition-all hover:bg-plain-green-muted active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-50 sm:flex-initial"
          >
            {pedido.estado === 'confirmado' ? '→ Preparando' : '→ Listo'}
          </button>
        )}
        {pedido.estado === 'listo' && (
          <>
            <button
              type="button"
              onClick={() => onEntregar(pedido.id, 'mesa')}
              disabled={ocupado}
              className={`${BTN_ENTREGA_BASE} bg-emerald-600 text-canvas-white hover:bg-emerald-500`}
            >
              <span>Entregado en mesa</span>
            </button>
            <button
              type="button"
              onClick={() => onEntregar(pedido.id, 'retiro')}
              disabled={ocupado}
              className={`${BTN_ENTREGA_BASE} border border-concrete bg-canvas-white text-ash-graphite hover:bg-ghost-fog`}
            >
              <span>Retirado por el cliente</span>
            </button>
          </>
        )}
        {pedido.estado === 'entregado' && (
          <span className="flex min-h-44 items-center rounded-lg border border-success/30 bg-success/15 px-12 py-8 text-12 font-semibold text-[#087645]">
            Pedido entregado
          </span>
        )}
      </div>
    </div>
  );
}

function MesaCard({
  grupo,
  onOpen,
  onHabilitarPago,
}: {
  grupo: MesaGrupo;
  onOpen: () => void;
  onHabilitarPago?: (grupo: MesaGrupo) => void;
}) {
  const total = grupo.pedidos.reduce(
    (sum, pedido) => sum + pedido.items.reduce((pedidoTotal, item) => pedidoTotal + item.precio * item.cantidad, 0),
    0,
  );
  const totalItems = grupo.pedidos.reduce(
    (sum, pedido) => sum + pedido.items.reduce((pedidoItems, item) => pedidoItems + item.cantidad, 0),
    0,
  );
  const cuentaSolicitada = grupo.cuentaSolicitada;
  const tienePedidosListos = grupo.pedidos.some(p => p.estado === 'listo');
  const tone: MesaCardTone = cuentaSolicitada ? 'account' : tienePedidosListos ? 'ready' : 'progress';
  const toneStyles = MESA_CARD_TONES[tone];
  const statusLabel = cuentaSolicitada
    ? grupo.pagoHabilitado ? 'Pago habilitado' : 'Cuenta'
    : tienePedidosListos ? 'Listo' : 'En curso';
  const ultimaHora = grupo.pedidos[grupo.pedidos.length - 1]?.timestamp;
  const comensales = new Set(
    grupo.pedidos.flatMap(pedido => pedido.items.map(item => item.comensalId).filter(Boolean)),
  ).size;

  return (
    <article
      className={`overflow-hidden rounded-xl border border-l-[3px] bg-canvas-white shadow-[0_1px_2px_rgba(15,23,42,0.04)] transition-all hover:-translate-y-0.5 hover:shadow-md ${toneStyles.border} ${toneStyles.accent}`}
    >
      <div className="px-16 py-12">
        <div className="flex items-start justify-between gap-12">
          <div className="min-w-0">
            <h3 className="text-16 font-bold tracking-[-0.01em] text-ash-graphite">Mesa {grupo.mesa}</h3>
          </div>
          <span className={`inline-flex shrink-0 items-center gap-6 rounded-full border px-8 py-4 text-10 font-semibold ${toneStyles.badge}`}>
            <span className={`h-6 w-6 rounded-full ${toneStyles.dot}`} aria-hidden="true" />
            {statusLabel}
          </span>
        </div>

        <div className="mt-8 flex flex-wrap items-center justify-between gap-8 text-11 text-sage-green">
          {comensales > 0 && (
            <span className="flex items-center gap-4">
              <span className="material-symbols-outlined text-15" aria-hidden="true">group</span>
              {comensales} {comensales === 1 ? 'comensal' : 'comensales'}
            </span>
          )}
          <span className="rounded-full bg-ghost-fog px-8 py-4 font-medium text-ash-graphite">
            {totalItems} {totalItems === 1 ? 'producto' : 'productos'}
          </span>
        </div>

        <div className="mt-10 flex items-center justify-between gap-12 border-t border-ghost-fog pt-8">
          <span className="flex min-w-0 items-center gap-6 text-10 text-sage-green">
            <span className="material-symbols-outlined text-15" aria-hidden="true">schedule</span>
            {ultimaHora ? `Último pedido ${ultimaHora}` : 'Pedido activo'}
          </span>
          <span className="shrink-0 font-mono text-14 font-semibold text-ash-graphite">${total.toLocaleString()}</span>
        </div>
      </div>

      <div className={`flex gap-8 border-t px-10 py-6 ${toneStyles.action}`}>
          {cuentaSolicitada && !grupo.pagoHabilitado && onHabilitarPago && (
            <button
              type="button"
              onClick={() => onHabilitarPago(grupo)}
              className="flex min-h-44 flex-1 items-center justify-center gap-6 rounded-lg bg-[#168AC1] px-10 py-8 text-12 font-semibold text-white shadow-2xs transition-all hover:bg-[#0E78AA] active:scale-[0.98]"
            >
              <span className="material-symbols-outlined text-17" aria-hidden="true">payments</span>
              <span>Habilitar pago</span>
            </button>
          )}
          <button
            type="button"
            onClick={onOpen}
            className={`flex min-h-44 ${cuentaSolicitada && !grupo.pagoHabilitado && onHabilitarPago ? 'flex-1' : 'w-full'} items-center justify-center gap-6 rounded-lg border border-current/15 bg-white/70 px-10 text-12 font-semibold transition-all hover:bg-white active:scale-[0.98]`}
          >
            <span>{tienePedidosListos ? 'Ver pedido' : 'Ver detalle'}</span>
            <span className="material-symbols-outlined text-16" aria-hidden="true">arrow_forward</span>
          </button>
      </div>
    </article>
  );
}

function PedidosSectionHeader({
  tone,
  icon,
  title,
  subtitle,
  count,
}: {
  tone: MesaCardTone;
  icon: string;
  title: string;
  subtitle: string;
  count: number;
}) {
  const styles = {
    account: {
      icon: 'bg-[#FFF1D8] text-[#B86B00]',
      badge: 'bg-[#FFF1D8] text-[#9A5700]',
    },
    ready: {
      icon: 'bg-[#E4F7F0] text-[#087657]',
      badge: 'bg-[#E4F7F0] text-[#087657]',
    },
    progress: {
      icon: 'bg-[#EAF3FF] text-[#2D6FB7]',
      badge: 'bg-[#EAF3FF] text-[#285F9F]',
    },
  }[tone];

  return (
    <div className="flex items-start gap-10">
      <span className={`flex h-32 w-32 shrink-0 items-center justify-center rounded-lg ${styles.icon}`}>
        <span className="material-symbols-outlined text-18" aria-hidden="true">{icon}</span>
      </span>
      <div className="min-w-0">
        <div className="flex flex-wrap items-center gap-8">
          <h3 className="text-14 font-semibold text-ash-graphite">{title}</h3>
          <span className={`rounded-full px-8 py-2 text-10 font-semibold ${styles.badge}`}>{count}</span>
        </div>
        <p className="mt-2 text-10 text-sage-green">{subtitle}</p>
      </div>
    </div>
  );
}

function MesaDetalleModal({
  grupo,
  onClose,
  onAvanzar,
  onConfirmar,
  onEntregar,
  accionesEnCurso,
  onTodoListo,
  onCerrarCuenta,
  onHabilitarPago,
  onImprimirMesa,
  onImprimirPedido,
  mesaAccionEnCurso,
}: {
  grupo: MesaGrupo;
  onClose: () => void;
  onAvanzar: (id: string) => void;
  onConfirmar: (id: string) => void;
  onEntregar: (id: string, modo: ModoEntregaPedido) => void;
  accionesEnCurso: ReadonlySet<string>;
  onTodoListo: (grupo: MesaGrupo) => void;
  onCerrarCuenta: (grupo: MesaGrupo) => void;
  onHabilitarPago: (grupo: MesaGrupo, habilitar?: boolean) => void;
  onImprimirMesa: (grupo: MesaGrupo) => void;
  onImprimirPedido: (pedido: PedidoVista) => void;
  mesaAccionEnCurso: 'todo-listo' | 'cerrar-cuenta' | 'habilitar-pago' | null;
}) {
  const total = grupo.pedidos.reduce(
    (sum, pedido) => sum + pedido.items.reduce((pedidoTotal, item) => pedidoTotal + item.precio * item.cantidad, 0),
    0,
  );
  const totalItems = grupo.pedidos.reduce(
    (sum, pedido) => sum + pedido.items.reduce((pedidoItems, item) => pedidoItems + item.cantidad, 0),
    0,
  );
  const estadoMesa = calcularEstadoMesa(grupo.pedidos);
  // "Todo listo" solo aplica a pedidos que cocina ya tiene (confirmados o en preparación).
  const hayPendientesDeCocina = grupo.pedidos.some(p => p.estado === 'confirmado' || p.estado === 'preparando');
  const gruposComensales = useMemo(
    () => agruparPorComensal(grupo.pedidos.flatMap(pedido => pedido.items)),
    [grupo.pedidos],
  );
  const etiquetaPorComensal = useMemo(
    () => new Map(gruposComensales.map(comensal => [comensal.id, comensal.etiqueta])),
    [gruposComensales],
  );

  useEffect(() => {
    const previousOverflow = document.body.style.overflow;
    const closeWithEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') onClose();
    };

    document.body.style.overflow = 'hidden';
    window.addEventListener('keydown', closeWithEscape);
    return () => {
      document.body.style.overflow = previousOverflow;
      window.removeEventListener('keydown', closeWithEscape);
    };
  }, [onClose]);

  return (
    <div
      className="fixed inset-0 z-50 flex items-end justify-center bg-black/75 p-12 sm:items-center sm:p-24"
      role="presentation"
      onMouseDown={event => {
        if (event.target === event.currentTarget) onClose();
      }}
    >
      <section
        className="flex max-h-[calc(100vh-24px)] w-full max-w-3xl flex-col overflow-hidden rounded-xl border border-concrete bg-canvas-white shadow-2xl sm:max-h-[calc(100vh-48px)]"
        role="dialog"
        aria-modal="true"
        aria-labelledby={`mesa-detalle-${grupo.key}`}
      >
        <header className="flex shrink-0 items-center justify-between gap-12 border-b border-concrete bg-canvas-white px-16 py-14 sm:px-20">
          <div className="min-w-0">
            <div className="flex flex-wrap items-center gap-8 sm:gap-12">
              <h2 id={`mesa-detalle-${grupo.key}`} className="text-18 font-semibold text-ash-graphite">
                Mesa {grupo.mesa}
              </h2>
              <span className={`shrink-0 rounded-full border px-10 py-3 text-11 font-semibold ${ESTADO_STYLES[estadoMesa]}`}>
                {ESTADO_LABELS[estadoMesa]}
              </span>
            </div>
            <p className="mt-2 text-11 font-mono text-sage-green">
              {grupo.pedidos.length} {grupo.pedidos.length === 1 ? 'pedido activo' : 'pedidos activos'} · {totalItems} {totalItems === 1 ? 'ítem' : 'ítems'}
            </p>
          </div>
          <button
            type="button"
            onClick={onClose}
            aria-label={`Cerrar detalle de Mesa ${grupo.mesa}`}
            className="flex h-44 w-44 shrink-0 items-center justify-center rounded-lg border border-concrete text-20 leading-none text-ash-graphite transition-colors hover:border-stone hover:bg-ghost-fog"
          >
            ×
          </button>
        </header>

        <div className="flex flex-wrap gap-8 border-b border-ghost-fog bg-canvas-white px-16 py-12 sm:px-20">
          <button
            type="button"
            onClick={() => onImprimirMesa(grupo)}
            className="flex min-h-44 cursor-pointer items-center justify-center gap-6 rounded-lg border border-concrete bg-canvas-white px-12 py-8 text-12 font-semibold text-ash-graphite transition-colors hover:border-stone hover:bg-ghost-fog"
            title="Imprimir ticket térmico completo de la mesa (58/80 mm)"
          >
            <span className="material-symbols-outlined text-16">print</span>
            <span>Imprimir ticket mesa</span>
          </button>
          <button
            type="button"
            onClick={() => onTodoListo(grupo)}
            disabled={!hayPendientesDeCocina || mesaAccionEnCurso !== null}
            className="min-h-44 flex-1 rounded-lg bg-success px-12 py-8 text-12 font-semibold text-ash-graphite transition-opacity hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-50 sm:flex-initial cursor-pointer"
          >
            {mesaAccionEnCurso === 'todo-listo' ? 'Actualizando...' : 'Todo listo'}
          </button>
          {grupo.cuentaSolicitada && !grupo.pagoHabilitado && (
            <button
              type="button"
              onClick={() => onHabilitarPago(grupo)}
              disabled={mesaAccionEnCurso !== null}
              className="min-h-44 flex-1 rounded-lg bg-[#009EE3] px-14 py-8 text-12 font-semibold text-white transition-opacity hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-50 sm:flex-initial flex items-center justify-center gap-6 shadow-xs cursor-pointer"
            >
              <span className="text-14">💳</span>
              <span>{mesaAccionEnCurso === 'habilitar-pago' ? 'Habilitando...' : 'Habilitar pago MP'}</span>
            </button>
          )}
          {grupo.cuentaSolicitada && grupo.pagoHabilitado && (
            <div className="flex flex-wrap items-center gap-8">
              <span className="flex min-h-44 items-center justify-center gap-6 rounded-lg border border-plain-green/30 bg-plain-green/10 px-12 py-8 text-12 font-semibold text-plain-green">
                <span>✓ Pago habilitado</span>
              </span>
              <button
                type="button"
                onClick={() => onHabilitarPago(grupo, false)}
                disabled={mesaAccionEnCurso !== null}
                className="min-h-44 rounded-lg border border-concrete px-12 py-8 text-12 font-medium text-sage-green hover:text-alert-red hover:border-alert-red/40 transition-colors cursor-pointer"
                title="Deshabilitar pago digital para esta mesa"
              >
                Revocar
              </button>
            </div>
          )}
          <button
            type="button"
            onClick={() => onCerrarCuenta(grupo)}
            disabled={mesaAccionEnCurso !== null}
            className="min-h-44 flex-1 rounded-lg bg-alert-red px-12 py-8 text-12 font-semibold text-canvas-white transition-opacity hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-50 sm:flex-initial cursor-pointer"
          >
            {mesaAccionEnCurso === 'cerrar-cuenta' ? 'Cerrando...' : 'Cerrar cuenta'}
          </button>
          {grupo.cuentaSolicitada && !grupo.pagoHabilitado && (
            <span className="flex min-h-44 items-center justify-center rounded-lg border border-alert-red/30 bg-warm-pink/15 px-12 py-8 text-12 font-semibold text-alert-red sm:ml-auto">
              Cuenta solicitada
            </span>
          )}
        </div>

        <div className="min-h-0 flex-1 overflow-y-auto">
          <div className="flex items-center justify-between gap-12 border-b border-ghost-fog px-16 py-14 sm:px-20">
            <span className="text-13 font-medium text-ash-graphite">Total acumulado de la mesa</span>
            <span className="shrink-0 font-mono text-16 font-semibold text-ash-graphite">${total.toLocaleString()}</span>
          </div>
          {gruposComensales.length > 0 && (
            <div className="border-b border-ghost-fog px-16 py-14 sm:px-20">
              <h3 className="text-13 font-semibold text-ash-graphite">Cuenta por comensal</h3>
              <div className="mt-10 grid gap-8 sm:grid-cols-2">
                {gruposComensales.map(comensal => (
                  <div key={comensal.id} className="rounded-lg border border-concrete bg-ghost-fog/70 px-12 py-10">
                    <div className="flex items-center justify-between gap-8">
                      <span className="text-12 font-semibold text-ash-graphite">{comensal.etiqueta}</span>
                      <span className="shrink-0 font-mono text-13 font-semibold text-plain-green">
                        ${comensal.subtotal.toLocaleString()}
                      </span>
                    </div>
                    <p className="mt-2 text-10 text-sage-green">
                      {comensal.items.reduce((cantidad, item) => cantidad + item.cantidad, 0)} ítems
                    </p>
                  </div>
                ))}
              </div>
            </div>
          )}
          {grupo.pedidos.map(pedido => (
            <PedidoDetalle
              key={pedido.id}
              pedido={pedido}
              onAvanzar={onAvanzar}
              onConfirmar={onConfirmar}
              onEntregar={onEntregar}
              ocupado={accionesEnCurso.has(pedido.id)}
              onImprimirPedido={onImprimirPedido}
              etiquetaPorComensal={etiquetaPorComensal}
            />
          ))}
        </div>
      </section>
    </div>
  );
}

export default function RecepcionistaSection() {
  const toast = useToast();
  const confirmar = useConfirm();
  const [pedidos, setPedidos] = useState<PedidoVista[]>([]);
  const [sucursal, setSucursal] = useState<Sucursal | null>(null);
	const [sucursales, setSucursales] = useState<Sucursal[]>([]);
  const [mesas, setMesas] = useState<Record<string, MesaAPI>>({});
  const [loading, setLoading] = useState(true);
  const [sseConectado, setSseConectado] = useState(false);
  const [mesaAccionEnCurso, setMesaAccionEnCurso] = useState<'todo-listo' | 'cerrar-cuenta' | 'habilitar-pago' | null>(null);
  const [accionesEnCurso, setAccionesEnCurso] = useState<ReadonlySet<string>>(new Set());
  const [ahora, setAhora] = useState(() => Date.now());
  const [mesaDetalleKey, setMesaDetalleKey] = useState<string | null>(null);
  const [pedidoAsistidoAbierto, setPedidoAsistidoAbierto] = useState(false);
  const [anchoTicket, setAnchoTicket] = useState<AnchoTicketTermico>("80mm");
  const [autoPrint, setAutoPrint] = useState<boolean>(false);
  const [ticketActual, setTicketActual] = useState<DatosTicketTermico | null>(null);
  const [modalTicketAbierto, setModalTicketAbierto] = useState<boolean>(false);
  const [errorImpresionTicket, setErrorImpresionTicket] = useState<string | null>(null);
  const eventSourceRef = useRef<EventSource | null>(null);
  // Guarda síncrona contra doble click; el estado React solo deshabilita los botones.
  const accionesEnCursoRef = useRef<Set<string>>(new Set());

  // Refresca la antigüedad de los pedidos por confirmar.
  useEffect(() => {
    const id = window.setInterval(() => setAhora(Date.now()), 30000);
    return () => window.clearInterval(id);
  }, []);

  useEffect(() => {
    setAnchoTicket(getAnchoTicketTermico());
    setAutoPrint(isAutoPrintEnabled());
  }, []);

  const transformarPedidoApi = useCallback((p: PedidoAPI, mesasMap: Record<string, MesaAPI>): PedidoVista => {
    const numMesa = mesasMap[p.mesa_id]?.numero || 1;
    return {
      id: p.id,
      mesa: numMesa,
      mesaId: p.mesa_id,
      sucursalId: p.sucursal_id,
      timestamp: p.created_at ? new Date(p.created_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) : new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
      creadoEn: p.created_at || undefined,
      ...estadoVista(p.estado),
      items: (p.items || []).map(i => ({
        id: i.id || i.articulo_id,
        nombre: i.nombre_articulo?.trim() || 'Producto sin nombre',
        cantidad: i.cantidad,
        precio: i.precio_unitario,
        nota: i.notas?.trim() || '',
        comensalId: i.comensal_id,
        comensalNombre: i.comensal_nombre?.trim() || undefined,
        variantes: i.variantes,
      })),
      cuentaSolicitada: Boolean(mesasMap[p.mesa_id]?.cuenta_solicitada),
    };
  }, []);

  const cargarDatos = useCallback(async () => {
    try {
      setLoading(true);
      const [sucursalesList, mesasList] = await Promise.all([
        api.listarSucursales(),
        api.listarMesas(),
      ]);

      const sMap: Record<string, MesaAPI> = {};
      if (mesasList) {
        mesasList.forEach(m => { sMap[m.id] = m; });
        setMesas(sMap);
      }

      if (sucursalesList && sucursalesList.length > 0) {
			setSucursales(sucursalesList);
        const primSuc = sucursalesList[0];
        setSucursal(primSuc);

        try {
          const pedidosApi = await api.listarPedidosActivos(primSuc.id);
          if (pedidosApi && pedidosApi.length >= 0) {
            const formateados = pedidosApi.map(p => transformarPedidoApi(p, sMap));
            setPedidos(formateados);
            return;
          }
        } catch (err) {
          console.error("Error al obtener pedidos activos:", err);
        }
      }

      setPedidos([]);
    } catch (err) {
      console.error("Error cargando pedidos para recepcionista:", err);
      setPedidos([]);
    } finally {
      setLoading(false);
    }
  }, [transformarPedidoApi]);

	useEffect(() => {
		if (!sucursal || loading) return;
		void api.listarPedidosActivos(sucursal.id).then((lista) => setPedidos(lista.map((p) => transformarPedidoApi(p, mesas)))).catch(() => setPedidos([]));
	}, [sucursal?.id, loading, mesas, transformarPedidoApi]);

  useEffect(() => {
    const timeoutId = window.setTimeout(() => {
      void cargarDatos();
    }, 0);

    return () => window.clearTimeout(timeoutId);
  }, [cargarDatos]);

  useEffect(() => {
    if (!sucursal) return;

    const sseUrl = api.obtenerEventosSucursalUrl(sucursal.id);
		const es = new EventSource(sseUrl, { withCredentials: true });
    eventSourceRef.current = es;

    es.onopen = () => {
      setSseConectado(true);
    };

    es.addEventListener('pedido_creado', (e: MessageEvent) => {
      try {
        const data: PedidoAPI = JSON.parse(e.data);
        if (data && data.id) {
          const nuevo = transformarPedidoApi(data, mesas);
          setPedidos(prev => {
            const existente = prev.find(p => p.id === data.id);
            if (existente && !avanza(estadoApiDeVista(existente), data.estado)) return prev;
            return [nuevo, ...prev.filter(p => p.id !== data.id)];
          });
        }
      } catch (err) {
        console.warn("Error parseando pedido_creado SSE:", err);
      }
    });

    es.addEventListener('pedido_actualizado', (e: MessageEvent) => {
      try {
        const data: PedidoAPI = JSON.parse(e.data);
        if (data && data.id) {
          setPedidos(prev => {
            const pedidoAnterior = prev.find(p => p.id === data.id);
            if (pedidoAnterior && !avanza(estadoApiDeVista(pedidoAnterior), data.estado)) return prev;
            const eraListo = pedidoAnterior?.estado === 'listo';
            const esAhoraListo = data.estado === 'listo';

            // Alerta sonora y toast cuando un pedido pasa a estar listo para retirar de cocina
            if (esAhoraListo && !eraListo) {
              const numMesa = mesas[data.mesa_id]?.numero || pedidoAnterior?.mesa || 'asignada';
              toast.success(`🛎️ ¡Mesa ${numMesa}: Pedido listo para retirar de cocina!`);
              playOrderReadySound();
            }

            const actualizado = transformarPedidoApi(data, mesas);
            const existe = prev.some(p => p.id === data.id);
            if (existe) {
              return prev.map(p => (p.id === data.id ? actualizado : p));
            }
            return [actualizado, ...prev];
          });
        }
      } catch (err) {
        console.warn("Error parseando pedido_actualizado SSE:", err);
      }
    });

    es.addEventListener('cuenta_solicitada', (e: MessageEvent) => {
      try {
        const data: MesaAPI = JSON.parse(e.data);
        if (!data?.id) return;
        setMesas(prev => ({
          ...prev,
          [data.id]: { ...prev[data.id], ...data, cuenta_solicitada: true },
        }));
        setPedidos(prev => prev.map(pedido =>
          pedido.mesaId === data.id ? { ...pedido, cuentaSolicitada: true } : pedido,
        ));
      } catch (err) {
        console.warn("Error parseando solicitud de cuenta SSE:", err);
      }
    });

    es.addEventListener('cuenta_cerrada', (e: MessageEvent) => {
      try {
        const data: MesaAPI = JSON.parse(e.data);
        if (!data?.id) return;
        setMesas(prev => ({
          ...prev,
          [data.id]: { ...prev[data.id], ...data, cuenta_solicitada: false },
        }));
        setPedidos(prev => prev.filter(pedido => pedido.mesaId !== data.id));
        setMesaDetalleKey(current => current === data.id ? null : current);
      } catch (err) {
        console.warn("Error parseando cierre de cuenta SSE:", err);
      }
    });

    es.addEventListener('pago_habilitado', (e: MessageEvent) => {
      try {
        const data: MesaAPI = JSON.parse(e.data);
        if (!data?.id) return;
        setMesas(prev => ({
          ...prev,
          [data.id]: { ...prev[data.id], ...data, pago_habilitado: true },
        }));
      } catch (err) {
        console.warn("Error parseando pago_habilitado SSE:", err);
      }
    });

    es.addEventListener('pago_deshabilitado', (e: MessageEvent) => {
      try {
        const data: MesaAPI = JSON.parse(e.data);
        if (!data?.id) return;
        setMesas(prev => ({
          ...prev,
          [data.id]: { ...prev[data.id], ...data, pago_habilitado: false },
        }));
      } catch (err) {
        console.warn("Error parseando pago_deshabilitado SSE:", err);
      }
    });

    es.addEventListener('pago_recibido', (e: MessageEvent) => {
      try {
        const data = JSON.parse(e.data);
        toast.success(`¡Pago recibido de Mesa ${data.numero}! Monto: $${data.monto}`);
        if (data.mesa_id) {
          setMesas(prev => ({
            ...prev,
            [data.mesa_id]: { ...prev[data.mesa_id], cuenta_solicitada: false, pago_habilitado: false },
          }));
        }
      } catch (err) {
        console.warn("Error parseando pago_recibido SSE:", err);
      }
    });

    es.onerror = () => {
      setSseConectado(false);
    };

    return () => {
      es.close();
      eventSourceRef.current = null;
    };
  }, [sucursal, mesas, transformarPedidoApi]);

  const avanzarEstado = async (pedidoId: string) => {
    const target = pedidos.find(p => p.id === pedidoId);
    if (!target) return;

    // Un pedido "recibido" se confirma (no salta a preparando): la API respondería PEDIDO_SIN_CONFIRMAR.
    if (target.estado !== 'confirmado' && target.estado !== 'preparando') return;
    const nextEstado: 'preparando' | 'listo' = target.estado === 'confirmado' ? 'preparando' : 'listo';

    await ejecutarAccionPedido(
      pedidoId,
      () => api.cambiarEstadoPedido(pedidoId, nextEstado),
      'Estado actualizado.',
    );
  };

  const ejecutarAccionPedido = async (
    pedidoId: string,
    accion: () => Promise<PedidoAPI>,
    mensajeExito: string,
  ) => {
    if (accionesEnCursoRef.current.has(pedidoId)) return;
    accionesEnCursoRef.current.add(pedidoId);
    setAccionesEnCurso(prev => new Set(prev).add(pedidoId));
    try {
      const actualizado = await accion();
      setPedidos(prev => prev.map(p => (
        p.id === pedidoId && avanza(estadoApiDeVista(p), actualizado.estado) ? { ...p, ...estadoVista(actualizado.estado) } : p
      )));
      toast.success(mensajeExito);
    } catch (err) {
      console.error("Error al actualizar el pedido:", err);
      toast.error(mensajeErrorPedido(err));
    } finally {
      accionesEnCursoRef.current.delete(pedidoId);
      setAccionesEnCurso(prev => {
        const siguiente = new Set(prev);
        siguiente.delete(pedidoId);
        return siguiente;
      });
    }
  };

  const confirmarPedido = (pedidoId: string) =>
    ejecutarAccionPedido(pedidoId, () => api.confirmarPedido(pedidoId), 'Pedido confirmado. Ya está en la cola de cocina.');

  const entregarPedido = (pedidoId: string, modo: ModoEntregaPedido) =>
    ejecutarAccionPedido(
      pedidoId,
      () => api.entregarPedido(pedidoId, modo),
      modo === 'mesa' ? 'Pedido marcado como entregado en mesa.' : 'Pedido marcado como retirado por el cliente.',
    );

  const marcarMesaTodoLista = async (grupo: MesaGrupo) => {
    const pendientes = grupo.pedidos.filter(pedido => pedido.estado === 'confirmado' || pedido.estado === 'preparando');
    if (pendientes.length === 0) return;
    const ok = await confirmar({
      titulo: 'Marcar todos como listos',
      mensaje: `¿Marcás como listos los ${pendientes.length} pedidos pendientes de la Mesa ${grupo.mesa}?`,
      labelAceptar: 'Sí, marcar listos',
      variante: 'warning',
    });
    if (!ok) return;

    setMesaAccionEnCurso('todo-listo');
    try {
      const resultados = await Promise.allSettled(
        pendientes.map(pedido => api.cambiarEstadoPedido(pedido.id, 'listo')),
      );
      const pedidosActualizados = new Set(
        pendientes.filter((_, index) => resultados[index].status === 'fulfilled').map(pedido => pedido.id),
      );
      setPedidos(prev => prev.map(pedido =>
        pedidosActualizados.has(pedido.id) && avanza(estadoApiDeVista(pedido), 'listo') ? { ...pedido, estado: 'listo' } : pedido,
      ));

      const rechazado = resultados.find((resultado): resultado is PromiseRejectedResult => resultado.status === 'rejected');
      if (rechazado) {
        toast.error(mensajeErrorPedido(rechazado.reason));
      } else {
        toast.success('Todos los pedidos marcados como listos.');
      }
    } catch (err) {
      console.error("Error al marcar la mesa como lista:", err);
      toast.error('No se pudo actualizar el estado de la mesa.');
    } finally {
      setMesaAccionEnCurso(null);
    }
  };

  const cerrarCuenta = async (grupo: MesaGrupo) => {
    if (!grupo.mesaId) {
      toast.error('No se pudo identificar la mesa para cerrarla.');
      return;
    }
    const ok = await confirmar({
      titulo: 'Cerrar cuenta',
      mensaje: `¿Cerrás la cuenta de la Mesa ${grupo.mesa}? Se finalizarán los pedidos actuales y la mesa quedará disponible para una nueva cuenta.`,
      labelAceptar: 'Sí, cerrar cuenta',
      variante: 'danger',
    });
    if (!ok) return;

    setMesaAccionEnCurso('cerrar-cuenta');
    try {
      const mesaActualizada = await api.cerrarCuenta(grupo.mesaId);
      setPedidos(prev => prev.filter(pedido => pedido.mesaId !== grupo.mesaId));
      setMesas(prev => ({
        ...prev,
        [grupo.mesaId as string]: {
          ...prev[grupo.mesaId as string],
          ...mesaActualizada,
          cuenta_solicitada: false,
        },
      }));
      setMesaDetalleKey(null);
      toast.success('Cuenta cerrada correctamente.');
    } catch (err) {
      console.error("Error al cerrar la cuenta:", err);
      toast.error('No se pudo cerrar la cuenta. Intenta nuevamente.');
    } finally {
      setMesaAccionEnCurso(null);
    }
  };

  const habilitarPago = async (grupo: MesaGrupo, habilitar = true) => {
    if (!grupo.mesaId) {
      toast.error('No se pudo identificar la mesa para modificar el pago.');
      return;
    }
    setMesaAccionEnCurso('habilitar-pago');
    try {
      const mesaActualizada = await api.habilitarPagoMesa(grupo.mesaId, habilitar);
      setMesas(prev => ({
        ...prev,
        [grupo.mesaId as string]: {
          ...prev[grupo.mesaId as string],
          ...mesaActualizada,
          pago_habilitado: habilitar,
        },
      }));
      if (habilitar) {
        toast.success(`¡Pago habilitado para Mesa ${grupo.mesa}! El comensal ya puede abonar con Mercado Pago desde su celular.`);
      } else {
        toast.info(`Pago digital deshabilitado para Mesa ${grupo.mesa}.`);
      }
    } catch (err) {
      console.error("Error al modificar estado del pago:", err);
      toast.error('No se pudo actualizar el pago. Intenta nuevamente.');
    } finally {
      setMesaAccionEnCurso(null);
    }
  };

  const handleCambiarAnchoTicket = useCallback((nuevoAncho: AnchoTicketTermico) => {
    setAnchoTicket(nuevoAncho);
    setAnchoTicketTermico(nuevoAncho);
  }, []);

  const handleCambiarAutoPrint = useCallback((activo: boolean) => {
    setAutoPrint(activo);
    setAutoPrintEnabled(activo);
  }, []);

  const handleImprimirPedido = useCallback(
    (pedido: PedidoVista) => {
      const datos: DatosTicketTermico = {
        id: pedido.id,
        numeroPedido: `#${pedido.id.slice(0, 6).toUpperCase()}`,
        mesa: pedido.mesa,
        sucursalNombre: sucursal?.nombre,
        fechaHora: `${new Date().toLocaleDateString("es-AR")} ${pedido.timestamp}`,
        items: pedido.items.map((it) => ({
          id: it.id,
          nombre: it.nombre,
          cantidad: it.cantidad,
          precioUnitario: it.precio,
          notas: it.nota || undefined,
          comensalNombre: it.comensalNombre,
          variantes: it.variantes,
        })),
        mostrarPrecios: true,
        esReimpresion: false,
      };
      setTicketActual(datos);
      setErrorImpresionTicket(null);
      setModalTicketAbierto(true);
    },
    [sucursal]
  );

  const handleImprimirMesa = useCallback(
    (grupo: MesaGrupo) => {
      const datos: DatosTicketTermico = {
        id: grupo.key,
        numeroPedido:
          grupo.pedidos.length > 1
            ? `${grupo.pedidos.length} PEDIDOS`
            : `#${(grupo.pedidos[0]?.id || "000000").slice(0, 6).toUpperCase()}`,
        mesa: grupo.mesa,
        sucursalNombre: sucursal?.nombre,
        fechaHora: new Date().toLocaleString("es-AR", {
          day: "2-digit",
          month: "2-digit",
          year: "2-digit",
          hour: "2-digit",
          minute: "2-digit",
        }),
        items: grupo.pedidos.flatMap((p) =>
          p.items.map((it) => ({
            id: `${p.id}-${it.id}`,
            nombre: it.nombre,
            cantidad: it.cantidad,
            precioUnitario: it.precio,
            notas: it.nota || undefined,
            comensalNombre: it.comensalNombre,
            variantes: it.variantes,
          }))
        ),
        mostrarPrecios: true,
        esReimpresion: false,
      };
      setTicketActual(datos);
      setErrorImpresionTicket(null);
      setModalTicketAbierto(true);
    },
    [sucursal]
  );

  const handleEjecutarImpresion = useCallback(() => {
    if (!ticketActual) return;
    const ok = dispararImpresionNavegador();
    if (ok) {
      setErrorImpresionTicket(null);
    } else {
      setErrorImpresionTicket(
        "No se pudo enviar el ticket a la impresora térmica. Verificá la conexión USB/Red y reintentá manualmente."
      );
    }
  }, [ticketActual]);

  const cerrarMesaDetalle = useCallback(() => setMesaDetalleKey(null), []);
  const mesasAgrupadas = useMemo(() => {
    const grupos = new Map<string, MesaGrupo>();

    pedidos.forEach(pedido => {
      const key = pedido.mesaId || `mesa-${pedido.mesa}`;
      const grupo = grupos.get(key);
      if (grupo) {
        grupo.pedidos.push(pedido);
        return;
      }

      grupos.set(key, {
        key,
        mesa: pedido.mesa,
        mesaId: pedido.mesaId,
        pedidos: [pedido],
        cuentaSolicitada: Boolean(pedido.mesaId && mesas[pedido.mesaId]?.cuenta_solicitada) || Boolean(pedido.cuentaSolicitada),
        pagoHabilitado: Boolean(pedido.mesaId && mesas[pedido.mesaId]?.pago_habilitado),
      });
    });

    return Array.from(grupos.values());
  }, [mesas, pedidos]);
  const mesaDetalle = mesasAgrupadas.find(grupo => grupo.key === mesaDetalleKey);
  const conAlerta = mesasAgrupadas.filter(grupo => grupo.cuentaSolicitada);
  const porConfirmar = useMemo(
    () => pedidos
      .filter(p => p.estado === 'recibido')
      .sort((a, b) => (Date.parse(a.creadoEn ?? '') || 0) - (Date.parse(b.creadoEn ?? '') || 0)),
    [pedidos],
  );
  const listosParaRetirar = pedidos.filter(p => p.estado === 'listo');
  const enCurso = mesasAgrupadas.filter(
    grupo => !grupo.cuentaSolicitada && grupo.pedidos.some(p => p.estado === 'confirmado' || p.estado === 'preparando')
  );
  const abrirMesaDePedido = (pedido: PedidoVista) => setMesaDetalleKey(pedido.mesaId || `mesa-${pedido.mesa}`);

  return (
    <div className="h-full space-y-28 overflow-y-auto bg-[#F7F8F8] p-16 font-inter sm:p-24 md:p-32">
      <div className="flex flex-col justify-between gap-16 sm:flex-row sm:items-center">
        <div className="min-w-0">
          <h2 className="text-24 font-semibold tracking-[-0.02em] text-ash-graphite sm:text-32">Panel Recepcionista</h2>
          <p className="mt-4 text-13 text-sage-green sm:text-14">
            {loading 
              ? 'Cargando pedidos...' 
              : `${mesasAgrupadas.length} ${mesasAgrupadas.length === 1 ? 'mesa activa' : 'mesas activas'} · ${pedidos.length} ${pedidos.length === 1 ? 'pedido' : 'pedidos'}`
            }
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-8 self-start sm:self-auto">
			{sucursales.length > 1 && <select aria-label="Sucursal" value={sucursal?.id ?? ''} onChange={(e) => setSucursal(sucursales.find((s) => s.id === e.target.value) ?? null)} className="h-44 rounded-lg border border-concrete bg-canvas-white px-10 text-12 font-semibold">{sucursales.map((s) => <option key={s.id} value={s.id}>{s.nombre}</option>)}</select>}
			<button onClick={() => setPedidoAsistidoAbierto(true)} className="flex h-44 items-center gap-6 rounded-lg bg-plain-green px-14 text-12 font-semibold text-white shadow-sm hover:bg-plain-green-muted"><span className="material-symbols-outlined text-18">add</span>Cargar pedido</button>
          <a
            href="/kds"
            target="_blank"
            rel="noopener noreferrer"
            className="flex h-44 items-center gap-6 rounded-full border border-amber-500/40 bg-amber-500/10 px-14 text-12 font-bold text-amber-800 hover:bg-amber-500/20 transition-all shadow-xs cursor-pointer"
            title="Abrir pantalla de cocina (KDS) en una pestaña nueva"
          >
            <span>🍳</span>
            <span>KDS Cocina ↗</span>
          </a>
          <div className="flex h-44 items-center gap-8 rounded-full border border-concrete bg-canvas-white px-14 shadow-sm">
            <span
              className={`h-8 w-8 rounded-full ${sseConectado ? 'bg-success animate-pulse' : 'bg-sage-green'}`}
            />
            <span className="text-11 font-mono text-sage-green uppercase tracking-wide">
              {sseConectado ? 'En vivo (SSE)' : 'Conectando...'}
            </span>
          </div>
        </div>
      </div>

      {loading && (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-16">
          {[1, 2, 3].map(i => (
            <div key={i} className="overflow-hidden rounded-xl border border-concrete bg-canvas-white shadow-sm">
              <Skeleton className="h-52 w-full rounded-none" />
              <div className="p-16 space-y-8">
                <Skeleton className="h-12 w-3/4" />
                <Skeleton className="h-12 w-1/2" />
                <Skeleton className="h-12 w-2/3" />
              </div>
            </div>
          ))}
        </div>
      )}

      {/* 0. POR CONFIRMAR: pedidos del QR que esperan a recepción */}
      {!loading && porConfirmar.length > 0 && (
        <section className="space-y-12" aria-labelledby="por-confirmar-title">
          <div id="por-confirmar-title">
            <PedidosSectionHeader
              tone="account"
              icon="notification_important"
              title="Por confirmar"
              subtitle="Cocina no los ve hasta que los confirmes"
              count={porConfirmar.length}
            />
          </div>
          <div className="grid grid-cols-1 gap-14 md:grid-cols-2 lg:grid-cols-3">
            {porConfirmar.map(pedido => (
              <PedidoAccionCard
                key={pedido.id}
                pedido={pedido}
                ahora={ahora}
                ocupado={accionesEnCurso.has(pedido.id)}
                onConfirmar={confirmarPedido}
                onEntregar={entregarPedido}
                onVerMesa={() => abrirMesaDePedido(pedido)}
              />
            ))}
          </div>
        </section>
      )}

      {/* 1. SOLICITUDES DE CUENTA */}
      {!loading && conAlerta.length > 0 && (
        <section className="space-y-12" aria-labelledby="solicitudes-cuenta-title">
          <div id="solicitudes-cuenta-title">
            <PedidosSectionHeader
              tone="account"
              icon="receipt_long"
              title="Solicitudes de cuenta"
              subtitle="Atender primero"
              count={conAlerta.length}
            />
          </div>
          <div className="grid grid-cols-1 gap-14 md:grid-cols-2 lg:grid-cols-3">
            {conAlerta.map(grupo => (
              <MesaCard
                key={grupo.key}
                grupo={grupo}
                onOpen={() => setMesaDetalleKey(grupo.key)}
                onHabilitarPago={habilitarPago}
              />
            ))}
          </div>
        </section>
      )}

      {/* 2. ALERTA DESTACADA: LISTOS PARA RETIRAR DE COCINA */}
      {!loading && listosParaRetirar.length > 0 && (
        <section className="space-y-12" aria-labelledby="listos-retirar-title">
          <div id="listos-retirar-title">
            <PedidosSectionHeader
              tone="ready"
              icon="notifications"
              title="Listos para retirar"
              subtitle="Marcalos cuando se entreguen en mesa o los retire el cliente"
              count={listosParaRetirar.length}
            />
          </div>
          <div className="grid grid-cols-1 gap-14 md:grid-cols-2 lg:grid-cols-3">
            {listosParaRetirar.map(pedido => (
              <PedidoAccionCard
                key={pedido.id}
                pedido={pedido}
                ahora={ahora}
                ocupado={accionesEnCurso.has(pedido.id)}
                onConfirmar={confirmarPedido}
                onEntregar={entregarPedido}
                onVerMesa={() => abrirMesaDePedido(pedido)}
              />
            ))}
          </div>
        </section>
      )}

      {/* 3. PEDIDOS EN CURSO / PREPARACIÓN */}
      {!loading && enCurso.length > 0 && (
        <section className="space-y-12" aria-labelledby="pedidos-preparacion-title">
          <div id="pedidos-preparacion-title">
            <PedidosSectionHeader
              tone="progress"
              icon="schedule"
              title="En preparación"
              subtitle="Pedidos activos en cocina"
              count={enCurso.length}
            />
          </div>
          <div className="grid grid-cols-1 gap-14 md:grid-cols-2 lg:grid-cols-3">
            {enCurso.map(grupo => (
              <MesaCard
                key={grupo.key}
                grupo={grupo}
                onOpen={() => setMesaDetalleKey(grupo.key)}
                onHabilitarPago={habilitarPago}
              />
            ))}
          </div>
        </section>
      )}

      {!loading && pedidos.length === 0 && (
        <EmptyState
          icon={sseConectado ? "receipt_long" : "wifi_off"}
          title="Sin pedidos activos"
          description={
            sseConectado
              ? "Cuando un cliente haga un pedido o pida la cuenta, aparecerá aquí en tiempo real."
              : "Conectando al canal en vivo..."
          }
        />
      )}

      {mesaDetalle && (
        <MesaDetalleModal
          grupo={mesaDetalle}
          onClose={cerrarMesaDetalle}
          onAvanzar={avanzarEstado}
          onConfirmar={confirmarPedido}
          onEntregar={entregarPedido}
          accionesEnCurso={accionesEnCurso}
          onTodoListo={marcarMesaTodoLista}
          onCerrarCuenta={cerrarCuenta}
          onHabilitarPago={habilitarPago}
          onImprimirMesa={handleImprimirMesa}
          onImprimirPedido={handleImprimirPedido}
          mesaAccionEnCurso={mesaAccionEnCurso}
        />
      )}

      <PedidoAsistidoModal
        abierto={pedidoAsistidoAbierto}
        mesas={Object.values(mesas).filter((m) => m.estado === 'activa')}
        onClose={() => setPedidoAsistidoAbierto(false)}
        onCreado={() => void cargarDatos()}
      />

      <TicketTermicoModal
        isOpen={modalTicketAbierto}
        datos={ticketActual}
        ancho={anchoTicket}
        onCambiarAncho={handleCambiarAnchoTicket}
        autoPrint={autoPrint}
        onCambiarAutoPrint={handleCambiarAutoPrint}
        errorImpresion={errorImpresionTicket}
        onImprimir={handleEjecutarImpresion}
        onClose={() => setModalTicketAbierto(false)}
      />
      <TicketTermicoPrintPortal datos={ticketActual} ancho={anchoTicket} />
    </div>
  );
}
