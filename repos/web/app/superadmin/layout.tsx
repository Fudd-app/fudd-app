import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "FUDD | Control Tower & Telemetría de Plataforma",
  description: "Torre de control y monitoreo exclusivo para los administradores de la plataforma Mesa CLICK",
};

export default function SuperadminLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="min-h-screen bg-neutral-950 text-neutral-100 antialiased selection:bg-neutral-800 selection:text-white">
      {children}
    </div>
  );
}
