"use client";
import React, { useState, useEffect } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import CartaSection from "@/components/dashboard/CartaSection";
import MesasSection from "@/components/dashboard/MesasSection";
import RecepcionistaSection from "@/components/dashboard/RecepcionistaSection";
import ConfiguracionSection from "@/components/dashboard/ConfiguracionSection";
import MetricasSection from "@/components/dashboard/MetricasSection";
import ReputacionSection from "@/components/dashboard/ReputacionSection";
import Logo from "@/components/brand/Logo";
import { api, cerrarSesion, estaAutenticado, Tenant } from "@/lib/api";
import { ToastProvider, ConfirmProvider } from "@/components/ui";
import OnboardingTour from "@/components/dashboard/OnboardingTour";

type Section = 'carta' | 'mesas' | 'recepcionista' | 'metricas' | 'resenas' | 'configuracion';

const SECTIONS: { id: Section; icon: string; label: string }[] = [
  { id: 'carta', icon: 'restaurant_menu', label: 'Carta' },
  { id: 'mesas', icon: 'table_restaurant', label: 'Mesas & QR' },
  { id: 'recepcionista', icon: 'receipt_long', label: 'Recepcionista' },
  { id: 'resenas', icon: 'star_rate', label: 'Reseñas' },
];

function renderSection(
  section: Section,
  configuracionTab: "negocio" | "apariencia" | "equipo" | "sucursales" | "planes" | "mercadopago",
  onTenantUpdate: (t: Tenant) => void,
) {
  switch (section) {
    case 'carta': return <CartaSection />;
    case 'mesas': return <MesasSection />;
    case 'recepcionista': return <RecepcionistaSection />;
    case 'metricas': return <MetricasSection />;
    case 'resenas': return <ReputacionSection />;
    case 'configuracion': return <ConfiguracionSection initialTab={configuracionTab} onTenantUpdate={onTenantUpdate} />;
  }
}

export default function DashboardPage() {
  const router = useRouter();
  const [activeSection, setActiveSection] = useState<Section>('carta');
  const [configuracionTab, setConfiguracionTab] = useState<"negocio" | "apariencia" | "equipo" | "sucursales" | "planes" | "mercadopago">("negocio");
  const [isExpanded, setIsExpanded] = useState(false);
  const [isUserMenuOpen, setIsUserMenuOpen] = useState(false);
  const [tenant, setTenant] = useState<Tenant | null>(null);
  const [isTourOpen, setIsTourOpen] = useState(false);

  useEffect(() => {
    if (typeof window !== "undefined") {
      const initialSec = localStorage.getItem("mesaclick_initial_section");
      const staffRol = localStorage.getItem("mesaclick_staff_rol");
      if (initialSec === "recepcionista" || staffRol === "mozo") {
        setActiveSection("recepcionista");
        localStorage.removeItem("mesaclick_initial_section");
      }
    }

    async function loadTenant() {
      if (estaAutenticado()) {
        try {
          const t = await api.obtenerMiTenant();
          setTenant(t);

          // Si el usuario registrado aún no vio el recorrido para este tenant en esta versión (solo admin)
          if (typeof window !== "undefined") {
            try {
              const staffRol = localStorage.getItem("mesaclick_staff_rol");
              if (staffRol && staffRol !== "admin") {
                return;
              }
              const tenantTourKey = `mesaclick_tour_seen_${t.id}`;
              const globalTourKey = "mesaclick_admin_tour_completed";
              const seen = localStorage.getItem(tenantTourKey) || localStorage.getItem(globalTourKey);
              if (!seen) {
                const timer = setTimeout(() => {
                  setIsTourOpen(true);
                }, 600);
                return () => clearTimeout(timer);
              }
            } catch (err) {
              console.warn("No se pudo verificar estado del tour:", err);
            }
          }
        } catch (err) {
          console.warn("No se pudo cargar tenant:", err);
        }
      }
    }
    loadTenant();
  }, []);

  const handleLogout = () => {
    const staffSlug = typeof window !== "undefined" ? localStorage.getItem("mesaclick_staff_slug") : null;
    if (typeof window !== "undefined") {
      localStorage.removeItem("mesaclick_staff_rol");
      localStorage.removeItem("mesaclick_staff_slug");
      localStorage.removeItem("mesaclick_initial_section");
    }
    cerrarSesion();
    router.push(staffSlug ? `/${staffSlug}/login` : "/login");
  };

  return (
    <ConfirmProvider>
    <ToastProvider>
    <div className="flex h-screen flex-col overflow-hidden bg-ghost-fog/45 font-inter">
      <header className="z-30 flex h-64 shrink-0 items-center justify-between border-b border-concrete bg-canvas-white px-12 shadow-sm sm:px-20">
        <div className="flex min-w-0 items-center gap-8 sm:gap-12">
          <button
            onClick={() => setIsExpanded(!isExpanded)}
            className="flex h-44 w-44 shrink-0 items-center justify-center rounded-lg text-ash-graphite transition-colors hover:bg-ghost-fog"
            aria-label={isExpanded ? "Contraer navegación" : "Expandir navegación"}
          >
            <span className="material-symbols-outlined text-20">{isExpanded ? 'menu_open' : 'menu'}</span>
          </button>
          <Link
            href="/"
            className="flex shrink-0 items-center gap-8 text-ash-graphite transition-opacity hover:opacity-70"
            aria-label="Ir al inicio"
          >
            <Logo className="h-24 w-24" />
            <h1 className="hidden text-15 font-bold uppercase tracking-tight sm:block">Mesa CLICK</h1>
          </Link>
          <span className="max-w-[90px] truncate border-l border-concrete pl-8 text-12 font-medium text-sage-green sm:max-w-[260px] sm:pl-12">
            {tenant ? tenant.nombre : "Admin"}
          </span>
          {tenant && (
            <>
              <span className="hidden rounded-full border border-concrete bg-ghost-fog px-8 py-2 text-10 font-mono text-sage-green lg:inline-block">
                /{tenant.slug}
              </span>
              <button
                type="button"
                onClick={() => {
                  setConfiguracionTab("planes");
                  setActiveSection("configuracion");
                }}
                className={`inline-flex items-center gap-4 rounded-full border px-8 py-2 text-10 font-mono transition-colors ${
                  tenant.plan === "pro"
                    ? "border-ash-graphite bg-ash-graphite text-canvas-white hover:bg-stone"
                    : "border-concrete bg-ghost-fog text-sage-green hover:border-stone hover:text-ash-graphite"
                }`}
                title="Ver planes y suscripción"
                aria-label={`Plan ${tenant.plan === "pro" ? "Pro" : "Free"}. Clic para gestionar plan.`}
              >
                {tenant.plan === "pro" ? "PRO" : "FREE"}
              </button>
            </>
          )}
        </div>
        <div className="flex shrink-0 items-center gap-4 sm:gap-8">
          <button
            className="flex h-44 w-44 items-center justify-center rounded-lg text-ash-graphite transition-colors hover:bg-ghost-fog"
            aria-label="Notificaciones"
          >
            <span className="material-symbols-outlined text-20">notifications</span>
          </button>
          <div className="relative">
            <button
              onClick={() => setIsUserMenuOpen(!isUserMenuOpen)}
              className="flex h-44 w-44 items-center justify-center rounded-lg border border-concrete bg-canvas-white transition-all hover:border-stone hover:bg-ghost-fog"
              aria-label="Abrir menú de usuario"
              aria-expanded={isUserMenuOpen}
            >
              <span className="material-symbols-outlined text-20 text-ash-graphite">person</span>
            </button>
            {isUserMenuOpen && (
              <>
                <div className="fixed inset-0 z-10" onClick={() => setIsUserMenuOpen(false)} />
                <div className="absolute right-0 z-20 mt-8 w-[240px] overflow-hidden rounded-xl border border-concrete bg-canvas-white py-6 shadow-xl">
                  <div className="mb-4 border-b border-ghost-fog px-16 py-10">
                    <p className="truncate text-13 font-semibold text-ash-graphite">{tenant ? tenant.nombre : "Administrador"}</p>
                    <p className="mt-2 truncate text-11 text-sage-green">Sesión activa</p>
                  </div>
                  <UserMenuItem icon="account_circle" label="Perfil" />
                  <div onClick={() => { setConfiguracionTab("planes"); setActiveSection("configuracion"); setIsUserMenuOpen(false); }}>
                    <UserMenuItem icon="credit_card" label="Planes y suscripción" />
                  </div>
                  <Link href="/kds" target="_blank" onClick={() => setIsUserMenuOpen(false)}>
                    <UserMenuItem icon="skillet" label="Pantalla Cocina (KDS) ↗" />
                  </Link>
                  <div onClick={() => { setIsTourOpen(true); setIsUserMenuOpen(false); }}>
                    <UserMenuItem icon="school" label="Recorrido tutorial" />
                  </div>
                  <div className="mt-8 pt-8 border-t border-ghost-fog">
                    <div onClick={handleLogout}>
                      <UserMenuItem icon="logout" label="Salir" isDanger />
                    </div>
                  </div>
                </div>
              </>
            )}
          </div>
        </div>
      </header>

      <div className="relative flex flex-1 overflow-hidden">
        {/* Mobile Backdrop */}
        {isExpanded && (
          <div 
            className="fixed inset-0 bg-system-black/40 z-40 md:hidden backdrop-blur-xs transition-opacity"
            onClick={() => setIsExpanded(false)}
          />
        )}

        {/* Mobile Drawer (Slide-out) */}
        <aside
          className={`fixed inset-y-0 left-0 z-50 flex w-240 flex-col border-r border-concrete bg-canvas-white py-16 shadow-2xl transition-transform duration-300 ease-in-out md:hidden ${
            isExpanded ? "translate-x-0" : "-translate-x-full"
          }`}
        >
          <div className="mb-8 flex items-center justify-between border-b border-ghost-fog px-16 pb-16">
            <Link
              href="/"
              className="flex items-center gap-8 text-ash-graphite hover:opacity-70 transition-opacity"
              aria-label="Ir al inicio"
            >
              <Logo className="w-22 h-22" />
              <span className="font-bold uppercase text-14">Menú Principal</span>
            </Link>
            <button 
              onClick={() => setIsExpanded(false)}
              className="flex h-44 w-44 items-center justify-center rounded-lg text-ash-graphite hover:bg-ghost-fog"
            >
              <span className="material-symbols-outlined text-20">close</span>
            </button>
          </div>
          <nav className="space-y-4 flex-1 px-8">
            {SECTIONS.map((s) => (
              <NavItem
                key={s.id}
                icon={s.icon}
                label={s.label}
                active={activeSection === s.id}
                expanded={true}
                dataTour={`nav-${s.id}`}
                onClick={() => {
                  setActiveSection(s.id);
                  setIsExpanded(false);
                }}
              />
            ))}
            {/* Botón Cocina (KDS) justo abajo de Recepcionista */}
            <Link
              href="/kds"
              target="_blank"
              rel="noopener noreferrer"
              className="flex min-h-44 items-center gap-12 rounded-lg px-12 py-8 text-sage-green transition-all hover:bg-ghost-fog hover:text-ash-graphite"
              onClick={() => setIsExpanded(false)}
              title="Abrir Pantalla de Cocina (KDS)"
            >
              <span className="material-symbols-outlined text-20">skillet</span>
              <span className="flex-1 text-13 font-medium">Cocina (KDS)</span>
              <span className="rounded border border-concrete bg-ghost-fog px-6 py-1 text-10 font-bold text-sage-green">PRO ↗</span>
            </Link>
            <NavItem
              icon="monitoring"
              label="Métricas"
              active={activeSection === "metricas"}
              expanded={true}
              dataTour="nav-metricas"
              onClick={() => {
                setActiveSection("metricas");
                setIsExpanded(false);
              }}
            />
          </nav>
          <div className="pt-16 border-t border-ghost-fog space-y-4 px-8">
            <NavItem
              icon="settings"
              label="Configuración"
              active={activeSection === "configuracion"}
              expanded={true}
              dataTour="nav-configuracion"
              onClick={() => {
                setActiveSection("configuracion");
                setIsExpanded(false);
              }}
            />
            <NavItem
              icon="school"
              label="Recorrido tutorial"
              expanded={true}
              dataTour="nav-tour"
              onClick={() => {
                setIsTourOpen(true);
                setIsExpanded(false);
              }}
            />
          </div>
        </aside>

        {/* Desktop Sidebar */}
        <aside
          className={`${
            isExpanded ? "w-200" : "w-72"
          } hidden shrink-0 flex-col overflow-x-hidden overflow-y-auto border-r border-concrete bg-canvas-white py-16 transition-all duration-300 ease-in-out md:flex`}
        >
          <nav className={`space-y-4 flex-1 flex flex-col ${isExpanded ? "px-8" : "items-center"}`}>
            {SECTIONS.map((s) => (
              <NavItem
                key={s.id}
                icon={s.icon}
                label={s.label}
                active={activeSection === s.id}
                expanded={isExpanded}
                dataTour={`nav-${s.id}`}
                onClick={() => setActiveSection(s.id)}
              />
            ))}
            {/* Botón Cocina (KDS) justo abajo de Recepcionista */}
            <Link
              href="/kds"
              target="_blank"
              rel="noopener noreferrer"
              className={`flex min-h-44 items-center rounded-lg text-sage-green transition-all hover:bg-ghost-fog hover:text-ash-graphite ${
                isExpanded ? "w-full gap-12 px-12 py-8" : "w-44 justify-center"
              }`}
              title="Abrir Pantalla de Cocina (KDS) en nueva pestaña"
            >
              <span className="material-symbols-outlined text-20">skillet</span>
              {isExpanded && (
                <div className="flex items-center justify-between flex-1 min-w-0">
                  <span className="text-13 font-medium whitespace-nowrap">Cocina (KDS)</span>
                  <span className="rounded border border-concrete bg-ghost-fog px-6 py-1 text-10 font-bold text-sage-green">PRO ↗</span>
                </div>
              )}
            </Link>
            <NavItem
              icon="monitoring"
              label="Métricas"
              active={activeSection === "metricas"}
              expanded={isExpanded}
              dataTour="nav-metricas"
              onClick={() => setActiveSection("metricas")}
            />
          </nav>
          <div className={`pt-16 border-t border-ghost-fog space-y-4 flex flex-col ${isExpanded ? "px-8" : "items-center"}`}>
            <NavItem
              icon="settings"
              label="Configuración"
              active={activeSection === "configuracion"}
              expanded={isExpanded}
              dataTour="nav-configuracion"
              onClick={() => setActiveSection("configuracion")}
            />
            <NavItem
              icon="school"
              label="Recorrido tutorial"
              expanded={isExpanded}
              dataTour="nav-tour"
              onClick={() => setIsTourOpen(true)}
            />
          </div>
        </aside>

        <main className="min-w-0 flex-1 overflow-hidden">
          {renderSection(activeSection, configuracionTab, setTenant)}
        </main>
      </div>

      <OnboardingTour
        isOpen={isTourOpen}
        onClose={() => setIsTourOpen(false)}
        activeSection={activeSection}
        onNavigateSection={setActiveSection}
        tenantName={tenant?.nombre}
        tenantId={tenant?.id}
      />
    </div>
    </ToastProvider>
    </ConfirmProvider>
  );
}

function NavItem({
  icon,
  label,
  active = false,
  expanded = false,
  onClick,
  dataTour,
}: {
  icon: string;
  label: string;
  active?: boolean;
  expanded?: boolean;
  onClick?: () => void;
  dataTour?: string;
}) {
  return (
    <div
      data-tour={dataTour}
      title={!expanded ? label : undefined}
      onClick={onClick}
      className={`flex min-h-44 cursor-pointer items-center rounded-lg transition-all ${
        expanded ? "w-full gap-12 px-12 py-8" : "h-44 w-44 justify-center"
      } ${active ? "bg-ash-graphite text-canvas-white shadow-sm" : "text-sage-green hover:bg-ghost-fog hover:text-ash-graphite"}`}
    >
      <span className="material-symbols-outlined text-20">{icon}</span>
      {expanded && <span className="text-13 font-medium whitespace-nowrap">{label}</span>}
    </div>
  );
}

function UserMenuItem({
  icon,
  label,
  isDanger = false,
}: {
  icon: string;
  label: string;
  isDanger?: boolean;
}) {
  return (
    <div
      className={`flex min-h-44 cursor-pointer items-center gap-12 px-16 py-8 transition-colors ${
        isDanger ? "text-alert-red hover:bg-warm-pink/20" : "text-ash-graphite hover:bg-ghost-fog"
      }`}
    >
      <span className="material-symbols-outlined text-18">{icon}</span>
      <span className="text-13 font-medium">{label}</span>
    </div>
  );
}
