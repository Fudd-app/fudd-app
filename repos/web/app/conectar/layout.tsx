import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Conectar asistente de IA | fudd",
  robots: { index: false },
};

export default function ConectarLayout({ children }: { children: React.ReactNode }) {
  return children;
}
