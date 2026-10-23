"use client";

import React, { useEffect, useState, useMemo, useCallback } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import Logo from "@/components/brand/Logo";
import {
  api,
  ApiError,
  cerrarSesion,
  SesionUsuario,
  StatusPlataforma,
  TenantResumenPlatform,
  TenantDetalleActividadPlatform,
} from "@/lib/api";

function formatearUptime(segundos: number): string {
  if (segundos < 60) return `${segundos}s`;
  const minutos = Math.floor(segundos / 60);
  if (minutos < 60) return `${minutos}m`;
  const horas = Math.floor(minutos / 60);
  const minRestantes = minutos % 60;
  if (horas < 24) return `${horas}h ${minRestantes}m`;
  const dias = Math.floor(horas / 24);
  const horasRestantes = horas % 24;
  return `${dias}d ${horasRestantes}h`;
}

function formatearFecha(isoStr?: string): string {
  if (!isoStr) return "Sin actividad";
  const d = new Date(isoStr);
  return d.toLocaleString("es-AR", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function formatearDinero(monto: number): string {
  return new Intl.NumberFormat("es-AR", {
    style: "currency",
    currency: "ARS",
    maximumFractionDigits: 0,
  }).format(monto);
}

export default function SuperadminPage() {
  const router = useRouter();
  const [sesion, setSesion] = useState<SesionUsuario | null>(null);
  const [cargando, setCargando] = useState(true);
  const [accesoDenegado, setAccesoDenegado] = useState(false);

  const [status, setStatus] = useState<StatusPlataforma | null>(null);
  const [tenants, setTenants] = useState<TenantResumenPlatform[]>([]);
  const [errorCarga, setErrorCarga] = useState<string | null>(null);
  const [actualizando, setActualizando] = useState(false);
  const [intervaloAutoRefresh, setIntervaloAutoRefresh] = useState<number>(30); // segundos (0 = off)

  // Filtros
  const [busqueda, setBusqueda] = useState("");
  const [filtroPlan, setFiltroPlan] = useState<"todos" | "pro" | "free" | "trial">("todos");

  // Detalle / Auditoría
  const [tenantSeleccionadoId, setTenantSeleccionadoId] = useState<string | null>(null);
  const [detalleActividad, setDetalleActividad] = useState<TenantDetalleActividadPlatform | null>(null);
  const [cargandoDetalle, setCargandoDetalle] = useState(false);

  const cargarDatos = useCallback(async (silencioso = false) => {
    if (!silencioso) setActualizando(true);
    try {
      const [stData, tenantsData] = await Promise.all([
        api.obtenerStatusPlataforma(),
        api.listarTenantsPlataforma(),
      ]);
      setStatus(stData);
      setTenants(tenantsData);
      setErrorCarga(null);
    } catch (err: unknown) {
      if (err instanceof ApiError && err.status === 403) {
        setAccesoDenegado(true);
      } else {
        setErrorCarga(err instanceof Error ? err.message : "Error al cargar la plataforma");
      }
    } finally {
      setActualizando(false);
      setCargando(false);
    }
  }, []);

  // 1. Cargar Sesión y verificar permisos
  useEffect(() => {
    async function init() {
      try {
        const s = await api.obtenerSesion();
        setSesion(s);
        await cargarDatos();
      } catch (err: unknown) {
        if (err instanceof ApiError && err.status === 401) {
          router.replace("/login?redirect=/superadmin");
        } else {
          setCargando(false);
          setAccesoDenegado(true);
        }
      }
    }
    init();
  }, [cargarDatos, router]);

  // 2. Auto refresh
  useEffect(() => {
    if (intervaloAutoRefresh <= 0 || accesoDenegado) return;
    const interval = setInterval(() => {
      cargarDatos(true);
    }, intervaloAutoRefresh * 1000);
    return () => clearInterval(interval);
  }, [cargarDatos, intervaloAutoRefresh, accesoDenegado]);

  // 3. Cargar detalle al seleccionar un tenant
  useEffect(() => {
    if (!tenantSeleccionadoId) {
      setDetalleActividad(null);
      return;
    }
    let cancelado = false;
    setCargandoDetalle(true);
    api
      .obtenerDetalleActividadTenant(tenantSeleccionadoId)
      .then((data) => {
        if (!cancelado) setDetalleActividad(data);
      })
      .catch((err) => {
        console.error("Error al obtener detalle del tenant", err);
      })
      .finally(() => {
        if (!cancelado) setCargandoDetalle(false);
      });
    return () => {
      cancelado = true;
    };
  }, [tenantSeleccionadoId]);

  // Filtrado de tenants
  const tenantsFiltrados = useMemo(() => {
    return tenants.filter((t) => {
      const q = busqueda.toLowerCase().trim();
      const coincideBusqueda =
        !q ||
        t.nombre.toLowerCase().includes(q) ||
        t.slug.toLowerCase().includes(q) ||
        (t.nombre_fantasia && t.nombre_fantasia.toLowerCase().includes(q));

      if (!coincideBusqueda) return false;

      if (filtroPlan === "todos") return true;
      if (filtroPlan === "trial") return !!t.trial_hasta && new Date(t.trial_hasta) > new Date();
      return t.plan.toLowerCase() === filtroPlan;
    });
  }, [tenants, busqueda, filtroPlan]);

  if (cargando) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center p-6 bg-neutral-950">
        <div className="h-10 w-10 animate-spin rounded-full border-2 border-neutral-700 border-t-white" />
        <p className="mt-4 text-xs font-mono tracking-wider text-neutral-400 uppercase">
          Verificando credenciales de plataforma...
        </p>
      </div>
    );
  }

  if (accesoDenegado) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center p-6 bg-neutral-950 text-neutral-200">
        <div className="max-w-md w-full rounded-2xl border border-neutral-800 bg-neutral-900 p-8 text-center shadow-2xl">
          <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-full bg-red-950/60 text-red-400 border border-red-800">
            <span className="material-symbols-outlined text-3xl">lock</span>
          </div>
          <h1 className="mt-5 text-xl font-bold tracking-tight text-white">Acceso Restringido</h1>
          <p className="mt-3 text-sm text-neutral-400 leading-relaxed">
            Esta sección es de uso exclusivo para los fundadores y administradores de la plataforma Mesa CLICK.
            Tu cuenta actual (<strong>{sesion?.email || "desconocida"}</strong>) no está en la nómina de superadministradores.
          </p>
          <div className="mt-6 flex flex-col gap-3">
            <Link
              href="/dashboard"
              className="inline-flex w-full items-center justify-center rounded-xl bg-white px-4 py-2.5 text-sm font-semibold text-black transition hover:bg-neutral-200"
            >
              Ir a mi panel de comercio
            </Link>
            <button
              onClick={() => cerrarSesion()}
              className="inline-flex w-full items-center justify-center rounded-xl border border-neutral-800 bg-neutral-950 px-4 py-2 text-sm text-neutral-400 hover:text-white hover:border-neutral-700"
            >
              Cerrar sesión
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-neutral-950 text-neutral-100">
      {/* 1. Header Superior */}
      <header className="sticky top-0 z-40 border-b border-neutral-800/80 bg-neutral-950/90 backdrop-blur-md">
        <div className="mx-auto flex max-w-7xl items-center justify-between px-4 py-3 sm:px-6">
          <div className="flex items-center gap-3">
            <Logo mark="full" ink="#ffffff" className="h-7 w-auto text-white" />
            <span className="hidden sm:inline-block rounded-md bg-neutral-800 px-2 py-0.5 text-[11px] font-mono font-medium uppercase tracking-wider text-neutral-300 border border-neutral-700">
              Control Tower
            </span>
          </div>

          <div className="flex items-center gap-3">
            {/* Auto refresh dropdown */}
            <div className="flex items-center gap-2 rounded-lg border border-neutral-800 bg-neutral-900 px-2.5 py-1 text-xs">
              <span className="h-2 w-2 rounded-full bg-emerald-500 animate-pulse" />
              <span className="text-neutral-400 hidden md:inline">Auto-refresco:</span>
              <select
                value={intervaloAutoRefresh}
                onChange={(e) => setIntervaloAutoRefresh(Number(e.target.value))}
                className="bg-transparent font-mono text-neutral-200 outline-none cursor-pointer"
              >
                <option value={0} className="bg-neutral-900">Pausado</option>
                <option value={15} className="bg-neutral-900">15s</option>
                <option value={30} className="bg-neutral-900">30s</option>
                <option value={60} className="bg-neutral-900">60s</option>
              </select>
            </div>

            {/* Botón Refrescar */}
            <button
              onClick={() => cargarDatos()}
              disabled={actualizando}
              className="flex h-8 w-8 items-center justify-center rounded-lg border border-neutral-800 bg-neutral-900 text-neutral-300 hover:text-white hover:border-neutral-700 disabled:opacity-50"
              title="Refrescar ahora"
            >
              <span className={`material-symbols-outlined text-base ${actualizando ? "animate-spin" : ""}`}>
                sync
              </span>
            </button>

            {/* Perfil */}
            <div className="flex items-center gap-2 border-l border-neutral-800 pl-3">
              <div className="hidden sm:block text-right text-xs">
                <div className="font-medium text-white">{sesion?.nombre}</div>
                <div className="font-mono text-[10px] text-neutral-400">{sesion?.email}</div>
              </div>
              <Link
                href="/dashboard"
                className="rounded-lg border border-neutral-800 bg-neutral-900 px-3 py-1.5 text-xs font-medium text-neutral-300 hover:text-white hover:border-neutral-700"
              >
                Mi Local
              </Link>
            </div>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-7xl px-4 py-8 sm:px-6 space-y-8">
        {errorCarga && (
          <div className="rounded-xl border border-red-800/80 bg-red-950/40 p-4 text-sm text-red-300 flex items-center justify-between">
            <div className="flex items-center gap-2">
              <span className="material-symbols-outlined text-red-400">warning</span>
              <span>{errorCarga}</span>
            </div>
            <button onClick={() => cargarDatos()} className="underline font-semibold hover:text-white">
              Reintentar
            </button>
          </div>
        )}

        {/* 2. Tarjetas de KPIs Globales */}
        {status && (
          <section className="grid grid-cols-2 gap-4 lg:grid-cols-5">
            {/* Salud del Servicio */}
            <div className="rounded-2xl border border-neutral-800/80 bg-neutral-900/60 p-5 shadow-sm">
              <div className="flex items-center justify-between text-neutral-400">
                <span className="text-xs font-medium uppercase tracking-wider">Infraestructura</span>
                <span className="flex h-2 w-2 rounded-full bg-emerald-500" />
              </div>
              <div className="mt-3 text-2xl font-bold tracking-tight text-white capitalize">
                {status.estado}
              </div>
              <div className="mt-2 text-xs text-neutral-400 space-y-0.5">
                <div>Uptime: <span className="font-mono text-neutral-200">{formatearUptime(status.uptime_segundos)}</span></div>
                <div>DB: <span className="font-mono text-neutral-200">{status.database}</span></div>
              </div>
            </div>

            {/* Total Comercios */}
            <div className="rounded-2xl border border-neutral-800/80 bg-neutral-900/60 p-5 shadow-sm">
              <div className="flex items-center justify-between text-neutral-400">
                <span className="text-xs font-medium uppercase tracking-wider">Comercios</span>
                <span className="material-symbols-outlined text-base text-neutral-400">storefront</span>
              </div>
              <div className="mt-3 text-2xl font-bold tracking-tight text-white">
                {status.total_tenants}
              </div>
              <div className="mt-2 text-xs text-neutral-400">
                <span>{status.total_sucursales} sucursales totales</span>
              </div>
            </div>

            {/* Mesas Operativas */}
            <div className="rounded-2xl border border-neutral-800/80 bg-neutral-900/60 p-5 shadow-sm">
              <div className="flex items-center justify-between text-neutral-400">
                <span className="text-xs font-medium uppercase tracking-wider">Mesas en Salón</span>
                <span className="material-symbols-outlined text-base text-neutral-400">table_bar</span>
              </div>
              <div className="mt-3 text-2xl font-bold tracking-tight text-white">
                {status.total_mesas}
              </div>
              <div className="mt-2 text-xs text-neutral-400">
                <span>{status.total_usuarios} integrantes de staff</span>
              </div>
            </div>

            {/* Pedidos del Día */}
            <div className="rounded-2xl border border-neutral-800/80 bg-neutral-900/60 p-5 shadow-sm">
              <div className="flex items-center justify-between text-neutral-400">
                <span className="text-xs font-medium uppercase tracking-wider">Operación Hoy</span>
                <span className="material-symbols-outlined text-base text-neutral-400">receipt_long</span>
              </div>
              <div className="mt-3 text-2xl font-bold tracking-tight text-white">
                {status.pedidos_hoy} <small className="text-sm font-normal text-neutral-400">pedidos</small>
              </div>
              <div className="mt-2 text-xs text-neutral-400 font-mono">
                {formatearDinero(status.total_ventas_hoy)}
              </div>
            </div>

            {/* Presencia en Tiempo Real */}
            <div className="rounded-2xl border border-neutral-800/80 bg-neutral-900/60 p-5 shadow-sm col-span-2 lg:col-span-1">
              <div className="flex items-center justify-between text-neutral-400">
                <span className="text-xs font-medium uppercase tracking-wider">En Línea Ahora</span>
                <span className="flex h-2.5 w-2.5 rounded-full bg-emerald-400 animate-ping" />
              </div>
              <div className="mt-3 text-2xl font-bold tracking-tight text-emerald-400">
                {status.usuarios_online} <small className="text-sm font-normal text-neutral-400">usuarios</small>
              </div>
              <div className="mt-2 text-xs text-neutral-400 space-y-0.5">
                <div>SSE: <span className="font-mono text-neutral-200">{status.sse_suscriptores} clientes</span></div>
                <div>RAM: <span className="font-mono text-neutral-200">{status.heap_alloc_mb.toFixed(1)} MB</span></div>
              </div>
            </div>
          </section>
        )}

        {/* 3. Directorio de Clientes / Tenants */}
        <section className="rounded-2xl border border-neutral-800/80 bg-neutral-900/40 p-6 shadow-sm space-y-6">
          <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h2 className="text-lg font-bold tracking-tight text-white">Directorio de Comercios</h2>
              <p className="text-xs text-neutral-400">Listado consolidado de clientes dados de alta y actividad en vivo</p>
            </div>

            {/* Buscador y Filtros */}
            <div className="flex flex-wrap items-center gap-3">
              <div className="relative min-w-[220px]">
                <span className="material-symbols-outlined absolute left-3 top-2.5 text-neutral-500 text-sm">
                  search
                </span>
                <input
                  type="text"
                  placeholder="Buscar comercio o slug..."
                  value={busqueda}
                  onChange={(e) => setBusqueda(e.target.value)}
                  className="w-full rounded-xl border border-neutral-800 bg-neutral-900/90 pl-9 pr-4 py-2 text-xs text-neutral-100 placeholder:text-neutral-500 focus:border-white focus:outline-none transition"
                />
              </div>

              {/* Filtro Plan */}
              <div className="flex rounded-xl border border-neutral-800 bg-neutral-900 p-0.5 text-xs">
                {(["todos", "pro", "free", "trial"] as const).map((p) => (
                  <button
                    key={p}
                    onClick={() => setFiltroPlan(p)}
                    className={`rounded-lg px-2.5 py-1 font-medium capitalize transition ${
                      filtroPlan === p ? "bg-white text-black" : "text-neutral-400 hover:text-white"
                    }`}
                  >
                    {p}
                  </button>
                ))}
              </div>
            </div>
          </div>

          {/* Tabla de Comercios */}
          <div className="overflow-x-auto rounded-xl border border-neutral-800/80">
            <table className="w-full text-left text-xs">
              <thead className="border-b border-neutral-800 bg-neutral-900/90 text-[11px] font-semibold text-neutral-400 uppercase tracking-wider">
                <tr>
                  <th className="px-4 py-3">Comercio</th>
                  <th className="px-4 py-3">Plan</th>
                  <th className="px-4 py-3 text-center">Sucursales</th>
                  <th className="px-4 py-3 text-center">Mesas (Activas)</th>
                  <th className="px-4 py-3 text-center">Equipo</th>
                  <th className="px-4 py-3 text-right">Pedidos</th>
                  <th className="px-4 py-3 text-right">Ventas Acumuladas</th>
                  <th className="px-4 py-3 text-right">Acción</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-neutral-800/60 font-mono">
                {tenantsFiltrados.length === 0 ? (
                  <tr>
                    <td colSpan={8} className="px-4 py-8 text-center text-neutral-500 font-sans">
                      No se encontraron comercios con el criterio especificado.
                    </td>
                  </tr>
                ) : (
                  tenantsFiltrados.map((t) => {
                    const esTrial = t.trial_hasta && new Date(t.trial_hasta) > new Date();
                    return (
                      <tr
                        key={t.id}
                        className={`transition hover:bg-neutral-800/40 cursor-pointer ${
                          tenantSeleccionadoId === t.id ? "bg-neutral-800/60" : ""
                        }`}
                        onClick={() => setTenantSeleccionadoId(t.id)}
                      >
                        <td className="px-4 py-3 font-sans">
                          <div className="font-semibold text-white">{t.nombre}</div>
                          <div className="flex items-center gap-2 text-xs text-neutral-400 font-mono">
                            <span>/{t.slug}</span>
                            {t.rubro && <span className="text-[10px] text-neutral-500 font-sans">· {t.rubro}</span>}
                          </div>
                        </td>
                        <td className="px-4 py-3 font-sans">
                          <span
                            className={`inline-flex items-center rounded-md px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider ${
                              esTrial
                                ? "bg-amber-950/70 text-amber-400 border border-amber-800"
                                : t.plan === "pro"
                                ? "bg-emerald-950/70 text-emerald-400 border border-emerald-800"
                                : "bg-neutral-800 text-neutral-400 border border-neutral-700"
                            }`}
                          >
                            {esTrial ? "Trial Pro" : t.plan}
                          </span>
                        </td>
                        <td className="px-4 py-3 text-center text-neutral-200">
                          {t.total_sucursales}
                        </td>
                        <td className="px-4 py-3 text-center text-neutral-200">
                          <span className="text-emerald-400 font-semibold">{t.mesas_activas}</span>
                          <span className="text-neutral-500"> / {t.total_mesas}</span>
                        </td>
                        <td className="px-4 py-3 text-center font-sans">
                          <span className="rounded-full bg-neutral-800 px-2 py-0.5 text-xs text-neutral-300">
                            {t.total_usuarios} personas
                          </span>
                        </td>
                        <td className="px-4 py-3 text-right text-neutral-300 font-mono">
                          {t.total_pedidos.toLocaleString()}
                        </td>
                        <td className="px-4 py-3 text-right font-mono font-semibold text-white">
                          {formatearDinero(t.total_ventas)}
                        </td>
                        <td className="px-4 py-3 text-right font-sans">
                          <button
                            onClick={(e) => {
                              e.stopPropagation();
                              setTenantSeleccionadoId(t.id);
                            }}
                            className="inline-flex items-center gap-1 rounded-lg border border-neutral-700 bg-neutral-800 px-2.5 py-1 text-[11px] font-medium text-neutral-200 hover:bg-white hover:text-black hover:border-white transition"
                          >
                            <span>Auditar</span>
                            <span className="material-symbols-outlined text-xs">arrow_forward</span>
                          </button>
                        </td>
                      </tr>
                    );
                  })
                )}
              </tbody>
            </table>
          </div>
        </section>
      </main>

      {/* 4. Modal / Drawer Lateral de Auditoría de Tenant */}
      {tenantSeleccionadoId && (
        <div className="fixed inset-0 z-50 flex justify-end bg-black/70 backdrop-blur-sm transition-opacity">
          <div className="w-full max-w-2xl bg-neutral-900 border-l border-neutral-800 h-full overflow-y-auto p-6 sm:p-8 space-y-6 shadow-2xl flex flex-col justify-between">
            <div className="space-y-6">
              {/* Header Drawer */}
              <div className="flex items-start justify-between border-b border-neutral-800 pb-4">
                <div>
                  <div className="flex items-center gap-2">
                    <span className="text-xs uppercase font-mono tracking-wider text-neutral-400">
                      Detalle & Auditoría
                    </span>
                    {detalleActividad && (
                      <span className="rounded bg-neutral-800 px-1.5 py-0.5 text-[10px] font-mono uppercase text-neutral-300">
                        {detalleActividad.tenant.plan}
                      </span>
                    )}
                  </div>
                  <h3 className="mt-1 text-2xl font-bold tracking-tight text-white">
                    {detalleActividad ? detalleActividad.tenant.nombre : "Cargando comercio..."}
                  </h3>
                  {detalleActividad && (
                    <div className="flex items-center gap-2 text-xs text-neutral-400 font-mono mt-1">
                      <a
                        href={`/${detalleActividad.tenant.slug}`}
                        target="_blank"
                        rel="noreferrer"
                        className="text-neutral-300 hover:text-white underline flex items-center gap-1"
                      >
                        /{detalleActividad.tenant.slug}
                        <span className="material-symbols-outlined text-[11px]">open_in_new</span>
                      </a>
                      <span>·</span>
                      <span>Alta: {formatearFecha(detalleActividad.tenant.created_at)}</span>
                    </div>
                  )}
                </div>

                <button
                  onClick={() => setTenantSeleccionadoId(null)}
                  className="rounded-lg p-1.5 text-neutral-400 hover:text-white hover:bg-neutral-800"
                >
                  <span className="material-symbols-outlined">close</span>
                </button>
              </div>

              {cargandoDetalle ? (
                <div className="flex flex-col items-center justify-center py-16">
                  <div className="h-8 w-8 animate-spin rounded-full border-2 border-neutral-700 border-t-white" />
                  <p className="mt-3 text-xs text-neutral-500 font-mono">Consultando telemetría del local...</p>
                </div>
              ) : detalleActividad ? (
                <div className="space-y-8">
                  {/* Presencia de Usuarios del Local */}
                  <div>
                    <div className="flex items-center justify-between mb-3">
                      <h4 className="text-sm font-semibold text-white tracking-wide flex items-center gap-2">
                        <span className="material-symbols-outlined text-base text-neutral-400">group</span>
                        Equipo & Presencia en Vivo ({detalleActividad.usuarios.length})
                      </h4>
                    </div>

                    <div className="rounded-xl border border-neutral-800 divide-y divide-neutral-800/80 bg-neutral-950/60 overflow-hidden">
                      {detalleActividad.usuarios.length === 0 ? (
                        <div className="p-4 text-xs text-neutral-500 text-center font-sans">
                          Sin integrantes registrados en este comercio.
                        </div>
                      ) : (
                        detalleActividad.usuarios.map((u) => {
                          const isOnline = u.estado_presencia === "online";
                          const isIdle = u.estado_presencia === "inactivo";
                          return (
                            <div key={u.id} className="p-3.5 flex items-center justify-between text-xs">
                              <div className="flex items-center gap-3">
                                {/* Semáforo de Conexión */}
                                <div className="relative flex items-center justify-center">
                                  <span
                                    className={`h-2.5 w-2.5 rounded-full ${
                                      isOnline
                                        ? "bg-emerald-500 ring-4 ring-emerald-500/20"
                                        : isIdle
                                        ? "bg-amber-400"
                                        : "bg-neutral-600"
                                    }`}
                                  />
                                </div>
                                <div>
                                  <div className="font-semibold text-white flex items-center gap-2 font-sans">
                                    <span>{u.nombre || "Sin nombre"}</span>
                                    <span className="rounded bg-neutral-800 px-1.5 py-0.2 text-[10px] font-mono text-neutral-400 uppercase">
                                      {u.rol}
                                    </span>
                                  </div>
                                  <div className="text-[11px] text-neutral-400 font-mono mt-0.5">
                                    {u.email || u.username || "sin contacto"}
                                    {u.sucursal_nombre && ` · Sucursal: ${u.sucursal_nombre}`}
                                  </div>
                                </div>
                              </div>

                              <div className="text-right font-mono text-[11px]">
                                <div
                                  className={`font-semibold capitalize ${
                                    isOnline
                                      ? "text-emerald-400"
                                      : isIdle
                                      ? "text-amber-400"
                                      : "text-neutral-500"
                                  }`}
                                >
                                  {isOnline ? "En línea" : isIdle ? "Inactivo" : "Desconectado"}
                                </div>
                                <div className="text-[10px] text-neutral-500">
                                  {u.ultimo_login_at
                                    ? `Último login: ${formatearFecha(u.ultimo_login_at)}`
                                    : "Nunca ingresó"}
                                </div>
                              </div>
                            </div>
                          );
                        })
                      )}
                    </div>
                  </div>

                  {/* Feed de Últimos Movimientos */}
                  <div>
                    <h4 className="text-sm font-semibold text-white tracking-wide mb-3 flex items-center gap-2">
                      <span className="material-symbols-outlined text-base text-neutral-400">history</span>
                      Últimos Movimientos & Actividad
                    </h4>

                    <div className="rounded-xl border border-neutral-800 divide-y divide-neutral-800/80 bg-neutral-950/60 overflow-hidden max-h-80 overflow-y-auto">
                      {detalleActividad.ultimos_movimientos.length === 0 ? (
                        <div className="p-4 text-xs text-neutral-500 text-center font-sans">
                          No hay movimientos registrados recientemente.
                        </div>
                      ) : (
                        detalleActividad.ultimos_movimientos.map((m) => (
                          <div key={m.id} className="p-3 text-xs flex items-center justify-between">
                            <div className="space-y-0.5">
                              <div className="text-neutral-200 font-medium font-sans">{m.descripcion}</div>
                              <div className="text-[11px] text-neutral-400">
                                Por: <span className="text-neutral-300 font-semibold">{m.actor}</span>
                              </div>
                            </div>
                            <div className="text-[11px] text-neutral-400 font-mono text-right shrink-0 ml-3">
                              {formatearFecha(m.fecha)}
                            </div>
                          </div>
                        ))
                      )}
                    </div>
                  </div>
                </div>
              ) : null}
            </div>

            <div className="pt-4 border-t border-neutral-800 flex justify-end">
              <button
                onClick={() => setTenantSeleccionadoId(null)}
                className="rounded-xl border border-neutral-800 bg-neutral-800 px-4 py-2 text-xs font-semibold text-white hover:bg-neutral-700"
              >
                Cerrar Auditoría
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
