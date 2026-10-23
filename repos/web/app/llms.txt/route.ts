export function GET(request: Request) {
  const origen = new URL(request.url).origin;
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
