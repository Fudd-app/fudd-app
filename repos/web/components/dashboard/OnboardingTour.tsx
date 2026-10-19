"use client";

import React, { useState, useEffect, useCallback } from "react";
import { useToast } from "@/components/ui";

export interface TourStep {
  id: string;
  section: "configuracion" | "mesas" | "carta" | "recepcionista";
  badge: string;
  title: string;
  subtitle: string;
  description: string;
  icon: string;
  bullets: string[];
}

export const TOUR_STEPS: TourStep[] = [
  {
    id: "step-configuracion",
    section: "configuracion",
    badge: "Paso 1 de 4",
    title: "Configuración & Negocio",
    subtitle: "Identidad y horarios",
    description:
      "Ajustá la identidad de tu marca (logo y colores), datos fiscales y los horarios de atención de tus sucursales.",
    icon: "settings",
    bullets: [
      "Branding y apariencia de la carta digital",
      "Horarios semanales y turnos de apertura",
    ],
  },
  {
    id: "step-mesas",
    section: "mesas",
    badge: "Paso 2 de 4",
    title: "Mesas & Códigos QR",
    subtitle: "Acceso para comensales",
    description:
      "Creá las mesas de tu salón. Cada mesa genera su código QR descargable listo para imprimir y colocar en el salón.",
    icon: "table_restaurant",
    bullets: [
      "Descarga de códigos QR en alta resolución",
      "Enlace directo para probar el menú digital",
    ],
  },
  {
    id: "step-carta",
    section: "carta",
    badge: "Paso 3 de 4",
    title: "Carta Digital",
    subtitle: "Platos, precios y fotos",
    description:
      "Estructurá tu menú en categorías y cargá tus productos con fotos, precios y control de disponibilidad inmediata.",
    icon: "restaurant_menu",
    bullets: [
      "Categorías adaptadas a navegación móvil",
      "Pausar o habilitar ítems en un clic",
    ],
  },
  {
    id: "step-recepcionista",
    section: "recepcionista",
    badge: "Paso 4 de 4",
    title: "Pedidos en Tiempo Real",
    subtitle: "Panel en vivo para salón",
    description:
      "Monitoreá los pedidos que envían los clientes desde las mesas con alertas sonoras y actualizá sus estados en vivo.",
    icon: "receipt_long",
    bullets: [
      "Alertas sonoras y conexión en tiempo real (SSE)",
      "Estados: Recibido → Preparación → Entregado",
    ],
  },
];

export const TOUR_STORAGE_KEY = "mesaclick_admin_tour_completed";

interface OnboardingTourProps {
  isOpen: boolean;
  onClose: () => void;
	activeSection: "carta" | "mesas" | "recepcionista" | "equipo" | "metricas" | "resenas" | "configuracion";
  onNavigateSection: (
		section: "carta" | "mesas" | "recepcionista" | "equipo" | "metricas" | "resenas" | "configuracion"
  ) => void;
  tenantName?: string;
  tenantId?: string;
}

export default function OnboardingTour({
  isOpen,
  onClose,
  activeSection,
  onNavigateSection,
  tenantName,
  tenantId,
}: OnboardingTourProps) {
  const [currentStepIndex, setCurrentStepIndex] = useState(0);
  const toast = useToast();

  // Sincronizar sección activa al cambiar de paso
  const goToStep = useCallback(
    (index: number) => {
      const step = TOUR_STEPS[index];
      if (!step) return;
      setCurrentStepIndex(index);
      if (activeSection !== step.section) {
        onNavigateSection(step.section);
      }
    },
    [activeSection, onNavigateSection]
  );

  const handleComplete = useCallback(() => {
    if (typeof window !== "undefined") {
      try {
        localStorage.setItem(TOUR_STORAGE_KEY, "true");
        if (tenantId) {
          localStorage.setItem(`mesaclick_tour_seen_${tenantId}`, "true");
        }
      } catch (err) {
        console.warn("No se pudo guardar estado del tour:", err);
      }
    }
    toast.success("¡Recorrido tutorial finalizado! Podés volver a verlo desde tu perfil.");
    setCurrentStepIndex(0);
    onClose();
  }, [tenantId, toast, onClose]);

  const handleSkip = useCallback(() => {
    if (typeof window !== "undefined") {
      try {
        localStorage.setItem(TOUR_STORAGE_KEY, "true");
        if (tenantId) {
          localStorage.setItem(`mesaclick_tour_seen_${tenantId}`, "true");
        }
      } catch (err) {
        console.warn("No se pudo guardar estado del tour:", err);
      }
    }
    toast.info("Recorrido omitido. Podés reactivarlo desde tu perfil en 'Recorrido tutorial'.");
    setCurrentStepIndex(0);
    onClose();
  }, [tenantId, toast, onClose]);

  // Al abrir el tour, sincronizar sección
  useEffect(() => {
    if (isOpen) {
      onNavigateSection(TOUR_STEPS[0].section);
    }
  }, [isOpen, onNavigateSection]);

  // Manejo de teclado (Flecha izq/der y Escape)
  useEffect(() => {
    if (!isOpen) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        handleSkip();
      } else if (e.key === "ArrowRight") {
        if (currentStepIndex < TOUR_STEPS.length - 1) {
          goToStep(currentStepIndex + 1);
        }
      } else if (e.key === "ArrowLeft") {
        if (currentStepIndex > 0) {
          goToStep(currentStepIndex - 1);
        }
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [isOpen, currentStepIndex, goToStep, handleSkip]);

  const handleNext = () => {
    if (currentStepIndex < TOUR_STEPS.length - 1) {
      goToStep(currentStepIndex + 1);
    } else {
      handleComplete();
    }
  };

  const handlePrev = () => {
    if (currentStepIndex > 0) {
      goToStep(currentStepIndex - 1);
    }
  };

  if (!isOpen) return null;

  const currentStep = TOUR_STEPS[currentStepIndex];
  const isLastStep = currentStepIndex === TOUR_STEPS.length - 1;

  return (
    <div
      role="dialog"
      aria-modal="false"
      aria-labelledby="tour-widget-title"
      className="fixed bottom-12 right-12 sm:bottom-20 sm:right-20 z-50 w-[clamp(440px,48vw,620px)] max-w-[calc(100vw-24px)] max-h-[calc(100vh-24px)] [@supports(height:100dvh)]:max-h-[calc(100dvh-24px)] bg-canvas-white border border-system-black rounded-lg shadow-xl overflow-hidden flex flex-col animate-in slide-in-from-bottom-4 duration-150"
    >
      {/* Cabecera compacta */}
      <div className="px-14 py-8 border-b border-concrete/60 bg-vanilla-cream/50 flex items-center justify-between gap-8 shrink-0">
        <div className="flex items-center gap-8 min-w-0">
          <div className="w-28 h-28 rounded bg-ash-graphite text-canvas-white flex items-center justify-center shrink-0">
            <span className="material-symbols-outlined text-16">
              {currentStep.icon}
            </span>
          </div>
          <div className="min-w-0">
            <div className="flex items-center gap-6">
              <span className="text-10 font-bold uppercase tracking-wider text-sage-green font-mono">
                {currentStep.badge}
              </span>
              {tenantName && (
                <span className="text-10 font-mono text-stone truncate max-w-120">
                  · {tenantName}
                </span>
              )}
            </div>
            <h2
              id="tour-widget-title"
              className="text-13 font-bold text-ash-graphite truncate"
            >
              {currentStep.title}
            </h2>
          </div>
        </div>
        <button
          onClick={handleSkip}
          className="h-32 w-32 min-h-32 min-w-32 text-sage-green hover:text-ash-graphite p-0 rounded hover:bg-vanilla-cream transition-colors cursor-pointer shrink-0 flex items-center justify-center"
          aria-label="Cerrar recorrido"
          title="Cerrar"
        >
          <span className="material-symbols-outlined text-18">close</span>
        </button>
      </div>

      {/* Barra de progreso sutil */}
      <div className="grid grid-cols-4 gap-3 px-14 pt-4 bg-canvas-white shrink-0">
        {TOUR_STEPS.map((step, idx) => {
          const isDone = idx < currentStepIndex;
          const isCurrent = idx === currentStepIndex;
          return (
            <button
              key={step.id}
              onClick={() => goToStep(idx)}
              className="h-16 min-h-16 min-w-0 flex items-center cursor-pointer focus:outline-hidden"
              title={`Ir al paso ${idx + 1}`}
            >
              <span
                aria-hidden="true"
                className="block h-[5px] w-full rounded-full transition-all duration-150"
                style={{
                  backgroundColor:
                    isDone || isCurrent
                      ? "var(--color-ash-graphite, #0a0a0a)"
                      : "var(--color-concrete, #d9d9d9)",
                }}
              />
            </button>
          );
        })}
      </div>

      {/* Contenido resumido */}
      <div className="px-14 py-8 space-y-6 text-12 min-h-0 overflow-y-auto overscroll-contain">
        <p className="text-deep-forest leading-relaxed">
          {currentStep.description}
        </p>

        <ul className="space-y-2 pt-2 border-t border-ghost-fog">
          {currentStep.bullets.map((bullet, i) => (
            <li key={i} className="flex items-start gap-6 text-11 text-sage-green leading-tight">
              <span className="material-symbols-outlined text-13 text-ash-graphite shrink-0 mt-0.5">
                check
              </span>
              <span>{bullet}</span>
            </li>
          ))}
        </ul>
      </div>

      {/* Pie de navegación compacto */}
      <div className="px-14 py-6 border-t border-concrete/50 bg-vanilla-cream/30 flex items-center justify-between gap-8 shrink-0">
        <button
          onClick={handleSkip}
          className="h-[36px] min-h-[36px] text-11 font-medium text-sage-green hover:text-ash-graphite transition-colors underline decoration-dotted underline-offset-2 cursor-pointer"
        >
          Omitir recorrido
        </button>

        <div className="flex items-center gap-6">
          {currentStepIndex > 0 && (
            <button
              onClick={handlePrev}
              className="h-[36px] min-h-[36px] px-10 py-0 rounded border border-concrete text-ash-graphite text-11 font-medium hover:bg-vanilla-cream transition-all flex items-center gap-3 cursor-pointer"
            >
              <span className="material-symbols-outlined text-13">
                arrow_back
              </span>
              <span>Ant</span>
            </button>
          )}

          <button
            onClick={handleNext}
            className="h-[36px] min-h-[36px] px-12 py-0 rounded bg-plain-green text-canvas-white text-11 font-medium hover:bg-plain-green-muted transition-all flex items-center gap-4 cursor-pointer shadow-xs"
          >
            <span>{isLastStep ? "Finalizar" : "Siguiente"}</span>
            <span className="material-symbols-outlined text-13">
              {isLastStep ? "done" : "arrow_forward"}
            </span>
          </button>
        </div>
      </div>
    </div>
  );
}
