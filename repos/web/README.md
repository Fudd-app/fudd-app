# fudd — Frontend (Next.js)

Frontend web de fudd: landing, alta y configuración del negocio, dashboard del equipo (salón, carta, mesas, equipo, métricas, planes), pantalla de cocina (KDS) y la vista del comensal que se abre al escanear el QR de la mesa.

Para el contexto completo del proyecto, ver el [README raíz](../../README.md) y [AGENTS.md](AGENTS.md).

## Estado actual

Fase 4 — Evolución, Monetización & Analítica. El sprint en curso está en [`AGENTS.md` raíz](../../AGENTS.md) y los issues de este repo (prefijo `[web]`) en el tablero [Fudd APP](https://github.com/orgs/Fudd-app/projects/1).

## Levantar localmente

Lo recomendado es levantar todo el sistema con Docker desde la raíz (`npm run dev`, ver [`LOCAL_DEV.md`](../../LOCAL_DEV.md)). Para correr solo el frontend contra una API ya levantada:

```powershell
npm install
npm run dev
```

Disponible en [http://localhost:3000](http://localhost:3000).

## Stack

- **Framework**: Next.js 16 (App Router)
- **Runtime**: Node.js 24 (`engines.node`)
- **UI**: React 19, Tailwind CSS v4
- **Lenguaje**: TypeScript estricto

## Estructura de carpetas

```
app/
├── page.tsx              ← landing
├── (alta)/               ← registro y onboarding del negocio
├── login/                ← login admin
├── [slug]/login/         ← login del staff por negocio
├── auth/verify/          ← verificación de magic link
├── invitacion/[token]/   ← aceptación de invitación al equipo
├── dashboard/            ← panel de administración y operación
├── kds/                  ← pantalla de cocina
├── mesa/[token]/         ← vista del comensal (QR)
└── privacidad/ · terminos/

components/               ← auth, brand, comensal, dashboard, kds, landing, legal, menu, onboarding, registro, ui
lib/api.ts                ← cliente HTTP tipado, SSE e interceptor PLAN_LIMIT_REACHED
```

## Scripts disponibles

| Comando | Descripción |
|---|---|
| `npm run dev` | Servidor de desarrollo con hot reload |
| `npm run build` | Build de producción (validación obligatoria antes de cada PR) |
| `npm run lint` | Verificar errores de ESLint |

## Convenciones y despliegue

- **TypeScript**: no usar `any`
- **Componentes**: funcionales con hooks, sin class components
- **Estilos**: exclusivamente con Tailwind CSS
- **Mobile-first**: la vista del comensal está optimizada para celulares
- **Flujo de trabajo**: issues y PRs con prefijo `[web]`, ramas desde `qa` y code review antes del merge ([`docs/proceso/flujo-de-trabajo.md`](../../docs/proceso/flujo-de-trabajo.md))
- **Despliegue**: Vercel — `qa` → `qa.fudd.com.ar`, `main` → `fudd.com.ar` ([`docs/deploy/README.md`](../../docs/deploy/README.md))
