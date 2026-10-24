"use client";

import { useEffect } from "react";
import {
  COMMIT_CORTO,
  ETIQUETA_TIPO,
  NOVEDADES,
  RolUsuario,
  VersionNovedades,
  formatearFechaVersion,
  itemsParaRol,
} from "@/lib/novedades";

interface NovedadesModalProps {
  isOpen: boolean;
  onClose: () => void;
  rol: RolUsuario;
}

export default function NovedadesModal({ isOpen, onClose, rol }: NovedadesModalProps) {
  useEffect(() => {
    if (!isOpen) return;
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  const versiones = NOVEDADES.filter((v) => itemsParaRol(v, rol).length > 0);
  const [actual, ...anteriores] = versiones;

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="novedades-modal-title"
      className="fixed inset-0 z-50 flex items-center justify-center p-16 sm:p-24"
    >
      <div onClick={onClose} className="fixed inset-0 bg-ash-graphite/40 backdrop-blur-xs transition-opacity" />

      <div className="relative flex max-h-[90vh] w-full max-w-lg flex-col overflow-hidden rounded-2xl border border-concrete bg-canvas-white font-inter shadow-2xl animate-in fade-in zoom-in-95 duration-200">
        <div className="flex items-start justify-between border-b border-concrete px-20 py-16 sm:px-24">
          <div className="flex items-center gap-12">
            <div className="flex h-40 w-40 shrink-0 items-center justify-center rounded-xl border border-concrete bg-ghost-fog text-ash-graphite">
              <span className="material-symbols-outlined text-20" aria-hidden="true">campaign</span>
            </div>
            <div>
              <span className="inline-block rounded bg-stone/15 px-6 py-1 text-10 font-bold uppercase tracking-wide text-ash-graphite">
                Novedades
              </span>
              <h2 id="novedades-modal-title" className="mt-2 text-16 font-bold text-ash-graphite sm:text-18">
                Qué hay de nuevo en fudd
              </h2>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            aria-label="Cerrar ventana"
            className="flex h-36 w-36 items-center justify-center rounded-lg text-sage-green transition-colors hover:bg-ghost-fog hover:text-ash-graphite"
          >
            ✕
          </button>
        </div>

        <div className="space-y-20 overflow-y-auto px-20 py-20 sm:px-24">
          {actual ? (
            <VersionDetalle version={actual} rol={rol} />
          ) : (
            <p className="text-13 text-sage-green">Todavía no hay novedades para tu rol.</p>
          )}

          {anteriores.length > 0 && (
            <div className="space-y-8 border-t border-concrete pt-16">
              <h3 className="text-12 font-bold uppercase tracking-wider text-ash-graphite">Versiones anteriores</h3>
              {anteriores.map((version) => (
                <details key={version.version} className="group rounded-xl border border-concrete">
                  <summary className="flex cursor-pointer list-none items-center justify-between px-16 py-12 text-13 font-medium text-ash-graphite">
                    <span>
                      v{version.version} · {formatearFechaVersion(version.fecha)}
                    </span>
                    <span className="material-symbols-outlined text-18 text-sage-green transition-transform group-open:rotate-180" aria-hidden="true">
                      expand_more
                    </span>
                  </summary>
                  <div className="border-t border-concrete px-16 py-12">
                    <VersionDetalle version={version} rol={rol} sinEncabezado />
                  </div>
                </details>
              ))}
            </div>
          )}
        </div>

        <div className="flex items-center justify-between gap-10 border-t border-concrete bg-ghost-fog/20 px-20 py-12 sm:px-24">
          <span className="font-mono text-11 text-sage-green">
            {/* La versión y el commit ayudan a soporte a saber qué está usando quien reporta un problema */}
            v{NOVEDADES[0].version}
            {COMMIT_CORTO ? ` · ${COMMIT_CORTO}` : ""}
          </span>
          <button
            type="button"
            onClick={onClose}
            className="flex h-44 items-center justify-center rounded-lg bg-ash-graphite px-20 text-13 font-semibold text-canvas-white transition-all hover:bg-stone active:scale-[0.98]"
          >
            Entendido
          </button>
        </div>
      </div>
    </div>
  );
}

function VersionDetalle({
  version,
  rol,
  sinEncabezado = false,
}: {
  version: VersionNovedades;
  rol: RolUsuario;
  sinEncabezado?: boolean;
}) {
  return (
    <div className="space-y-12">
      {!sinEncabezado && (
        <p className="font-mono text-11 text-sage-green">
          v{version.version} · {formatearFechaVersion(version.fecha)}
        </p>
      )}
      <ul className="space-y-12">
        {itemsParaRol(version, rol).map((item) => (
          <li key={item.titulo} className="space-y-4">
            <div className="flex items-start gap-8">
              <TipoPill tipo={item.tipo} />
              <p className="text-13 font-semibold text-ash-graphite">{item.titulo}</p>
            </div>
            <p className="text-12 leading-relaxed text-sage-green">{item.descripcion}</p>
          </li>
        ))}
      </ul>
    </div>
  );
}

export function TipoPill({ tipo }: { tipo: keyof typeof ETIQUETA_TIPO }) {
  const estilos =
    tipo === "nuevo"
      ? "border-ash-graphite bg-ash-graphite text-canvas-white"
      : "border-concrete bg-ghost-fog text-ash-graphite";
  return (
    <span className={`mt-2 inline-flex shrink-0 rounded-full border px-6 py-1 text-10 font-bold uppercase tracking-wide ${estilos}`}>
      {ETIQUETA_TIPO[tipo]}
    </span>
  );
}
