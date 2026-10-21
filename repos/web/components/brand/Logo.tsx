import React from "react";

interface LogoProps {
  className?: string;
  size?: number;
  mark?: "tile" | "iso";
  ink?: string;
  lens?: string;
}

/**
 * Isotipo fudd:
 * - mark="tile" (default): Icono de app sobre contenedor Albahaca (#12382C) con ligadura dd (#F5F3EC) y lente Tomate (#FF6A3D).
 * - mark="iso": Isotipo sin contenedor (ligadura dd con astas y lente).
 */
export const Logo: React.FC<LogoProps> = ({
  className = "w-24 h-24",
  size,
  mark = "tile",
  ink,
  lens = "#FF6A3D",
}) => {
  if (mark === "iso") {
    const strokeColor = ink ?? "#12382C";
    return (
      <svg
        width={size}
        height={size}
        viewBox="182 1 200 200"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
        className={className}
        role="img"
        aria-label="fudd"
      >
        <g fill="none" stroke={strokeColor} strokeWidth="26">
          <circle cx="237" cy="120" r="37" />
          <path d="M274 20 V120" />
          <circle cx="311" cy="120" r="37" />
          <path d="M348 20 V170" />
        </g>
        <path
          d="M274 86.4 A50 50 0 0 0 274 153.6 A50 50 0 0 0 274 86.4 Z"
          fill={lens}
        />
      </svg>
    );
  }

  const tileInk = ink ?? "#F5F3EC";
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 160 160"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={className}
      role="img"
      aria-label="fudd"
    >
      <rect width="160" height="160" rx="36" fill="#12382C" />
      <g transform="translate(21 21) scale(0.59) translate(-182 -1)">
        <g fill="none" stroke={tileInk} strokeWidth="26">
          <circle cx="237" cy="120" r="37" />
          <path d="M274 20 V120" />
          <circle cx="311" cy="120" r="37" />
          <path d="M348 20 V170" />
        </g>
        <path
          d="M274 86.4 A50 50 0 0 0 274 153.6 A50 50 0 0 0 274 86.4 Z"
          fill={lens}
        />
      </g>
    </svg>
  );
};

export const Wordmark: React.FC<{ className?: string }> = ({
  className = "text-18 font-extrabold tracking-tight text-ash-graphite",
}) => <span className={className}>fudd</span>;

export default Logo;
