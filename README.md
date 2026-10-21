# Mesa CLICK — Menú Digital y Gestión de Pedidos

Mesa CLICK es una plataforma web PWA para el sector gastronómico que permite digitalizar menús y gestionar pedidos en tiempo real mediante códigos QR en las mesas.

## Estado del proyecto

| | |
|---|---|
| **Fase actual** | Fase 3 — Integración y QA (Completada ✅) |
| **Sprint en curso** | Sprint 9 · 24/07 – 30/07/2026 (Completado ✅) |
| **Estado MVP** | 100% Finalizado y Desplegado en Producción |

Ver el roadmap completo en `docs/presentations/mesa-click-presentacion.html`.

## Estructura del monorepo

```
mesa-click/
├── docs/
│   ├── presentations/
│   │   ├── mesa-click-presentacion.html   ← roadmap, US, backlog y línea de tiempo
│   │   ├── status-report-01.html          ← status report #1 (Sprint 3 completado)
│   │   ├── status-report-02.html          ← status report #2 (Fase 2 Backend completada)
│   │   └── status-report-03.html          ← status report #3 (Fase 3 y Cierre del Proyecto)
│   ├── product/
│   │   └── arquitectura-back.md           ← entidades, servicios y auth (Fase 2)
│   ├── flows/
│   │   ├── happy-path-admin-negocio.md    ← flujo del admin de negocio
│   │   └── happy-path-cliente.md          ← flujo del cliente / comensal
│   └── tasks/
│       └── Modelo de Status Report.docx.pdf
│   └── agents/                            ← memoria, planes y specs generados con agentes de IA (ver docs/agents/README.md)
├── repos/
│   ├── api/                               ← backend Go
│   │   ├── internal/db/                   ← conexión y migraciones
│   │   ├── migrations/                    ← 12 archivos SQL (001 → 012)
│   │   ├── main.go                        ← punto de entrada del servidor
│   │   ├── rutas.go                       ← registro de endpoints
│   │   ├── go.mod / go.sum
│   │   └── .env.example                   ← plantilla de variables de entorno
│   └── web/                               ← frontend Next.js
│       ├── app/
│       │   ├── page.tsx                   ← landing
│       │   ├── login/page.tsx
│       │   ├── onboarding/page.tsx
│       │   ├── dashboard/page.tsx         ← panel admin (Carta, Mesas, Equipo, Recepcionista)
│       │   └── mesa/[token]/page.tsx      ← carta del cliente (escaneo de QR)
│       ├── components/
│       │   ├── dashboard/                 ← CartaSection, MesasSection, EquipoSection, RecepcionistaSection
│       │   └── menu/                      ← CategoriaNav, ItemCard, CartDrawer, SeguimientoView
│       └── lib/api.ts                     ← cliente HTTP tipado y conexión SSE en tiempo real
└── AGENTS.md                              ← reglas para todos los agentes de IA
```

## Plan de desarrollo — 3 fases

| Fase | Sprints | Período | Objetivo | Estado |
|---|---|---|---|---|
| **1 — Frontend** | 0–2 | 04/05 → 24/05 | Happy paths navegables sin backend | ✅ Completa |
| **2 — Backend** | 3–6 | 25/05 → 25/06 | API, DB, servicios, auth, tests | ✅ Completa |
| **3 — Integración y QA** | 7–9 | 26/06 → 30/07 | Conectar front con back, QA, deploy | ✅ Completa |

---

## Levantar la API (backend Go)

### Requisitos

- Go 1.22+
- PostgreSQL corriendo (local o Docker)

### 1. Configurar variables de entorno

> 💡 **Para el equipo de desarrollo (Mateo y Martín):** Pedir el archivo `.env` con las credenciales y configuración directamente a **Pablo Aguirre**, o crear uno propio copiando el `.env.example`:

```powershell
cd repos/api
cp .env.example .env
```

Editá `.env` con tus datos locales:

```env
DATABASE_URL=postgresql://postgres:tu_password@localhost:5432/mesa_click
PORT=8080
JWT_SECRET=cambiar_esto_por_algo_muy_secreto
APP_ENV=development
```

> **Importante:** no uses `?schema=public` al final de la URL — ese parámetro es de Prisma y pgx no lo reconoce.

### 2. Crear la base de datos (si no existe)

```sql
CREATE DATABASE mesa_click;
```

O con Docker (si no tenés Postgres instalado):

```powershell
docker run -d --name mesa-click-db \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres_password \
  -e POSTGRES_DB=mesa_click \
  -p 5432:5432 \
  postgres:16-alpine
```

### 3. Levantar el servidor

```powershell
cd repos/api
go run .
```

Al arrancar, el servidor:
1. Conecta a PostgreSQL
2. Ejecuta automáticamente las migraciones pendientes (la primera vez crea las 12 tablas)
3. Escucha en `http://localhost:8080`

Las migraciones son idempotentes: si ya corrieron, el servidor las saltea en los arranques siguientes.

### 4. Verificar que funciona

```powershell
curl http://localhost:8080/health
# → {"estado":"ok","timestamp":"..."}
```

### Estructura del paquete

```
repos/api/
├── main.go          ← carga .env, conecta DB, corre migraciones, levanta HTTP
├── rutas.go         ← registra los endpoints (GET /health + próximas rutas del Sprint 4)
├── internal/db/
│   ├── db.go        ← pool de conexiones pgx (Conectar / Cerrar)
│   └── migraciones.go ← ejecuta archivos .sql en orden, registra en _migraciones
└── migrations/
    ├── 001_create_tenants.sql
    ├── 002_create_usuarios.sql
    ├── 003_create_magic_tokens.sql
    ├── 004_create_sucursales.sql
    ├── 005_create_sectores.sql
    ├── 006_create_mesas.sql
    ├── 007_create_categorias.sql
    ├── 008_create_articulos.sql
    ├── 009_create_variantes.sql
    ├── 010_create_pedidos.sql
    ├── 011_create_pedido_items.sql
    └── 012_create_pedido_item_variantes.sql
```

---

## Levantar el frontend localmente

Requisitos: Node.js 20+.

```powershell
cd repos/web
npm install
npm run dev
```

Disponible en [http://localhost:3000](http://localhost:3000).

## Cómo probar el frontend (Fase 1 completa)

Todo funciona con datos mockeados — sin backend, sin base de datos. Solo necesitás tener el servidor levantado (`npm run dev`).

---

### Panel de administración — `http://localhost:3000/dashboard`

Navegá con la barra lateral izquierda. Hacé clic en el ícono de hamburguesa (☰) para expandir las etiquetas.

#### 📋 Carta

| Acción | Cómo probarlo |
|---|---|
| Ver categorías e ítems | Al entrar ya se ven las 3 categorías mock (Hamburguesas, Entradas, Bebidas) |
| Agregar categoría | Botón **"+ Nueva categoría"** → escribí el nombre → Enter o "Agregar" |
| Agregar ítem | Dentro de una categoría, botón **"+ Ítem"** → completar nombre y precio (obligatorios) |
| Ocultar/mostrar ítem | Link **"Ocultar"** o **"Mostrar"** al lado de cada ítem |
| Eliminar ítem | Botón **✕** — pide confirmación antes de borrar |
| Eliminar categoría | Botón **"Eliminar cat."** — pide confirmación |

#### 🪑 Mesas & QR

| Acción | Cómo probarlo |
|---|---|
| Ver QR de cada mesa | El QR se genera automáticamente al cargar la sección |
| Descargar QR | Botón **"↓ Descargar QR"** debajo de cada QR — descarga un PNG |
| Ver URL del QR | El QR apunta a `http://localhost:3000/mesa/mesa-token-N` |
| Probar el QR | Escanearlo con el celular (si estás en la misma red) o abrir la URL directamente |

#### 👥 Equipo

| Acción | Cómo probarlo |
|---|---|
| Ver equipo actual | Lista los 3 usuarios mock (Pablo, Martín, Mateo) |
| Invitar usuario | Completar nombre + email + rol (Encargado/Mozo) → "Enviar invitación" |
| Ver validación | Intentar enviar con campos vacíos o email sin @ → aparece mensaje de error |
| Confirmar éxito | Al invitar correctamente aparece "✓ Invitación enviada" por 3 segundos |

#### 🧾 Recepcionista

| Acción | Cómo probarlo |
|---|---|
| Ver pedidos activos | Se muestran 3 pedidos mock de las mesas 2, 4 y 7 |
| Ver alerta de cuenta | Mesa 7 aparece destacada en rojo con **⚠️ Pide la cuenta** |
| Avanzar estado | Botón **"→ Preparando"** o **"→ Listo"** — avanza el estado del pedido |
| Cerrar pedido | Cuando un pedido llega a "Listo", aparece **"Cerrar pedido"** que lo elimina de la lista |
| Estado vacío | Cerrá todos los pedidos → aparece "No hay pedidos activos en este momento" |

---

### Vista del cliente (comensal) — `http://localhost:3000/mesa/[token]`

Simulá la experiencia del cliente que escaneó el QR de su mesa:

```
http://localhost:3000/mesa/mesa-token-1   ← Mesa 1
http://localhost:3000/mesa/mesa-token-2   ← Mesa 2
...hasta mesa-token-8
```

| Paso | Acción | Resultado esperado |
|---|---|---|
| 1 | Abrir cualquier URL de mesa | Se ve la carta con chips de categorías arriba |
| 2 | Tocar **"+"** en varios ítems | Aparece barra azul en el fondo con el total |
| 3 | Tocar la barra azul | Se abre la vista del carrito |
| 4 | Ajustar cantidades con **−** y **+** | La cantidad baja a 0 → el ítem desaparece del carrito |
| 5 | Escribir una nota en un ítem | Campo de texto "Nota para la cocina" debajo de cada ítem |
| 6 | Tocar **"Enviar pedido"** | Cambia a la vista de seguimiento |
| 7 | Esperar 4 segundos | El estado avanza automáticamente de "Recibido" a "Preparando" |
| 8 | Esperar 8 segundos más | El estado avanza a "¡Listo para retirar!" |
| 9 | Tocar **"+ Agregar más ítems"** | Vuelve a la carta sin borrar el pedido anterior |
| 10 | Tocar **"Pedir la cuenta"** | Aparece "✓ El mozo fue notificado" y el botón se deshabilita |

> **Nota:** Los ítems marcados como "No disponible" en la sección Carta del admin no aparecen en la carta del cliente.

---

### Presentaciones del proyecto

```
docs/presentations/mesa-click-presentacion.html   ← roadmap, backlog y línea de tiempo
docs/presentations/status-report-01.html          ← status report #1 (Sprint 3 completado)
docs/presentations/status-report-02.html          ← status report #2 (Fase 2 Backend completada)
docs/presentations/status-report-03.html          ← status report #3 (Fase 3 y Cierre del Proyecto)
```

Abrí los archivos directamente en el browser. La presentación principal se navega con ← → o con la barra superior.

---

## Stack tecnológico

| Capa | Tecnología |
|---|---|
| Frontend | Next.js (v16+), React 19+, Tailwind CSS v4+, TypeScript |
| Backend | Go 1.22, net/http estándar, slog |
| Base de datos | PostgreSQL 16+ |
| Auth (Sprint 4) | Magic link por email |
| Tiempo real (Sprint 5) | Server-Sent Events (SSE) |

## Flujo de Trabajo y Entornos de Despliegue

### Ramas y Ambientes
- **`main` (Producción):**
  - **Backend (`repos/api`)**: Desplegado en **Render** (Free Web Service / Docker, región Ohio, servicio: `mesa-click-api`).
  - **Frontend (`repos/web`)**: Desplegado en **Render** (Free Web Service / Docker, región Ohio, servicio: `mesa-click-web`).
- **`qa` (QA / Staging):** Rama base de integración y pruebas de calidad previa a producción.

### Flujo de Desarrollo (Git Workflow)
> ⚠️ **Regla fundamental:** Nunca se trabaja ni se commitea directamente en `main`. Todo nuevo feature o corrección debe partir obligatoriamente de `qa`.

1. **Partir siempre desde `qa` actualizado:**
   ```powershell
   git checkout qa
   git pull origin qa
   ```
2. **Crear una rama descriptiva:**
   ```powershell
   git checkout -b feat/US-XX-descripcion   # Para nuevas funcionalidades
   # o bien:
   git checkout -b fix/descripcion-del-bug # Para correcciones
   ```
3. **Desarrollar y probar localmente.**
4. **Subir la rama y crear Pull Request hacia `qa`:**
   ```powershell
   git push origin feat/US-XX-descripcion
   ```
5. **Validación en QA:** Se revisa, testea e integra en la rama `qa`.
6. **Pase a Producción (`main`):** Cuando todo está aprobado y validado en `qa`, se mergea hacia `main`. Esto dispara de forma automática el despliegue a producción en Render (`mesa-click-api` y `mesa-click-web`) mediante Docker.

---

## Equipo

| Alumno | Rol |
|---|---|
| Oviedo Martín | Desarrollo |
| Silvestrin Mateo | Desarrollo |
| Aguirre Pablo | Desarrollo |

Práctica Profesionalizante I — IRESM.

## Para agentes de IA

Antes de empezar cualquier tarea, leer:
1. `AGENTS.md` — reglas, fases, sprint en curso y convenciones
2. `docs/presentations/mesa-click-presentacion.html` → slide "Backlog" — US del sprint actual
3. `docs/flows/` — happy paths de los flujos a implementar
