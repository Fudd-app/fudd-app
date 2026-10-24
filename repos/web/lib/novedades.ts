import data from "@/content/novedades.json";
import type { SesionUsuario } from "@/lib/api";

export type RolUsuario = SesionUsuario["rol"];

// Solo se comunican cambios que el usuario percibe: funcionalidades nuevas
// o cambios en cómo funciona algo que ya usaba. Lo técnico no va acá.
export type TipoNovedad = "nuevo" | "cambio";

export interface ItemNovedad {
  tipo: TipoNovedad;
  titulo: string;
  descripcion: string;
  roles: RolUsuario[];
}

export interface VersionNovedades {
  version: string;
  fecha: string;
  items: ItemNovedad[];
}

// La primera entrada del archivo es la versión publicada actualmente.
export const NOVEDADES = data as VersionNovedades[];
export const VERSION_ACTUAL = NOVEDADES[0];

export const ETIQUETA_TIPO: Record<TipoNovedad, string> = {
  nuevo: "Nuevo",
  cambio: "Cambio",
};

export function itemsParaRol(version: VersionNovedades, rol: RolUsuario): ItemNovedad[] {
  return version.items.filter((item) => item.roles.includes(rol));
}

export function formatearFechaVersion(fecha: string): string {
  const [anio, mes, dia] = fecha.split("-");
  return `${dia}/${mes}/${anio}`;
}

// Vercel expone el commit del deploy; en local no existe y no se muestra.
export const COMMIT_CORTO = process.env.NEXT_PUBLIC_VERCEL_GIT_COMMIT_SHA?.slice(0, 7);

function claveVista(usuarioId: string) {
  return `fudd_novedades_vista_${usuarioId}`;
}

export function versionYaVista(usuarioId: string): boolean {
  try {
    return localStorage.getItem(claveVista(usuarioId)) === VERSION_ACTUAL.version;
  } catch {
    // Sin acceso a localStorage (modo privado, bloqueado) no molestamos con el indicador.
    return true;
  }
}

export function marcarVersionVista(usuarioId: string) {
  try {
    localStorage.setItem(claveVista(usuarioId), VERSION_ACTUAL.version);
  } catch (err) {
    console.warn("No se pudo guardar el estado de novedades:", err);
  }
}
