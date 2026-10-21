"use client";

import React, { useCallback, useEffect, useMemo, useState } from "react";
import type { SesionUsuario } from "@/lib/api";
import { useToast } from "@/components/ui";

export type TourRole = SesionUsuario["rol"];
export type TourSection =
  | "carta"
  | "mesas"
  | "recepcionista"
  | "equipo"
  | "metricas"
  | "resenas"
  | "configuracion";
export type TourConfigurationTab =
  | "negocio"
  | "apariencia"
  | "equipo"
  | "sucursales"
  | "planes"
  | "mercadopago";

export interface TourStep {
  id: string;
  section: TourSection;
  configurationTab?: TourConfigurationTab;
  roles: TourRole[];
  location: string;
  title: string;
  subtitle: string;
  description: string;
  icon: string;
  bullets: string[];
  externalAction?: {
    label: string;
    href: string;
  };
}

const ALL_ROLES: TourRole[] = ["admin", "encargado", "mozo", "cocina"];

export const TOUR_STEPS: TourStep[] = [
  {
    id: "configuracion-negocio",
    section: "configuracion",
    configurationTab: "negocio",
    roles: ["admin"],
    location: "Configuración · Negocio",
    title: "Poné a punto tu negocio",
    subtitle: "La base de toda la operación",
    description:
      "Completá los datos del comercio y recorré las pestañas de configuración para definir sucursales, horarios, cobros y plan.",
    icon: "tune",
    bullets: [
      "Datos comerciales y fiscales centralizados",
      "Sucursales, turnos y horarios de atención",
      "Mercado Pago, Plan Free y beneficios Pro",
    ],
  },
  {
    id: "apariencia",
    section: "configuracion",
    configurationTab: "apariencia",
    roles: ["admin"],
    location: "Configuración · Apariencia",
    title: "Hacé que la carta sea tuya",
    subtitle: "Identidad visual con vista previa",
    description:
      "Probá el logo, los colores y el estilo de tu carta antes de guardarlos. El teléfono interactivo permite revisar Carta, Mi pedido y Cuenta.",
    icon: "palette",
    bullets: [
      "Vista previa móvil clara u oscura en tiempo real",
      "Colores independientes para categorías y acciones",
      "Logo, tipografía y marca de agua según el plan",
    ],
  },
  {
    id: "carta",
    section: "carta",
    roles: ALL_ROLES,
    location: "Carta",
    title: "Construí una carta visual",
    subtitle: "Productos listos para vender",
    description:
      "Organizá categorías e ítems con precios, variantes, fotos e íconos. También podés controlar qué se vende y en qué horario.",
    icon: "restaurant_menu",
    bullets: [
      "Fotos de productos e íconos de categoría",
      "Carga masiva y ajuste porcentual de precios",
      "Visibilidad de productos y franjas horarias",
    ],
  },
  {
    id: "mesas",
    section: "mesas",
    roles: ["admin", "encargado", "mozo"],
    location: "Mesas & QR",
    title: "Conectá cada mesa con la carta",
    subtitle: "QR y atención asistida",
    description:
      "Cada mesa tiene un acceso único a la carta. Desde acá podés revisar su estado, probar el enlace y asistir un pedido cuando haga falta.",
    icon: "qr_code_2",
    bullets: [
      "QR individual listo para descargar e imprimir",
      "Enlace directo para probar la experiencia del cliente",
      "Pedido asistido con trazabilidad del integrante",
    ],
  },
  {
    id: "equipo",
    section: "equipo",
    roles: ["admin", "encargado"],
    location: "Equipo",
    title: "Dale acceso a tu equipo",
    subtitle: "Roles y credenciales por sucursal",
    description:
      "Creá accesos para encargados, mozos y cocina, asigná su sucursal y controlá el ciclo de vida de cada cuenta.",
    icon: "groups",
    bullets: [
      "Alta con usuario y clave o invitación por Google",
      "Permisos específicos para cada rol y sucursal",
      "Bloqueo, archivo y blanqueo de contraseña",
    ],
  },
  {
    id: "recepcionista",
    section: "recepcionista",
    roles: ["admin", "encargado", "mozo"],
    location: "Recepcionista",
    title: "Operá el salón en tiempo real",
    subtitle: "Pedidos, cuentas y mesas en vivo",
    description:
      "Seguí lo que pasa en el salón sin recargar la pantalla: pedidos nuevos, platos listos y solicitudes de cuenta aparecen en vivo.",
    icon: "receipt_long",
    bullets: [
      "Actualizaciones y alertas mediante conexión en tiempo real",
      "Pedidos asistidos y seguimiento por mesa",
      "Impresión o reimpresión de tickets térmicos",
    ],
  },
  {
    id: "kds",
    section: "recepcionista",
    roles: ["admin", "encargado", "cocina"],
    location: "Cocina (KDS) · Pantalla independiente",
    title: "Coordiná la cocina con KDS",
    subtitle: "Comandas y tiempos por ítem",
    description:
      "La cocina trabaja en una pantalla separada y de alto contraste. Las comandas avanzan por estado y sus cambios vuelven al salón al instante.",
    icon: "skillet",
    bullets: [
      "Pendientes, en preparación y listos para servir",
      "Temporizadores visuales para detectar demoras",
      "Tickets de 58/80 mm con reintento manual",
    ],
    externalAction: {
      label: "Abrir Cocina (KDS)",
      href: "/kds",
    },
  },
  {
    id: "metricas",
    section: "metricas",
    roles: ["admin", "encargado"],
    location: "Métricas",
    title: "Entendé cómo rinde el negocio",
    subtitle: "Indicadores para decidir mejor",
    description:
      "Consultá en un solo lugar los resultados operativos y financieros para detectar tendencias y tomar decisiones con datos reales.",
    icon: "monitoring",
    bullets: [
      "Ventas, pedidos y ticket promedio",
      "Rendimiento de categorías y productos",
      "Lectura rápida del período y evolución del negocio",
    ],
  },
  {
    id: "resenas",
    section: "resenas",
    roles: ["admin", "encargado"],
    location: "Reseñas",
    title: "Convertí opiniones en mejoras",
    subtitle: "Reputación y feedback del cliente",
    description:
      "Revisá las calificaciones posteriores al consumo y atendé primero los comentarios que necesitan una respuesta del equipo.",
    icon: "reviews",
    bullets: [
      "CSAT y distribución de calificaciones",
      "Feedback privado para experiencias de 1 a 3 estrellas",
      "Derivación de clientes satisfechos a Google Reviews",
    ],
  },
];

const ROLE_LABELS: Record<TourRole, string> = {
  admin: "Administrador",
  encargado: "Encargado",
  mozo: "Mozo",
  cocina: "Cocina",
};

export function getTourSteps(role: TourRole): TourStep[] {
  return TOUR_STEPS.filter((step) => step.roles.includes(role)).map((step) => {
    if (step.id === "equipo" && role === "admin") {
      return {
        ...step,
        section: "configuracion",
        configurationTab: "equipo",
        location: "Configuración · Equipo de trabajo",
      };
    }

    if (step.id === "kds" && role === "cocina") {
      return { ...step, section: "carta" };
    }

    if (step.id === "carta" && role === "cocina") {
      return {
        ...step,
        title: "Consultá la carta vigente",
        subtitle: "Referencia rápida para cocina",
        description:
          "Revisá desde una vista de solo lectura los productos, variantes y detalles que el negocio tiene publicados antes de pasar al KDS.",
        bullets: [
          "Productos y categorías siempre actualizados",
          "Variantes y detalles visibles sin riesgo de edición",
          "Acceso directo a la pantalla operativa de cocina",
        ],
      };
    }

    return step;
  });
}

interface OnboardingTourProps {
  isOpen: boolean;
  onClose: () => void;
  onNavigateSection: (
    section: TourSection,
    configurationTab?: TourConfigurationTab
  ) => void;
  role: TourRole;
  tenantName?: string;
  storageKey?: string;
}

export default function OnboardingTour({
  isOpen,
  onClose,
  onNavigateSection,
  role,
  tenantName,
  storageKey,
}: OnboardingTourProps) {
  const [currentStepIndex, setCurrentStepIndex] = useState(0);
  const toast = useToast();
  const steps = useMemo(() => getTourSteps(role), [role]);

  const goToStep = useCallback(
    (index: number) => {
      const step = steps[index];
      if (!step) return;
      setCurrentStepIndex(index);
      onNavigateSection(step.section, step.configurationTab);
    },
    [onNavigateSection, steps]
  );

  const markAsSeen = useCallback(() => {
    if (!storageKey || typeof window === "undefined") return;
    try {
      localStorage.setItem(storageKey, "true");
    } catch (err) {
      console.warn("No se pudo guardar el estado del recorrido:", err);
    }
  }, [storageKey]);

  const handleComplete = useCallback(() => {
    markAsSeen();
    toast.success(
      "¡Recorrido finalizado! Podés volver a abrirlo desde el menú lateral o tu perfil."
    );
    setCurrentStepIndex(0);
    onClose();
  }, [markAsSeen, onClose, toast]);

  const handleSkip = useCallback(() => {
    markAsSeen();
    toast.info(
      "Recorrido cerrado. Podés retomarlo cuando quieras desde ‘Recorrido tutorial’."
    );
    setCurrentStepIndex(0);
    onClose();
  }, [markAsSeen, onClose, toast]);

  useEffect(() => {
    if (!isOpen || steps.length === 0) return;
    const firstStep = steps[0];
    onNavigateSection(firstStep.section, firstStep.configurationTab);
  }, [isOpen, onNavigateSection, steps]);

  useEffect(() => {
    if (!isOpen) return;

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        handleSkip();
      } else if (event.key === "ArrowRight") {
        if (currentStepIndex < steps.length - 1) {
          goToStep(currentStepIndex + 1);
        }
      } else if (event.key === "ArrowLeft" && currentStepIndex > 0) {
        goToStep(currentStepIndex - 1);
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [currentStepIndex, goToStep, handleSkip, isOpen, steps.length]);

  if (!isOpen || steps.length === 0) return null;

  const currentStep = steps[currentStepIndex] ?? steps[0];
  const isLastStep = currentStepIndex === steps.length - 1;

  return (
    <div
      role="dialog"
      aria-modal="false"
      aria-labelledby="tour-widget-title"
      aria-describedby="tour-widget-description"
      className="fixed bottom-12 right-12 z-50 flex max-h-[calc(100vh-24px)] w-[min(520px,calc(100vw-24px))] flex-col overflow-hidden rounded-lg border border-system-black bg-canvas-white shadow-xl animate-in slide-in-from-bottom-4 duration-150 [@supports(height:100dvh)]:max-h-[calc(100dvh-24px)] sm:bottom-20 sm:right-20"
    >
      <div className="flex shrink-0 items-start justify-between gap-8 border-b border-concrete/60 bg-vanilla-cream/50 px-14 py-10">
        <div className="flex min-w-0 items-start gap-8">
          <div className="flex h-32 w-32 shrink-0 items-center justify-center rounded bg-ash-graphite text-canvas-white">
            <span className="material-symbols-outlined text-18" aria-hidden="true">
              {currentStep.icon}
            </span>
          </div>
          <div className="min-w-0">
            <div className="flex flex-wrap items-center gap-x-6 gap-y-2 font-mono text-10">
              <span className="font-bold uppercase tracking-wider text-plain-green-muted">
                Paso {currentStepIndex + 1} de {steps.length}
              </span>
              <span className="text-stone">· {ROLE_LABELS[role]}</span>
              {tenantName && <span className="truncate text-stone">· {tenantName}</span>}
            </div>
            <h2
              id="tour-widget-title"
              className="mt-2 text-15 font-bold leading-tight text-ash-graphite"
            >
              {currentStep.title}
            </h2>
            <p className="mt-1 text-11 text-sage-green">{currentStep.subtitle}</p>
          </div>
        </div>
        <button
          type="button"
          onClick={handleSkip}
          className="flex h-32 min-h-32 w-32 min-w-32 shrink-0 cursor-pointer items-center justify-center rounded p-0 text-sage-green transition-colors hover:bg-vanilla-cream hover:text-ash-graphite"
          aria-label="Cerrar recorrido"
          title="Cerrar"
        >
          <span className="material-symbols-outlined text-18" aria-hidden="true">
            close
          </span>
        </button>
      </div>

      <div
        className="grid shrink-0 gap-3 bg-canvas-white px-14 pt-5"
        style={{ gridTemplateColumns: `repeat(${steps.length}, minmax(0, 1fr))` }}
        aria-label="Progreso del recorrido"
      >
        {steps.map((step, index) => {
          const isReached = index <= currentStepIndex;
          const isCurrent = index === currentStepIndex;
          return (
            <button
              type="button"
              key={step.id}
              onClick={() => goToStep(index)}
              className="flex h-16 min-h-16 min-w-0 cursor-pointer items-center focus:outline-hidden"
              title={`Paso ${index + 1}: ${step.title}`}
              aria-label={`Ir al paso ${index + 1}: ${step.title}`}
              aria-current={isCurrent ? "step" : undefined}
            >
              <span
                aria-hidden="true"
                className="block h-[5px] w-full rounded-full transition-colors duration-150"
                style={{
                  backgroundColor: isReached
                    ? "var(--color-ash-graphite, #0a0a0a)"
                    : "var(--color-concrete, #d9d9d9)",
                }}
              />
            </button>
          );
        })}
      </div>

      <div
        className="min-h-0 space-y-8 overflow-y-auto overscroll-contain px-14 py-10"
        aria-live="polite"
      >
        <div className="inline-flex items-center gap-4 rounded-full border border-concrete bg-ghost-fog px-8 py-3 font-mono text-10 text-sage-green">
          <span className="material-symbols-outlined text-13" aria-hidden="true">
            location_on
          </span>
          <span>{currentStep.location}</span>
        </div>

        <p
          id="tour-widget-description"
          className="text-12 leading-relaxed text-deep-forest"
        >
          {currentStep.description}
        </p>

        <ul className="space-y-5 border-t border-ghost-fog pt-8">
          {currentStep.bullets.map((bullet) => (
            <li
              key={bullet}
              className="flex items-start gap-6 text-11 leading-snug text-sage-green"
            >
              <span
                className="material-symbols-outlined mt-0.5 shrink-0 text-14 text-plain-green-muted"
                aria-hidden="true"
              >
                check_circle
              </span>
              <span>{bullet}</span>
            </li>
          ))}
        </ul>

        {currentStep.externalAction && (
          <a
            href={currentStep.externalAction.href}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex min-h-36 items-center gap-5 rounded border border-ash-graphite px-10 py-6 text-11 font-semibold text-ash-graphite transition-colors hover:bg-ash-graphite hover:text-canvas-white"
          >
            <span className="material-symbols-outlined text-15" aria-hidden="true">
              open_in_new
            </span>
            {currentStep.externalAction.label}
          </a>
        )}
      </div>

      <div className="flex shrink-0 items-center justify-between gap-8 border-t border-concrete/50 bg-vanilla-cream/30 px-14 py-8">
        <button
          type="button"
          onClick={handleSkip}
          className="min-h-36 cursor-pointer text-11 font-medium text-sage-green underline decoration-dotted underline-offset-2 transition-colors hover:text-ash-graphite"
        >
          Salir del recorrido
        </button>

        <div className="flex items-center gap-6">
          {currentStepIndex > 0 && (
            <button
              type="button"
              onClick={() => goToStep(currentStepIndex - 1)}
              className="flex min-h-36 cursor-pointer items-center gap-3 rounded border border-concrete px-10 py-0 text-11 font-medium text-ash-graphite transition-colors hover:bg-vanilla-cream"
            >
              <span className="material-symbols-outlined text-13" aria-hidden="true">
                arrow_back
              </span>
              <span>Anterior</span>
            </button>
          )}

          <button
            type="button"
            onClick={() =>
              isLastStep ? handleComplete() : goToStep(currentStepIndex + 1)
            }
            className="flex min-h-36 cursor-pointer items-center gap-4 rounded bg-plain-green px-12 py-0 text-11 font-medium text-canvas-white shadow-xs transition-colors hover:bg-plain-green-muted"
          >
            <span>{isLastStep ? "Finalizar" : "Siguiente"}</span>
            <span className="material-symbols-outlined text-13" aria-hidden="true">
              {isLastStep ? "done" : "arrow_forward"}
            </span>
          </button>
        </div>
      </div>
    </div>
  );
}
