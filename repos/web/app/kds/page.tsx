// repos/web/app/kds/page.tsx
"use client";
import React, { useState, useEffect, useCallback, useMemo } from "react";
import { useRouter } from "next/navigation";
import {
  api,
  PedidoAPI,
  Sucursal,
} from "@/lib/api";
import { KDSHeader } from "@/components/kds/KDSHeader";
import { ComandaCard } from "@/components/kds/ComandaCard";
import { HistorialDespachoDrawer } from "@/components/kds/HistorialDespachoDrawer";
import { playNewOrderSound, playOrderReadySound } from "@/components/kds/AudioAlerts";

const VENTANA_AGRUPACION_MS = 5 * 60 * 1000;

interface GrupoComandas {
  id: string;
  mesaId: string;
  pedidos: PedidoAPI[];
  createdAt: string;
}

function agruparPedidosPorMesa(pedidos: PedidoAPI[]): GrupoComandas[] {
  const grupos: GrupoComandas[] = [];
  const ultimoGrupoPorMesa = new Map<string, GrupoComandas>();

  pedidos.forEach((pedido) => {
    const grupoExistente = ultimoGrupoPorMesa.get(pedido.mesa_id);
    const fechaPedido = new Date(pedido.created_at).getTime();
    const fechaInicioGrupo = grupoExistente
      ? new Date(grupoExistente.createdAt).getTime()
      : 0;

    if (
      grupoExistente &&
      fechaPedido - fechaInicioGrupo <= VENTANA_AGRUPACION_MS
    ) {
      grupoExistente.pedidos.push(pedido);
      return;
    }

    const nuevoGrupo: GrupoComandas = {
      id: `${pedido.mesa_id}-${pedido.id}`,
      mesaId: pedido.mesa_id,
      pedidos: [pedido],
      createdAt: pedido.created_at,
    };
    grupos.push(nuevoGrupo);
    ultimoGrupoPorMesa.set(pedido.mesa_id, nuevoGrupo);
  });

  return grupos;
}

export default function KDSPage() {
  const router = useRouter();

  // Estados de datos
  const [sucursales, setSucursales] = useState<Sucursal[]>([]);
  const [sucursalSeleccionadaId, setSucursalSeleccionadaId] = useState<string>("");
  const [mesasMap, setMesasMap] = useState<Record<string, { numero: number; sector?: string }>>({});
  const [pedidosActivos, setPedidosActivos] = useState<PedidoAPI[]>([]);
  const [pedidosDespachados, setPedidosDespachados] = useState<PedidoAPI[]>([]);

  // Estados de interfaz y conexión
  const [conectadoSSE, setConectadoSSE] = useState<boolean>(false);
  const [cargando, setCargando] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [drawerHistorialAbierto, setDrawerHistorialAbierto] = useState<boolean>(false);
	const [loginDestino, setLoginDestino] = useState('/login');

  // 1. Verificación de autenticación y carga inicial de sucursales y mesas
  useEffect(() => {
    async function inicializarKDS() {
      try {
        setCargando(true);
        setError(null);
				const sesion = await api.obtenerSesion();
				if (!['admin', 'encargado', 'cocina'].includes(sesion.rol)) {
					router.replace('/dashboard');
					return;
				}
				const tenant = await api.obtenerMiTenant();
				setLoginDestino(sesion.rol === 'admin' ? '/login' : `/${tenant.slug}/login`);

        // Cargar sucursales
        const sucs = await api.listarSucursales();
        setSucursales(sucs);

        if (sucs.length === 0) {
          setError("No se encontraron sucursales registradas.");
          setCargando(false);
          return;
        }

        // Determinar sucursal activa
        const guardada = typeof window !== "undefined" ? localStorage.getItem("kds_sucursal_id") : null;
        const inicial = sucs.find((s) => s.id === guardada) || sucs[0];
        setSucursalSeleccionadaId(inicial.id);

        // Cargar mapa de mesas y sectores
        try {
          const [mesas, sectores] = await Promise.all([
            api.listarMesas(),
            api.listarSectores(inicial.id).catch(() => []),
          ]);

          const sectoresLookup: Record<string, string> = {};
          sectores.forEach((sec) => {
            sectoresLookup[sec.id] = sec.nombre;
          });

          const map: Record<string, { numero: number; sector?: string }> = {};
          mesas.forEach((m) => {
            map[m.id] = {
              numero: m.numero,
              sector: m.sector_id ? sectoresLookup[m.sector_id] : undefined,
            };
          });
          setMesasMap(map);
        } catch (e) {
          console.warn("No se pudieron cargar nombres de mesas/sectores:", e);
        }
      } catch (err: unknown) {
        console.error("Error inicializando KDS:", err);
			setError(err instanceof Error ? err.message : "Error al inicializar pantalla de cocina");
			if (err instanceof Error && err.message.toLowerCase().includes('sesión')) router.replace('/login');
      } finally {
        setCargando(false);
      }
    }

    inicializarKDS();
  }, [router]);

  // 2. Cargar comandas activas al cambiar de sucursal
  const cargarPedidosSucursal = useCallback(async (sucursalId: string) => {
    if (!sucursalId) return;
    try {
      const pedidos = await api.listarPedidosActivos(sucursalId);
      // Filtrar sólo pedidos no cerrados para el tablero activo
      const activos = pedidos.filter((p) => p.estado !== "cerrado");
      setPedidosActivos(activos);
    } catch (err) {
      console.error("Error al listar pedidos activos:", err);
    }
  }, []);

  useEffect(() => {
    if (!sucursalSeleccionadaId) return;
    const timeoutId = window.setTimeout(() => {
      void cargarPedidosSucursal(sucursalSeleccionadaId);
    }, 0);
    return () => window.clearTimeout(timeoutId);
  }, [sucursalSeleccionadaId, cargarPedidosSucursal]);

  // 3. Conexión en tiempo real SSE al canal KDS (US-64 & US-65)
  useEffect(() => {
    if (!sucursalSeleccionadaId) return;

    const sseUrl = api.obtenerEventosKDSUrl(sucursalSeleccionadaId);
    let eventSource: EventSource | null = null;

    try {
		eventSource = new EventSource(sseUrl, { withCredentials: true });

      eventSource.onopen = () => {
        setConectadoSSE(true);
      };

      eventSource.onerror = () => {
        setConectadoSSE(false);
      };

      // Ping keepalive
      eventSource.addEventListener("ping", () => {
        setConectadoSSE(true);
      });

      // Nueva comanda ingresada (US-66: sonido de campana / chime)
      eventSource.addEventListener("pedido_creado", (e: MessageEvent) => {
        try {
          const nuevoPedido: PedidoAPI = JSON.parse(e.data);
          playNewOrderSound();
          setPedidosActivos((prev) => {
            if (prev.some((p) => p.id === nuevoPedido.id)) {
              return prev.map((p) => (p.id === nuevoPedido.id ? nuevoPedido : p));
            }
            return [...prev, nuevoPedido];
          });
        } catch (err) {
          console.error("Error procesando pedido_creado SSE:", err);
        }
      });

      // Pedido actualizado globalmente
      eventSource.addEventListener("pedido_actualizado", (e: MessageEvent) => {
        try {
          const pedidoActualizado: PedidoAPI = JSON.parse(e.data);

          if (pedidoActualizado.estado === "cerrado") {
            // Remueve de activos y agrega a despachados
            setPedidosActivos((prev) => prev.filter((p) => p.id !== pedidoActualizado.id));
            setPedidosDespachados((prev) => {
              const filtrados = prev.filter((p) => p.id !== pedidoActualizado.id);
              return [pedidoActualizado, ...filtrados].slice(0, 30);
            });
          } else {
            // Actualiza en activos
            setPedidosActivos((prev) => {
              const existe = prev.some((p) => p.id === pedidoActualizado.id);
              if (existe) {
                return prev.map((p) => (p.id === pedidoActualizado.id ? pedidoActualizado : p));
              }
              return [...prev, pedidoActualizado];
            });

            // Si pasa a listo, sonar alerta de comanda lista
            if (pedidoActualizado.estado === "listo") {
              playOrderReadySound();
            }
          }
        } catch (err) {
          console.error("Error procesando pedido_actualizado SSE:", err);
        }
      });

      // Ítem individual actualizado en cocina (US-66)
      eventSource.addEventListener("pedido_item_actualizado", (e: MessageEvent) => {
        try {
          const pedidoActualizado: PedidoAPI = JSON.parse(e.data);

          if (pedidoActualizado.estado === "cerrado") {
            setPedidosActivos((prev) => prev.filter((p) => p.id !== pedidoActualizado.id));
          } else {
            setPedidosActivos((prev) =>
              prev.map((p) => (p.id === pedidoActualizado.id ? pedidoActualizado : p))
            );

            // Verificar si todos los ítems están listos
            const todosListos =
              pedidoActualizado.items &&
              pedidoActualizado.items.length > 0 &&
              pedidoActualizado.items.every((it) => it.estado === "listo");

            if (todosListos || pedidoActualizado.estado === "listo") {
              playOrderReadySound();
            }
          }
        } catch (err) {
          console.error("Error procesando pedido_item_actualizado SSE:", err);
        }
      });
    } catch (err) {
      console.warn("Error creando EventSource KDS:", err);
    }

    return () => {
      if (eventSource) {
        eventSource.close();
      }
    };
  }, [sucursalSeleccionadaId]);

  // Cambio de sucursal desde el header
  function handleCambiarSucursal(sucId: string) {
    setSucursalSeleccionadaId(sucId);
    if (typeof window !== "undefined") {
      localStorage.setItem("kds_sucursal_id", sucId);
    }
  }

  // 4. Acciones interactivas táctiles (US-66)
  const handleCambiarEstadoItem = useCallback(
    async (itemId: string, nuevoEstado: "pendiente" | "preparando" | "listo") => {
      // Optimistic update
      setPedidosActivos((prev) =>
        prev.map((ped) => {
          if (!ped.items || !ped.items.some((it) => it.id === itemId)) return ped;
          const itemsActualizados = ped.items.map((it) =>
            it.id === itemId ? { ...it, estado: nuevoEstado } : it
          );
          return { ...ped, items: itemsActualizados };
        })
      );

      try {
        const pedActualizado = await api.cambiarEstadoItem(itemId, nuevoEstado);
        setPedidosActivos((prev) =>
          prev.map((p) => (p.id === pedActualizado.id ? pedActualizado : p))
        );
        if (pedActualizado.estado === "listo") {
          playOrderReadySound();
        }
      } catch (err) {
        console.error("Error al cambiar estado de ítem:", err);
        // Recargar pedidos en caso de error
        if (sucursalSeleccionadaId) {
          cargarPedidosSucursal(sucursalSeleccionadaId);
        }
      }
    },
    [sucursalSeleccionadaId, cargarPedidosSucursal]
  );

  const handleComandasCompletasLista = useCallback(
    async (pedidoIds: string[]) => {
      const pedidosIds = new Set(pedidoIds);

      // Optimistic update
      setPedidosActivos((prev) =>
        prev.map((p) => {
          if (!pedidosIds.has(p.id)) return p;
          const itemsListos = p.items?.map((it) => ({ ...it, estado: "listo" as const }));
          return { ...p, estado: "listo", items: itemsListos };
        })
      );
      playOrderReadySound();

      try {
        const actualizados = await Promise.all(
          pedidoIds.map((pedidoId) => api.cambiarEstadoPedido(pedidoId, "listo"))
        );
        const actualizadosMap = new Map(actualizados.map((pedido) => [pedido.id, pedido]));
        setPedidosActivos((prev) =>
          prev.map((pedido) => actualizadosMap.get(pedido.id) ?? pedido)
        );
      } catch (err) {
        console.error("Error al marcar grupo de comandas como listo:", err);
        if (sucursalSeleccionadaId) {
          cargarPedidosSucursal(sucursalSeleccionadaId);
        }
      }
    },
    [sucursalSeleccionadaId, cargarPedidosSucursal]
  );

  const handleDespacharComandas = useCallback(
    async (pedidoIds: string[]) => {
      const pedidosIds = new Set(pedidoIds);
      const pedidosADespachar = pedidosActivos.filter((pedido) => pedidosIds.has(pedido.id));

      // Optimistic update
      setPedidosActivos((prev) => prev.filter((pedido) => !pedidosIds.has(pedido.id)));
      if (pedidosADespachar.length > 0) {
        setPedidosDespachados((prev) => [
          ...pedidosADespachar.map((pedido) => ({ ...pedido, estado: "cerrado" as const })),
          ...prev,
        ]);
      }

      try {
        await Promise.all(
          pedidoIds.map((pedidoId) => api.cambiarEstadoPedido(pedidoId, "cerrado"))
        );
      } catch (err) {
        console.error("Error al despachar grupo de comandas:", err);
        if (sucursalSeleccionadaId) {
          cargarPedidosSucursal(sucursalSeleccionadaId);
        }
      }
    },
    [pedidosActivos, sucursalSeleccionadaId, cargarPedidosSucursal]
  );

  const handleReabrirComanda = useCallback(
    async (pedidoId: string) => {
      try {
        const reabierto = await api.cambiarEstadoPedido(pedidoId, "listo");
        setPedidosDespachados((prev) => prev.filter((p) => p.id !== pedidoId));
        setPedidosActivos((prev) => [reabierto, ...prev]);
      } catch (err) {
        console.error("Error al reabrir comanda:", err);
      }
    },
    []
  );

  // 5. Clasificación Kanban por columnas (US-65)
  // Ordenamiento FIFO: más antiguas primero (created_at ascendente)
  const { pendientes, enPreparacion, listos } = useMemo(() => {
    const ordenados = [...pedidosActivos].sort(
      (a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
    );

    const pend: PedidoAPI[] = [];
    const prep: PedidoAPI[] = [];
    const list: PedidoAPI[] = [];

    ordenados.forEach((p) => {
      const items = p.items || [];
      const todosListos = items.length > 0 && items.every((i) => i.estado === "listo");

      if (p.estado === "listo" || todosListos) {
        list.push(p);
      } else if (
        p.estado === "preparando" ||
        items.some((i) => i.estado === "preparando")
      ) {
        prep.push(p);
      } else {
        pend.push(p);
      }
    });

    return {
      pendientes: agruparPedidosPorMesa(pend),
      enPreparacion: agruparPedidosPorMesa(prep),
      listos: agruparPedidosPorMesa(list),
    };
  }, [pedidosActivos]);

  return (
    <div className="flex min-h-screen flex-col bg-[#F4F6F7] font-inter text-ash-graphite">
      {/* HEADER DE KDS */}
      <KDSHeader
        sucursales={sucursales}
        sucursalSeleccionadaId={sucursalSeleccionadaId}
        onCambiarSucursal={handleCambiarSucursal}
        conectadoSSE={conectadoSSE}
        totalComandasActivas={pedidosActivos.length}
        totalDespachadas={pedidosDespachados.length}
        onAbrirHistorial={() => setDrawerHistorialAbierto(true)}
		loginDestino={loginDestino}
      />

      {/* CONTENIDO PRINCIPAL */}
      <main className="flex-1 overflow-x-auto p-16 md:p-24">
        {cargando ? (
          <div className="flex h-96 flex-col items-center justify-center text-center">
            <div className="mb-16 h-40 w-40 animate-spin rounded-full border-4 border-[#4D8EDB] border-t-transparent" />
            <p className="text-16 font-bold text-ash-graphite">Cargando comandas de cocina...</p>
            <p className="mt-4 text-12 text-sage-green">Sincronizando con el servidor</p>
          </div>
        ) : error ? (
          <div className="mx-auto max-w-lg rounded-xl border border-alert-red/30 bg-canvas-white p-24 text-center shadow-sm">
            <span className="material-symbols-outlined mb-12 inline-block text-36 text-alert-red">error</span>
            <h2 className="text-18 font-black text-alert-red">Error en Pantalla KDS</h2>
            <p className="mt-6 text-13 text-sage-green">{error}</p>
            <button
              type="button"
              onClick={() => router.push("/dashboard")}
              className="mt-16 min-h-44 cursor-pointer rounded-lg bg-plain-green px-16 py-8 text-13 font-bold text-canvas-white transition-colors hover:bg-plain-green-muted"
            >
              Volver al Dashboard
            </button>
          </div>
        ) : (
          /* TABLERO KANBAN DE 3 COLUMNAS */
          <div className="grid min-h-[calc(100vh-120px)] grid-cols-1 items-start gap-16 md:grid-cols-3">
            {/* COLUMNA 1: PENDIENTES */}
            <section className="flex min-h-[500px] flex-col rounded-xl border border-[#F3D4A3] bg-[#FFF9EF] p-12">
              {/* HEADER COLUMNA */}
              <div className="mb-12 flex items-center justify-between border-b border-[#F3D4A3] pb-12">
                <div className="flex items-center gap-8">
                  <span className="flex h-40 w-40 items-center justify-center rounded-lg bg-[#FFF1D8] text-[#A96100]">
                    <span className="material-symbols-outlined text-20">pending_actions</span>
                  </span>
                  <h2 className="text-16 font-black tracking-wide text-ash-graphite uppercase">
                    Pendientes
                  </h2>
                </div>
                <span className="rounded-full border border-[#F3D4A3] bg-[#FFF1D8] px-10 py-3 font-mono text-12 font-black text-[#9A5700]">
                  {pendientes.length}
                </span>
              </div>

              {/* LISTA DE COMANDAS */}
              <div className="space-y-12 flex-1">
                {pendientes.length === 0 ? (
                  <div className="flex flex-col items-center justify-center py-48 text-center text-sage-green">
                    <span className="material-symbols-outlined mb-8 text-32 text-[#B86B00]">check_circle</span>
                    <p className="text-14 font-bold text-ash-graphite">Sin comandas pendientes</p>
                    <p className="mt-2 text-11">Nuevos pedidos ingresarán automáticamente</p>
                  </div>
                ) : (
                  pendientes.map((grupo) => {
                    const mesaInfo = mesasMap[grupo.mesaId];
                    return (
                      <ComandaCard
                        key={grupo.id}
                        pedidos={grupo.pedidos}
                        numeroMesa={mesaInfo?.numero ?? 0}
                        nombreSector={mesaInfo?.sector}
                        onCambiarEstadoItem={handleCambiarEstadoItem}
                        onComandasCompletasLista={handleComandasCompletasLista}
                        onDespacharComandas={handleDespacharComandas}
                      />
                    );
                  })
                )}
              </div>
            </section>

            {/* COLUMNA 2: EN PREPARACIÓN */}
            <section className="flex min-h-[500px] flex-col rounded-xl border border-[#C9DCF7] bg-[#F4F8FD] p-12">
              {/* HEADER COLUMNA */}
              <div className="mb-12 flex items-center justify-between border-b border-[#C9DCF7] pb-12">
                <div className="flex items-center gap-8">
                  <span className="flex h-40 w-40 items-center justify-center rounded-lg bg-[#EAF3FF] text-[#2D6FB7]">
                    <span className="material-symbols-outlined text-20">skillet</span>
                  </span>
                  <h2 className="text-16 font-black tracking-wide text-[#285F9F] uppercase">
                    En Preparación
                  </h2>
                </div>
                <span className="rounded-full border border-[#C9DCF7] bg-[#EAF3FF] px-10 py-3 font-mono text-12 font-black text-[#285F9F]">
                  {enPreparacion.length}
                </span>
              </div>

              {/* LISTA DE COMANDAS */}
              <div className="space-y-12 flex-1">
                {enPreparacion.length === 0 ? (
                  <div className="flex flex-col items-center justify-center py-48 text-center text-sage-green">
                    <span className="material-symbols-outlined mb-8 text-32 text-[#4D8EDB]">skillet</span>
                    <p className="text-14 font-bold text-ash-graphite">Nada en preparación</p>
                    <p className="mt-2 text-11">Tocá un ítem pendiente para comenzar a elaborarlo</p>
                  </div>
                ) : (
                  enPreparacion.map((grupo) => {
                    const mesaInfo = mesasMap[grupo.mesaId];
                    return (
                      <ComandaCard
                        key={grupo.id}
                        pedidos={grupo.pedidos}
                        numeroMesa={mesaInfo?.numero ?? 0}
                        nombreSector={mesaInfo?.sector}
                        onCambiarEstadoItem={handleCambiarEstadoItem}
                        onComandasCompletasLista={handleComandasCompletasLista}
                        onDespacharComandas={handleDespacharComandas}
                      />
                    );
                  })
                )}
              </div>
            </section>

            {/* COLUMNA 3: LISTOS PARA SERVIR */}
            <section className="flex min-h-[500px] flex-col rounded-xl border border-[#B7EAD8] bg-[#F0FBF7] p-12">
              {/* HEADER COLUMNA */}
              <div className="mb-12 flex items-center justify-between border-b border-[#B7EAD8] pb-12">
                <div className="flex items-center gap-8">
                  <span className="flex h-40 w-40 items-center justify-center rounded-lg bg-[#E4F7F0] text-[#087657]">
                    <span className="material-symbols-outlined text-20">notifications</span>
                  </span>
                  <h2 className="text-16 font-black tracking-wide text-[#087657] uppercase">
                    Listos para Retirar
                  </h2>
                </div>
                <span className="rounded-full border border-[#B7EAD8] bg-[#E4F7F0] px-10 py-3 font-mono text-12 font-black text-[#087657]">
                  {listos.length}
                </span>
              </div>

              {/* LISTA DE COMANDAS */}
              <div className="space-y-12 flex-1">
                {listos.length === 0 ? (
                  <div className="flex flex-col items-center justify-center py-48 text-center text-sage-green">
                    <span className="material-symbols-outlined mb-8 text-32 text-[#14A77B]">notifications</span>
                    <p className="text-14 font-bold text-ash-graphite">Sin comandas listas</p>
                    <p className="mt-2 text-11">Los pedidos completados aguardando retiro aparecerán aquí</p>
                  </div>
                ) : (
                  listos.map((grupo) => {
                    const mesaInfo = mesasMap[grupo.mesaId];
                    return (
                      <ComandaCard
                        key={grupo.id}
                        pedidos={grupo.pedidos}
                        numeroMesa={mesaInfo?.numero ?? 0}
                        nombreSector={mesaInfo?.sector}
                        onCambiarEstadoItem={handleCambiarEstadoItem}
                        onComandasCompletasLista={handleComandasCompletasLista}
                        onDespacharComandas={handleDespacharComandas}
                      />
                    );
                  })
                )}
              </div>
            </section>
          </div>
        )}
      </main>

      {/* DRAWER HISTORIAL DE DESPACHOS (US-66) */}
      <HistorialDespachoDrawer
        isOpen={drawerHistorialAbierto}
        onClose={() => setDrawerHistorialAbierto(false)}
        pedidosDespachados={pedidosDespachados}
        mesasMap={mesasMap}
        onReabrir={handleReabrirComanda}
      />
    </div>
  );
}
