"use client";

import React, { useState, useEffect, useCallback } from "react";
import { api, Tenant, EstadoPlan, getErrorMessage } from "@/lib/api";
import { formatearPrecioPlan, PLAN_MONEDA, PLAN_PRUEBA_MESES, PLAN_PRO_PRECIO_MENSUAL_ARS } from "@/lib/planes";
import { Progress, Skeleton } from "@/components/ui";

interface Props {
  tenant: Tenant | null;
  onTenantUpdate?: (actualizado: Tenant) => void;
}

export default function PlanesSection({ tenant, onTenantUpdate }: Props) {
  const [estadoPlan, setEstadoPlan] = useState<EstadoPlan | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [solicitando, setSolicitando] = useState(false);
  const [solicitudEnviada, setSolicitudEnviada] = useState(false);
  const [nota, setNota] = useState("");

  const recargarPlan = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const datos = await api.obtenerMiPlan();
      setEstadoPlan(datos);
      if (datos.upgrade_solicitado_at) {
        setSolicitudEnviada(true);
      }
    } catch (err) {
      setError(getErrorMessage(err, "No pudimos cargar la información del plan."));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    let ignore = false;
    api.obtenerMiPlan()
      .then((datos) => {
        if (ignore) return;
        setEstadoPlan(datos);
        if (datos.upgrade_solicitado_at) {
          setSolicitudEnviada(true);
        }
        setLoading(false);
      })
      .catch((err) => {
        if (ignore) return;
        setError(getErrorMessage(err, "No pudimos cargar la información del plan."));
        setLoading(false);
      });
    return () => {
      ignore = true;
    };
  }, []);

  const esPro = estadoPlan?.plan === "pro";
  const enPrueba = esPro && Boolean(estadoPlan?.en_prueba);
  const mesesPrueba = estadoPlan?.meses_prueba ?? PLAN_PRUEBA_MESES;
  const precioMensual = estadoPlan?.precio_mensual ?? PLAN_PRO_PRECIO_MENSUAL_ARS;
  const moneda = estadoPlan?.moneda ?? PLAN_MONEDA;
  const precioPro = formatearPrecioPlan(precioMensual);

  const handleSolicitarUpgrade = async () => {
    if (solicitando || solicitudEnviada) return;
    setSolicitando(true);
    try {
      const tenantActualizado = await api.solicitarUpgradePro(nota.trim());
      setSolicitudEnviada(true);
      if (onTenantUpdate) {
        onTenantUpdate(tenantActualizado);
      }

      // Preparar link pre-cargado de contacto
      const nombreNegocio = tenant?.nombre || "Mi Negocio";
      const slugNegocio = tenant?.slug || "";
      const usoMesas = estadoPlan ? `${estadoPlan.uso.mesas || 0}/${estadoPlan.limites.mesas || 10}` : "";
      const usoProd = estadoPlan ? `${estadoPlan.uso.productos || 0}/${estadoPlan.limites.productos || 30}` : "";
      const usoSuc = estadoPlan ? `${estadoPlan.uso.sucursales || 0}/${estadoPlan.limites.sucursales || 1}` : "";

      const cuerpo = encodeURIComponent(
        `Hola, solicito el Upgrade a Plan Pro para el negocio "${nombreNegocio}" (slug: ${slugNegocio}).\n` +
        `Precio informado: ${precioPro} ${moneda} por mes.\n` +
        `Uso actual: ${usoMesas} mesas, ${usoProd} productos, ${usoSuc} sucursales.\n` +
        (nota.trim() ? `Nota: ${nota.trim()}` : "")
      );

      const mailtoUrl = `mailto:supportmesaclick@gmail.com?subject=${encodeURIComponent(`Solicitud Upgrade Pro - ${nombreNegocio}`)}&body=${cuerpo}`;
      if (typeof window !== "undefined") {
        window.open(mailtoUrl, "_blank");
      }
    } catch (err) {
      setError(getErrorMessage(err, "No pudimos registrar tu solicitud. Por favor contactanos directamente."));
    } finally {
      setSolicitando(false);
    }
  };

  const handleConsultarWhatsApp = () => {
    const nombreNegocio = tenant?.nombre || "Mi Negocio";
    const slugNegocio = tenant?.slug || "";
    const texto = encodeURIComponent(
      `Hola! Quiero solicitar el Upgrade a Plan Pro por ${precioPro} ${moneda} por mes para "${nombreNegocio}" (slug: ${slugNegocio}).`
    );
    if (typeof window !== "undefined") {
      window.open(`https://wa.me/5491100000000?text=${texto}`, "_blank");
    }
  };

  if (loading) {
    return (
      <div className="space-y-16">
        <Skeleton className="h-160 w-full rounded-xl" />
        <Skeleton className="h-320 w-full rounded-xl" />
      </div>
    );
  }

  if (error && !estadoPlan) {
    return (
      <div className="rounded-xl border border-concrete bg-canvas-white p-24 text-center">
        <p className="text-14 font-medium text-alert-red">{error}</p>
        <button
          onClick={recargarPlan}
          className="mt-12 h-44 rounded-lg bg-ash-graphite px-16 text-12 font-semibold text-canvas-white hover:bg-plain-green-muted transition-colors"
        >
          Reintentar
        </button>
      </div>
    );
  }

  const usoMesas = estadoPlan?.uso.mesas ?? 0;
  const limiteMesas = estadoPlan?.limites.mesas ?? 10;
  const usoProd = estadoPlan?.uso.productos ?? 0;
  const limiteProd = estadoPlan?.limites.productos ?? 30;
  const usoSuc = estadoPlan?.uso.sucursales ?? 0;
  const limiteSuc = estadoPlan?.limites.sucursales ?? 1;

  return (
    <div className="space-y-16 font-inter">
      {/* ── CARD ESTADO DEL PLAN ── */}
      <section className="rounded-xl border border-concrete bg-canvas-white p-20">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-12 border-b border-concrete pb-16">
          <div>
            <div className="flex items-center gap-8">
              <span className="text-11 font-bold uppercase tracking-wider text-sage-green">Estado del plan</span>
              <span
                className={`inline-flex items-center px-10 py-2 rounded-full text-11 font-mono font-semibold uppercase ${
                  esPro
                    ? "bg-ash-graphite text-canvas-white"
                    : "bg-ghost-fog border border-concrete text-ash-graphite"
                }`}
              >
                {enPrueba ? "Prueba Pro" : esPro ? "Plan Pro" : "Plan Free"}
              </span>
            </div>
            <p className="mt-4 text-13 text-sage-green">
              {enPrueba
                ? `Tenés todas las funciones Pro sin cargo durante ${mesesPrueba} meses. Después podés seguir con Free o contratar Pro.`
                : esPro
                  ? "Disfrutás de mesas, platos y sucursales ilimitadas con acceso a todas las herramientas avanzadas."
                : "Plan gratuito con límites de cuota operativos para locales iniciales."}
            </p>
          </div>

          {esPro && estadoPlan?.plan_hasta && (
            <div className="text-right text-12 text-sage-green">
              <p>
                {enPrueba ? "Prueba válida hasta:" : "Válido hasta:"}{" "}
                <span className="font-semibold text-ash-graphite">
                  {new Date(estadoPlan.plan_hasta).toLocaleDateString("es-AR", {
                    day: "2-digit",
                    month: "short",
                    year: "numeric",
                  })}
                </span>
              </p>
              {(enPrueba ? estadoPlan.dias_restantes_prueba : estadoPlan.dias_restantes_pro) != null && (
                <p className="text-11 text-stone mt-2">
                  {enPrueba ? estadoPlan.dias_restantes_prueba : estadoPlan.dias_restantes_pro} días restantes
                </p>
              )}
            </div>
          )}
        </div>

        {/* BARRAS DE PROGRESO DE CUOTAS */}
        <div className="mt-20 space-y-16">
          <h3 className="text-13 font-semibold text-ash-graphite">Consumo de cuotas</h3>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-16">
            {/* Mesas */}
            <div className="rounded-lg border border-concrete bg-ghost-fog/40 p-16">
              <p className="text-12 font-medium text-sage-green mb-8">Mesas activas</p>
              {esPro ? (
                <p className="text-18 font-mono font-bold text-ash-graphite">
                  {usoMesas}{" "}
                  <span className="text-12 font-normal text-sage-green font-sans">(Ilimitado)</span>
                </p>
              ) : (
                <Progress
                  valor={usoMesas}
                  max={limiteMesas}
                  etiqueta={`${usoMesas}/${limiteMesas} mesas usadas`}
                  tono={usoMesas >= limiteMesas ? "warning" : "default"}
                />
              )}
            </div>

            {/* Productos */}
            <div className="rounded-lg border border-concrete bg-ghost-fog/40 p-16">
              <p className="text-12 font-medium text-sage-green mb-8">Productos en carta</p>
              {esPro ? (
                <p className="text-18 font-mono font-bold text-ash-graphite">
                  {usoProd}{" "}
                  <span className="text-12 font-normal text-sage-green font-sans">(Ilimitado)</span>
                </p>
              ) : (
                <Progress
                  valor={usoProd}
                  max={limiteProd}
                  etiqueta={`${usoProd}/${limiteProd} platos usados`}
                  tono={usoProd >= limiteProd ? "warning" : "default"}
                />
              )}
            </div>

            {/* Sucursales */}
            <div className="rounded-lg border border-concrete bg-ghost-fog/40 p-16">
              <p className="text-12 font-medium text-sage-green mb-8">Sucursales físicas</p>
              {esPro ? (
                <p className="text-18 font-mono font-bold text-ash-graphite">
                  {usoSuc}{" "}
                  <span className="text-12 font-normal text-sage-green font-sans">(Ilimitado)</span>
                </p>
              ) : (
                <Progress
                  valor={usoSuc}
                  max={limiteSuc}
                  etiqueta={`${usoSuc}/${limiteSuc} sucursales usadas`}
                  tono={usoSuc >= limiteSuc ? "warning" : "default"}
                />
              )}
            </div>
          </div>
        </div>
      </section>

      {/* ── TABLA COMPARATIVA FREE VS PRO (CANÓNICA LANDINGPRICING) ── */}
      <section className="space-y-20 rounded-xl border border-concrete bg-canvas-white p-20">
        <div>
          <h3 className="text-16 font-bold text-ash-graphite">Comparativa de planes</h3>
          <p className="text-13 text-sage-green mt-2">
            Elegí el plan que mejor se adapte al volumen de tu local.
          </p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-16">
          {/* Card Free */}
          <div className="rounded-lg border border-concrete bg-canvas-white p-24 flex flex-col justify-between">
            <div>
              <div className="flex justify-between items-center">
                <span className="text-12 font-bold uppercase tracking-widest text-sage-green">Plan Free</span>
                {!esPro && (
                  <span className="px-8 py-2 rounded-full text-10 font-semibold bg-concrete/60 text-ash-graphite">
                    Tu plan actual
                  </span>
                )}
              </div>
              <div className="mt-8 text-28 font-bold text-ash-graphite">
                $0 <span className="text-14 font-normal text-sage-green">/ mes</span>
              </div>
              <p className="mt-4 text-13 text-sage-green">Ideal para cafeterías y locales chicos.</p>

              <ul className="mt-20 pt-16 border-t border-concrete space-y-10 text-13 text-ash-graphite">
                <li className="flex items-center gap-8">
                  <span className="material-symbols-outlined text-18 text-ash-graphite">check</span>
                  <span>1 sucursal activa</span>
                </li>
                <li className="flex items-center gap-8">
                  <span className="material-symbols-outlined text-18 text-ash-graphite">check</span>
                  <span>Hasta 10 mesas con QR</span>
                </li>
                <li className="flex items-center gap-8">
                  <span className="material-symbols-outlined text-18 text-ash-graphite">check</span>
                  <span>Hasta 30 productos</span>
                </li>
                <li className="flex items-center gap-8">
                  <span className="material-symbols-outlined text-18 text-ash-graphite">check</span>
                  <span>Pedidos y dashboard en tiempo real</span>
                </li>
                <li className="flex items-center gap-8">
                  <span className="material-symbols-outlined text-18 text-ash-graphite">check</span>
                  <span>Disponibilidad de ítems (86) en vivo</span>
                </li>
              </ul>
            </div>
          </div>

          {/* Card Pro */}
          <div className="rounded-lg border border-ash-graphite bg-ash-graphite text-canvas-white p-24 flex flex-col justify-between relative shadow-md">
            {esPro ? (
              <span className="absolute top-12 right-12 bg-canvas-white text-ash-graphite text-10 font-bold uppercase tracking-wider px-10 py-4 rounded-full">
                {enPrueba ? "Prueba activa" : "Plan activo"}
              </span>
            ) : (
              <span className="absolute top-12 right-12 bg-canvas-white text-ash-graphite text-10 font-bold uppercase tracking-wider px-10 py-4 rounded-full">
                Recomendado
              </span>
            )}
            <div>
              <span className="text-12 font-bold uppercase tracking-widest text-canvas-white/70">Plan Pro</span>
              <div className="mt-8 text-28 font-bold text-canvas-white">
                {precioPro} <span className="text-14 font-normal text-canvas-white/70">{moneda} / mes</span>
              </div>
              <p className="mt-4 text-13 text-canvas-white/80">{mesesPrueba} meses gratis para cada negocio nuevo.</p>

              <ul className="mt-20 pt-16 border-t border-canvas-white/20 space-y-10 text-13 text-canvas-white">
                <li className="flex items-center gap-8">
                  <span className="material-symbols-outlined text-18 text-canvas-white">check</span>
                  <span>Multi-sucursal, mesas y carta ilimitadas</span>
                </li>
                <li className="flex items-center gap-8">
                  <span className="material-symbols-outlined text-18 text-canvas-white">check</span>
                  <span>Carga masiva Excel/CSV + ajuste de precios</span>
                </li>
                <li className="flex items-center gap-8">
                  <span className="material-symbols-outlined text-18 text-canvas-white">check</span>
                  <span>Pantalla de cocina (KDS) e impresión</span>
                </li>
                <li className="flex items-center gap-8">
                  <span className="material-symbols-outlined text-18 text-canvas-white">check</span>
                  <span>Métricas y analítica de negocio</span>
                </li>
                <li className="flex items-center gap-8">
                  <span className="material-symbols-outlined text-18 text-canvas-white">check</span>
                  <span>Personalización de marca del menú</span>
                </li>
              </ul>
            </div>
          </div>
        </div>

        {/* ── SECCIÓN CTA UPGRADE ── */}
        {!esPro && (
          <div className="mt-24 pt-20 border-t border-concrete space-y-16">
            {solicitudEnviada || estadoPlan?.upgrade_solicitado_at ? (
              <div className="rounded-lg border border-concrete bg-ghost-fog p-16 flex items-start gap-12">
                <span className="material-symbols-outlined text-22 text-success shrink-0">check_circle</span>
                <div>
                  <p className="text-13 font-semibold text-ash-graphite">Solicitud registrada</p>
                  <p className="text-12 text-sage-green mt-2">
                    Te contactaremos a la brevedad por {tenant?.email_contacto || "tu correo registrado"} para activar Pro por {precioPro} {moneda} mensuales y coordinar el pago.
                  </p>
                </div>
              </div>
            ) : (
              <div className="space-y-12">
                <div>
                  <label htmlFor="upgrade-nota" className="block text-12 font-medium text-ash-graphite mb-4">
                    Comentario adicional o consulta para el equipo comercial (opcional):
                  </label>
                  <input
                    id="upgrade-nota"
                    type="text"
                    value={nota}
                    onChange={(e) => setNota(e.target.value)}
                    placeholder="Ej: Necesito agregar 2 sucursales más esta semana..."
                    className="h-44 w-full rounded-lg border border-concrete bg-canvas-white px-12 text-13 text-ash-graphite placeholder:text-stone focus:border-ash-graphite focus:outline-none"
                  />
                </div>

                <div className="flex flex-wrap items-center gap-12 pt-4">
                  <button
                    onClick={handleSolicitarUpgrade}
                    disabled={solicitando}
                    className="h-44 rounded-lg bg-plain-green px-20 text-12 font-semibold text-canvas-white transition-colors hover:bg-plain-green-muted disabled:cursor-not-allowed disabled:opacity-40"
                  >
                    {solicitando ? "Registrando..." : "Solicitar Upgrade Pro"}
                  </button>

                  <button
                    onClick={handleConsultarWhatsApp}
                    className="h-44 rounded-lg border border-concrete bg-canvas-white px-16 text-12 font-medium text-ash-graphite transition-colors hover:bg-ghost-fog active:scale-95"
                  >
                    Consultar por WhatsApp
                  </button>
                </div>
              </div>
            )}
          </div>
        )}
      </section>
    </div>
  );
}

