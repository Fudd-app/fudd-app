import type { ReactNode } from "react";
import Link from "next/link";
import Logo from "@/components/brand/Logo";
import { LEGAL } from "@/lib/legal";

export type LegalSection = {
  id: string;
  title: string;
  content: ReactNode;
};

type LegalDocumentProps = {
  eyebrow: string;
  title: string;
  description: string;
  sections: LegalSection[];
  alternateHref: "/privacidad" | "/terminos";
  alternateLabel: string;
};

export const legalLinkClass =
  "font-semibold text-ash-graphite underline decoration-concrete underline-offset-4 transition-colors hover:decoration-ash-graphite";

export default function LegalDocument({
  eyebrow,
  title,
  description,
  sections,
  alternateHref,
  alternateLabel,
}: LegalDocumentProps) {
  return (
    <div className="min-h-screen bg-[#F4F6F7] font-inter text-ash-graphite">
      <header className="border-b border-concrete bg-canvas-white">
        <div className="mx-auto flex min-h-64 w-full max-w-[1240px] items-center justify-between gap-16 px-16 sm:px-24">
          <Link href="/" className="inline-flex items-center gap-8 whitespace-nowrap" aria-label="Volver a fudd">
            <Logo mark="full" className="h-28 w-auto" />
            <span className="sr-only">fudd</span>
          </Link>
          <nav className="flex items-center gap-10 whitespace-nowrap text-11 font-semibold sm:gap-16" aria-label="Navegación legal">
            <Link href={alternateHref} className="text-sage-green transition-colors hover:text-ash-graphite">
              {alternateLabel}
            </Link>
            <Link
              href="/"
              className="inline-flex h-40 items-center rounded-lg border border-concrete px-12 text-ash-graphite transition-colors hover:bg-ghost-fog"
            >
              Inicio
            </Link>
          </nav>
        </div>
      </header>

      <main>
        <section className="bg-ash-graphite text-canvas-white">
          <div className="mx-auto w-full max-w-[1240px] px-16 py-48 sm:px-24 sm:py-64 lg:py-72">
            <p className="text-10 font-mono font-semibold uppercase tracking-[0.16em] text-concrete">{eyebrow}</p>
            <h1 className="display mt-10 max-w-[900px] text-44 leading-[0.96] sm:text-64 lg:text-72">{title}</h1>
            <p className="mt-20 max-w-[760px] text-14 leading-7 text-concrete sm:text-16">{description}</p>
            <div className="mt-24 flex flex-wrap gap-x-20 gap-y-8 border-t border-white/15 pt-16 text-11 text-stone">
              <span>Vigente desde el {LEGAL.effectiveDate}</span>
              <span>Jurisdicción principal: {LEGAL.country}</span>
            </div>
          </div>
        </section>

        <div className="mx-auto grid w-full max-w-[1240px] gap-24 px-16 py-32 sm:px-24 lg:grid-cols-[260px_minmax(0,1fr)] lg:py-48">
          <details className="group self-start rounded-xl border border-concrete bg-canvas-white lg:hidden">
            <summary className="flex cursor-pointer list-none items-center justify-between gap-12 px-16 py-12">
              <span className="text-10 font-mono font-semibold uppercase tracking-[0.12em] text-stone">Contenido</span>
              <span className="material-symbols-outlined text-18 transition-transform group-open:rotate-180">expand_more</span>
            </summary>
            <ol className="grid gap-2 border-t border-concrete p-10 sm:grid-cols-2">
              {sections.map((section, index) => (
                <li key={section.id}>
                  <a
                    href={`#${section.id}`}
                    className="flex gap-8 rounded-md px-8 py-6 text-11 leading-5 text-sage-green transition-colors hover:bg-ghost-fog hover:text-ash-graphite"
                  >
                    <span className="font-mono text-stone">{String(index + 1).padStart(2, "0")}</span>
                    <span>{section.title}</span>
                  </a>
                </li>
              ))}
            </ol>
          </details>

          <aside className="hidden self-start rounded-xl border border-concrete bg-canvas-white p-16 lg:sticky lg:top-20 lg:block">
            <p className="text-10 font-mono font-semibold uppercase tracking-[0.12em] text-stone">Contenido</p>
            <ol className="mt-12 space-y-3">
              {sections.map((section, index) => (
                <li key={section.id}>
                  <a
                    href={`#${section.id}`}
                    className="group flex gap-8 rounded-md px-8 py-6 text-11 leading-5 text-sage-green transition-colors hover:bg-ghost-fog hover:text-ash-graphite"
                  >
                    <span className="font-mono text-stone">{String(index + 1).padStart(2, "0")}</span>
                    <span>{section.title}</span>
                  </a>
                </li>
              ))}
            </ol>
          </aside>

          <article className="overflow-hidden rounded-xl border border-concrete bg-canvas-white">
            <div className="border-b border-concrete bg-ghost-fog/55 px-16 py-16 sm:px-28">
              <div className="flex items-start gap-10">
                <span className="material-symbols-outlined mt-1 text-20">info</span>
                <div>
                  <p className="text-12 font-semibold">Documento público y de lectura obligatoria</p>
                  <p className="mt-4 text-11 leading-5 text-sage-green">
                    Este documento busca explicar las reglas con claridad. Las normas imperativas y los derechos de consumidores y titulares de datos prevalecen sobre cualquier disposición incompatible.
                  </p>
                </div>
              </div>
            </div>

            <div className="divide-y divide-concrete">
              {sections.map((section, index) => (
                <section
                  id={section.id}
                  key={section.id}
                  className="scroll-mt-20 px-16 py-24 sm:px-28 sm:py-30"
                >
                  <div className="flex items-start gap-12">
                    <span className="mt-1 shrink-0 font-mono text-11 text-stone">{String(index + 1).padStart(2, "0")}</span>
                    <div className="min-w-0 flex-1">
                      <h2 className="text-18 font-semibold tracking-[-0.02em] sm:text-20">{section.title}</h2>
                      <div className="mt-12 space-y-12 text-13 leading-7 text-deep-forest sm:text-14">
                        {section.content}
                      </div>
                    </div>
                  </div>
                </section>
              ))}
            </div>
          </article>
        </div>
      </main>

      <footer className="border-t border-concrete bg-canvas-white">
        <div className="mx-auto flex w-full max-w-[1240px] flex-col gap-12 px-16 py-28 text-11 text-sage-green sm:flex-row sm:items-center sm:justify-between sm:px-24">
          <p>© 2026 fudd · Hecho en Argentina</p>
          <div className="flex flex-wrap gap-16">
            <a href={`mailto:${LEGAL.supportEmail}`} className="hover:text-ash-graphite">{LEGAL.supportEmail}</a>
            <Link href="/privacidad" className="hover:text-ash-graphite">Privacidad</Link>
            <Link href="/terminos" className="hover:text-ash-graphite">Términos</Link>
          </div>
        </div>
      </footer>
    </div>
  );
}
