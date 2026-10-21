"use client";

import { useState, type CSSProperties } from "react";
import ComensalIcon, { type ComensalIconName } from "@/components/menu/ComensalIcon";
import { buildMesaTheme } from "@/components/menu/BrandHeader";

type PreviewTab = "carta" | "pedido" | "cuenta";

type AparienciaMenuPreviewProps = {
  nombre: string;
  logoUrl: string;
  colorPrincipal: string;
  colorCategorias: string;
  colorAccion: string;
  estilo: "claro" | "oscuro";
  tipoFuente: string;
  mostrarMarcaAgua: boolean;
};

const PREVIEW_TABS: Array<{
  id: PreviewTab;
  label: string;
  icon: ComensalIconName;
}> = [
  { id: "carta", label: "Carta", icon: "book" },
  { id: "pedido", label: "Mi pedido", icon: "clock" },
  { id: "cuenta", label: "Cuenta", icon: "card" },
];

const PRODUCTOS_DEMO = [
  { nombre: "Latte vainilla", detalle: "Café, leche y vainilla", precio: "$2.800", icono: "local_cafe" },
  { nombre: "Medialuna", detalle: "Manteca artesanal", precio: "$900", icono: "bakery_dining" },
  { nombre: "Tostado completo", detalle: "Jamón y queso", precio: "$3.400", icono: "lunch_dining" },
];

function DemoWatermark({ visible }: { visible: boolean }) {
  if (!visible) return null;

  return (
    <div className="shrink-0 py-[4px] text-center text-[7px] font-semibold tracking-[0.09em] text-[var(--mesa-muted)]">
      POTENCIADO POR FUDD
    </div>
  );
}

function CartaPreview() {
  return (
    <div className="flex h-full min-h-0 flex-col px-[11px] pb-[7px] pt-[9px]">
      <div className="flex h-[30px] shrink-0 items-center gap-[6px] rounded-[9px] border border-[var(--mesa-border)] bg-[var(--mesa-surface)] px-[9px] text-[8px] text-[var(--mesa-muted)]">
        <ComensalIcon name="search" size={12} />
        <span>Buscar en la carta</span>
      </div>

      <div className="mt-[8px] flex shrink-0 gap-[5px] overflow-hidden">
        {[
          { label: "Cafés", active: true },
          { label: "Dulces", active: false },
          { label: "Comidas", active: false },
        ].map((categoria) => (
          <span
            key={categoria.label}
            className="grid h-[27px] shrink-0 place-items-center rounded-full border px-[8px] text-[8px] font-semibold leading-none"
            style={
              categoria.active
                ? {
                    background: "var(--mesa-category)",
                    borderColor: "var(--mesa-category)",
                    color: "var(--mesa-category-contrast)",
                  }
                : {
                    background: "var(--mesa-surface)",
                    borderColor: "var(--mesa-border)",
                    color: "var(--mesa-muted)",
                  }
            }
          >
            {categoria.label}
          </span>
        ))}
      </div>

      <div className="mt-[8px] min-h-0 flex-1 space-y-[6px] overflow-hidden">
        {PRODUCTOS_DEMO.map((producto) => (
          <div
            key={producto.nombre}
            className="flex h-[60px] items-center gap-[7px] rounded-[10px] border border-[var(--mesa-border)] bg-[var(--mesa-surface)] p-[6px]"
          >
            <div
              className="grid h-[46px] w-[46px] shrink-0 place-items-center rounded-[8px]"
              style={{ background: "var(--mesa-primary-soft)" }}
            >
              <span className="material-symbols-outlined text-[18px] text-[var(--mesa-primary)]">
                {producto.icono}
              </span>
            </div>
            <div className="min-w-0 flex-1">
              <p className="truncate text-[9px] font-bold leading-[12px]">{producto.nombre}</p>
              <p className="mt-[1px] truncate text-[7px] leading-[10px] text-[var(--mesa-muted)]">
                {producto.detalle}
              </p>
              <p className="mt-[3px] text-[9px] font-bold leading-[11px]">{producto.precio}</p>
            </div>
            <span
              className="grid h-[28px] w-[28px] shrink-0 place-items-center rounded-full text-[16px] font-medium leading-none"
              style={{ background: "var(--mesa-action)", color: "var(--mesa-action-contrast)" }}
              aria-hidden="true"
            >
              +
            </span>
          </div>
        ))}
      </div>

      <div
        className="mt-[7px] flex h-[36px] shrink-0 items-center justify-between rounded-[12px] px-[11px]"
        style={{ background: "var(--mesa-action)", color: "var(--mesa-action-contrast)" }}
      >
        <span className="flex items-center gap-[6px] text-[8px] font-bold">
          <span
            className="grid h-[20px] w-[20px] place-items-center rounded-full text-[8px]"
            style={{ background: "var(--mesa-action-contrast)", color: "var(--mesa-action)" }}
          >
            2
          </span>
          Ver pedido
        </span>
        <strong className="text-[9px]">$3.700</strong>
      </div>
    </div>
  );
}

function PedidoPreview() {
  return (
    <div className="flex h-full min-h-0 flex-col px-[11px] pb-[8px] pt-[10px]">
      <div className="shrink-0">
        <p className="text-[14px] font-bold leading-[18px]">Mi pedido</p>
        <p className="mt-[2px] text-[8px] text-[var(--mesa-muted)]">Seguí el estado de tu pedido.</p>
      </div>

      <div
        className="mt-[9px] shrink-0 rounded-[11px] border border-[var(--mesa-border)] p-[9px]"
        style={{ background: "var(--mesa-primary-soft)" }}
      >
        <div className="flex items-center gap-[7px]">
          <span
            className="grid h-[30px] w-[30px] shrink-0 place-items-center rounded-full"
            style={{ background: "var(--mesa-primary)", color: "var(--mesa-primary-contrast)" }}
          >
            <ComensalIcon name="kitchen" size={15} />
          </span>
          <div className="min-w-0 flex-1">
            <p className="text-[9px] font-bold">Preparando tu pedido</p>
            <p className="mt-[1px] text-[7px] text-[var(--mesa-muted)]">Pedido #1042 · aprox. 12 min</p>
          </div>
          <span className="h-[7px] w-[7px] rounded-full bg-[var(--mesa-primary)]" />
        </div>

        <div className="mt-[8px] flex items-center">
          {["Recibido", "Preparando", "Listo"].map((paso, index) => (
            <div key={paso} className="flex flex-1 items-center last:flex-none">
              <div className="flex flex-col items-center gap-[3px]">
                <span
                  className="grid h-[18px] w-[18px] place-items-center rounded-full border text-[7px] font-bold"
                  style={
                    index <= 1
                      ? {
                          background: "var(--mesa-primary)",
                          borderColor: "var(--mesa-primary)",
                          color: "var(--mesa-primary-contrast)",
                        }
                      : { borderColor: "var(--mesa-border)", color: "var(--mesa-muted)" }
                  }
                >
                  {index === 0 ? "✓" : index + 1}
                </span>
                <span className="text-[6px] text-[var(--mesa-muted)]">{paso}</span>
              </div>
              {index < 2 && (
                <span
                  className="mb-[10px] h-px flex-1"
                  style={{ background: index === 0 ? "var(--mesa-primary)" : "var(--mesa-border)" }}
                />
              )}
            </div>
          ))}
        </div>
      </div>

      <div className="mt-[8px] min-h-0 flex-1 rounded-[11px] border border-[var(--mesa-border)] bg-[var(--mesa-surface)] p-[9px]">
        <div className="flex items-center justify-between">
          <p className="text-[9px] font-bold">Tu pedido</p>
          <span className="text-[7px] text-[var(--mesa-muted)]">3 productos</span>
        </div>
        <div className="mt-[5px] divide-y divide-[var(--mesa-border)]">
          {[
            ["1× Latte vainilla", "$2.800"],
            ["1× Medialuna", "$900"],
            ["1× Tostado completo", "$3.400"],
          ].map(([nombre, precio]) => (
            <div key={nombre} className="flex items-center justify-between py-[6px] text-[8px]">
              <span className="truncate pr-[8px] text-[var(--mesa-muted)]">{nombre}</span>
              <span className="shrink-0 font-semibold">{precio}</span>
            </div>
          ))}
        </div>
        <div className="mt-[4px] flex items-center justify-between border-t border-[var(--mesa-border)] pt-[7px] text-[10px] font-bold">
          <span>Total</span>
          <span className="text-[var(--mesa-primary)]">$7.100</span>
        </div>
      </div>

      <div
        className="mt-[8px] grid h-[34px] shrink-0 place-items-center rounded-[10px] text-[9px] font-bold"
        style={{ background: "var(--mesa-action)", color: "var(--mesa-action-contrast)" }}
      >
        Agregar más productos
      </div>
    </div>
  );
}

function CuentaPreview() {
  return (
    <div className="flex h-full min-h-0 flex-col px-[11px] pb-[8px] pt-[10px]">
      <div className="shrink-0">
        <p className="text-[14px] font-bold leading-[18px]">Cuenta</p>
        <p className="mt-[2px] text-[8px] text-[var(--mesa-muted)]">Revisá el consumo de la mesa.</p>
      </div>

      <div className="mt-[9px] grid h-[31px] shrink-0 grid-cols-2 rounded-[9px] bg-[var(--mesa-subtle-surface)] p-[2px] text-[7px] font-semibold">
        <span
          className="grid place-items-center rounded-[7px]"
          style={{ background: "var(--mesa-category)", color: "var(--mesa-category-contrast)" }}
        >
          Cuenta de la mesa
        </span>
        <span className="grid place-items-center text-[var(--mesa-muted)]">Por persona</span>
      </div>

      <div className="mt-[8px] min-h-0 flex-1 rounded-[11px] border border-[var(--mesa-border)] bg-[var(--mesa-surface)] p-[9px]">
        <div className="grid grid-cols-[1fr_auto_auto] gap-[9px] border-b border-[var(--mesa-border)] pb-[6px] text-[6px] font-bold uppercase tracking-[0.08em] text-[var(--mesa-muted)]">
          <span>Comensal</span>
          <span>Ítems</span>
          <span>Importe</span>
        </div>
        <div className="divide-y divide-[var(--mesa-border)]">
          {[
            ["J", "Juli", "2", "$3.700"],
            ["T", "Tomás", "1", "$3.400"],
            ["A", "Ana", "1", "$2.800"],
          ].map(([inicial, nombre, items, importe], index) => (
            <div key={nombre} className="grid grid-cols-[1fr_auto_auto] items-center gap-[9px] py-[7px] text-[8px]">
              <span className="flex min-w-0 items-center gap-[6px] font-semibold">
                <span
                  className="grid h-[22px] w-[22px] shrink-0 place-items-center rounded-full text-[7px] font-bold"
                  style={
                    index === 0
                      ? { background: "var(--mesa-primary)", color: "var(--mesa-primary-contrast)" }
                      : { background: "var(--mesa-primary-soft)", color: "var(--mesa-primary)" }
                  }
                >
                  {inicial}
                </span>
                <span className="truncate">{nombre}</span>
              </span>
              <span className="text-[var(--mesa-muted)]">{items}</span>
              <span className="font-semibold">{importe}</span>
            </div>
          ))}
        </div>
        <div className="mt-[3px] flex items-center justify-between border-t border-[var(--mesa-border)] pt-[8px] text-[10px] font-bold">
          <span>Total de la mesa</span>
          <span className="text-[var(--mesa-primary)]">$9.900</span>
        </div>
      </div>

      <div className="mt-[8px] flex h-[30px] shrink-0 items-center gap-[7px] rounded-[9px] border border-[var(--mesa-border)] bg-[var(--mesa-primary-soft)] px-[9px] text-[7px] text-[var(--mesa-muted)]">
        <ComensalIcon name="help" size={13} className="text-[var(--mesa-primary)]" />
        <span>Podés pedir la cuenta cuando quieras.</span>
      </div>

      <div
        className="mt-[7px] grid h-[34px] shrink-0 place-items-center rounded-[10px] text-[9px] font-bold"
        style={{ background: "var(--mesa-action)", color: "var(--mesa-action-contrast)" }}
      >
        Pedir la cuenta
      </div>
    </div>
  );
}

export default function AparienciaMenuPreview({
  nombre,
  logoUrl,
  colorPrincipal,
  colorCategorias,
  colorAccion,
  estilo,
  tipoFuente,
  mostrarMarcaAgua,
}: AparienciaMenuPreviewProps) {
  const [tabActiva, setTabActiva] = useState<PreviewTab>("carta");
  const tema = buildMesaTheme({
    nombre,
    logo_url: logoUrl,
    color_primario: colorPrincipal,
    color_categoria: colorCategorias,
    color_accion: colorAccion,
    estilo_visual: estilo,
    plan: "pro",
    tipo_fuente: tipoFuente,
  });
  const previewFont =
    tipoFuente === "system-ui"
      ? "system-ui, sans-serif"
      : tipoFuente
        ? `${tipoFuente}, sans-serif`
        : "inherit";
  const screenStyle = {
    ...tema.style,
    background: "var(--mesa-background)",
    color: "var(--mesa-text)",
    fontFamily: previewFont,
  } as CSSProperties;

  return (
    <div className="mx-auto w-full max-w-[280px]">
      <div className="mb-10 text-center">
        <p className="text-[9px] font-semibold text-ash-graphite">Probá las vistas del comensal</p>
        <p className="mt-2 text-[8px] text-stone">Contenido ilustrativo · sin acciones reales</p>
      </div>

      <div className="relative mx-auto h-[538px] w-[260px]">
        <span className="absolute -left-[3px] top-[78px] h-[34px] w-[3px] rounded-l bg-[#4a4d50]" />
        <span className="absolute -left-[3px] top-[124px] h-[56px] w-[3px] rounded-l bg-[#4a4d50]" />
        <span className="absolute -right-[3px] top-[114px] h-[76px] w-[3px] rounded-r bg-[#4a4d50]" />

        <div className="h-full w-full rounded-[38px] border border-[#5f6367] bg-[#17191b] p-[6px] shadow-[0_22px_55px_rgba(0,0,0,0.28),inset_0_0_0_1px_rgba(255,255,255,0.16)]">
          <div className="relative flex h-full flex-col overflow-hidden rounded-[32px]" style={screenStyle}>
            <div className="relative flex h-[34px] shrink-0 items-start justify-between px-[16px] pt-[10px] font-sans text-[9px] font-bold leading-none">
              <div className="absolute left-1/2 top-[6px] h-[18px] w-[68px] -translate-x-1/2 rounded-full bg-system-black" />
              <span>9:41</span>
              <div className="flex items-center gap-[4px]">
                <span className="flex h-[8px] items-end gap-px" aria-hidden="true">
                  {[3, 5, 7, 9].map((height) => (
                    <span key={height} className="w-[2px] rounded-full bg-current" style={{ height }} />
                  ))}
                </span>
                <span className="h-[7px] w-[15px] rounded-[2px] border border-current p-px">
                  <span className="block h-full w-[9px] rounded-[1px] bg-current" />
                </span>
              </div>
            </div>

            <div className="flex h-[58px] shrink-0 items-center gap-[9px] border-b border-[var(--mesa-border)] bg-[var(--mesa-header)] px-[13px]">
              <div className="grid h-[38px] w-[38px] shrink-0 place-items-center overflow-hidden rounded-[10px] border border-[var(--mesa-border)] bg-[var(--mesa-surface)]">
                {logoUrl ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img src={logoUrl} alt="" className="h-full w-full object-cover" />
                ) : (
                  <ComensalIcon name="brand" size={20} className="text-[var(--mesa-primary)]" />
                )}
              </div>
              <div className="min-w-0 flex-1">
                <p className="truncate text-[12px] font-bold leading-[15px]">{nombre || "Tu negocio"}</p>
                <p className="mt-[2px] text-[8px] text-[var(--mesa-muted)]">Mesa 12 · Juli</p>
              </div>
              <span
                className="grid h-[25px] shrink-0 place-items-center rounded-full px-[8px] text-[7px] font-bold"
                style={{ background: "var(--mesa-primary-soft)", color: "var(--mesa-primary)" }}
              >
                Abierto
              </span>
            </div>

            <main className="min-h-0 flex-1 overflow-hidden" aria-live="polite">
              {tabActiva === "carta" && <CartaPreview />}
              {tabActiva === "pedido" && <PedidoPreview />}
              {tabActiva === "cuenta" && <CuentaPreview />}
            </main>

            <DemoWatermark visible={mostrarMarcaAgua} />

            <nav
              className="flex h-[58px] shrink-0 border-t border-[var(--mesa-border)] bg-[var(--mesa-header)] px-[4px]"
              aria-label="Vistas simuladas del menú"
            >
              {PREVIEW_TABS.map((tab) => {
                const activa = tab.id === tabActiva;
                return (
                  <button
                    key={tab.id}
                    type="button"
                    onClick={() => setTabActiva(tab.id)}
                    className="relative flex h-full min-h-0 min-w-0 flex-1 flex-col items-center justify-center gap-[3px] rounded-[9px] text-[7px] font-semibold transition-colors"
                    style={{ color: activa ? "var(--mesa-primary)" : "var(--mesa-muted)" }}
                    aria-pressed={activa}
                    aria-label={`Ver simulación de ${tab.label}`}
                  >
                    {activa && (
                      <span
                        className="absolute inset-x-[9px] top-[4px] h-[2px] rounded-full"
                        style={{ background: "var(--mesa-primary)" }}
                      />
                    )}
                    <span className="relative">
                      <ComensalIcon name={tab.icon} size={16} />
                      {tab.id === "pedido" && (
                        <span
                          className="absolute -right-[7px] -top-[5px] grid h-[13px] min-w-[13px] place-items-center rounded-full px-[3px] text-[6px] font-bold"
                          style={{ background: "var(--mesa-primary)", color: "var(--mesa-primary-contrast)" }}
                        >
                          1
                        </span>
                      )}
                    </span>
                    <span>{tab.label}</span>
                  </button>
                );
              })}
            </nav>

            <div className="grid h-[12px] shrink-0 place-items-center bg-[var(--mesa-header)]">
              <span className="h-[3px] w-[76px] rounded-full bg-[var(--mesa-text)] opacity-80" />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
