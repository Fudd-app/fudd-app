// repos/web/components/kds/KDSHeader.tsx
"use client";
import React, { useState, useEffect } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { isAudioEnabled, setAudioEnabled } from "./AudioAlerts";
import { cerrarSesion, Sucursal } from "@/lib/api";

export interface KDSHeaderProps {
  sucursales: Sucursal[];
  sucursalSeleccionadaId: string;
  onCambiarSucursal: (sucursalId: string) => void;
  conectadoSSE: boolean;
  totalComandasActivas: number;
  totalDespachadas: number;
  onAbrirHistorial: () => void;
  loginDestino: string;
  autoPrint?: boolean;
  onToggleAutoPrint?: () => void;
  anchoTicket?: "58mm" | "80mm";
  onCambiarAnchoTicket?: (ancho: "58mm" | "80mm") => void;
}

export function KDSHeader({
  sucursales,
  sucursalSeleccionadaId,
  onCambiarSucursal,
  conectadoSSE,
  totalComandasActivas,
  totalDespachadas,
  onAbrirHistorial,
  loginDestino,
  autoPrint = false,
  onToggleAutoPrint,
  anchoTicket = "80mm",
  onCambiarAnchoTicket,
}: KDSHeaderProps) {
	const router = useRouter();
  const [horaActual, setHoraActual] = useState<string>("");
  const [audioActivo, setAudioActivo] = useState<boolean>(true);
  const [esPantallaCompleta, setEsPantallaCompleta] = useState<boolean>(false);

  // Reloj digital en vivo
  useEffect(() => {
    function actualizarHora() {
      const ahora = new Date();
      setHoraActual(
        ahora.toLocaleTimeString([], {
          hour: "2-digit",
          minute: "2-digit",
          second: "2-digit",
        })
      );
    }
    actualizarHora();
    const interval = setInterval(actualizarHora, 1000);
    return () => clearInterval(interval);
  }, []);

  // Sincronizar estado inicial de sonido
  useEffect(() => {
    const timeoutId = window.setTimeout(() => setAudioActivo(isAudioEnabled()), 0);
    return () => window.clearTimeout(timeoutId);
  }, []);

  // Listener para estado de pantalla completa
  useEffect(() => {
    function onFullscreenChange() {
      setEsPantallaCompleta(Boolean(document.fullscreenElement));
    }
    document.addEventListener("fullscreenchange", onFullscreenChange);
    return () => document.removeEventListener("fullscreenchange", onFullscreenChange);
  }, []);

  function toggleAudio() {
    const nuevoEstado = !audioActivo;
    setAudioActivo(nuevoEstado);
    setAudioEnabled(nuevoEstado);
  }

  async function togglePantallaCompleta() {
    try {
      if (!document.fullscreenElement) {
        await document.documentElement.requestFullscreen();
      } else {
        await document.exitFullscreen();
      }
    } catch (err) {
      console.warn("Error alternando pantalla completa:", err);
    }
  }

  return (
    <header className="sticky top-0 z-30 flex flex-wrap items-center justify-between gap-12 border-b border-concrete bg-canvas-white px-16 py-12 shadow-sm">
      {/* LADO IZQUIERDO: BRAND + SUCURSAL + VOLVER */}
      <div className="flex items-center gap-12">
        <Link
          href="/dashboard"
          className="flex h-44 w-44 items-center justify-center rounded-lg border border-concrete bg-canvas-white text-ash-graphite transition-colors hover:border-stone hover:bg-ghost-fog"
          title="Volver al Dashboard"
          aria-label="Volver al dashboard"
        >
          <span className="material-symbols-outlined text-20">arrow_back</span>
        </Link>

        <div className="flex items-center gap-8">
          <span className="flex h-44 w-44 items-center justify-center rounded-lg bg-plain-green text-canvas-white">
            <span className="material-symbols-outlined text-24">skillet</span>
          </span>
          <div>
            <div className="flex items-center gap-8">
              <h1 className="text-18 font-black tracking-[-0.01em] text-ash-graphite">
                KDS Cocina
              </h1>
              <span className="rounded-full border border-concrete bg-ghost-fog px-6 py-2 text-9 font-bold text-sage-green">
                PRO
              </span>
            </div>
            <p className="mt-2 text-11 font-medium text-sage-green">
              {totalComandasActivas} {totalComandasActivas === 1 ? "comanda activa" : "comandas activas"}
            </p>
          </div>
        </div>

        {/* SELECTOR DE SUCURSAL */}
        {sucursales.length > 1 ? (
          <select
            value={sucursalSeleccionadaId}
            onChange={(e) => onCambiarSucursal(e.target.value)}
            className="h-44 rounded-lg border border-concrete bg-canvas-white px-10 text-13 font-semibold text-ash-graphite outline-none focus:border-plain-green"
          >
            {sucursales.map((s) => (
              <option key={s.id} value={s.id}>
                {s.nombre}
              </option>
            ))}
          </select>
        ) : sucursales.length === 1 ? (
          <span className="hidden h-44 items-center gap-6 rounded-lg border border-concrete bg-ghost-fog px-10 text-12 font-semibold text-ash-graphite sm:inline-flex">
            <span className="material-symbols-outlined text-16 text-sage-green">location_on</span>
            {sucursales[0].nombre}
          </span>
        ) : null}
      </div>

      {/* CENTRO: RELOJ Y ESTADO SSE */}
      <div className="flex items-center gap-10">
        {/* RELOJ DIGITAL */}
        <div className="flex h-44 items-center gap-6 rounded-lg bg-plain-green px-12 font-mono text-18 font-black text-canvas-white shadow-sm">
          <span className="material-symbols-outlined text-20">schedule</span>
          <span>{horaActual || "--:--:--"}</span>
        </div>

        {/* ESTADO EN VIVO SSE */}
        <div
          className={`flex items-center gap-6 rounded-full border px-10 py-4 text-11 font-bold ${
            conectadoSSE
              ? "border-[#B7EAD8] bg-[#EAF8F2] text-[#087657]"
              : "animate-pulse border-[#F3D4A3] bg-[#FFF4DB] text-[#9A5700]"
          }`}
          title={conectadoSSE ? "Conectado al servidor en tiempo real" : "Reconectando con el servidor..."}
        >
          <span
            className={`inline-block h-8 w-8 rounded-full ${
              conectadoSSE ? "animate-pulse bg-[#14A77B]" : "bg-[#F2A51A]"
            }`}
          />
          <span>{conectadoSSE ? "EN VIVO" : "RECONECTANDO"}</span>
        </div>
      </div>

      {/* LADO DERECHO: HERRAMIENTAS Y ACCIONES */}
      <div className="flex items-center gap-8">
        {/* CONFIGURACIÓN IMPRESIÓN TÉRMICA (US-67) */}
        {onCambiarAnchoTicket && (
          <button
            type="button"
            onClick={() => onCambiarAnchoTicket(anchoTicket === "80mm" ? "58mm" : "80mm")}
            className="hidden min-h-44 cursor-pointer items-center gap-4 rounded-lg border border-concrete bg-canvas-white px-10 py-8 font-mono text-11 font-bold text-ash-graphite transition-all hover:border-stone hover:bg-ghost-fog sm:flex"
            title="Alternar ancho de papel térmico (58mm / 80mm)"
          >
            <span className="material-symbols-outlined text-16 text-sage-green">receipt_long</span>
            <span>{anchoTicket}</span>
          </button>
        )}

        {onToggleAutoPrint && (
          <button
            type="button"
            onClick={onToggleAutoPrint}
            className={`flex min-h-44 cursor-pointer items-center gap-6 rounded-lg border px-10 py-8 text-11 font-bold transition-all ${
              autoPrint
                ? "border-[#B7EAD8] bg-[#EAF8F2] text-[#087657] hover:bg-[#DDF4EA]"
                : "border-concrete bg-canvas-white text-sage-green hover:border-stone hover:text-ash-graphite"
            }`}
            title={
              autoPrint
                ? "Auto-impresión térmica ACTIVA al entrar nuevo pedido"
                : "Auto-impresión térmica desactivada (clic para activar)"
            }
          >
            <span className="material-symbols-outlined text-18">print</span>
            <span className="hidden lg:inline">{autoPrint ? "Auto-Print ON" : "Auto-Print OFF"}</span>
          </button>
        )}

        <button
          type="button"
          onClick={() => void cerrarSesion().finally(() => router.push(loginDestino))}
          className="flex h-44 w-44 items-center justify-center rounded-lg border border-concrete text-sage-green hover:bg-ghost-fog"
          title="Cerrar sesión"
        >
          <span className="material-symbols-outlined text-20">logout</span>
        </button>
        {/* BOTÓN HISTORIAL DE DESPACHOS */}
        <button
          type="button"
          onClick={onAbrirHistorial}
          className="relative flex min-h-44 cursor-pointer items-center gap-6 rounded-lg border border-concrete bg-canvas-white px-12 py-8 text-12 font-bold text-ash-graphite transition-all hover:border-stone hover:bg-ghost-fog"
          title="Ver comandas despachadas recientemente"
        >
          <span className="material-symbols-outlined text-18">history</span>
          <span className="hidden md:inline">Despachos</span>
          {totalDespachadas > 0 && (
            <span className="rounded-full bg-[#14A77B] px-6 py-2 text-10 font-black text-white">
              {totalDespachadas}
            </span>
          )}
        </button>

        {/* TOGGLE SONIDO (US-66) */}
        <button
          type="button"
          onClick={toggleAudio}
          className={`flex h-44 w-44 cursor-pointer items-center justify-center rounded-lg border transition-all ${
            audioActivo
              ? "border-[#C9DCF7] bg-[#EAF3FF] text-[#285F9F] hover:bg-[#DDEBFD]"
              : "border-concrete bg-ghost-fog text-sage-green hover:text-ash-graphite"
          }`}
          title={audioActivo ? "Sonido activado (clic para silenciar)" : "Sonido silenciado (clic para activar)"}
        >
          <span className="material-symbols-outlined text-20">{audioActivo ? "notifications_active" : "notifications_off"}</span>
        </button>

        {/* TOGGLE FULLSCREEN (US-65) */}
        <button
          type="button"
          onClick={togglePantallaCompleta}
          className="flex h-44 w-44 cursor-pointer items-center justify-center rounded-lg border border-plain-green bg-plain-green text-canvas-white transition-all hover:bg-plain-green-muted"
          title={esPantallaCompleta ? "Salir de pantalla completa" : "Pantalla completa para cocina"}
        >
          <span className="material-symbols-outlined text-20">{esPantallaCompleta ? "fullscreen_exit" : "fullscreen"}</span>
        </button>
      </div>
    </header>
  );
}
