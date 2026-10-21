# Frontend Sprints 1 y 2 — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implementar US-07 a US-18 — dashboard admin funcional (Carta, Mesas+QR, Equipo, Recepcionista) y flujo completo del cliente en `/mesa/[token]` con carrito flotante, todo con mock data.

**Architecture:** El dashboard usa sidebar fijo con 4 secciones, cada una como componente independiente con su propio estado local. El flujo del cliente usa `useReducer` en `app/mesa/[token]/page.tsx` para manejar la máquina de estados carta → carrito → seguimiento. Todo en `repos/web/`.

**Tech Stack:** Next.js 16 (App Router), React 19, Tailwind CSS v4, TypeScript estricto, qrcode (para QR en MesasSection).

---

## Mapa de archivos

| Archivo | Acción | Responsabilidad |
|---|---|---|
| `repos/web/lib/mock/menu.ts` | Crear | Tipos y datos mock de categorías e ítems |
| `repos/web/lib/mock/mesas.ts` | Crear | Tipos y datos mock de mesas con tokens |
| `repos/web/lib/mock/pedidos.ts` | Crear | Tipos y datos mock de pedidos activos |
| `repos/web/lib/mock/equipo.ts` | Crear | Tipos y datos mock del equipo |
| `repos/web/app/dashboard/page.tsx` | Reemplazar | Layout con sidebar funcional + sección activa |
| `repos/web/components/dashboard/CartaSection.tsx` | Crear | US-08: CRUD de carta |
| `repos/web/components/dashboard/MesasSection.tsx` | Crear | US-07: grid de mesas + QR descargable |
| `repos/web/components/dashboard/EquipoSection.tsx` | Crear | US-09: lista equipo + formulario invitación |
| `repos/web/components/dashboard/RecepcionistaSection.tsx` | Crear | US-10/11/12: pedidos activos + cambio de estado |
| `repos/web/app/mesa/[token]/page.tsx` | Crear | Página cliente con useReducer |
| `repos/web/components/menu/CategoriaNav.tsx` | Crear | Chips de navegación por categoría |
| `repos/web/components/menu/ItemCard.tsx` | Crear | Ítem del menú con botón + |
| `repos/web/components/menu/CartDrawer.tsx` | Crear | Vista completa del carrito |
| `repos/web/components/menu/SeguimientoView.tsx` | Crear | Seguimiento del pedido + pedir cuenta |

---

## Task 1: Instalar dependencia qrcode

**Files:**
- Modify: `repos/web/package.json` (vía npm install)

- [ ] **Step 1: Instalar qrcode**

```powershell
cd repos/web
npm install qrcode @types/qrcode
```

- [ ] **Step 2: Verificar que aparece en package.json**

```powershell
Select-String "qrcode" repos/web/package.json
```

Esperado: línea con `"qrcode"` en dependencies y `"@types/qrcode"` en devDependencies.

- [ ] **Step 3: Commit**

```powershell
git add repos/web/package.json repos/web/package-lock.json
git commit -m "chore: agrega dependencia qrcode para generación de QR de mesas"
```

---

## Task 2: Mock data — 4 archivos de datos

**Files:**
- Create: `repos/web/lib/mock/menu.ts`
- Create: `repos/web/lib/mock/mesas.ts`
- Create: `repos/web/lib/mock/pedidos.ts`
- Create: `repos/web/lib/mock/equipo.ts`

- [ ] **Step 1: Crear `repos/web/lib/mock/menu.ts`**

```typescript
export interface MenuItem {
  id: string;
  nombre: string;
  descripcion: string;
  precio: number;
  disponible: boolean;
}

export interface Categoria {
  id: string;
  nombre: string;
  items: MenuItem[];
}

export const mockMenu: Categoria[] = [
  {
    id: 'cat-1',
    nombre: 'Hamburguesas',
    items: [
      { id: 'item-1', nombre: 'Classic Burger', descripcion: 'Carne 200g, lechuga, tomate, cheddar', precio: 850, disponible: true },
      { id: 'item-2', nombre: 'Bacon Burger', descripcion: 'Carne 200g, bacon crocante, cebolla caramelizada', precio: 1050, disponible: true },
      { id: 'item-3', nombre: 'Veggie Burger', descripcion: 'Medallón de garbanzo, aguacate, rúcula', precio: 920, disponible: false },
      { id: 'item-4', nombre: 'Double Smash', descripcion: 'Dos medallones 120g, doble cheddar, salsa especial', precio: 1250, disponible: true },
    ],
  },
  {
    id: 'cat-2',
    nombre: 'Entradas',
    items: [
      { id: 'item-5', nombre: 'Papas fritas', descripcion: 'Porción grande con sal y pimentón', precio: 450, disponible: true },
      { id: 'item-6', nombre: 'Aros de cebolla', descripcion: 'Rebozados con salsa BBQ', precio: 520, disponible: true },
      { id: 'item-7', nombre: 'Nachos', descripcion: 'Con queso fundido, jalapeños y guacamole', precio: 680, disponible: true },
    ],
  },
  {
    id: 'cat-3',
    nombre: 'Bebidas',
    items: [
      { id: 'item-8', nombre: 'Gaseosa 500ml', descripcion: 'Coca-Cola, Sprite o Fanta', precio: 350, disponible: true },
      { id: 'item-9', nombre: 'Agua mineral', descripcion: 'Con o sin gas, 500ml', precio: 250, disponible: true },
      { id: 'item-10', nombre: 'Limonada natural', descripcion: 'Exprimida al momento, con menta', precio: 480, disponible: true },
      { id: 'item-11', nombre: 'Cerveza artesanal', descripcion: 'IPA o Stout, 473ml', precio: 750, disponible: true },
    ],
  },
];
```

- [ ] **Step 2: Crear `repos/web/lib/mock/mesas.ts`**

```typescript
export interface Mesa {
  id: string;
  numero: number;
  token: string;
  estado: 'disponible' | 'ocupada';
}

export const mockMesas: Mesa[] = [
  { id: 'mesa-1', numero: 1, token: 'mesa-token-1', estado: 'disponible' },
  { id: 'mesa-2', numero: 2, token: 'mesa-token-2', estado: 'ocupada' },
  { id: 'mesa-3', numero: 3, token: 'mesa-token-3', estado: 'disponible' },
  { id: 'mesa-4', numero: 4, token: 'mesa-token-4', estado: 'ocupada' },
  { id: 'mesa-5', numero: 5, token: 'mesa-token-5', estado: 'disponible' },
  { id: 'mesa-6', numero: 6, token: 'mesa-token-6', estado: 'disponible' },
  { id: 'mesa-7', numero: 7, token: 'mesa-token-7', estado: 'ocupada' },
  { id: 'mesa-8', numero: 8, token: 'mesa-token-8', estado: 'disponible' },
];
```

- [ ] **Step 3: Crear `repos/web/lib/mock/pedidos.ts`**

```typescript
export interface PedidoItem {
  id: string;
  nombre: string;
  cantidad: number;
  precio: number;
}

export interface Pedido {
  id: string;
  mesa: number;
  items: PedidoItem[];
  estado: 'recibido' | 'preparando' | 'listo';
  cuentaSolicitada: boolean;
  timestamp: string;
}

export const mockPedidos: Pedido[] = [
  {
    id: 'pedido-1',
    mesa: 2,
    items: [
      { id: 'pi-1', nombre: 'Classic Burger', cantidad: 2, precio: 850 },
      { id: 'pi-2', nombre: 'Papas fritas', cantidad: 2, precio: 450 },
    ],
    estado: 'recibido',
    cuentaSolicitada: false,
    timestamp: '20:15',
  },
  {
    id: 'pedido-2',
    mesa: 4,
    items: [
      { id: 'pi-3', nombre: 'Bacon Burger', cantidad: 1, precio: 1050 },
      { id: 'pi-4', nombre: 'Cerveza artesanal', cantidad: 2, precio: 750 },
    ],
    estado: 'preparando',
    cuentaSolicitada: false,
    timestamp: '20:05',
  },
  {
    id: 'pedido-3',
    mesa: 7,
    items: [
      { id: 'pi-5', nombre: 'Double Smash', cantidad: 3, precio: 1250 },
      { id: 'pi-6', nombre: 'Aros de cebolla', cantidad: 1, precio: 520 },
      { id: 'pi-7', nombre: 'Limonada natural', cantidad: 3, precio: 480 },
    ],
    estado: 'listo',
    cuentaSolicitada: true,
    timestamp: '19:50',
  },
];
```

- [ ] **Step 4: Crear `repos/web/lib/mock/equipo.ts`**

```typescript
export interface UsuarioEquipo {
  id: string;
  nombre: string;
  email: string;
  rol: 'admin' | 'encargado' | 'mozo';
  estado: 'activo' | 'invitado';
}

export const mockEquipo: UsuarioEquipo[] = [
  { id: 'u-1', nombre: 'Pablo Aguirre', email: 'pablo@mesaclick.com', rol: 'admin', estado: 'activo' },
  { id: 'u-2', nombre: 'Martín Oviedo', email: 'martin@mesaclick.com', rol: 'encargado', estado: 'activo' },
  { id: 'u-3', nombre: 'Mateo Silvestrin', email: 'mateo@mesaclick.com', rol: 'mozo', estado: 'invitado' },
];
```

- [ ] **Step 5: Verificar que TypeScript no tira errores**

```powershell
cd repos/web && npx tsc --noEmit
```

Esperado: sin errores.

- [ ] **Step 6: Commit**

```powershell
git add repos/web/lib/
git commit -m "feat: agrega mock data para menu, mesas, pedidos y equipo (Fase 1)"
```

---

## Task 3: Refactorizar dashboard con sidebar funcional

**Files:**
- Modify: `repos/web/app/dashboard/page.tsx` (reemplazar completamente)

- [ ] **Step 1: Reemplazar `repos/web/app/dashboard/page.tsx`**

```tsx
"use client";
import React, { useState } from "react";
import CartaSection from "@/components/dashboard/CartaSection";
import MesasSection from "@/components/dashboard/MesasSection";
import EquipoSection from "@/components/dashboard/EquipoSection";
import RecepcionistaSection from "@/components/dashboard/RecepcionistaSection";

type Section = 'carta' | 'mesas' | 'equipo' | 'recepcionista';

const SECTIONS: { id: Section; icon: string; label: string }[] = [
  { id: 'carta', icon: 'restaurant_menu', label: 'Carta' },
  { id: 'mesas', icon: 'table_restaurant', label: 'Mesas & QR' },
  { id: 'equipo', icon: 'group', label: 'Equipo' },
  { id: 'recepcionista', icon: 'receipt_long', label: 'Recepcionista' },
];

function renderSection(section: Section) {
  switch (section) {
    case 'carta': return <CartaSection />;
    case 'mesas': return <MesasSection />;
    case 'equipo': return <EquipoSection />;
    case 'recepcionista': return <RecepcionistaSection />;
  }
}

export default function DashboardPage() {
  const [activeSection, setActiveSection] = useState<Section>('carta');
  const [isExpanded, setIsExpanded] = useState(false);
  const [isUserMenuOpen, setIsUserMenuOpen] = useState(false);

  return (
    <div className="h-screen flex flex-col bg-canvas-white font-inter overflow-hidden">
      <header className="h-44 border-b border-system-black px-16 flex items-center justify-between shrink-0 bg-canvas-white z-30">
        <div className="flex items-center gap-8">
          <button
            onClick={() => setIsExpanded(!isExpanded)}
            className="material-symbols-outlined text-ash-graphite hover:text-plain-green transition-colors text-20 mr-4"
          >
            {isExpanded ? 'menu_open' : 'menu'}
          </button>
          <h1 className="text-15 font-medium tracking-tight">Mesa CLICK</h1>
          <span className="text-11 font-mono text-sage-green uppercase tracking-wider border-l border-ghost-fog pl-8">
            Admin
          </span>
        </div>
        <div className="flex items-center gap-16">
          <button className="material-symbols-outlined text-ash-graphite hover:text-plain-green transition-colors text-20">
            notifications
          </button>
          <div className="relative">
            <button
              onClick={() => setIsUserMenuOpen(!isUserMenuOpen)}
              className="w-28 h-28 bg-vanilla-cream border border-ash-graphite rounded-md flex items-center justify-center hover:border-plain-green transition-all"
            >
              <span className="material-symbols-outlined text-18 text-ash-graphite">person</span>
            </button>
            {isUserMenuOpen && (
              <>
                <div className="fixed inset-0 z-10" onClick={() => setIsUserMenuOpen(false)} />
                <div className="absolute right-0 mt-8 w-90 bg-canvas-white border border-system-black rounded-md shadow-sm z-20 py-8 overflow-hidden">
                  <div className="px-16 py-8 border-b border-ghost-fog mb-4">
                    <p className="text-11 font-medium text-ash-graphite">Administrador</p>
                    <p className="text-11 font-mono text-sage-green truncate">admin@mesaclick.com</p>
                  </div>
                  <UserMenuItem icon="account_circle" label="Perfil" />
                  <div className="mt-8 pt-8 border-t border-ghost-fog">
                    <UserMenuItem icon="logout" label="Salir" isDanger />
                  </div>
                </div>
              </>
            )}
          </div>
        </div>
      </header>

      <div className="flex-1 flex overflow-hidden">
        <aside
          className={`${
            isExpanded ? "w-200" : "w-64"
          } border-r border-system-black hidden md:flex flex-col py-16 transition-all duration-300 ease-in-out bg-canvas-white overflow-y-auto overflow-x-hidden shrink-0`}
        >
          <nav className={`space-y-4 flex-1 flex flex-col ${isExpanded ? "px-8" : "items-center"}`}>
            {SECTIONS.map((s) => (
              <NavItem
                key={s.id}
                icon={s.icon}
                label={s.label}
                active={activeSection === s.id}
                expanded={isExpanded}
                onClick={() => setActiveSection(s.id)}
              />
            ))}
          </nav>
          <div className={`pt-16 border-t border-ghost-fog space-y-4 flex flex-col ${isExpanded ? "px-8" : "items-center"}`}>
            <NavItem icon="settings" label="Configuración" expanded={isExpanded} />
            <NavItem icon="help_outline" label="Ayuda" expanded={isExpanded} />
          </div>
        </aside>

        <main className="flex-1 overflow-y-auto">
          {renderSection(activeSection)}
        </main>
      </div>
    </div>
  );
}

function NavItem({
  icon,
  label,
  active = false,
  expanded = false,
  onClick,
}: {
  icon: string;
  label: string;
  active?: boolean;
  expanded?: boolean;
  onClick?: () => void;
}) {
  return (
    <div
      title={!expanded ? label : undefined}
      onClick={onClick}
      className={`flex items-center rounded-md cursor-pointer transition-all ${
        expanded ? "w-full px-12 py-8 gap-12" : "justify-center w-40 h-40"
      } ${active ? "bg-ghost-fog text-plain-green" : "text-ash-graphite hover:bg-vanilla-cream"}`}
    >
      <span className="material-symbols-outlined text-20">{icon}</span>
      {expanded && <span className="text-13 font-medium whitespace-nowrap">{label}</span>}
    </div>
  );
}

function UserMenuItem({
  icon,
  label,
  isDanger = false,
}: {
  icon: string;
  label: string;
  isDanger?: boolean;
}) {
  return (
    <div
      className={`flex items-center gap-12 px-16 py-8 cursor-pointer transition-colors ${
        isDanger ? "text-alert-red hover:bg-red-50" : "text-ash-graphite hover:bg-vanilla-cream"
      }`}
    >
      <span className="material-symbols-outlined text-18">{icon}</span>
      <span className="text-13 font-medium">{label}</span>
    </div>
  );
}
```

> Nota: Este paso va a fallar el build de TypeScript porque los componentes `CartaSection`, `MesasSection`, `EquipoSection` y `RecepcionistaSection` aún no existen. Está bien — se crean en los próximos tasks. Verificar el build final recién en Task 7.

- [ ] **Step 2: Commit**

```powershell
git add repos/web/app/dashboard/page.tsx
git commit -m "feat: refactoriza dashboard con sidebar funcional y 4 secciones (US-07 al US-12)"
```

---

## Task 4: CartaSection — CRUD de carta (US-08)

**Files:**
- Create: `repos/web/components/dashboard/CartaSection.tsx`

- [ ] **Step 1: Crear `repos/web/components/dashboard/CartaSection.tsx`**

```tsx
"use client";
import { useState } from "react";
import { mockMenu, Categoria, MenuItem } from "@/lib/mock/menu";

type NuevoItemForm = { nombre: string; descripcion: string; precio: string };

export default function CartaSection() {
  const [categorias, setCategorias] = useState<Categoria[]>(mockMenu);
  const [nuevaCatNombre, setNuevaCatNombre] = useState('');
  const [mostrarFormCat, setMostrarFormCat] = useState(false);
  const [mostrarFormItem, setMostrarFormItem] = useState<string | null>(null);
  const [nuevoItem, setNuevoItem] = useState<NuevoItemForm>({ nombre: '', descripcion: '', precio: '' });

  const agregarCategoria = () => {
    if (!nuevaCatNombre.trim()) return;
    setCategorias(prev => [
      ...prev,
      { id: crypto.randomUUID(), nombre: nuevaCatNombre.trim(), items: [] },
    ]);
    setNuevaCatNombre('');
    setMostrarFormCat(false);
  };

  const eliminarCategoria = (catId: string) => {
    if (!confirm('¿Eliminar esta categoría y todos sus ítems?')) return;
    setCategorias(prev => prev.filter(c => c.id !== catId));
  };

  const agregarItem = (catId: string) => {
    if (!nuevoItem.nombre.trim() || !nuevoItem.precio) return;
    const item: MenuItem = {
      id: crypto.randomUUID(),
      nombre: nuevoItem.nombre.trim(),
      descripcion: nuevoItem.descripcion.trim(),
      precio: parseFloat(nuevoItem.precio),
      disponible: true,
    };
    setCategorias(prev =>
      prev.map(c => c.id === catId ? { ...c, items: [...c.items, item] } : c)
    );
    setNuevoItem({ nombre: '', descripcion: '', precio: '' });
    setMostrarFormItem(null);
  };

  const eliminarItem = (catId: string, itemId: string) => {
    if (!confirm('¿Eliminar este ítem?')) return;
    setCategorias(prev =>
      prev.map(c =>
        c.id === catId ? { ...c, items: c.items.filter(i => i.id !== itemId) } : c
      )
    );
  };

  const toggleDisponible = (catId: string, itemId: string) => {
    setCategorias(prev =>
      prev.map(c =>
        c.id === catId
          ? { ...c, items: c.items.map(i => i.id === itemId ? { ...i, disponible: !i.disponible } : i) }
          : c
      )
    );
  };

  const totalItems = categorias.reduce((n, c) => n + c.items.length, 0);

  return (
    <div className="p-24 md:p-32 space-y-24 overflow-y-auto h-full">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-20 font-medium text-ash-graphite">Gestión de Carta</h2>
          <p className="text-13 text-sage-green mt-4">
            {categorias.length} categorías · {totalItems} ítems
          </p>
        </div>
        <button
          onClick={() => setMostrarFormCat(true)}
          className="px-16 py-8 bg-plain-green text-ash-graphite text-13 font-medium rounded-md hover:opacity-90 transition-opacity"
        >
          + Nueva categoría
        </button>
      </div>

      {mostrarFormCat && (
        <div className="flex items-center gap-8 p-16 border border-plain-green rounded-md bg-ghost-fog">
          <input
            className="flex-1 px-12 py-8 text-13 rounded-md"
            placeholder="Nombre de la categoría"
            value={nuevaCatNombre}
            onChange={e => setNuevaCatNombre(e.target.value)}
            onKeyDown={e => e.key === 'Enter' && agregarCategoria()}
            autoFocus
          />
          <button
            onClick={agregarCategoria}
            className="px-12 py-8 bg-plain-green text-ash-graphite text-13 font-medium rounded-md"
          >
            Agregar
          </button>
          <button
            onClick={() => setMostrarFormCat(false)}
            className="px-12 py-8 text-sage-green text-13 hover:text-ash-graphite"
          >
            Cancelar
          </button>
        </div>
      )}

      <div className="space-y-16">
        {categorias.map(cat => (
          <div key={cat.id} className="border border-ash-graphite rounded-lg overflow-hidden">
            <div className="flex items-center justify-between px-20 py-12 bg-vanilla-cream border-b border-ash-graphite">
              <h3 className="text-15 font-medium text-ash-graphite">{cat.nombre}</h3>
              <div className="flex items-center gap-8">
                <button
                  onClick={() => {
                    setMostrarFormItem(cat.id);
                    setNuevoItem({ nombre: '', descripcion: '', precio: '' });
                  }}
                  className="px-10 py-4 text-12 font-medium text-plain-green-muted border border-plain-green-muted rounded-md hover:bg-ghost-fog"
                >
                  + Ítem
                </button>
                <button
                  onClick={() => eliminarCategoria(cat.id)}
                  className="px-10 py-4 text-12 font-medium text-alert-red border border-alert-red rounded-md hover:bg-red-50"
                >
                  Eliminar cat.
                </button>
              </div>
            </div>

            <div className="divide-y divide-ghost-fog">
              {cat.items.map(item => (
                <div key={item.id} className="flex items-center gap-16 px-20 py-12">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-8">
                      <span className="text-13 font-medium text-ash-graphite">{item.nombre}</span>
                      {!item.disponible && (
                        <span className="px-6 py-1 text-10 font-medium bg-vanilla-cream text-sage-green rounded-md border border-ghost-fog">
                          No disponible
                        </span>
                      )}
                    </div>
                    <p className="text-12 text-sage-green truncate">{item.descripcion}</p>
                  </div>
                  <span className="text-13 font-medium text-ash-graphite font-mono">
                    ${item.precio.toLocaleString()}
                  </span>
                  <button
                    onClick={() => toggleDisponible(cat.id, item.id)}
                    className="text-12 font-medium text-sage-green hover:text-ash-graphite underline"
                  >
                    {item.disponible ? 'Ocultar' : 'Mostrar'}
                  </button>
                  <button
                    onClick={() => eliminarItem(cat.id, item.id)}
                    className="text-12 text-alert-red hover:opacity-70 font-medium"
                  >
                    ✕
                  </button>
                </div>
              ))}

              {cat.items.length === 0 && mostrarFormItem !== cat.id && (
                <p className="px-20 py-12 text-13 text-sage-green italic">Sin ítems aún.</p>
              )}

              {mostrarFormItem === cat.id && (
                <div className="px-20 py-16 bg-ghost-fog space-y-8">
                  <div className="flex gap-8">
                    <input
                      className="flex-1 px-10 py-6 text-13 rounded-md"
                      placeholder="Nombre del ítem *"
                      value={nuevoItem.nombre}
                      onChange={e => setNuevoItem(p => ({ ...p, nombre: e.target.value }))}
                      autoFocus
                    />
                    <input
                      type="number"
                      className="w-100 px-10 py-6 text-13 rounded-md"
                      placeholder="Precio *"
                      value={nuevoItem.precio}
                      onChange={e => setNuevoItem(p => ({ ...p, precio: e.target.value }))}
                    />
                  </div>
                  <input
                    className="w-full px-10 py-6 text-13 rounded-md"
                    placeholder="Descripción (opcional)"
                    value={nuevoItem.descripcion}
                    onChange={e => setNuevoItem(p => ({ ...p, descripcion: e.target.value }))}
                  />
                  <div className="flex gap-8">
                    <button
                      onClick={() => agregarItem(cat.id)}
                      className="px-12 py-6 bg-plain-green text-ash-graphite text-13 font-medium rounded-md"
                    >
                      Agregar ítem
                    </button>
                    <button
                      onClick={() => setMostrarFormItem(null)}
                      className="px-12 py-6 text-sage-green text-13 hover:text-ash-graphite"
                    >
                      Cancelar
                    </button>
                  </div>
                </div>
              )}
            </div>
          </div>
        ))}

        {categorias.length === 0 && (
          <div className="text-center py-40 text-sage-green text-13">
            No hay categorías. Creá la primera para empezar.
          </div>
        )}
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Verificar en browser**

Arrancar o recargar `npm run dev` en `repos/web`. Ir a `http://localhost:3000/dashboard`.
Verificar:
- El sidebar muestra los 4 ítems (Carta, Mesas & QR, Equipo, Recepcionista)
- Al hacer clic en "Carta" se ve la lista de categorías mock
- Se puede agregar una nueva categoría
- Se puede agregar un ítem a una categoría
- Se puede ocultar/mostrar un ítem
- Se puede eliminar un ítem (con confirm)
- Se puede eliminar una categoría (con confirm)

- [ ] **Step 3: Commit**

```powershell
git add repos/web/components/dashboard/CartaSection.tsx
git commit -m "feat(US-08): CartaSection con CRUD de categorías e ítems (mock)"
```

---

## Task 5: MesasSection — Grid de mesas con QR descargable (US-07)

**Files:**
- Create: `repos/web/components/dashboard/MesasSection.tsx`

- [ ] **Step 1: Crear `repos/web/components/dashboard/MesasSection.tsx`**

```tsx
"use client";
import { useState, useEffect, useRef } from "react";
import QRCode from "qrcode";
import { mockMesas, Mesa } from "@/lib/mock/mesas";

function QRCanvas({ token }: { token: string }) {
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    if (!canvasRef.current) return;
    const url = `${window.location.origin}/mesa/${token}`;
    QRCode.toCanvas(canvasRef.current, url, { width: 150, margin: 1 });
  }, [token]);

  const handleDownload = () => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const link = document.createElement('a');
    link.download = `qr-mesa-${token}.png`;
    link.href = canvas.toDataURL('image/png');
    link.click();
  };

  return (
    <div className="flex flex-col items-center gap-8">
      <canvas ref={canvasRef} className="rounded-md" />
      <button
        onClick={handleDownload}
        className="w-full px-10 py-6 text-12 font-medium text-plain-green-muted border border-plain-green-muted rounded-md hover:bg-ghost-fog transition-colors"
      >
        ↓ Descargar QR
      </button>
    </div>
  );
}

export default function MesasSection() {
  const [mesas] = useState<Mesa[]>(mockMesas);
  const ocupadas = mesas.filter(m => m.estado === 'ocupada').length;

  return (
    <div className="p-24 md:p-32 space-y-24">
      <div>
        <h2 className="text-20 font-medium text-ash-graphite">Mesas & QR</h2>
        <p className="text-13 text-sage-green mt-4">
          {mesas.length} mesas · {ocupadas} ocupadas · {mesas.length - ocupadas} libres
        </p>
      </div>

      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-16">
        {mesas.map(mesa => (
          <div key={mesa.id} className="border border-ash-graphite rounded-lg p-16 space-y-12">
            <div className="flex items-center justify-between">
              <span className="text-15 font-medium text-ash-graphite">Mesa {mesa.numero}</span>
              <span
                className={`px-8 py-2 text-10 font-medium rounded-md border ${
                  mesa.estado === 'ocupada'
                    ? 'bg-vanilla-cream text-ash-graphite border-ash-graphite'
                    : 'bg-ghost-fog text-plain-green-muted border-plain-green-muted'
                }`}
              >
                {mesa.estado === 'ocupada' ? 'Ocupada' : 'Libre'}
              </span>
            </div>
            <QRCanvas token={mesa.token} />
            <p className="text-9 font-mono text-sage-green text-center break-all">{mesa.token}</p>
          </div>
        ))}
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Verificar en browser**

En `http://localhost:3000/dashboard`, hacer clic en "Mesas & QR".
Verificar:
- Grid de 8 cards, una por mesa
- Cada card muestra el QR generado
- El badge "Libre"/"Ocupada" aparece correctamente
- El botón "Descargar QR" descarga un PNG

- [ ] **Step 3: Commit**

```powershell
git add repos/web/components/dashboard/MesasSection.tsx
git commit -m "feat(US-07): MesasSection con QR por mesa descargable"
```

---

## Task 6: EquipoSection — Invitación de usuarios (US-09)

**Files:**
- Create: `repos/web/components/dashboard/EquipoSection.tsx`

- [ ] **Step 1: Crear `repos/web/components/dashboard/EquipoSection.tsx`**

```tsx
"use client";
import { useState } from "react";
import { mockEquipo, UsuarioEquipo } from "@/lib/mock/equipo";

type RolInvitable = 'encargado' | 'mozo';
type FormState = { nombre: string; email: string; rol: RolInvitable };

const ROL_LABELS: Record<UsuarioEquipo['rol'], string> = {
  admin: 'Admin',
  encargado: 'Encargado',
  mozo: 'Mozo',
};

export default function EquipoSection() {
  const [equipo, setEquipo] = useState<UsuarioEquipo[]>(mockEquipo);
  const [form, setForm] = useState<FormState>({ nombre: '', email: '', rol: 'mozo' });
  const [error, setError] = useState('');
  const [enviado, setEnviado] = useState(false);

  const handleInvitar = (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    if (!form.nombre.trim()) { setError('El nombre es requerido.'); return; }
    if (!form.email.trim() || !form.email.includes('@')) { setError('Email inválido.'); return; }

    setEquipo(prev => [
      ...prev,
      {
        id: crypto.randomUUID(),
        nombre: form.nombre.trim(),
        email: form.email.trim(),
        rol: form.rol,
        estado: 'invitado',
      },
    ]);
    setForm({ nombre: '', email: '', rol: 'mozo' });
    setEnviado(true);
    setTimeout(() => setEnviado(false), 3000);
  };

  return (
    <div className="p-24 md:p-32 space-y-32">
      <div>
        <h2 className="text-20 font-medium text-ash-graphite">Equipo</h2>
        <p className="text-13 text-sage-green mt-4">{equipo.length} miembros</p>
      </div>

      <div className="border border-ash-graphite rounded-lg overflow-hidden">
        <div className="px-20 py-10 bg-vanilla-cream border-b border-ash-graphite">
          <p className="text-11 font-mono text-sage-green uppercase tracking-wider">Miembros actuales</p>
        </div>
        <div className="divide-y divide-ghost-fog">
          {equipo.map(u => (
            <div key={u.id} className="flex items-center justify-between px-20 py-12">
              <div>
                <p className="text-13 font-medium text-ash-graphite">{u.nombre}</p>
                <p className="text-12 text-sage-green font-mono">{u.email}</p>
              </div>
              <div className="flex items-center gap-8">
                <span className="px-8 py-2 text-11 font-medium text-sage-green bg-vanilla-cream border border-ghost-fog rounded-md">
                  {ROL_LABELS[u.rol]}
                </span>
                {u.estado === 'invitado' && (
                  <span className="px-8 py-2 text-11 font-medium text-plain-green-muted bg-ghost-fog border border-plain-green-muted rounded-md">
                    Invitado
                  </span>
                )}
              </div>
            </div>
          ))}
        </div>
      </div>

      <div className="border border-ash-graphite rounded-lg overflow-hidden">
        <div className="px-20 py-10 bg-vanilla-cream border-b border-ash-graphite">
          <p className="text-11 font-mono text-sage-green uppercase tracking-wider">Invitar usuario</p>
        </div>
        <form onSubmit={handleInvitar} className="p-20 space-y-12">
          <div className="flex gap-12">
            <input
              className="flex-1 px-12 py-8 text-13 rounded-md"
              placeholder="Nombre completo *"
              value={form.nombre}
              onChange={e => setForm(p => ({ ...p, nombre: e.target.value }))}
            />
            <input
              type="email"
              className="flex-1 px-12 py-8 text-13 rounded-md"
              placeholder="Email *"
              value={form.email}
              onChange={e => setForm(p => ({ ...p, email: e.target.value }))}
            />
          </div>
          <div className="flex items-center gap-12">
            <select
              className="px-12 py-8 text-13 rounded-md bg-canvas-white"
              value={form.rol}
              onChange={e => setForm(p => ({ ...p, rol: e.target.value as RolInvitable }))}
            >
              <option value="encargado">Encargado</option>
              <option value="mozo">Mozo</option>
            </select>
            <button
              type="submit"
              className="px-16 py-8 bg-plain-green text-ash-graphite text-13 font-medium rounded-md hover:opacity-90 transition-opacity"
            >
              Enviar invitación
            </button>
          </div>
          {error && <p className="text-12 text-alert-red">{error}</p>}
          {enviado && <p className="text-12 text-plain-green-muted">✓ Invitación enviada correctamente.</p>}
        </form>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Verificar en browser**

En `http://localhost:3000/dashboard`, hacer clic en "Equipo".
Verificar:
- Lista los 3 usuarios mock
- Mateo Silvestrin muestra el badge "Invitado"
- El formulario valida nombre y email vacíos
- Al enviar, el nuevo usuario aparece en la lista con badge "Invitado"
- El mensaje de confirmación aparece 3 segundos y desaparece

- [ ] **Step 3: Commit**

```powershell
git add repos/web/components/dashboard/EquipoSection.tsx
git commit -m "feat(US-09): EquipoSection con lista de equipo e invitación de usuarios"
```

---

## Task 7: RecepcionistaSection — Pedidos activos y cambio de estado (US-10/11/12)

**Files:**
- Create: `repos/web/components/dashboard/RecepcionistaSection.tsx`

- [ ] **Step 1: Crear `repos/web/components/dashboard/RecepcionistaSection.tsx`**

```tsx
"use client";
import { useState } from "react";
import { mockPedidos, Pedido } from "@/lib/mock/pedidos";

const ESTADOS: Pedido['estado'][] = ['recibido', 'preparando', 'listo'];

const ESTADO_LABELS: Record<Pedido['estado'], string> = {
  recibido: 'Recibido',
  preparando: 'Preparando',
  listo: 'Listo',
};

const ESTADO_STYLES: Record<Pedido['estado'], string> = {
  recibido: 'bg-vanilla-cream text-ash-graphite border-ash-graphite',
  preparando: 'bg-ghost-fog text-sage-green border-sage-green',
  listo: 'bg-plain-green text-ash-graphite border-plain-green',
};

function PedidoCard({
  pedido,
  onAvanzar,
  onCerrar,
}: {
  pedido: Pedido;
  onAvanzar: (id: string) => void;
  onCerrar: (id: string) => void;
}) {
  const total = pedido.items.reduce((sum, i) => sum + i.precio * i.cantidad, 0);
  const esAlerta = pedido.cuentaSolicitada;

  return (
    <div className={`border rounded-lg overflow-hidden ${esAlerta ? 'border-alert-red' : 'border-ash-graphite'}`}>
      <div
        className={`flex items-center justify-between px-20 py-10 border-b ${
          esAlerta ? 'bg-red-50 border-alert-red' : 'bg-vanilla-cream border-ash-graphite'
        }`}
      >
        <div className="flex items-center gap-12">
          <span className="text-15 font-medium text-ash-graphite">Mesa {pedido.mesa}</span>
          {esAlerta && (
            <span className="text-12 font-medium text-alert-red">⚠️ Pide la cuenta</span>
          )}
          <span className="text-12 font-mono text-sage-green">{pedido.timestamp}</span>
        </div>
        <span className={`px-10 py-3 text-11 font-medium rounded-md border ${ESTADO_STYLES[pedido.estado]}`}>
          {ESTADO_LABELS[pedido.estado]}
        </span>
      </div>

      <div className="px-20 py-12 space-y-4">
        {pedido.items.map(item => (
          <div key={item.id} className="flex items-center justify-between text-13">
            <span className="text-ash-graphite">{item.cantidad}× {item.nombre}</span>
            <span className="font-mono text-sage-green">${(item.precio * item.cantidad).toLocaleString()}</span>
          </div>
        ))}
        <div className="flex items-center justify-between text-13 font-medium border-t border-ghost-fog pt-8 mt-4">
          <span className="text-ash-graphite">Total</span>
          <span className="font-mono text-ash-graphite">${total.toLocaleString()}</span>
        </div>
      </div>

      <div className="flex items-center gap-8 px-20 py-12 border-t border-ghost-fog">
        {pedido.estado !== 'listo' && (
          <button
            onClick={() => onAvanzar(pedido.id)}
            className="px-12 py-6 bg-plain-green text-ash-graphite text-12 font-medium rounded-md hover:opacity-90"
          >
            {pedido.estado === 'recibido' ? '→ Preparando' : '→ Listo'}
          </button>
        )}
        {pedido.estado === 'listo' && (
          <button
            onClick={() => onCerrar(pedido.id)}
            className="px-12 py-6 bg-ash-graphite text-canvas-white text-12 font-medium rounded-md hover:opacity-90"
          >
            Cerrar pedido
          </button>
        )}
      </div>
    </div>
  );
}

export default function RecepcionistaSection() {
  const [pedidos, setPedidos] = useState<Pedido[]>(mockPedidos);

  const avanzarEstado = (pedidoId: string) => {
    setPedidos(prev =>
      prev.map(p => {
        if (p.id !== pedidoId) return p;
        const idx = ESTADOS.indexOf(p.estado);
        if (idx >= ESTADOS.length - 1) return p;
        return { ...p, estado: ESTADOS[idx + 1] };
      })
    );
  };

  const cerrarPedido = (pedidoId: string) => {
    setPedidos(prev => prev.filter(p => p.id !== pedidoId));
  };

  const conAlerta = pedidos.filter(p => p.cuentaSolicitada);
  const sinAlerta = pedidos.filter(p => !p.cuentaSolicitada);

  return (
    <div className="p-24 md:p-32 space-y-24">
      <div>
        <h2 className="text-20 font-medium text-ash-graphite">Panel Recepcionista</h2>
        <p className="text-13 text-sage-green mt-4">{pedidos.length} pedidos activos</p>
      </div>

      {conAlerta.length > 0 && (
        <div className="space-y-8">
          <p className="text-11 font-mono text-sage-green uppercase tracking-wider">
            ⚠️ Solicitudes de cuenta
          </p>
          {conAlerta.map(p => (
            <PedidoCard key={p.id} pedido={p} onAvanzar={avanzarEstado} onCerrar={cerrarPedido} />
          ))}
        </div>
      )}

      {sinAlerta.length > 0 && (
        <div className="space-y-8">
          <p className="text-11 font-mono text-sage-green uppercase tracking-wider">Pedidos activos</p>
          {sinAlerta.map(p => (
            <PedidoCard key={p.id} pedido={p} onAvanzar={avanzarEstado} onCerrar={cerrarPedido} />
          ))}
        </div>
      )}

      {pedidos.length === 0 && (
        <div className="text-center py-40 text-sage-green text-13">
          No hay pedidos activos en este momento.
        </div>
      )}
    </div>
  );
}
```

- [ ] **Step 2: Verificar en browser**

En `http://localhost:3000/dashboard`, hacer clic en "Recepcionista".
Verificar:
- Mesa 7 aparece en la sección "Solicitudes de cuenta" con borde rojo y badge ⚠️
- Mesas 2 y 4 aparecen en "Pedidos activos"
- Mesa 2 (Recibido): botón "→ Preparando" — al hacer clic cambia estado a "Preparando"
- Mesa 4 (Preparando): botón "→ Listo" — al hacer clic cambia a "Listo" y aparece "Cerrar pedido"
- Al cerrar un pedido Listo, desaparece de la lista
- Si todos se cierran, aparece el mensaje "No hay pedidos activos"

- [ ] **Step 3: Commit**

```powershell
git add repos/web/components/dashboard/RecepcionistaSection.tsx
git commit -m "feat(US-10/11/12): RecepcionistaSection con pedidos, cambio de estado y alertas de cuenta"
```

---

## Task 8: Página cliente — setup con useReducer (US-13 base)

**Files:**
- Create: `repos/web/app/mesa/[token]/page.tsx`
- Create: `repos/web/components/menu/CategoriaNav.tsx`
- Create: `repos/web/components/menu/ItemCard.tsx`

- [ ] **Step 1: Crear `repos/web/components/menu/CategoriaNav.tsx`**

```tsx
import { Categoria } from "@/lib/mock/menu";

interface Props {
  categorias: Categoria[];
  activa: string;
  onSelect: (id: string) => void;
}

export default function CategoriaNav({ categorias, activa, onSelect }: Props) {
  return (
    <div className="flex gap-8 overflow-x-auto px-16 py-12">
      {categorias.map(cat => (
        <button
          key={cat.id}
          onClick={() => onSelect(cat.id)}
          className={`flex-shrink-0 px-16 py-8 text-13 font-medium rounded-full border transition-colors ${
            activa === cat.id
              ? 'bg-blue-600 text-white border-blue-600'
              : 'bg-white text-slate-600 border-slate-200 hover:border-slate-400'
          }`}
        >
          {cat.nombre}
        </button>
      ))}
    </div>
  );
}
```

- [ ] **Step 2: Crear `repos/web/components/menu/ItemCard.tsx`**

```tsx
import { MenuItem } from "@/lib/mock/menu";

interface Props {
  item: MenuItem;
  cantidad: number;
  onAgregar: () => void;
}

export default function ItemCard({ item, cantidad, onAgregar }: Props) {
  return (
    <div className="bg-white rounded-lg border border-slate-200 p-16 flex items-start justify-between gap-12">
      <div className="flex-1 min-w-0">
        <h3 className="text-14 font-medium text-slate-900">{item.nombre}</h3>
        {item.descripcion && (
          <p className="text-12 text-slate-500 mt-2">{item.descripcion}</p>
        )}
        <p className="text-14 font-medium text-blue-600 mt-8">${item.precio.toLocaleString()}</p>
      </div>
      <button
        onClick={onAgregar}
        className={`flex-shrink-0 w-32 h-32 rounded-full flex items-center justify-center text-16 font-medium transition-colors ${
          cantidad > 0
            ? 'bg-blue-600 text-white'
            : 'bg-slate-100 text-slate-600 hover:bg-slate-200'
        }`}
      >
        {cantidad > 0 ? cantidad : '+'}
      </button>
    </div>
  );
}
```

- [ ] **Step 3: Crear `repos/web/app/mesa/[token]/page.tsx`**

```tsx
"use client";
import { useReducer, useState, useEffect } from "react";
import { useParams } from "next/navigation";
import { mockMesas } from "@/lib/mock/mesas";
import { mockMenu } from "@/lib/mock/menu";
import CategoriaNav from "@/components/menu/CategoriaNav";
import ItemCard from "@/components/menu/ItemCard";
import CartDrawer from "@/components/menu/CartDrawer";
import SeguimientoView from "@/components/menu/SeguimientoView";

export type CartItem = {
  id: string;
  nombre: string;
  precio: number;
  cantidad: number;
  nota: string;
};

export type EstadoPedido = 'recibido' | 'preparando' | 'listo';
type Vista = 'carta' | 'carrito' | 'seguimiento';

type State = {
  items: CartItem[];
  vista: Vista;
  estadoPedido: EstadoPedido;
  cuentaSolicitada: boolean;
};

type Action =
  | { type: 'ADD_ITEM'; payload: { id: string; nombre: string; precio: number } }
  | { type: 'SET_CANTIDAD'; payload: { id: string; cantidad: number } }
  | { type: 'SET_NOTA'; payload: { id: string; nota: string } }
  | { type: 'SET_VISTA'; payload: Vista }
  | { type: 'CONFIRMAR_PEDIDO' }
  | { type: 'AVANZAR_ESTADO' }
  | { type: 'PEDIR_CUENTA' };

const INITIAL_STATE: State = {
  items: [],
  vista: 'carta',
  estadoPedido: 'recibido',
  cuentaSolicitada: false,
};

function reducer(state: State, action: Action): State {
  switch (action.type) {
    case 'ADD_ITEM': {
      const exists = state.items.find(i => i.id === action.payload.id);
      if (exists) {
        return {
          ...state,
          items: state.items.map(i =>
            i.id === action.payload.id ? { ...i, cantidad: i.cantidad + 1 } : i
          ),
        };
      }
      return {
        ...state,
        items: [...state.items, { ...action.payload, cantidad: 1, nota: '' }],
      };
    }
    case 'SET_CANTIDAD':
      if (action.payload.cantidad <= 0) {
        return { ...state, items: state.items.filter(i => i.id !== action.payload.id) };
      }
      return {
        ...state,
        items: state.items.map(i =>
          i.id === action.payload.id ? { ...i, cantidad: action.payload.cantidad } : i
        ),
      };
    case 'SET_NOTA':
      return {
        ...state,
        items: state.items.map(i =>
          i.id === action.payload.id ? { ...i, nota: action.payload.nota } : i
        ),
      };
    case 'SET_VISTA':
      return { ...state, vista: action.payload };
    case 'CONFIRMAR_PEDIDO':
      return { ...state, vista: 'seguimiento', estadoPedido: 'recibido' };
    case 'AVANZAR_ESTADO': {
      const next: EstadoPedido =
        state.estadoPedido === 'recibido' ? 'preparando' : 'listo';
      return { ...state, estadoPedido: next };
    }
    case 'PEDIR_CUENTA':
      return { ...state, cuentaSolicitada: true };
    default:
      return state;
  }
}

export default function MesaPage() {
  const params = useParams();
  const token = params.token as string;
  const mesa = mockMesas.find(m => m.token === token);

  const [state, dispatch] = useReducer(reducer, INITIAL_STATE);
  const [categoriaActiva, setCategoriaActiva] = useState<string>(mockMenu[0]?.id ?? '');

  useEffect(() => {
    if (state.vista !== 'seguimiento') return;
    const t1 = setTimeout(() => dispatch({ type: 'AVANZAR_ESTADO' }), 4000);
    const t2 = setTimeout(() => dispatch({ type: 'AVANZAR_ESTADO' }), 12000);
    return () => { clearTimeout(t1); clearTimeout(t2); };
  }, [state.vista]);

  if (!mesa) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-slate-50">
        <p className="text-slate-500 text-15">Mesa no encontrada.</p>
      </div>
    );
  }

  const totalItems = state.items.reduce((n, i) => n + i.cantidad, 0);
  const totalPrecio = state.items.reduce((n, i) => n + i.precio * i.cantidad, 0);
  const categoriaItems = mockMenu.find(c => c.id === categoriaActiva)?.items ?? [];

  if (state.vista === 'seguimiento') {
    return (
      <SeguimientoView
        items={state.items}
        estadoPedido={state.estadoPedido}
        cuentaSolicitada={state.cuentaSolicitada}
        mesa={mesa.numero}
        onAgregarMas={() => dispatch({ type: 'SET_VISTA', payload: 'carta' })}
        onPedirCuenta={() => dispatch({ type: 'PEDIR_CUENTA' })}
      />
    );
  }

  if (state.vista === 'carrito') {
    return (
      <CartDrawer
        items={state.items}
        totalPrecio={totalPrecio}
        onSetCantidad={(id, cantidad) => dispatch({ type: 'SET_CANTIDAD', payload: { id, cantidad } })}
        onSetNota={(id, nota) => dispatch({ type: 'SET_NOTA', payload: { id, nota } })}
        onVolver={() => dispatch({ type: 'SET_VISTA', payload: 'carta' })}
        onConfirmar={() => dispatch({ type: 'CONFIRMAR_PEDIDO' })}
      />
    );
  }

  return (
    <div className="min-h-screen bg-slate-50 pb-24">
      <header className="bg-white border-b border-slate-200 px-16 py-12 sticky top-0 z-10">
        <div className="max-w-lg mx-auto">
          <p className="text-11 text-slate-500 font-mono">Mesa {mesa.numero}</p>
          <h1 className="text-16 font-medium text-slate-900">Menú Digital</h1>
        </div>
      </header>

      <div className="max-w-lg mx-auto">
        <CategoriaNav
          categorias={mockMenu}
          activa={categoriaActiva}
          onSelect={setCategoriaActiva}
        />
        <div className="px-16 space-y-12">
          {categoriaItems.filter(i => i.disponible).map(item => (
            <ItemCard
              key={item.id}
              item={item}
              cantidad={state.items.find(i => i.id === item.id)?.cantidad ?? 0}
              onAgregar={() =>
                dispatch({ type: 'ADD_ITEM', payload: { id: item.id, nombre: item.nombre, precio: item.precio } })
              }
            />
          ))}
        </div>
      </div>

      {totalItems > 0 && (
        <div className="fixed bottom-0 left-0 right-0 bg-blue-600 px-16 py-12 z-20">
          <button
            onClick={() => dispatch({ type: 'SET_VISTA', payload: 'carrito' })}
            className="max-w-lg mx-auto flex items-center justify-between w-full text-white"
          >
            <span className="text-13 font-medium">🛒 {totalItems} {totalItems === 1 ? 'ítem' : 'ítems'}</span>
            <span className="text-13 font-medium">${totalPrecio.toLocaleString()} · Ver carrito →</span>
          </button>
        </div>
      )}
    </div>
  );
}
```

> Nota: Este paso fallará en build porque `CartDrawer` y `SeguimientoView` aún no existen. Se crean en Tasks 9 y 10.

- [ ] **Step 4: Commit**

```powershell
git add repos/web/app/mesa/ repos/web/components/menu/CategoriaNav.tsx repos/web/components/menu/ItemCard.tsx
git commit -m "feat(US-13/14): página cliente /mesa/[token] con useReducer, carta y carrito flotante"
```

---

## Task 9: CartDrawer — Vista completa del carrito (US-15)

**Files:**
- Create: `repos/web/components/menu/CartDrawer.tsx`

- [ ] **Step 1: Crear `repos/web/components/menu/CartDrawer.tsx`**

```tsx
import { CartItem } from "@/app/mesa/[token]/page";

interface Props {
  items: CartItem[];
  totalPrecio: number;
  onSetCantidad: (id: string, cantidad: number) => void;
  onSetNota: (id: string, nota: string) => void;
  onVolver: () => void;
  onConfirmar: () => void;
}

export default function CartDrawer({
  items,
  totalPrecio,
  onSetCantidad,
  onSetNota,
  onVolver,
  onConfirmar,
}: Props) {
  return (
    <div className="min-h-screen bg-slate-50 flex flex-col">
      <header className="bg-white border-b border-slate-200 px-16 py-12 flex items-center gap-12 sticky top-0 z-10">
        <button
          onClick={onVolver}
          className="text-slate-600 hover:text-slate-900 text-20 leading-none"
        >
          ←
        </button>
        <h2 className="text-15 font-medium text-slate-900">Tu pedido</h2>
        <span className="text-13 text-slate-500">({items.length} {items.length === 1 ? 'ítem' : 'ítems'})</span>
      </header>

      <div className="flex-1 px-16 py-16 max-w-lg mx-auto w-full space-y-12">
        {items.map(item => (
          <div key={item.id} className="bg-white rounded-lg border border-slate-200 p-16 space-y-10">
            <div className="flex items-center justify-between">
              <span className="text-14 font-medium text-slate-900">{item.nombre}</span>
              <span className="text-14 font-medium text-slate-700">
                ${(item.precio * item.cantidad).toLocaleString()}
              </span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-12 text-slate-500">${item.precio.toLocaleString()} c/u</span>
              <div className="flex items-center gap-12">
                <button
                  onClick={() => onSetCantidad(item.id, item.cantidad - 1)}
                  className="w-28 h-28 rounded-full bg-slate-100 text-slate-600 flex items-center justify-center hover:bg-slate-200 text-16 leading-none"
                >
                  −
                </button>
                <span className="text-14 font-medium text-slate-900 w-16 text-center">
                  {item.cantidad}
                </span>
                <button
                  onClick={() => onSetCantidad(item.id, item.cantidad + 1)}
                  className="w-28 h-28 rounded-full bg-blue-600 text-white flex items-center justify-center hover:bg-blue-700 text-16 leading-none"
                >
                  +
                </button>
              </div>
            </div>
            <input
              className="w-full px-10 py-6 text-12 text-slate-600 border border-slate-200 rounded-md bg-slate-50 placeholder-slate-400 outline-none focus:border-blue-400"
              placeholder="Nota para la cocina (opcional)"
              value={item.nota}
              onChange={e => onSetNota(item.id, e.target.value)}
            />
          </div>
        ))}
      </div>

      <div className="bg-white border-t border-slate-200 px-16 py-16 max-w-lg mx-auto w-full space-y-12">
        <div className="flex items-center justify-between text-15 font-medium text-slate-900">
          <span>Total</span>
          <span>${totalPrecio.toLocaleString()}</span>
        </div>
        <button
          onClick={onConfirmar}
          className="w-full py-14 bg-green-600 text-white text-15 font-medium rounded-lg hover:bg-green-700 transition-colors"
        >
          Enviar pedido
        </button>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Verificar en browser**

Ir a `http://localhost:3000/mesa/mesa-token-1`.
Verificar:
- Se ve la carta con categorías y los ítems de "Hamburguesas"
- El ítem "Veggie Burger" NO aparece (disponible: false)
- Al hacer clic en "+" aparece la barra azul en el fondo con el total
- Al tocar la barra azul se abre la vista del carrito
- En el carrito se pueden ajustar cantidades (0 elimina el ítem)
- Se puede escribir notas por ítem
- El botón "←" vuelve a la carta

- [ ] **Step 3: Commit**

```powershell
git add repos/web/components/menu/CartDrawer.tsx
git commit -m "feat(US-14/15): CartDrawer con ítems, cantidades, notas y confirmación"
```

---

## Task 10: SeguimientoView — Estado del pedido y pedir cuenta (US-16/17/18)

**Files:**
- Create: `repos/web/components/menu/SeguimientoView.tsx`

- [ ] **Step 1: Crear `repos/web/components/menu/SeguimientoView.tsx`**

```tsx
import { CartItem, EstadoPedido } from "@/app/mesa/[token]/page";

interface Props {
  items: CartItem[];
  estadoPedido: EstadoPedido;
  cuentaSolicitada: boolean;
  mesa: number;
  onAgregarMas: () => void;
  onPedirCuenta: () => void;
}

const PASOS: EstadoPedido[] = ['recibido', 'preparando', 'listo'];

const PASO_LABELS: Record<EstadoPedido, string> = {
  recibido: 'Pedido recibido',
  preparando: 'En preparación',
  listo: '¡Listo para retirar!',
};

const DEMORA_LABELS: Record<EstadoPedido, string> = {
  recibido: 'Estimado: ~15 min',
  preparando: 'Estimado: ~8 min',
  listo: '¡Tu pedido está listo!',
};

export default function SeguimientoView({
  items,
  estadoPedido,
  cuentaSolicitada,
  mesa,
  onAgregarMas,
  onPedirCuenta,
}: Props) {
  const total = items.reduce((n, i) => n + i.precio * i.cantidad, 0);
  const pasoActual = PASOS.indexOf(estadoPedido);

  return (
    <div className="min-h-screen bg-slate-50">
      <header className="bg-white border-b border-slate-200 px-16 py-12">
        <div className="max-w-lg mx-auto">
          <p className="text-11 text-slate-500 font-mono">Mesa {mesa}</p>
          <h1 className="text-16 font-medium text-slate-900">Estado de tu pedido</h1>
        </div>
      </header>

      <div className="max-w-lg mx-auto px-16 py-24 space-y-20">
        {/* Stepper */}
        <div className="bg-white rounded-lg border border-slate-200 p-20 space-y-16">
          <h2 className="text-14 font-medium text-slate-900">Estado actual</h2>
          <div className="space-y-14">
            {PASOS.map((paso, i) => {
              const activo = i === pasoActual;
              const completado = i < pasoActual;
              return (
                <div key={paso} className="flex items-start gap-12">
                  <div
                    className={`w-24 h-24 rounded-full flex items-center justify-center flex-shrink-0 text-12 font-medium mt-1 ${
                      completado
                        ? 'bg-green-500 text-white'
                        : activo
                        ? 'bg-blue-600 text-white'
                        : 'bg-slate-100 text-slate-400'
                    }`}
                  >
                    {completado ? '✓' : i + 1}
                  </div>
                  <div>
                    <p
                      className={`text-13 font-medium ${
                        activo
                          ? 'text-slate-900'
                          : completado
                          ? 'text-green-600'
                          : 'text-slate-400'
                      }`}
                    >
                      {PASO_LABELS[paso]}
                    </p>
                    {activo && (
                      <p className="text-12 text-slate-500 mt-1">{DEMORA_LABELS[paso]}</p>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        </div>

        {/* Resumen del pedido */}
        <div className="bg-white rounded-lg border border-slate-200 p-20 space-y-8">
          <h3 className="text-13 font-medium text-slate-900">Resumen</h3>
          {items.map(item => (
            <div key={item.id} className="flex justify-between text-13 text-slate-600">
              <span>{item.cantidad}× {item.nombre}</span>
              <span>${(item.precio * item.cantidad).toLocaleString()}</span>
            </div>
          ))}
          <div className="flex justify-between text-13 font-medium text-slate-900 border-t border-slate-100 pt-8">
            <span>Total</span>
            <span>${total.toLocaleString()}</span>
          </div>
        </div>

        {/* Acciones */}
        <div className="space-y-8">
          <button
            onClick={onAgregarMas}
            className="w-full py-12 border border-slate-300 text-slate-700 text-13 font-medium rounded-lg hover:bg-white transition-colors"
          >
            + Agregar más ítems
          </button>

          {!cuentaSolicitada ? (
            <button
              onClick={onPedirCuenta}
              className="w-full py-12 bg-slate-900 text-white text-13 font-medium rounded-lg hover:bg-slate-800 transition-colors"
            >
              Pedir la cuenta
            </button>
          ) : (
            <div className="w-full py-12 bg-green-50 border border-green-200 rounded-lg text-center">
              <p className="text-13 font-medium text-green-700">✓ El mozo fue notificado</p>
              <p className="text-12 text-green-600 mt-2">Enseguida se acerca a tu mesa</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Verificar flujo completo del cliente**

Ir a `http://localhost:3000/mesa/mesa-token-2` (mesa 2, estado: ocupada).
Verificar el flujo completo:
1. Agregar ítems → aparece barra azul → tocar barra → carrito
2. Ajustar cantidades, agregar notas → "Enviar pedido"
3. Aparece vista de seguimiento con estado "Recibido"
4. A los 4 segundos cambia a "Preparando"
5. A los 12 segundos cambia a "Listo"
6. "Agregar más ítems" vuelve a la carta sin borrar el carrito
7. "Pedir la cuenta" muestra confirmación y deshabilita el botón

Verificar también `http://localhost:3000/mesa/token-inexistente` → muestra "Mesa no encontrada".

- [ ] **Step 3: Verificar build sin errores**

```powershell
cd repos/web && npx tsc --noEmit
```

Esperado: sin errores de TypeScript.

- [ ] **Step 4: Commit final**

```powershell
git add repos/web/components/menu/SeguimientoView.tsx
git commit -m "feat(US-16/17/18): SeguimientoView con stepper, agregar ítems y pedir cuenta"
```

---

## Verificación final

- [ ] Recorrer el happy path del admin: `/dashboard` → Carta (CRUD) → Mesas & QR (descargar QR) → Equipo (invitar) → Recepcionista (avanzar estado)
- [ ] Recorrer el happy path del cliente: `/mesa/mesa-token-1` → agregar ítems → carrito → confirmar → seguimiento → avanzar auto → pedir cuenta
- [ ] `npx tsc --noEmit` sin errores
- [ ] Commit de cierre si hay cambios menores pendientes

```powershell
git add -A
git commit -m "feat: frontend Sprints 1 y 2 completo — US-07 a US-18 (Fase 1 mock)"
```
