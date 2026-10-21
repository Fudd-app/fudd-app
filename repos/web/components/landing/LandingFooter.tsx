import React from "react";
import Link from "next/link";
import { LandingBrand } from "@/components/landing/LandingHeader";

export default function LandingFooter() {
  return (
    <footer className="landing-footer">
      <div className="landing-footer-brand">
        <LandingBrand />
        <p>Tecnología simple para restaurantes que quieren brindar un mejor servicio.</p>
      </div>
      <div><strong>Producto</strong><a href="#como-funciona">Cómo funciona</a><a href="#beneficios">Beneficios</a><a href="#planes">Planes</a><a href="#faq">Preguntas frecuentes</a></div>
      <div><strong>Compañía</strong><a href="mailto:supportmesaclick@gmail.com">Contacto</a><Link href="/terminos">Términos</Link><Link href="/privacidad">Privacidad</Link></div>
      <div className="footer-status"><span><i /> Todos los sistemas operativos</span><small>© 2026 Zelko · IRESM</small></div>
    </footer>
  );
}
