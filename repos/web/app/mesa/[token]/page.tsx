"use client";

import { useCallback, useMemo, useReducer, useState, useEffect, useRef, type CSSProperties } from "react";
import { useParams } from "next/navigation";
import { api } from "@/lib/api";
import type { ArticuloPublico, CategoriaPublica, MesaPublica, PedidoAPI, VariantePublica } from "@/lib/api";
import {
  clearComensalIdentity,
  createComensalIdentity,
  getComensalIdentity,
  saveComensalIdentity,
  type ComensalIdentity,
} from "@/lib/comensal";
import ModalNombreComensal from "@/components/comensal/ModalNombreComensal";
import ModalResena from "@/components/menu/ModalResena";
import { buildMesaTheme } from "@/components/menu/BrandHeader";
import type { MesaBranding } from "@/components/menu/BrandHeader";
import MarcaAgua from "@/components/menu/MarcaAgua";
import ComensalIcon from "@/components/menu/ComensalIcon";
import WelcomeView from "@/components/comensal/WelcomeView";
import GuestHeader from "@/components/comensal/GuestHeader";
import GuestFoodCard, { type GuestCardItem } from "@/components/comensal/GuestFoodCard";
import GuestDetailSheet from "@/components/comensal/GuestDetailSheet";
import GuestCartSheet from "@/components/comensal/GuestCartSheet";
import GuestOrderStatusView from "@/components/comensal/GuestOrderStatusView";
import { DEMO_MESA, DEMO_ITEMS } from "@/components/menu/demoData";
import "@/components/menu/comensal.css";

export type CartItem = {
  id: string;
  articuloId: string;
  nombre: string;
  precio: number;
  precioBase?: number;
  cantidad: number;
  nota: string;
  variantes?: VariantePublica[];
  comensalId?: string;
  comensalNombre?: string;
};

export type EstadoPedido = "recibido" | "confirmado" | "preparando" | "listo" | "entregado" | "cerrado";
type Vista = "carta" | "seguimiento";

export type PedidoSesion = {
  id: string;
  items: CartItem[];
  estado: EstadoPedido;
};

type State = {
  items: CartItem[];
  pedidos: PedidoSesion[];
  vista: Vista;
  estadoPedido: EstadoPedido;
  cuentaSolicitada: boolean;
  pagoHabilitado: boolean;
  cuentaVersion: number;
  pedidoId: string | null;
};

type Action =
  | {
      type: "ADD_ITEM";
      payload: {
        id?: string;
        articuloId: string;
        nombre: string;
        precio: number;
        precioBase?: number;
        cantidad?: number;
        nota?: string;
        variantes?: VariantePublica[];
      };
    }
  | { type: "SET_CANTIDAD"; payload: { id: string; cantidad: number } }
  | { type: "SET_NOTA"; payload: { id: string; nota: string } }
  | { type: "SET_VISTA"; payload: Vista }
  | { type: "HYDRATE"; payload: State }
  | { type: "CONFIRMAR_PEDIDO"; payload: { pedidoId: string; items: CartItem[] } }
  | { type: "SET_PEDIDOS"; payload: PedidoSesion[] }
  | { type: "UPSERT_PEDIDO"; payload: PedidoSesion }
  | { type: "SET_ESTADO_PEDIDO"; payload: { pedidoId: string; estado: EstadoPedido } }
  | { type: "SYNC_CUENTA"; payload: { cuentaSolicitada: boolean; pagoHabilitado?: boolean; cuentaVersion: number } }
  | { type: "RESET_SESSION" };

const INITIAL_STATE: State = {
  items: [],
  pedidos: [],
  vista: "carta",
  estadoPedido: "recibido",
  cuentaSolicitada: false,
  pagoHabilitado: false,
  cuentaVersion: 1,
  pedidoId: null,
};

const SESSION_VERSION = 3;

function generarLineKey(articuloId: string, variantes?: VariantePublica[], nota?: string): string {
  const vKeys = (variantes ?? []).map((v) => v.id).sort().join("-");
  const n = (nota ?? "").trim();
  return `${articuloId}_${vKeys}_${n}`;
}

function reducer(state: State, action: Action): State {
  switch (action.type) {
    case "ADD_ITEM": {
      const lineKey = action.payload.id || generarLineKey(action.payload.articuloId, action.payload.variantes, action.payload.nota);
      const exists = state.items.find((i) => i.id === lineKey);
      const cantidadToAdd = action.payload.cantidad && action.payload.cantidad > 0 ? action.payload.cantidad : 1;
      if (exists) {
        return {
          ...state,
          items: state.items.map((i) =>
            i.id === lineKey ? { ...i, cantidad: i.cantidad + cantidadToAdd } : i
          ),
        };
      }
      return {
        ...state,
        items: [
          ...state.items,
          {
            id: lineKey,
            articuloId: action.payload.articuloId,
            nombre: action.payload.nombre,
            precio: action.payload.precio,
            precioBase: action.payload.precioBase ?? action.payload.precio,
            cantidad: cantidadToAdd,
            nota: action.payload.nota ?? "",
            variantes: action.payload.variantes,
          },
        ],
      };
    }
    case "SET_CANTIDAD":
      if (action.payload.cantidad <= 0) {
        return { ...state, items: state.items.filter((i) => i.id !== action.payload.id) };
      }
      return {
        ...state,
        items: state.items.map((i) =>
          i.id === action.payload.id ? { ...i, cantidad: action.payload.cantidad } : i
        ),
      };
    case "SET_NOTA":
      return {
        ...state,
        items: state.items.map((i) =>
          i.id === action.payload.id ? { ...i, nota: action.payload.nota } : i
        ),
      };
    case "SET_VISTA":
      return { ...state, vista: action.payload };
    case "HYDRATE":
      return action.payload;
    case "CONFIRMAR_PEDIDO":
      return {
        ...state,
        items: [],
        pedidos: [
          ...state.pedidos.filter((pedido) => pedido.id !== action.payload.pedidoId),
          { id: action.payload.pedidoId, items: action.payload.items, estado: "recibido" },
        ],
        vista: "seguimiento",
        estadoPedido: "recibido",
        pedidoId: action.payload.pedidoId,
      };
    case "SET_PEDIDOS": {
      const ultimoPedido = action.payload[action.payload.length - 1];
      return {
        ...state,
        pedidos: action.payload,
        pedidoId: ultimoPedido?.id ?? null,
        estadoPedido: ultimoPedido?.estado ?? state.estadoPedido,
      };
    }
    case "UPSERT_PEDIDO":
      return {
        ...state,
        pedidos: [
          ...state.pedidos.filter((pedido) => pedido.id !== action.payload.id),
          action.payload,
        ],
        pedidoId: action.payload.id,
        estadoPedido: action.payload.estado,
      };
    case "SET_ESTADO_PEDIDO":
      return {
        ...state,
        pedidos: state.pedidos.map((pedido) =>
          pedido.id === action.payload.pedidoId
            ? { ...pedido, estado: action.payload.estado }
            : pedido
        ),
        estadoPedido:
          state.pedidoId === action.payload.pedidoId ? action.payload.estado : state.estadoPedido,
      };
    case "SYNC_CUENTA": {
      const cuentaCambio = state.cuentaVersion !== action.payload.cuentaVersion;
      const tieneSesionAnterior = state.items.length > 0 || state.pedidos.length > 0 || state.cuentaSolicitada;
      if (cuentaCambio && tieneSesionAnterior) {
        return { ...INITIAL_STATE, cuentaVersion: action.payload.cuentaVersion };
      }
      return {
        ...state,
        cuentaSolicitada: action.payload.cuentaSolicitada,
        pagoHabilitado: action.payload.pagoHabilitado ?? state.pagoHabilitado,
        cuentaVersion: action.payload.cuentaVersion,
        vista: action.payload.cuentaSolicitada ? "seguimiento" : state.vista,
      };
    }
    case "RESET_SESSION":
      return { ...INITIAL_STATE };
    default:
      return state;
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function isVista(value: unknown): value is Vista {
  return value === "carta" || value === "seguimiento";
}

function isEstadoPedido(value: unknown): value is EstadoPedido {
  return (
    value === "recibido" ||
    value === "confirmado" ||
    value === "preparando" ||
    value === "listo" ||
    value === "entregado" ||
    value === "cerrado"
  );
}

function parseCartItems(value: unknown): CartItem[] {
  if (!Array.isArray(value)) return [];

  return value.flatMap((candidate) => {
    if (!isRecord(candidate)) return [];
    if (
      typeof candidate.id !== "string" ||
      typeof candidate.nombre !== "string" ||
      typeof candidate.precio !== "number" ||
      !Number.isFinite(candidate.precio) ||
      typeof candidate.cantidad !== "number" ||
      !Number.isInteger(candidate.cantidad) ||
      candidate.cantidad <= 0 ||
      typeof candidate.nota !== "string"
    ) {
      return [];
    }

    const articuloId =
      typeof candidate.articuloId === "string" && candidate.articuloId.trim()
        ? candidate.articuloId
        : candidate.id;

    const comensalId =
      typeof candidate.comensalId === "string" && candidate.comensalId.trim()
        ? candidate.comensalId
        : undefined;
    const comensalNombre =
      typeof candidate.comensalNombre === "string" && candidate.comensalNombre.trim()
        ? candidate.comensalNombre.trim()
        : undefined;

    let variantes: VariantePublica[] | undefined;
    if (Array.isArray(candidate.variantes)) {
      variantes = candidate.variantes.flatMap((v) => {
        if (!isRecord(v) || typeof v.id !== "string" || typeof v.nombre !== "string") return [];
        return [
          {
            id: v.id,
            articulo_id: typeof v.articulo_id === "string" ? v.articulo_id : articuloId,
            nombre: v.nombre,
            precio_adicional: typeof v.precio_adicional === "number" ? v.precio_adicional : 0,
            grupo: typeof v.grupo === "string" ? v.grupo : undefined,
            seleccion_unica: Boolean(v.seleccion_unica),
            orden: typeof v.orden === "number" ? v.orden : 0,
          },
        ];
      });
    }

    return [
      {
        id: candidate.id,
        articuloId,
        nombre: candidate.nombre,
        precio: candidate.precio,
        precioBase: typeof candidate.precioBase === "number" ? candidate.precioBase : undefined,
        cantidad: candidate.cantidad,
        nota: candidate.nota,
        variantes,
        comensalId,
        comensalNombre,
      },
    ];
  });
}

function pedidoApiToSession(pedido: PedidoAPI): PedidoSesion {
  return {
    id: pedido.id,
    estado: pedido.estado,
    items: (pedido.items ?? []).map((item) => ({
      id: item.id || item.articulo_id,
      articuloId: item.articulo_id,
      nombre: item.nombre_articulo?.trim() || "Producto sin nombre",
      precio: item.precio_unitario,
      cantidad: item.cantidad,
      nota: item.notas?.trim() || "",
      comensalId: item.comensal_id,
      comensalNombre: item.comensal_nombre?.trim() || undefined,
      variantes: item.variantes?.map((v) => ({
        id: v.variante_id,
        articulo_id: item.articulo_id,
        nombre: v.nombre,
        precio_adicional: v.precio_adicional,
        seleccion_unica: false,
        orden: 0,
      })),
    })),
  };
}

function parsePedidoSessions(value: unknown): PedidoSesion[] {
  if (!Array.isArray(value)) return [];

  return value.flatMap((candidate) => {
    if (!isRecord(candidate) || typeof candidate.id !== "string" || !candidate.id.trim()) return [];
    const items = parseCartItems(candidate.items);
    const estado = isEstadoPedido(candidate.estado) ? candidate.estado : "recibido";
    if (items.length === 0) return [];

    return [{ id: candidate.id, items, estado }];
  });
}

function getSessionKey(token: string) {
  return `mesa_click_session_${token}`;
}

function readStoredSession(raw: string): State | null {
  try {
    const parsed: unknown = JSON.parse(raw);
    if (!isRecord(parsed) || (parsed.version !== 1 && parsed.version !== 2 && parsed.version !== SESSION_VERSION)) return null;

    const pedidoId = typeof parsed.pedidoId === "string" && parsed.pedidoId.trim() ? parsed.pedidoId : null;
    const estadoPedido = isEstadoPedido(parsed.estadoPedido) ? parsed.estadoPedido : "recibido";
    const items = parseCartItems(parsed.items);
    const pedidosGuardados = parsePedidoSessions(parsed.pedidos);
    const pedidoItemsAnteriores = parseCartItems(parsed.pedidoItems);
    const pedidos =
      pedidosGuardados.length > 0
        ? pedidosGuardados
        : pedidoId && pedidoItemsAnteriores.length > 0
        ? [{ id: pedidoId, items: pedidoItemsAnteriores, estado: estadoPedido }]
        : pedidoId && items.length > 0
        ? [{ id: pedidoId, items, estado: estadoPedido }]
        : [];
    const ultimoPedido = pedidos[pedidos.length - 1];
    const rawVista = parsed.vista;
    const vista: Vista = rawVista === "carrito" ? "carta" : isVista(rawVista) ? rawVista : "carta";
    const cuentaVersion =
      typeof parsed.cuentaVersion === "number" && Number.isInteger(parsed.cuentaVersion) && parsed.cuentaVersion > 0
        ? parsed.cuentaVersion
        : 1;

    return {
      items,
      pedidos,
      vista: pedidos.length > 0 && items.length === 0 ? "seguimiento" : vista,
      estadoPedido: ultimoPedido?.estado ?? estadoPedido,
      cuentaSolicitada: parsed.cuentaSolicitada === true,
      pagoHabilitado: parsed.pagoHabilitado === true,
      cuentaVersion,
      pedidoId: ultimoPedido?.id ?? pedidoId,
    };
  } catch {
    return null;
  }
}

interface MenuCategoryView {
  id: string;
  nombre: string;
  icono?: string;
  disponible: boolean;
  disponibleDesde?: string;
  items: GuestCardItem[];
}

export default function MesaPage() {
  const params = useParams();
  const rawToken = params.token;
  const token = Array.isArray(rawToken) ? rawToken[0] : rawToken;
  const isDemo = token === "demo";

  const [mesa, setMesa] = useState<MesaPublica | null>(() => (isDemo ? DEMO_MESA : null));
  const [menu, setMenu] = useState<MenuCategoryView[]>([]);
  const [categoriaActiva, setCategoriaActiva] = useState<string>("Todos");
  const [busqueda, setBusqueda] = useState("");
  const [carritoAbierto, setCarritoAbierto] = useState(false);
  const [itemParaPersonalizar, setItemParaPersonalizar] = useState<GuestCardItem | null>(null);
  const [loading, setLoading] = useState(() => !isDemo && Boolean(token));
  const [enviandoPedido, setEnviandoPedido] = useState(false);
  const [comensal, setComensal] = useState<ComensalIdentity | null>(null);
  const [identidadLista, setIdentidadLista] = useState(false);
  const [editandoComensal, setEditandoComensal] = useState(false);
  const [pagandoMP, setPagandoMP] = useState(false);
  const [pagoExitoso, setPagoExitoso] = useState(() => {
    if (typeof window === "undefined") return false;
    const urlParams = new URLSearchParams(window.location.search);
    const pagoParam = urlParams.get("pago");
    const statusParam = urlParams.get("status") || urlParams.get("collection_status");
    return pagoParam === "exitoso" || statusParam === "approved";
  });
  const [pagoError, setPagoError] = useState(() => {
    if (typeof window === "undefined") return false;
    const urlParams = new URLSearchParams(window.location.search);
    const pagoParam = urlParams.get("pago");
    const statusParam = urlParams.get("status") || urlParams.get("collection_status");
    return pagoParam === "fallido" || statusParam === "rejected";
  });
  const [modalResenaAbierto, setModalResenaAbierto] = useState(false);
  const [resenaCalificadaCuenta, setResenaCalificadaCuenta] = useState<number | null>(null);

  const [state, dispatch] = useReducer(reducer, INITIAL_STATE);

  const resenaYaEnviada = useMemo(() => {
    if (resenaCalificadaCuenta === state.cuentaVersion) return true;
    if (typeof window === "undefined" || !token) return false;
    try {
      return window.localStorage.getItem(`mesa_click_resena_${token}_${state.cuentaVersion}`) === "true";
    } catch {
      return false;
    }
  }, [resenaCalificadaCuenta, token, state.cuentaVersion]);

  const handleResenaEnviada = useCallback(() => {
    if (typeof window !== "undefined" && token) {
      try {
        window.localStorage.setItem(`mesa_click_resena_${token}_${state.cuentaVersion}`, "true");
      } catch {
        // localStorage fallback
      }
    }
    setResenaCalificadaCuenta(state.cuentaVersion);
  }, [token, state.cuentaVersion]);

  const skipNextPersistRef = useRef(false);
  const hydratedTokenRef = useRef<string | null>(null);

  // Lista de nombres de comensales ya activos en esta mesa
  const nombresComensalesActivos = useMemo(() => {
    const set = new Set<string>();
    for (const p of state.pedidos) {
      for (const item of p.items) {
        if (item.comensalNombre?.trim()) {
          set.add(item.comensalNombre.trim());
        }
      }
    }
    return Array.from(set);
  }, [state.pedidos]);

  const sincronizarPedidosMesa = useCallback(async () => {
    if (!token || isDemo) return;
    try {
      const pedidosApi = await api.listarPedidosMesa(token);
      dispatch({
        type: "SET_PEDIDOS",
        payload: (pedidosApi ?? []).map(pedidoApiToSession),
      });
    } catch (error) {
      console.warn("No se pudo sincronizar la cuenta de la mesa:", error);
    }
  }, [token, isDemo]);

  // 1. Restaurar sesión persistida
  useEffect(() => {
    if (!token) return;

    skipNextPersistRef.current = true;
    dispatch({ type: "RESET_SESSION" });

    try {
      const key = getSessionKey(token);
      const rawSession = window.localStorage.getItem(key);
      if (rawSession) {
        const restored = readStoredSession(rawSession);
        if (restored) {
          dispatch({ type: "HYDRATE", payload: restored });
        } else {
          window.localStorage.removeItem(key);
        }
      }
    } catch (error) {
      console.warn("No se pudo restaurar la sesión de la mesa:", error);
    }

    hydratedTokenRef.current = token;
  }, [token]);

  // 2. Persistir sesión al cambiar
  useEffect(() => {
    if (!token || hydratedTokenRef.current !== token) return;
    if (skipNextPersistRef.current) {
      skipNextPersistRef.current = false;
      return;
    }

    try {
      window.localStorage.setItem(
        getSessionKey(token),
        JSON.stringify({ version: SESSION_VERSION, ...state })
      );
    } catch (error) {
      console.warn("No se pudo guardar la sesión de la mesa:", error);
    }
  }, [state, token]);

  // 3. Carga de datos inicial (Modo Demo vs Modo Real QR)
  useEffect(() => {
    let isMounted = true;

    if (isDemo) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- la identidad del comensal vive en localStorage y se lee tras hidratar; refactor de la carga fuera del alcance de #114
      setMesa(DEMO_MESA);
      const categoriasMock: MenuCategoryView[] = [
        {
          id: "Entradas",
          nombre: "Entradas",
          disponible: true,
          items: DEMO_ITEMS.filter((i) => i.categoria === "Entradas"),
        },
        {
          id: "Principales",
          nombre: "Principales",
          disponible: true,
          items: DEMO_ITEMS.filter((i) => i.categoria === "Principales"),
        },
        {
          id: "Bebidas",
          nombre: "Bebidas",
          disponible: true,
          items: DEMO_ITEMS.filter((i) => i.categoria === "Bebidas"),
        },
      ];
      setMenu(categoriasMock);
      const stored = getComensalIdentity(DEMO_MESA.id, DEMO_MESA.cuenta_version);
      setComensal(stored);
      setIdentidadLista(true);
      setLoading(false);
      return;
    }

    if (!token) return;
    const currentToken = token;

    async function cargarDatos() {
      try {
        setLoading(true);
        const mesaApi = await api.obtenerMesaPorQR(currentToken);
        if (!isMounted) return;
        setMesa(mesaApi);

        const identidad =
          mesaApi.estado === "inactiva"
            ? null
            : getComensalIdentity(mesaApi.id, mesaApi.cuenta_version);
        if (mesaApi.estado === "inactiva") {
          clearComensalIdentity(mesaApi.id);
        }
        setComensal(identidad);
        setEditandoComensal(false);
        setIdentidadLista(true);

        dispatch({
          type: "SYNC_CUENTA",
          payload: {
            cuentaSolicitada: mesaApi.cuenta_solicitada,
            pagoHabilitado: mesaApi.pago_habilitado,
            cuentaVersion: mesaApi.cuenta_version,
          },
        });

        await sincronizarPedidosMesa();

        // Cargar carta pública de la sucursal
        try {
          const cartaResp = await api.obtenerCartaPublica(mesaApi.sucursal_id);
          if (cartaResp && cartaResp.categorias && cartaResp.categorias.length > 0) {
            const formateadas: MenuCategoryView[] = cartaResp.categorias.map(
              (c: CategoriaPublica) => ({
                id: c.id,
                nombre: c.nombre,
                icono: c.icono,
                disponible: c.disponible,
                disponibleDesde: c.disponible_desde,
                items: (c.articulos || []).map((a: ArticuloPublico) => ({
                  id: a.id,
                  nombre: a.nombre,
                  descripcion: a.descripcion,
                  foto_url: a.foto_url,
                  precio: a.precio,
                  variantes: a.variantes,
                  tag: c.nombre,
                })),
              })
            );

            if (isMounted) {
              setMenu(formateadas);
            }
            return;
          }
        } catch (err) {
          console.error("Error cargando carta pública:", err);
        }

        if (isMounted) setMenu([]);
      } catch (error) {
        console.error("Error cargando mesa por QR:", error);
        if (isMounted) setMesa(null);
      } finally {
        if (isMounted) setLoading(false);
      }
    }

    void cargarDatos();

    return () => {
      isMounted = false;
    };
  }, [isDemo, sincronizarPedidosMesa, token]);

  // 4. SSE en tiempo real para salón y cocina
  const mesaId = mesa?.id;
  const mesaEstado = mesa?.estado;

  useEffect(() => {
    if (!mesaId || !token || isDemo) return;

    const eventSource = new EventSource(
      api.obtenerEventosMesaUrl(mesaId, mesa?.tenant_id, mesa?.sucursal_id)
    );
    eventSource.onopen = () => {
      void sincronizarPedidosMesa();
    };

    eventSource.addEventListener("pedido_creado", (event: MessageEvent<string>) => {
      try {
        const payload: unknown = JSON.parse(event.data);
        if (!isRecord(payload) || typeof payload.id !== "string" || !isEstadoPedido(payload.estado))
          return;
        dispatch({ type: "UPSERT_PEDIDO", payload: pedidoApiToSession(payload as unknown as PedidoAPI) });
      } catch (error) {
        console.warn("Error parseando pedido_creado:", error);
      }
    });

    eventSource.addEventListener("pedido_actualizado", (event: MessageEvent<string>) => {
      try {
        const payload: unknown = JSON.parse(event.data);
        if (!isRecord(payload) || typeof payload.id !== "string" || !isEstadoPedido(payload.estado))
          return;
        dispatch({
          type: "SET_ESTADO_PEDIDO",
          payload: { pedidoId: payload.id, estado: payload.estado },
        });
      } catch (error) {
        console.warn("Error parseando pedido_actualizado:", error);
      }
    });

    const sincronizarCuentaDesdeEvento = (event: MessageEvent<string>) => {
      try {
        const payload: unknown = JSON.parse(event.data);
        if (!isRecord(payload) || typeof payload.cuenta_solicitada !== "boolean" || typeof payload.cuenta_version !== "number")
          return;

        setMesa((actual) =>
          actual
            ? {
                ...actual,
                cuenta_solicitada: payload.cuenta_solicitada as boolean,
                pago_habilitado:
                  payload.pago_habilitado !== undefined
                    ? (payload.pago_habilitado as boolean)
                    : actual.pago_habilitado,
                cuenta_version: payload.cuenta_version as number,
              }
            : actual
        );

        if (mesaId) {
          const identidad = getComensalIdentity(mesaId, payload.cuenta_version);
          setComensal(identidad);
          setEditandoComensal(false);
          setIdentidadLista(true);
        }

        dispatch({
          type: "SYNC_CUENTA",
          payload: {
            cuentaSolicitada: payload.cuenta_solicitada,
            pagoHabilitado: payload.pago_habilitado !== undefined ? Boolean(payload.pago_habilitado) : undefined,
            cuentaVersion: payload.cuenta_version,
          },
        });

        if (event.type === "cuenta_cerrada") {
          void sincronizarPedidosMesa();
        }
      } catch (error) {
        console.warn("Error parseando evento cuenta:", error);
      }
    };

    const handlePagoAprobado = () => {
      setPagoExitoso(true);
      setPagoError(false);
      dispatch({ type: "SET_VISTA", payload: "seguimiento" });
      void sincronizarPedidosMesa();
    };

    eventSource.addEventListener("cuenta_solicitada", sincronizarCuentaDesdeEvento);
    eventSource.addEventListener("cuenta_cerrada", sincronizarCuentaDesdeEvento);
    eventSource.addEventListener("pago_habilitado", sincronizarCuentaDesdeEvento);
    eventSource.addEventListener("pago_deshabilitado", sincronizarCuentaDesdeEvento);
    eventSource.addEventListener("pago_aprobado", handlePagoAprobado);

    return () => {
      eventSource.close();
    };
  }, [mesaId, isDemo, sincronizarPedidosMesa, token]);

  // 5. Manejo de retorno de Mercado Pago
  useEffect(() => {
    if (typeof window === "undefined" || !token || isDemo) return;
    const urlParams = new URLSearchParams(window.location.search);
    const paymentId = urlParams.get("payment_id") || urlParams.get("collection_id");

    if (pagoExitoso) {
      dispatch({ type: "SET_VISTA", payload: "seguimiento" });
      if (paymentId) {
        api.confirmarPagoMP(token, paymentId).catch((err) => {
          console.warn("Aviso confirmando pago MP:", err);
        });
      }
    } else if (pagoError) {
      dispatch({ type: "SET_VISTA", payload: "seguimiento" });
    }
  }, [token, isDemo, pagoExitoso, pagoError]);

  // 6. Acciones del comensal
  const handleGuardarComensal = (nombre: string) => {
    if (!mesa) return;
    const identidad = comensal
      ? { ...comensal, nombre: nombre.trim(), cuentaVersion: mesa.cuenta_version }
      : createComensalIdentity(nombre, mesa.cuenta_version);
    saveComensalIdentity(identidad, mesa.id);
    setComensal(identidad);
    setEditandoComensal(false);
    setIdentidadLista(true);
  };

  const handleContinuarSinAlias = () => {
    if (!mesa) return;
    const identidad = createComensalIdentity("", mesa.cuenta_version);
    saveComensalIdentity(identidad, mesa.id);
    setComensal(identidad);
    setEditandoComensal(false);
    setIdentidadLista(true);
  };

  const handleAddToCart = (
    item: GuestCardItem,
    cantidad: number,
    variantes: VariantePublica[],
    nota: string
  ) => {
    const recargos = variantes.reduce((acc, v) => acc + (v.precio_adicional || 0), 0);
    const precioUnitario = item.precio + recargos;

    dispatch({
      type: "ADD_ITEM",
      payload: {
        articuloId: item.id,
        nombre: item.nombre,
        precio: precioUnitario,
        precioBase: item.precio,
        cantidad,
        nota,
        variantes,
      },
    });
  };

  const handleConfirmarPedido = async () => {
    if (!mesa || state.items.length === 0 || state.cuentaSolicitada) return;
    if (!comensal) {
      setEditandoComensal(true);
      return;
    }

    setEnviandoPedido(true);
    const itemsEnviados = state.items.map((item) => ({
      ...item,
      comensalId: comensal.id,
      comensalNombre: comensal.nombre,
    }));

    // Simulación en modo demo
    if (isDemo) {
      setTimeout(() => {
        const demoId = `demo-ped-${Date.now().toString().slice(-4)}`;
        dispatch({ type: "CONFIRMAR_PEDIDO", payload: { pedidoId: demoId, items: itemsEnviados } });
        setCarritoAbierto(false);
        setEnviandoPedido(false);
      }, 700);
      return;
    }

    try {
      const resp = await api.crearPedido({
        mesa_id: mesa.id,
        items: itemsEnviados.map((item) => ({
          articulo_id: item.articuloId,
          cantidad: item.cantidad,
          notas: item.nota,
          comensal_id: comensal.id,
          comensal_nombre: comensal.nombre,
          variantes: item.variantes?.map((v) => v.id) ?? [],
        })),
      });

      if (!resp.id) throw new Error("La API no devolvió el ID del pedido.");
      dispatch({ type: "CONFIRMAR_PEDIDO", payload: { pedidoId: resp.id, items: itemsEnviados } });
      setCarritoAbierto(false);
    } catch (err: unknown) {
      console.error("Error enviando pedido:", err);
      const msg = err instanceof Error ? err.message : "";
      alert(msg || "Error al enviar el pedido. Por favor intenta nuevamente.");
    } finally {
      setEnviandoPedido(false);
    }
  };

  const handlePedirCuenta = async () => {
    if (state.cuentaSolicitada) return;
    if (isDemo) {
      dispatch({
        type: "SYNC_CUENTA",
        payload: { cuentaSolicitada: true, pagoHabilitado: true, cuentaVersion: 1 },
      });
      return;
    }
    if (!token) return;
    try {
      const mesaActualizada = await api.solicitarCuenta(token);
      setMesa(mesaActualizada);
      dispatch({
        type: "SYNC_CUENTA",
        payload: {
          cuentaSolicitada: mesaActualizada.cuenta_solicitada,
          pagoHabilitado: mesaActualizada.pago_habilitado,
          cuentaVersion: mesaActualizada.cuenta_version,
        },
      });
    } catch (error) {
      console.error("Error pidiendo la cuenta:", error);
      alert("No se pudo solicitar la cuenta. Por favor avisa al mozo.");
    }
  };

  const handlePagarMercadoPago = async () => {
    if (isDemo) {
      alert("Modo demostración: En un restaurante real, aquí se abre el checkout seguro de Mercado Pago.");
      setPagoExitoso(true);
      return;
    }
    if (!token || !mesa) return;
    setPagandoMP(true);
    setPagoError(false);

    try {
      const resp = await api.crearPreferenciaPagoMP(token);
      const urlDestino = resp.sandbox_init_point || resp.init_point;
      if (urlDestino) {
        window.location.href = urlDestino;
      } else {
        throw new Error("No se obtuvo la URL de pago.");
      }
    } catch (error) {
      console.error("Error creando preferencia MP:", error);
      alert("No se pudo iniciar el cobro con Mercado Pago. Consulta al mozo.");
      setPagandoMP(false);
    }
  };

  // 7. Render de estados de carga y error
  if (loading) {
    return (
      <div className="guest-page flex items-center justify-center p-24">
        <div className="flex flex-col items-center">
          <ComensalIcon name="brand" size={40} className="animate-bounce text-[var(--brand,#f06c4f)]" />
          <p className="mt-14 font-semibold text-13 text-[var(--ink,#17242b)]">Cargando la carta...</p>
        </div>
      </div>
    );
  }

  if (!mesa) {
    return (
      <div className="guest-page flex items-center justify-center p-24 text-center">
        <div className="guest-phone welcome-view flex items-center justify-center p-24 text-center">
          <ComensalIcon name="qr" size={48} className="text-[var(--muted,#778489)]" />
          <h2 className="mt-16 text-18 font-bold">Mesa no encontrada</h2>
          <p className="mt-6 text-13 text-[var(--muted,#778489)] max-w-xs">
            El código QR escaneado no coincide con una mesa activa del local.
          </p>
          <button
            type="button"
            className="guest-secondary mt-24"
            onClick={() => {
              window.location.href = "/";
            }}
          >
            Ir al inicio
          </button>
        </div>
      </div>
    );
  }

  // Branding y estilos
  const branding: MesaBranding = mesa;
  const theme = buildMesaTheme(branding);
  const themeStyle = theme.style as CSSProperties;
  const mesaCerrada = mesa.estado === "inactiva";
  const totalItemsEnCarrito = state.items.reduce((acc, i) => acc + i.cantidad, 0);
  const totalPrecioCarrito = state.items.reduce((acc, i) => acc + i.precio * i.cantidad, 0);
  const formatPrice = (p: number) => `$ ${p.toLocaleString("es-AR")}`;

  // Si la mesa está cerrada por el negocio
  if (mesaCerrada) {
    return (
      <div className="guest-page" data-estilo-visual={theme.visualStyle} style={themeStyle}>
        <div className="guest-phone welcome-view flex items-center justify-center p-24 text-center">
          <ComensalIcon name="close" size={48} className="text-[var(--brand,#f06c4f)]" />
          <h2 className="mt-16 text-20 font-bold">Esta mesa está cerrada</h2>
          <p className="mt-8 text-13 text-[var(--muted)] leading-relaxed">
            Ya no se aceptan pedidos en esta mesa. Consultá al personal del local si necesitás ayuda.
          </p>
        </div>
      </div>
    );
  }

  // Si el usuario aún no se ha sumado a la mesa (fase Welcome Screen de la maqueta)
  const yaSeSumo = Boolean(comensal && !editandoComensal);
  if (!yaSeSumo) {
    return (
      <div className="guest-page" data-estilo-visual={theme.visualStyle} style={themeStyle}>
        <div className="guest-phone">
          <WelcomeView
            mesa={mesa}
            dinerNames={nombresComensalesActivos}
            onJoin={handleGuardarComensal}
            onJoinAnonymous={handleContinuarSinAlias}
            isDemo={isDemo}
          />
        </div>
      </div>
    );
  }

  // Filtrado de carta por categoría y búsqueda
  const categoriasNombres = ["Todos", ...menu.map((c) => c.nombre)];
  const busquedaQuery = busqueda.trim().toLowerCase();

  const articulosFiltrados = menu.flatMap((c) => {
    if (categoriaActiva !== "Todos" && c.nombre !== categoriaActiva) return [];
    return c.items.filter((item) => {
      if (!busquedaQuery) return true;
      return (
        item.nombre.toLowerCase().includes(busquedaQuery) ||
        (item.descripcion && item.descripcion.toLowerCase().includes(busquedaQuery))
      );
    });
  });

  return (
    <div className="guest-page" data-estilo-visual={theme.visualStyle} style={themeStyle}>
      <div className="guest-phone menu-view">
        {/* Cabecera del restaurante y avatar */}
        <GuestHeader
          mesa={mesa}
          dinerName={comensal?.nombre}
          onOpenCuenta={() => dispatch({ type: "SET_VISTA", payload: "seguimiento" })}
          onEditDiner={() => setEditandoComensal(true)}
        />

        {/* Vista: Seguimiento / Estado del pedido */}
        {state.vista === "seguimiento" ? (
          <GuestOrderStatusView
            mesaNumero={mesa.numero}
            sector={mesa.sector}
            pedidos={state.pedidos}
            estadoPedido={state.estadoPedido}
            cuentaSolicitada={state.cuentaSolicitada}
            pagoHabilitado={state.pagoHabilitado}
            mercadopagoHabilitado={Boolean(mesa.mercadopago_habilitado)}
            comensalId={comensal?.id}
            comensalNombre={comensal?.nombre}
            onVolverCarta={() => dispatch({ type: "SET_VISTA", payload: "carta" })}
            onPedirCuenta={handlePedirCuenta}
            onPagarMercadoPago={mesa.mercadopago_habilitado ? handlePagarMercadoPago : undefined}
            pagandoMP={pagandoMP}
            pagoExitoso={pagoExitoso}
            pagoError={pagoError}
            todosListos={
              state.pedidos.length > 0 &&
              state.pedidos.every((p) => p.estado === "listo" || p.estado === "entregado" || p.estado === "cerrado")
            }
            yaCalificado={resenaYaEnviada}
            onCalificar={() => setModalResenaAbierto(true)}
            formatPrice={formatPrice}
          />
        ) : (
          /* Vista: Carta / Menú del local */
          <>
            {/* Buscador y carrusel de categorías sticky */}
            <div className="guest-sticky">
              <label className="guest-search">
                <ComensalIcon name="search" size={19} />
                <input
                  type="search"
                  value={busqueda}
                  onChange={(e) => setBusqueda(e.target.value)}
                  placeholder="¿Qué te gustaría comer?"
                  aria-label="Buscar en la carta"
                />
                {busqueda && (
                  <button
                    type="button"
                    className="guest-search-clear"
                    onClick={() => setBusqueda("")}
                    aria-label="Limpiar búsqueda"
                  >
                    <ComensalIcon name="close" size={16} />
                  </button>
                )}
              </label>

              <div className="category-scroll">
                {categoriasNombres.map((catName) => (
                  <button
                    key={catName}
                    type="button"
                    className={categoriaActiva === catName ? "active" : ""}
                    onClick={() => setCategoriaActiva(catName)}
                  >
                    {catName}
                  </button>
                ))}
              </div>
            </div>

            {/* Lista de productos */}
            <main className="guest-menu-content">
              {/* Nota de franja horaria si corresponde */}
              <div className="lunch-note">
                <ComensalIcon name="clock" size={17} />
                <span>
                  <strong>Servicio en vivo</strong> · Pedidos conectados directamente con cocina
                </span>
              </div>

              <div className="guest-section-title">
                <div>
                  <span>Nuestra selección</span>
                  <h1>{categoriaActiva}</h1>
                </div>
                <small>{articulosFiltrados.length} opciones</small>
              </div>

              <div className="food-list">
                {articulosFiltrados.map((item) => {
                  const cant = state.items
                    .filter((i) => i.articuloId === item.id)
                    .reduce((acc, i) => acc + i.cantidad, 0);

                  return (
                    <GuestFoodCard
                      key={item.id}
                      item={item}
                      cantidadEnCarrito={cant}
                      onSelect={(selected) => setItemParaPersonalizar(selected)}
                      formatPrice={formatPrice}
                    />
                  );
                })}

                {articulosFiltrados.length === 0 && (
                  <div className="py-36 text-center text-muted">
                    <ComensalIcon name="search" size={32} className="mx-auto text-muted mb-8" />
                    <p className="font-semibold text-13">No encontramos opciones disponibles</p>
                    <p className="text-11 mt-2">Probá buscando con otro nombre o seleccioná otra categoría.</p>
                  </div>
                )}
              </div>

              {/* Marca de agua si está habilitada */}
              {mesa.mostrar_marca_agua !== false && (
                <div className="mt-28 mb-12 flex justify-center">
                  <MarcaAgua />
                </div>
              )}
            </main>
          </>
        )}

        {/* Barra flotante de carrito (cuando hay productos seleccionados) */}
        {totalItemsEnCarrito > 0 && state.vista === "carta" && (
          <button
            type="button"
            className="cart-bar"
            onClick={() => setCarritoAbierto(true)}
            aria-label="Ver pedido en curso"
          >
            <span>
              <b>{totalItemsEnCarrito}</b> Ver pedido
            </span>
            <strong>{formatPrice(totalPrecioCarrito)}</strong>
          </button>
        )}

        {/* Navegación inferior persistente */}
        <nav className="guest-bottom-nav" aria-label="Navegación comensal">
          <button
            type="button"
            className={state.vista === "carta" ? "active" : ""}
            onClick={() => dispatch({ type: "SET_VISTA", payload: "carta" })}
          >
            <ComensalIcon name="book" size={20} />
            <span>Carta</span>
          </button>

          <button
            type="button"
            className={state.vista === "seguimiento" ? "active" : ""}
            onClick={() => dispatch({ type: "SET_VISTA", payload: "seguimiento" })}
          >
            <ComensalIcon name="clock" size={20} />
            <span>Mi pedido</span>
            {state.pedidos.length > 0 && (
              <span className="nav-badge">{state.pedidos.length}</span>
            )}
          </button>

          <button
            type="button"
            className={state.cuentaSolicitada ? "active text-amber-600" : ""}
            onClick={() => dispatch({ type: "SET_VISTA", payload: "seguimiento" })}
          >
            <ComensalIcon name="card" size={20} />
            <span>Cuenta</span>
          </button>
        </nav>

        {/* Modal de personalización de artículo */}
        {itemParaPersonalizar && (
          <GuestDetailSheet
            item={itemParaPersonalizar}
            onClose={() => setItemParaPersonalizar(null)}
            onAddToCart={handleAddToCart}
            formatPrice={formatPrice}
          />
        )}

        {/* Bottom sheet de carrito */}
        {carritoAbierto && (
          <GuestCartSheet
            mesaNumero={mesa.numero}
            dinerName={comensal?.nombre}
            items={state.items}
            totalPrecio={totalPrecioCarrito}
            enviando={enviandoPedido}
            onClose={() => setCarritoAbierto(false)}
            onSetCantidad={(id, cantidad) =>
              dispatch({ type: "SET_CANTIDAD", payload: { id, cantidad } })
            }
            onConfirmar={handleConfirmarPedido}
            formatPrice={formatPrice}
          />
        )}

        {/* Modal para renombrar alias */}
        {editandoComensal && (
          <ModalNombreComensal
            nombreInicial={comensal?.nombre}
            editando={true}
            onConfirmar={handleGuardarComensal}
            onCancelar={() => setEditandoComensal(false)}
          />
        )}

        {/* Smart Google Review Funnel Modal */}
        {token && (
          <ModalResena
            isOpen={modalResenaAbierto}
            onClose={() => setModalResenaAbierto(false)}
            qrToken={token}
            googleReviewUrl={mesa.google_review_url}
            onResenaEnviada={handleResenaEnviada}
          />
        )}
      </div>
    </div>
  );
}
