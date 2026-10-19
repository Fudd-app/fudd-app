"use client";

import { useEffect, useMemo, useState } from "react";
import { api, ArticuloAPI, CategoriaAPI, getErrorMessage } from "@/lib/api";

export default function CartaLecturaSection() {
  const [categorias, setCategorias] = useState<CategoriaAPI[]>([]);
  const [articulos, setArticulos] = useState<ArticuloAPI[]>([]);
  const [error, setError] = useState('');
  useEffect(() => { void Promise.all([api.listarCategorias(), api.listarArticulos()]).then(([c, a]) => { setCategorias(c); setArticulos(a); }).catch((err: unknown) => setError(getErrorMessage(err, 'No pudimos cargar la carta.'))); }, []);
  const porCategoria = useMemo(() => new Map(categorias.map((c) => [c.id, articulos.filter((a) => a.categoria_id === c.id)])), [articulos, categorias]);
  return <div className="h-full overflow-y-auto bg-ghost-fog/45 p-16 font-inter sm:p-24 md:p-32"><p className="text-11 font-mono uppercase tracking-[0.14em] text-sage-green">Carta · solo lectura</p><h2 className="mt-4 text-28 font-semibold text-ash-graphite">Disponibilidad de cocina</h2><p className="mt-4 text-13 text-sage-green">Podés consultar productos y variantes. Los cambios se realizan desde salón.</p>{error && <p className="mt-16 rounded-lg bg-warm-pink/20 p-12 text-alert-red">{error}</p>}<div className="mt-24 space-y-18">{categorias.map((c) => <section key={c.id} className="overflow-hidden rounded-xl border border-concrete bg-white"><h3 className="border-b border-concrete px-16 py-12 text-15 font-semibold">{c.nombre}</h3><div className="divide-y divide-ghost-fog">{(porCategoria.get(c.id) ?? []).map((a) => <article key={a.id} className="flex items-center justify-between gap-12 px-16 py-12"><div><p className="text-13 font-semibold">{a.nombre}</p><p className="text-11 text-sage-green">{a.descripcion}</p></div><span className={`rounded-full px-9 py-3 text-11 font-semibold ${a.disponible === false ? 'bg-warm-pink/30 text-alert-red' : 'bg-success/15 text-[#087645]'}`}>{a.disponible === false ? 'Agotado' : 'Disponible'}</span></article>)}</div></section>)}</div></div>;
}
