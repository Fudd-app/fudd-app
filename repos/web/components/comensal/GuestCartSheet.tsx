"use client";

import ComensalIcon from "@/components/menu/ComensalIcon";
import type { CartItem } from "@/app/mesa/[token]/page";

interface GuestCartSheetProps {
  mesaNumero: number;
  dinerName?: string;
  items: CartItem[];
  totalPrecio: number;
  enviando?: boolean;
  onClose: () => void;
  onSetCantidad: (id: string, cantidad: number) => void;
  onConfirmar: () => void;
  formatPrice?: (price: number) => string;
}

export default function GuestCartSheet({
  mesaNumero,
  dinerName,
  items,
  totalPrecio,
  enviando = false,
  onClose,
  onSetCantidad,
  onConfirmar,
  formatPrice = (p) => `$ ${p.toLocaleString("es-AR")}`,
}: GuestCartSheetProps) {
  const initial = (dinerName?.trim() || "I").slice(0, 1).toUpperCase();

  return (
    <div
      className="guest-sheet-layer"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget && !enviando) onClose();
      }}
    >
      <section
        className="guest-sheet cart-sheet"
        role="dialog"
        aria-modal="true"
        aria-label={`Tu pedido de la Mesa ${mesaNumero}`}
      >
        <span className="sheet-handle" aria-hidden="true" />

        <div className="cart-title">
          <div>
            <small>Mesa {mesaNumero}</small>
            <h2>Tu pedido</h2>
          </div>
          <button
            type="button"
            className="sheet-close static"
            aria-label="Cerrar pedido"
            onClick={onClose}
            disabled={enviando}
          >
            <ComensalIcon name="close" size={18} />
          </button>
        </div>

        <div className="cart-person">
          <span aria-hidden="true">{initial}</span>
          <strong>Tus elecciones ({dinerName || "Invitado"})</strong>
        </div>

        <div className="mt-14 divide-y divide-[var(--line)] max-h-[48vh] overflow-y-auto">
          {items.map((item) => {
            const variantesTexto = (item.variantes || []).map((v) => v.nombre).join(" · ");

            return (
              <div className="cart-item" key={item.id}>
                <b>{item.cantidad}×</b>
                <span>
                  <strong>{item.nombre}</strong>
                  {variantesTexto && <small>{variantesTexto}</small>}
                  {item.nota && <small className="italic text-muted">“{item.nota}”</small>}
                </span>
                <em>{formatPrice(item.precio * item.cantidad)}</em>
                <button
                  type="button"
                  className="remove-btn"
                  onClick={() => onSetCantidad(item.id, 0)}
                  aria-label={`Quitar ${item.nombre}`}
                  title="Quitar"
                  disabled={enviando}
                >
                  <ComensalIcon name="close" size={16} />
                </button>
              </div>
            );
          })}

          {items.length === 0 && (
            <div className="py-24 text-center text-muted text-12">
              No agregaste ningún plato a este pedido todavía.
            </div>
          )}
        </div>

        <div className="cart-total">
          <span>Subtotal de esta ronda</span>
          <strong>{formatPrice(totalPrecio)}</strong>
        </div>

        <button
          type="button"
          className="guest-primary"
          onClick={onConfirmar}
          disabled={enviando || items.length === 0}
        >
          <ComensalIcon name="kitchen" size={18} />
          {enviando ? "Enviando comanda..." : "Confirmar y enviar a cocina"}
        </button>

        <button
          type="button"
          className="guest-link"
          onClick={onClose}
          disabled={enviando}
        >
          Seguir agregando platos
        </button>
      </section>
    </div>
  );
}
