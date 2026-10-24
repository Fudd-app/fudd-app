import React from "react";
import Logo from "@/components/brand/Logo";
import { LandingHeaderActions } from "@/components/landing/SessionCta";

export function LandingBrand() {
  return (
    <a href="#inicio" className="landing-brand" aria-label="Ir al inicio de fudd">
      <Logo mark="full" className="h-28 w-auto" />
    </a>
  );
}

export default function LandingHeader() {
  return (
    <header className="landing-header">
      <LandingBrand />
      <nav className="landing-nav">
        <a href="#como-funciona">Cómo funciona</a>
        <a href="#beneficios">Beneficios</a>
        <a href="#planes">Planes</a>
        <a href="#faq">Preguntas frecuentes</a>
      </nav>
      <LandingHeaderActions />
    </header>
  );
}
