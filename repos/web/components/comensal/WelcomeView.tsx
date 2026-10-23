"use client";

import { useState } from "react";
import ComensalIcon from "@/components/menu/ComensalIcon";
import type { MesaPublica } from "@/lib/api";

interface WelcomeViewProps {
  mesa: MesaPublica;
  dinerNames?: string[];
  onJoin: (name: string) => void;
  onJoinAnonymous: () => void;
  isDemo?: boolean;
}

export default function WelcomeView({
  mesa,
  dinerNames = [],
  onJoin,
  onJoinAnonymous,
  isDemo = false,
}: WelcomeViewProps) {
  const [name, setName] = useState("");
  const venueName = mesa.nombre_fantasia?.trim() || mesa.nombre?.trim() || "Tu restaurante";
  const sectorLabel = mesa.sector ? ` · ${mesa.sector}` : "";
  const initialLetter = venueName.slice(0, 2).toUpperCase() || "MC";

  const handleJoin = () => {
    const trimmed = name.trim();
    if (!trimmed) return;
    onJoin(trimmed);
  };

  // Nombres de comensales ya activos en la mesa
  const nombresActivos = dinerNames.filter(Boolean);

  return (
    <div className="welcome-view">
      <div className="welcome-photo">
        {/* eslint-disable-next-line @next/next/no-img-element */}
        <img
          src="https://images.unsplash.com/photo-1517248135467-4c7edcad34c4?auto=format&fit=crop&w=900&q=84"
          alt={venueName}
        />
        {isDemo && (
          <button
            type="button"
            className="guest-demo-exit"
            onClick={() => {
              window.location.assign("/#inicio");
            }}
            aria-label="Volver a la landing"
          >
            <ComensalIcon name="close" size={20} />
          </button>
        )}
        <div className="welcome-overlay">
          <span className="guest-logo">
            {mesa.logo_url ? (
              // eslint-disable-next-line @next/next/no-img-element
              <img src={mesa.logo_url} alt={venueName} />
            ) : (
              initialLetter
            )}
          </span>
          <span>{venueName}</span>
        </div>
      </div>

      <div className="welcome-content">
        <span className="table-chip">
          <ComensalIcon name="table" size={15} /> Mesa {mesa.numero}{sectorLabel}
        </span>

        <h1>Qué bueno tenerte acá</h1>
        <p>Sumate a la mesa para empezar a pedir. No necesitás registrarte ni descargar nada.</p>

        <label className="guest-field">
          <span>¿Cómo te llamás?</span>
          <input
            value={name}
            onChange={(e) => setName(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") handleJoin();
            }}
            placeholder="Tu nombre o apodo"
            autoFocus
            maxLength={60}
          />
        </label>

        <div className="guest-active">
          {nombresActivos.length > 0 ? (
            <>
              <span className="guest-faces">
                {nombresActivos.slice(0, 3).map((n, i) => (
                  <i key={i}>{n.slice(0, 2).toUpperCase()}</i>
                ))}
              </span>
              <span>
                <strong>{nombresActivos.slice(0, 2).join(" y ")}</strong>{" "}
                {nombresActivos.length > 1 ? "ya están pidiendo" : "ya está pidiendo"}
              </span>
            </>
          ) : (
            <>
              <span className="guest-faces">
                <i>👋</i>
              </span>
              <span>Sé el primero en pedir de la mesa</span>
            </>
          )}
        </div>

        <button
          type="button"
          className="guest-primary"
          disabled={!name.trim()}
          onClick={handleJoin}
        >
          Unirme a la mesa <ComensalIcon name="chevron" size={17} />
        </button>

        <button
          type="button"
          className="guest-link"
          onClick={onJoinAnonymous}
        >
          Continuar sin nombre
        </button>

        <small className="secure-note">
          <ComensalIcon name="check" size={14} /> Sesión privada y segura para esta mesa
        </small>
      </div>
    </div>
  );
}
