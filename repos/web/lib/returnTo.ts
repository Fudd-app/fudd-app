// Valida un returnTo para evitar open redirect. Devuelve una ruta interna normalizada o null.
// Sin runner de tests en el repo; casos esperados:
//   rechaza: "//evil.com", "/\\evil.com", "/\t/evil.com", "/\n/evil.com",
//            "javascript:alert(1)", "https://evil.com", "\\/evil.com", ""
//   acepta:  "/conectar?solicitud=abc", "/dashboard"
export function returnToSeguro(valor: string | null, origin: string): string | null {
  if (!valor) return null;
  if (/[\x00-\x20\x7f]/.test(valor) || valor.includes("\\")) return null;
  if (!valor.startsWith("/") || valor.startsWith("//")) return null;
  try {
    const url = new URL(valor, origin);
    if (url.origin !== origin) return null;
    return url.pathname + url.search + url.hash;
  } catch {
    return null;
  }
}
