"use client";

import { useEffect, useMemo, useState } from "react";
import { api, ArticuloAPI, CategoriaAPI, getErrorMessage } from "@/lib/api";

export default function CartaLecturaSection() {
  const [categorias, setCategorias] = useState<CategoriaAPI[]>([]);
  const [articulos, setArticulos] = useState<ArticuloAPI[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    void Promise.all([api.listarCategorias(), api.listarArticulos()])
      .then(([categoriasResponse, articulosResponse]) => {
        setCategorias(categoriasResponse);
        setArticulos(articulosResponse);
      })
      .catch((err: unknown) =>
        setError(getErrorMessage(err, "No pudimos cargar la carta."))
      );
  }, []);

  const porCategoria = useMemo(
    () =>
      new Map(
        categorias.map((categoria) => [
          categoria.id,
          articulos.filter((articulo) => articulo.categoria_id === categoria.id),
        ])
      ),
    [articulos, categorias]
  );

  return (
    <div className="h-full overflow-y-auto bg-ghost-fog/45 p-16 font-inter sm:p-24 md:p-32">
      <p className="text-11 font-mono uppercase tracking-[0.14em] text-sage-green">
        Carta · solo lectura
      </p>
      <h2 className="mt-4 text-28 font-semibold text-ash-graphite">Carta del local</h2>
      <p className="mt-4 text-13 text-sage-green">
        Consultá los productos, precios y variantes publicados. Los cambios se realizan desde salón.
      </p>

      {error && (
        <p className="mt-16 rounded-lg bg-warm-pink/20 p-12 text-alert-red">{error}</p>
      )}

      <div className="mt-24 space-y-18">
        {categorias.map((categoria) => (
          <section
            key={categoria.id}
            className="overflow-hidden rounded-xl border border-concrete bg-white"
          >
            <h3 className="border-b border-concrete px-16 py-12 text-15 font-semibold">
              {categoria.nombre}
            </h3>
            <div className="divide-y divide-ghost-fog">
              {(porCategoria.get(categoria.id) ?? []).map((articulo) => (
                <article
                  key={articulo.id}
                  className="flex items-center justify-between gap-12 px-16 py-12"
                >
                  <div>
                    <p className="text-13 font-semibold">{articulo.nombre}</p>
                    <p className="text-11 text-sage-green">{articulo.descripcion}</p>
                  </div>
                  <span className="whitespace-nowrap font-mono text-12 font-semibold text-ash-graphite">
                    ${articulo.precio.toLocaleString("es-AR")}
                  </span>
                </article>
              ))}
            </div>
          </section>
        ))}
      </div>
    </div>
  );
}
