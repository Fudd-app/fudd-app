import type { NextConfig } from "next";

const apiOrigin = (process.env.API_URL ?? process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080").replace(/\/$/, "");

const nextConfig: NextConfig = {
  reactStrictMode: false,
  output: "standalone",
  // Vercel define VERCEL_GIT_COMMIT_SHA en el build; se expone al cliente para mostrar el commit
  // en Novedades sin depender de "Automatically expose System Environment Variables".
  env: {
    NEXT_PUBLIC_COMMIT_SHA: process.env.VERCEL_GIT_COMMIT_SHA ?? "",
  },
  allowedDevOrigins: ["192.168.4.48", "192.168.4.188", "localhost"],
  turbopack: {
    root: process.cwd(),
  },
  async rewrites() {
    return [
      {
        source: "/api-proxy/:path*",
        destination: `${apiOrigin}/:path*`,
      },
    ];
  },
};

export default nextConfig;
