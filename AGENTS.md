# Mesa CLICK — Agent Rules

Estas reglas aplican a cualquier agente de IA (Claude, Gemini, CLI, etc.) que trabaje en este repositorio. **Este es el documento principal de referencia para el contexto del proyecto.**

---

## ANTES DE EMPEZAR

1. **Seguir el flujo de trabajo obligatorio: [`docs/proceso/flujo-de-trabajo.md`](docs/proceso/flujo-de-trabajo.md)**.
   Leelo completo y aplicalo paso a paso **cada vez** que:
   - llegue un requerimiento, una idea o un bug (aunque sea por chat): primero se refina, se crea el issue `[web]`/`[api]`, se carga en el proyecto y se estima. **No se escribe código antes de eso**;
   - se tome un issue para desarrollar (rama desde `qa`, tests, regresión, commit, PR);
   - se abra, revise o mergee un PR (code review obligatorio antes del merge).

2. **Leer el issue a desarrollar y los del sprint en curso** en el tablero de GitHub [Fudd APP](https://github.com/orgs/Fudd-app/projects/1) (`gh project item-list 1 --owner Fudd-app`). Es la única fuente de verdad del backlog: las US y bugs no se versionan en `docs/`.

3. **Leer el happy path correspondiente** a la tarea:
   - `docs/flows/happy-path-admin-negocio.md` — flujo del admin de negocio
   - `docs/flows/happy-path-cliente.md` — flujo del cliente / comensal

4. **Consultar el Status Report más reciente** para ver bloqueantes o decisiones pendientes:
   - `docs/presentations/parte-2/status-report-20261019.html`

> No implementar nada que no esté cubierto por un issue refinado y estimado del sprint en curso. Si hay dudas, preguntar antes de avanzar.

---

## Resumen del proyecto

**Mesa CLICK** es una plataforma web PWA para digitalizar pedidos en locales gastronómicos (bares, cafeterías, restaurantes). Los clientes escanean un QR por mesa, ven la carta digital y hacen el pedido desde el celular. El negocio gestiona todo desde un dashboard en tiempo real.

### Actores principales
| Actor | Acceso | Descripción |
|---|---|---|
| Admin de negocio | Login (magic link) | Configura sucursales, mesas, carta y usuarios |
| Encargado | Login (magic link) | Gestiona su sucursal |
| Recepcionista / Mozo | Login (magic link) | Dashboard de pedidos en vivo |
| Cliente / Comensal | Sin login (QR) | Carta, pedido, seguimiento, cuenta |

---

## Estado del proyecto

| | |
|---|---|
| **Fase actual** | Fase 4 — Evolución, Monetización & Analítica (2do Cuatrimestre) |
| **Sprint en curso** | **Sprint 18 (Business Analytics, Reputación & Exportación)** — pendientes US-74 y US-75. Sprints semanales (S10 a S19, 31/08 → 08/11/2026); 16, 17 y 19 completados por adelantado. El estado al día de cada ítem está en el [tablero](https://github.com/orgs/Fudd-app/projects/1) |
| **Objetivo Fase 4** | Llevar Mesa CLICK a nivel comercial: SaaS Freemium (Free vs Pro), Mobile-First comensal, carga masiva CSV/Excel, KDS de cocina, disponibilidad/franjas horarias, métricas y analítica de negocio |

### Sprints detallados
| Sprint | Objetivo | Estado |
|---|---|---|
| 0 | Maqueta base: landing, login, onboarding, dashboard | ✓ Completado |
| 1 | Happy path Admin completo + dashboard recepcionista (mock) | ✓ Completado |
| 2 | Happy path Cliente completo (carta, pedido, seguimiento, cuenta) | ✓ Completado |
| 3 | Stack + DB: entidades, relaciones, migraciones | ✓ Completado |
| 4 | Servicios core (tenant, sucursal, mesa, carta) + auth magic link | ✓ Completado |
| 5 | Servicio de pedidos + tiempo real (SSE) | ✓ Completado |
| 6 | Tests unitarios e integración | ✓ Completado |
| 7 | Integración flujo admin y autenticación real | ✓ Completado |
| 8 | Integración flujo cliente y recepcionista en tiempo real | ✓ Completado |
| 9 | QA end-to-end, polish responsive y deploy a producción | ✓ Completado |
| 10 | Configuración avanzada de cuenta (perfil, fiscal, horarios/turnos) · 31/08–06/09 | ✓ Completado |
| 11 | Rediseño UI/UX Base & Onboarding guiado (tour interactivo) · 07/09–13/09 | ✓ Completado |
| 12 | Mobile-First Comensal (sticky, bottom-sheet, personalización, pedidos colaborativos) · 14/09–20/09 | ✓ Completado |
| 13 | Carga masiva CSV/Excel & ajuste porcentual de precios · 21/09–27/09 | ✓ Completado |
| 14 | Disponibilidad de ítems (86) & menús por franja horaria · 28/09–04/10 | ✓ Completado |
| 15 | KDS, impresión de comandas térmicas & acceso del equipo (credenciales, login por negocio, Google) · 05/10–11/10 | ✓ Completado |
| 16 | Modelo Freemium (Free vs Pro) & control de suscripciones · 12/10–18/10 | ✓ Completado (adelantado) |
| 17 | Dashboard con métricas (KPIs, agregaciones SQL) · 19/10–25/10 | ✓ Completado (adelantado) |
| 18 | Business Analytics, Reputación (Reseñas & Google Funnel) & exportación · 26/10–01/11 | ⚡ En Curso (US-74, US-75) |
| 19 | QA E2E, Load Testing, Polish final & Demo de cierre · 02/11–08/11 | ✓ Completado (adelantado) |

---

## Stack tecnológico

| Capa | Tecnología |
|---|---|
| Frontend | Next.js (v16+), React 19+, Tailwind CSS v4+, TypeScript |
| Backend | Go 1.25, PostgreSQL (`pgx/v5`), net/http estándar |
| Logging | `slog` (estructurado) |
| Auth | Google Identity (dueño y staff) + magic link/contraseña heredados para admins; JWT en cookie HttpOnly |
| Tiempo real | Server-Sent Events (SSE) |

---

## Estructura del monorepo

```
mesa-click/
├── repos/
│   ├── api/          ← Backend Go
│   └── web/          ← Frontend Next.js
├── docs/
│   ├── proceso/flujo-de-trabajo.md        ← flujo obligatorio: requerimiento → issue → PR → review
│   ├── deploy/README.md                   ← ambientes prod y qa (Vercel, Render, Neon, DNS)
│   ├── presentations/
│   │   ├── parte-1/                       ← presentación y status reports 01–03 (1er cuatrimestre)
│   │   └── parte-2/                       ← index.html (roadmap Fase 4) y status reports 04 → actual
│   ├── product/
│   │   ├── arquitectura-back.md           ← diseño técnico del backend
│   │   └── propuesta-reseñas-local.html   ← propuesta base de US-79 / US-80
│   ├── flows/                             ← happy paths admin y cliente, mapa de pantallas
│   ├── testing/                           ← reportes de load testing, Lighthouse y guion de demo
│   ├── legal/                             ← checklist de compliance
│   └── diseño/
│       ├── maqueta/                       ← maqueta de referencia UX (ver "Guía de diseño")
│       └── fudd/                          ← identidad de marca fudd
├── load-tests/                            ← scripts k6
├── scripts/                               ← scripts PowerShell de desarrollo local (ver LOCAL_DEV.md)
└── AGENTS.md    ← este archivo (fuente de verdad principal)
```

---

## Guía de diseño — Maqueta de referencia

En `docs/diseño/maqueta/` hay una maqueta (Vite + React + Tailwind, exportada de Figma Make) con un diseño nuevo propuesto por el grupo. **Es una referencia de experiencia, no un diseño a copiar tal cual.**

| Pantalla de la maqueta | Componente en `src/App.tsx` | Equivalente en `repos/web` |
|---|---|---|
| Landing | `LandingApp` | `app/page.tsx` |
| Comensal (carta, carrito, pedido enviado) | `CustomerApp` | `app/mesa/[token]` |
| Salón en vivo (mesas por estado + drawer) | `DashboardApp` | `components/dashboard/RecepcionistaSection.tsx` / `MesasSection.tsx` |
| Gestión de carta | `CatalogModule` | `components/dashboard/CartaSection.tsx` |
| Configuración (negocio, sucursales, pagos, equipo) | `SettingsModule` | `components/dashboard/ConfiguracionSection.tsx` / `EquipoSection.tsx` |
| Cocina (KDS) | `KitchenApp` | `app/kds` |

**Qué tomar:**
- **La experiencia**: flujos, jerarquía de información, orden de las secciones, textos de ayuda y microcopy, feedback (toasts, estados vacíos, confirmaciones), drawers/modales y la navegación.
- **Patrones de componentes**: estructura de botones (`primary-button`, `secondary-button`, `icon-button`), tarjetas (`metric-card`, `table-card`, `settings-card`), `status-pill`, filtros segmentados, encabezados de página (`eyebrow` + título + descripción) y layouts de drawer. Los estilos están en `src/index.css`.

**Qué NO tomar:**
- **Colores, tipografías y tokens** de la maqueta. Se mantiene la identidad visual actual de `repos/web`.
- **Datos y lógica**: la maqueta usa datos mock y estado local. Todo dato real se consume de la API existente (ver "No duplicar lógica").
- **Código copiado literal**: adaptar cada patrón a los componentes y a Tailwind de `repos/web`, sin agregar dependencias de la maqueta.

Los cambios de UI inspirados en la maqueta siguen la regla general: solo dentro del alcance de una US del sprint en curso.

---

## Reglas por fase
 
### Fase 4 — Evolución, Monetización & Analítica (Actual - 2do Cuatrimestre)
- **Control de Suscripciones (Freemium)**: El backend valida cuotas del plan (Free: 10 mesas activas, 30 productos, 1 sucursal; Pro: ilimitado) en middleware `tenant.RequerirCuota`. Al alcanzar el límite, responde HTTP 403 Forbidden con formato estructurado `{"error": "...", "codigo": "PLAN_LIMIT_REACHED", "detalle": {"recurso": "...", "limite": N, "actual": N}}`. El cliente frontend (`lib/api.ts`) detecta `PLAN_LIMIT_REACHED` y abre preventivamente `UpgradeModal` sin mostrar carteles de error genéricos.
- **Condiciones comerciales Freemium**: Todo negocio nuevo comienza con una prueba completa del Plan Pro por 3 meses calendario. Al vencer, pasa al Plan Free salvo contratación. El Plan Pro cuesta $30.000 ARS por mes.
- **Carga Masiva**: Las operaciones de importación de menús por CSV/Excel deben ser atómicas y devolver reportes claros de filas procesadas con error. Exclusivo de Plan Pro.
- **Métricas**: Consultas de agregación SQL optimizadas con índices adecuados para no penalizar la performance del servidor.
- **KDS (cocina)**: La pantalla de cocina es una vista independiente del dashboard de mozo. Los estados por ítem se propagan por SSE; la impresión térmica (ESC/POS 58/80 mm) debe tolerar fallos con reintento manual.
- **Disponibilidad & Franjas horarias**: La carta visible del comensal se resuelve combinando disponibilidad del ítem (86), franja horaria de la sucursal y estado de apertura en una sola consulta.
- **Reseñas & Reputación (Smart Google Funnel)**: Calificación 1 a 5★ + comentario post-consumo (estado "Listo"). Comensales satisfechos (4-5★) reciben CTA directo para recomendar en Google Reviews vía Place ID; valoraciones bajas (1-3★) se retienen internamente como feedback privado y alertan en tiempo real al encargado.
- **Sprints semanales**: Cada sprint dura 1 semana y culmina con un release desplegado (ver [ambientes](docs/deploy/README.md)).
- Mantener cobertura de tests en Go y validar compilación con `go test ./...` y `npm run build` en web antes de cada merge.

---

## Estrategia de Ramas y Despliegue (Git Workflow)

### Ramas Principales y Entornos
| Rama | Ambiente | Frontend (`repos/web`) | Backend (`repos/api`) | Base de datos |
|---|---|---|---|---|
| `main` | Producción | Vercel → `fudd.com.ar` | Render → `fudd-app-api-prod` | Neon (prod) |
| `qa` | QA / Staging | Vercel → `qa.fudd.com.ar` | Render → `fidd-app-api-qa` | Neon (qa) |

Ambos ambientes se despliegan automáticamente al pushear a su rama. Detalle de cuentas, dominios y servicios en [`docs/deploy/README.md`](docs/deploy/README.md).

### Flujo de Desarrollo
1. **Creación de ramas**: Todo nuevo desarrollo (feature o corrección de bug) **debe crearse a partir de `qa`**:
   - `git checkout qa && git pull`
   - `git checkout -b feat/US-XX-descripcion` (features)
   - `git checkout -b fix/<nro-issue>-descripcion` (correcciones de bugs)
2. **Integración en QA**: Al finalizar, abrir PR hacia **`qa`**; se mergea solo después del code review (ver [flujo de trabajo](docs/proceso/flujo-de-trabajo.md)).
3. **Pase a Producción**: Una vez testeado y validado en `qa`, se mergea hacia **`main`**, lo que despliega automáticamente producción (Vercel + Render).

---

## Convenciones generales
- **Issues y PRs**: el título lleva el repo que toca: `[web]` o `[api]` (ej. `[api] US-88: ...`, `[web] BUG: ...`). Un ítem toca un solo repo; si necesita ambos, se divide.
- **Branches**: `feat/US-XX-descripcion` o `fix/<nro-issue>-descripcion` (creadas **siempre y sin excepción** a partir de `qa`). **Nunca commitear directo en `main`**.
- **Commits**: Conventional Commits con scope del repo (`feat(web): ...`, `fix(api): ...`). Explicar el "por qué" en lugar del "qué".
- **Variables de entorno (`.env`)**: Solicitar credenciales y `.env` para la API directamente a **Pablo Aguirre**.
- **No duplicar lógica**: Si algo ya está en el backend, el frontend solo lo consume.
- **Nunca pushear sin confirmación**: antes de `git push`, indicar los pasos para probar lo implementado y esperar la confirmación explícita del usuario.
- **Mantener la documentación al día**: al terminar una tarea que cambie fase, sprint, stack, estructura de carpetas, ambientes o arquitectura, actualizar sin que lo pidan `AGENTS.md`, `repos/web/AGENTS.md`, `.github/copilot-instructions.md`, los README (raíz, `repos/api`, `repos/web`) y los docs afectados.
