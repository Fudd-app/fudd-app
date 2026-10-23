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
      <div><strong>Producto</strong><Link href="/#como-funciona">Cómo funciona</Link><Link href="/#beneficios">Beneficios</Link><Link href="/#planes">Planes</Link><Link href="/#faq">Preguntas frecuentes</Link><Link href="/ia">Asistentes de IA</Link></div>
      <div><strong>Compañía</strong><a href="mailto:supportmesaclick@gmail.com">Contacto</a><Link href="/terminos">Términos</Link><Link href="/privacidad">Privacidad</Link></div>
      <div className="footer-status"><span><i /> Todos los sistemas operativos</span><small>© 2026 fudd · IRESM</small></div>
    </footer>
  );
}
