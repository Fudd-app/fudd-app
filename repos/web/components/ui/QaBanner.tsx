"use client";

import React, { useEffect, useState } from "react";

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

export default function QaBanner() {
  const [isQa, setIsQa] = useState<boolean>(() => isQaFromEnv());

  useEffect(() => {
    const detected =
      isQaFromEnv() ||
      (typeof window !== "undefined" &&
        QA_HOST_PATTERN.test(window.location.hostname));

    setIsQa(detected);

    if (typeof document !== "undefined") {
      if (detected) {
        document.documentElement.style.setProperty("--qa-banner-offset", "32px");
      } else {
        document.documentElement.style.removeProperty("--qa-banner-offset");
      }
    }

    return () => {
      if (typeof document !== "undefined") {
        document.documentElement.style.removeProperty("--qa-banner-offset");
      }
    };
  }, []);

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
