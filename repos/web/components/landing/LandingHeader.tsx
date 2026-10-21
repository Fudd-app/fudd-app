import React from "react";
import Link from "next/link";
import Logo from "@/components/brand/Logo";
import { LandingHeaderActions } from "@/components/landing/SessionCta";

export function LandingBrand() {
  return (
    <Link href="/" className="landing-brand">
      <span><Logo className="w-full h-full" /></span>zelko
    </Link>
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
