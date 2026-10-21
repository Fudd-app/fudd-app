"use client";

import { useState, useEffect } from "react";
import { api } from "@/lib/api";
import { Progress } from "@/components/ui/Progress";
import { formatearPrecioPlan, PLAN_MONEDA } from "@/lib/planes";

export interface UpgradeModalProps {
  isOpen: boolean;
  onClose: () => void;
  recurso: "mesas" | "productos" | "sucursales" | "carga_masiva" | "personalizacion";
  limite?: number;
  uso?: number;
  onUpgradeSolicitado?: () => void;
}

const RECURSO_INFO: Record<
  "mesas" | "productos" | "sucursales" | "carga_masiva" | "personalizacion",
  {
    nombre: string;
    nombrePlural: string;
    limiteDefault: number;
    mensaje: string;
    icono: string;
  }
> = {
  mesas: {
    nombre: "mesa",
    nombrePlural: "mesas",
    limiteDefault: 10,
    mensaje:
      "Has alcanzado el límite de 10 mesas activas de tu plan Free. Para seguir sumando mesas a tu salón, actualizá a Pro.",
    icono: "table_restaurant",
  },
  productos: {
    nombre: "producto",
    nombrePlural: "productos",
    limiteDefault: 30,
    mensaje:
      "Has alcanzado el límite de 30 productos de tu plan Free. Para ampliar tu carta sin límites, actualizá a Pro.",
    icono: "restaurant_menu",
  },
  sucursales: {
    nombre: "sucursal",
    nombrePlural: "sucursales",
    limiteDefault: 1,
    mensaje:
      "Has alcanzado el límite de 1 sucursal de tu plan Free. Para gestionar múltiples locales desde un mismo panel, actualizá a Pro.",
    icono: "storefront",
  },
  carga_masiva: {
    nombre: "carga masiva",
    nombrePlural: "cargas masivas",
    limiteDefault: 0,
    mensaje:
      "La importación masiva de productos vía archivo es una función exclusiva del plan Pro.",
    icono: "upload_file",
  },
  personalizacion: {
    nombre: "personalización",
    nombrePlural: "personalizaciones",
    limiteDefault: 0,
    mensaje:
      "Los íconos de categorías, colores por zona, tipografías exclusivas y el retiro de la marca de agua son funciones del plan Pro.",
    icono: "branding_watermark",
  },
};

const BENEFICIOS_PRO = [
  "Mesas y pedidos ilimitados",
  "Productos y categorías sin límite",
  "Múltiples sucursales en simultáneo",
  "Carta digital personalizada sin marca de agua",
  "Soporte prioritario por WhatsApp y atención dedicada",
];

export default function UpgradeModal({
  isOpen,
  onClose,
  recurso,
  limite,
  uso,
  onUpgradeSolicitado,
}: UpgradeModalProps) {
  const [solicitando, setSolicitando] = useState(false);

  useEffect(() => {
    if (!isOpen) return;
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        onClose();
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  const info = RECURSO_INFO[recurso] || RECURSO_INFO.mesas;
  const limiteEfectivo = limite ?? info.limiteDefault;
  const usoEfectivo = uso ?? limiteEfectivo;
  const esProOnlyFeature = recurso === "carga_masiva" || recurso === "personalizacion";

  const handleContactarPro = async () => {
    if (solicitando) return;
    setSolicitando(true);

    try {
      await api.solicitarUpgradePro(`Desde modal de límite de ${recurso}`);
    } catch (err) {
      console.warn("No se pudo registrar solicitud de upgrade:", err);
    } finally {
      setSolicitando(false);
      onUpgradeSolicitado?.();

      const asunto = encodeURIComponent(`Solicitud Upgrade Pro - Límite de ${info.nombrePlural}`);
      const cuerpo = encodeURIComponent(
        `Hola equipo de fudd,\n\nAlcancé el límite de ${recurso} en mi plan Free y quiero pasar a Pro por ${formatearPrecioPlan()} ${PLAN_MONEDA} por mes para seguir expandiendo mi negocio.\n\n¡Gracias!`
      );
      const mailtoUrl = `mailto:supportmesaclick@gmail.com?subject=${asunto}&body=${cuerpo}`;

      if (typeof window !== "undefined") {
        window.open(mailtoUrl, "_blank");
      }
      onClose();
    }
  };

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="upgrade-modal-title"
      className="fixed inset-0 z-50 flex items-center justify-center p-16 sm:p-24"
    >
      {/* Backdrop */}
      <div
        onClick={onClose}
        className="fixed inset-0 bg-ash-graphite/40 backdrop-blur-xs transition-opacity"
      />

      {/* Modal Card */}
      <div className="relative flex max-h-[90vh] w-full max-w-lg flex-col rounded-2xl border border-concrete bg-canvas-white shadow-2xl overflow-hidden font-inter animate-in fade-in zoom-in-95 duration-200">
        {/* Header */}
        <div className="flex items-start justify-between border-b border-concrete px-20 py-16 sm:px-24">
          <div className="flex items-center gap-12">
            <div className="flex h-40 w-40 shrink-0 items-center justify-center rounded-xl border border-concrete bg-ghost-fog text-ash-graphite">
              <span className="material-symbols-outlined text-20">{info.icono}</span>
            </div>
            <div>
              <span className="inline-block rounded bg-stone/15 px-6 py-1 text-10 font-bold uppercase tracking-wide text-ash-graphite">
                Plan Free
              </span>
              <h2
                id="upgrade-modal-title"
                className="mt-2 text-16 font-bold text-ash-graphite sm:text-18"
              >
                Llegaste al límite de tu plan Free
              </h2>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            aria-label="Cerrar ventana"
            className="flex h-36 w-36 items-center justify-center rounded-lg text-sage-green hover:bg-ghost-fog hover:text-ash-graphite transition-colors"
          >
            ✕
          </button>
        </div>

        {/* Body */}
        <div className="overflow-y-auto px-20 py-20 sm:px-24 space-y-20">
          {/* Mensaje principal */}
          <p className="text-13 leading-relaxed text-sage-green">
            {info.mensaje}
          </p>

          {/* Métrica de cupo (si no es función Pro-only) */}
          {!esProOnlyFeature && (
            <div className="rounded-xl border border-concrete bg-ghost-fog/40 p-16 space-y-8">
              <div className="flex items-center justify-between text-12 font-medium">
                <span className="text-ash-graphite">Cupo utilizado</span>
                <span className="font-mono font-semibold text-ash-graphite">
                  {usoEfectivo} / {limiteEfectivo} {info.nombrePlural} usadas
                </span>
              </div>
              <Progress
                valor={usoEfectivo}
                max={limiteEfectivo}
              />
            </div>
          )}

          {/* Beneficios Pro */}
          <div className="rounded-xl border border-concrete bg-canvas-white p-16 space-y-12">
            <div className="flex items-center gap-6">
              <span className="material-symbols-outlined text-18 text-plain-green">
                verified
              </span>
              <h3 className="text-12 font-bold uppercase tracking-wider text-ash-graphite">
                Todo lo que incluye el Plan Pro
              </h3>
            </div>
            <ul className="space-y-8 text-12 text-ash-graphite">
              {BENEFICIOS_PRO.map((b) => (
                <li key={b} className="flex items-center gap-8">
                  <span className="material-symbols-outlined text-16 text-plain-green shrink-0">
                    check_circle
                  </span>
                  <span>{b}</span>
                </li>
              ))}
            </ul>
            <p className="border-t border-concrete pt-10 text-12 font-semibold text-ash-graphite">
              {formatearPrecioPlan()} {PLAN_MONEDA} por mes
            </p>
          </div>
        </div>

        {/* Footer */}
        <div className="flex flex-col-reverse sm:flex-row sm:items-center sm:justify-end gap-10 border-t border-concrete bg-ghost-fog/20 px-20 py-16 sm:px-24">
          <button
            type="button"
            onClick={onClose}
            className="flex h-44 items-center justify-center rounded-lg border border-concrete bg-canvas-white px-16 text-13 font-medium text-ash-graphite hover:border-stone hover:bg-ghost-fog transition-colors"
          >
            Ahora no
          </button>
          <button
            type="button"
            disabled={solicitando}
            onClick={handleContactarPro}
            className="flex h-44 items-center justify-center gap-8 rounded-lg bg-ash-graphite px-20 text-13 font-semibold text-canvas-white hover:bg-stone transition-all active:scale-[0.98] disabled:opacity-60"
          >
            <span className="material-symbols-outlined text-18">upgrade</span>
            <span>{solicitando ? "Registrando solicitud..." : "Contactar para pasar a Pro"}</span>
          </button>
        </div>
      </div>
    </div>
  );
}
