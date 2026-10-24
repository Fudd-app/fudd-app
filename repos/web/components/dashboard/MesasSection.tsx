"use client";
import { useState, useEffect, useRef, useCallback } from "react";
import QRCode from "qrcode";
import { api, MesaAPI, Sucursal, EstadoPlan, esPlanLimitReached, detallePlanLimit, getErrorMessage } from "@/lib/api";
import { EmptyState, Skeleton, useToast, useConfirm } from "@/components/ui";
import UpgradeModal from "./UpgradeModal";
import PedidoAsistidoModal from "./PedidoAsistidoModal";
import { useValorCliente } from "@/lib/useValorCliente";

interface QRCanvasProps {
  token: string;
  mesaNumero: number;
  compacto?: boolean;
  estado?: MesaAPI['estado'];
  onTomarPedido?: () => void;
}

const origenCliente = () => process.env.NEXT_PUBLIC_APP_URL?.trim().replace(/\/$/, '') || window.location.origin;
const origenServidor = () => '';

function QRCanvas({ token, mesaNumero, compacto = false, estado, onTomarPedido }: QRCanvasProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const toast = useToast();

  const origin = useValorCliente(origenCliente, origenServidor);
  const publicUrl = origin ? new URL(`/mesa/${encodeURIComponent(token)}`, `${origin}/`).toString() : '';
  const hostname = publicUrl ? new URL(publicUrl).hostname : '';
  const usaOrigenLocal = hostname === 'localhost' || hostname === '127.0.0.1' || hostname === '::1';

  useEffect(() => {
    if (publicUrl && canvasRef.current) {
      void QRCode.toCanvas(canvasRef.current, publicUrl, { width: 180, margin: 1 });
    }
  }, [publicUrl]);

  const handleDownload = () => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const link = document.createElement('a');
    link.download = `qr-mesa-${mesaNumero}.png`;
    link.href = canvas.toDataURL('image/png');
    link.click();
  };

  const handleCopy = async () => {
    if (!publicUrl) return;
    try {
      await navigator.clipboard.writeText(publicUrl);
      toast.success(`Enlace de Mesa ${mesaNumero} copiado.`);
    } catch {
      toast.error('No se pudo copiar el enlace.');
    }
  };

  if (compacto) {
    const mesaActiva = estado === 'activa';
    return (
      <div className="space-y-10">
        <div className="flex items-center justify-between gap-8 rounded-lg border border-ghost-fog bg-ghost-fog/50 p-10 text-12 text-sage-green">
          <span>Estado</span>
          <strong className="text-ash-graphite">{mesaActiva ? 'Activa' : 'Inactiva'}</strong>
        </div>

        <div className="rounded-lg border border-concrete bg-[#F7F8F8] p-10">
          <p className="mb-5 text-10 font-mono uppercase tracking-[0.12em] text-sage-green">Link de la carta</p>
          <p className="break-all text-11 font-mono leading-relaxed text-ash-graphite" title={publicUrl}>
            {publicUrl || 'Preparando enlace…'}
          </p>
        </div>

        <div className="grid grid-cols-2 gap-8">
          <a
            href={publicUrl || undefined}
            target="_blank"
            rel="noopener noreferrer"
            aria-disabled={!publicUrl}
            className="flex min-h-44 items-center justify-center gap-5 rounded-lg border border-concrete bg-canvas-white px-8 text-11 font-semibold text-ash-graphite transition-colors hover:border-stone hover:bg-ghost-fog aria-disabled:pointer-events-none aria-disabled:opacity-50"
          >
            <span className="material-symbols-outlined text-16">menu_book</span>
            Abrir carta
          </a>
          <button
            type="button"
            onClick={handleCopy}
            disabled={!publicUrl}
            className="flex min-h-44 items-center justify-center gap-5 rounded-lg border border-concrete bg-canvas-white px-8 text-11 font-semibold text-ash-graphite transition-colors hover:border-stone hover:bg-ghost-fog disabled:cursor-not-allowed disabled:opacity-50"
          >
            <span className="material-symbols-outlined text-16">content_copy</span>
            Copiar link
          </button>
        </div>

        <button
          type="button"
          onClick={onTomarPedido}
          disabled={!mesaActiva || !onTomarPedido}
          className="flex min-h-46 w-full items-center justify-center gap-6 rounded-lg bg-plain-green px-10 text-12 font-semibold text-white shadow-sm transition-colors hover:bg-plain-green-muted disabled:cursor-not-allowed disabled:opacity-45"
          title={mesaActiva ? 'Cargar un pedido asistido para esta mesa' : 'La mesa debe estar activa para cargar pedidos'}
        >
          <span className="material-symbols-outlined text-17">add_shopping_cart</span>
          Tomar pedido
        </button>

        {usaOrigenLocal && (
          <p className="flex items-start gap-5 rounded-lg border border-amber-300 bg-amber-50 px-9 py-7 text-10 leading-relaxed text-amber-800">
            <span className="material-symbols-outlined text-14">warning</span>
            Este link usa localhost y no abrirá desde otro celular.
          </p>
        )}
      </div>
    );
  }

  return (
    <div className="flex w-full flex-col gap-10">
      <div className="flex min-h-200 items-center justify-center rounded-xl border border-ghost-fog bg-[#F7F8F8] p-10">
        <canvas ref={canvasRef} className="h-auto w-full max-w-[180px] rounded-lg bg-white" />
      </div>
      {publicUrl && (
        <>
          <div className="flex h-40 w-full items-center gap-8 rounded-lg border border-ghost-fog bg-[#F7F8F8] px-10">
            <span className="min-w-0 flex-1 truncate text-10 font-mono text-sage-green" title={publicUrl}>
              {publicUrl}
            </span>
            <button
              type="button"
              onClick={handleCopy}
              className="flex h-32 w-32 shrink-0 items-center justify-center rounded-md text-sage-green transition-colors hover:bg-canvas-white hover:text-ash-graphite"
              aria-label={`Copiar enlace de Mesa ${mesaNumero}`}
              title="Copiar enlace"
            >
              <span className="material-symbols-outlined text-16">content_copy</span>
            </button>
          </div>
          {usaOrigenLocal && (
            <p className="flex items-start gap-6 rounded-lg border border-amber-300 bg-amber-50 px-10 py-8 text-10 leading-relaxed text-amber-800">
              <span className="material-symbols-outlined mt-0.5 text-14">warning</span>
              Este QR usa localhost y no se abrirá desde otro celular. Configurá NEXT_PUBLIC_APP_URL con la URL de red o pública.
            </p>
          )}
        </>
      )}
      <div className="grid grid-cols-2 gap-8">
        <a
          href={publicUrl}
          target="_blank"
          rel="noopener noreferrer"
          className="flex min-h-44 items-center justify-center gap-6 rounded-lg border border-concrete bg-canvas-white px-8 text-center text-11 font-semibold text-ash-graphite transition-colors hover:border-stone hover:bg-ghost-fog"
          aria-label={`Abrir menú de la Mesa ${mesaNumero} en una nueva pestaña`}
        >
          <span className="material-symbols-outlined text-16">open_in_new</span>
          <span>Abrir menú</span>
        </a>
        <button
          type="button"
          onClick={handleDownload}
          className="flex min-h-44 items-center justify-center gap-6 rounded-lg bg-[#1685F8] px-8 text-center text-11 font-semibold text-white shadow-2xs transition-colors hover:bg-[#0F73DB]"
        >
          <span className="material-symbols-outlined text-16">download</span>
          <span>Descargar QR</span>
        </button>
      </div>
    </div>
  );
}

export default function MesasSection({ readOnly = false }: { readOnly?: boolean }) {
  const toast = useToast();
  const confirmar = useConfirm();
  const [mesas, setMesas] = useState<MesaAPI[]>([]);
  const [sucursales, setSucursales] = useState<Sucursal[]>([]);
  const [loading, setLoading] = useState(true);
  const [mostrarFormMesa, setMostrarFormMesa] = useState(false);
  const [nuevoNumero, setNuevoNumero] = useState('');
  const [nuevaCapacidad, setNuevaCapacidad] = useState('4');
  const [errorMsg, setErrorMsg] = useState('');
  const [estadoPlan, setEstadoPlan] = useState<EstadoPlan | null>(null);
  const [modalUpgradeOpen, setModalUpgradeOpen] = useState(false);
  const [modalLimiteInfo, setModalLimiteInfo] = useState<{ limite?: number; uso?: number }>({});
  const [mesaPedidoId, setMesaPedidoId] = useState<string | null>(null);

  const cargarPlan = useCallback(async () => {
    try {
      const plan = await api.obtenerMiPlan();
      setEstadoPlan(plan);
    } catch (err) {
      console.warn("No se pudo cargar el plan en MesasSection:", err);
    }
  }, []);

  const cargarMesas = useCallback(async () => {
    try {
      setLoading(true);
      setErrorMsg('');
      const [mesasList, sucursalesList] = await Promise.all([
        api.listarMesas(),
        api.listarSucursales(),
      ]);

      if (sucursalesList && sucursalesList.length > 0) {
        setSucursales(sucursalesList);
      } else {
        setSucursales([]);
      }

      if (mesasList && mesasList.length > 0) {
        setMesas(mesasList);
      } else {
        setMesas([]);
      }
    } catch (err: unknown) {
      console.error("Error al cargar mesas desde la API:", err);
      setMesas([]);
      setErrorMsg(getErrorMessage(err, 'Error al conectar con la API de mesas.'));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    const timeoutId = window.setTimeout(() => {
      void cargarMesas();
			if (!readOnly) void cargarPlan();
    }, 0);

    return () => window.clearTimeout(timeoutId);
	}, [cargarMesas, cargarPlan, readOnly]);

  const limiteMesasAlcanzado = Boolean(
    estadoPlan &&
      estadoPlan.plan === "free" &&
      (estadoPlan.alcanzado?.mesas === true ||
        (estadoPlan.disponibles?.mesas !== undefined && estadoPlan.disponibles.mesas <= 0))
  );

  const handleAbrirCrearMesa = () => {
    if (limiteMesasAlcanzado) {
      setModalLimiteInfo({
        limite: estadoPlan?.limites?.mesas ?? 10,
        uso: estadoPlan?.uso?.mesas ?? mesas.length,
      });
      setModalUpgradeOpen(true);
      return;
    }
    setErrorMsg('');
    setMostrarFormMesa(true);
  };

  const handleCrearMesa = async (e: React.FormEvent) => {
    e.preventDefault();
    const num = parseInt(nuevoNumero, 10);
    const cap = parseInt(nuevaCapacidad, 10);

    if (isNaN(num) || num <= 0) {
      setErrorMsg('Ingresa un número de mesa válido.');
      return;
    }

    if (isNaN(cap) || cap <= 0) {
      setErrorMsg('Ingresa una capacidad válida.');
      return;
    }

    if (sucursales.length === 0) {
      setErrorMsg('No hay una sucursal registrada para crear mesas.');
      return;
    }

    setErrorMsg('');
    const sucursalId = sucursales[0].id;

    const yaExiste = mesas.some(
      mesa => mesa.sucursal_id === sucursalId && mesa.numero === num
    );
    if (yaExiste) {
      setErrorMsg(`Ya existe una mesa con el número ${num}.`);
      return;
    }

    try {
      const creada = await api.crearMesa({
        sucursal_id: sucursalId,
        numero: num,
        capacidad: cap,
      });
      setMesas(prev => [...prev, creada]);
      setNuevoNumero('');
      setNuevaCapacidad('4');
      setMostrarFormMesa(false);
      toast.success('Mesa creada correctamente.');
      void cargarPlan();
    } catch (err: unknown) {
      if (esPlanLimitReached(err)) {
        const detalle = detallePlanLimit(err);
        setModalLimiteInfo({
          limite: detalle?.limite ?? 10,
          uso: detalle?.uso ?? 10,
        });
        setModalUpgradeOpen(true);
        return;
      }
      setErrorMsg(getErrorMessage(err, 'Error al crear la mesa.'));
    }
  };

  const handleEliminarMesa = async (id: string) => {
    const ok = await confirmar({
      titulo: 'Eliminar mesa',
      mensaje: '¿Eliminás esta mesa? Se borrará el QR asociado y no podrá recuperarse.',
      labelAceptar: 'Sí, eliminar',
      variante: 'danger',
    });
    if (!ok) return;
    try {
      await api.eliminarMesa(id);
      setMesas(prev => prev.filter(m => m.id !== id));
      toast.success('Mesa eliminada.');
      void cargarPlan();
    } catch (err: unknown) {
      console.error("No se pudo eliminar mesa:", err);
      setErrorMsg(getErrorMessage(err, 'Error al eliminar la mesa.'));
    }
  };

  return (
    <div className="h-full space-y-24 overflow-y-auto bg-[#F7F8F8] p-16 font-inter sm:p-24 md:p-32">
      <div className="flex flex-col justify-between gap-16 sm:flex-row sm:items-center">
        <div className="min-w-0">
          <h2 className="text-24 font-semibold tracking-[-0.02em] text-ash-graphite sm:text-32">Mesas & Códigos QR</h2>
          <p className="mt-4 text-13 text-sage-green sm:text-14">
            {loading
              ? 'Cargando mesas...'
              : readOnly
                ? `${mesas.length} mesas disponibles para consultar la carta y tomar pedidos`
                : `${mesas.length} mesas configuradas con QR activo`}
          </p>
        </div>
		<div className="flex items-center gap-8">
			{readOnly && <span className="rounded-full border border-concrete bg-canvas-white px-10 py-4 text-11 text-sage-green">Sin edición de mesas</span>}
			{!readOnly && <>
          {limiteMesasAlcanzado && (
            <span
              className="inline-flex items-center gap-4 rounded-full border border-stone/30 bg-ghost-fog px-10 py-4 text-11 font-mono font-medium text-stone"
              title="Límite del plan Free alcanzado"
            >
              <span className="material-symbols-outlined text-14">lock</span>
              Límite alcanzado
            </span>
          )}
          <button
            onClick={handleAbrirCrearMesa}
            className={`flex h-48 shrink-0 items-center justify-center gap-8 whitespace-nowrap rounded-lg px-20 text-13 font-semibold shadow-sm transition-colors ${
              limiteMesasAlcanzado
                ? "border border-concrete bg-ghost-fog text-ash-graphite hover:border-stone hover:bg-stone/10"
                : "bg-plain-green text-canvas-white hover:bg-plain-green-muted"
            }`}
            aria-label={limiteMesasAlcanzado ? "Límite de mesas alcanzado. Ver opciones de plan Pro." : "Nueva mesa"}
          >
            <span className="material-symbols-outlined text-16">
              {limiteMesasAlcanzado ? "lock" : "add"}
            </span>
            Nueva mesa
			</button>
			</>}
        </div>
      </div>

      {errorMsg && (
        <div className="p-10 bg-red-50 border border-alert-red/30 rounded text-12 text-alert-red">
          {errorMsg}
        </div>
      )}

		{mostrarFormMesa && !readOnly && (
        <form onSubmit={handleCrearMesa} className="flex flex-col items-stretch gap-12 rounded-xl border border-concrete bg-canvas-white p-16 shadow-sm sm:flex-row sm:items-end sm:p-20">
          <div className="space-y-4 flex-1">
            <label className="text-11 font-mono text-sage-green uppercase">Número de Mesa</label>
            <input
              type="number"
              required
              min="1"
              autoFocus
              placeholder="Ej: 1"
              value={nuevoNumero}
              onChange={e => setNuevoNumero(e.target.value)}
              className="h-44 w-full rounded-lg border border-concrete bg-canvas-white px-12 text-13 outline-none focus:border-plain-green"
            />
          </div>
          <div className="space-y-4 flex-1">
            <label className="text-11 font-mono text-sage-green uppercase">Capacidad (Sillas)</label>
            <input
              type="number"
              required
              min="1"
              placeholder="4"
              value={nuevaCapacidad}
              onChange={e => setNuevaCapacidad(e.target.value)}
              className="h-44 w-full rounded-lg border border-concrete bg-canvas-white px-12 text-13 outline-none focus:border-plain-green"
            />
          </div>
          <div className="flex items-center gap-8 pt-4 sm:pt-0">
            <button
              type="submit"
              className="h-44 flex-1 rounded-lg bg-plain-green px-16 text-13 font-semibold text-canvas-white transition-colors hover:bg-plain-green-muted sm:flex-initial"
            >
              Guardar
            </button>
            <button
              type="button"
              onClick={() => {
                setErrorMsg('');
                setMostrarFormMesa(false);
              }}
              className="h-44 rounded-lg px-12 text-13 text-sage-green hover:bg-ghost-fog hover:text-ash-graphite"
            >
              Cancelar
            </button>
          </div>
        </form>
      )}

      {/* Skeleton de carga */}
      {loading && (
        <div className="grid grid-cols-1 gap-16 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          {[1, 2, 3, 4].map(i => (
            <div key={i} className="space-y-12 rounded-xl border border-concrete bg-canvas-white p-16 shadow-sm">
              <Skeleton className="h-14 w-2/3" />
              <Skeleton className="h-140 w-full" />
              <Skeleton className="h-10 w-1/2 mx-auto" />
            </div>
          ))}
        </div>
      )}

      {/* Grid de mesas */}
      {!loading && (
        <>
          <div className="grid grid-cols-1 gap-16 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
            {mesas.map(mesa => (
              <article key={mesa.id} className="overflow-visible rounded-xl border border-concrete bg-canvas-white shadow-sm transition-all hover:-translate-y-0.5 hover:shadow-md">
                <div className="flex items-start justify-between gap-10 px-14 pb-10 pt-14">
                  <div className="min-w-0">
                    <p className="truncate text-16 font-semibold text-ash-graphite">Mesa {mesa.numero}</p>
                    <p className="mt-4 flex items-center gap-4 text-11 text-sage-green">
                      <span className="material-symbols-outlined text-15">group</span>
                      {mesa.capacidad} personas
                    </p>
                  </div>
					{!readOnly && <details className="group relative shrink-0">
                    <summary
                      className="flex h-40 w-40 cursor-pointer list-none items-center justify-center rounded-lg text-sage-green transition-colors hover:bg-ghost-fog hover:text-ash-graphite [&::-webkit-details-marker]:hidden"
                      aria-label={`Acciones de Mesa ${mesa.numero}`}
                      title="Más acciones"
                    >
                      <span className="material-symbols-outlined text-20">more_vert</span>
                    </summary>
                    <div className="absolute right-0 top-44 z-20 min-w-160 rounded-lg border border-concrete bg-canvas-white p-4 shadow-lg">
                      <button
                        type="button"
                        onClick={() => handleEliminarMesa(mesa.id)}
                        className="flex min-h-40 w-full items-center gap-8 rounded-md px-10 text-left text-11 font-medium text-alert-red transition-colors hover:bg-warm-pink/20"
                      >
                        <span className="material-symbols-outlined text-17">delete</span>
                        Eliminar mesa
                      </button>
                    </div>
					</details>}
                </div>
				<div className="px-14 pb-14">
					<QRCanvas
            token={mesa.qr_token}
            mesaNumero={mesa.numero}
            compacto={readOnly}
            estado={mesa.estado}
            onTomarPedido={readOnly ? () => setMesaPedidoId(mesa.id) : undefined}
          />
                </div>
              </article>
            ))}
          </div>

          {mesas.length === 0 && (
            <EmptyState
              icon="table_restaurant"
              title="No hay mesas configuradas"
              description="Creá tu primera mesa para generar el código QR que los clientes escanean."
				actionLabel={readOnly ? undefined : "Nueva Mesa"}
				onAction={readOnly ? undefined : handleAbrirCrearMesa}
            />
          )}
        </>
      )}

		{!readOnly && <UpgradeModal
        isOpen={modalUpgradeOpen}
        onClose={() => setModalUpgradeOpen(false)}
        recurso="mesas"
        limite={modalLimiteInfo.limite ?? estadoPlan?.limites?.mesas ?? 10}
        uso={modalLimiteInfo.uso ?? estadoPlan?.uso?.mesas ?? mesas.length}
        onUpgradeSolicitado={() => void cargarPlan()}
		/>}
    {readOnly && (
      <PedidoAsistidoModal
        abierto={mesaPedidoId !== null}
        mesas={mesas.filter((mesa) => mesa.estado === 'activa')}
        mesaInicialId={mesaPedidoId ?? undefined}
        onClose={() => setMesaPedidoId(null)}
        onCreado={() => toast.success('Pedido enviado a cocina.')}
      />
    )}
    </div>
  );
}
