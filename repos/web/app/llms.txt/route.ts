// Origen fijo de configuración: armar el link con el Host del request permitía
// envenenar la caché pública con un dominio ajeno.
export const dynamic = "force-static";

const origen = (process.env.NEXT_PUBLIC_APP_URL?.trim() || "https://fudd.com.ar").replace(/\/$/, "");

export function GET() {
  const cuerpo = [
    "# Fudd",
    "",
    "> Plataforma para locales gastronómicos: carta digital por QR, pedidos en tiempo real y panel de gestión. Un servidor MCP de solo lectura permite consultar salón, cocina y turno desde un asistente de IA.",
    "",
    "## Docs",
    "",
    `- [Conectar asistentes de IA (MCP)](${origen}/ia.md): cómo conectar y qué tools expone`,
    "",
  ].join("\n");
  return new Response(cuerpo, {
    headers: { "Content-Type": "text/plain; charset=utf-8", "Cache-Control": "public, max-age=3600" },
  });
}
