import type { Metadata, Viewport } from "next";
import { Outfit, DM_Mono, Anton } from "next/font/google";
import localFont from "next/font/local";
import "./globals.css";

const outfit = Outfit({
  variable: "--font-inter",
  weight: ["400", "500", "700", "800"],
  subsets: ["latin"],
  display: "swap",
});

const dmMono = DM_Mono({
  variable: "--font-mono",
  weight: ["400", "500"],
  subsets: ["latin"],
  display: "swap",
});

const anton = Anton({
  variable: "--font-anton",
  weight: "400",
  subsets: ["latin"],
  display: "swap",
});

const materialSymbols = localFont({
  src: "./fonts/material-symbols-outlined.woff2",
  variable: "--font-material-symbols-outlined",
  weight: "400",
  style: "normal",
  display: "block",
  preload: true,
  adjustFontFallback: false,
});

export const viewport: Viewport = {
  width: "device-width",
  initialScale: 1,
  themeColor: "#12382C",
};

export const metadata: Metadata = {
  title: "fudd | Del QR a la mesa — Carta Digital, KDS de Cocina y Gestión Gastronómica",
  description:
    "Del QR a la mesa. Plataforma web PWA para digitalizar pedidos por QR en bares, cafeterías y restaurantes con salón en vivo, KDS de cocina y analítica.",
  metadataBase: new URL("https://mesa-click-web.onrender.com"),
  icons: {
    icon: [{ url: "/icon.svg?v=fudd20261021", type: "image/svg+xml", sizes: "any" }],
    apple: "/mesa-click-logo-120.png",
  },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html
      lang="es"
      className={`${outfit.variable} ${dmMono.variable} ${anton.variable} ${materialSymbols.variable} h-full antialiased`}
    >
      <body className="min-h-full flex flex-col bg-canvas-white text-ash-graphite">
        {children}
      </body>
    </html>
  );
}
