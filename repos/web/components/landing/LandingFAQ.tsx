"use client";

import React, { useState } from "react";
import SessionCta from "@/components/landing/SessionCta";
import { formatearPrecioPlan, PLAN_MONEDA, PLAN_PRUEBA_MESES } from "@/lib/planes";

const FAQS = [
  {
    question: "¿Necesito instalar equipos especiales?",
    answer: "No. fudd funciona desde cualquier navegador en celulares, tablets y computadoras. Podés empezar con los dispositivos que ya tenés en el local.",
  },
  {
    question: "¿El comensal tiene que descargar una app?",
    answer: "No. Escanea el QR de la mesa y accede directamente a la carta desde su navegador, sin registros obligatorios ni descargas.",
  },
  {
    question: "¿Puedo actualizar precios y marcar platos agotados?",
    answer: "Sí. Los cambios de precio, descripción o disponibilidad se reflejan en tiempo real en todos los celulares que tengan la carta abierta.",
  },
  {
    question: "¿Cómo llegan los pedidos a cocina?",
    answer: "Cada pedido aparece instantáneamente en el KDS de cocina, ordenado por antigüedad y con variantes, notas y alertas claramente visibles.",
  },
  {
    question: "¿Funciona para varias sucursales?",
    answer: "Sí, con el plan Pro. Podés administrar sucursales, cartas, horarios, mesas y equipos desde una misma cuenta, manteniendo cada operación separada.",
  },
  {
    question: "¿Cómo funciona la prueba gratuita?",
    answer: `Al crear tu negocio activamos el plan Pro completo durante ${PLAN_PRUEBA_MESES} meses, sin cargo. Al finalizar podés continuar con el plan Free o contratar Pro por ${formatearPrecioPlan()} ${PLAN_MONEDA} por mes.`,
  },
];

export default function LandingFAQ() {
  const [openFaq, setOpenFaq] = useState(0);

  return (
    <section className="landing-section faq-section" id="faq">
      <div className="faq-intro">
        <span className="landing-eyebrow">Preguntas frecuentes</span>
        <h2>Todo lo que necesitás saber antes de empezar</h2>
        <p>¿Tenés otra pregunta? Creá tu cuenta y probá Pro gratis durante {PLAN_PRUEBA_MESES} meses con tu operación real.</p>
        <SessionCta href="/registro" label={`Probar ${PLAN_PRUEBA_MESES} meses gratis`} className="landing-demo" iconFirst />
      </div>
      <div className="faq-list">
        {FAQS.map((faq, index) => (
          <article className={openFaq === index ? "faq-item open" : "faq-item"} key={faq.question}>
            <button onClick={() => setOpenFaq(openFaq === index ? -1 : index)} aria-expanded={openFaq === index}>
              <span>{faq.question}</span><i>{openFaq === index ? "−" : "+"}</i>
            </button>
            <div><p>{faq.answer}</p></div>
          </article>
        ))}
      </div>
    </section>
  );
}
