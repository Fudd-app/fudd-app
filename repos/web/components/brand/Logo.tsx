import React from "react";

const ZELKO_PRIMARY = "#7F77DD";
const ZELKO_TINT = "#EEEDFE";

/**
 * Marca Zelko — símbolo Z sobre tile violeta (#7F77DD / #EEEDFE).
 */
export default function Logo({
  className = "w-28 h-28",
  title = "Zelko",
}: {
  className?: string;
  title?: string;
}) {
  return (
    <svg viewBox="0 0 88 88" className={className} role="img" aria-label={title}>
      <rect width="88" height="88" rx="20" fill={ZELKO_PRIMARY} />
      <path
        d="M 24 26 H 64 L 24 62 H 64"
        fill="none"
        stroke={ZELKO_TINT}
        strokeWidth="7"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

/** Logo + palabra, para cabeceras. */
export function Wordmark({
  className = "",
  markClassName = "w-24 h-24",
}: {
  className?: string;
  markClassName?: string;
}) {
  return (
    <span className={`inline-flex items-center gap-8 ${className}`}>
      <Logo className={markClassName} />
      <span className="text-17 font-medium tracking-[-0.03em] lowercase">zelko</span>
    </span>
  );
}
