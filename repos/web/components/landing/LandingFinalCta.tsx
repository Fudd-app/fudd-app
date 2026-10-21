import React from "react";
import SessionCta from "@/components/landing/SessionCta";
import { PLAN_PRUEBA_MESES } from "@/lib/planes";

export default function LandingFinalCta() {
  return (
    <section className="final-cta">
      <div>
        <span className="hero-kicker light"><i /> Empezá hoy</span>
        <h2>Tu próximo servicio puede ser más simple</h2>
        <p>Probá todas las funciones Pro durante {PLAN_PRUEBA_MESES} meses y descubrí una operación más ágil, clara y conectada.</p>
      </div>
      <div>
        <SessionCta href="/registro" label={`Probar ${PLAN_PRUEBA_MESES} meses gratis`} className="landing-cta light-button" />
        <small>Sin tarjeta · Después seguís con Free o elegís Pro</small>
      </div>
    </section>
  );
}
