"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import { PLAN_PRUEBA_MESES } from "@/lib/planes";
import LandingIcon from "@/components/landing/LandingIcon";

export const PANEL_LABEL = "Ir a mi panel";

export function useSesionActiva(): boolean {
	const [activa, setActiva] = useState(false);
	useEffect(() => {
		let vigente = true;
		void api.obtenerSesion().then(() => { if (vigente) setActiva(true); }).catch(() => { if (vigente) setActiva(false); });
		return () => { vigente = false; };
	}, []);
	return activa;
}

interface SessionCtaProps {
  href: string;
  label: string;
  className: string;
  icon?: "chevron" | "none";
  iconSize?: number;
  iconFirst?: boolean;
}

// CTA de la landing: para visitantes lleva a registro/login; con sesión activa, al panel.
export default function SessionCta({ href, label, className, icon = "chevron", iconSize = 20, iconFirst = false }: SessionCtaProps) {
  const conSesion = useSesionActiva();
  const destino = conSesion ? "/dashboard" : href;
  const texto = conSesion ? PANEL_LABEL : label;
  const iconNode = icon === "chevron" ? <LandingIcon name="chevron" size={iconSize} /> : null;

  if (iconFirst) {
    return <Link href={destino} className={className}><span>{iconNode}</span> {texto}</Link>;
  }
  return <Link href={destino} className={className}>{texto} {iconNode}</Link>;
}

// Acciones del header: con sesión activa se reemplazan "Ingresar" y la prueba por un único acceso al panel.
export function LandingHeaderActions() {
  const conSesion = useSesionActiva();

  if (conSesion) {
    return (
      <div className="landing-header-actions">
        <Link href="/dashboard" className="landing-cta small">{PANEL_LABEL} <LandingIcon name="chevron" size={16} /></Link>
      </div>
    );
  }
  return (
    <div className="landing-header-actions">
      <Link href="/login" className="landing-login">Ingresar</Link>
      <Link href="/registro" className="landing-cta small">{PLAN_PRUEBA_MESES} meses gratis <LandingIcon name="chevron" size={16} /></Link>
    </div>
  );
}
