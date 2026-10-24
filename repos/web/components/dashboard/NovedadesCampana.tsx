"use client";

import { useEffect, useState } from "react";
import NovedadesModal, { TipoPill } from "@/components/dashboard/NovedadesModal";
import {
  RolUsuario,
  VERSION_ACTUAL,
  formatearFechaVersion,
  itemsParaRol,
  marcarVersionVista,
  versionYaVista,
} from "@/lib/novedades";

const MAX_ITEMS_RESUMEN = 3;

interface NovedadesCampanaProps {
  usuarioId: string;
  rol: RolUsuario;
}

export default function NovedadesCampana({ usuarioId, rol }: NovedadesCampanaProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const items = itemsParaRol(VERSION_ACTUAL, rol);
  // Se monta recién con la sesión cargada (solo en el navegador), así que puede leer localStorage al iniciar.
  const [hayNoVistas, setHayNoVistas] = useState(() => items.length > 0 && !versionYaVista(usuarioId));

  useEffect(() => {
    if (!isOpen) return;
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") setIsOpen(false);
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [isOpen]);

  function abrir() {
    setIsOpen(!isOpen);
    if (hayNoVistas) {
      marcarVersionVista(usuarioId);
      setHayNoVistas(false);
    }
  }

  return (
    <div className="relative">
      <button
        type="button"
        onClick={abrir}
        className="relative flex h-44 w-44 items-center justify-center rounded-lg text-ash-graphite transition-colors hover:bg-ghost-fog"
        aria-label={hayNoVistas ? "Notificaciones: hay novedades sin ver" : "Notificaciones"}
        aria-expanded={isOpen}
        aria-haspopup="dialog"
      >
        <span className="material-symbols-outlined text-20" aria-hidden="true">notifications</span>
        {hayNoVistas && (
          <span className="absolute right-10 top-10 h-8 w-8 rounded-full bg-plain-green ring-2 ring-canvas-white" aria-hidden="true" />
        )}
      </button>

      {isOpen && (
        <>
          <div className="fixed inset-0 z-10" onClick={() => setIsOpen(false)} />
          <div
            role="dialog"
            aria-label="Novedades"
            className="absolute right-0 z-20 mt-8 w-[min(320px,calc(100vw-32px))] overflow-hidden rounded-xl border border-concrete bg-canvas-white shadow-xl"
          >
            <div className="border-b border-ghost-fog px-16 py-10">
              <p className="text-13 font-semibold text-ash-graphite">Novedades</p>
              <p className="mt-2 font-mono text-11 text-sage-green">
                v{VERSION_ACTUAL.version} · {formatearFechaVersion(VERSION_ACTUAL.fecha)}
              </p>
            </div>

            {items.length === 0 ? (
              <p className="px-16 py-14 text-12 text-sage-green">No hay novedades para tu rol en esta versión.</p>
            ) : (
              <ul className="space-y-10 px-16 py-12">
                {items.slice(0, MAX_ITEMS_RESUMEN).map((item) => (
                  <li key={item.titulo} className="flex items-start gap-8">
                    <TipoPill tipo={item.tipo} />
                    <span className="text-12 font-medium text-ash-graphite">{item.titulo}</span>
                  </li>
                ))}
              </ul>
            )}

            <div className="border-t border-ghost-fog p-6">
              <button
                type="button"
                onClick={() => {
                  setIsOpen(false);
                  setIsModalOpen(true);
                }}
                className="flex h-40 w-full items-center justify-center gap-6 rounded-lg text-13 font-medium text-ash-graphite transition-colors hover:bg-ghost-fog"
              >
                Ver más
                <span className="material-symbols-outlined text-16" aria-hidden="true">arrow_forward</span>
              </button>
            </div>
          </div>
        </>
      )}

      <NovedadesModal isOpen={isModalOpen} onClose={() => setIsModalOpen(false)} rol={rol} />
    </div>
  );
}
