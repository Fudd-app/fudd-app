"use client";

import { useEffect, useMemo, useState } from "react";
import { api, ArticuloAPI, getErrorMessage, MesaAPI, VariantePublica } from "@/lib/api";

interface PedidoAsistidoModalProps {
  abierto: boolean;
  mesas: MesaAPI[];
  mesaInicialId?: string;
  onClose: () => void;
  onCreado: () => void;
}

export default function PedidoAsistidoModal({ abierto, mesas, mesaInicialId, onClose, onCreado }: PedidoAsistidoModalProps) {
  const [articulos, setArticulos] = useState<ArticuloAPI[]>([]);
  const [variantes, setVariantes] = useState<VariantePublica[]>([]);
  const [mesaId, setMesaId] = useState('');
  const [articuloId, setArticuloId] = useState('');
  const [cantidad, setCantidad] = useState(1);
  const [alias, setAlias] = useState('');
  const [notas, setNotas] = useState('');
  const [seleccionadas, setSeleccionadas] = useState<string[]>([]);
  const [error, setError] = useState('');
  const [guardando, setGuardando] = useState(false);
  const [aperturaPrevia, setAperturaPrevia] = useState({ abierto: false, mesaInicialId });

  // Al abrir se parte de la mesa inicial. Se ajusta durante el render en lugar de en un
  // efecto para evitar un render extra. No se compara `mesas`: los padres pasan un array
  // nuevo en cada render (SSE, polling) y eso pisaría la mesa que eligió el mozo.
  if (aperturaPrevia.abierto !== abierto || aperturaPrevia.mesaInicialId !== mesaInicialId) {
    setAperturaPrevia({ abierto, mesaInicialId });
    if (abierto) {
      setMesaId(mesaInicialId ?? '');
      setError('');
    }
  }

  useEffect(() => {
    if (!abierto) return;
    void api.listarArticulos().then((lista) => {
      const visibles = lista.filter((a) => a.activo);
      setArticulos(visibles);
      setArticuloId((actual) => actual || visibles[0]?.id || '');
    }).catch((err: unknown) => setError(getErrorMessage(err, 'No pudimos cargar la carta.')));
  }, [abierto]);

  useEffect(() => {
    if (!articuloId) return;
    void api.listarVariantes(articuloId).then(setVariantes).catch(() => setVariantes([]));
  }, [articuloId]);

  const articulo = useMemo(() => articulos.find((a) => a.id === articuloId), [articuloId, articulos]);
  // Sin mesa elegida se propone la primera activa. Si la elegida deja de estar activa
  // (SSE, polling) no se cambia en silencio por otra: queda vacía y el mozo vuelve a elegir.
  const mesaElegidaActiva = mesas.some((mesa) => mesa.id === mesaId);
  const mesaNoDisponible = mesaId !== '' && !mesaElegidaActiva;
  const mesaSeleccionada = mesaId === '' ? mesas[0]?.id ?? '' : mesaElegidaActiva ? mesaId : '';
  if (!abierto) return null;

  const enviar = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!mesaSeleccionada || !articuloId || cantidad < 1) { setError('Elegí mesa, producto y cantidad.'); return; }
    setGuardando(true); setError('');
    try {
      await api.crearPedidoAsistido({ mesa_id: mesaSeleccionada, alias_cliente: alias.trim(), items: [{ articulo_id: articuloId, cantidad, notas: notas.trim(), comensal_id: '', comensal_nombre: alias.trim(), variantes: seleccionadas }] });
      setAlias(''); setNotas(''); setCantidad(1); setSeleccionadas([]);
      onCreado(); onClose();
    } catch (err: unknown) { setError(getErrorMessage(err, 'No pudimos cargar el pedido.')); }
    finally { setGuardando(false); }
  };

  return <div className="fixed inset-0 z-50 flex items-end justify-center bg-ash-graphite/50 p-0 sm:items-center sm:p-20" role="dialog" aria-modal="true">
    <form onSubmit={enviar} className="max-h-[92vh] w-full max-w-[620px] overflow-y-auto rounded-t-2xl border border-concrete bg-canvas-white p-20 shadow-2xl sm:rounded-2xl sm:p-24">
      <div className="flex items-start justify-between gap-12"><div><p className="text-11 font-mono uppercase tracking-[0.14em] text-sage-green">Origen: mozo</p><h3 className="mt-4 text-22 font-semibold text-ash-graphite">Cargar pedido para una mesa</h3><p className="mt-4 text-12 text-sage-green">Usa la carta, validaciones y circuito KDS reales.</p></div><button type="button" onClick={onClose} className="h-40 w-40 rounded-lg hover:bg-ghost-fog"><span className="material-symbols-outlined">close</span></button></div>
      <div className="mt-20 grid gap-12 sm:grid-cols-2">
        <label className="text-12 font-semibold">Mesa<select value={mesaSeleccionada} onChange={(e) => setMesaId(e.target.value)} className="mt-6 h-44 w-full rounded-lg border border-concrete bg-white px-10 text-13">{mesaSeleccionada === '' && <option value="" disabled>Elegí una mesa</option>}{mesas.map((m) => <option key={m.id} value={m.id}>Mesa {m.numero}</option>)}</select>{mesaNoDisponible && <span role="alert" className="mt-4 block text-11 font-normal text-alert-red">La mesa elegida ya no está activa. Elegí otra.</span>}</label>
        <label className="text-12 font-semibold">Alias del cliente (opcional)<input value={alias} onChange={(e) => setAlias(e.target.value)} className="mt-6 h-44 w-full rounded-lg border border-concrete px-10 text-13" placeholder="Ej. Juli" /></label>
        <label className="text-12 font-semibold sm:col-span-2">Producto<select value={articuloId} onChange={(e) => { setArticuloId(e.target.value); setSeleccionadas([]); setVariantes([]); }} className="mt-6 h-44 w-full rounded-lg border border-concrete bg-white px-10 text-13">{articulos.map((a) => <option key={a.id} value={a.id}>{a.nombre} · ${a.precio.toLocaleString()}</option>)}</select></label>
        <label className="text-12 font-semibold">Cantidad<input type="number" min={1} value={cantidad} onChange={(e) => setCantidad(Number(e.target.value))} className="mt-6 h-44 w-full rounded-lg border border-concrete px-10 text-13" /></label>
        <label className="text-12 font-semibold">Notas<input value={notas} onChange={(e) => setNotas(e.target.value)} className="mt-6 h-44 w-full rounded-lg border border-concrete px-10 text-13" placeholder="Sin cebolla…" /></label>
      </div>
      {variantes.length > 0 && <fieldset className="mt-16 rounded-xl border border-concrete p-14"><legend className="px-6 text-12 font-semibold">Variantes de {articulo?.nombre}</legend><div className="mt-8 grid gap-8 sm:grid-cols-2">{variantes.map((v) => <label key={v.id} className="flex min-h-40 items-center gap-8 rounded-lg bg-ghost-fog/60 px-10 text-12"><input type={v.seleccion_unica ? 'radio' : 'checkbox'} name={v.seleccion_unica ? v.grupo || 'variante' : undefined} checked={seleccionadas.includes(v.id)} onChange={(e) => setSeleccionadas((actual) => e.target.checked ? [...actual.filter((id) => !v.seleccion_unica || !variantes.some((otra) => otra.id === id && otra.grupo === v.grupo)), v.id] : actual.filter((id) => id !== v.id))}/><span>{v.nombre}{v.precio_adicional ? ` (+$${v.precio_adicional.toLocaleString()})` : ''}</span></label>)}</div></fieldset>}
      {error && <p role="alert" className="mt-14 rounded-lg bg-warm-pink/20 p-10 text-12 text-alert-red">{error}</p>}
      <div className="mt-20 flex gap-10"><button type="button" onClick={onClose} className="h-48 flex-1 rounded-lg border border-concrete text-13 font-semibold">Cancelar</button><button disabled={guardando || mesas.length === 0 || articulos.length === 0} className="h-48 flex-1 rounded-lg bg-plain-green text-13 font-semibold text-white disabled:opacity-50">{guardando ? 'Cargando…' : 'Enviar a cocina'}</button></div>
    </form>
  </div>;
}
