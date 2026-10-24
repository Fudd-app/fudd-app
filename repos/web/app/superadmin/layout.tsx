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
    <div className="min-h-screen bg-[#f4f6f7] text-[#18272d] antialiased selection:bg-purple-100 selection:text-purple-900">
      {children}
    </div>
  );
}
