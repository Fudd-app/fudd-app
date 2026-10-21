import type { Metadata, Viewport } from "next";
import { Inter, Geist_Mono, Anton } from "next/font/google";
import localFont from "next/font/local";
import "./globals.css";

const inter = Inter({
  variable: "--font-inter",
  subsets: ["latin"],
  display: "swap",
});

const geistMono = Geist_Mono({
  variable: "--font-mono",
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
  themeColor: "#ffffff",
};

export const metadata: Metadata = {
  title: "Mesa CLICK | Carta Digital QR, KDS de Cocina y Gestión Gastronómica",
  description:
    "Plataforma web PWA para digitalizar pedidos por QR en bares, cafeterías y restaurantes con salón en vivo, KDS de cocina y analítica.",
  metadataBase: new URL("https://mesa-click-web.onrender.com"),
  icons: {
    icon: [{ url: "/icon.svg?v=20261020", type: "image/svg+xml", sizes: "any" }],
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
      className={`${inter.variable} ${geistMono.variable} ${anton.variable} ${materialSymbols.variable} h-full antialiased`}
    >
      <body className="min-h-full flex flex-col bg-canvas-white text-ash-graphite">
        {children}
      </body>
    </html>
  );
}
