import React from "react";
import SessionCta from "@/components/landing/SessionCta";
import LandingIcon from "@/components/landing/LandingIcon";
import { formatearPrecioPlan, PLAN_MONEDA, PLAN_PRUEBA_MESES } from "@/lib/planes";

const FREE = [
  "1 sucursal activa",
  "Hasta 10 mesas con QR",
  "Hasta 30 productos",
  "Pedidos y dashboard en tiempo real",
  "Pedidos y carta sincronizados en tiempo real",
];

const PRO = [
  "Multi-sucursal, mesas y carta ilimitadas",
  "Carga masiva Excel/CSV + ajuste de precios",
  "Pantalla de cocina (KDS)",
  "Métricas y analítica de negocio",
  "Personalización de marca del menú",
];

export default function LandingPricing() {
  return (
    <section className="landing-section" id="planes">
      <div className="section-intro">
        <span className="landing-eyebrow">Planes</span>
        <h2>Elegí el plan según el tamaño de tu operación</h2>
        <p>Todo negocio nuevo recibe {PLAN_PRUEBA_MESES} meses de Pro gratis. Después podés seguir con Free o contratar Pro.</p>
      </div>
      <div className="pricing-grid">
        <article className="plan-card">
          <small>Plan Free</small>
          <strong>$0 <span>/ mes</span></strong>
          <p>Para seguir operando sin cargo cuando termine la prueba.</p>
          <ul>
            {FREE.map((item) => <li key={item}><span><LandingIcon name="check" size={12} /></span>{item}</li>)}
          </ul>
          <SessionCta href="/registro" label="Comenzar gratis" className="landing-cta outline" icon="none" />
        </article>
        <article className="plan-card pro">
          <span className="plan-badge">{PLAN_PRUEBA_MESES} meses gratis</span>
          <small>Plan Pro</small>
          <strong>{formatearPrecioPlan()} <span>{PLAN_MONEDA} / mes</span></strong>
          <p>Todas las funciones Pro, sin cargo durante tus primeros {PLAN_PRUEBA_MESES} meses.</p>
          <ul>
            {PRO.map((item) => <li key={item}><span><LandingIcon name="check" size={12} /></span>{item}</li>)}
          </ul>
          <SessionCta href="/registro" label={`Probar ${PLAN_PRUEBA_MESES} meses gratis`} className="landing-cta light-button" iconSize={16} />
        </article>
      </div>
    </section>
  );
}
