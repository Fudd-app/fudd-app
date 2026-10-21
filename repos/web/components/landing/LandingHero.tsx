/* eslint-disable @next/next/no-img-element -- imágenes de referencia de la maqueta (Unsplash) */
import React from "react";
import Link from "next/link";
import LandingIcon from "@/components/landing/LandingIcon";
import SessionCta from "@/components/landing/SessionCta";
import { PLAN_PRUEBA_MESES } from "@/lib/planes";

export default function LandingHero() {
  return (
    <>
      <section className="landing-hero">
        <div className="hero-copy">
          <span className="hero-kicker"><i /> El servicio de tu restaurante, conectado</span>
          <h1>Más mesas atendidas.<br /><em>Menos esperas.</em></h1>
          <p>Conectá salón, cocina y comensales en un solo lugar. Pedidos por QR, comandas en vivo y cuentas más simples para que tu equipo se enfoque en atender.</p>
          <div className="hero-actions">
            <SessionCta href="/registro" label={`Empezar ${PLAN_PRUEBA_MESES} meses gratis`} className="landing-cta" />
            <Link href="/mesa/demo" className="landing-demo"><span><LandingIcon name="qr" /></span> Ver experiencia del comensal</Link>
          </div>
          <div className="hero-proof">
            <div className="proof-faces"><span>ML</span><span>RD</span><span>CP</span></div>
            <div><strong>Hecho con restaurantes, para restaurantes</strong><small>Configuración simple · Sin instalaciones</small></div>
          </div>
        </div>
        <div className="hero-visual">
          <div className="hero-photo"><img src="https://images.unsplash.com/photo-1517248135467-4c7edcad34c4?auto=format&fit=crop&w=1200&q=85" alt="Salón de restaurante moderno" /></div>
          <div className="hero-order-card">
            <span className="landing-mini-icon"><LandingIcon name="bell" size={18} /></span>
            <span><small>Pedido listo</small><strong>Mesa 12 · Terraza</strong></span>
            <i><LandingIcon name="check" size={14} /></i>
          </div>
          <div className="hero-metric-card">
            <small>Tiempo promedio</small>
            <strong>−32%</strong>
            <span>en toma de pedidos</span>
            <div><i /><i /><i /><i /><i /></div>
          </div>
          <div className="hero-dots"><span /><span /><span /></div>
        </div>
      </section>

      <section className="trusted-strip">
        <span>Una operación más simple para</span>
        <div><b>CAFETERÍAS</b><b>RESTAURANTES</b><b>CERVECERÍAS</b><b>HOTELERÍA</b><b>FOOD HALLS</b></div>
      </section>
    </>
  );
}
