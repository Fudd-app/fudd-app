"use client";

import { useState, useMemo } from "react";
import ComensalIcon from "@/components/menu/ComensalIcon";
import type { VariantePublica } from "@/lib/api";
import type { GuestCardItem } from "./GuestFoodCard";

interface GuestDetailSheetProps {
  item: GuestCardItem;
  onClose: () => void;
  onAddToCart: (
    item: GuestCardItem,
    cantidad: number,
    variantes: VariantePublica[],
    nota: string
  ) => void;
  formatPrice?: (price: number) => string;
}

export default function GuestDetailSheet({
  item,
  onClose,
  onAddToCart,
  formatPrice = (p) => `$ ${p.toLocaleString("es-AR")}`,
}: GuestDetailSheetProps) {
  const [cantidad, setCantidad] = useState(1);
  const [nota, setNota] = useState("");

  // Agrupar variantes por grupo
  const grupos = useMemo(() => {
    const list = item.variantes || [];
    const map = new Map<string, VariantePublica[]>();
    for (const v of list) {
      const g = v.grupo?.trim() || "Opciones";
      if (!map.has(g)) map.set(g, []);
      map.get(g)!.push(v);
    }
    return Array.from(map.entries()).map(([nombreGrupo, items]) => ({
      nombreGrupo,
      seleccionUnica: items.some((i) => i.seleccion_unica) || items.length <= 4,
      items,
    }));
  }, [item.variantes]);

  // Estado de variantes seleccionadas
  const [seleccionadas, setSeleccionadas] = useState<Record<string, string[]>>(() => {
    const initial: Record<string, string[]> = {};
    for (const g of grupos) {
      if (g.seleccionUnica && g.items.length > 0) {
        // Seleccionar por defecto la primera opción si es requerida/única
        initial[g.nombreGrupo] = [g.items[0].id];
      } else {
        initial[g.nombreGrupo] = [];
      }
    }
    return initial;
  });

  const toggleVariante = (grupoNombre: string, varId: string, seleccionUnica: boolean) => {
    setSeleccionadas((prev) => {
      const actuales = prev[grupoNombre] || [];
      if (seleccionUnica) {
        return { ...prev, [grupoNombre]: [varId] };
      }
      if (actuales.includes(varId)) {
        return { ...prev, [grupoNombre]: actuales.filter((id) => id !== varId) };
      }
      return { ...prev, [grupoNombre]: [...actuales, varId] };
    });
  };

  // Calcular precio total con recargos de variantes
  const todasSeleccionadasList = useMemo(() => {
    const ids = Object.values(seleccionadas).flat();
    return (item.variantes || []).filter((v) => ids.includes(v.id));
  }, [item.variantes, seleccionadas]);

  const recargos = todasSeleccionadasList.reduce((acc, v) => acc + (v.precio_adicional || 0), 0);
  const precioUnitario = item.precio + recargos;
  const precioTotal = precioUnitario * cantidad;

  const handleConfirmar = () => {
    onAddToCart(item, cantidad, todasSeleccionadasList, nota.trim());
    onClose();
  };

  return (
    <div
      className="guest-sheet-layer"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <section
        className="guest-sheet"
        role="dialog"
        aria-modal="true"
        aria-label={`Personalizar ${item.nombre}`}
      >
        <span className="sheet-handle" aria-hidden="true" />
        <button
          type="button"
          className="sheet-close"
          aria-label="Cerrar detalle del plato"
          onClick={onClose}
        >
          <ComensalIcon name="close" size={18} />
        </button>

        {item.foto_url && (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            className="sheet-photo"
            src={item.foto_url}
            alt={item.nombre}
            loading="lazy"
            decoding="async"
          />
        )}

        <div className="sheet-content">
          {item.tag && <span className="food-tag">{item.tag}</span>}
          <h2>{item.nombre}</h2>
          {item.descripcion && <p>{item.descripcion}</p>}

          {/* Opciones / Grupos de variantes */}
          {grupos.map((grupo) => (
            <div key={grupo.nombreGrupo}>
              <div className="option-title">
                <span>
                  <strong>{grupo.nombreGrupo}</strong>
                  <small>{grupo.seleccionUnica ? "Elegí 1 opción" : "Opcional"}</small>
                </span>
                <b>{grupo.seleccionUnica ? "Elegí 1" : "Opcional"}</b>
              </div>

              <div className="space-y-2 mt-2">
                {grupo.items.map((v) => {
                  const check = (seleccionadas[grupo.nombreGrupo] || []).includes(v.id);
                  return (
                    <label
                      key={v.id}
                      className="radio-row"
                      onClick={() => toggleVariante(grupo.nombreGrupo, v.id, grupo.seleccionUnica)}
                    >
                      <div className="flex items-center gap-8">
                        <span>{v.nombre}</span>
                        {v.precio_adicional > 0 && (
                          <small className="text-muted">
                            (+{formatPrice(v.precio_adicional)})
                          </small>
                        )}
                      </div>
                      <input
                        type={grupo.seleccionUnica ? "radio" : "checkbox"}
                        name={grupo.nombreGrupo}
                        checked={check}
                        onChange={() => {}}
                        aria-label={v.nombre}
                      />
                    </label>
                  );
                })}
              </div>
            </div>
          ))}

          {/* Notas para la cocina */}
          <div className="mt-20">
            <label className="guest-field">
              <span>Aclaraciones para la cocina</span>
              <input
                type="text"
                value={nota}
                onChange={(e) => setNota(e.target.value)}
                placeholder="Ej. sin cebolla, aderezo aparte..."
                maxLength={120}
              />
            </label>
          </div>

          {/* Selector de cantidad y agregar */}
          <div className="mt-20 flex items-center gap-12">
            <div className="flex items-center border border-[var(--line)] rounded-xl h-52 px-12 gap-14 bg-[var(--canvas)]">
              <button
                type="button"
                aria-label="Disminuir cantidad"
                className="text-18 font-bold text-[var(--muted)] hover:text-[var(--ink)]"
                onClick={() => setCantidad((c) => Math.max(1, c - 1))}
                disabled={cantidad <= 1}
              >
                −
              </button>
              <span className="font-bold text-14 min-w-[20px] text-center" aria-live="polite">
                {cantidad}
              </span>
              <button
                type="button"
                aria-label="Aumentar cantidad"
                className="text-18 font-bold text-[var(--muted)] hover:text-[var(--ink)]"
                onClick={() => setCantidad((c) => c + 1)}
              >
                +
              </button>
            </div>

            <button
              type="button"
              className="guest-primary flex-1"
              onClick={handleConfirmar}
            >
              Agregar al pedido · {formatPrice(precioTotal)}
            </button>
          </div>
        </div>
      </section>
    </div>
  );
}
