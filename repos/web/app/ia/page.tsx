import React from "react";
import type { Metadata } from "next";
import Link from "next/link";
import { LandingBrand } from "@/components/landing/LandingHeader";
import LandingFooter from "@/components/landing/LandingFooter";
import BotonCopiar from "@/components/ia/BotonCopiar";
import {
  AL_AUTORIZAR,
  CLIENTES,
  COMO_DESCONECTAR,
  DATOS_VISIBLES,
  EJEMPLOS_PREGUNTAS,
  NOTA_MENUS,
  REQUISITOS,
  comandoConUrl,
  urlMcp,
} from "@/lib/ia/contenido";
import "@/components/landing/landing.css";

export const metadata: Metadata = {
  title: "Conectá Fudd con tu asistente de IA",
  description:
    "Conectá Claude, ChatGPT, Gemini o Perplexity con tu cuenta de Fudd y consultá en tiempo real el salón, la cocina y el turno de tu negocio.",
  alternates: { canonical: "/ia" },
};

const seccion = "mt-14";
const titulo = "text-2xl font-semibold tracking-tight text-[var(--l-ink)]";
const texto = "mt-3 text-base leading-relaxed text-[var(--l-muted)]";

export default function IaPage() {
  const url = urlMcp();

  return (
    <div className="landing font-inter antialiased">
      <header className="landing-header">
        <LandingBrand />
        <Link href="/" className="landing-login">Volver al inicio</Link>
      </header>

      <main className="mx-auto max-w-3xl px-6 pb-20 pt-12 sm:pt-16">
        <p className="text-xs font-bold uppercase tracking-wider text-[var(--l-soft)]">Asistentes de IA</p>
        <h1 className="mt-3 text-4xl font-normal leading-tight tracking-tight text-[var(--l-ink)] sm:text-5xl">
          Preguntale a tu asistente cómo está tu negocio
        </h1>
        <p className="mt-5 text-lg leading-relaxed text-[var(--l-muted)]">
          Conectá Claude, ChatGPT, Gemini o Perplexity con tu cuenta de Fudd y consultá en tiempo real el salón, la cocina y el turno.
        </p>

        <section className="mt-10 rounded-2xl border border-[var(--l-line)] bg-[var(--l-fog)] p-5" aria-labelledby="url-conector">
          <h2 id="url-conector" className="text-sm font-bold text-[var(--l-ink)]">URL del conector</h2>
          <div className="mt-3 flex flex-col gap-3 sm:flex-row">
            <input
              readOnly
              value={url}
              aria-label="URL del conector"
              className="min-h-11 w-full min-w-0 rounded-xl border border-[var(--l-line)] bg-white px-4 font-mono text-sm text-[var(--l-ink)]"
            />
            <BotonCopiar texto={url} />
          </div>
        </section>

        <section className={seccion}>
          <h2 className={titulo}>Qué podés preguntar</h2>
          <ul className="mt-4 space-y-2">
            {EJEMPLOS_PREGUNTAS.map((p) => (
              <li key={p} className="rounded-xl border border-[var(--l-line)] px-4 py-3 text-[var(--l-ink-2)]">«{p}»</li>
            ))}
          </ul>
        </section>

        <section className={seccion}>
          <h2 className={titulo}>Qué datos ve</h2>
          <ul className="mt-4 space-y-2 text-[var(--l-muted)]">
            {DATOS_VISIBLES.map((d) => (
              <li key={d.titulo}><strong className="text-[var(--l-ink)]">{d.titulo}:</strong> {d.detalle}</li>
            ))}
          </ul>
          <p className={texto}>
            <strong className="text-[var(--l-ink)]">Es de solo lectura:</strong> el asistente no puede tomar pedidos ni modificar nada.
          </p>
        </section>

        <section className={seccion}>
          <h2 className={titulo}>Requisitos</h2>
          <ul className="mt-4 list-disc space-y-2 pl-5 text-[var(--l-muted)]">
            {REQUISITOS.map((r) => <li key={r}>{r}</li>)}
          </ul>
        </section>

        <section className={seccion}>
          <h2 className={titulo}>Cómo conectarlo</h2>
          <div className="mt-4 space-y-3">
            {CLIENTES.map((c, i) => {
              const comando = comandoConUrl(c);
              return (
                <details key={c.id} open={i === 0} className="group rounded-2xl border border-[var(--l-line)] bg-white p-5">
                  <summary className="flex min-h-11 cursor-pointer list-none items-center justify-between gap-3 font-semibold text-[var(--l-ink)]">
                    <span>
                      {c.nombre}
                      {c.disponibilidad && <span className="ml-2 text-sm font-normal text-[var(--l-soft)]">{c.disponibilidad}</span>}
                    </span>
                    <span aria-hidden className="text-[var(--l-soft)] transition group-open:rotate-180">⌄</span>
                  </summary>
                  <ol className="mt-3 list-decimal space-y-2 pl-5 text-[var(--l-muted)]">
                    {c.pasos.map((p) => <li key={p}>{p}</li>)}
                  </ol>
                  {comando && (
                    <div className="mt-4 flex flex-col gap-3 sm:flex-row sm:items-center">
                      <pre className="min-w-0 flex-1 overflow-x-auto rounded-xl bg-[var(--l-dark)] p-4 font-mono text-xs text-white"><code>{comando}</code></pre>
                      <BotonCopiar texto={comando} />
                    </div>
                  )}
                  {c.notas?.map((n) => <p key={n} className="mt-3 text-sm text-[var(--l-soft)]">{n}</p>)}
                </details>
              );
            })}
          </div>
          <p className="mt-4 text-sm text-[var(--l-soft)]">{NOTA_MENUS}</p>
        </section>

        <section className={seccion}>
          <h2 className={titulo}>Al autorizar</h2>
          <p className={texto}>{AL_AUTORIZAR}</p>
        </section>

        <section className={seccion}>
          <h2 className={titulo}>Cómo desconectar</h2>
          <p className={texto}>
            {COMO_DESCONECTAR}{" "}
            <Link href="/dashboard" className="font-semibold text-[var(--l-ink)] underline">Ir a mi panel</Link>
          </p>
        </section>

        <p className="mt-14 text-sm text-[var(--l-soft)]">
          <a href="/ia.md" className="underline">Versión para asistentes de IA (Markdown)</a>
        </p>
      </main>

      <LandingFooter />
    </div>
  );
}
