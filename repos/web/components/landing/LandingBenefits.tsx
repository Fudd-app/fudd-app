/* eslint-disable @next/next/no-img-element -- imágenes de referencia de la maqueta (Unsplash) */
import React from "react";
import SessionCta from "@/components/landing/SessionCta";
import LandingIcon from "@/components/landing/LandingIcon";

export default function LandingBenefits() {
  return (
    <>
      <section className="landing-section benefits-section" id="beneficios">
        <div className="benefit-visual">
          <img src="https://images.unsplash.com/photo-1485182708500-e8f1f318ba72?auto=format&fit=crop&w=1100&q=84" alt="Personas disfrutando en un restaurante" />
          <div className="benefit-stat"><strong>4.9</strong><span>Experiencia del cliente<br /><b>+24% este mes</b></span></div>
        </div>
        <div className="benefit-copy">
          <span className="landing-eyebrow">Menos fricción, más servicio</span>
          <h2>Tu equipo recupera tiempo para lo que realmente importa</h2>
          <p>Zelko no reemplaza la hospitalidad: elimina las tareas repetitivas que se interponen en el camino.</p>
          <ul>
            <li><span><LandingIcon name="check" /></span><div><strong>Pedidos claros, desde el inicio</strong><p>Menos errores de carga y notas que llegan completas a cocina.</p></div></li>
            <li><span><LandingIcon name="check" /></span><div><strong>Información en tiempo real</strong><p>Todos saben qué está pasando, sin depender de recorridas o gritos.</p></div></li>
            <li><span><LandingIcon name="check" /></span><div><strong>Decisiones basadas en datos</strong><p>Entendé tus tiempos, productos y ventas desde un panel simple.</p></div></li>
          </ul>
          <SessionCta href="/login" label="Explorar el dashboard" className="text-cta" />
        </div>
      </section>

      <section className="quote-section">
        <span className="quote-mark">“</span>
        <blockquote>Antes perdíamos tiempo confirmando comandas entre salón y cocina. Ahora todo el equipo ve lo mismo y podemos enfocarnos en que la gente la pase bien.</blockquote>
        <div><span>FP</span><p><strong>Florencia Paredes</strong><small>Fundadora, Casa Clara</small></p></div>
      </section>
    </>
  );
}
