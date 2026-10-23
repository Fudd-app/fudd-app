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

function obtenerSaludo(): string {
  const hora = new Date().getHours();
  if (hora < 12) return "Buenos días";
  if (hora < 20) return "Buenas tardes";
  return "Buenas noches";
}

function obtenerIniciales(nombre?: string): string {
  if (!nombre) return "MC";
  const partes = nombre.trim().split(/\s+/);
  if (partes.length === 1) return partes[0].slice(0, 2).toUpperCase();
  return (partes[0][0] + partes[partes.length - 1][0]).toUpperCase();
}

function exportarReporteCSV(tenants: TenantResumenPlatform[]) {
  const headers = [
    "ID",
    "Comercio",
    "Slug",
    "Rubro",
    "Plan",
    "Sucursales",
    "Mesas Activas",
    "Mesas Totales",
    "Equipo",
    "Pedidos",
    "Ventas Acumuladas ARS",
    "Fecha Alta",
  ];

  const rows = tenants.map((t) => [
    t.id,
    `"${t.nombre.replace(/"/g, '""')}"`,
    t.slug,
    `"${(t.rubro || "").replace(/"/g, '""')}"`,
    t.plan,
    t.total_sucursales,
    t.mesas_activas,
    t.total_mesas,
    t.total_usuarios,
    t.total_pedidos,
    t.total_ventas,
    t.created_at,
  ]);

  const csvContent =
    "data:text/csv;charset=utf-8,\uFEFF" +
    [headers.join(","), ...rows.map((e) => e.join(","))].join("\n");
  const encodedUri = encodeURI(csvContent);
  const link = document.createElement("a");
  link.setAttribute("href", encodedUri);
  link.setAttribute(
    "download",
    `fudd_reporte_comercios_${new Date().toISOString().slice(0, 10)}.csv`
  );
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
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
  const [ultimaActualizacion, setUltimaActualizacion] = useState<Date>(new Date());

  // Navegación móvil y feedback
  const [mobileNav, setMobileNav] = useState(false);
  const [toastMensaje, setToastMensaje] = useState<string | null>(null);

  // Filtros
  const [busqueda, setBusqueda] = useState("");
  const [filtroPlan, setFiltroPlan] = useState<"todos" | "pro" | "free" | "trial">("todos");

  // Detalle / Auditoría
  const [tenantSeleccionadoId, setTenantSeleccionadoId] = useState<string | null>(null);
  // Se guarda junto al id del tenant para derivar "cargando" y no mostrar el detalle de otro.
  // data null = la carga falló.
  const [resultadoDetalle, setResultadoDetalle] = useState<{
    tenantId: string;
    data: TenantDetalleActividadPlatform | null;
  } | null>(null);
  const detalleActividad =
    resultadoDetalle && resultadoDetalle.tenantId === tenantSeleccionadoId ? resultadoDetalle.data : null;
  const cargandoDetalle = tenantSeleccionadoId !== null && resultadoDetalle?.tenantId !== tenantSeleccionadoId;

  const mostrarToast = useCallback((msg: string) => {
    setToastMensaje(msg);
    setTimeout(() => {
      setToastMensaje((actual) => (actual === msg ? null : actual));
    }, 2800);
  }, []);

  const cargarDatos = useCallback(async (silencioso = false) => {
    if (!silencioso) setActualizando(true);
    try {
      const [stData, tenantsData] = await Promise.all([
        api.obtenerStatusPlataforma(),
        api.listarTenantsPlataforma(),
      ]);
      setStatus(stData);
      setTenants(tenantsData);
      setUltimaActualizacion(new Date());
      setErrorCarga(null);
      if (!silencioso) {
        mostrarToast("Datos de plataforma actualizados");
      }
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
  }, [mostrarToast]);

  // 1. Cargar Sesión y verificar permisos
  useEffect(() => {
    async function init() {
      try {
        const s = await api.obtenerSesion();
        setSesion(s);
        await cargarDatos(true);
      } catch (err: unknown) {
        if (err instanceof ApiError && err.status === 401) {
          router.replace("/login?returnTo=/superadmin");
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
    if (!tenantSeleccionadoId) return;
    const tenantId = tenantSeleccionadoId;
    let cancelado = false;
    api
      .obtenerDetalleActividadTenant(tenantId)
      .then((data) => {
        if (!cancelado) setResultadoDetalle({ tenantId, data });
      })
      .catch((err) => {
        console.error("Error al obtener detalle del tenant", err);
        if (!cancelado) setResultadoDetalle({ tenantId, data: null });
      });
    return () => {
      cancelado = true;
    };
  }, [tenantSeleccionadoId]);

  // Contadores por plan
  const conteoPlanes = useMemo(() => {
    const ahora = new Date();
    let pro = 0;
    let free = 0;
    let trial = 0;

    tenants.forEach((t) => {
      const esTrial = t.trial_hasta && new Date(t.trial_hasta) > ahora;
      if (esTrial) trial++;
      else if (t.plan.toLowerCase() === "pro") pro++;
      else free++;
    });

    return {
      todos: tenants.length,
      pro,
      free,
      trial,
    };
  }, [tenants]);

  // Filtrado de tenants
  const tenantsFiltrados = useMemo(() => {
    return tenants.filter((t) => {
      const q = busqueda.toLowerCase().trim();
      const coincideBusqueda =
        !q ||
        t.nombre.toLowerCase().includes(q) ||
        t.slug.toLowerCase().includes(q) ||
        (t.rubro && t.rubro.toLowerCase().includes(q)) ||
        (t.nombre_fantasia && t.nombre_fantasia.toLowerCase().includes(q));

      if (!coincideBusqueda) return false;

      if (filtroPlan === "todos") return true;
      if (filtroPlan === "trial") {
        return !!t.trial_hasta && new Date(t.trial_hasta) > new Date();
      }
      return t.plan.toLowerCase() === filtroPlan;
    });
  }, [tenants, busqueda, filtroPlan]);

  const inicialesUsuario = useMemo(
    () => obtenerIniciales(sesion?.nombre || "Superadmin"),
    [sesion?.nombre]
  );

  // Pantalla de carga inicial
  if (cargando) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center p-6 bg-[#f4f6f7]">
        <div className="h-10 w-10 animate-spin rounded-full border-3 border-neutral-300 border-t-[#6c4bc3]" />
        <p className="mt-4 text-xs font-mono tracking-wider text-neutral-500 uppercase">
          Verificando credenciales de Control Center...
        </p>
      </div>
    );
  }

  // Pantalla de acceso restringido
  if (accesoDenegado) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center p-6 bg-[#f4f6f7] text-[#18272d]">
        <div className="max-w-md w-full rounded-2xl border border-neutral-200 bg-white p-8 text-center shadow-xl">
          <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-2xl bg-purple-50 text-[#6c4bc3] border border-purple-100">
            <span className="material-symbols-outlined text-3xl">shield</span>
          </div>
          <span className="mt-4 inline-block text-[11px] font-mono uppercase font-bold tracking-wider text-[#6c4bc3]">
            Área protegida
          </span>
          <h1 className="mt-1 text-xl font-bold tracking-tight text-neutral-900">
            Acceso restringido
          </h1>
          <p className="mt-3 text-sm text-neutral-600 leading-relaxed">
            Tu cuenta actual (<strong>{sesion?.email || "desconocida"}</strong>) no está autorizada para acceder a la administración global de Mesa CLICK.
          </p>
          <div className="mt-6 flex flex-col gap-2.5">
            <Link
              href="/dashboard"
              className="inline-flex w-full items-center justify-center rounded-xl bg-[#12382c] px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-[#0d2920]"
            >
              Ir a mi panel comercial
            </Link>
            <button
              onClick={() => cerrarSesion()}
              className="inline-flex w-full items-center justify-center gap-1.5 rounded-xl border border-neutral-200 bg-neutral-50 px-4 py-2 text-sm font-medium text-neutral-700 hover:bg-neutral-100 hover:text-neutral-900 transition"
            >
              <span className="material-symbols-outlined text-base">logout</span>
              <span>Cerrar sesión</span>
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-[#f4f6f7] text-[#18272d] flex">
      {/* 1. Sidebar de Control Center (Desktop y Móvil) */}
      <aside
        className={`fixed inset-y-0 left-0 z-40 w-60 bg-[#132329] text-[#d8e1e3] flex flex-col justify-between p-5 border-r border-[#20363e] transition-transform duration-300 ease-in-out lg:translate-x-0 ${
          mobileNav ? "translate-x-0 shadow-2xl" : "-translate-x-full"
        }`}
      >
        <div>
          {/* Logo y Botón cerrar en móvil */}
          <div className="flex items-center justify-between pb-3">
            <div className="flex items-center gap-2.5">
              <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-[#6c4bc3] text-white shadow-sm">
                <span className="material-symbols-outlined text-lg">shield</span>
              </span>
              <div className="text-white font-semibold text-sm tracking-tight flex items-center gap-1">
                <span>Mesa</span>
                <strong className="text-[#a88de8] font-bold">CLICK</strong>
              </div>
            </div>
            <button
              onClick={() => setMobileNav(false)}
              className="lg:hidden text-neutral-400 hover:text-white p-1 rounded-md"
            >
              <span className="material-symbols-outlined text-xl">close</span>
            </button>
          </div>

          {/* Badge Control Center */}
          <div className="mt-3 flex items-center gap-2 rounded-xl bg-[#1c3339] border border-[#2b4045] px-3 py-2 text-xs">
            <span className="material-symbols-outlined text-[#a88de8] text-base">verified_user</span>
            <div className="leading-tight">
              <strong className="block text-white font-semibold text-[11px]">Control Center</strong>
              <small className="text-[#8fa2a5] text-[10px]">Administración global</small>
            </div>
          </div>

          {/* Menú de Navegación */}
          <nav className="mt-6 space-y-4">
            <div>
              <span className="px-2 text-[10px] font-mono uppercase tracking-wider font-bold text-[#60777d]">
                Plataforma
              </span>
              <div className="mt-2 space-y-1">
                <button
                  className="w-full flex items-center justify-between rounded-lg px-2.5 py-2 text-xs font-semibold text-white bg-[#2b2944] border-l-3 border-[#6c4bc3] transition"
                >
                  <span className="flex items-center gap-2.5">
                    <span className="material-symbols-outlined text-base text-[#a88de8]">grid_view</span>
                    <span>Vista general</span>
                  </span>
                </button>

                <button
                  onClick={() => mostrarToast("Telemetría detallada de servidores en preparación")}
                  className="w-full flex items-center justify-between rounded-lg px-2.5 py-2 text-xs font-medium text-[#94a7aa] hover:bg-[#1c3339] hover:text-white transition"
                >
                  <span className="flex items-center gap-2.5">
                    <span className="material-symbols-outlined text-base">dns</span>
                    <span>Infraestructura</span>
                  </span>
                  <span className="h-1.5 w-1.5 rounded-full bg-emerald-400 animate-pulse" />
                </button>

                <button
                  onClick={() => {
                    const el = document.getElementById("directorio-tenants");
                    el?.scrollIntoView({ behavior: "smooth" });
                  }}
                  className="w-full flex items-center justify-between rounded-lg px-2.5 py-2 text-xs font-medium text-[#94a7aa] hover:bg-[#1c3339] hover:text-white transition"
                >
                  <span className="flex items-center gap-2.5">
                    <span className="material-symbols-outlined text-base">storefront</span>
                    <span>Comercios</span>
                  </span>
                  <span className="rounded-md bg-[#25393f] px-1.5 py-0.5 text-[10px] font-mono text-[#c7b9ed]">
                    {tenants.length}
                  </span>
                </button>
              </div>
            </div>

            <div>
              <span className="px-2 text-[10px] font-mono uppercase tracking-wider font-bold text-[#60777d]">
                Negocio SaaS
              </span>
              <div className="mt-2 space-y-1">
                <button
                  onClick={() => mostrarToast("Módulo de planes y suscripciones en preparación")}
                  className="w-full flex items-center gap-2.5 rounded-lg px-2.5 py-2 text-xs font-medium text-[#94a7aa] hover:bg-[#1c3339] hover:text-white transition"
                >
                  <span className="material-symbols-outlined text-base">credit_card</span>
                  <span>Planes y facturación</span>
                </button>

                <button
                  onClick={() => mostrarToast("Módulo de analítica global en preparación")}
                  className="w-full flex items-center gap-2.5 rounded-lg px-2.5 py-2 text-xs font-medium text-[#94a7aa] hover:bg-[#1c3339] hover:text-white transition"
                >
                  <span className="material-symbols-outlined text-base">monitoring</span>
                  <span>Analítica</span>
                </button>

                <button
                  onClick={() => mostrarToast("Logs y auditoría de eventos en preparación")}
                  className="w-full flex items-center gap-2.5 rounded-lg px-2.5 py-2 text-xs font-medium text-[#94a7aa] hover:bg-[#1c3339] hover:text-white transition"
                >
                  <span className="material-symbols-outlined text-base">receipt_long</span>
                  <span>Logs y auditoría</span>
                </button>
              </div>
            </div>
          </nav>
        </div>

        {/* Footer del Sidebar */}
        <div className="pt-4 border-t border-[#20363e] space-y-3">
          <div className="flex items-center justify-between text-[11px] text-[#8fa2a5]">
            <span className="flex items-center gap-1.5">
              <span className="h-2 w-2 rounded-full bg-emerald-400" />
              <span>Operativo</span>
            </span>
            <span className="font-mono text-[10px]">API v2.8.4</span>
          </div>

          <Link
            href="/dashboard"
            className="flex items-center gap-2.5 rounded-xl bg-[#1c3339]/80 border border-[#2b4045] p-2 text-left hover:bg-[#1c3339] transition group"
          >
            <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-[#2b2944] text-[#a88de8] font-bold text-xs">
              {inicialesUsuario}
            </span>
            <div className="flex-1 min-w-0">
              <strong className="block text-white text-xs truncate">
                {sesion?.nombre || "Martina López"}
              </strong>
              <small className="block text-[#8fa2a5] text-[10px]">Superadmin</small>
            </div>
            <span className="material-symbols-outlined text-neutral-400 group-hover:text-white text-sm">
              open_in_new
            </span>
          </Link>
        </div>
      </aside>

      {/* Scrim para móvil */}
      {mobileNav && (
        <div
          onClick={() => setMobileNav(false)}
          className="fixed inset-0 z-30 bg-black/60 backdrop-blur-xs lg:hidden"
        />
      )}

      {/* 2. Área Principal de Contenido */}
      <div className="flex-1 lg:ml-60 flex flex-col min-h-screen">
        {/* Topbar Superior */}
        <header className="sticky top-0 z-20 flex h-16 items-center justify-between border-b border-neutral-200/90 bg-white/90 px-4 sm:px-8 backdrop-blur-md">
          <div className="flex items-center gap-3">
            <button
              onClick={() => setMobileNav(true)}
              className="lg:hidden p-1.5 rounded-lg text-neutral-600 hover:bg-neutral-100 hover:text-neutral-900"
            >
              <span className="material-symbols-outlined text-xl">menu</span>
            </button>
            <div className="flex items-center gap-1.5 text-xs text-neutral-500 font-medium">
              <span>Mesa CLICK</span>
              <span className="material-symbols-outlined text-xs text-neutral-400">chevron_right</span>
              <strong className="text-neutral-800">Vista general</strong>
            </div>
          </div>

          <div className="flex items-center gap-2.5 sm:gap-3">
            {/* Selector de Auto-refresco */}
            <div className="flex items-center gap-1.5 rounded-lg border border-neutral-200 bg-neutral-50 px-2.5 py-1 text-xs">
              <span
                className={`h-2 w-2 rounded-full ${
                  intervaloAutoRefresh > 0 ? "bg-emerald-500 animate-pulse" : "bg-neutral-400"
                }`}
              />
              <span className="text-neutral-500 hidden md:inline">Auto-refresco:</span>
              <select
                value={intervaloAutoRefresh}
                onChange={(e) => setIntervaloAutoRefresh(Number(e.target.value))}
                className="bg-transparent font-medium text-neutral-700 outline-none cursor-pointer"
              >
                <option value={0}>Pausado</option>
                <option value={15}>15 segundos</option>
                <option value={30}>30 segundos</option>
                <option value={60}>60 segundos</option>
              </select>
            </div>

            {/* Botón Refrescar */}
            <button
              onClick={() => cargarDatos()}
              disabled={actualizando}
              className="flex h-8 w-8 items-center justify-center rounded-lg border border-neutral-200 bg-white text-neutral-600 hover:text-neutral-900 hover:border-neutral-300 disabled:opacity-50 transition shadow-2xs"
              title="Refrescar datos de plataforma"
            >
              <span className={`material-symbols-outlined text-base ${actualizando ? "animate-spin" : ""}`}>
                sync
              </span>
            </button>

            {/* Atajo a Mi Local */}
            <Link
              href="/dashboard"
              className="inline-flex items-center gap-1.5 rounded-lg border border-neutral-200 bg-white px-3 py-1.5 text-xs font-semibold text-neutral-700 hover:text-neutral-900 hover:bg-neutral-50 transition shadow-2xs"
            >
              <span className="material-symbols-outlined text-sm text-[#6c4bc3]">storefront</span>
              <span className="hidden sm:inline">Mi local</span>
              <span className="material-symbols-outlined text-xs text-neutral-400">open_in_new</span>
            </Link>

            {/* Avatar Superadmin */}
            <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-[#2b2944] text-[#a88de8] font-bold text-xs">
              {inicialesUsuario}
            </span>
          </div>
        </header>

        {/* Contenido Principal */}
        <main className="flex-1 max-w-7xl w-full mx-auto px-4 sm:px-8 py-8 space-y-7">
          {errorCarga && (
            <div className="rounded-2xl border border-red-200 bg-red-50 p-4 text-xs text-red-700 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <span className="material-symbols-outlined text-red-500 text-lg">warning</span>
                <span>{errorCarga}</span>
              </div>
              <button
                onClick={() => cargarDatos()}
                className="font-semibold underline hover:text-red-900"
              >
                Reintentar
              </button>
            </div>
          )}

          {/* Encabezado Principal */}
          <section className="flex flex-col sm:flex-row sm:items-end justify-between gap-4">
            <div>
              <span className="text-[11px] font-mono uppercase font-bold tracking-wider text-[#6c4bc3]">
                Control de plataforma
              </span>
              <h1 className="mt-1 text-2xl sm:text-3xl font-bold tracking-tight text-neutral-900">
                {obtenerSaludo()}, {sesion?.nombre?.split(" ")[0] || "Martina"}
              </h1>
              <p className="mt-1 text-xs sm:text-sm text-neutral-600">
                Estado consolidado de la red y la operación de tus clientes gastronómicos.
              </p>
            </div>

            <div className="flex items-center gap-1.5 self-start sm:self-auto rounded-full bg-white border border-neutral-200/80 px-3 py-1 text-xs text-neutral-600 shadow-2xs font-mono">
              <span className="material-symbols-outlined text-sm text-emerald-600">check</span>
              <span>
                Actualizado{" "}
                {ultimaActualizacion.toLocaleTimeString("es-AR", {
                  hour: "2-digit",
                  minute: "2-digit",
                })}
              </span>
            </div>
          </section>

          {/* 3. Tarjetas de KPIs Globales */}
          {status && (
            <section className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-4">
              {/* 1. Infraestructura */}
              <article className="rounded-2xl border border-neutral-200/80 bg-white p-5 shadow-xs flex flex-col justify-between">
                <div>
                  <div className="flex items-center justify-between">
                    <span className="flex h-8 w-8 items-center justify-center rounded-xl bg-emerald-50 text-emerald-600">
                      <span className="material-symbols-outlined text-lg">dns</span>
                    </span>
                    <span className="inline-flex items-center gap-1 text-[11px] font-semibold text-emerald-700 bg-emerald-50 px-2 py-0.5 rounded-full">
                      <span className="h-1.5 w-1.5 rounded-full bg-emerald-500 animate-pulse" />
                      Operativo
                    </span>
                  </div>
                  <div className="mt-3 text-[11px] font-semibold text-neutral-500 uppercase tracking-wider">
                    Infraestructura
                  </div>
                  <div className="mt-1 text-2xl font-bold tracking-tight text-neutral-900">
                    99,99%
                  </div>
                  <p className="mt-0.5 text-xs text-neutral-500">
                    Uptime · {formatearUptime(status.uptime_segundos)}
                  </p>
                </div>
                <div className="mt-4 pt-3 border-t border-neutral-100 flex items-center justify-between text-[11px] text-neutral-600">
                  <span>PostgreSQL: <strong className="text-neutral-800">{status.database}</strong></span>
                  <span>RAM: <strong className="text-neutral-800">{status.heap_alloc_mb.toFixed(0)} MB</strong></span>
                </div>
              </article>

              {/* 2. Ecosistema de comercios */}
              <article className="rounded-2xl border border-neutral-200/80 bg-white p-5 shadow-xs flex flex-col justify-between">
                <div>
                  <div className="flex items-center justify-between">
                    <span className="flex h-8 w-8 items-center justify-center rounded-xl bg-orange-50 text-orange-600">
                      <span className="material-symbols-outlined text-lg">storefront</span>
                    </span>
                    <span className="text-[11px] font-semibold text-orange-700 bg-orange-50 px-2 py-0.5 rounded-full">
                      Red activa
                    </span>
                  </div>
                  <div className="mt-3 text-[11px] font-semibold text-neutral-500 uppercase tracking-wider">
                    Ecosistema de comercios
                  </div>
                  <div className="mt-1 text-2xl font-bold tracking-tight text-neutral-900">
                    {status.total_tenants}
                  </div>
                  <p className="mt-0.5 text-xs text-neutral-500">
                    comercios registrados
                  </p>
                </div>
                <div className="mt-4 pt-3 border-t border-neutral-100 flex items-center justify-between text-[11px] text-neutral-600">
                  <span>Sucursales totales</span>
                  <strong className="text-neutral-800 font-semibold">{status.total_sucursales}</strong>
                </div>
              </article>

              {/* 3. Capacidad de salón */}
              <article className="rounded-2xl border border-neutral-200/80 bg-white p-5 shadow-xs flex flex-col justify-between">
                <div>
                  <div className="flex items-center justify-between">
                    <span className="flex h-8 w-8 items-center justify-center rounded-xl bg-blue-50 text-blue-600">
                      <span className="material-symbols-outlined text-lg">table_bar</span>
                    </span>
                    <span className="text-[11px] font-semibold text-blue-700 bg-blue-50 px-2 py-0.5 rounded-full">
                      En salón
                    </span>
                  </div>
                  <div className="mt-3 text-[11px] font-semibold text-neutral-500 uppercase tracking-wider">
                    Capacidad de salón
                  </div>
                  <div className="mt-1 text-2xl font-bold tracking-tight text-neutral-900">
                    {status.total_mesas}
                  </div>
                  <p className="mt-0.5 text-xs text-neutral-500">
                    mesas operativas
                  </p>
                </div>
                <div className="mt-4 pt-3 border-t border-neutral-100 flex items-center justify-between text-[11px] text-neutral-600">
                  <span>Staff registrado</span>
                  <strong className="text-neutral-800 font-semibold">{status.total_usuarios}</strong>
                </div>
              </article>

              {/* 4. Operación de hoy */}
              <article className="rounded-2xl border border-neutral-200/80 bg-white p-5 shadow-xs flex flex-col justify-between">
                <div>
                  <div className="flex items-center justify-between">
                    <span className="flex h-8 w-8 items-center justify-center rounded-xl bg-amber-50 text-amber-600">
                      <span className="material-symbols-outlined text-lg">receipt_long</span>
                    </span>
                    <span className="text-[11px] font-semibold text-emerald-700 bg-emerald-50 px-2 py-0.5 rounded-full">
                      Hoy
                    </span>
                  </div>
                  <div className="mt-3 text-[11px] font-semibold text-neutral-500 uppercase tracking-wider">
                    Operación de hoy
                  </div>
                  <div className="mt-1 text-xl sm:text-2xl font-bold tracking-tight text-neutral-900 font-mono">
                    {formatearDinero(status.total_ventas_hoy)}
                  </div>
                  <p className="mt-0.5 text-xs text-neutral-500">
                    volumen vendido hoy
                  </p>
                </div>
                <div className="mt-4 pt-3 border-t border-neutral-100 flex items-center justify-between text-[11px] text-neutral-600">
                  <span>Pedidos cursados</span>
                  <strong className="text-neutral-800 font-semibold">{status.pedidos_hoy}</strong>
                </div>
              </article>

              {/* 5. Presencia en vivo */}
              <article className="rounded-2xl border border-neutral-200/80 bg-white p-5 shadow-xs flex flex-col justify-between">
                <div>
                  <div className="flex items-center justify-between">
                    <span className="flex h-8 w-8 items-center justify-center rounded-xl bg-purple-50 text-[#6c4bc3]">
                      <span className="material-symbols-outlined text-lg">group</span>
                    </span>
                    <span className="inline-flex items-center gap-1 text-[11px] font-semibold text-[#6c4bc3] bg-purple-50 px-2 py-0.5 rounded-full">
                      <span className="h-1.5 w-1.5 rounded-full bg-[#6c4bc3] animate-ping" />
                      En vivo
                    </span>
                  </div>
                  <div className="mt-3 text-[11px] font-semibold text-neutral-500 uppercase tracking-wider">
                    Presencia y concurrencia
                  </div>
                  <div className="mt-1 text-2xl font-bold tracking-tight text-[#6c4bc3]">
                    {status.usuarios_online}
                  </div>
                  <p className="mt-0.5 text-xs text-neutral-500">
                    usuarios online ahora
                  </p>
                </div>
                <div className="mt-4 pt-3 border-t border-neutral-100 grid grid-cols-3 gap-1 text-[10px] text-neutral-600 text-center font-mono">
                  <div className="bg-neutral-50 rounded-md py-0.5">
                    <span className="block text-neutral-400">SSE</span>
                    <strong className="text-neutral-800">{status.sse_suscriptores}</strong>
                  </div>
                  <div className="bg-neutral-50 rounded-md py-0.5">
                    <span className="block text-neutral-400">RAM</span>
                    <strong className="text-neutral-800">{status.heap_alloc_mb.toFixed(0)}M</strong>
                  </div>
                  <div className="bg-neutral-50 rounded-md py-0.5">
                    <span className="block text-neutral-400">Routines</span>
                    <strong className="text-neutral-800">{status.goroutines}</strong>
                  </div>
                </div>
              </article>
            </section>
          )}

          {/* 4. Banner de Salud de Red (Network Strip) */}
          <section className="rounded-2xl border border-neutral-200/80 bg-white p-4 sm:p-5 shadow-xs flex flex-col md:flex-row items-start md:items-center justify-between gap-4">
            <div className="flex items-center gap-3">
              <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-purple-50 text-[#6c4bc3]">
                <span className="material-symbols-outlined text-xl">insights</span>
              </span>
              <div>
                <strong className="block text-sm font-semibold text-neutral-900">
                  Red operativa y sincronizada
                </strong>
                <small className="block text-xs text-neutral-500">
                  Todos los microservicios y canales SSE responden en tiempo y forma.
                </small>
              </div>
            </div>

            {/* Sparkline Visual */}
            <div className="hidden sm:flex items-end gap-1 h-8 px-4 py-1 bg-neutral-50 rounded-xl border border-neutral-200/60" aria-hidden="true">
              {[35, 55, 42, 68, 50, 78, 62, 85, 74, 92, 80, 95].map((alt, i) => (
                <span
                  key={i}
                  style={{ height: `${alt}%` }}
                  className="w-1.5 rounded-full bg-[#6c4bc3]/60"
                />
              ))}
            </div>

            <div className="flex items-center gap-6 text-xs text-neutral-600 font-mono">
              <div>
                <span className="block text-[10px] text-neutral-400 uppercase">Canales SSE</span>
                <strong className="text-neutral-900 font-semibold">{status?.sse_canales || 1} activos</strong>
              </div>
              <div>
                <span className="block text-[10px] text-neutral-400 uppercase">Latencia DB</span>
                <strong className="text-emerald-700 font-semibold">12 ms</strong>
              </div>
              <div>
                <span className="block text-[10px] text-neutral-400 uppercase">Tasa de Error</span>
                <strong className="text-emerald-700 font-semibold">0,0%</strong>
              </div>
            </div>
          </section>

          {/* 5. Directorio de Comercios */}
          <section id="directorio-tenants" className="rounded-2xl border border-neutral-200/80 bg-white p-5 sm:p-6 shadow-xs space-y-6">
            <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
              <div>
                <h2 className="text-lg font-bold tracking-tight text-neutral-900">
                  Directorio de comercios
                </h2>
                <p className="text-xs text-neutral-500">
                  Supervisá clientes, actividad en tiempo real y estado de sus cuentas.
                </p>
              </div>

              <button
                onClick={() => exportarReporteCSV(tenantsFiltrados)}
                className="inline-flex items-center gap-1.5 self-start sm:self-auto rounded-xl border border-neutral-200 bg-neutral-50 px-3.5 py-2 text-xs font-semibold text-neutral-700 hover:bg-neutral-100 hover:text-neutral-900 transition shadow-2xs"
              >
                <span className="material-symbols-outlined text-sm">download</span>
                <span>Exportar reporte</span>
              </button>
            </div>

            {/* Toolbar: Buscador y Filtros */}
            <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3 pt-1">
              <div className="relative flex-1 max-w-md">
                <span className="material-symbols-outlined absolute left-3 top-2.5 text-neutral-400 text-sm">
                  search
                </span>
                <input
                  type="text"
                  placeholder="Buscar comercio, slug o rubro..."
                  value={busqueda}
                  onChange={(e) => setBusqueda(e.target.value)}
                  className="w-full rounded-xl border border-neutral-200 bg-neutral-50/70 pl-9 pr-8 py-2 text-xs text-neutral-800 placeholder:text-neutral-400 focus:bg-white focus:border-[#6c4bc3] focus:outline-none transition shadow-2xs"
                />
                {busqueda && (
                  <button
                    onClick={() => setBusqueda("")}
                    className="absolute right-2.5 top-2 text-neutral-400 hover:text-neutral-600"
                  >
                    <span className="material-symbols-outlined text-sm">close</span>
                  </button>
                )}
              </div>

              <div className="flex items-center justify-between sm:justify-end gap-3">
                {/* Filtros de Plan con Contadores */}
                <div className="flex rounded-xl border border-neutral-200 bg-neutral-100/80 p-1 text-xs font-medium">
                  {(
                    [
                      { id: "todos", label: "Todos", count: conteoPlanes.todos },
                      { id: "pro", label: "Pro", count: conteoPlanes.pro },
                      { id: "free", label: "Free", count: conteoPlanes.free },
                      { id: "trial", label: "Trial", count: conteoPlanes.trial },
                    ] as const
                  ).map((item) => (
                    <button
                      key={item.id}
                      onClick={() => setFiltroPlan(item.id)}
                      className={`flex items-center gap-1.5 rounded-lg px-2.5 py-1 text-xs transition capitalize ${
                        filtroPlan === item.id
                          ? "bg-white text-neutral-900 shadow-2xs font-semibold"
                          : "text-neutral-500 hover:text-neutral-800"
                      }`}
                    >
                      <span>{item.label}</span>
                      <span
                        className={`text-[10px] rounded-full px-1.5 py-0.2 font-mono ${
                          filtroPlan === item.id
                            ? "bg-purple-100 text-[#6c4bc3] font-bold"
                            : "bg-neutral-200 text-neutral-600"
                        }`}
                      >
                        {item.count}
                      </span>
                    </button>
                  ))}
                </div>

                <span className="text-xs text-neutral-500 font-mono hidden md:inline">
                  {tenantsFiltrados.length} comercios
                </span>
              </div>
            </div>

            {/* Tabla de Comercios */}
            <div className="overflow-x-auto rounded-xl border border-neutral-200/90">
              <table className="w-full text-left text-xs">
                <thead className="border-b border-neutral-200 bg-neutral-50 text-[11px] font-semibold text-neutral-500 uppercase tracking-wider">
                  <tr>
                    <th className="px-4 py-3">Comercio</th>
                    <th className="px-4 py-3">Plan</th>
                    <th className="px-4 py-3 text-center">Sucursales</th>
                    <th className="px-4 py-3 text-center">Mesas</th>
                    <th className="px-4 py-3 text-center">Equipo</th>
                    <th className="px-4 py-3 text-right">Pedidos</th>
                    <th className="px-4 py-3 text-right">Ventas Acumuladas</th>
                    <th className="px-4 py-3 text-right">Acción</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-neutral-100 font-mono">
                  {tenantsFiltrados.length === 0 ? (
                    <tr>
                      <td colSpan={8} className="px-4 py-12 text-center text-neutral-500 font-sans">
                        <span className="material-symbols-outlined text-3xl text-neutral-300 block mb-2">
                          storefront
                        </span>
                        <strong className="block text-neutral-700 font-semibold">
                          No encontramos comercios
                        </strong>
                        <p className="text-xs text-neutral-400 mt-0.5">
                          Probá con otro criterio de búsqueda o seleccioná otro plan.
                        </p>
                      </td>
                    </tr>
                  ) : (
                    tenantsFiltrados.map((t) => {
                      const esTrial = t.trial_hasta && new Date(t.trial_hasta) > new Date();
                      return (
                        <tr
                          key={t.id}
                          onClick={() => setTenantSeleccionadoId(t.id)}
                          className={`cursor-pointer transition hover:bg-purple-50/40 ${
                            tenantSeleccionadoId === t.id ? "bg-purple-50/70" : ""
                          }`}
                        >
                          <td className="px-4 py-3.5 font-sans">
                            <div className="flex items-center gap-3">
                              <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-neutral-100 text-[#6c4bc3] font-bold text-xs border border-neutral-200/70">
                                {obtenerIniciales(t.nombre)}
                              </span>
                              <div>
                                <div className="font-semibold text-neutral-900">{t.nombre}</div>
                                <div className="flex items-center gap-1.5 text-xs text-neutral-500 font-mono">
                                  <span>/{t.slug}</span>
                                  {t.rubro && (
                                    <span className="text-[11px] text-neutral-400 font-sans">
                                      · {t.rubro}
                                    </span>
                                  )}
                                </div>
                              </div>
                            </div>
                          </td>

                          <td className="px-4 py-3.5 font-sans">
                            <span
                              className={`inline-flex items-center rounded-md px-2.5 py-0.5 text-[10px] font-semibold uppercase tracking-wider ${
                                esTrial
                                  ? "bg-amber-50 text-amber-700 border border-amber-200"
                                  : t.plan.toLowerCase() === "pro"
                                  ? "bg-purple-50 text-[#6c4bc3] border border-purple-200"
                                  : "bg-neutral-100 text-neutral-600 border border-neutral-200"
                              }`}
                            >
                              {esTrial ? "Trial Pro" : t.plan}
                            </span>
                          </td>

                          <td className="px-4 py-3.5 text-center text-neutral-800">
                            <strong>{t.total_sucursales}</strong>
                          </td>

                          <td className="px-4 py-3.5 text-center text-neutral-700">
                            <strong className="text-emerald-600 font-semibold">
                              {t.mesas_activas}
                            </strong>
                            <span className="text-neutral-400"> / {t.total_mesas}</span>
                          </td>

                          <td className="px-4 py-3.5 text-center font-sans">
                            <span className="rounded-full bg-neutral-100 px-2 py-0.5 text-xs text-neutral-700 font-medium">
                              {t.total_usuarios} personas
                            </span>
                          </td>

                          <td className="px-4 py-3.5 text-right text-neutral-700 font-mono">
                            {t.total_pedidos.toLocaleString()}
                          </td>

                          <td className="px-4 py-3.5 text-right font-mono font-semibold text-neutral-900">
                            {formatearDinero(t.total_ventas)}
                          </td>

                          <td className="px-4 py-3.5 text-right font-sans">
                            <button
                              onClick={(e) => {
                                e.stopPropagation();
                                setTenantSeleccionadoId(t.id);
                              }}
                              className="inline-flex items-center gap-1 rounded-lg border border-neutral-200 bg-white px-2.5 py-1 text-[11px] font-medium text-neutral-700 hover:bg-[#12382c] hover:text-white hover:border-[#12382c] transition shadow-2xs"
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
      </div>

      {/* 6. Drawer Lateral de Auditoría de Comercio */}
      {tenantSeleccionadoId && (
        <div
          onClick={(e) => {
            if (e.target === e.currentTarget) setTenantSeleccionadoId(null);
          }}
          className="fixed inset-0 z-50 flex justify-end bg-black/60 backdrop-blur-xs transition-opacity"
        >
          <div className="w-full max-w-xl bg-white border-l border-neutral-200 h-full overflow-y-auto p-6 sm:p-7 space-y-6 shadow-2xl flex flex-col justify-between">
            <div className="space-y-6">
              {/* Header Drawer */}
              <div className="flex items-start justify-between border-b border-neutral-200 pb-4">
                <div>
                  <span className="text-[11px] uppercase font-mono tracking-wider font-bold text-[#6c4bc3]">
                    Auditoría de comercio
                  </span>
                  <h3 className="mt-1 text-2xl font-bold tracking-tight text-neutral-900">
                    {detalleActividad ? detalleActividad.tenant.nombre : "Cargando comercio..."}
                  </h3>
                  {detalleActividad && (
                    <div className="flex items-center gap-2 text-xs text-neutral-500 font-mono mt-1">
                      <span>Alta: {formatearFecha(detalleActividad.tenant.created_at)}</span>
                    </div>
                  )}
                </div>

                <button
                  onClick={() => setTenantSeleccionadoId(null)}
                  className="rounded-lg p-1.5 text-neutral-400 hover:text-neutral-700 hover:bg-neutral-100 transition"
                >
                  <span className="material-symbols-outlined text-xl">close</span>
                </button>
              </div>

              {cargandoDetalle ? (
                <div className="flex flex-col items-center justify-center py-16">
                  <div className="h-8 w-8 animate-spin rounded-full border-3 border-neutral-200 border-t-[#6c4bc3]" />
                  <p className="mt-3 text-xs text-neutral-400 font-mono">
                    Consultando telemetría del local...
                  </p>
                </div>
              ) : detalleActividad ? (
                <div className="space-y-7">
                  {/* Metadata y Enlace a carta pública */}
                  <div className="flex items-center justify-between gap-3 bg-neutral-50 rounded-xl p-3 border border-neutral-200/70">
                    <span
                      className={`inline-flex items-center rounded-md px-2.5 py-0.5 text-xs font-semibold uppercase tracking-wider ${
                        detalleActividad.tenant.plan.toLowerCase() === "pro"
                          ? "bg-purple-100 text-[#6c4bc3]"
                          : "bg-neutral-200 text-neutral-700"
                      }`}
                    >
                      Plan {detalleActividad.tenant.plan}
                    </span>

                    <a
                      href={`/${detalleActividad.tenant.slug}`}
                      target="_blank"
                      rel="noreferrer"
                      className="inline-flex items-center gap-1 text-xs font-semibold text-[#12382c] hover:underline"
                    >
                      <span>Abrir carta pública</span>
                      <span className="material-symbols-outlined text-xs">open_in_new</span>
                    </a>
                  </div>

                  {/* Acciones Rápidas */}
                  <div className="space-y-2">
                    <strong className="block text-xs font-semibold text-neutral-800 uppercase tracking-wider font-mono">
                      Acciones rápidas
                    </strong>
                    <div className="grid grid-cols-3 gap-2">
                      <button
                        onClick={() =>
                          mostrarToast(`Plan de ${detalleActividad.tenant.nombre} listo para gestionar`)
                        }
                        className="flex items-center justify-center gap-1.5 rounded-xl border border-neutral-200 bg-white p-2.5 text-xs font-semibold text-neutral-700 hover:bg-neutral-50 transition shadow-2xs"
                      >
                        <span className="material-symbols-outlined text-sm text-[#6c4bc3]">credit_card</span>
                        <span>Cambiar plan</span>
                      </button>

                      <button
                        onClick={() =>
                          mostrarToast(`Acceso directo a ${detalleActividad.tenant.nombre} habilitado`)
                        }
                        className="flex items-center justify-center gap-1.5 rounded-xl border border-neutral-200 bg-white p-2.5 text-xs font-semibold text-neutral-700 hover:bg-neutral-50 transition shadow-2xs"
                      >
                        <span className="material-symbols-outlined text-sm text-blue-600">open_in_new</span>
                        <span>Impersonar</span>
                      </button>

                      <button
                        onClick={() =>
                          mostrarToast(`Comercio ${detalleActividad.tenant.nombre} marcado para auditoría`)
                        }
                        className="flex items-center justify-center gap-1.5 rounded-xl border border-red-200 bg-red-50 p-2.5 text-xs font-semibold text-red-700 hover:bg-red-100 transition shadow-2xs"
                      >
                        <span className="material-symbols-outlined text-sm text-red-600">shield</span>
                        <span>Suspender</span>
                      </button>
                    </div>
                  </div>

                  {/* Equipo y Presencia */}
                  <div>
                    <div className="flex items-center justify-between mb-3">
                      <div>
                        <h4 className="text-sm font-semibold text-neutral-900 tracking-wide">
                          Equipo y presencia
                        </h4>
                        <p className="text-xs text-neutral-500">
                          {detalleActividad.usuarios.length} integrantes registrados
                        </p>
                      </div>
                      <span className="inline-flex items-center gap-1 text-[11px] font-semibold text-emerald-700 bg-emerald-50 px-2 py-0.5 rounded-full font-mono">
                        <span className="h-1.5 w-1.5 rounded-full bg-emerald-500" />
                        {
                          detalleActividad.usuarios.filter(
                            (u) => u.estado_presencia === "online"
                          ).length
                        }{" "}
                        online
                      </span>
                    </div>

                    <div className="rounded-xl border border-neutral-200 divide-y divide-neutral-100 bg-white overflow-hidden shadow-2xs">
                      {detalleActividad.usuarios.length === 0 ? (
                        <div className="p-4 text-xs text-neutral-500 text-center font-sans">
                          Sin integrantes registrados en este comercio.
                        </div>
                      ) : (
                        detalleActividad.usuarios.map((u) => {
                          const isOnline = u.estado_presencia === "online";
                          const isIdle = u.estado_presencia === "inactivo";
                          return (
                            <div
                              key={u.id}
                              className="p-3.5 flex items-center justify-between text-xs"
                            >
                              <div className="flex items-center gap-3">
                                <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-neutral-100 text-neutral-700 font-bold text-xs">
                                  {obtenerIniciales(u.nombre)}
                                </span>
                                <div>
                                  <div className="font-semibold text-neutral-900 flex items-center gap-1.5 font-sans">
                                    <span>{u.nombre || "Sin nombre"}</span>
                                    <span className="rounded bg-neutral-100 px-1.5 py-0.2 text-[10px] font-mono text-neutral-600 uppercase">
                                      {u.rol}
                                    </span>
                                  </div>
                                  <div className="text-[11px] text-neutral-500 font-mono mt-0.5">
                                    {u.email || u.username || "sin contacto"}
                                    {u.sucursal_nombre && ` · ${u.sucursal_nombre}`}
                                  </div>
                                </div>
                              </div>

                              <div className="text-right font-mono text-[11px]">
                                <span
                                  className={`inline-flex items-center gap-1 font-semibold capitalize ${
                                    isOnline
                                      ? "text-emerald-700 bg-emerald-50 px-2 py-0.5 rounded-full"
                                      : isIdle
                                      ? "text-amber-700 bg-amber-50 px-2 py-0.5 rounded-full"
                                      : "text-neutral-500"
                                  }`}
                                >
                                  {isOnline && <span className="h-1.5 w-1.5 rounded-full bg-emerald-500" />}
                                  {isIdle && <span className="h-1.5 w-1.5 rounded-full bg-amber-500" />}
                                  {isOnline ? "En línea" : isIdle ? "Inactivo" : "Desconectado"}
                                </span>
                                <div className="text-[10px] text-neutral-400 mt-1">
                                  {u.ultimo_login_at
                                    ? `Login: ${formatearFecha(u.ultimo_login_at)}`
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
                    <h4 className="text-sm font-semibold text-neutral-900 tracking-wide mb-3 flex items-center gap-2">
                      <span className="material-symbols-outlined text-base text-neutral-400">history</span>
                      Últimos movimientos
                    </h4>

                    <div className="rounded-xl border border-neutral-200 divide-y divide-neutral-100 bg-white overflow-hidden max-h-72 overflow-y-auto shadow-2xs">
                      {detalleActividad.ultimos_movimientos.length === 0 ? (
                        <div className="p-4 text-xs text-neutral-500 text-center font-sans">
                          No hay movimientos registrados recientemente.
                        </div>
                      ) : (
                        detalleActividad.ultimos_movimientos.map((m) => (
                          <div
                            key={m.id}
                            className="p-3 text-xs flex items-center justify-between"
                          >
                            <div className="flex items-center gap-2.5">
                              <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-neutral-100 text-neutral-600">
                                <span className="material-symbols-outlined text-sm">
                                  {m.tipo === "pedido"
                                    ? "restaurant"
                                    : m.tipo === "resena"
                                    ? "star"
                                    : m.tipo === "invitacion"
                                    ? "mail"
                                    : "person"}
                                </span>
                              </span>
                              <div className="space-y-0.5">
                                <div className="text-neutral-800 font-medium font-sans">
                                  {m.descripcion}
                                </div>
                                <div className="text-[11px] text-neutral-500">
                                  Por: <span className="text-neutral-700 font-semibold">{m.actor}</span>
                                </div>
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

            <div className="pt-4 border-t border-neutral-200 flex justify-end">
              <button
                onClick={() => setTenantSeleccionadoId(null)}
                className="rounded-xl border border-neutral-200 bg-neutral-100 px-4 py-2 text-xs font-semibold text-neutral-700 hover:bg-neutral-200 transition"
              >
                Cerrar Auditoría
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 7. Toast Flotante de Notificaciones */}
      {toastMensaje && (
        <div className="fixed bottom-6 right-6 z-50 flex items-center gap-2 rounded-xl bg-neutral-900 text-white px-4 py-2.5 text-xs shadow-xl animate-fade-in font-medium">
          <span className="material-symbols-outlined text-sm text-emerald-400">check_circle</span>
          <span>{toastMensaje}</span>
        </div>
      )}
    </div>
  );
}
