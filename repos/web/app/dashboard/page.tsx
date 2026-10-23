"use client";
import React, { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import dynamic from "next/dynamic";
import { useRouter } from "next/navigation";
import CartaSection from "@/components/dashboard/CartaSection";
import Logo from "@/components/brand/Logo";
import { api, cerrarSesion, SesionUsuario, Tenant } from "@/lib/api";
import { ToastProvider, ConfirmProvider } from "@/components/ui";

const CartaLecturaSection = dynamic(() => import("@/components/dashboard/CartaLecturaSection"));
const MesasSection = dynamic(() => import("@/components/dashboard/MesasSection"));
const RecepcionistaSection = dynamic(() => import("@/components/dashboard/RecepcionistaSection"));
const ConfiguracionSection = dynamic(() => import("@/components/dashboard/ConfiguracionSection"));
const MetricasSection = dynamic(() => import("@/components/dashboard/MetricasSection"));
const ReputacionSection = dynamic(() => import("@/components/dashboard/ReputacionSection"));
const EquipoSection = dynamic(() => import("@/components/dashboard/EquipoSection"));
const OnboardingTour = dynamic(() => import("@/components/dashboard/OnboardingTour"), { ssr: false });

type Section = 'carta' | 'mesas' | 'recepcionista' | 'equipo' | 'metricas' | 'resenas' | 'configuracion';
type ConfiguracionTab = "negocio" | "apariencia" | "equipo" | "sucursales" | "planes" | "mercadopago" | "asistentes";

const DASHBOARD_TOUR_VERSION = "2026-10";

function getDashboardTourStorageKey(tenantId: string, role: SesionUsuario["rol"]) {
  return `mesaclick_dashboard_tour_${DASHBOARD_TOUR_VERSION}_${tenantId}_${role}`;
}

const SECTIONS: { id: Section; icon: string; label: string }[] = [
  { id: 'carta', icon: 'restaurant_menu', label: 'Carta' },
  { id: 'mesas', icon: 'table_restaurant', label: 'Mesas & QR' },
  { id: 'recepcionista', icon: 'receipt_long', label: 'Recepcionista' },
	{ id: 'equipo', icon: 'groups', label: 'Equipo' },
  { id: 'resenas', icon: 'star_rate', label: 'Reseñas' },
];

function renderSection(
  section: Section,
  configuracionTab: ConfiguracionTab,
  onTenantUpdate: (t: Tenant) => void,
	sesion: SesionUsuario | null,
) {
  switch (section) {
		case 'carta': return sesion?.rol === 'cocina' ? <CartaLecturaSection /> : <CartaSection />;
		case 'mesas': return <MesasSection readOnly={sesion?.rol === 'mozo'} />;
    case 'recepcionista': return <RecepcionistaSection />;
    case 'metricas': return <MetricasSection />;
    case 'resenas': return <ReputacionSection />;
		case 'equipo': return <EquipoSection sesion={sesion} />;
    case 'configuracion': return <ConfiguracionSection initialTab={configuracionTab} onTenantUpdate={onTenantUpdate} />;
  }
}

export default function DashboardPage() {
  const router = useRouter();
  const [activeSection, setActiveSection] = useState<Section>('carta');
  const [configuracionTab, setConfiguracionTab] = useState<ConfiguracionTab>("negocio");
  const [isExpanded, setIsExpanded] = useState(false);
  const [isUserMenuOpen, setIsUserMenuOpen] = useState(false);
  const [tenant, setTenant] = useState<Tenant | null>(null);
	const [sesion, setSesion] = useState<SesionUsuario | null>(null);
  const [isTourOpen, setIsTourOpen] = useState(false);

  useEffect(() => {
    let cancelled = false;
    let tourTimer: ReturnType<typeof setTimeout> | undefined;

    async function loadTenant() {
			try {
				const sesionActual = await api.obtenerSesion();
				const t = await api.obtenerMiTenant();
        if (cancelled) return;

				setSesion(sesionActual);
        setTenant(t);

        // La versión nueva vuelve a mostrarse una sola vez al administrador.
        // Los demás roles pueden abrir su recorrido contextual desde el menú.
        if (sesionActual.rol === "admin" && typeof window !== "undefined") {
          try {
            const storageKey = getDashboardTourStorageKey(t.id, sesionActual.rol);
            if (!localStorage.getItem(storageKey)) {
              tourTimer = setTimeout(() => {
                if (!cancelled) setIsTourOpen(true);
              }, 600);
            }
          } catch (err) {
            console.warn("No se pudo verificar estado del recorrido:", err);
          }
        }
			} catch (err) {
				if (cancelled) return;
				console.warn("No se pudo cargar la sesión:", err);
				router.replace('/login');
			}
    }
    loadTenant();

		return () => {
			cancelled = true;
			if (tourTimer) clearTimeout(tourTimer);
		};
	}, [router]);

	const handleTourNavigate = useCallback(
		(section: Section, nextConfigurationTab?: ConfiguracionTab) => {
			if (nextConfigurationTab) setConfiguracionTab(nextConfigurationTab);
			setActiveSection(section);
			setIsExpanded(false);
		},
		[]
	);

	const handleLogout = async () => {
		const destino = sesion?.rol === 'admin' || !tenant ? '/login' : `/${tenant.slug}/login`;
		try { await cerrarSesion(); } finally { router.push(destino); }
  };

	const seccionesVisibles = SECTIONS.filter((seccion) => {
		if (!sesion) return false;
		if (sesion.rol === 'admin') return seccion.id !== 'equipo';
		if (sesion.rol === 'encargado') return seccion.id !== 'configuracion';
		if (sesion.rol === 'mozo') return ['carta', 'mesas', 'recepcionista'].includes(seccion.id);
		if (sesion.rol === 'cocina') return seccion.id === 'carta';
		return false;
	});

  return (
    <ConfirmProvider>
    <ToastProvider>
    <div className="flex h-[calc(100dvh-var(--qa-banner-offset,0px))] flex-col overflow-hidden bg-ghost-fog/45 font-inter">
      <a
        href="#dashboard-main"
        className="sr-only focus:not-sr-only focus:fixed focus:top-8 focus:left-8 focus:z-50 focus:rounded-lg focus:bg-ash-graphite focus:px-16 focus:py-8 focus:text-13 focus:font-semibold focus:text-canvas-white"
      >
        Saltar al contenido principal
      </a>
      <header className="z-30 flex h-64 shrink-0 items-center justify-between border-b border-concrete bg-canvas-white px-12 shadow-sm sm:px-20">
        <div className="flex min-w-0 items-center gap-8 sm:gap-12">
          <button
            type="button"
            onClick={() => setIsExpanded(!isExpanded)}
            className="flex h-44 w-44 shrink-0 items-center justify-center rounded-lg text-ash-graphite transition-colors hover:bg-ghost-fog"
            aria-label={isExpanded ? "Contraer navegación" : "Expandir navegación"}
            aria-expanded={isExpanded}
          >
            <span className="material-symbols-outlined text-20" aria-hidden="true">{isExpanded ? 'menu_open' : 'menu'}</span>
          </button>
          <Link
            href="/"
            className="flex shrink-0 items-center gap-8 text-ash-graphite transition-opacity hover:opacity-70"
            aria-label="Ir al inicio"
          >
            <Logo mark="full" className="h-24 w-auto" />
            <h1 className="sr-only">fudd</h1>
          </Link>
          <span className="max-w-[90px] truncate border-l border-concrete pl-8 text-12 font-medium text-sage-green sm:max-w-[260px] sm:pl-12">
            {tenant ? tenant.nombre : "Admin"}
          </span>
			{tenant && sesion?.rol === 'admin' && (
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
            type="button"
            className="flex h-44 w-44 items-center justify-center rounded-lg text-ash-graphite transition-colors hover:bg-ghost-fog"
            aria-label="Notificaciones"
          >
            <span className="material-symbols-outlined text-20" aria-hidden="true">notifications</span>
          </button>
          <div className="relative">
            <button
              type="button"
              onClick={() => setIsUserMenuOpen(!isUserMenuOpen)}
              className="flex h-44 w-44 items-center justify-center rounded-lg border border-concrete bg-canvas-white transition-all hover:border-stone hover:bg-ghost-fog"
              aria-label="Abrir menú de usuario"
              aria-expanded={isUserMenuOpen}
              aria-haspopup="menu"
            >
              <span className="material-symbols-outlined text-20 text-ash-graphite" aria-hidden="true">person</span>
            </button>
            {isUserMenuOpen && (
              <>
                <div className="fixed inset-0 z-10" onClick={() => setIsUserMenuOpen(false)} />
                <div
                  role="menu"
                  aria-label="Menú de usuario"
                  className="absolute right-0 z-20 mt-8 w-[240px] overflow-hidden rounded-xl border border-concrete bg-canvas-white py-6 shadow-xl"
                >
                  <div className="mb-4 border-b border-ghost-fog px-16 py-10">
                    <p className="truncate text-13 font-semibold text-ash-graphite">{tenant ? tenant.nombre : "Administrador"}</p>
                    <p className="mt-2 truncate text-11 text-sage-green">Sesión activa</p>
                  </div>
                  <UserMenuItem icon="account_circle" label="Perfil" />
					{sesion?.rol === 'admin' && (
                    <UserMenuItem
                      icon="credit_card"
                      label="Planes y suscripción"
                      onClick={() => {
                        setConfiguracionTab("planes");
                        setActiveSection("configuracion");
                        setIsUserMenuOpen(false);
                      }}
                    />
                  )}
					{sesion?.rol !== 'mozo' && (
                    <Link href="/kds" target="_blank" onClick={() => setIsUserMenuOpen(false)}>
                      <UserMenuItem icon="skillet" label="Pantalla Cocina (KDS) ↗" asSpan />
                    </Link>
                  )}
                  <UserMenuItem
                    icon="school"
                    label="Recorrido tutorial"
                    onClick={() => {
                      setIsTourOpen(true);
                      setIsUserMenuOpen(false);
                    }}
                  />
                  <div className="mt-8 pt-8 border-t border-ghost-fog">
                    <UserMenuItem icon="logout" label="Salir" isDanger onClick={handleLogout} />
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
          aria-label="Menú lateral móvil"
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
              type="button"
              onClick={() => setIsExpanded(false)}
              aria-label="Cerrar menú de navegación"
              className="flex h-44 w-44 items-center justify-center rounded-lg text-ash-graphite hover:bg-ghost-fog"
            >
              <span className="material-symbols-outlined text-20" aria-hidden="true">close</span>
            </button>
          </div>
          <nav className="space-y-4 flex-1 px-8" aria-label="Secciones principales móvil">
			{seccionesVisibles.map((s) => (
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
			{sesion?.rol !== 'mozo' && <Link
              href="/kds"
              target="_blank"
              rel="noopener noreferrer"
              className="flex min-h-44 items-center gap-12 rounded-lg px-12 py-8 text-sage-green transition-all hover:bg-ghost-fog hover:text-ash-graphite"
              onClick={() => setIsExpanded(false)}
              title="Abrir Pantalla de Cocina (KDS)"
              aria-label="Abrir Pantalla de Cocina (KDS) en nueva pestaña"
            >
              <span className="material-symbols-outlined text-20" aria-hidden="true">skillet</span>
              <span className="flex-1 text-13 font-medium">Cocina (KDS)</span>
              <span className="rounded border border-concrete bg-ghost-fog px-6 py-1 text-10 font-bold text-sage-green">PRO ↗</span>
			</Link>}
			{(sesion?.rol === 'admin' || sesion?.rol === 'encargado') && <NavItem
              icon="monitoring"
              label="Métricas"
              active={activeSection === "metricas"}
              expanded={true}
              dataTour="nav-metricas"
              onClick={() => {
                setActiveSection("metricas");
                setIsExpanded(false);
              }}
			/>}
          </nav>
          <div className="pt-16 border-t border-ghost-fog space-y-4 px-8">
			{sesion?.rol === 'admin' && <NavItem
              icon="settings"
              label="Configuración"
              active={activeSection === "configuracion"}
              expanded={true}
              dataTour="nav-configuracion"
              onClick={() => {
                setActiveSection("configuracion");
                setIsExpanded(false);
              }}
			/>}
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
          aria-label="Barra lateral de navegación"
          className={`${
            isExpanded ? "w-200" : "w-72"
          } hidden shrink-0 flex-col overflow-x-hidden overflow-y-auto border-r border-concrete bg-canvas-white py-16 transition-all duration-300 ease-in-out md:flex`}
        >
          <nav className={`space-y-4 flex-1 flex flex-col ${isExpanded ? "px-8" : "items-center"}`} aria-label="Secciones del panel">
			{seccionesVisibles.map((s) => (
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
			{sesion?.rol !== 'mozo' && <Link
              href="/kds"
              target="_blank"
              rel="noopener noreferrer"
              className={`flex min-h-44 items-center rounded-lg text-sage-green transition-all hover:bg-ghost-fog hover:text-ash-graphite ${
                isExpanded ? "w-full gap-12 px-12 py-8" : "w-44 justify-center"
              }`}
              title="Abrir Pantalla de Cocina (KDS) en nueva pestaña"
              aria-label="Abrir Pantalla de Cocina (KDS) en nueva pestaña"
            >
              <span className="material-symbols-outlined text-20" aria-hidden="true">skillet</span>
              {isExpanded && (
                <div className="flex items-center justify-between flex-1 min-w-0">
                  <span className="text-13 font-medium whitespace-nowrap">Cocina (KDS)</span>
                  <span className="rounded border border-concrete bg-ghost-fog px-6 py-1 text-10 font-bold text-sage-green">PRO ↗</span>
                </div>
              )}
			</Link>}
			{(sesion?.rol === 'admin' || sesion?.rol === 'encargado') && <NavItem
              icon="monitoring"
              label="Métricas"
              active={activeSection === "metricas"}
              expanded={isExpanded}
              dataTour="nav-metricas"
              onClick={() => setActiveSection("metricas")}
			/>}
          </nav>
          <div className={`pt-16 border-t border-ghost-fog space-y-4 flex flex-col ${isExpanded ? "px-8" : "items-center"}`}>
			{sesion?.rol === 'admin' && <NavItem
              icon="settings"
              label="Configuración"
              active={activeSection === "configuracion"}
              expanded={isExpanded}
              dataTour="nav-configuracion"
              onClick={() => setActiveSection("configuracion")}
			/>}
            <NavItem
              icon="school"
              label="Recorrido tutorial"
              expanded={isExpanded}
              dataTour="nav-tour"
              onClick={() => setIsTourOpen(true)}
            />
          </div>
        </aside>

        <main id="dashboard-main" className="min-w-0 flex-1 overflow-hidden">
			{renderSection(activeSection, configuracionTab, setTenant, sesion)}
        </main>
      </div>

      <OnboardingTour
        isOpen={isTourOpen}
        onClose={() => setIsTourOpen(false)}
        onNavigateSection={handleTourNavigate}
        role={sesion?.rol ?? "admin"}
        tenantName={tenant?.nombre}
        storageKey={
          tenant && sesion
            ? getDashboardTourStorageKey(tenant.id, sesion.rol)
            : undefined
        }
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
    <button
      type="button"
      data-tour={dataTour}
      title={!expanded ? label : undefined}
      aria-label={label}
      aria-current={active ? "page" : undefined}
      onClick={onClick}
      className={`flex min-h-44 cursor-pointer items-center rounded-lg text-left transition-all focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ash-graphite ${
        expanded ? "w-full gap-12 px-12 py-8" : "h-44 w-44 justify-center"
      } ${active ? "bg-ash-graphite text-canvas-white shadow-sm" : "text-sage-green hover:bg-ghost-fog hover:text-ash-graphite"}`}
    >
      <span className="material-symbols-outlined text-20" aria-hidden="true">{icon}</span>
      {expanded && <span className="text-13 font-medium whitespace-nowrap">{label}</span>}
    </button>
  );
}

function UserMenuItem({
  icon,
  label,
  isDanger = false,
  onClick,
  asSpan = false,
}: {
  icon: string;
  label: string;
  isDanger?: boolean;
  onClick?: () => void;
  asSpan?: boolean;
}) {
  const className = `flex w-full min-h-44 cursor-pointer items-center gap-12 px-16 py-8 text-left transition-colors ${
    isDanger ? "text-alert-red hover:bg-warm-pink/20" : "text-ash-graphite hover:bg-ghost-fog"
  }`;

  if (asSpan) {
    return (
      <span role="menuitem" className={className}>
        <span className="material-symbols-outlined text-18" aria-hidden="true">{icon}</span>
        <span className="text-13 font-medium">{label}</span>
      </span>
    );
  }

  return (
    <button type="button" role="menuitem" onClick={onClick} className={className}>
      <span className="material-symbols-outlined text-18" aria-hidden="true">{icon}</span>
      <span className="text-13 font-medium">{label}</span>
    </button>
  );
}
