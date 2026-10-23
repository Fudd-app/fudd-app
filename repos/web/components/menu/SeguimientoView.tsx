import { useMemo, useState } from "react";
import type { CartItem, EstadoPedido } from "@/app/mesa/[token]/page";
import { agruparPorComensal } from "@/lib/desgloseCuenta";
import BrandHeader from "./BrandHeader";
import type { MesaBranding } from "./BrandHeader";
import MarcaAgua from "./MarcaAgua";

interface Props {
  branding: MesaBranding;
  items: CartItem[];
  estadoPedido: EstadoPedido;
  todosListos: boolean;
  cuentaSolicitada: boolean;
  pagoHabilitado?: boolean;
  mesa: number;
  comensalId?: string;
  comensalNombre?: string;
  mostrarMarcaAgua?: boolean;
  onCambiarComensal?: () => void;
  onAgregarMas: () => void;
  onPedirCuenta: () => Promise<void> | void;
  onPagarMercadoPago?: () => Promise<void> | void;
  pagandoMP?: boolean;
  pagoExitoso?: boolean;
  pagoError?: boolean;
  yaCalificado?: boolean;
  onCalificar?: () => void;
}

const PASOS: EstadoPedido[] = ['recibido', 'confirmado', 'preparando', 'listo', 'entregado'];

const PASO_LABELS: Record<EstadoPedido, string> = {
  recibido: 'Pedido enviado',
  confirmado: 'Confirmado',
  preparando: 'En preparación',
  listo: '¡Listo!',
  entregado: 'Entregado',
  cerrado: 'Pedido finalizado',
};

const DEMORA_LABELS: Record<EstadoPedido, string> = {
  recibido: 'Esperando confirmación del salón',
  confirmado: 'Estimado: ~15 min',
  preparando: 'Estimado: ~8 min',
  listo: '¡Tu pedido está listo!',
  entregado: '¡Buen provecho!',
  cerrado: 'Mesa cerrada',
};

export default function SeguimientoView({
  branding,
  items,
  estadoPedido,
  todosListos,
  cuentaSolicitada,
  pagoHabilitado = false,
  mesa,
  comensalId,
  comensalNombre,
  mostrarMarcaAgua = true,
  onCambiarComensal,
  onAgregarMas,
  onPedirCuenta,
  onPagarMercadoPago,
  pagandoMP = false,
  pagoExitoso = false,
  pagoError = false,
  yaCalificado = false,
  onCalificar,
}: Props) {
  const [mostrarConfirmacion, setMostrarConfirmacion] = useState(false);
  const [solicitandoCuenta, setSolicitandoCuenta] = useState(false);
  const [modoCuenta, setModoCuenta] = useState<'mesa' | 'comensales'>('mesa');
  const total = items.reduce((n, i) => n + i.precio * i.cantidad, 0);
  // "cerrado" se muestra como el último paso completado.
  const pasoActual = estadoPedido === 'cerrado' ? PASOS.length - 1 : PASOS.indexOf(estadoPedido);
  const gruposComensales = useMemo(() => {
    const nombresActuales = new Map<string, string>();
    if (comensalId?.trim() && comensalNombre?.trim()) {
      nombresActuales.set(comensalId.trim(), comensalNombre.trim());
    }
    return agruparPorComensal(items, nombresActuales);
  }, [comensalId, comensalNombre, items]);
  const aliasesRegistrados = gruposComensales.filter(grupo => grupo.tieneAlias);
  const todosConAlias = gruposComensales.length > 0 && aliasesRegistrados.length === gruposComensales.length;

  const confirmarSolicitudCuenta = async () => {
    setSolicitandoCuenta(true);
    try {
      await onPedirCuenta();
      setMostrarConfirmacion(false);
    } catch {
      // El flujo que invoca la API muestra el error al comensal.
    } finally {
      setSolicitandoCuenta(false);
    }
  };

  return (
    <div className="mesa-background min-h-screen pb-32 font-inter">
      <BrandHeader
        branding={branding}
        mesa={mesa}
        title="Estado de la mesa"
        comensalNombre={comensalNombre}
        onCambiarComensal={onCambiarComensal}
      />

      <div className="max-w-lg mx-auto px-16 py-20 space-y-16">
        {/* Stepper Card */}
        <div className="mesa-surface mesa-border space-y-16 rounded-lg border p-20 shadow-2xs">
          <h2 className="mesa-text text-14 font-medium">Progreso del pedido</h2>
          <div className="space-y-16">
            {PASOS.map((paso, i) => {
              const activo = i === pasoActual;
              const completado = i < pasoActual;
              return (
                <div key={paso} className="flex items-start gap-14">
                  <div
                    className={`w-28 h-28 rounded-full flex items-center justify-center flex-shrink-0 text-13 font-semibold mt-1 transition-all ${
                    completado
                        ? 'bg-emerald-500 text-white shadow-xs'
                        : activo
                        ? 'mesa-primary-bg shadow-md ring-4 ring-[var(--mesa-primary-soft)]'
                        : 'mesa-subtle-surface mesa-subtle-text'
                    }`}
                  >
                    {completado ? '✓' : i + 1}
                  </div>
                  <div className="flex-1 min-w-0">
                    <p
                      className={`text-14 font-medium ${
                        activo
                          ? 'mesa-text font-semibold'
                          : completado
                          ? 'text-emerald-700'
                          : 'mesa-subtle-text'
                      }`}
                    >
                      {PASO_LABELS[paso]}
                    </p>
                    {activo && (
                      <p className="mesa-primary mesa-primary-soft mt-2 inline-block rounded px-8 py-2 text-12 font-medium">
                        {DEMORA_LABELS[paso]}
                      </p>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        </div>

        {todosListos && (
          <div className="mesa-primary-soft mesa-border rounded-lg border px-16 py-14 text-center shadow-xs space-y-10">
            <div>
              <p className="mesa-primary text-14 font-semibold">✓ No hay pedidos pendientes</p>
              <p className="mesa-muted mt-2 text-12">Todos los pedidos de esta mesa están listos.</p>
            </div>
            {!yaCalificado && onCalificar && (
              <div className="pt-8 border-t mesa-border flex justify-center">
                <button
                  type="button"
                  onClick={onCalificar}
                  className="mesa-surface mesa-text mesa-border border hover:border-[var(--mesa-primary)] inline-flex items-center justify-center gap-8 rounded-lg px-16 py-8 text-13 font-semibold shadow-2xs transition-all active:scale-98"
                >
                  <span className="material-symbols-outlined text-18 text-amber-500 fill-1">star</span>
                  <span>⭐ Calificar tu experiencia</span>
                </button>
              </div>
            )}
          </div>
        )}

        {/* Resumen del pedido */}
        <div className="mesa-surface mesa-border space-y-10 rounded-lg border p-20 shadow-2xs">
          <div className="flex items-center justify-between gap-12">
            <h3 className="mesa-text text-13 font-medium">Resumen de la cuenta</h3>
            <span className="mesa-muted text-11">{gruposComensales.length} {gruposComensales.length === 1 ? 'comensal' : 'comensales'}</span>
          </div>

          <div className="mesa-subtle-surface grid grid-cols-2 rounded-lg p-2" role="tablist" aria-label="Modo de cuenta">
            <button
              type="button"
              role="tab"
              aria-selected={modoCuenta === 'mesa'}
              onClick={() => setModoCuenta('mesa')}
              className={`min-h-44 rounded-md px-8 text-12 font-medium transition-colors ${modoCuenta === 'mesa' ? 'mesa-surface mesa-text shadow-xs' : 'mesa-muted'}`}
            >
              Cuenta de la mesa
            </button>
            <button
              type="button"
              role="tab"
              aria-selected={modoCuenta === 'comensales'}
              onClick={() => setModoCuenta('comensales')}
              className={`min-h-44 rounded-md px-8 text-12 font-medium transition-colors ${modoCuenta === 'comensales' ? 'mesa-surface mesa-text shadow-xs' : 'mesa-muted'}`}
            >
              Por persona
            </button>
          </div>

          {modoCuenta === 'comensales' && !todosConAlias && (
            <div className="mesa-primary-soft mesa-border rounded-lg border p-12" role="status">
              <p className="mesa-text text-12 font-semibold">No todos los comensales ingresaron un alias.</p>
              <p className="mesa-muted mt-3 text-11">La cuenta por persona se habilitará cuando todos estén identificados.</p>
              <div className="mt-8 flex flex-wrap items-center gap-6">
                <span className="mesa-muted text-10 font-medium">Alias registrados:</span>
                {aliasesRegistrados.length > 0 ? aliasesRegistrados.map(grupo => (
                  <span key={grupo.id} className="mesa-surface mesa-border rounded-full border px-8 py-3 text-10 font-medium">
                    {grupo.etiqueta}
                  </span>
                )) : (
                  <span className="mesa-muted text-10">Ninguno todavía</span>
                )}
              </div>
            </div>
          )}

          {modoCuenta === 'mesa' ? (
            <div className="divide-y divide-[var(--mesa-border)]">
              {items.map((item, index) => (
                <div key={`${item.id}-${index}`} className="mesa-muted flex items-start justify-between gap-10 py-8 text-13">
                  <div className="min-w-0">
                    <span>{item.cantidad}× {item.nombre}</span>
                    {item.variantes && item.variantes.length > 0 && (
                      <p className="mt-2 text-11 text-slate-500">
                        {item.variantes.map(v => v.nombre).join(", ")}
                      </p>
                    )}
                    {item.nota && (
                      <p className="mt-1 text-11 italic text-slate-400">
                        Nota: {item.nota}
                      </p>
                    )}
                  </div>
                  <span className="mesa-text shrink-0 font-mono font-medium">${(item.precio * item.cantidad).toLocaleString()}</span>
                </div>
              ))}
            </div>
          ) : todosConAlias ? (
            <div className="space-y-10">
              {gruposComensales.map(grupo => (
                <section key={grupo.id} className="mesa-subtle-surface mesa-border rounded-lg border p-12">
                  <div className="flex items-center justify-between gap-10">
                    <span className="mesa-text text-13 font-semibold">{grupo.etiqueta}</span>
                    <span className="mesa-primary shrink-0 font-mono text-14 font-semibold">${grupo.subtotal.toLocaleString()}</span>
                  </div>
                  <div className="mesa-muted mt-8 space-y-4 text-12">
                    {grupo.items.map((item, index) => (
                      <div key={`${item.id}-${index}`} className="flex justify-between gap-8">
                        <div className="min-w-0">
                          <span>{item.cantidad}× {item.nombre}</span>
                          {item.variantes && item.variantes.length > 0 && (
                            <p className="text-10 text-slate-500">
                              {item.variantes.map(v => v.nombre).join(", ")}
                            </p>
                          )}
                          {item.nota && (
                            <p className="text-10 italic text-slate-400">
                              Nota: {item.nota}
                            </p>
                          )}
                        </div>
                        <span className="shrink-0 font-mono">${(item.precio * item.cantidad).toLocaleString()}</span>
                      </div>
                    ))}
                  </div>
                </section>
              ))}
            </div>
          ) : null}
          <div className="mesa-text flex justify-between border-t pt-10 text-14 font-semibold mesa-border">
            <span>Total</span>
            <span className="mesa-primary font-mono font-semibold">${total.toLocaleString()}</span>
          </div>
        </div>

        {/* Acciones */}
        <div className="space-y-10 pt-4">
          {pagoExitoso && (
            <div className="w-full rounded-lg border border-emerald-500/30 bg-emerald-500/10 p-16 text-center shadow-xs animate-in fade-in">
              <p className="text-14 font-semibold text-emerald-600 dark:text-emerald-400">✓ ¡Pago acreditado por Mercado Pago!</p>
              <p className="mesa-muted mt-2 text-12">El pago fue registrado con éxito y el personal del local fue notificado en tiempo real.</p>
            </div>
          )}

          {pagoError && (
            <div className="w-full rounded-lg border border-rose-500/30 bg-rose-500/10 p-16 text-center shadow-xs animate-in fade-in">
              <p className="text-14 font-semibold text-rose-600 dark:text-rose-400">No se pudo procesar el pago con Mercado Pago</p>
              <p className="mesa-muted mt-2 text-12">Podés reintentar abonar con Mercado Pago o solicitar la cuenta para pagar al mozo.</p>
            </div>
          )}

          {items.length > 0 && !pagoExitoso && (
            <div className="mesa-primary-soft mesa-border w-full rounded-lg border py-16 text-center shadow-xs">
              <p className="mesa-primary text-14 font-semibold">✓ Pedido realizado con éxito</p>
              <p className="mesa-muted mt-2 text-12">Nuestro equipo ya está trabajando en tu pedido.</p>
            </div>
          )}

          {!cuentaSolicitada && !pagoExitoso && (
            <button
              onClick={onAgregarMas}
              className="mesa-surface mesa-muted mesa-border flex min-h-52 w-full items-center justify-center gap-6 rounded-lg border py-12 text-13 font-medium shadow-2xs transition-all hover:border-[var(--mesa-primary)] active:scale-[0.98]"
            >
              <span className="text-16 font-semibold">+</span>
              <span>Agregar más ítems</span>
            </button>
          )}

          {!cuentaSolicitada && !pagoExitoso ? (
            <button
              onClick={() => setMostrarConfirmacion(true)}
              className="mesa-primary-bg flex min-h-52 w-full items-center justify-center gap-6 rounded-lg py-14 text-14 font-semibold shadow-md transition-all active:scale-[0.98] cursor-pointer"
            >
              <span>Pedir la cuenta</span>
              <span className="text-16">🧾</span>
            </button>
          ) : !pagoExitoso ? (
            <div className="space-y-12">
              {!pagoHabilitado ? (
                <div className="mesa-primary-soft mesa-border w-full rounded-lg border py-16 px-16 text-center shadow-xs animate-in fade-in space-y-4">
                  <p className="mesa-primary text-14 font-semibold">✓ Solicitud de cuenta enviada</p>
                  <p className="mesa-muted text-12">
                    El personal del local está revisando tu mesa. En instantes habilitarán el cobro digital o podés pagarle en persona al mozo.
                  </p>
                  <div className="flex items-center justify-center gap-6 pt-4 text-11 font-medium text-amber-600">
                    <span className="material-symbols-outlined text-16 animate-pulse">hourglass_top</span>
                    <span>Aguardando habilitación de recepción...</span>
                  </div>
                </div>
              ) : (
                <>
                  <div className="w-full rounded-lg border border-plain-green/30 bg-plain-green/10 py-14 px-16 text-center shadow-xs animate-in fade-in space-y-2">
                    <p className="text-14 font-semibold text-plain-green flex items-center justify-center gap-6">
                      <span className="material-symbols-outlined text-18">check_circle</span>
                      <span>¡Pago habilitado por recepción!</span>
                    </p>
                    <p className="mesa-muted text-12">
                      Ya podés abonar el total de tu cuenta de forma digital con Mercado Pago.
                    </p>
                  </div>

                  {onPagarMercadoPago && items.length > 0 && (
                    <button
                      type="button"
                      onClick={() => void onPagarMercadoPago()}
                      disabled={pagandoMP}
                      className="flex min-h-52 w-full items-center justify-center gap-8 rounded-lg bg-[#009EE3] hover:bg-[#0086c2] text-white py-14 px-16 text-14 font-semibold shadow-md transition-all active:scale-[0.98] disabled:opacity-60 cursor-pointer"
                    >
                      {pagandoMP ? (
                        <>
                          <span className="material-symbols-outlined animate-spin text-18">progress_activity</span>
                          <span>Conectando con Mercado Pago...</span>
                        </>
                      ) : (
                        <>
                          <span className="text-16">💳</span>
                          <span>Pagar con Mercado Pago</span>
                        </>
                      )}
                    </button>
                  )}
                </>
              )}
            </div>
          ) : null}

          {mostrarMarcaAgua && <MarcaAgua />}
        </div>
      </div>

      {mostrarConfirmacion && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/75 p-16"
          role="presentation"
          onMouseDown={event => {
            if (event.target === event.currentTarget && !solicitandoCuenta) {
              setMostrarConfirmacion(false);
            }
          }}
        >
          <section
            className="mesa-surface mesa-border w-full max-w-sm rounded-xl border p-20 shadow-2xl"
            role="dialog"
            aria-modal="true"
            aria-labelledby="confirmar-cuenta-titulo"
          >
            <h2 id="confirmar-cuenta-titulo" className="mesa-text text-16 font-semibold">
              ¿Solicitar la cuenta?
            </h2>
            <p className="mesa-muted mt-8 text-13 leading-relaxed">
              Una vez solicitada, no vas a poder agregar más ítems a esta cuenta. El mozo confirmará el cierre cuando se realice el pago.
            </p>
            <div className="mt-20 flex flex-col-reverse gap-8 sm:flex-row sm:justify-end">
              <button
                type="button"
                onClick={() => setMostrarConfirmacion(false)}
                disabled={solicitandoCuenta}
                className="mesa-surface mesa-muted mesa-border min-h-44 rounded-md border px-14 py-8 text-13 font-medium disabled:opacity-50"
              >
                Cancelar
              </button>
              <button
                type="button"
                onClick={() => void confirmarSolicitudCuenta()}
                disabled={solicitandoCuenta}
                className="mesa-primary-bg min-h-44 rounded-md px-14 py-8 text-13 font-semibold disabled:opacity-50"
              >
                {solicitandoCuenta ? 'Enviando...' : 'Sí, solicitar cuenta'}
              </button>
            </div>
          </section>
        </div>
      )}
    </div>
  );
}
