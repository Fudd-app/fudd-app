"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { api, getErrorMessage, ResenaItem, ResenaResumen, Sucursal } from "@/lib/api";
import EmptyState from "@/components/ui/EmptyState";
import Skeleton from "@/components/ui/Skeleton";

interface AlertaCritica {
  id: string;
  mesaNumero: number;
  estrellas: number;
  comentario: string | null;
  timestamp: string;
}

function formatearTiempoRelativo(fechaStr: string): string {
  try {
    const fecha = new Date(fechaStr);
    const ahora = new Date();
    const diffMs = ahora.getTime() - fecha.getTime();
    const diffSeg = Math.floor(diffMs / 1000);
    if (diffSeg < 60) return "hace unos segundos";
    const diffMin = Math.floor(diffSeg / 60);
    if (diffMin < 60) return `hace ${diffMin} min`;
    const diffHoras = Math.floor(diffMin / 60);
    if (diffHoras < 24) return `hace ${diffHoras} h`;
    const diffDias = Math.floor(diffHoras / 24);
    if (diffDias === 1) return "ayer";
    if (diffDias < 30) return `hace ${diffDias} d`;
    return fecha.toLocaleDateString("es-AR", { day: "2-digit", month: "2-digit" });
  } catch {
    return fechaStr;
  }
}

function ReputacionSkeleton() {
  return (
    <div className="space-y-20 animate-pulse" aria-label="Cargando reseñas">
      <div className="grid gap-16 md:grid-cols-3">
        <div className="rounded-xl border border-concrete bg-canvas-white p-20">
          <Skeleton className="h-14 w-120 mb-12" />
          <Skeleton className="h-44 w-160 mb-8" />
          <Skeleton className="h-12 w-200" />
        </div>
        <div className="rounded-xl border border-concrete bg-canvas-white p-20">
          <Skeleton className="h-14 w-120 mb-12" />
          <Skeleton className="h-44 w-160 mb-8" />
          <Skeleton className="h-12 w-200" />
        </div>
        <div className="rounded-xl border border-concrete bg-canvas-white p-20">
          <Skeleton className="h-14 w-120 mb-12" />
          <Skeleton className="h-44 w-160 mb-8" />
          <Skeleton className="h-12 w-200" />
        </div>
      </div>
      <div className="rounded-xl border border-concrete bg-canvas-white p-20">
        <Skeleton className="h-20 w-180 mb-16" />
        <div className="space-y-12">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-14 w-full" />
          ))}
        </div>
      </div>
    </div>
  );
}

export default function ReputacionSection() {
  const [resumen, setResumen] = useState<ResenaResumen | null>(null);
  const [sucursales, setSucursales] = useState<Sucursal[]>([]);
  const [sucursalId, setSucursalId] = useState<string>("");
  const [cargando, setCargando] = useState<boolean>(true);
  const [error, setError] = useState<string>("");
  const [alertasCriticas, setAlertasCriticas] = useState<AlertaCritica[]>([]);
  const eventSourceRef = useRef<EventSource | null>(null);

  // 1. Cargar sucursales del tenant
  useEffect(() => {
    let activo = true;
    api.listarSucursales()
      .then((items) => {
        if (activo) setSucursales(items || []);
      })
      .catch(() => {
        if (activo) setSucursales([]);
      });
    return () => {
      activo = false;
    };
  }, []);

  // 2. Cargar métricas y reseñas del backend
  const cargarDatos = useCallback(async () => {
    try {
      setCargando(true);
      setError("");
      const datos = await api.obtenerResumenResenas(sucursalId || undefined);
      setResumen(datos);
    } catch (err) {
      setError(getErrorMessage(err, "No pudimos obtener el resumen de reseñas y reputación."));
    } finally {
      setCargando(false);
    }
  }, [sucursalId]);

  useEffect(() => {
    const timeoutId = window.setTimeout(() => {
      void cargarDatos();
    }, 0);
    return () => window.clearTimeout(timeoutId);
  }, [cargarDatos]);

  // 3. Suscripción SSE en tiempo real a la sucursal activa para capturar calificaciones bajas
  const sucursalParaSSE = useMemo(() => {
    if (sucursalId) return sucursalId;
    if (sucursales.length > 0) return sucursales[0].id;
    return null;
  }, [sucursalId, sucursales]);

  useEffect(() => {
    if (!sucursalParaSSE) return;

    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      eventSourceRef.current = null;
    }

    const sseUrl = api.obtenerEventosSucursalUrl(sucursalParaSSE);
		const es = new EventSource(sseUrl, { withCredentials: true });
    eventSourceRef.current = es;

    es.addEventListener("resena_creada", (e: MessageEvent) => {
      try {
        const data = JSON.parse(e.data);
        if (data && typeof data.estrellas === "number") {
          // Si es rating bajo (1 a 3★), desplegar banner de alerta inmediata
          if (data.estrellas <= 3) {
            setAlertasCriticas((prev) => [
              {
                id: data.id || String(Date.now()),
                mesaNumero: data.mesa_numero || 0,
                estrellas: data.estrellas,
                comentario: data.comentario || null,
                timestamp: data.created_at || new Date().toISOString(),
              },
              ...prev,
            ]);
          }

          // Refrescar automáticamente el resumen y listado
          void cargarDatos();
        }
      } catch (err) {
        console.warn("Error procesando resena_creada SSE:", err);
      }
    });

    es.onerror = () => {
      // Reintento automático del navegador
    };

    return () => {
      es.close();
      eventSourceRef.current = null;
    };
  }, [sucursalParaSSE, cargarDatos]);

  const descartarAlerta = (id: string) => {
    setAlertasCriticas((prev) => prev.filter((a) => a.id !== id));
  };

  const totalResenas = resumen?.total ?? 0;
  const csat = resumen?.csat ?? 0;
  const promedio = resumen?.promedio ?? 0;
  const clicsGoogle = resumen?.clics_google ?? 0;

  return (
    <div className="h-full overflow-y-auto bg-[#F4F6F7] p-16 font-inter sm:p-24 md:p-32">
      <div className="mx-auto w-full max-w-[1440px] space-y-20">
        {/* Encabezado con selector de sucursal */}
        <header className="flex flex-col gap-12 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <p className="text-10 font-mono font-semibold uppercase tracking-[0.12em] text-sage-green">
              Voz del cliente & Reputación
            </p>
            <h1 className="mt-4 text-24 font-semibold tracking-[-0.03em] text-ash-graphite sm:text-32">
              Reseñas y Reputación
            </h1>
            <p className="mt-4 text-13 text-sage-green">
              Monitoreá el índice de satisfacción (CSAT), las derivaciones a Google y el feedback en tiempo real.
            </p>
          </div>

          <div className="flex flex-col gap-8 sm:flex-row sm:items-center">
            {sucursales.length > 1 && (
              <div>
                <label className="sr-only" htmlFor="filtro-sucursal-resenas">
                  Sucursal
                </label>
                <select
                  id="filtro-sucursal-resenas"
                  value={sucursalId}
                  onChange={(e) => setSucursalId(e.target.value)}
                  className="min-h-40 rounded-lg border border-concrete bg-canvas-white px-12 py-8 text-13 font-medium text-ash-graphite shadow-2xs outline-none focus:border-plain-green"
                >
                  <option value="">Todas las sucursales</option>
                  {sucursales.map((s) => (
                    <option key={s.id} value={s.id}>
                      {s.nombre}
                    </option>
                  ))}
                </select>
              </div>
            )}

            <button
              type="button"
              onClick={() => void cargarDatos()}
              disabled={cargando}
              className="inline-flex min-h-40 items-center justify-center gap-6 rounded-lg border border-concrete bg-canvas-white px-14 py-8 text-13 font-medium text-ash-graphite shadow-2xs hover:bg-ghost-fog active:scale-98 transition-all disabled:opacity-50"
            >
              <span className={`material-symbols-outlined text-18 ${cargando ? "animate-spin" : ""}`}>
                refresh
              </span>
              <span>Actualizar</span>
            </button>
          </div>
        </header>

        {/* Alertas destacadas en tiempo real para 1-3★ */}
        {alertasCriticas.length > 0 && (
          <div className="space-y-10">
            {alertasCriticas.map((alerta) => (
              <div
                key={alerta.id}
                className="rounded-xl border-2 border-rose-300 bg-rose-50/90 p-16 shadow-xs flex flex-col sm:flex-row sm:items-center justify-between gap-12 animate-in slide-in-from-top-2 duration-300"
              >
                <div className="flex items-start gap-12">
                  <div className="flex h-40 w-40 shrink-0 items-center justify-center rounded-full bg-rose-200 text-rose-800 font-bold text-20">
                    ⚠️
                  </div>
                  <div>
                    <div className="flex items-center gap-8 flex-wrap">
                      <span className="font-bold text-14 text-rose-900">
                        Atención inmediata — Mesa {alerta.mesaNumero || "?"}
                      </span>
                      <span className="rounded-full bg-rose-200/80 px-8 py-2 text-11 font-bold text-rose-800">
                        {alerta.estrellas} {alerta.estrellas === 1 ? "estrella" : "estrellas"}
                      </span>
                      <span className="text-11 text-rose-600">
                        {formatearTiempoRelativo(alerta.timestamp)}
                      </span>
                    </div>
                    {alerta.comentario ? (
                      <p className="mt-4 text-13 text-rose-800 italic leading-relaxed">
                        &ldquo;{alerta.comentario}&rdquo;
                      </p>
                    ) : (
                      <p className="mt-2 text-12 text-rose-700">
                        Calificación baja recibida sin comentario adicional.
                      </p>
                    )}
                  </div>
                </div>

                <div className="flex items-center justify-end">
                  <button
                    type="button"
                    onClick={() => descartarAlerta(alerta.id)}
                    className="rounded-lg bg-rose-600 hover:bg-rose-700 text-white px-12 py-6 text-12 font-semibold shadow-2xs transition-colors shrink-0"
                  >
                    Marcar como asistido
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}

        {/* Contenido principal */}
        {cargando && !resumen ? (
          <ReputacionSkeleton />
        ) : error ? (
          <div className="rounded-xl border border-concrete bg-canvas-white p-32 text-center">
            <EmptyState
              icon="error"
              title="Error al cargar reputación"
              description={error}
              actionLabel="Reintentar"
              onAction={() => void cargarDatos()}
            />
          </div>
        ) : totalResenas === 0 ? (
          <div className="rounded-xl border border-concrete bg-canvas-white p-32 text-center">
            <EmptyState
              icon="star"
              title="Sin reseñas todavía"
              description="A medida que los comensales finalicen sus pedidos y califiquen su experiencia en el local, verás aquí su satisfacción, métricas CSAT y comentarios."
            />
          </div>
        ) : (
          <div className="space-y-20">
            {/* Tarjetas KPI Superiores */}
            <div className="grid gap-16 md:grid-cols-3">
              {/* Tarjeta CSAT */}
              <article className="rounded-xl border border-concrete bg-canvas-white p-20 shadow-2xs">
                <div className="flex items-center justify-between">
                  <span className="text-10 font-mono font-semibold uppercase tracking-[0.08em] text-sage-green">
                    Índice de Satisfacción
                  </span>
                  <span className="rounded-full bg-emerald-100 px-8 py-2 text-11 font-bold text-emerald-800">
                    CSAT
                  </span>
                </div>
                <p className="mt-10 text-32 font-bold tracking-tight text-ash-graphite">
                  {csat}%
                </p>
                <p className="mt-4 text-12 text-sage-green">
                  Porcentaje de clientes con calificación de 4 o 5 estrellas.
                </p>
              </article>

              {/* Tarjeta Promedio */}
              <article className="rounded-xl border border-concrete bg-canvas-white p-20 shadow-2xs">
                <div className="flex items-center justify-between">
                  <span className="text-10 font-mono font-semibold uppercase tracking-[0.08em] text-sage-green">
                    Promedio Global
                  </span>
                  <span className="flex items-center text-amber-500 font-bold text-14">
                    ★
                  </span>
                </div>
                <div className="mt-10 flex items-baseline gap-6">
                  <span className="text-32 font-bold tracking-tight text-ash-graphite">
                    {promedio.toFixed(1).replace(".", ",")}
                  </span>
                  <span className="text-16 font-semibold text-amber-500">
                    ★
                  </span>
                </div>
                <p className="mt-4 text-12 text-sage-green">
                  Basado en {totalResenas} {totalResenas === 1 ? "reseña registrada" : "reseñas registradas"}.
                </p>
              </article>

              {/* Tarjeta Google Funnel */}
              <article className="rounded-xl border border-concrete bg-canvas-white p-20 shadow-2xs">
                <div className="flex items-center justify-between">
                  <span className="text-10 font-mono font-semibold uppercase tracking-[0.08em] text-sage-green">
                    Conversión a Google
                  </span>
                  <span className="material-symbols-outlined text-18 text-[#1a73e8]">
                    open_in_new
                  </span>
                </div>
                <p className="mt-10 text-32 font-bold tracking-tight text-ash-graphite">
                  {clicsGoogle}
                </p>
                <p className="mt-4 text-12 text-sage-green">
                  Comensales que abrieron el enlace de Google Reviews.
                </p>
              </article>
            </div>

            {/* Distribución por Estrellas (Barras CSS puro) */}
            <section className="rounded-xl border border-concrete bg-canvas-white p-20 sm:p-24 shadow-2xs">
              <h2 className="text-14 font-semibold text-ash-graphite mb-16">
                Desglose por estrellas
              </h2>

              <div className="space-y-10 max-w-xl">
                {[5, 4, 3, 2, 1].map((est) => {
                  const cantidad = resumen?.distribucion[est] ?? 0;
                  const porcentaje = totalResenas > 0 ? Math.round((cantidad / totalResenas) * 100) : 0;
                  return (
                    <div key={est} className="flex items-center gap-12 text-13">
                      <div className="flex items-center justify-end gap-3 w-50 shrink-0 font-medium text-ash-graphite">
                        <span>{est}</span>
                        <span className="text-amber-400 font-semibold text-13">★</span>
                      </div>

                      <div className="flex-1 h-10 bg-ghost-fog rounded-full overflow-hidden">
                        <div
                          className={`h-full rounded-full transition-all duration-500 ${
                            est >= 4 ? "bg-plain-green" : est === 3 ? "bg-amber-400" : "bg-rose-500"
                          }`}
                          style={{ width: `${porcentaje}%` }}
                        />
                      </div>

                      <div className="flex items-center justify-between w-70 shrink-0 text-12 font-mono text-sage-green">
                        <span className="font-semibold text-ash-graphite">{cantidad}</span>
                        <span className="text-10 text-sage-green">({porcentaje}%)</span>
                      </div>
                    </div>
                  );
                })}
              </div>
            </section>

            {/* Muro de Comentarios y Feedback */}
            <section className="rounded-xl border border-concrete bg-canvas-white p-20 sm:p-24 shadow-2xs space-y-16">
              <div className="flex items-center justify-between border-b border-ghost-fog pb-14">
                <h2 className="text-14 font-semibold text-ash-graphite">
                  Muro de comentarios y valoraciones
                </h2>
                <span className="text-12 text-sage-green">
                  {resumen?.resenas.length ?? 0} {resumen?.resenas.length === 1 ? "valoración" : "valoraciones"}
                </span>
              </div>

              {resumen?.resenas && resumen.resenas.length > 0 ? (
                <div className="space-y-12">
                  {resumen.resenas.map((r: ResenaItem) => {
                    const esPositiva = r.estrellas >= 4;
                    const esNeutral = r.estrellas === 3;

                    return (
                      <article
                        key={r.id}
                        className="rounded-xl border border-concrete bg-[#FAFBFB] p-16 transition-colors hover:bg-canvas-white space-y-8"
                      >
                        <div className="flex items-center justify-between flex-wrap gap-8">
                          <div className="flex items-center gap-8 flex-wrap">
                            {/* Píldora de estrellas */}
                            <span
                              className={`inline-flex items-center gap-2 rounded-full px-10 py-3 text-12 font-bold ${
                                esPositiva
                                  ? "bg-emerald-100 text-emerald-800"
                                  : esNeutral
                                    ? "bg-amber-100 text-amber-800"
                                    : "bg-rose-100 text-rose-800"
                              }`}
                            >
                              <span>{"★".repeat(r.estrellas)}</span>
                              <span className="text-11 opacity-75 font-mono">({r.estrellas}/5)</span>
                            </span>

                            <span className="text-13 font-semibold text-ash-graphite">
                              Mesa {r.mesa_numero || "?"}
                            </span>

                            {r.sucursal_nombre && (
                              <span className="rounded-md border border-concrete bg-canvas-white px-8 py-2 text-11 text-sage-green">
                                {r.sucursal_nombre}
                              </span>
                            )}

                            {r.google_cliqueado && (
                              <span className="inline-flex items-center gap-4 rounded-full bg-blue-50 border border-blue-200 px-8 py-2 text-10 font-semibold text-blue-700">
                                <span className="material-symbols-outlined text-12">verified</span>
                                Clic en Google
                              </span>
                            )}
                          </div>

                          <time className="text-11 text-sage-green font-mono">
                            {formatearTiempoRelativo(r.created_at)}
                          </time>
                        </div>

                        {r.comentario ? (
                          <p className="text-13 text-ash-graphite leading-relaxed pt-4">
                            &ldquo;{r.comentario}&rdquo;
                          </p>
                        ) : (
                          <p className="text-12 text-sage-green italic pt-2">
                            Sin comentario adicional
                          </p>
                        )}
                      </article>
                    );
                  })}
                </div>
              ) : (
                <p className="text-center text-13 text-sage-green py-20">
                  No hay comentarios registrados para los filtros seleccionados.
                </p>
              )}
            </section>
          </div>
        )}
      </div>
    </div>
  );
}

