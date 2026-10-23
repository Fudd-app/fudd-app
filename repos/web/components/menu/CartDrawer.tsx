import type { CartItem } from "@/app/mesa/[token]/page";
import { BrandMark } from "./BrandHeader";
import type { MesaBranding } from "./BrandHeader";

interface Props {
  branding: MesaBranding;
  items: CartItem[];
  totalPrecio: number;
  enviando?: boolean;
  onSetCantidad: (id: string, cantidad: number) => void;
  onSetNota: (id: string, nota: string) => void;
  onVolver: () => void;
  onConfirmar: () => void;
}

export default function CartDrawer({
  branding,
  items,
  totalPrecio,
  enviando = false,
  onSetCantidad,
  onSetNota,
  onVolver,
  onConfirmar,
}: Props) {
  return (
    <div className="mesa-background flex min-h-screen flex-col font-inter">
      <header className="mesa-surface mesa-border sticky top-0 z-10 flex items-center justify-between border-b px-16 py-12 shadow-2xs">
        <div className="flex items-center gap-12">
          <button
            onClick={onVolver}
            aria-label="Volver al menú"
            className="mesa-muted mesa-subtle-surface flex h-44 w-44 items-center justify-center rounded-full text-20 leading-none transition-colors hover:text-[var(--mesa-primary)]"
          >
            ←
          </button>
          <BrandMark branding={branding} className="h-40 w-40" />
          <div>
            <h2 className="mesa-text text-15 font-medium leading-tight">Tu pedido</h2>
            <span className="mesa-muted text-12 font-mono">({items.length} {items.length === 1 ? 'ítem' : 'ítems'})</span>
          </div>
        </div>
        <button
          onClick={onVolver}
          className="mesa-primary px-8 text-12 font-medium hover:underline"
        >
          + Agregar más
        </button>
      </header>

      <div className="flex-1 px-16 py-16 max-w-lg mx-auto w-full space-y-12">
        {items.map(item => (
          <div key={item.id} className="mesa-surface mesa-border space-y-12 rounded-lg border p-16 shadow-2xs">
            <div className="flex items-center justify-between gap-8">
              <span className="mesa-text text-14 font-medium">{item.nombre}</span>
              <span className="mesa-text text-14 font-mono font-medium">
                ${(item.precio * item.cantidad).toLocaleString()}
              </span>
            </div>
            <div className="flex items-center justify-between">
              <span className="mesa-muted text-12 font-mono">${item.precio.toLocaleString()} c/u</span>
              <div className="mesa-subtle-surface mesa-border flex items-center gap-12 rounded-full border p-2">
                <button
                  onClick={() => onSetCantidad(item.id, item.cantidad - 1)}
                  aria-label="Disminuir cantidad"
                  className="mesa-surface mesa-muted flex h-44 w-44 items-center justify-center rounded-full text-16 font-semibold shadow-2xs transition-transform hover:text-[var(--mesa-primary)] active:scale-90"
                >
                  −
                </button>
                <span className="mesa-text w-20 text-center text-14 font-mono font-medium">
                  {item.cantidad}
                </span>
                <button
                  onClick={() => onSetCantidad(item.id, item.cantidad + 1)}
                  aria-label="Aumentar cantidad"
                  className="mesa-primary-bg flex h-44 w-44 items-center justify-center rounded-full text-16 font-semibold shadow-2xs transition-transform active:scale-90"
                >
                  +
                </button>
              </div>
            </div>
            <input
              className="mesa-subtle-surface mesa-muted mesa-border w-full rounded-md border px-12 py-8 text-12 outline-none transition-colors placeholder:text-slate-400 focus:border-[var(--mesa-primary)] focus:bg-[var(--mesa-surface)]"
              placeholder="Nota para el equipo (ej: sin sal, hielo aparte)..."
              value={item.nota}
              onChange={e => onSetNota(item.id, e.target.value)}
            />
          </div>
        ))}

        {items.length === 0 && (
          <div className="mesa-subtle-text py-48 text-center text-13">
            El carrito está vacío.
          </div>
        )}
      </div>

      <div className="mesa-surface mesa-border sticky bottom-0 z-10 mx-auto w-full max-w-lg space-y-12 border-t px-16 py-16 shadow-lg">
        <div className="mesa-text flex items-center justify-between text-16 font-medium">
          <span>Total a pagar</span>
          <span className="mesa-primary text-18 font-mono font-semibold">${totalPrecio.toLocaleString()}</span>
        </div>
        <button
          onClick={onConfirmar}
          disabled={items.length === 0 || enviando}
          className="mesa-primary-bg flex min-h-52 w-full items-center justify-center gap-8 rounded-lg px-16 py-14 text-15 font-semibold shadow-md transition-all active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-50"
        >
          <span>{enviando ? 'Enviando pedido...' : 'Confirmar pedido'}</span>
          <span className="text-16">→</span>
        </button>
      </div>
    </div>
  );
}

