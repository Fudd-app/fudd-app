"use client";

import React, { useEffect } from "react";
import { useValorCliente } from "@/lib/useValorCliente";

const QA_HOST_PATTERN = /(^|[-./])(qa|staging)([-./:]|$)/i;

function isQaFromEnv(): boolean {
  const appEnv = (process.env.NEXT_PUBLIC_APP_ENV || "").trim().toLowerCase();
  if (appEnv === "qa" || appEnv === "staging") {
    return true;
  }

  const appUrl = (process.env.NEXT_PUBLIC_APP_URL || "").trim();
  const apiUrl = (process.env.NEXT_PUBLIC_API_URL || "").trim();
  return QA_HOST_PATTERN.test(appUrl) || QA_HOST_PATTERN.test(apiUrl);
}

function isQaEnCliente(): boolean {
  return isQaFromEnv() || QA_HOST_PATTERN.test(window.location.hostname);
}

export default function QaBanner() {
  // En el servidor solo se conoce el env; en el cliente también el hostname.
  const isQa = useValorCliente(isQaEnCliente, isQaFromEnv);

  useEffect(() => {
    if (isQa) {
      document.documentElement.style.setProperty("--qa-banner-offset", "32px");
    } else {
      document.documentElement.style.removeProperty("--qa-banner-offset");
    }

    return () => {
      document.documentElement.style.removeProperty("--qa-banner-offset");
    };
  }, [isQa]);

  if (!isQa) {
    return null;
  }

  return (
    <div
      role="status"
      aria-label="Aviso de entorno de pruebas QA"
      className="z-50 flex min-h-32 w-full shrink-0 items-center justify-center gap-8 border-b border-fudd-cacao/15 bg-fudd-tomate px-12 py-4 text-center font-inter text-12 font-semibold text-canvas-white shadow-xs print:hidden"
    >
      <span className="inline-flex shrink-0 items-center gap-4 rounded-sm bg-fudd-cacao px-6 py-2 font-mono text-10 font-bold tracking-wider text-canvas-white uppercase">
        <span
          className="h-6 w-6 rounded-full bg-fudd-tomate"
          aria-hidden="true"
        />
        QA
      </span>
      <span className="truncate">
        Ambiente de pruebas (QA)
        <span className="hidden sm:inline">
          {" "}
          — Los cambios, pedidos y pagos realizados aquí no afectan a producción.
        </span>
      </span>
    </div>
  );
}
