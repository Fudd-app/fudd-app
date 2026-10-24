import { useSyncExternalStore } from "react";

// Los valores que lee este hook no cambian durante la sesión: no hace falta suscribirse a nada.
const sinSuscripcion = () => () => {};

/**
 * Lee un valor que solo existe en el navegador (window.location, hostname, etc.) sin
 * romper la hidratación: en el servidor y al hidratar usa `getServidor`, y después
 * `getCliente`. Reemplaza el patrón useState + useEffect que marca react-hooks.
 */
export function useValorCliente<T>(getCliente: () => T, getServidor: () => T): T {
  return useSyncExternalStore(sinSuscripcion, getCliente, getServidor);
}
