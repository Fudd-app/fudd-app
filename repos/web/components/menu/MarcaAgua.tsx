import React from "react";

export function MarcaAgua({ className = "" }: { className?: string }) {
  return (
    <div
      className={`mx-auto flex items-center justify-center gap-6 rounded-lg border border-dashed border-concrete/90 bg-ghost-fog/50 px-12 py-6 text-center text-11 font-mono text-sage-green select-none ${className}`}
    >
      <span className="text-12">⚡</span>
      <span>
        Carta digitalizada con{" "}
        <strong className="font-bold text-ash-graphite">fudd</strong>
      </span>
    </div>
  );
}

export default MarcaAgua;

