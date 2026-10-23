"use client";

import React, { useState } from "react";

interface BotonCopiarProps {
  texto: string;
  etiqueta?: string;
}

// Fallback para contextos sin Clipboard API (http, navegadores viejos).
function copiarPorSeleccion(texto: string): boolean {
  const area = document.createElement("textarea");
  area.value = texto;
  area.setAttribute("readonly", "");
  area.style.position = "fixed";
  area.style.opacity = "0";
  document.body.appendChild(area);
  area.select();
  const ok = document.execCommand("copy");
  document.body.removeChild(area);
  return ok;
}

export default function BotonCopiar({ texto, etiqueta = "Copiar" }: BotonCopiarProps) {
  const [copiado, setCopiado] = useState(false);

  async function copiar() {
    let ok = false;
    try {
      await navigator.clipboard.writeText(texto);
      ok = true;
    } catch {
      ok = copiarPorSeleccion(texto);
    }
    if (ok) {
      setCopiado(true);
      setTimeout(() => setCopiado(false), 2000);
    }
  }

  return (
    <button
      type="button"
      onClick={() => void copiar()}
      className="min-h-11 shrink-0 rounded-xl bg-[var(--l-brand)] px-5 text-sm font-bold text-white transition hover:bg-[var(--l-brand-dark)]"
    >
      <span aria-live="polite">{copiado ? "¡Copiada!" : etiqueta}</span>
    </button>
  );
}
