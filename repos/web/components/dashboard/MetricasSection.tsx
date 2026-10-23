"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import {
  api,
  getErrorMessage,
  MetricasResumenAPI,
  Sucursal,
} from "@/lib/api";
import { Skeleton } from "@/components/ui";

const FORMATO_MONEDA = new Intl.NumberFormat("es-AR", {
  style: "currency",
  currency: "ARS",
  maximumFractionDigits: 0,
});

const ESTADO_LABELS: Record<string, string> = {
  recibido: "Recibidos",
  confirmado: "Confirmados",
  preparando: "En preparación",
  listo: "Listos",
  entregado: "Entregados",
  cerrado: "Cerrados",
};

const ESTADO_STYLES: Record<string, string> = {
  recibido: "border-[#F3D4A3] bg-[#FFF9EF] text-[#8A4D00]",
  confirmado: "border-[#C9DCF7] bg-[#EFF6FF] text-[#285F9F]",
  entregado: "border-[#A7E4CC] bg-[#EAF8F2] text-[#087657]",
  preparando: "border-[#C9DCF7] bg-[#EFF6FF] text-[#285F9F]",
  listo: "border-[#A7E4CC] bg-[#EAF8F2] text-[#087657]",
  cerrado: "border-concrete bg-ghost-fog text-sage-green",
};

function fechaActualArgentina() {
  const partes = new Intl.DateTimeFormat("en-CA", {
    timeZone: "America/Argentina/Buenos_Aires",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).formatToParts(new Date());
  const valor = Object.fromEntries(partes.map((parte) => [parte.type, parte.value]));
  return `${valor.year}-${valor.month}-${valor.day}`;
}

function formatearMoneda(valor: number) {
  return FORMATO_MONEDA.format(Number.isFinite(valor) ? valor : 0).replace(/\s/g, " ");
}

function formatearTiempo(valor: number) {
  if (!Number.isFinite(valor) || valor <= 0) return "0 min";
  return `${Math.round(valor)} min`;
}

function obtenerMaximo<T>(items: T[], valor: (item: T) => number) {
  return items.reduce<T | null>((maximo, item) => {
    if (maximo === null || valor(item) > valor(maximo)) return item;
    return maximo;
  }, null);
}

function consultarMetricas(sucursalId: string) {
  const hoy = fechaActualArgentina();
  return api.obtenerMetricasResumen({
    desde: hoy,
    hasta: hoy,
    sucursalId: sucursalId || undefined,
  });
}

function VariacionBadge({
  valor,
  invertir = false,
}: {
  valor: number | null;
  invertir?: boolean;
}) {
  if (valor === null) {
    return <span className="text-11 font-medium text-stone">Sin datos previos</span>;
  }

  const sinCambio = valor === 0;
  const favorable = sinCambio || (invertir ? valor < 0 : valor > 0);
  const clase = sinCambio
    ? "bg-ghost-fog text-sage-green"
    : favorable
      ? "bg-[#EAF8F2] text-[#087657]"
      : "bg-[#FEF3F2] text-[#B42318]";
  const icono = sinCambio ? "remove" : valor > 0 ? "north_east" : "south_east";

  return (
    <span className={`inline-flex items-center gap-4 rounded-full px-8 py-3 text-10 font-semibold ${clase}`}>
      <span className="material-symbols-outlined text-13">{icono}</span>
      {Math.abs(valor).toLocaleString("es-AR", { maximumFractionDigits: 1 })}% vs. ayer
    </span>
  );
}

function KPICard({
  icono,
  etiqueta,
  valor,
  detalle,
  variacion,
  invertirVariacion = false,
}: {
  icono: string;
  etiqueta: string;
  valor: string;
  detalle: string;
  variacion: number | null;
  invertirVariacion?: boolean;
}) {
  return (
    <article className="rounded-xl border border-concrete bg-canvas-white p-16 sm:p-20">
      <div className="flex items-start justify-between gap-12">
        <div className="grid h-40 w-40 shrink-0 place-items-center rounded-lg bg-ghost-fog text-ash-graphite">
          <span className="material-symbols-outlined text-20">{icono}</span>
        </div>
        <VariacionBadge valor={variacion} invertir={invertirVariacion} />
      </div>
      <p className="mt-16 text-11 font-mono font-semibold uppercase tracking-[0.08em] text-sage-green">
        {etiqueta}
      </p>
      <p className="mt-6 text-24 font-semibold tracking-[-0.03em] text-ash-graphite sm:text-32">{valor}</p>
      <p className="mt-4 text-11 text-sage-green">{detalle}</p>
    </article>
  );
}

function MetricasSkeleton() {
  return (
    <div className="space-y-16" aria-label="Calculando métricas">
      <div className="grid gap-12 sm:grid-cols-2 xl:grid-cols-4">
        {Array.from({ length: 4 }, (_, index) => (
          <div key={index} className="rounded-xl border border-concrete bg-canvas-white p-20">
            <div className="flex justify-between">
              <Skeleton className="h-40 w-40" />
              <Skeleton className="h-24 w-80" />
            </div>
            <Skeleton className="mt-20 h-12 w-120" />
            <Skeleton className="mt-10 h-32 w-160" />
            <Skeleton className="mt-8 h-12 w-120" />
          </div>
        ))}
      </div>
      <div className="grid gap-16 xl:grid-cols-[minmax(0,1.7fr)_minmax(280px,0.8fr)]">
        <Skeleton className="h-[320px] w-full rounded-xl" />
        <Skeleton className="h-[320px] w-full rounded-xl" />
      </div>
    </div>
  );
}

export default function MetricasSection() {
  const [resumen, setResumen] = useState<MetricasResumenAPI | null>(null);
  const [sucursales, setSucursales] = useState<Sucursal[]>([]);
  const [sucursalId, setSucursalId] = useState("");
  const [cargando, setCargando] = useState(true);
  const [error, setError] = useState("");
  const [actualizadoA, setActualizadoA] = useState<Date | null>(null);

  useEffect(() => {
    let activo = true;
    api.listarSucursales()
      .then((items) => {
        if (activo) setSucursales(items);
      })
      .catch(() => {
        if (activo) setSucursales([]);
      });
    return () => {
      activo = false;
    };
  }, []);

  useEffect(() => {
    let activo = true;
    consultarMetricas("")
      .then((data) => {
        if (!activo) return;
        setResumen(data);
        setActualizadoA(new Date());
      })
      .catch((err: unknown) => {
        if (activo) setError(getErrorMessage(err, "No pudimos calcular las métricas del negocio."));
      })
      .finally(() => {
        if (activo) setCargando(false);
      });
    return () => {
      activo = false;
    };
  }, []);

  const cargarMetricas = useCallback(async (sucursalSeleccionada = sucursalId) => {
    setCargando(true);
    setError("");
    try {
      const data = await consultarMetricas(sucursalSeleccionada);
      setResumen(data);
      setActualizadoA(new Date());
    } catch (err) {
      setError(getErrorMessage(err, "No pudimos calcular las métricas del negocio."));
    } finally {
      setCargando(false);
    }
  }, [sucursalId]);

  const maximoUnidades = useMemo(
    () => Math.max(...(resumen?.platos_estrella.map((plato) => plato.unidades) ?? [0]), 1),
    [resumen],
  );
  const resumenHorarios = useMemo(() => {
    const turnos = (resumen?.por_turno ?? []).map((turno) => ({
      ...turno,
      ticketPromedio: turno.pedidos_cerrados > 0
        ? turno.facturacion_total / turno.pedidos_cerrados
        : 0,
    }));

    return {
      turnos,
      picoPedidos: obtenerMaximo(turnos, (turno) => turno.pedidos_totales),
      picoFacturacion: obtenerMaximo(turnos, (turno) => turno.facturacion_total),
      picoTicket: obtenerMaximo(turnos, (turno) => turno.ticketPromedio),
      maximoPedidos: Math.max(...turnos.map((turno) => turno.pedidos_totales), 1),
    };
  }, [resumen]);

  return (
    <div className="h-full overflow-y-auto bg-[#F4F6F7] p-16 font-inter sm:p-24 md:p-32">
      <div className="mx-auto w-full max-w-[1440px] space-y-20">
        <header className="flex flex-col gap-12 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <p className="text-10 font-mono font-semibold uppercase tracking-[0.12em] text-sage-green">Rendimiento de hoy</p>
            <h2 className="mt-4 text-24 font-semibold tracking-[-0.03em] text-ash-graphite sm:text-32">Métricas</h2>
            <p className="mt-4 text-13 text-sage-green">
              Una lectura rápida de ventas, pedidos y tiempos de atención.
            </p>
          </div>
          <div className="flex flex-col gap-8 sm:flex-row sm:items-center">
            <label className="sr-only" htmlFor="filtro-sucursal-metricas">Sucursal</label>
            <select
              id="filtro-sucursal-metricas"
              value={sucursalId}
              onChange={(event) => {
                const siguienteSucursal = event.target.value;
                setSucursalId(siguienteSucursal);
                void cargarMetricas(siguienteSucursal);
              }}
              className="h-44 min-w-[220px] rounded-lg border border-concrete bg-canvas-white px-12 text-12 text-ash-graphite"
            >
              <option value="">Todo el negocio</option>
              {sucursales.map((sucursal) => (
                <option key={sucursal.id} value={sucursal.id}>{sucursal.nombre}</option>
              ))}
            </select>
            <button
              type="button"
              onClick={() => void cargarMetricas()}
              disabled={cargando}
              className="inline-flex h-44 items-center justify-center gap-6 rounded-lg border border-concrete bg-canvas-white px-14 text-12 font-semibold text-ash-graphite transition-colors hover:bg-ghost-fog disabled:cursor-wait disabled:opacity-60"
            >
              <span className={`material-symbols-outlined text-16 ${cargando ? "animate-spin" : ""}`}>refresh</span>
              Actualizar
            </button>
          </div>
        </header>

        <div className="flex flex-wrap items-center justify-between gap-8 border-y border-concrete py-10 text-11 text-sage-green">
          <span className="inline-flex items-center gap-6">
            <span className="material-symbols-outlined text-16">calendar_today</span>
            Hoy · 00:00 a 23:59 · Argentina
          </span>
          {actualizadoA && (
            <span>Actualizado {actualizadoA.toLocaleTimeString("es-AR", { hour: "2-digit", minute: "2-digit" })}</span>
          )}
        </div>

        {error ? (
          <section className="rounded-xl border border-[#F3D4A3] bg-[#FFF9EF] p-20">
            <div className="flex flex-col gap-14 sm:flex-row sm:items-center sm:justify-between">
              <div className="flex items-start gap-10">
                <span className="material-symbols-outlined text-20 text-[#8A4D00]">query_stats</span>
                <div>
                  <h3 className="text-14 font-semibold text-ash-graphite">No pudimos cargar las métricas</h3>
                  <p className="mt-4 text-12 text-sage-green">{error}</p>
                </div>
              </div>
              <button
                type="button"
                onClick={() => void cargarMetricas()}
                className="h-40 rounded-lg bg-ash-graphite px-14 text-12 font-semibold text-canvas-white"
              >
                Reintentar
              </button>
            </div>
          </section>
        ) : cargando || !resumen ? (
          <MetricasSkeleton />
        ) : (
          <>
            <section className="grid gap-12 sm:grid-cols-2 xl:grid-cols-4" aria-label="Indicadores principales">
              <KPICard
                icono="payments"
                etiqueta="Facturación de hoy"
                valor={formatearMoneda(resumen.facturacion_total)}
                detalle={`${resumen.pedidos_cerrados} pedidos cerrados`}
                variacion={resumen.variaciones.facturacion_total}
              />
              <KPICard
                icono="receipt_long"
                etiqueta="Ticket promedio"
                valor={formatearMoneda(resumen.ticket_promedio)}
                detalle="Promedio por pedido cerrado"
                variacion={resumen.variaciones.ticket_promedio}
              />
              <KPICard
                icono="orders"
                etiqueta="Pedidos totales"
                valor={resumen.pedidos_totales.toLocaleString("es-AR")}
                detalle={`${resumen.pedidos_activos} activos · ${resumen.pedidos_cerrados} cerrados`}
                variacion={resumen.variaciones.pedidos_totales}
              />
              <KPICard
                icono="timer"
                etiqueta="Tiempo medio de atención"
                valor={formatearTiempo(resumen.tiempo_promedio_despacho_minutos)}
                detalle="Desde el pedido hasta quedar listo"
                variacion={resumen.variaciones.tiempo_promedio_despacho_minutos}
                invertirVariacion
              />
            </section>

            <section className="overflow-hidden rounded-xl border border-concrete bg-canvas-white" aria-labelledby="horarios-pico-titulo">
              <div className="flex flex-col gap-4 border-b border-concrete px-16 py-14 sm:px-20">
                <div className="flex items-center gap-8">
                  <span className="material-symbols-outlined text-20 text-ash-graphite">schedule</span>
                  <h3 id="horarios-pico-titulo" className="text-15 font-semibold text-ash-graphite">Horarios pico</h3>
                </div>
                <p className="text-11 text-sage-green">
                  Pedidos, facturación y ticket promedio por franja para la sucursal seleccionada.
                </p>
              </div>

              {resumenHorarios.turnos.length === 0 ? (
                <div className="grid min-h-160 place-items-center p-24 text-center">
                  <div>
                    <span className="material-symbols-outlined text-30 text-stone">schedule</span>
                    <p className="mt-8 text-13 font-semibold text-ash-graphite">Todavía no hay horarios para comparar</p>
                    <p className="mt-4 text-11 text-sage-green">Aparecerán cuando se registren pedidos durante el día.</p>
                  </div>
                </div>
              ) : (
                <div className="grid gap-16 p-16 sm:p-20 xl:grid-cols-[minmax(280px,0.8fr)_minmax(0,1.8fr)]">
                  <div className="grid gap-10 sm:grid-cols-3 xl:grid-cols-1">
                    <div className="rounded-lg border border-[#C9DCF7] bg-[#EFF6FF] p-14">
                      <p className="text-10 font-mono font-semibold uppercase tracking-[0.08em] text-[#285F9F]">Más pedidos</p>
                      <div className="mt-7 flex items-end justify-between gap-10">
                        <p className="text-16 font-semibold text-ash-graphite">{resumenHorarios.picoPedidos?.turno}</p>
                        <p className="shrink-0 text-12 font-semibold text-[#285F9F]">
                          {resumenHorarios.picoPedidos?.pedidos_totales} pedidos
                        </p>
                      </div>
                    </div>
                    <div className="rounded-lg border border-[#A7E4CC] bg-[#EAF8F2] p-14">
                      <p className="text-10 font-mono font-semibold uppercase tracking-[0.08em] text-[#087657]">Mayor facturación</p>
                      <div className="mt-7 flex items-end justify-between gap-10">
                        <p className="text-16 font-semibold text-ash-graphite">{resumenHorarios.picoFacturacion?.turno}</p>
                        <p className="shrink-0 text-12 font-semibold text-[#087657]">
                          {formatearMoneda(resumenHorarios.picoFacturacion?.facturacion_total ?? 0)}
                        </p>
                      </div>
                    </div>
                    <div className="rounded-lg border border-[#F3D4A3] bg-[#FFF9EF] p-14">
                      <p className="text-10 font-mono font-semibold uppercase tracking-[0.08em] text-[#8A4D00]">Ticket promedio más alto</p>
                      <div className="mt-7 flex items-end justify-between gap-10">
                        <p className="text-16 font-semibold text-ash-graphite">{resumenHorarios.picoTicket?.turno}</p>
                        <p className="shrink-0 text-12 font-semibold text-[#8A4D00]">
                          {formatearMoneda(resumenHorarios.picoTicket?.ticketPromedio ?? 0)}
                        </p>
                      </div>
                    </div>
                  </div>

                  <div className="divide-y divide-ghost-fog">
                    {resumenHorarios.turnos.map((turno) => {
                      const ancho = Math.max((turno.pedidos_totales / resumenHorarios.maximoPedidos) * 100, 4);
                      return (
                        <div key={turno.turno} className="grid gap-8 py-11 first:pt-0 last:pb-0 sm:grid-cols-[90px_minmax(120px,1fr)_auto] sm:items-center">
                          <p className="text-12 font-semibold text-ash-graphite">{turno.turno}</p>
                          <div className="flex items-center gap-10">
                            <div className="h-6 flex-1 overflow-hidden rounded-full bg-ghost-fog">
                              <div className="h-full rounded-full bg-[#287CC1]" style={{ width: `${ancho}%` }} />
                            </div>
                            <span className="w-72 shrink-0 whitespace-nowrap text-right text-10 font-mono text-sage-green">
                              {turno.pedidos_totales} pedidos
                            </span>
                          </div>
                          <div className="flex items-center justify-between gap-12 sm:min-w-[220px] sm:justify-end">
                            <span className="text-11 font-semibold text-ash-graphite">{formatearMoneda(turno.facturacion_total)}</span>
                            <span className="text-10 text-sage-green">Ticket {formatearMoneda(turno.ticketPromedio)}</span>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                </div>
              )}
            </section>

            <div className="grid gap-16 xl:grid-cols-[minmax(0,1.7fr)_minmax(300px,0.8fr)]">
              <section className="overflow-hidden rounded-xl border border-concrete bg-canvas-white">
                <div className="flex items-center justify-between gap-12 border-b border-concrete px-16 py-14 sm:px-20">
                  <div>
                    <h3 className="text-15 font-semibold text-ash-graphite">Platos estrella</h3>
                    <p className="mt-3 text-11 text-sage-green">Ranking por unidades vendidas en pedidos cerrados.</p>
                  </div>
                  <span className="rounded-full bg-ghost-fog px-8 py-3 text-10 font-mono text-sage-green">TOP 5</span>
                </div>

                {resumen.platos_estrella.length === 0 ? (
                  <div className="grid min-h-240 place-items-center p-24 text-center">
                    <div>
                      <span className="material-symbols-outlined text-32 text-stone">restaurant</span>
                      <p className="mt-8 text-13 font-semibold text-ash-graphite">Todavía no hay ventas cerradas hoy</p>
                      <p className="mt-4 text-11 text-sage-green">El ranking aparecerá cuando cierres los primeros pedidos.</p>
                    </div>
                  </div>
                ) : (
                  <ol>
                    {resumen.platos_estrella.map((plato, index) => {
                      const ancho = Math.max((plato.unidades / maximoUnidades) * 100, 6);
                      return (
                        <li key={plato.articulo_id} className="border-b border-ghost-fog px-16 py-14 last:border-b-0 sm:px-20">
                          <div className="flex items-center gap-12">
                            <span className={`grid h-32 w-32 shrink-0 place-items-center rounded-lg text-12 font-bold ${
                              index === 0 ? "bg-ash-graphite text-canvas-white" : "bg-ghost-fog text-sage-green"
                            }`}>
                              {index + 1}
                            </span>
                            <div className="min-w-0 flex-1">
                              <div className="flex flex-wrap items-baseline justify-between gap-6">
                                <p className="truncate text-13 font-semibold text-ash-graphite">{plato.nombre}</p>
                                <p className="text-12 font-semibold text-ash-graphite">{formatearMoneda(plato.monto)}</p>
                              </div>
                              <div className="mt-7 flex items-center gap-10">
                                <div className="h-4 flex-1 overflow-hidden rounded-full bg-ghost-fog">
                                  <div className="h-full rounded-full bg-ash-graphite" style={{ width: `${ancho}%` }} />
                                </div>
                                <span className="shrink-0 whitespace-nowrap text-right text-10 font-mono text-sage-green">
                                  {plato.unidades} unidades
                                </span>
                              </div>
                            </div>
                          </div>
                        </li>
                      );
                    })}
                  </ol>
                )}
              </section>

              <section className="overflow-hidden rounded-xl border border-concrete bg-canvas-white">
                <div className="border-b border-concrete px-16 py-14 sm:px-20">
                  <h3 className="text-15 font-semibold text-ash-graphite">Operación de hoy</h3>
                  <p className="mt-3 text-11 text-sage-green">Estado actual de los pedidos recibidos.</p>
                </div>
                <div className="space-y-14 p-16 sm:p-20">
                  {resumen.por_estado.length === 0 ? (
                    <p className="py-24 text-center text-12 text-sage-green">Sin pedidos registrados en el período.</p>
                  ) : (
                    resumen.por_estado.map((estado) => (
                      <div key={estado.estado} className="flex items-center justify-between gap-10">
                        <span className={`rounded-full border px-10 py-4 text-11 font-semibold ${
                          ESTADO_STYLES[estado.estado] ?? ESTADO_STYLES.cerrado
                        }`}>
                          {ESTADO_LABELS[estado.estado] ?? estado.estado}
                        </span>
                        <span className="text-14 font-semibold text-ash-graphite">{estado.cantidad}</span>
                      </div>
                    ))
                  )}

                  <div className="border-t border-concrete pt-14">
                    <p className="text-10 font-mono font-semibold uppercase tracking-[0.08em] text-sage-green">Más vendido</p>
                    <p className="mt-6 text-14 font-semibold text-ash-graphite">
                      {resumen.plato_mas_vendido?.nombre ?? "Sin datos todavía"}
                    </p>
                    {resumen.plato_mas_vendido && (
                      <p className="mt-3 text-11 text-sage-green">
                        {resumen.plato_mas_vendido.unidades} unidades · {formatearMoneda(resumen.plato_mas_vendido.monto)}
                      </p>
                    )}
                  </div>
                </div>
              </section>
            </div>
          </>
        )}
      </div>
    </div>
  );
}
