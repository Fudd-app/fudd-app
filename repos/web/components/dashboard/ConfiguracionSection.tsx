"use client";

import React, { useEffect, useMemo, useRef, useState } from "react";
import { api, Tenant, Sucursal, getErrorMessage, getApiBaseUrl } from "@/lib/api";
import { DEFAULT_MESA_PRIMARY } from "@/components/menu/BrandHeader";
import EquipoSection from "./EquipoSection";
import PlanesSection from "./PlanesSection";
import UpgradeModal from "./UpgradeModal";
import AparienciaMenuPreview from "./AparienciaMenuPreview";
import AsistentesIaTab from "./AsistentesIaTab";

/* ─────────────────────────── contenedor ─────────────────────────── */

const TABS = ["negocio", "apariencia", "equipo", "sucursales", "planes", "mercadopago", "asistentes"] as const;
type Tab = (typeof TABS)[number];
const TAB_LABELS: Record<Tab, string> = {
  negocio: "Negocio",
  apariencia: "Apariencia",
  equipo: "Equipo de trabajo",
  sucursales: "Sucursales",
  planes: "Planes y Suscripción",
  mercadopago: "Mercado Pago",
  asistentes: "Asistentes de IA",
};

export default function ConfiguracionSection({
  initialTab,
  onTenantUpdate,
}: {
  initialTab?: Tab;
  onTenantUpdate?: (t: Tenant) => void;
} = {}) {
  const [tab, setTab] = useState<Tab>(initialTab || "negocio");
  const [prevInitialTab, setPrevInitialTab] = useState(initialTab);
  if (initialTab !== prevInitialTab) {
    setPrevInitialTab(initialTab);
    if (initialTab) {
      setTab(initialTab);
    }
  }

  const [tenant, setTenant] = useState<Tenant | null>(null);
  const [sucursales, setSucursales] = useState<Sucursal[]>([]);
  const [sucursalSelId, setSucursalSelId] = useState("");

  useEffect(() => {
    let vivo = true;
    (async () => {
      try {
        const t = await api.obtenerMiTenant();
        if (vivo) setTenant(t);
      } catch {
        /* sin tenant: la vista sigue funcionando con datos vacíos */
      }
      try {
        const s = await api.listarSucursales();
        if (vivo && s && s.length) {
          setSucursales(s);
          setSucursalSelId(s[0].id);
        }
      } catch {
        /* sin sucursales */
      }
    })();
    return () => {
      vivo = false;
    };
  }, []);

  const sucursalSel = sucursales.find((s) => s.id === sucursalSelId) ?? null;

  return (
    <div className="h-full w-full space-y-20 overflow-y-auto bg-canvas-white p-16 font-inter sm:p-24 md:p-32">
      <header className="flex flex-col gap-12 md:flex-row md:items-center md:justify-between">
        <div className="min-w-0">
          <h2 className="text-24 font-semibold tracking-[-0.02em] text-ash-graphite sm:text-32">Configuración</h2>
          <p className="mt-4 text-13 text-sage-green sm:text-14">
            Administrá negocio, apariencia del menú, equipo, sucursales y cobros digitales.
          </p>
        </div>
        {sucursales.length > 0 && (
          <select
            value={sucursalSelId}
            onChange={(e) => setSucursalSelId(e.target.value)}
            className="h-44 rounded-lg border border-concrete bg-canvas-white px-12 text-13 outline-none focus:border-system-black"
          >
            {sucursales.map((s) => (
              <option key={s.id} value={s.id}>
                Sucursal: {s.nombre}
              </option>
            ))}
          </select>
        )}
      </header>

      <nav className="flex max-w-full gap-4 overflow-x-auto border-b border-concrete no-scrollbar" aria-label="Secciones de configuración">
        {TABS.map((t) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={`h-44 shrink-0 border-x-0 border-t-0 border-b-2 bg-transparent px-14 text-12 font-semibold transition-colors ${
              tab === t
                ? "border-ash-graphite text-ash-graphite"
                : "border-transparent text-sage-green hover:text-ash-graphite"
            }`}
            aria-current={tab === t ? "page" : undefined}
          >
            {TAB_LABELS[t]}
          </button>
        ))}
      </nav>

      {tab === "negocio" && (
        <NegocioTab
          key={tenant?.id ?? "sin-tenant"}
          tenant={tenant}
          onIrAMercadoPago={() => setTab("mercadopago")}
          onIrAPlanes={() => setTab("planes")}
        />
      )}
      {tab === "apariencia" && (
        <AparienciaTab
          sucursal={sucursalSel}
          tenant={tenant}
          onTenantUpdate={(actualizado) => {
            setTenant(actualizado);
            if (onTenantUpdate) onTenantUpdate(actualizado);
          }}
        />
      )}
      {tab === "equipo" && <EquipoTab />}
      {tab === "asistentes" && <AsistentesIaTab />}
      {tab === "sucursales" && (
        <SucursalesTab
          key={sucursalSelId || "sin-sucursal"}
          tenant={tenant}
          sucursales={sucursales}
          setSucursales={setSucursales}
          selId={sucursalSelId}
          setSelId={setSucursalSelId}
          onIrAMercadoPago={() => setTab("mercadopago")}
          onIrAPlanes={() => setTab("planes")}
        />
      )}
      {tab === "planes" && (
        <PlanesSection
          key={tenant?.id ?? "sin-tenant"}
          tenant={tenant}
          onTenantUpdate={(actualizado) => {
            setTenant(actualizado);
            if (onTenantUpdate) onTenantUpdate(actualizado);
          }}
        />
      )}
      {tab === "mercadopago" && (
        <MercadoPagoTab
          key={`${tenant?.id ?? "sin-tenant"}-${sucursalSel?.id ?? "sin-sucursal"}`}
          tenant={tenant}
          setTenant={setTenant}
          sucursales={sucursales}
          setSucursales={setSucursales}
          selId={sucursalSelId}
          setSelId={setSucursalSelId}
        />
      )}
    </div>
  );
}

/* ─────────────────────────── piezas comunes ─────────────────────────── */

function Card({ titulo, children }: { titulo: string; children: React.ReactNode }) {
  return (
    <section className="overflow-hidden rounded-xl border border-concrete bg-canvas-white">
      <div className="border-b border-concrete/70 px-16 py-14 sm:px-20">
        <p className="text-15 font-semibold tracking-[-0.01em] text-ash-graphite">{titulo}</p>
      </div>
      <div className="space-y-16 p-16 sm:p-20">{children}</div>
    </section>
  );
}

function Campo({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <label className="block">
      <span className="block text-11 font-mono text-sage-green uppercase tracking-wider mb-6">{label}</span>
      {children}
    </label>
  );
}

const INPUT =
  "w-full h-44 px-12 text-14 rounded-lg border border-concrete bg-canvas-white outline-none focus:border-system-black";



function Guardado({ visible }: { visible: boolean }) {
  if (!visible) return null;
  return (
    <p className="flex items-center gap-6 text-12 font-medium text-success-muted">
      <span className="material-symbols-outlined text-16">check</span>
      Cambios guardados.
    </p>
  );
}

function PillPrimaria({
  children,
  onClick,
  disabled,
}: {
  children: React.ReactNode;
  onClick?: () => void;
  disabled?: boolean;
}) {
  return (
    <button
      onClick={onClick}
      disabled={disabled}
      className="h-44 rounded-lg bg-plain-green px-16 text-12 font-semibold text-canvas-white transition-colors hover:bg-plain-green-muted disabled:cursor-not-allowed disabled:opacity-40"
    >
      {children}
    </button>
  );
}

/* ─────────────────────────── tab: Negocio ─────────────────────────── */

const RUBROS = ["Cafetería", "Bar", "Restaurante", "Cervecería", "Pizzería", "Otro"];

function NegocioTab({
  tenant,
  onIrAMercadoPago,
  onIrAPlanes,
}: {
  tenant: Tenant | null;
  onIrAMercadoPago?: () => void;
  onIrAPlanes?: () => void;
}) {
  const rubro = tenant?.rubro;
  const rubroNormalizado = useMemo(() => {
    if (!rubro) return RUBROS[0];
    const clean = rubro.toLowerCase().replace(/í/g, "i").replace(/é/g, "e");
    const found = RUBROS.find((r) => r.toLowerCase().replace(/í/g, "i").replace(/é/g, "e") === clean);
    return found ?? RUBROS[0];
  }, [rubro]);

  const fiscalData = (tenant?.datos_fiscales ?? {}) as Record<string, string>;

  const [form, setForm] = useState(() => ({
    nombre: tenant?.nombre ?? "",
    rubro: rubroNormalizado,
    emailContacto: tenant?.email_contacto ?? "",
    whatsapp: tenant?.whatsapp ?? "",
    descripcion: tenant?.descripcion ?? "",
    googleReviewUrl: tenant?.google_review_url ?? "",
    razonSocial: typeof fiscalData.razon_social === "string" ? fiscalData.razon_social : "",
    cuit: typeof fiscalData.cuit === "string" ? fiscalData.cuit : "",
    condicionIva: typeof fiscalData.condicion_iva === "string" ? fiscalData.condicion_iva : "",
  }));
  const [loading, setLoading] = useState(false);
  const [ok, setOk] = useState(false);
  const [error, setError] = useState("");

  const linkBase = `mesa-click-web.onrender.com/${tenant?.slug ?? "tu-negocio"}`;
  const set = <K extends keyof typeof form>(k: K, v: (typeof form)[K]) => {
    setForm((f) => ({ ...f, [k]: v }));
    setOk(false);
    setError("");
  };

  const handleGuardar = async () => {
    setLoading(true);
    setError("");
    setOk(false);
    try {
      await api.actualizarMiTenant({
        nombre: form.nombre,
        rubro: form.rubro,
        email_contacto: form.emailContacto,
        whatsapp: form.whatsapp,
        descripcion: form.descripcion,
        google_review_url: form.googleReviewUrl,
        datos_fiscales: {
          razon_social: form.razonSocial,
          cuit: form.cuit,
          condicion_iva: form.condicionIva,
        },
      });
      setOk(true);
    } catch (err: unknown) {
      setError(getErrorMessage(err, "Error al guardar los datos del negocio."));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="space-y-16">
      <div className="grid lg:grid-cols-[1fr_320px] gap-16 items-start">
        <Card titulo="Datos del negocio">
          <div className="grid sm:grid-cols-2 gap-12">
            <Campo label="Nombre del negocio">
              <input className={INPUT} value={form.nombre} onChange={(e) => set("nombre", e.target.value)} />
            </Campo>
            <Campo label="Rubro">
              <select className={INPUT} value={form.rubro} onChange={(e) => set("rubro", e.target.value)}>
                {RUBROS.map((r) => (
                  <option key={r}>{r}</option>
                ))}
              </select>
            </Campo>
            <Campo label="Email de contacto">
              <input
                type="email"
                className={INPUT}
                value={form.emailContacto}
                onChange={(e) => set("emailContacto", e.target.value)}
                placeholder="contacto@minegocio.com"
              />
            </Campo>
            <Campo label="WhatsApp general">
              <input
                className={INPUT}
                value={form.whatsapp}
                onChange={(e) => set("whatsapp", e.target.value)}
                placeholder="+54 9 ..."
              />
            </Campo>
          </div>
          <Campo label="Link público base">
            <input className={`${INPUT} font-mono text-13`} value={linkBase} readOnly />
          </Campo>
          <Campo label="Descripción breve">
            <input
              className={INPUT}
              value={form.descripcion}
              onChange={(e) => set("descripcion", e.target.value)}
              placeholder="Café de especialidad, meriendas y opciones rápidas."
            />
          </Campo>
          <Campo label="Enlace / Place ID de Google Reviews (Smart Funnel)">
            <input
              className={INPUT}
              value={form.googleReviewUrl}
              onChange={(e) => set("googleReviewUrl", e.target.value)}
              placeholder="https://g.page/r/.../review o enlace de Google Maps"
            />
            <span className="block text-11 text-sage-green mt-4">
              Usado para el Smart Funnel: comensales que califiquen con 4 o 5 estrellas serán invitados a recomendarte en Google Maps.
            </span>
          </Campo>

          <div className="pt-16 border-t border-concrete space-y-12">
            <p className="text-12 font-mono text-sage-green uppercase tracking-wider">
              Datos fiscales y facturación (opcional)
            </p>
            <div className="grid sm:grid-cols-2 gap-12">
              <Campo label="Razón social">
                <input
                  className={INPUT}
                  value={form.razonSocial}
                  onChange={(e) => set("razonSocial", e.target.value)}
                  placeholder="Ej: Gastronómica Central S.R.L."
                />
              </Campo>
              <Campo label="CUIT / CUIL">
                <input
                  className={INPUT}
                  value={form.cuit}
                  onChange={(e) => set("cuit", e.target.value)}
                  placeholder="30-12345678-9"
                />
              </Campo>
              <Campo label="Condición frente al IVA">
                <select
                  className={INPUT}
                  value={form.condicionIva}
                  onChange={(e) => set("condicionIva", e.target.value)}
                >
                  <option value="">Sin especificar</option>
                  <option value="Responsable Inscripto">Responsable Inscripto</option>
                  <option value="Monotributo">Monotributo</option>
                  <option value="Exento">Exento</option>
                  <option value="Consumidor Final">Consumidor Final</option>
                </select>
              </Campo>
            </div>
          </div>

          <div className="pt-16 border-t border-concrete flex flex-col sm:flex-row sm:items-center sm:justify-between gap-12">
            <div>
              <p className="text-12 font-mono text-sage-green uppercase tracking-wider">
                Cobros con Mercado Pago
              </p>
              <p className="text-12 text-sage-green mt-2">
                Habilitá pagos desde la mesa, cargá credenciales y revisá las URLs de webhook en la pestaña dedicada.
              </p>
            </div>
            {onIrAMercadoPago && (
              <button
                type="button"
                onClick={onIrAMercadoPago}
                className="h-44 shrink-0 rounded-lg border border-concrete bg-ghost-fog/40 px-14 text-12 font-semibold text-ash-graphite hover:border-ash-graphite hover:bg-canvas-white transition-colors cursor-pointer"
              >
                Configurar Mercado Pago →
              </button>
            )}
          </div>

          {error && <p className="text-12 text-alert-red">{error}</p>}
          <div className="flex items-center gap-16">
            <PillPrimaria onClick={handleGuardar} disabled={loading}>
              {loading ? "Guardando..." : "Guardar cambios"}
            </PillPrimaria>
            <Guardado visible={ok} />
          </div>
        </Card>

        <Card titulo="Estado del plan">
          <div>
            <div className="flex items-center gap-8">
              <p className="text-16 font-bold text-ash-graphite">
                {tenant?.plan === "pro" ? "Plan Pro" : "Plan Free"}
              </p>
              <span
                className={`inline-flex items-center px-8 py-2 rounded-full text-10 font-mono font-semibold uppercase ${
                  tenant?.plan === "pro"
                    ? "bg-ash-graphite text-canvas-white"
                    : "bg-ghost-fog border border-concrete text-ash-graphite"
                }`}
              >
                {tenant?.plan === "pro" ? "Activo" : "Gratuito"}
              </span>
            </div>
            <p className="text-13 text-sage-green mt-4">
              {tenant?.plan === "pro"
                ? "Disfrutás de sucursales, mesas y productos ilimitados."
                : "Plan con límites de hasta 1 sucursal, 10 mesas y 30 productos."}
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-8 pt-4">
            <button
              type="button"
              onClick={onIrAPlanes}
              className="h-44 rounded-lg border border-concrete bg-canvas-white px-16 text-12 font-medium text-ash-graphite hover:bg-ghost-fog active:scale-95 transition-all cursor-pointer"
            >
              Ver límites y suscripción →
            </button>
          </div>
        </Card>
      </div>
    </div>
  );
}

/* ─────────────────────────── tab: Apariencia ─────────────────────────── */

type EstiloVisual = "claro" | "oscuro";

type AparienciaForm = {
  nombreVisible: string;
  color: string;
  colorCategoria: string;
  colorAccion: string;
  estilo: EstiloVisual;
  logoUrl: string;
  tipoFuente: string;
  mostrarMarcaAgua: boolean;
};

const COLOR_DEFAULT = DEFAULT_MESA_PRIMARY;
const FUENTES_CARTA = [
  { value: "Inter", label: "Inter", detail: "Moderna", family: "var(--font-inter), sans-serif" },
  { value: "Space Grotesk", label: "Space Grotesk", detail: "Técnica / urbana", family: "var(--font-space-grotesk), sans-serif" },
  { value: "Playfair Display", label: "Playfair Display", detail: "Elegante / clásica", family: "var(--font-playfair-display), serif" },
  { value: "system-ui", label: "Sistema", detail: "Nativa del dispositivo", family: "system-ui, sans-serif" },
] as const;

function esColorHex(valor: string) {
  return /^#[0-9a-fA-F]{6}$/.test(valor);
}

function aparienciaDefault(sucursal: Sucursal | null, tenant: Tenant | null): AparienciaForm {
  const base = tenant?.nombre_fantasia || tenant?.nombre || "Tu negocio";
  return {
    nombreVisible: sucursal ? `${base} - ${sucursal.nombre}` : base,
    color: tenant?.color_primario && esColorHex(tenant.color_primario) ? tenant.color_primario : COLOR_DEFAULT,
    colorCategoria: tenant?.color_categoria && esColorHex(tenant.color_categoria) ? tenant.color_categoria : COLOR_DEFAULT,
    colorAccion: tenant?.color_accion && esColorHex(tenant.color_accion) ? tenant.color_accion : COLOR_DEFAULT,
    estilo: tenant?.estilo_visual === "claro" || tenant?.estilo_visual === "oscuro" ? tenant.estilo_visual : "oscuro",
    logoUrl: tenant?.logo_url ?? "",
    tipoFuente: tenant?.tipo_fuente ?? "Inter",
    mostrarMarcaAgua: tenant?.mostrar_marca_agua ?? true,
  };
}

function leerAparienciaGuardada(storageKey: string, fallback: AparienciaForm): AparienciaForm {
  if (typeof window === "undefined") return fallback;
  try {
    const raw = localStorage.getItem(storageKey);
    if (!raw) return fallback;
    const data = JSON.parse(raw) as Partial<AparienciaForm>;
    return {
      nombreVisible: typeof data.nombreVisible === "string" ? data.nombreVisible : fallback.nombreVisible,
      color: typeof data.color === "string" && esColorHex(data.color) ? data.color : fallback.color,
      colorCategoria: typeof data.colorCategoria === "string" && esColorHex(data.colorCategoria) ? data.colorCategoria : fallback.colorCategoria,
      colorAccion: typeof data.colorAccion === "string" && esColorHex(data.colorAccion) ? data.colorAccion : fallback.colorAccion,
      estilo: data.estilo === "claro" || data.estilo === "oscuro" ? data.estilo : fallback.estilo,
      logoUrl: typeof data.logoUrl === "string" ? data.logoUrl : fallback.logoUrl,
      tipoFuente: typeof data.tipoFuente === "string" ? data.tipoFuente : fallback.tipoFuente,
      mostrarMarcaAgua: typeof data.mostrarMarcaAgua === "boolean" ? data.mostrarMarcaAgua : fallback.mostrarMarcaAgua,
    };
  } catch {
    return fallback;
  }
}

function guardarAparienciaLocal(storageKey: string, apariencia: AparienciaForm) {
  if (typeof window === "undefined") return;
  try {
    localStorage.setItem(storageKey, JSON.stringify(apariencia));
  } catch {
    /* Si el navegador bloquea localStorage, la vista previa sigue funcionando en memoria. */
  }
}

function EditorColorEnVivo({
  label,
  detalle,
  value,
  pro = false,
  onChange,
}: {
  label: string;
  detalle: string;
  value: string;
  pro?: boolean;
  onChange: (value: string) => void;
}) {
  const colorSeguro = esColorHex(value) ? value : COLOR_DEFAULT;

  return (
    <div className="w-full rounded-xl border border-concrete bg-canvas-white p-12 shadow-sm">
      <div className="flex items-center justify-between gap-8">
        <span className="text-10 font-mono font-semibold uppercase tracking-[0.08em] text-sage-green">
          {label}
        </span>
        {pro && (
          <span className="rounded-full bg-ash-graphite px-7 py-2 text-9 font-mono font-bold uppercase tracking-wider text-canvas-white">
            Pro
          </span>
        )}
      </div>
      <div className="mt-8 flex items-center gap-8">
        <label
          className="relative h-40 w-40 shrink-0 cursor-pointer overflow-hidden rounded-lg border border-concrete shadow-inner"
          style={{ backgroundColor: colorSeguro }}
          title={`Elegir ${label.toLowerCase()}`}
        >
          <input
            type="color"
            value={colorSeguro}
            onChange={(event) => onChange(event.target.value)}
            className="absolute inset-0 h-full w-full cursor-pointer opacity-0"
            aria-label={`Elegir ${label.toLowerCase()}`}
          />
        </label>
        <input
          value={value.toUpperCase()}
          maxLength={7}
          spellCheck={false}
          onChange={(event) => onChange(event.target.value)}
          className="h-40 min-w-0 flex-1 rounded-lg border border-concrete bg-canvas-white px-9 font-mono text-11 font-semibold uppercase text-ash-graphite outline-none focus:border-plain-green"
          aria-label={`${label} en hexadecimal`}
        />
      </div>
      <p className="mt-7 text-10 leading-relaxed text-sage-green">{detalle}</p>
    </div>
  );
}

function EditorIdentidadEnVivo({
  nombre,
  logoUrl,
  onNombreChange,
  onLogo,
  onRemoveLogo,
}: {
  nombre: string;
  logoUrl: string;
  onNombreChange: (value: string) => void;
  onLogo: (event: React.ChangeEvent<HTMLInputElement>) => void;
  onRemoveLogo: () => void;
}) {
  return (
    <div className="w-full rounded-xl border border-concrete bg-canvas-white p-12 shadow-sm">
      <span className="text-10 font-mono font-semibold uppercase tracking-[0.08em] text-sage-green">
        Nombre y logo
      </span>
      <input
        value={nombre}
        onChange={(event) => onNombreChange(event.target.value)}
        className="mt-8 h-40 w-full rounded-lg border border-concrete bg-canvas-white px-9 text-11 font-semibold text-ash-graphite outline-none focus:border-plain-green"
        aria-label="Nombre visible del menú"
      />
      <div className="mt-8 flex items-center gap-8">
        <div className="grid h-40 w-40 shrink-0 place-items-center overflow-hidden rounded-lg border border-concrete bg-vanilla-cream">
          {logoUrl ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img src={logoUrl} alt="Logo actual" className="h-full w-full object-cover" />
          ) : (
            <span className="material-symbols-outlined text-18 text-stone">image</span>
          )}
        </div>
        <label className="flex h-40 flex-1 cursor-pointer items-center justify-center gap-5 rounded-lg border border-concrete bg-canvas-white px-8 text-10 font-semibold text-ash-graphite transition-colors hover:border-ash-graphite hover:bg-ghost-fog">
          <span className="material-symbols-outlined text-15">upload</span>
          {logoUrl ? "Cambiar imagen" : "Subir imagen"}
          <input type="file" accept="image/png,image/jpeg" onChange={onLogo} className="hidden" />
        </label>
        {logoUrl && (
          <button
            type="button"
            onClick={onRemoveLogo}
            className="grid h-40 w-40 shrink-0 place-items-center rounded-lg border border-concrete text-stone transition-colors hover:border-alert-red hover:text-alert-red"
            aria-label="Quitar logo"
            title="Quitar logo"
          >
            <span className="material-symbols-outlined text-17">delete</span>
          </button>
        )}
      </div>
    </div>
  );
}

function AparienciaTab({
  sucursal,
  tenant,
  onTenantUpdate,
}: {
  sucursal: Sucursal | null;
  tenant: Tenant | null;
  onTenantUpdate?: (t: Tenant) => void;
}) {
  const esPro = tenant?.plan === "pro";
  const [modalUpgradeOpen, setModalUpgradeOpen] = useState(false);

  const storageKey = `mesa-click:apariencia:${tenant?.id ?? "sin-tenant"}:${sucursal?.id ?? "sin-sucursal"}`;
  const [apariencia, setApariencia] = useState(() =>
    leerAparienciaGuardada(storageKey, aparienciaDefault(sucursal, tenant)),
  );
  const [guardando, setGuardando] = useState(false);
  const [ok, setOk] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const aparienciaHidratadaKey = useRef<string | null>(null);
  const saltarGuardadoInicial = useRef(false);
  const { nombreVisible, color, colorCategoria, colorAccion, estilo, logoUrl, tipoFuente, mostrarMarcaAgua } = apariencia;

  useEffect(() => {
    const fallback = aparienciaDefault(sucursal, tenant);
    const guardada = leerAparienciaGuardada(storageKey, fallback);
    aparienciaHidratadaKey.current = storageKey;
    saltarGuardadoInicial.current = true;
    // eslint-disable-next-line react-hooks/set-state-in-effect -- re-hidrata desde localStorage al cambiar de sucursal; leerlo en el render rompe la hidratación
    setApariencia(guardada);
    setOk(false);
    setError(null);
  }, [storageKey, sucursal?.id, tenant?.id]);

  useEffect(() => {
    if (aparienciaHidratadaKey.current !== storageKey || saltarGuardadoInicial.current) {
      saltarGuardadoInicial.current = false;
      return;
    }
    guardarAparienciaLocal(storageKey, apariencia);
  }, [storageKey, apariencia]);

  const handleGuardar = async () => {
    setGuardando(true);
    setError(null);
    setOk(false);
    try {
      guardarAparienciaLocal(storageKey, apariencia);
      if (tenant) {
        const actualizado = await api.actualizarMiTenant({
          nombre_fantasia: nombreVisible.trim() || undefined,
          color_primario: esColorHex(color) ? color : undefined,
          color_categoria: esPro && esColorHex(colorCategoria) ? colorCategoria : undefined,
          color_accion: esPro && esColorHex(colorAccion) ? colorAccion : undefined,
          estilo_visual: estilo,
          logo_url: logoUrl || undefined,
          tipo_fuente: esPro ? tipoFuente : undefined,
          mostrar_marca_agua: esPro ? mostrarMarcaAgua : undefined,
        });
        if (onTenantUpdate) onTenantUpdate(actualizado);
      }
      setOk(true);
    } catch (err: unknown) {
      setError(getErrorMessage(err, "Error al guardar la apariencia del menú."));
    } finally {
      setGuardando(false);
    }
  };

  const cargarLogo = (file?: File | null) => {
    if (!file) return;
    if (!["image/png", "image/jpeg"].includes(file.type)) {
      setError("Elegí un logo PNG o JPG.");
      return;
    }
    if (file.size > 1024 * 1024) {
      setError("El logo no puede superar 1 MB.");
      return;
    }
    setError(null);
    const reader = new FileReader();
    reader.onload = () => {
      if (typeof reader.result !== "string") return;
      setApariencia((prev) => ({ ...prev, logoUrl: reader.result as string }));
      setOk(false);
    };
    reader.readAsDataURL(file);
  };

  const onLogo = (e: React.ChangeEvent<HTMLInputElement>) => {
    cargarLogo(e.target.files?.[0]);
    e.target.value = "";
  };

  const colorPrincipal = esColorHex(color) ? color : COLOR_DEFAULT;
  const colorCategorias = esColorHex(colorCategoria) ? colorCategoria : colorPrincipal;
  const colorBotonAccion = esColorHex(colorAccion) ? colorAccion : colorPrincipal;

  return (
    <div className="flex flex-col space-y-16">
      <div className="order-2">
        <Card titulo="Ajustes generales">
        {/* ─── Personalización Avanzada Pro (US-71) ─── */}
        <div className="pt-16 border-t border-concrete space-y-14">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-8">
              <span className="text-12 font-mono text-ash-graphite font-semibold uppercase tracking-wider">
                Personalización de Marca
              </span>
              <span className="rounded-full bg-ash-graphite text-canvas-white px-8 py-2 text-10 font-mono font-semibold uppercase">
                Pro
              </span>
            </div>
            {!esPro && (
              <button
                type="button"
                onClick={() => setModalUpgradeOpen(true)}
                className="text-11 font-medium text-ash-graphite hover:underline flex items-center gap-4 cursor-pointer"
              >
                <span className="material-symbols-outlined text-14 text-stone">lock</span>
                Desbloquear con Pro
              </button>
            )}
          </div>

          <div>
            {/* Tipografía de la carta */}
            <Campo label="Tipografía de la carta digital">
              {esPro ? (
                <div className="grid grid-cols-1 gap-8 sm:grid-cols-2" role="radiogroup" aria-label="Tipografía de la carta digital">
                  {FUENTES_CARTA.map((fuente) => {
                    const activa = tipoFuente === fuente.value;
                    return (
                      <button
                        key={fuente.value}
                        type="button"
                        role="radio"
                        aria-checked={activa}
                        onClick={() => {
                          setApariencia((prev) => ({ ...prev, tipoFuente: fuente.value }));
                          setOk(false);
                        }}
                        className={`flex min-h-56 items-center justify-between rounded-lg border px-12 py-8 text-left transition-colors ${
                          activa
                            ? "border-plain-green bg-plain-green/5 text-ash-graphite"
                            : "border-concrete bg-canvas-white text-sage-green hover:border-stone"
                        }`}
                        style={{ fontFamily: fuente.family }}
                      >
                        <span>
                          <strong className="block text-14 font-semibold">{fuente.label}</strong>
                          <small className="mt-1 block text-10 opacity-75">{fuente.detail}</small>
                        </span>
                        <span className={`material-symbols-outlined text-18 ${activa ? "text-plain-green" : "text-concrete"}`} aria-hidden="true">
                          {activa ? "check_circle" : "circle"}
                        </span>
                      </button>
                    );
                  })}
                </div>
              ) : (
                <div
                  onClick={() => setModalUpgradeOpen(true)}
                  className="h-44 w-full px-12 rounded-lg border border-concrete/70 bg-ghost-fog/40 flex items-center justify-between cursor-pointer group"
                >
                  <span className="text-13 text-sage-green" style={{ fontFamily: "var(--font-inter), sans-serif" }}>Inter (Moderna)</span>
                  <div className="flex items-center gap-4 text-stone group-hover:text-ash-graphite">
                    <span className="text-10 font-mono uppercase font-semibold">Pro</span>
                    <span className="material-symbols-outlined text-16">lock</span>
                  </div>
                </div>
              )}
            </Campo>
          </div>

          {/* Toggle Marca de Agua */}
          <div className="pt-2">
            <div
              className={`p-14 rounded-xl border flex flex-col sm:flex-row sm:items-center sm:justify-between gap-12 transition-colors ${
                esPro
                  ? "border-concrete bg-canvas-white"
                  : "border-concrete/70 bg-ghost-fog/30 cursor-pointer"
              }`}
              onClick={!esPro ? () => setModalUpgradeOpen(true) : undefined}
            >
              <div className="space-y-2">
                <div className="flex items-center gap-6">
                  <p className="text-13 font-semibold text-ash-graphite">
                    Marca de agua &quot;Potenciado por fudd&quot;
                  </p>
                  {!esPro && (
                    <span className="material-symbols-outlined text-16 text-stone">lock</span>
                  )}
                </div>
                <p className="text-12 text-sage-green max-w-lg">
                  {esPro
                    ? "Activá o desactivá el crédito de fudd al pie de la carta digital de tus comensales."
                    : "En el plan Free, la carta incluye un pie discreto 'Potenciado por fudd'. Actualizá a Pro para una experiencia 100% marca blanca."}
                </p>
              </div>

              <div className="shrink-0 flex items-center gap-8">
                {esPro ? (
                  <label className="flex items-center gap-8 cursor-pointer select-none">
                    <input
                      type="checkbox"
                      checked={mostrarMarcaAgua}
                      onChange={(e) => {
                        setApariencia((prev) => ({ ...prev, mostrarMarcaAgua: e.target.checked }));
                        setOk(false);
                      }}
                      style={{ minHeight: 20, minWidth: 20 }}
                      className="h-20 min-h-0 w-20 min-w-0 rounded border-concrete accent-plain-green cursor-pointer"
                    />
                    <span className="text-12 font-medium text-ash-graphite">
                      {mostrarMarcaAgua ? "Visible" : "Oculta (Marca blanca)"}
                    </span>
                  </label>
                ) : (
                  <button
                    type="button"
                    onClick={() => setModalUpgradeOpen(true)}
                    className="h-40 px-12 rounded-lg border border-ash-graphite bg-canvas-white text-11 font-semibold text-ash-graphite hover:bg-ghost-fog transition-colors cursor-pointer"
                  >
                    Quitar marca de agua
                  </button>
                )}
              </div>
            </div>
          </div>
        </div>

        {error && <p className="text-12 text-alert-red">{error}</p>}
        <div className="flex items-center gap-16 pt-8">
          <PillPrimaria onClick={handleGuardar} disabled={guardando}>
            {guardando ? "Guardando..." : "Guardar apariencia"}
          </PillPrimaria>
          <Guardado visible={ok} />
        </div>

        <div className="mt-16 rounded-xl border border-concrete bg-ghost-fog/35 px-14 py-12">
          <p className="text-12 font-semibold text-ash-graphite">Editá la identidad directamente sobre la vista previa</p>
          <p className="mt-3 text-11 leading-relaxed text-sage-green">
            El nombre, el logo y los colores se modifican en los controles conectados al teléfono y se actualizan en tiempo real.
          </p>
        </div>

        </Card>
      </div>

      <div className="order-1 overflow-hidden rounded-2xl border border-concrete bg-canvas-white p-16 sm:p-20 lg:p-28">
        <div className="mb-18 text-center">
          <p className="font-mono text-[12px] uppercase tracking-[0.08em] text-sage-green">Vista previa del menú</p>
          <p className="mt-3 text-11 text-stone">Editá en tiempo real y recorré Carta, Mi pedido y Cuenta dentro del teléfono.</p>
        </div>

        <div className="grid items-center gap-18 lg:grid-cols-[minmax(250px,1fr)_300px_minmax(250px,1fr)] lg:gap-12 xl:gap-20">
          <div className="order-2 space-y-18 lg:order-1">
            <div className="flex items-center gap-8">
              <EditorIdentidadEnVivo
                nombre={nombreVisible}
                logoUrl={logoUrl}
                onNombreChange={(value) => {
                  setApariencia((prev) => ({ ...prev, nombreVisible: value }));
                  setOk(false);
                }}
                onLogo={onLogo}
                onRemoveLogo={() => {
                  setApariencia((prev) => ({ ...prev, logoUrl: "" }));
                  setOk(false);
                }}
              />
              <span className="hidden h-px min-w-16 flex-1 bg-concrete lg:block" />
              <span className="hidden h-8 w-8 shrink-0 rounded-full bg-ash-graphite lg:block" />
            </div>
            <div className="flex items-center gap-8 lg:pt-8">
              <EditorColorEnVivo
                label="Color principal"
                detalle="Carrito y acentos generales."
                value={color}
                onChange={(value) => {
                  setApariencia((prev) => ({ ...prev, color: value }));
                  setOk(false);
                }}
              />
              <span className="hidden h-px min-w-16 flex-1 bg-concrete lg:block" />
              <span className="hidden h-8 w-8 shrink-0 rounded-full lg:block" style={{ background: colorPrincipal }} />
            </div>
          </div>

          <div className="order-1 mx-auto w-full max-w-[300px] lg:order-2">
          <div className="mb-10 flex items-center justify-center gap-8">
            <span className="mr-2 text-10 font-mono uppercase tracking-[0.08em] text-sage-green">Estilo</span>
            {(["claro", "oscuro"] as const).map((op) => (
              <button
                key={op}
                type="button"
                onClick={() => {
                  setApariencia((prev) => ({ ...prev, estilo: op }));
                  setOk(false);
                }}
                className={`h-40 rounded-lg border px-12 text-11 font-semibold capitalize transition-colors ${
                  estilo === op
                    ? "border-ash-graphite bg-ash-graphite text-canvas-white"
                    : "border-concrete bg-canvas-white text-ash-graphite hover:border-ash-graphite"
                }`}
              >
                {op}
              </button>
            ))}
          </div>
          <AparienciaMenuPreview
            nombre={nombreVisible || "Tu negocio"}
            logoUrl={logoUrl}
            colorPrincipal={colorPrincipal}
            colorCategorias={colorCategorias}
            colorAccion={colorBotonAccion}
            estilo={estilo}
            tipoFuente={tipoFuente}
            mostrarMarcaAgua={mostrarMarcaAgua}
          />
        </div>

          <div className="order-3 space-y-18">
            <div className="flex items-center gap-8 lg:pb-8">
              <span className="hidden h-8 w-8 shrink-0 rounded-full lg:block" style={{ background: colorCategorias }} />
              <span className="hidden h-px min-w-16 flex-1 bg-concrete lg:block" />
              <EditorColorEnVivo
                label="Color de categorías"
                detalle={esPro ? "Categoría seleccionada e íconos." : "Probalo en vivo. Requiere Pro para guardar."}
                value={colorCategoria}
                pro
                onChange={(value) => {
                  setApariencia((prev) => ({ ...prev, colorCategoria: value }));
                  setOk(false);
                }}
              />
            </div>
            <div className="flex items-center gap-8 lg:pt-8">
              <span className="hidden h-8 w-8 shrink-0 rounded-full lg:block" style={{ background: colorBotonAccion }} />
              <span className="hidden h-px min-w-16 flex-1 bg-concrete lg:block" />
              <EditorColorEnVivo
                label="Botón Agregar"
                detalle={esPro ? "Acciones de producto y llamados principales." : "Probalo en vivo. Requiere Pro para guardar."}
                value={colorAccion}
                pro
                onChange={(value) => {
                  setApariencia((prev) => ({ ...prev, colorAccion: value }));
                  setOk(false);
                }}
              />
            </div>
          </div>
        </div>
      </div>

      <UpgradeModal
        isOpen={modalUpgradeOpen}
        onClose={() => setModalUpgradeOpen(false)}
        recurso="personalizacion"
      />
    </div>
  );
}

/* ─────────────────────────── tab: Equipo ─────────────────────────── */

const ROLES = [
  ["Admin", "Acceso completo a configuración, carta, mesas, sucursales y equipo."],
  ["Encargado", "Gestiona la operación diaria: carta, mesas y pedidos de su sucursal."],
  ["Mozo / Recepcionista", "Ve pedidos en vivo, cambia estados y atiende solicitudes de cuenta."],
];

function EquipoTab() {
  return (
    <div className="grid lg:grid-cols-[1fr_300px] gap-16 items-start">
      <div>
        <EquipoSection embedded />
      </div>
      <aside className="space-y-16 rounded-xl border border-concrete bg-canvas-white p-20">
        <p className="text-13 font-bold text-ash-graphite">Roles disponibles</p>
        {ROLES.map(([r, d]) => (
          <div key={r}>
            <p className="text-13 font-semibold text-ash-graphite">{r}</p>
            <p className="text-12 text-sage-green mt-2 leading-normal">{d}</p>
          </div>
        ))}
      </aside>
    </div>
  );
}

/* ─────────────────────────── tab: Sucursales ─────────────────────────── */

const DIAS = [
  ["lu", "lunes"],
  ["ma", "martes"],
  ["mi", "miercoles"],
  ["ju", "jueves"],
  ["vi", "viernes"],
  ["sa", "sabado"],
  ["do", "domingo"],
] as const;

type Turno = { apertura: string; cierre: string };

function parseHorarios(raw?: string): { abiertos: Set<string>; turnos: Turno[] } {
  const abiertos = new Set<string>();
  let turnos: Turno[] = [{ apertura: "08:00", cierre: "00:00" }];
  if (!raw) return { abiertos: new Set(DIAS.map((d) => d[1])), turnos };
  try {
    const obj = JSON.parse(raw) as Record<string, { abierto?: boolean; tramos?: Turno[] }>;
    DIAS.forEach(([, key]) => {
      if (obj[key]?.abierto) abiertos.add(key);
    });
    const primer = DIAS.map((d) => d[1]).find((k) => obj[k]?.tramos?.length);
    if (primer && obj[primer]?.tramos) turnos = obj[primer]!.tramos as Turno[];
  } catch {
    return { abiertos: new Set(DIAS.map((d) => d[1])), turnos };
  }
  return { abiertos, turnos };
}

function serializeHorarios(abiertos: Set<string>, turnos: Turno[]): string {
  const out: Record<string, { abierto: boolean; tramos: Turno[] }> = {};
  DIAS.forEach(([, key]) => {
    out[key] = { abierto: abiertos.has(key), tramos: turnos };
  });
  return JSON.stringify(out);
}

function SucursalesTab({
  tenant,
  sucursales,
  setSucursales,
  selId,
  setSelId,
  onIrAMercadoPago,
  onIrAPlanes,
}: {
  tenant: Tenant | null;
  sucursales: Sucursal[];
  setSucursales: React.Dispatch<React.SetStateAction<Sucursal[]>>;
  selId: string;
  setSelId: (id: string) => void;
  onIrAMercadoPago?: () => void;
  onIrAPlanes?: () => void;
}) {
  const sel = useMemo(() => sucursales.find((s) => s.id === selId) ?? null, [sucursales, selId]);
  const horariosIniciales = parseHorarios(sel?.horarios);

  const [nuevaSucursalNombre, setNuevaSucursalNombre] = useState("");
  const [nuevaSucursalWhatsapp, setNuevaSucursalWhatsapp] = useState("");
  const [creandoSucursal, setCreandoSucursal] = useState(false);
  const [errorCrearSucursal, setErrorCrearSucursal] = useState<string | null>(null);

  const handleCrearSucursalPro = async () => {
    if (!nuevaSucursalNombre.trim()) return;
    setCreandoSucursal(true);
    setErrorCrearSucursal(null);
    try {
      const nueva = await api.crearSucursal({
        tenant_id: tenant?.id,
        nombre: nuevaSucursalNombre.trim(),
        whatsapp: nuevaSucursalWhatsapp.trim() || undefined,
      });
      setSucursales((prev) => [...prev, nueva]);
      setSelId(nueva.id);
      setNuevaSucursalNombre("");
      setNuevaSucursalWhatsapp("");
    } catch (err) {
      setErrorCrearSucursal(getErrorMessage(err, "No se pudo crear la sucursal."));
    } finally {
      setCreandoSucursal(false);
    }
  };

  const [form, setForm] = useState(() => ({
    nombre: sel?.nombre ?? "",
    whatsapp: sel?.whatsapp ?? "",
    email: sel?.email ?? "",
  }));
  const [abiertos, setAbiertos] = useState<Set<string>>(() => horariosIniciales.abiertos);
  const [turnos, setTurnos] = useState<Turno[]>(() =>
    horariosIniciales.turnos.length ? horariosIniciales.turnos : [{ apertura: "08:00", cierre: "00:00" }],
  );
  const [mpModo, setMpModo] = useState<"heredar" | "propia" | "deshabilitar">(() => {
    if (sel?.mp_activo === false) return "deshabilitar";
    if (sel?.mp_activo === true || (sel?.mp_access_token && sel.mp_access_token.trim() !== "")) return "propia";
    return "heredar";
  });
  const [mpAccessToken, setMpAccessToken] = useState(sel?.mp_access_token ?? "");
  const [mpPublicKey, setMpPublicKey] = useState(sel?.mp_public_key ?? "");
  const [guardando, setGuardando] = useState(false);
  const [msg, setMsg] = useState("");

  const toggleDia = (key: string) =>
    setAbiertos((prev) => {
      const n = new Set(prev);
      if (n.has(key)) n.delete(key);
      else n.add(key);
      return n;
    });

  const guardar = async () => {
    if (!sel) return;
    setGuardando(true);
    setMsg("");
    try {
      let mpPayload: Partial<Sucursal> = {};
      if (mpModo === "heredar") {
        mpPayload = { mp_activo: null, mp_access_token: null, mp_public_key: null };
      } else if (mpModo === "deshabilitar") {
        mpPayload = { mp_activo: false, mp_access_token: null, mp_public_key: null };
      } else {
        mpPayload = {
          mp_activo: true,
          mp_access_token: mpAccessToken.trim() || null,
          mp_public_key: mpPublicKey.trim() || null,
        };
      }

      const payload: Partial<Sucursal> = {
        nombre: form.nombre,
        whatsapp: form.whatsapp,
        email: form.email,
        horarios: serializeHorarios(abiertos, turnos),
        ...mpPayload,
      };
      const actualizada = await api.actualizarSucursal(sel.id, payload);
      setSucursales((prev) => prev.map((s) => (s.id === sel.id ? { ...s, ...actualizada } : s)));
      setMsg("Sucursal guardada.");
    } catch {
      setMsg("No se pudo guardar en el servidor. Revisá la conexión o el soporte de horarios en la API.");
    } finally {
      setGuardando(false);
    }
  };

  return (
    <div className="grid items-start gap-16 xl:grid-cols-[240px_minmax(0,1fr)_260px]">
      {/* lista */}
      <Card titulo="Sucursales actuales">
        <div className="space-y-8">
          {sucursales.map((s) => (
            <button
              key={s.id}
              onClick={() => setSelId(s.id)}
              className={`w-full rounded-lg border px-12 py-10 text-left transition-colors ${
                s.id === selId
                  ? "border-ash-graphite bg-ghost-fog"
                  : "border-concrete hover:border-stone hover:bg-ghost-fog/50"
              }`}
            >
              <p className="text-13 font-semibold text-ash-graphite">{s.nombre}</p>
              <div className="flex items-center gap-6 mt-2">
                <span className="text-11 text-sage-green">{s.activa ? "Abierto" : "Cerrado"}</span>
                {s.mp_activo === false ? (
                  <span className="text-10 rounded bg-concrete px-4 py-1 text-sage-green font-mono">MP inactivo</span>
                ) : s.mp_activo === true ? (
                  <span className="text-10 rounded bg-plain-green/15 px-4 py-1 text-plain-green font-mono font-medium">MP propio</span>
                ) : (
                  <span className="text-10 rounded bg-ghost-fog px-4 py-1 text-sage-green font-mono">MP negocio</span>
                )}
              </div>
            </button>
          ))}
          {sucursales.length === 0 && (
            <p className="text-12 text-sage-green">Sin sucursales cargadas.</p>
          )}
        </div>
      </Card>

      {/* datos */}
      <Card titulo="Datos de la sucursal">
        {sel ? (
          <>
            <div className="grid sm:grid-cols-2 gap-12">
              <Campo label="Nombre de sucursal">
                <input
                  className={INPUT}
                  value={form.nombre}
                  onChange={(e) => setForm((f) => ({ ...f, nombre: e.target.value }))}
                />
              </Campo>
              <Campo label="WhatsApp / Teléfono">
                <input
                  className={INPUT}
                  value={form.whatsapp}
                  onChange={(e) => setForm((f) => ({ ...f, whatsapp: e.target.value }))}
                />
              </Campo>
            </div>
            <Campo label="Email de contacto">
              <input
                type="email"
                className={INPUT}
                value={form.email}
                onChange={(e) => setForm((f) => ({ ...f, email: e.target.value }))}
              />
            </Campo>

            <div>
              <span className="block text-11 font-mono text-sage-green uppercase tracking-wider mb-6">
                Días abiertos
              </span>
              <div className="flex flex-wrap gap-6">
                {DIAS.map(([abrev, key]) => (
                  <button
                    key={key}
                    onClick={() => toggleDia(key)}
                    className={`h-44 w-44 rounded-lg border text-11 font-mono uppercase ${
                      abiertos.has(key)
                        ? "bg-ash-graphite text-canvas-white border-ash-graphite"
                        : "border-concrete text-sage-green hover:border-ash-graphite"
                    }`}
                  >
                    {abrev}
                  </button>
                ))}
              </div>
            </div>

            <div>
              <span className="block text-11 font-mono text-sage-green uppercase tracking-wider mb-6">
                Horario para días abiertos
              </span>
              <div className="space-y-8">
                <div className="flex flex-col md:flex-row md:items-start gap-12 md:justify-between">
                  <div className="space-y-8">
                    {turnos.map((t, i) => (
                      <div key={i} className="flex flex-wrap items-center gap-8">
                        <input
                          type="time"
                          value={t.apertura}
                          onChange={(e) =>
                            setTurnos((prev) =>
                              prev.map((x, j) => (j === i ? { ...x, apertura: e.target.value } : x)),
                            )
                          }
                          className="h-44 w-120 max-w-full rounded-lg border border-concrete bg-canvas-white px-10 text-13 font-mono outline-none focus:border-system-black"
                        />
                        <span className="text-12 text-sage-green">a</span>
                        <input
                          type="time"
                          value={t.cierre}
                          onChange={(e) =>
                            setTurnos((prev) =>
                              prev.map((x, j) => (j === i ? { ...x, cierre: e.target.value } : x)),
                            )
                          }
                          className="h-44 w-120 max-w-full rounded-lg border border-concrete bg-canvas-white px-10 text-13 font-mono outline-none focus:border-system-black"
                        />
                        {turnos.length > 1 && (
                          <button
                            onClick={() => setTurnos((prev) => prev.filter((_, j) => j !== i))}
                            className="grid h-44 w-44 place-items-center rounded-lg text-12 text-alert-red hover:bg-warm-pink/20"
                            title="Quitar turno"
                          >
                            ✕
                          </button>
                        )}
                      </div>
                    ))}
                  </div>

                  {turnos.length < 2 && (
                  <button
                    onClick={() => setTurnos((prev) => [...prev, { apertura: "16:00", cierre: "00:00" }])}
                    className="h-44 rounded-lg border border-concrete px-12 text-11 font-semibold text-ash-graphite hover:border-stone hover:bg-ghost-fog md:mt-4 md:shrink-0"
                  >
                    + Agregar segundo turno
                  </button>
                  )}
                </div>
              </div>
            </div>

            <div className="pt-16 border-t border-concrete space-y-12">
              <div className="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-8">
                <div>
                  <span className="block text-11 font-mono text-sage-green uppercase tracking-wider mb-2">
                    Cobro con Mercado Pago en esta sucursal
                  </span>
                  <p className="text-12 text-sage-green">
                    Podés heredar la cuenta configurada a nivel negocio, asignar credenciales propias para esta sucursal o deshabilitar el cobro digital aquí.
                  </p>
                </div>
                {onIrAMercadoPago && (
                  <button
                    type="button"
                    onClick={onIrAMercadoPago}
                    className="h-40 shrink-0 rounded-lg border border-concrete bg-ghost-fog/40 px-12 text-11 font-semibold text-ash-graphite hover:border-ash-graphite hover:bg-canvas-white transition-colors cursor-pointer"
                  >
                    Ver panel de Mercado Pago →
                  </button>
                )}
              </div>

              <div className="space-y-8">
                <label className="flex items-center gap-8 cursor-pointer select-none">
                  <input
                    type="radio"
                    name="mp_modo"
                    value="heredar"
                    checked={mpModo === "heredar"}
                    onChange={() => setMpModo("heredar")}
                    style={{ minHeight: 20, minWidth: 20 }}
                    className="h-20 min-h-0 w-20 min-w-0 shrink-0 accent-plain-green cursor-pointer"
                  />
                  <span className="text-13 text-ash-graphite font-medium">
                    Heredar configuración del negocio (recomendado)
                  </span>
                </label>

                <label className="flex items-center gap-8 cursor-pointer select-none">
                  <input
                    type="radio"
                    name="mp_modo"
                    value="propia"
                    checked={mpModo === "propia"}
                    onChange={() => setMpModo("propia")}
                    style={{ minHeight: 20, minWidth: 20 }}
                    className="h-20 min-h-0 w-20 min-w-0 shrink-0 accent-plain-green cursor-pointer"
                  />
                  <span className="text-13 text-ash-graphite font-medium">
                    Usar cuenta propia de Mercado Pago para esta sucursal
                  </span>
                </label>

                <label className="flex items-center gap-8 cursor-pointer select-none">
                  <input
                    type="radio"
                    name="mp_modo"
                    value="deshabilitar"
                    checked={mpModo === "deshabilitar"}
                    onChange={() => setMpModo("deshabilitar")}
                    style={{ minHeight: 20, minWidth: 20 }}
                    className="h-20 min-h-0 w-20 min-w-0 shrink-0 accent-plain-green cursor-pointer"
                  />
                  <span className="text-13 text-ash-graphite font-medium">
                    Deshabilitar cobro con Mercado Pago en esta sucursal
                  </span>
                </label>
              </div>

              {mpModo === "propia" && (
                <div className="grid sm:grid-cols-2 gap-12 pt-8 animate-in fade-in">
                  <Campo label="Access Token de la sucursal">
                    <input
                      type="password"
                      className={INPUT}
                      value={mpAccessToken}
                      onChange={(e) => setMpAccessToken(e.target.value)}
                      placeholder="TEST-... o APP_USR-..."
                    />
                    <span className="block text-11 text-sage-green mt-4">
                      Los pagos de las mesas de esta sucursal se acreditarán en esta cuenta.
                    </span>
                  </Campo>
                  <Campo label="Public Key de la sucursal (Opcional)">
                    <input
                      className={INPUT}
                      value={mpPublicKey}
                      onChange={(e) => setMpPublicKey(e.target.value)}
                      placeholder="TEST-... o APP_USR-..."
                    />
                    <span className="block text-11 text-sage-green mt-4">
                      Clave pública de la cuenta de esta sucursal.
                    </span>
                  </Campo>
                </div>
              )}
            </div>

            <div className="flex items-center gap-12">
              <PillPrimaria onClick={guardar} disabled={guardando}>
                {guardando ? "Guardando…" : "Guardar sucursal"}
              </PillPrimaria>
              {msg && <span className="text-12 text-sage-green">{msg}</span>}
            </div>
          </>
        ) : (
          <p className="text-13 text-sage-green">Elegí una sucursal de la lista.</p>
        )}
      </Card>

      {/* crear PRO */}
      {tenant?.plan === "pro" ? (
        <aside className="space-y-12 rounded-xl border border-concrete bg-canvas-white p-20">
          <div className="flex items-center gap-8">
            <p className="text-13 font-bold text-ash-graphite">Crear sucursal PRO</p>
            <span className="rounded-full bg-ash-graphite text-canvas-white px-8 py-2 text-10 font-mono font-semibold uppercase">
              Pro
            </span>
          </div>
          <p className="text-12 text-sage-green">Agregá una nueva sucursal física para tu negocio.</p>
          <Campo label="Nombre">
            <input
              className={INPUT}
              placeholder="Sucursal nueva"
              value={nuevaSucursalNombre}
              onChange={(e) => setNuevaSucursalNombre(e.target.value)}
            />
          </Campo>
          <Campo label="Contacto / WhatsApp">
            <input
              className={INPUT}
              placeholder="+54 9 ..."
              value={nuevaSucursalWhatsapp}
              onChange={(e) => setNuevaSucursalWhatsapp(e.target.value)}
            />
          </Campo>
          {errorCrearSucursal && (
            <p className="text-12 text-alert-red">{errorCrearSucursal}</p>
          )}
          <button
            type="button"
            onClick={handleCrearSucursalPro}
            disabled={creandoSucursal || !nuevaSucursalNombre.trim()}
            className="h-44 w-full rounded-lg bg-plain-green px-16 text-12 font-semibold text-canvas-white transition-colors hover:bg-plain-green-muted disabled:cursor-not-allowed disabled:opacity-40 cursor-pointer"
          >
            {creandoSucursal ? "Creando..." : "Crear sucursal"}
          </button>
        </aside>
      ) : (
        <aside className="space-y-12 rounded-xl border border-concrete bg-canvas-white p-20">
          <div className="flex items-center gap-8">
            <p className="text-13 font-bold text-ash-graphite">Crear sucursal PRO</p>
            <span className="material-symbols-outlined text-16 text-stone">lock</span>
          </div>
          <p className="text-12 text-sage-green">
            El plan Free incluye 1 sucursal activa. Para gestionar múltiples sucursales con horarios y menús independientes, actualizá a Pro.
          </p>
          <Campo label="Nombre">
            <input className={INPUT} placeholder="Sucursal nueva" disabled />
          </Campo>
          <Campo label="Contacto">
            <input className={INPUT} placeholder="+54 9 ..." disabled />
          </Campo>
          <button
            type="button"
            onClick={onIrAPlanes}
            className="h-44 w-full rounded-lg border border-ash-graphite bg-ash-graphite text-11 font-semibold text-canvas-white hover:bg-plain-green-muted transition-colors cursor-pointer"
          >
            Solicitar Upgrade Pro
          </button>
        </aside>
      )}
    </div>
  );
}

/* ─────────────────────────── pestaña mercado pago ─────────────────────────── */

function MercadoPagoTab({
  tenant,
  setTenant,
  sucursales,
  setSucursales,
  selId,
  setSelId,
}: {
  tenant: Tenant | null;
  setTenant?: React.Dispatch<React.SetStateAction<Tenant | null>>;
  sucursales: Sucursal[];
  setSucursales: React.Dispatch<React.SetStateAction<Sucursal[]>>;
  selId: string;
  setSelId: (id: string) => void;
}) {
  // Estado negocio (default)
  const [tenantMpActivo, setTenantMpActivo] = useState(tenant?.mp_activo ?? false);
  const [tenantAccessToken, setTenantAccessToken] = useState(tenant?.mp_access_token ?? "");
  const [tenantPublicKey, setTenantPublicKey] = useState(tenant?.mp_public_key ?? "");
  const [mostrarTenantToken, setMostrarTenantToken] = useState(false);
  const [guardandoTenant, setGuardandoTenant] = useState(false);
  const [msgTenant, setMsgTenant] = useState("");
  const [errTenant, setErrTenant] = useState("");

  // Sucursal seleccionada
  const sucursalSel = useMemo(
    () => sucursales.find((s) => s.id === selId) ?? sucursales[0] ?? null,
    [sucursales, selId],
  );
  const [mpModoSucursal, setMpModoSucursal] = useState<"heredar" | "propia" | "deshabilitar">(() => {
    if (sucursalSel?.mp_activo === false) return "deshabilitar";
    if (sucursalSel?.mp_activo === true || (sucursalSel?.mp_access_token && sucursalSel.mp_access_token.trim() !== "")) {
      return "propia";
    }
    return "heredar";
  });
  const [sucursalAccessToken, setSucursalAccessToken] = useState(sucursalSel?.mp_access_token ?? "");
  const [sucursalPublicKey, setSucursalPublicKey] = useState(sucursalSel?.mp_public_key ?? "");
  const [mostrarSucursalToken, setMostrarSucursalToken] = useState(false);
  const [guardandoSucursal, setGuardandoSucursal] = useState(false);
  const [msgSucursal, setMsgSucursal] = useState("");
  const [errSucursal, setErrSucursal] = useState("");

  // Estado de portapapeles
  const [copiadoWebhook, setCopiadoWebhook] = useState(false);
  const [guiaAbierta, setGuiaAbierta] = useState(false);

  useEffect(() => {
    if (!guiaAbierta) return;
    const cerrarConEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") setGuiaAbierta(false);
    };
    window.addEventListener("keydown", cerrarConEscape);
    return () => window.removeEventListener("keydown", cerrarConEscape);
  }, [guiaAbierta]);

  const apiBase = getApiBaseUrl();
  const webhookUrl = `${apiBase}/publica/pago/mercadopago/webhook`;
  const credencialesConfiguradas = tenantAccessToken.trim().length > 0;
  const configuracionCompleta = tenantMpActivo && credencialesConfiguradas;

  const copiarWebhook = async () => {
    try {
      await navigator.clipboard.writeText(webhookUrl);
      setCopiadoWebhook(true);
      setTimeout(() => setCopiadoWebhook(false), 2500);
    } catch {
      const input = document.createElement("textarea");
      input.value = webhookUrl;
      document.body.appendChild(input);
      input.select();
      document.execCommand("copy");
      document.body.removeChild(input);
      setCopiadoWebhook(true);
      setTimeout(() => setCopiadoWebhook(false), 2500);
    }
  };

  const guardarTenantMP = async () => {
    setGuardandoTenant(true);
    setMsgTenant("");
    setErrTenant("");
    try {
      const resp = await api.actualizarMiTenant({
        mp_activo: tenantMpActivo,
        mp_access_token: tenantAccessToken.trim() || undefined,
        mp_public_key: tenantPublicKey.trim() || undefined,
      });
      if (setTenant) setTenant((prev) => (prev ? { ...prev, ...resp } : resp));
      setMsgTenant("Credenciales del negocio guardadas correctamente.");
    } catch (err: unknown) {
      setErrTenant(getErrorMessage(err, "No se pudieron guardar las credenciales."));
    } finally {
      setGuardandoTenant(false);
    }
  };

  const guardarSucursalMP = async () => {
    if (!sucursalSel) return;
    setGuardandoSucursal(true);
    setMsgSucursal("");
    setErrSucursal("");
    try {
      let mpPayload: Partial<Sucursal> = {};
      if (mpModoSucursal === "heredar") {
        mpPayload = { mp_activo: null, mp_access_token: null, mp_public_key: null };
      } else if (mpModoSucursal === "deshabilitar") {
        mpPayload = { mp_activo: false, mp_access_token: null, mp_public_key: null };
      } else {
        mpPayload = {
          mp_activo: true,
          mp_access_token: sucursalAccessToken.trim() || null,
          mp_public_key: sucursalPublicKey.trim() || null,
        };
      }
      const actualizada = await api.actualizarSucursal(sucursalSel.id, mpPayload);
      setSucursales((prev) => prev.map((s) => (s.id === sucursalSel.id ? { ...s, ...actualizada } : s)));
      setMsgSucursal("Configuración de la sucursal actualizada.");
    } catch (err: unknown) {
      setErrSucursal(getErrorMessage(err, "No se pudo actualizar la sucursal."));
    } finally {
      setGuardandoSucursal(false);
    }
  };

  const renderBadgeEntorno = (activo: boolean, token: string) => {
    if (!activo) {
      return (
        <span className="inline-flex items-center gap-4 rounded-full bg-concrete/60 px-8 py-2 text-11 font-mono font-medium text-sage-green">
          <span className="h-6 w-6 rounded-full bg-sage-green/60" />
          Deshabilitado
        </span>
      );
    }
    const t = token.trim();
    if (!t) {
      return (
        <span className="inline-flex items-center gap-4 rounded-full bg-amber-500/10 px-8 py-2 text-11 font-mono font-medium text-amber-700">
          <span className="h-6 w-6 rounded-full bg-amber-500" />
          Sin token
        </span>
      );
    }
    if (t.startsWith("TEST-")) {
      return (
        <span className="inline-flex items-center gap-4 rounded-full bg-blue-500/10 px-8 py-2 text-11 font-mono font-medium text-blue-700">
          <span className="h-6 w-6 rounded-full bg-blue-500" />
          Sandbox (Pruebas)
        </span>
      );
    }
    if (t.startsWith("APP_USR-")) {
      return (
        <span className="inline-flex items-center gap-4 rounded-full bg-plain-green/15 px-8 py-2 text-11 font-mono font-medium text-plain-green">
          <span className="h-6 w-6 rounded-full bg-plain-green" />
          Producción
        </span>
      );
    }
    return (
      <span className="inline-flex items-center gap-4 rounded-full bg-concrete/80 px-8 py-2 text-11 font-mono font-medium text-ash-graphite">
        Token personalizado
      </span>
    );
  };

  return (
    <>
      <div className="grid items-start gap-16 xl:grid-cols-[minmax(0,1fr)_360px]">
        <div className="space-y-16">
          <section className="overflow-hidden rounded-xl border border-concrete bg-canvas-white">
            <div className="flex flex-col gap-10 border-b border-concrete/70 px-16 py-14 sm:flex-row sm:items-center sm:justify-between sm:px-20">
              <div>
                <h3 className="text-15 font-semibold tracking-[-0.01em] text-ash-graphite">Mercado Pago</h3>
                <p className="mt-4 text-12 text-sage-green">Cuenta predeterminada para todos los cobros del negocio.</p>
              </div>
              <div className="flex items-center gap-10">
                {renderBadgeEntorno(tenantMpActivo, tenantAccessToken)}
                <label className="flex cursor-pointer select-none items-center gap-6">
                  <span className="text-12 font-medium text-sage-green">Habilitar</span>
                  <input
                    type="checkbox"
                    checked={tenantMpActivo}
                    onChange={(event) => setTenantMpActivo(event.target.checked)}
                    style={{ minHeight: 20, minWidth: 20 }}
                    className="h-20 min-h-0 w-20 min-w-0 cursor-pointer accent-plain-green"
                  />
                </label>
              </div>
            </div>

            <div className="space-y-16 p-16 sm:p-20">
              <div className="grid gap-12 sm:grid-cols-2">
                <Campo label="Access Token del negocio">
                  <div className="relative">
                    <input
                      type={mostrarTenantToken ? "text" : "password"}
                      className={`${INPUT} pr-80 font-mono text-13`}
                      value={tenantAccessToken}
                      onChange={(event) => setTenantAccessToken(event.target.value)}
                      placeholder="TEST-... o APP_USR-..."
                    />
                    <button
                      type="button"
                      onClick={() => setMostrarTenantToken((actual) => !actual)}
                      className="absolute right-8 top-1/2 -translate-y-1/2 rounded px-8 py-4 text-11 font-medium text-sage-green hover:text-ash-graphite"
                    >
                      {mostrarTenantToken ? "Ocultar" : "Mostrar"}
                    </button>
                  </div>
                  <span className="mt-4 block text-11 text-sage-green">
                    Token privado generado en Mercado Pago Developers.
                  </span>
                </Campo>

                <Campo label="Public Key del negocio · Opcional">
                  <input
                    className={`${INPUT} font-mono text-13`}
                    value={tenantPublicKey}
                    onChange={(event) => setTenantPublicKey(event.target.value)}
                    placeholder="TEST-... o APP_USR-..."
                  />
                  <span className="mt-4 block text-11 text-sage-green">
                    Clave pública asociada a tu aplicación.
                  </span>
                </Campo>
              </div>

              <div className="border-t border-concrete pt-16">
                <div className="flex flex-col gap-10 sm:flex-row sm:items-start sm:justify-between">
                  <div className="min-w-0">
                    <p className="text-14 font-semibold text-ash-graphite">Webhook</p>
                    <p className="mt-4 text-11 text-sage-green">
                      Pegá esta URL en la sección Webhooks de Mercado Pago.
                    </p>
                  </div>
                  <button
                    type="button"
                    onClick={copiarWebhook}
                    className="h-40 shrink-0 rounded-lg border border-concrete bg-canvas-white px-12 text-11 font-semibold text-ash-graphite transition-colors hover:bg-ghost-fog"
                  >
                    {copiadoWebhook ? "URL copiada" : "Copiar URL"}
                  </button>
                </div>
                <div className="mt-10 break-all rounded-lg bg-ghost-fog px-12 py-10 font-mono text-11 text-ash-graphite">
                  {webhookUrl}
                </div>
              </div>

              {errTenant && <p className="text-12 text-alert-red">{errTenant}</p>}
              <div className="flex flex-wrap items-center gap-12">
                <PillPrimaria onClick={guardarTenantMP} disabled={guardandoTenant}>
                  {guardandoTenant ? "Guardando…" : "Guardar cambios"}
                </PillPrimaria>
                {msgTenant && <span className="text-12 font-medium text-success-muted">{msgTenant}</span>}
              </div>
            </div>
          </section>

          <section className="overflow-hidden rounded-xl border border-concrete bg-canvas-white">
            <div className="border-b border-concrete/70 px-16 py-14 sm:px-20">
              <h3 className="text-15 font-semibold tracking-[-0.01em] text-ash-graphite">
                Configuración por sucursal
              </h3>
              <p className="mt-4 text-12 text-sage-green">
                Cada sucursal puede heredar la cuenta general, usar credenciales propias o deshabilitar el cobro digital.
              </p>
            </div>

            <div className="space-y-16 p-16 sm:p-20">
              {sucursales.length > 0 ? (
                <>
                  <div className="flex flex-wrap gap-6">
                    {sucursales.map((sucursal) => (
                      <button
                        key={sucursal.id}
                        type="button"
                        onClick={() => setSelId(sucursal.id)}
                        className={`h-40 rounded-lg border px-12 text-11 font-semibold transition-colors ${
                          sucursal.id === sucursalSel?.id
                            ? "border-ash-graphite bg-ash-graphite text-canvas-white"
                            : "border-concrete bg-canvas-white text-sage-green hover:bg-ghost-fog hover:text-ash-graphite"
                        }`}
                      >
                        {sucursal.nombre}
                      </button>
                    ))}
                  </div>

                  {sucursalSel && (
                    <div className="rounded-xl border border-concrete bg-[#FAFAFA] p-14 sm:p-16">
                      <div className="flex flex-col gap-8 border-b border-concrete/70 pb-12 sm:flex-row sm:items-center sm:justify-between">
                        <div>
                          <p className="text-14 font-semibold text-ash-graphite">{sucursalSel.nombre}</p>
                          <p className="mt-2 text-11 text-sage-green">
                            {sucursalSel.direccion || "Sin dirección cargada"}
                          </p>
                        </div>
                        {renderBadgeEntorno(
                          mpModoSucursal === "deshabilitar"
                            ? false
                            : mpModoSucursal === "propia"
                              ? true
                              : tenantMpActivo,
                          mpModoSucursal === "propia" ? sucursalAccessToken : tenantAccessToken,
                        )}
                      </div>

                      <div className="mt-14 grid gap-8">
                        {[
                          {
                            valor: "heredar" as const,
                            titulo: "Heredar configuración del negocio",
                            detalle: "Usa la cuenta general configurada arriba. Es la opción recomendada.",
                          },
                          {
                            valor: "propia" as const,
                            titulo: "Usar una cuenta propia",
                            detalle: "Los cobros de esta sucursal se acreditarán en otra cuenta de Mercado Pago.",
                          },
                          {
                            valor: "deshabilitar" as const,
                            titulo: "Deshabilitar Mercado Pago",
                            detalle: "Los comensales solo podrán abonar presencialmente.",
                          },
                        ].map((opcion) => (
                          <label
                            key={opcion.valor}
                            className={`flex cursor-pointer items-start gap-10 rounded-lg border p-12 transition-colors ${
                              mpModoSucursal === opcion.valor
                                ? "border-ash-graphite bg-canvas-white"
                                : "border-concrete bg-transparent hover:bg-canvas-white"
                            }`}
                          >
                            <input
                              type="radio"
                              name="mp_modo_sucursal"
                              value={opcion.valor}
                              checked={mpModoSucursal === opcion.valor}
                              onChange={() => setMpModoSucursal(opcion.valor)}
                              style={{ minHeight: 20, minWidth: 20 }}
                              className="mt-2 h-20 min-h-0 w-20 min-w-0 shrink-0 cursor-pointer accent-plain-green"
                            />
                            <span>
                              <span className="block text-12 font-semibold text-ash-graphite">{opcion.titulo}</span>
                              <span className="mt-2 block text-11 leading-relaxed text-sage-green">{opcion.detalle}</span>
                            </span>
                          </label>
                        ))}
                      </div>

                      {mpModoSucursal === "propia" && (
                        <div className="mt-14 grid gap-12 border-t border-concrete pt-14 sm:grid-cols-2">
                          <Campo label="Access Token de la sucursal">
                            <div className="relative">
                              <input
                                type={mostrarSucursalToken ? "text" : "password"}
                                className={`${INPUT} pr-80 font-mono text-13`}
                                value={sucursalAccessToken}
                                onChange={(event) => setSucursalAccessToken(event.target.value)}
                                placeholder="TEST-... o APP_USR-..."
                              />
                              <button
                                type="button"
                                onClick={() => setMostrarSucursalToken((actual) => !actual)}
                                className="absolute right-8 top-1/2 -translate-y-1/2 rounded px-8 py-4 text-11 font-medium text-sage-green hover:text-ash-graphite"
                              >
                                {mostrarSucursalToken ? "Ocultar" : "Mostrar"}
                              </button>
                            </div>
                          </Campo>
                          <Campo label="Public Key · Opcional">
                            <input
                              className={`${INPUT} font-mono text-13`}
                              value={sucursalPublicKey}
                              onChange={(event) => setSucursalPublicKey(event.target.value)}
                              placeholder="TEST-... o APP_USR-..."
                            />
                          </Campo>
                        </div>
                      )}

                      <div className="mt-14 flex flex-wrap items-center gap-12">
                        <PillPrimaria onClick={guardarSucursalMP} disabled={guardandoSucursal}>
                          {guardandoSucursal ? "Guardando…" : "Guardar sucursal"}
                        </PillPrimaria>
                        {msgSucursal && <span className="text-12 font-medium text-success-muted">{msgSucursal}</span>}
                        {errSucursal && <span className="text-12 font-medium text-alert-red">{errSucursal}</span>}
                      </div>
                    </div>
                  )}
                </>
              ) : (
                <p className="text-13 text-sage-green">No hay sucursales registradas todavía.</p>
              )}
            </div>
          </section>
        </div>

        <aside className="overflow-hidden rounded-xl border border-concrete bg-canvas-white">
          <div className="border-b border-concrete p-16">
            <div
              className={`rounded-xl p-16 ${
                configuracionCompleta
                  ? "bg-[#EAF8F2] text-[#087657]"
                  : "bg-[#FFF4DB] text-[#7A4700]"
              }`}
            >
              <div className="flex items-start gap-10">
                <span className="material-symbols-outlined text-20">
                  {configuracionCompleta ? "check_circle" : "pending_actions"}
                </span>
                <div>
                  <p className="text-14 font-semibold">
                    {configuracionCompleta ? "Configuración lista" : "Configuración incompleta"}
                  </p>
                  <p className="mt-4 text-11 leading-relaxed">
                    {configuracionCompleta
                      ? "Mercado Pago está habilitado y tiene credenciales cargadas."
                      : "Completá las credenciales y habilitá el cobro para comenzar a operar."}
                  </p>
                </div>
              </div>
            </div>
          </div>

          <div className="space-y-12 border-b border-concrete p-16">
            <p className="text-10 font-mono font-semibold uppercase tracking-wider text-sage-green">
              Lista de control
            </p>
            {[
              { label: "Webhook disponible", listo: true },
              { label: "Credenciales cargadas", listo: credencialesConfiguradas },
              { label: "Cobro habilitado", listo: tenantMpActivo },
            ].map((item) => (
              <div key={item.label} className="flex items-center justify-between gap-10">
                <span className="text-12 text-sage-green">{item.label}</span>
                <span
                  className={`inline-flex items-center gap-6 text-11 font-semibold ${
                    item.listo ? "text-[#087657]" : "text-sage-green"
                  }`}
                >
                  <span className={`h-8 w-8 rounded-full ${item.listo ? "bg-[#14A77B]" : "bg-stone/60"}`} />
                  {item.listo ? "Listo" : "Pendiente"}
                </span>
              </div>
            ))}
          </div>

          <div className="p-16">
            <p className="text-14 font-semibold text-ash-graphite">¿Necesitás ayuda?</p>
            <p className="mt-4 text-11 leading-relaxed text-sage-green">
              Encontrá tus credenciales, configurá el webhook y probá pagos sin dinero real.
            </p>
            <button
              type="button"
              onClick={() => setGuiaAbierta(true)}
              className="mt-12 flex h-44 w-full items-center justify-center gap-6 rounded-lg border border-concrete bg-canvas-white px-12 text-12 font-semibold text-ash-graphite transition-colors hover:bg-ghost-fog"
            >
              Ver guía de configuración
              <span className="material-symbols-outlined text-16">arrow_forward</span>
            </button>
          </div>
        </aside>
      </div>

      {guiaAbierta && (
        <div className="fixed inset-0 z-[100] flex justify-end bg-system-black/30" role="presentation">
          <button
            type="button"
            aria-label="Cerrar guía"
            className="absolute inset-0 cursor-default"
            onClick={() => setGuiaAbierta(false)}
          />
          <aside
            role="dialog"
            aria-modal="true"
            aria-labelledby="guia-mercadopago-titulo"
            className="relative z-10 flex h-full w-full max-w-[520px] flex-col bg-canvas-white shadow-2xl"
          >
            <header className="flex items-start justify-between gap-12 border-b border-concrete px-20 py-16">
              <div>
                <p className="text-10 font-mono font-semibold uppercase tracking-wider text-sage-green">
                  Mercado Pago
                </p>
                <h3 id="guia-mercadopago-titulo" className="mt-4 text-20 font-semibold text-ash-graphite">
                  Guía de configuración
                </h3>
                <p className="mt-4 text-12 text-sage-green">
                  Seguí estos pasos sin salir de tu configuración.
                </p>
              </div>
              <button
                type="button"
                onClick={() => setGuiaAbierta(false)}
                className="grid h-44 w-44 shrink-0 place-items-center rounded-lg text-sage-green hover:bg-ghost-fog hover:text-ash-graphite"
                aria-label="Cerrar guía de configuración"
              >
                <span className="material-symbols-outlined text-20">close</span>
              </button>
            </header>

            <div className="flex-1 space-y-20 overflow-y-auto p-20">
              <section className="flex gap-12">
                <span className="grid h-32 w-32 shrink-0 place-items-center rounded-full bg-ash-graphite text-12 font-bold text-canvas-white">1</span>
                <div>
                  <h4 className="text-14 font-semibold text-ash-graphite">Copiá tus credenciales</h4>
                  <p className="mt-4 text-12 leading-relaxed text-sage-green">
                    Abrí tu aplicación en Mercado Pago Developers y copiá el Access Token. La Public Key es opcional.
                  </p>
                  <a
                    href="https://www.mercadopago.com.ar/developers/panel/app"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="mt-8 inline-flex h-40 items-center gap-6 rounded-lg border border-concrete px-12 text-11 font-semibold text-ash-graphite hover:bg-ghost-fog"
                  >
                    Abrir Mercado Pago Developers
                    <span className="material-symbols-outlined text-14">open_in_new</span>
                  </a>
                </div>
              </section>

              <section className="flex gap-12">
                <span className="grid h-32 w-32 shrink-0 place-items-center rounded-full bg-ash-graphite text-12 font-bold text-canvas-white">2</span>
                <div className="min-w-0 flex-1">
                  <h4 className="text-14 font-semibold text-ash-graphite">Configurá el webhook</h4>
                  <p className="mt-4 text-12 leading-relaxed text-sage-green">
                    En la sección Webhooks de tu aplicación, pegá esta URL y seleccioná únicamente el evento Pagos (payments).
                  </p>
                  <div className="mt-8 break-all rounded-lg bg-ghost-fog p-10 font-mono text-11 text-ash-graphite">
                    {webhookUrl}
                  </div>
                  <button
                    type="button"
                    onClick={copiarWebhook}
                    className="mt-8 h-40 rounded-lg border border-concrete px-12 text-11 font-semibold text-ash-graphite hover:bg-ghost-fog"
                  >
                    {copiadoWebhook ? "URL copiada" : "Copiar webhook"}
                  </button>
                </div>
              </section>

              <section className="flex gap-12">
                <span className="grid h-32 w-32 shrink-0 place-items-center rounded-full bg-ash-graphite text-12 font-bold text-canvas-white">3</span>
                <div>
                  <h4 className="text-14 font-semibold text-ash-graphite">Probá antes de cobrar</h4>
                  <p className="mt-4 text-12 leading-relaxed text-sage-green">
                    Usá un token que comience con TEST- y una cuenta compradora de prueba. Al cambiar a APP_USR-, los cobros comenzarán a utilizar dinero real.
                  </p>
                  <div className="mt-8 rounded-lg bg-[#EAF3FF] p-12 text-11 leading-relaxed text-[#285F9F]">
                    Las credenciales TEST permiten completar el flujo sin debitar dinero.
                  </div>
                </div>
              </section>

              <section className="border-t border-concrete pt-16">
                <div className="flex items-center justify-between gap-10">
                  <h4 className="text-13 font-semibold text-ash-graphite">URLs de retorno</h4>
                  <span className="rounded-full bg-ghost-fog px-8 py-2 text-10 font-mono text-sage-green">Automático</span>
                </div>
                <p className="mt-6 text-11 leading-relaxed text-sage-green">
                  No necesitás configurar back_urls. fudd genera las rutas hacia la mesa y actualiza el pedido en tiempo real.
                </p>
              </section>
            </div>
          </aside>
        </div>
      )}
    </>
  );
}
