export const PLAN_PRO_PRECIO_MENSUAL_ARS = 30_000;
export const PLAN_PRUEBA_MESES = 3;
export const PLAN_MONEDA = "ARS";

export function formatearPrecioPlan(valor = PLAN_PRO_PRECIO_MENSUAL_ARS): string {
  return new Intl.NumberFormat("es-AR", {
    style: "currency",
    currency: PLAN_MONEDA,
    maximumFractionDigits: 0,
  }).format(valor);
}
