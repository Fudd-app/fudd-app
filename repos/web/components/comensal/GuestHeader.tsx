"use client";

import ComensalIcon from "@/components/menu/ComensalIcon";
import { BrandMark } from "@/components/menu/BrandHeader";
import type { MesaPublica } from "@/lib/api";

interface GuestHeaderProps {
  mesa: MesaPublica;
  dinerName?: string;
  onOpenCuenta: () => void;
  onEditDiner: () => void;
}

export default function GuestHeader({
  mesa,
  dinerName,
  onOpenCuenta,
  onEditDiner,
}: GuestHeaderProps) {
  const venueName = mesa.nombre_fantasia?.trim() || mesa.nombre?.trim() || "Tu restaurante";
  const sectorLabel = mesa.sector ? ` · ${mesa.sector}` : "";
  const initial = (dinerName?.trim() || "I").slice(0, 1).toUpperCase();

  return (
    <header className="guest-header">
      <div className="guest-identity">
        <BrandMark branding={{ ...mesa, nombre: venueName }} className="guest-header-logo" />
        <div>
          <button
            type="button"
            className="guest-venue"
            onClick={onEditDiner}
            title="Cambiar nombre del comensal"
          >
            {venueName}
          </button>
          <span>
            Mesa {mesa.numero}{sectorLabel}
          </span>
        </div>
      </div>

      <div className="guest-header-actions">
        <button
          type="button"
          aria-label="Ver cuenta y seguimiento"
          title="Ver cuenta y estado"
          onClick={onOpenCuenta}
        >
          <ComensalIcon name="card" size={18} />
        </button>

        <button
          type="button"
          className="guest-avatar"
          onClick={onEditDiner}
          aria-label={`Cambiar nombre (${dinerName || "Invitado"})`}
          title={`Identificado como: ${dinerName || "Invitado"} (tocar para editar)`}
        >
          {initial}
        </button>
      </div>
    </header>
  );
}
