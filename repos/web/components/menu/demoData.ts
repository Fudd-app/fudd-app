import type { MesaPublica, VariantePublica } from "@/lib/api";

export interface DemoItem {
  id: string;
  nombre: string;
  descripcion: string;
  precio: number;
  tag: string;
  categoria: string;
  foto_url: string;
  disponible: boolean;
  variantes?: VariantePublica[];
}

export const DEMO_MESA: MesaPublica = {
  id: "demo-mesa-14",
  tenant_id: "demo-tenant",
  sucursal_id: "demo-sucursal",
  numero: 14,
  sector: "Terraza",
  estado: "activa",
  cuenta_solicitada: false,
  pago_habilitado: false,
  cuenta_version: 1,
  nombre: "Bajo Limonero",
  nombre_fantasia: "Bajo Limonero",
  logo_url: "/demo/bajo-limonero-logo.png",
  color_primario: "#f06c4f",
  color_categoria: "#182825",
  color_accion: "#f06c4f",
  estilo_visual: "claro",
  plan: "pro",
  tipo_fuente: "Inter",
  mostrar_marca_agua: false,
  mercadopago_habilitado: true,
};

export const DEMO_ITEMS: DemoItem[] = [
  {
    id: "demo-1",
    nombre: "Smash Limonero",
    descripcion: "Doble carne, cheddar, cebolla crispy y salsa de la casa.",
    precio: 9200,
    tag: "Más pedido",
    categoria: "Principales",
    foto_url: "https://images.unsplash.com/photo-1643757343278-5d50309dfa44?auto=format&fit=crop&w=700&q=82",
    disponible: true,
    variantes: [
      {
        id: "var-1",
        articulo_id: "demo-1",
        nombre: "Jugoso",
        precio_adicional: 0,
        grupo: "Punto de cocción",
        seleccion_unica: true,
        orden: 1,
      },
      {
        id: "var-2",
        articulo_id: "demo-1",
        nombre: "A punto",
        precio_adicional: 0,
        grupo: "Punto de cocción",
        seleccion_unica: true,
        orden: 2,
      },
      {
        id: "var-3",
        articulo_id: "demo-1",
        nombre: "Bien cocido",
        precio_adicional: 0,
        grupo: "Punto de cocción",
        seleccion_unica: true,
        orden: 3,
      },
    ],
  },
  {
    id: "demo-2",
    nombre: "Ensalada tibia",
    descripcion: "Vegetales de estación, queso grillado, verdes y almendras.",
    precio: 7800,
    tag: "Vegetariano",
    categoria: "Principales",
    foto_url: "https://images.unsplash.com/photo-1778690103044-88ad0e274e32?auto=format&fit=crop&w=700&q=82",
    disponible: true,
  },
  {
    id: "demo-3",
    nombre: "Pesca del día",
    descripcion: "Filet grillado, crema de coliflor y vegetales crocantes.",
    precio: 12600,
    tag: "Sin TACC",
    categoria: "Principales",
    foto_url: "https://images.unsplash.com/photo-1692197275931-0793e08efcc1?auto=format&fit=crop&w=700&q=82",
    disponible: true,
  },
  {
    id: "demo-4",
    nombre: "Burrata de estación",
    descripcion: "Tomates reliquia, pesto de albahaca, reducción balsámica y focaccia.",
    precio: 8400,
    tag: "Fresco",
    categoria: "Entradas",
    foto_url: "https://images.unsplash.com/photo-1761315631508-eb81f826e6c3?auto=format&fit=crop&w=700&q=82",
    disponible: true,
  },
  {
    id: "demo-5",
    nombre: "Limonada de menta",
    descripcion: "Limón natural, menta fresca del huerto y jengibre.",
    precio: 3900,
    tag: "Bebida",
    categoria: "Bebidas",
    foto_url: "https://images.unsplash.com/photo-1551632436-cbf8dd35adfa?auto=format&fit=crop&w=700&q=82",
    disponible: true,
  },
];
