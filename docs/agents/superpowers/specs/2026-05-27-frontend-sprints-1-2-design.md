# Diseño — Frontend Sprints 1 y 2 (US-07 a US-18)

**Fecha:** 2026-05-27  
**Fase:** 1 — Frontend sin backend  
**Alcance:** US-07 a US-18, todo con mock data, cero llamadas a API real

---

## Decisiones de diseño

| Pregunta | Decisión |
|---|---|
| Navegación del dashboard admin | Sidebar fijo, contenido dinámico — una sola ruta `/dashboard` |
| Vista del recepcionista | Sección dentro del sidebar del admin (no URL separada) |
| Flujo del cliente | Carta con carrito flotante (drawer inferior), estados internos |
| Tema del cliente | Light theme (fondo claro, tipografía oscura) |

---

## Routing

```
/dashboard          Panel admin (sidebar fijo, sección activa por useState)
/mesa/[token]       Carta del cliente (light theme, carrito flotante)
```

No se crean rutas adicionales. El estado de navegación dentro del dashboard y el estado del flujo del cliente viven en React state local.

---

## Dashboard Admin (`/dashboard`)

### Layout

`app/dashboard/page.tsx` se reemplaza por un componente con sidebar fijo a la izquierda y área de contenido principal a la derecha.

```
┌─────────────────┬──────────────────────────────┐
│ MESA CLICK      │                              │
│                 │   Contenido de la sección    │
│ 📋 Carta        │   activa                     │
│ 🪑 Mesas & QR   │                              │
│ 👥 Equipo       │                              │
│ 🧾 Recepcionista│                              │
└─────────────────┴──────────────────────────────┘
```

El ítem activo del sidebar se resalta. Al hacer clic cambia `activeSection` y se renderiza la sección correspondiente.

### Secciones

#### 📋 Carta (US-08)
- Lista de categorías con sus ítems
- Botón "Nueva categoría" → formulario inline (nombre)
- Botón "Nuevo ítem" por categoría → formulario inline (nombre, descripción, precio, disponible)
- Cada ítem tiene botones Editar y Eliminar
- Al editar: los campos del ítem se vuelven inputs in-place
- Al eliminar: confirmación simple antes de borrar del state
- Estado: `useState` con array de categorías + ítems anidados, inicializado desde `lib/mock/menu.ts`

#### 🪑 Mesas & QR (US-07)
- Grid de cards, una por mesa (Número de mesa, estado: disponible/ocupada)
- Cada card muestra el QR generado con la lib `qrcode` apuntando a `/mesa/[token]`
- Botón "Descargar QR" → descarga la imagen PNG del QR via canvas
- Estado: array de mesas desde `lib/mock/mesas.ts` (token incluido en cada mesa)

#### 👥 Equipo (US-09)
- Lista de usuarios actuales (nombre, rol, email) desde `lib/mock/equipo.ts`
- Formulario "Invitar usuario": nombre, email, rol (Encargado / Mozo)
- Al enviar: agrega el usuario al array con estado "Invitación enviada" (mock)
- Validación básica: email no vacío, nombre no vacío

#### 🧾 Recepcionista (US-10, US-11, US-12)
- Lista de pedidos activos desde `lib/mock/pedidos.ts`
- Cada pedido muestra: número de mesa, ítems, estado actual, timestamp
- Botón de cambio de estado: Recibido → Preparando → Listo (un clic avanza al siguiente)
- Pedidos con solicitud de cuenta muestran badge de alerta ⚠️ destacado en rojo
- Al marcar "Listo" un pedido que pidió la cuenta, se puede marcar como "Cerrado" (se elimina de la lista)

---

## Carta del cliente (`/mesa/[token]`)

### Estados internos

```
carta → carrito → seguimiento
         ↑ (volver a agregar desde seguimiento)
```

Un único `useReducer` en `page.tsx` maneja:
- `items[]` — carrito actual
- `vista` — `'carta' | 'carrito' | 'seguimiento'`
- `estadoPedido` — `'recibido' | 'preparando' | 'listo'`
- `cuentaSolicitada` — boolean

### Vista: `carta`

- Header: nombre del negocio + número de mesa
- Navegación horizontal de categorías (chips/tabs)
- Lista de ítems de la categoría activa: nombre, descripción, precio, botón `+`
- Al agregar un ítem: aparece la barra flotante en la parte inferior
- **Barra flotante**: `🛒 N ítems · $TOTAL · Ver carrito →` (fondo azul)
- Tocar la barra abre la vista `carrito`

### Vista: `carrito`

- Header con botón `←` para volver a la carta
- Lista de ítems: nombre, precio, cantidad (− / número / +), campo de nota opcional por ítem
- Total general
- Botón "Enviar pedido" (verde) → cambia vista a `seguimiento` con `estadoPedido: 'recibido'`

### Vista: `seguimiento`

- Muestra el estado del pedido con stepper visual: Recibido → Preparando → Listo
- El estado avanza automáticamente con `setTimeout`: 4 s después de confirmar pasa a "Preparando", otros 8 s después pasa a "Listo". Simula el tiempo real sin backend.
- Demora estimada mockeada según estado (ej. "~15 min" en Preparando, "¡Listo!" en Listo)
- Botón "Agregar más ítems" → vuelve a `carta` sin borrar el carrito
- Botón "Pedir la cuenta" → muestra mensaje de confirmación "✓ El mozo fue notificado" y activa `cuentaSolicitada: true` (deshabilita el botón)

---

## Estructura de archivos

```
app/
  dashboard/
    page.tsx                    ← reemplazar completamente
  mesa/
    [token]/
      page.tsx                  ← nuevo

components/
  dashboard/
    Sidebar.tsx                 ← sidebar fijo con 4 secciones
    CartaSection.tsx            ← US-08
    MesasSection.tsx            ← US-07
    EquipoSection.tsx           ← US-09
    RecepcionistaSection.tsx    ← US-10, 11, 12
  menu/
    CategoriaNav.tsx            ← chips de categorías
    ItemCard.tsx                ← ítem con botón +
    CartDrawer.tsx              ← barra flotante + panel carrito
    SeguimientoView.tsx         ← estado del pedido + pedir cuenta

lib/
  mock/
    menu.ts                     ← categorías e ítems
    mesas.ts                    ← mesas con tokens QR
    pedidos.ts                  ← pedidos activos (recepcionista)
    equipo.ts                   ← usuarios del equipo
```

---

## Mock data

### `lib/mock/menu.ts`
Array de categorías, cada una con array de ítems. Mínimo 3 categorías, 3–5 ítems por categoría. Campos por ítem: `id`, `nombre`, `descripcion`, `precio` (número), `disponible` (boolean).

### `lib/mock/mesas.ts`
Array de mesas. Campos: `id`, `numero` (1–8), `token` (string único, eg. `"mesa-token-1"`), `estado` (`'disponible' | 'ocupada'`).

### `lib/mock/pedidos.ts`
Array de pedidos activos. Campos: `id`, `mesa` (número), `items[]`, `estado` (`'recibido' | 'preparando' | 'listo'`), `cuentaSolicitada` (boolean), `timestamp`.

### `lib/mock/equipo.ts`
Array de usuarios. Campos: `id`, `nombre`, `email`, `rol` (`'admin' | 'encargado' | 'mozo'`), `estado` (`'activo' | 'invitado'`).

---

## Token del cliente

La página `/mesa/[token]` lee el param `token` de la URL pero en Fase 1 lo usa solo para mostrar el número de mesa (buscándolo en `lib/mock/mesas.ts`). Si el token no existe en el mock, muestra "Mesa no encontrada". El menú que se muestra es siempre el mismo mock data — no hay filtro por negocio en esta fase.

---

## QR

Se usa la librería `qrcode` (ya disponible en npm) para generar el QR en el browser como canvas o SVG. La URL que codifica el QR es `/mesa/[token]`. El botón "Descargar" convierte el canvas a PNG y dispara la descarga via `<a download>`.

---

## Convenciones

- Dark theme en el admin (consistente con lo ya construido en Sprint 0)
- Light theme en `/mesa/[token]`: fondo `#f8fafc`, texto `#0f172a`, acento `#2563eb`
- Tailwind CSS v4 para todos los estilos
- TypeScript estricto, sin `any`
- Sin llamadas fetch — todo desde los archivos de mock
- Sin librerías de formularios — HTML nativo + useState
