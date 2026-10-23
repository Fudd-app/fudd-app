"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import {
  api,
  CeldaCalorAPI,
  getErrorMessage,
  MetricaCategoriaAPI,
  MetricasResumenAPI,
  Sucursal,
} from "@/lib/api";
import { exportarExcelReporte, imprimirInformePDF } from "@/lib/export";
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

const COLORES_CATEGORIA = [
  { bg: "#287CC1", text: "#FFFFFF", border: "#1E5F94" },
  { bg: "#087657", text: "#FFFFFF", border: "#065A42" },
  { bg: "#D97706", text: "#FFFFFF", border: "#B45309" },
  { bg: "#7C3AED", text: "#FFFFFF", border: "#6D28D9" },
  { bg: "#DC2626", text: "#FFFFFF", border: "#B91C1C" },
  { bg: "#0D9488", text: "#FFFFFF", border: "#0F766E" },
  { bg: "#4B5563", text: "#FFFFFF", border: "#374151" },
  { bg: "#EC4899", text: "#FFFFFF", border: "#BE185D" },
];

export type PresetRango = "hoy" | "ayer" | "esta_semana" | "este_mes" | "personalizado";

interface RangoFechas {
  desde: string;
  hasta: string;
  etiqueta: string;
}

function formatearFechaISO(date: Date): string {
  const partes = new Intl.DateTimeFormat("en-CA", {
    timeZone: "America/Argentina/Buenos_Aires",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).formatToParts(date);
  const valor = Object.fromEntries(partes.map((p) => [p.type, p.value]));
  return `${valor.year}-${valor.month}-${valor.day}`;
}

function calcularRango(
  preset: PresetRango,
  personalizadoDesde?: string,
  personalizadoHasta?: string
): RangoFechas {
  const ahora = new Date();
  const hoyStr = formatearFechaISO(ahora);

  if (preset === "hoy") {
    return { desde: hoyStr, hasta: hoyStr, etiqueta: "Hoy" };
  }

  if (preset === "ayer") {
    const ayer = new Date(ahora.getTime() - 24 * 60 * 60 * 1000);
    const ayerStr = formatearFechaISO(ayer);
    return { desde: ayerStr, hasta: ayerStr, etiqueta: "Ayer" };
  }

  if (preset === "esta_semana") {
    const diaSemana = ahora.getDay();
    const diasDesdeLunes = diaSemana === 0 ? 6 : diaSemana - 1;
    const lunes = new Date(ahora.getTime() - diasDesdeLunes * 24 * 60 * 60 * 1000);
    return {
      desde: formatearFechaISO(lunes),
      hasta: hoyStr,
      etiqueta: "Esta semana",
    };
  }

  if (preset === "este_mes") {
    const primerDia = new Date(ahora.getFullYear(), ahora.getMonth(), 1);
    return {
      desde: formatearFechaISO(primerDia),
      hasta: hoyStr,
      etiqueta: "Este mes",
    };
  }

  if (preset === "personalizado" && personalizadoDesde && personalizadoHasta) {
    return {
      desde: personalizadoDesde,
      hasta: personalizadoHasta,
      etiqueta: "Personalizado",
    };
  }

  return { desde: hoyStr, hasta: hoyStr, etiqueta: "Hoy" };
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
      {Math.abs(valor).toLocaleString("es-AR", { maximumFractionDigits: 1 })}% vs. anterior
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
    <article className="rounded-xl border border-concrete bg-canvas-white p-16 sm:p-20 shadow-xs">
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

// ==========================================
// Componente de Distribución por Categoría (US-74)
// ==========================================
function DistribucionCategoriasCard({ categorias }: { categorias: MetricaCategoriaAPI[] }) {
  const [categoriaSeleccionada, setCategoriaSeleccionada] = useState<string | null>(null);

  const totalFacturacion = useMemo(
    () => categorias.reduce((acc, cat) => acc + cat.monto, 0),
    [categorias]
  );
  const totalUnidades = useMemo(
    () => categorias.reduce((acc, cat) => acc + cat.unidades, 0),
    [categorias]
  );

  return (
    <section className="overflow-hidden rounded-xl border border-concrete bg-canvas-white shadow-xs">
      <div className="flex items-center justify-between gap-12 border-b border-concrete px-16 py-14 sm:px-20">
        <div>
          <div className="flex items-center gap-8">
            <span className="material-symbols-outlined text-20 text-ash-graphite">donut_small</span>
            <h3 className="text-15 font-semibold text-ash-graphite">Ventas por categoría</h3>
          </div>
          <p className="mt-2 text-11 text-sage-green">
            Distribución porcentual de facturación sobre pedidos cerrados.
          </p>
        </div>
        {categorias.length > 0 && (
          <span className="rounded-full bg-ghost-fog px-10 py-3 text-10 font-mono font-semibold text-sage-green">
            {categorias.length} rubros
          </span>
        )}
      </div>

      {categorias.length === 0 ? (
        <div className="grid min-h-200 place-items-center p-24 text-center">
          <div>
            <span className="material-symbols-outlined text-32 text-stone">category</span>
            <p className="mt-8 text-13 font-semibold text-ash-graphite">Sin ventas por categoría en el período</p>
            <p className="mt-4 text-11 text-sage-green">Aparecerán cuando se registren pedidos cerrados con artículos.</p>
          </div>
        </div>
      ) : (
        <div className="p-16 sm:p-20 space-y-16">
          {/* Barra apilada segmentada proporcional */}
          <div className="space-y-6">
            <div className="h-14 w-full overflow-hidden rounded-lg flex bg-ghost-fog shadow-inner">
              {categorias.map((cat, idx) => {
                const color = COLORES_CATEGORIA[idx % COLORES_CATEGORIA.length];
                const ancho = Math.max(cat.porcentaje, 1.5);
                const isSelected = categoriaSeleccionada === cat.categoria_id;
                return (
                  <div
                    key={cat.categoria_id}
                    className={`h-full transition-all cursor-pointer relative ${
                      categoriaSeleccionada && !isSelected ? "opacity-35" : "opacity-100"
                    }`}
                    style={{ width: `${ancho}%`, backgroundColor: color.bg }}
                    onMouseEnter={() => setCategoriaSeleccionada(cat.categoria_id)}
                    onMouseLeave={() => setCategoriaSeleccionada(null)}
                    onClick={() =>
                      setCategoriaSeleccionada(isSelected ? null : cat.categoria_id)
                    }
                    title={`${cat.nombre}: ${cat.porcentaje}% (${formatearMoneda(cat.monto)})`}
                  />
                );
              })}
            </div>
            <div className="flex justify-between text-10 font-mono text-sage-green">
              <span>0%</span>
              <span>Total: {formatearMoneda(totalFacturacion)} · {totalUnidades} unidades</span>
              <span>100%</span>
            </div>
          </div>

          {/* Listado desglosado interactivo */}
          <div className="divide-y divide-ghost-fog">
            {categorias.map((cat, idx) => {
              const color = COLORES_CATEGORIA[idx % COLORES_CATEGORIA.length];
              const isSelected = categoriaSeleccionada === cat.categoria_id;
              return (
                <div
                  key={cat.categoria_id}
                  className={`grid gap-8 py-10 transition-colors rounded-lg px-8 -mx-8 cursor-pointer ${
                    isSelected ? "bg-ghost-fog/70" : "hover:bg-ghost-fog/40"
                  }`}
                  onMouseEnter={() => setCategoriaSeleccionada(cat.categoria_id)}
                  onMouseLeave={() => setCategoriaSeleccionada(null)}
                  onClick={() =>
                    setCategoriaSeleccionada(isSelected ? null : cat.categoria_id)
                  }
                >
                  <div className="flex items-center justify-between gap-12">
                    <div className="flex items-center gap-10 min-w-0">
                      <span
                        className="h-10 w-10 shrink-0 rounded-full"
                        style={{ backgroundColor: color.bg }}
                      />
                      <span className="truncate text-13 font-semibold text-ash-graphite">
                        {cat.nombre}
                      </span>
                    </div>
                    <div className="flex items-center gap-12 shrink-0">
                      <span className="text-11 font-mono text-sage-green">
                        {cat.unidades} un.
                      </span>
                      <span className="text-12 font-semibold text-ash-graphite">
                        {formatearMoneda(cat.monto)}
                      </span>
                      <span className="w-48 text-right text-11 font-bold text-ash-graphite">
                        {cat.porcentaje.toFixed(1)}%
                      </span>
                    </div>
                  </div>
                  {/* Barra de progreso por fila */}
                  <div className="h-4 w-full overflow-hidden rounded-full bg-ghost-fog">
                    <div
                      className="h-full rounded-full transition-all duration-300"
                      style={{
                        width: `${Math.min(cat.porcentaje, 100)}%`,
                        backgroundColor: color.bg,
                      }}
                    />
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}
    </section>
  );
}

// ==========================================
// Componente de Mapa de Calor de Horarios (US-74)
// ==========================================
const DIAS_SEMANA_HEATMAP = [
  { dow: 1, label: "Lun", nombre: "Lunes" },
  { dow: 2, label: "Mar", nombre: "Martes" },
  { dow: 3, label: "Mié", nombre: "Miércoles" },
  { dow: 4, label: "Jue", nombre: "Jueves" },
  { dow: 5, label: "Vie", nombre: "Viernes" },
  { dow: 6, label: "Sáb", nombre: "Sábado" },
  { dow: 0, label: "Dom", nombre: "Domingo" },
];

const HORARIOS_HEATMAP = [8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 0, 1, 2];

function MapaCalorCard({ mapaCalor }: { mapaCalor: CeldaCalorAPI[] }) {
  const [metricaActiva, setMetricaActiva] = useState<"pedidos" | "monto">("pedidos");
  const [celdaHover, setCeldaHover] = useState<{
    diaNombre: string;
    hora: number;
    pedidos: number;
    monto: number;
  } | null>(null);

  const celdaMap = useMemo(() => {
    const mapa = new Map<string, CeldaCalorAPI>();
    for (const celda of mapaCalor) {
      mapa.set(`${celda.dia_semana}_${celda.hora}`, celda);
    }
    return mapa;
  }, [mapaCalor]);

  const { maximoValor, picoDia, picoHora, totalPedidos, totalMonto } = useMemo(() => {
    let max = 0;
    let pedidosTotales = 0;
    let montoTotal = 0;

    const pedidosPorDia = new Map<number, number>();
    const pedidosPorHora = new Map<number, number>();

    for (const celda of mapaCalor) {
      const valor = metricaActiva === "pedidos" ? celda.pedidos : celda.monto;
      if (valor > max) max = valor;

      pedidosTotales += celda.pedidos;
      montoTotal += celda.monto;

      pedidosPorDia.set(celda.dia_semana, (pedidosPorDia.get(celda.dia_semana) || 0) + celda.pedidos);
      pedidosPorHora.set(celda.hora, (pedidosPorHora.get(celda.hora) || 0) + celda.pedidos);
    }

    let mejorDiaDow = 5;
    let maxDia = 0;
    pedidosPorDia.forEach((cant, dow) => {
      if (cant > maxDia) {
        maxDia = cant;
        mejorDiaDow = dow;
      }
    });

    let mejorHora = 21;
    let maxHora = 0;
    pedidosPorHora.forEach((cant, hora) => {
      if (cant > maxHora) {
        maxHora = cant;
        mejorHora = hora;
      }
    });

    const diaObj = DIAS_SEMANA_HEATMAP.find((d) => d.dow === mejorDiaDow);

    return {
      maximoValor: Math.max(max, 1),
      picoDia: diaObj ? diaObj.nombre : "Viernes",
      picoHora: `${mejorHora.toString().padStart(2, "0")}:00 hs`,
      totalPedidos: pedidosTotales,
      totalMonto: montoTotal,
    };
  }, [mapaCalor, metricaActiva]);

  function obtenerEstiloCelda(pedidos: number, monto: number) {
    const valor = metricaActiva === "pedidos" ? pedidos : monto;
    if (valor <= 0) {
      return "bg-[#F4F6F7] border-concrete/60 hover:border-ash-graphite";
    }
    const ratio = valor / maximoValor;
    if (ratio < 0.2) return "bg-[#EAF8F2] border-[#A7E4CC] text-[#087657]";
    if (ratio < 0.45) return "bg-[#A7E4CC] border-[#6EE7B7] text-[#065F46]";
    if (ratio < 0.7) return "bg-[#34D399] border-[#10B981] text-[#064E3B] font-semibold";
    if (ratio < 0.9) return "bg-[#087657] border-[#065F46] text-white font-bold";
    return "bg-[#044E39] border-[#023124] text-white font-bold ring-1 ring-[#044E39]";
  }

  return (
    <section className="overflow-hidden rounded-xl border border-concrete bg-canvas-white shadow-xs">
      <div className="flex flex-col gap-12 border-b border-concrete px-16 py-14 sm:flex-row sm:items-center sm:justify-between sm:px-20">
        <div>
          <div className="flex items-center gap-8">
            <span className="material-symbols-outlined text-20 text-ash-graphite">grid_view</span>
            <h3 className="text-15 font-semibold text-ash-graphite">Mapa de calor de afluencia</h3>
          </div>
          <p className="mt-2 text-11 text-sage-green">
            Patrones de concentración horaria por día de la semana (08:00 a 02:00).
          </p>
        </div>

        <div className="flex items-center gap-6 rounded-lg border border-concrete bg-ghost-fog p-3">
          <button
            type="button"
            onClick={() => setMetricaActiva("pedidos")}
            className={`rounded-md px-10 py-5 text-11 font-semibold transition-all ${
              metricaActiva === "pedidos"
                ? "bg-canvas-white text-ash-graphite shadow-xs"
                : "text-sage-green hover:text-ash-graphite"
            }`}
          >
            Por pedidos
          </button>
          <button
            type="button"
            onClick={() => setMetricaActiva("monto")}
            className={`rounded-md px-10 py-5 text-11 font-semibold transition-all ${
              metricaActiva === "monto"
                ? "bg-canvas-white text-ash-graphite shadow-xs"
                : "text-sage-green hover:text-ash-graphite"
            }`}
          >
            Por facturación ($)
          </button>
        </div>
      </div>

      <div className="p-16 sm:p-20 space-y-16">
        <div className="overflow-x-auto pb-8">
          <div className="min-w-[680px]">
            <div className="grid grid-cols-[54px_repeat(19,minmax(28px,1fr))] gap-4 pb-6 text-center text-10 font-mono text-sage-green">
              <span className="text-left font-semibold">Día</span>
              {HORARIOS_HEATMAP.map((hora) => (
                <span key={hora} className="truncate">
                  {hora.toString().padStart(2, "0")}h
                </span>
              ))}
            </div>

            <div className="space-y-4">
              {DIAS_SEMANA_HEATMAP.map((dia) => (
                <div
                  key={dia.dow}
                  className="grid grid-cols-[54px_repeat(19,minmax(28px,1fr))] items-center gap-4"
                >
                  <span className="text-11 font-semibold text-ash-graphite">{dia.label}</span>
                  {HORARIOS_HEATMAP.map((hora) => {
                    const celda = celdaMap.get(`${dia.dow}_${hora}`) ?? {
                      dia_semana: dia.dow,
                      hora,
                      pedidos: 0,
                      monto: 0,
                    };
                    const valorActual = metricaActiva === "pedidos" ? celda.pedidos : celda.monto;
                    const estilo = obtenerEstiloCelda(celda.pedidos, celda.monto);

                    return (
                      <div
                        key={hora}
                        tabIndex={0}
                        aria-label={`${dia.nombre} ${hora}:00 - ${celda.pedidos} pedidos`}
                        className={`h-28 rounded-md border flex items-center justify-center transition-transform hover:scale-115 hover:z-10 cursor-pointer ${estilo}`}
                        onMouseEnter={() =>
                          setCeldaHover({
                            diaNombre: dia.nombre,
                            hora,
                            pedidos: celda.pedidos,
                            monto: celda.monto,
                          })
                        }
                        onMouseLeave={() => setCeldaHover(null)}
                      >
                        {valorActual > 0 && metricaActiva === "pedidos" && (
                          <span className="text-9 leading-none">{celda.pedidos}</span>
                        )}
                      </div>
                    );
                  })}
                </div>
              ))}
            </div>
          </div>
        </div>

        <div className="min-h-44 rounded-lg border border-concrete bg-ghost-fog/80 px-14 py-8 flex flex-wrap items-center justify-between gap-10 text-11">
          {celdaHover ? (
            <>
              <div className="flex items-center gap-8 text-ash-graphite font-semibold">
                <span className="material-symbols-outlined text-16 text-[#087657]">schedule</span>
                <span>
                  {celdaHover.diaNombre} · {celdaHover.hora.toString().padStart(2, "0")}:00 a{" "}
                  {((celdaHover.hora + 1) % 24).toString().padStart(2, "0")}:00 hs
                </span>
              </div>
              <div className="flex items-center gap-14">
                <span className="text-sage-green">
                  Comandas: <strong className="text-ash-graphite">{celdaHover.pedidos}</strong>
                </span>
                <span className="text-sage-green">
                  Facturación: <strong className="text-ash-graphite">{formatearMoneda(celdaHover.monto)}</strong>
                </span>
                {celdaHover.pedidos > 0 && (
                  <span className="text-sage-green">
                    Ticket Prom:{" "}
                    <strong className="text-ash-graphite">
                      {formatearMoneda(celdaHover.monto / celdaHover.pedidos)}
                    </strong>
                  </span>
                )}
              </div>
            </>
          ) : (
            <div className="flex items-center gap-8 text-sage-green">
              <span className="material-symbols-outlined text-16">info</span>
              <span>Pasá el cursor o tocá cualquier celda para ver el detalle de pedidos y facturación.</span>
            </div>
          )}
        </div>

        <div className="flex flex-col gap-12 sm:flex-row sm:items-center sm:justify-between border-t border-concrete pt-14 text-11 text-sage-green">
          <div className="flex items-center gap-8">
            <span className="text-10 font-mono">Baja demanda</span>
            <div className="flex items-center gap-3">
              <span className="h-12 w-12 rounded border border-concrete/60 bg-[#F4F6F7]" />
              <span className="h-12 w-12 rounded border border-[#A7E4CC] bg-[#EAF8F2]" />
              <span className="h-12 w-12 rounded border border-[#6EE7B7] bg-[#A7E4CC]" />
              <span className="h-12 w-12 rounded border border-[#10B981] bg-[#34D399]" />
              <span className="h-12 w-12 rounded border border-[#065F46] bg-[#087657]" />
              <span className="h-12 w-12 rounded border border-[#023124] bg-[#044E39]" />
            </div>
            <span className="text-10 font-mono">Pico</span>
          </div>

          <div className="flex flex-wrap items-center gap-14 text-11 font-medium text-ash-graphite">
            <span>
              Día más fuerte: <strong className="text-[#087657]">{picoDia}</strong>
            </span>
            <span>
              Horario pico: <strong className="text-[#087657]">{picoHora}</strong>
            </span>
            <span>
              Total período: <strong className="text-ash-graphite">{totalPedidos} pedidos</strong> (
              {formatearMoneda(totalMonto)})
            </span>
          </div>
        </div>
      </div>
    </section>
  );
}

function MetricasSkeleton() {
  return (
    <div className="space-y-16" aria-label="Calculando métricas">
      <div className="grid gap-12 sm:grid-cols-2 xl:grid-cols-4">
        {Array.from({ length: 4 }, (_, index) => (
          <div key={index} className="rounded-xl border border-concrete bg-canvas-white p-20 shadow-xs">
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
      <div className="grid gap-16 xl:grid-cols-2">
        <Skeleton className="h-[320px] w-full rounded-xl" />
        <Skeleton className="h-[320px] w-full rounded-xl" />
      </div>
      <Skeleton className="h-[280px] w-full rounded-xl" />
    </div>
  );
}

export default function MetricasSection() {
  const [resumen, setResumen] = useState<MetricasResumenAPI | null>(null);
  const [sucursales, setSucursales] = useState<Sucursal[]>([]);
  const [sucursalId, setSucursalId] = useState("");
  const [presetRango, setPresetRango] = useState<PresetRango>("esta_semana");
  const [fechaDesdePersonalizada, setFechaDesdePersonalizada] = useState(() => formatearFechaISO(new Date()));
  const [fechaHastaPersonalizada, setFechaHastaPersonalizada] = useState(() => formatearFechaISO(new Date()));
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

  const cargarMetricas = useCallback(
    async (
      sucursalSeleccionada = sucursalId,
      presetSeleccionado = presetRango,
      desdePersonalizado = fechaDesdePersonalizada,
      hastaPersonalizado = fechaHastaPersonalizada
    ) => {
      setCargando(true);
      setError("");
      try {
        const rango = calcularRango(presetSeleccionado, desdePersonalizado, hastaPersonalizado);
        const data = await api.obtenerMetricasResumen({
          desde: rango.desde,
          hasta: rango.hasta,
          sucursalId: sucursalSeleccionada || undefined,
        });
        setResumen(data);
        setActualizadoA(new Date());
      } catch (err) {
        setError(getErrorMessage(err, "No pudimos calcular las métricas del negocio."));
      } finally {
        setCargando(false);
      }
    },
    [sucursalId, presetRango, fechaDesdePersonalizada, fechaHastaPersonalizada]
  );

  useEffect(() => {
    void Promise.resolve().then(() => cargarMetricas());
  }, [cargarMetricas]);

  const maximoUnidades = useMemo(
    () => Math.max(...(resumen?.platos_estrella.map((plato) => plato.unidades) ?? [0]), 1),
    [resumen]
  );

  const resumenHorarios = useMemo(() => {
    const turnos = (resumen?.por_turno ?? []).map((turno) => ({
      ...turno,
      ticketPromedio:
        turno.pedidos_cerrados > 0
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

  const rangoActual = useMemo(
    () => calcularRango(presetRango, fechaDesdePersonalizada, fechaHastaPersonalizada),
    [presetRango, fechaDesdePersonalizada, fechaHastaPersonalizada]
  );

  const sucursalNombre = useMemo(
    () => sucursales.find((s) => s.id === sucursalId)?.nombre ?? "Todo el negocio",
    [sucursales, sucursalId]
  );

  function handleExportarExcel() {
    if (!resumen) return;
    exportarExcelReporte(resumen, "Mesa CLICK", sucursalNombre);
  }

  function handleImprimirPDF() {
    if (!resumen) return;
    imprimirInformePDF(resumen, "Mesa CLICK", sucursalNombre);
  }

  return (
    <div className="h-full overflow-y-auto bg-[#F4F6F7] p-16 font-inter sm:p-24 md:p-32">
      <div className="mx-auto w-full max-w-[1440px] space-y-20">
        {/* Encabezado principal */}
        <header className="flex flex-col gap-14 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <p className="text-10 font-mono font-semibold uppercase tracking-[0.12em] text-sage-green">
              Business Analytics & Rendimiento
            </p>
            <h2 className="mt-4 text-24 font-semibold tracking-[-0.03em] text-ash-graphite sm:text-32">
              Métricas & Estadísticas
            </h2>
            <p className="mt-4 text-13 text-sage-green">
              Lectura consolidada de ventas, demanda por categoría, horarios pico y atención.
            </p>
          </div>

          <div className="flex flex-wrap items-center gap-10">
            {/* Selector de Presets Temporales (US-75) */}
            <div className="flex items-center rounded-lg border border-concrete bg-canvas-white p-3 shadow-xs">
              {(
                [
                  { id: "hoy", label: "Hoy" },
                  { id: "ayer", label: "Ayer" },
                  { id: "esta_semana", label: "Esta semana" },
                  { id: "este_mes", label: "Este mes" },
                  { id: "personalizado", label: "Personalizado" },
                ] as const
              ).map((preset) => (
                <button
                  key={preset.id}
                  type="button"
                  onClick={() => {
                    setPresetRango(preset.id);
                    if (preset.id !== "personalizado") {
                      void cargarMetricas(sucursalId, preset.id);
                    }
                  }}
                  className={`rounded-md px-10 py-5 text-11 font-semibold transition-all ${
                    presetRango === preset.id
                      ? "bg-ash-graphite text-canvas-white shadow-xs"
                      : "text-sage-green hover:text-ash-graphite"
                  }`}
                >
                  {preset.label}
                </button>
              ))}
            </div>

            {/* Selector de Sucursal */}
            <label className="sr-only" htmlFor="filtro-sucursal-metricas">
              Sucursal
            </label>
            <select
              id="filtro-sucursal-metricas"
              value={sucursalId}
              onChange={(event) => {
                const siguienteSucursal = event.target.value;
                setSucursalId(siguienteSucursal);
                void cargarMetricas(siguienteSucursal, presetRango);
              }}
              className="h-40 min-w-[180px] rounded-lg border border-concrete bg-canvas-white px-12 text-12 text-ash-graphite shadow-xs"
            >
              <option value="">Todo el negocio</option>
              {sucursales.map((sucursal) => (
                <option key={sucursal.id} value={sucursal.id}>
                  {sucursal.nombre}
                </option>
              ))}
            </select>

            {/* Botones de Exportación (US-75) */}
            <div className="flex items-center gap-6">
              <button
                type="button"
                onClick={handleExportarExcel}
                disabled={!resumen || cargando}
                className="inline-flex h-40 items-center justify-center gap-6 rounded-lg border border-concrete bg-canvas-white px-12 text-12 font-semibold text-ash-graphite shadow-xs transition-colors hover:bg-ghost-fog disabled:cursor-not-allowed disabled:opacity-50"
                title="Exportar a Excel / CSV (.csv UTF-8 compatible)"
              >
                <span className="material-symbols-outlined text-18 text-[#087657]">table_view</span>
                Excel
              </button>
              <button
                type="button"
                onClick={handleImprimirPDF}
                disabled={!resumen || cargando}
                className="inline-flex h-40 items-center justify-center gap-6 rounded-lg border border-concrete bg-canvas-white px-12 text-12 font-semibold text-ash-graphite shadow-xs transition-colors hover:bg-ghost-fog disabled:cursor-not-allowed disabled:opacity-50"
                title="Descargar o imprimir informe ejecutivo en PDF (A4)"
              >
                <span className="material-symbols-outlined text-18 text-[#287CC1]">picture_as_pdf</span>
                PDF
              </button>
            </div>

            {/* Botón Refrescar */}
            <button
              type="button"
              onClick={() => void cargarMetricas()}
              disabled={cargando}
              className="inline-flex h-40 items-center justify-center gap-6 rounded-lg border border-concrete bg-canvas-white px-12 text-12 font-semibold text-ash-graphite shadow-xs transition-colors hover:bg-ghost-fog disabled:cursor-wait disabled:opacity-60"
            >
              <span className={`material-symbols-outlined text-16 ${cargando ? "animate-spin" : ""}`}>
                refresh
              </span>
              Actualizar
            </button>
          </div>
        </header>

        {/* Barra de Rango Personalizado si está activo (US-75) */}
        {presetRango === "personalizado" && (
          <section className="flex flex-wrap items-center justify-between gap-12 rounded-xl border border-concrete bg-canvas-white p-14 shadow-xs">
            <div className="flex items-center gap-8 text-12 font-semibold text-ash-graphite">
              <span className="material-symbols-outlined text-18 text-sage-green">date_range</span>
              <span>Filtro de fechas personalizado:</span>
            </div>
            <div className="flex flex-wrap items-center gap-10">
              <div className="flex items-center gap-6 text-11 text-sage-green">
                <span>Desde:</span>
                <input
                  type="date"
                  value={fechaDesdePersonalizada}
                  max={fechaHastaPersonalizada}
                  onChange={(e) => setFechaDesdePersonalizada(e.target.value)}
                  className="h-36 rounded-lg border border-concrete bg-ghost-fog px-10 text-12 text-ash-graphite"
                />
              </div>
              <div className="flex items-center gap-6 text-11 text-sage-green">
                <span>Hasta:</span>
                <input
                  type="date"
                  value={fechaHastaPersonalizada}
                  min={fechaDesdePersonalizada}
                  onChange={(e) => setFechaHastaPersonalizada(e.target.value)}
                  className="h-36 rounded-lg border border-concrete bg-ghost-fog px-10 text-12 text-ash-graphite"
                />
              </div>
              <button
                type="button"
                onClick={() =>
                  void cargarMetricas(
                    sucursalId,
                    "personalizado",
                    fechaDesdePersonalizada,
                    fechaHastaPersonalizada
                  )
                }
                className="h-36 rounded-lg bg-ash-graphite px-14 text-12 font-semibold text-canvas-white transition-opacity hover:opacity-90"
              >
                Aplicar filtro
              </button>
            </div>
          </section>
        )}

        {/* Barra de estado del período activo */}
        <div className="flex flex-wrap items-center justify-between gap-8 border-y border-concrete py-10 text-11 text-sage-green">
          <span className="inline-flex items-center gap-6 font-medium">
            <span className="material-symbols-outlined text-16 text-ash-graphite">calendar_month</span>
            Período: <strong className="text-ash-graphite">{rangoActual.etiqueta}</strong> ({rangoActual.desde} al{" "}
            {rangoActual.hasta}) · {sucursalNombre} · ART
          </span>
          {actualizadoA && (
            <span>
              Actualizado {actualizadoA.toLocaleTimeString("es-AR", { hour: "2-digit", minute: "2-digit" })}
            </span>
          )}
        </div>

        {error ? (
          <section className="rounded-xl border border-[#F3D4A3] bg-[#FFF9EF] p-20 shadow-xs">
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
            {/* 1. KPIs Principales */}
            <section className="grid gap-12 sm:grid-cols-2 xl:grid-cols-4" aria-label="Indicadores principales">
              <KPICard
                icono="payments"
                etiqueta="Facturación período"
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
                etiqueta="Tiempo medio despacho"
                valor={formatearTiempo(resumen.tiempo_promedio_despacho_minutos)}
                detalle="Desde comanda hasta quedar listo"
                variacion={resumen.variaciones.tiempo_promedio_despacho_minutos}
                invertirVariacion
              />
            </section>

            {/* 2. Fila Analítica: Ventas por Categoría (US-74) + Horarios Pico (Turnos) */}
            <div className="grid gap-16 xl:grid-cols-2">
              <DistribucionCategoriasCard categorias={resumen.por_categoria ?? []} />

              {/* Turnos / Franjas Horarias */}
              <section
                className="overflow-hidden rounded-xl border border-concrete bg-canvas-white shadow-xs"
                aria-labelledby="horarios-pico-titulo"
              >
                <div className="flex flex-col gap-4 border-b border-concrete px-16 py-14 sm:px-20">
                  <div className="flex items-center gap-8">
                    <span className="material-symbols-outlined text-20 text-ash-graphite">schedule</span>
                    <h3 id="horarios-pico-titulo" className="text-15 font-semibold text-ash-graphite">
                      Rendimiento por turno
                    </h3>
                  </div>
                  <p className="text-11 text-sage-green">
                    Pedidos y facturación desglosados por turno para la sucursal seleccionada.
                  </p>
                </div>

                {resumenHorarios.turnos.length === 0 ? (
                  <div className="grid min-h-200 place-items-center p-24 text-center">
                    <div>
                      <span className="material-symbols-outlined text-30 text-stone">schedule</span>
                      <p className="mt-8 text-13 font-semibold text-ash-graphite">Todavía no hay turnos registrados</p>
                      <p className="mt-4 text-11 text-sage-green">Aparecerán cuando se registren pedidos en el período.</p>
                    </div>
                  </div>
                ) : (
                  <div className="space-y-16 p-16 sm:p-20">
                    <div className="grid gap-10 sm:grid-cols-3">
                      <div className="rounded-lg border border-[#C9DCF7] bg-[#EFF6FF] p-12">
                        <p className="text-10 font-mono font-semibold uppercase tracking-[0.08em] text-[#285F9F]">
                          Más pedidos
                        </p>
                        <p className="mt-4 text-14 font-semibold text-ash-graphite">
                          {resumenHorarios.picoPedidos?.turno ?? "-"}
                        </p>
                        <p className="mt-2 text-11 font-semibold text-[#285F9F]">
                          {resumenHorarios.picoPedidos?.pedidos_totales ?? 0} pedidos
                        </p>
                      </div>
                      <div className="rounded-lg border border-[#A7E4CC] bg-[#EAF8F2] p-12">
                        <p className="text-10 font-mono font-semibold uppercase tracking-[0.08em] text-[#087657]">
                          Mayor facturación
                        </p>
                        <p className="mt-4 text-14 font-semibold text-ash-graphite">
                          {resumenHorarios.picoFacturacion?.turno ?? "-"}
                        </p>
                        <p className="mt-2 text-11 font-semibold text-[#087657]">
                          {formatearMoneda(resumenHorarios.picoFacturacion?.facturacion_total ?? 0)}
                        </p>
                      </div>
                      <div className="rounded-lg border border-[#F3D4A3] bg-[#FFF9EF] p-12">
                        <p className="text-10 font-mono font-semibold uppercase tracking-[0.08em] text-[#8A4D00]">
                          Mayor ticket
                        </p>
                        <p className="mt-4 text-14 font-semibold text-ash-graphite">
                          {resumenHorarios.picoTicket?.turno ?? "-"}
                        </p>
                        <p className="mt-2 text-11 font-semibold text-[#8A4D00]">
                          {formatearMoneda(resumenHorarios.picoTicket?.ticketPromedio ?? 0)}
                        </p>
                      </div>
                    </div>

                    <div className="divide-y divide-ghost-fog">
                      {resumenHorarios.turnos.map((turno) => {
                        const ancho = Math.max((turno.pedidos_totales / resumenHorarios.maximoPedidos) * 100, 4);
                        return (
                          <div
                            key={turno.turno}
                            className="grid gap-8 py-10 first:pt-0 last:pb-0 sm:grid-cols-[90px_minmax(120px,1fr)_auto] sm:items-center"
                          >
                            <p className="text-12 font-semibold text-ash-graphite">{turno.turno}</p>
                            <div className="flex items-center gap-10">
                              <div className="h-6 flex-1 overflow-hidden rounded-full bg-ghost-fog">
                                <div className="h-full rounded-full bg-[#287CC1]" style={{ width: `${ancho}%` }} />
                              </div>
                              <span className="w-72 shrink-0 whitespace-nowrap text-right text-10 font-mono text-sage-green">
                                {turno.pedidos_totales} pedidos
                              </span>
                            </div>
                            <div className="flex items-center justify-between gap-12 sm:min-w-[200px] sm:justify-end">
                              <span className="text-11 font-semibold text-ash-graphite">
                                {formatearMoneda(turno.facturacion_total)}
                              </span>
                              <span className="text-10 text-sage-green">
                                Ticket {formatearMoneda(turno.ticketPromedio)}
                              </span>
                            </div>
                          </div>
                        );
                      })}
                    </div>
                  </div>
                )}
              </section>
            </div>

            {/* 3. Matriz de Calor de Horarios (US-74) */}
            <MapaCalorCard mapaCalor={resumen.mapa_calor ?? []} />

            {/* 4. Platos Estrella y Operación */}
            <div className="grid gap-16 xl:grid-cols-[minmax(0,1.7fr)_minmax(300px,0.8fr)]">
              <section className="overflow-hidden rounded-xl border border-concrete bg-canvas-white shadow-xs">
                <div className="flex items-center justify-between gap-12 border-b border-concrete px-16 py-14 sm:px-20">
                  <div>
                    <h3 className="text-15 font-semibold text-ash-graphite">Platos estrella</h3>
                    <p className="mt-3 text-11 text-sage-green">Ranking por unidades vendidas en pedidos cerrados.</p>
                  </div>
                  <span className="rounded-full bg-ghost-fog px-8 py-3 text-10 font-mono font-semibold text-sage-green">
                    TOP 5
                  </span>
                </div>

                {resumen.platos_estrella.length === 0 ? (
                  <div className="grid min-h-240 place-items-center p-24 text-center">
                    <div>
                      <span className="material-symbols-outlined text-32 text-stone">restaurant</span>
                      <p className="mt-8 text-13 font-semibold text-ash-graphite">Todavía no hay ventas cerradas en el período</p>
                      <p className="mt-4 text-11 text-sage-green">El ranking aparecerá al registrarse los pedidos.</p>
                    </div>
                  </div>
                ) : (
                  <ol>
                    {resumen.platos_estrella.map((plato, index) => {
                      const ancho = Math.max((plato.unidades / maximoUnidades) * 100, 6);
                      return (
                        <li key={plato.articulo_id} className="border-b border-ghost-fog px-16 py-14 last:border-b-0 sm:px-20">
                          <div className="flex items-center gap-12">
                            <span
                              className={`grid h-32 w-32 shrink-0 place-items-center rounded-lg text-12 font-bold ${
                                index === 0 ? "bg-ash-graphite text-canvas-white" : "bg-ghost-fog text-sage-green"
                              }`}
                            >
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

              <section className="overflow-hidden rounded-xl border border-concrete bg-canvas-white shadow-xs">
                <div className="border-b border-concrete px-16 py-14 sm:px-20">
                  <h3 className="text-15 font-semibold text-ash-graphite">Operación del período</h3>
                  <p className="mt-3 text-11 text-sage-green">Estado de los pedidos registrados.</p>
                </div>
                <div className="space-y-14 p-16 sm:p-20">
                  {resumen.por_estado.length === 0 ? (
                    <p className="py-24 text-center text-12 text-sage-green">Sin pedidos registrados en el período.</p>
                  ) : (
                    resumen.por_estado.map((estado) => (
                      <div key={estado.estado} className="flex items-center justify-between gap-10">
                        <span
                          className={`rounded-full border px-10 py-4 text-11 font-semibold ${
                            ESTADO_STYLES[estado.estado] ?? ESTADO_STYLES.cerrado
                          }`}
                        >
                          {ESTADO_LABELS[estado.estado] ?? estado.estado}
                        </span>
                        <span className="text-14 font-semibold text-ash-graphite">{estado.cantidad}</span>
                      </div>
                    ))
                  )}

                  <div className="border-t border-concrete pt-14">
                    <p className="text-10 font-mono font-semibold uppercase tracking-[0.08em] text-sage-green">
                      Más vendido
                    </p>
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
