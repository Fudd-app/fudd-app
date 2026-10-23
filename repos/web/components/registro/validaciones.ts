import { ApiError } from "@/lib/api";

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export function emailValido(email: string) {
  return EMAIL_RE.test(email.trim());
}

// Mismas reglas que auth.ValidarPassword en el backend.
export function errorPassword(password: string): string | null {
  if (password.length < 10) return "Usá al menos 10 caracteres.";
  if (!/[A-Za-zÁÉÍÓÚÜÑáéíóúüñ]/.test(password) || !/\d/.test(password)) {
    return "Incluí al menos una letra y un número.";
  }
  return null;
}

export function generarSlug(nombre: string) {
  return nombre
    .trim()
    .toLowerCase()
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/(^-|-$)/g, "");
}

// Mientras se edita a mano se permite un guion al final; la validación final usa SLUG_RE.
export function limpiarSlugEscrito(valor: string) {
  return valor
    .toLowerCase()
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .replace(/[^a-z0-9-]+/g, "-")
    .replace(/-{2,}/g, "-")
    .replace(/^-/, "");
}

export const SLUG_RE = /^[a-z0-9]+(-[a-z0-9]+)*$/;

export const SLUGS_RESERVADOS = new Set([
  "admin",
  "superadmin",
  "platform",
  "dashboard",
  "login",
  "logout",
  "auth",
  "kds",
  "cocina",
  "mesa",
  "mesas",
  "api",
  "public",
  "publica",
  "static",
  "pedidos",
  "pedido",
  "carta",
  "menu",
  "onboarding",
  "registro",
  "terminos",
  "privacidad",
  "ayuda",
  "status",
  "health",
  "configuracion",
]);

export function esSlugReservado(slug: string): boolean {
  return SLUGS_RESERVADOS.has(slug.trim().toLowerCase());
}

// Lee nombre y correo del ID token de Google solo para precompletar el registro;
// el backend vuelve a verificar la credencial y usa el correo verificado.
export function leerIdentidadGoogle(credential: string): { email: string; nombre: string } {
  try {
    const base64 = credential.split(".")[1].replace(/-/g, "+").replace(/_/g, "/");
    const bytes = Uint8Array.from(atob(base64), (c) => c.charCodeAt(0));
    const payload = JSON.parse(new TextDecoder().decode(bytes));
    return { email: String(payload.email ?? ""), nombre: String(payload.name ?? "") };
  } catch {
    return { email: "", nombre: "" };
  }
}

// Las prevalidaciones (correo y nombre en URL) no bloquean si el servidor no
// responde: POST /tenants vuelve a validar todo de forma autoritativa.
export function esErrorDeConexion(err: unknown) {
  if (err instanceof ApiError && (err.status === 404 || err.status === 408)) return true;
  if (err instanceof TypeError) return true;

  const message = err instanceof Error ? err.message.trim().toLowerCase() : "";
  return message === "failed to fetch" || message.includes("networkerror");
}
