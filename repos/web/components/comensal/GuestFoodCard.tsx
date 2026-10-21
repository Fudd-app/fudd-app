"use client";

import type { VariantePublica } from "@/lib/api";

export interface GuestCardItem {
  id: string;
  nombre: string;
  descripcion?: string;
  precio: number;
  foto_url?: string;
  disponible?: boolean;
  variantes?: VariantePublica[];
  tag?: string;
}

interface GuestFoodCardProps {
  item: GuestCardItem;
  cantidadEnCarrito?: number;
  onSelect: (item: GuestCardItem) => void;
  formatPrice?: (price: number) => string;
}

export default function GuestFoodCard({
  item,
  cantidadEnCarrito = 0,
  onSelect,
  formatPrice = (p) => `$ ${p.toLocaleString("es-AR")}`,
}: GuestFoodCardProps) {
  const agotado = item.disponible === false;
  const tieneVariantes = Boolean(item.variantes && item.variantes.length > 0);
  const tagTexto = agotado
    ? "Agotado"
    : item.tag || (tieneVariantes ? "Opciones" : null);

  return (
    <article className={`food-card ${agotado ? "unavailable" : ""}`}>
      <button
        type="button"
        className="food-info"
        onClick={() => onSelect(item)}
        aria-label={`${item.nombre}, ${formatPrice(item.precio)}${agotado ? ", agotado" : ""}`}
      >
        {tagTexto && (
          <span className={`food-tag ${agotado ? "out-tag" : ""}`}>
            {tagTexto}
          </span>
        )}
        <h2>{item.nombre}</h2>
        {item.descripcion && <p>{item.descripcion}</p>}
        <strong>{formatPrice(item.precio)}</strong>
      </button>

      <button
        type="button"
        className="food-image"
        onClick={() => onSelect(item)}
        aria-label={`Ver y agregar ${item.nombre}`}
      >
        {item.foto_url ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={item.foto_url}
            alt={item.nombre}
            width={96}
            height={96}
            className={agotado ? "grayscale" : ""}
            loading="lazy"
            decoding="async"
          />
        ) : (
          <div className="food-image-placeholder" aria-hidden="true">
            <span className="material-symbols-outlined text-32">restaurant</span>
          </div>
        )}

        <span
          aria-hidden="true"
          className={`plus-badge ${
            agotado ? "out-badge" : cantidadEnCarrito > 0 ? "in-cart" : ""
          }`}
        >
          {agotado ? "✕" : cantidadEnCarrito > 0 ? cantidadEnCarrito : "+"}
        </span>
      </button>
    </article>
  );
}
