import React from "react";

const FOREST = "#144336";
const PAPER = "#f7f6f2";

/**
 * Marca Mesa CLICK — arco de mesa + puntero sobre tile verde bosque.
 * El borde del puntero va en el color del tile para "cortar" el arco donde se apoya.
 */
export default function Logo({
  className = "w-28 h-28",
  title = "Mesa CLICK",
}: {
  className?: string;
  title?: string;
}) {
  return (
    <svg viewBox="0 0 100 100" className={className} role="img" aria-label={title}>
      <rect x="2" y="2" width="96" height="96" rx="26" fill={FOREST} />
      <path
        d="M21 74 V36 A8 8 0 0 1 29 28 H70 A8 8 0 0 1 78 36 V60"
        fill="none"
        stroke={PAPER}
        strokeWidth="10.5"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <path
        d="M0 0 V30 L7.5 23.5 L12.5 34 L18 31.5 L13 21.5 H23 Z"
        transform="translate(52 50) scale(.95)"
        fill={PAPER}
        stroke={FOREST}
        strokeWidth="5"
        strokeLinejoin="round"
        paintOrder="stroke"
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
      <span className="text-16 font-bold tracking-tight uppercase">Mesa CLICK</span>
    </span>
  );
}
