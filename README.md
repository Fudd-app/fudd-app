# fudd (Mesa CLICK) — Menú Digital y Gestión de Pedidos

Plataforma web PWA para el sector gastronómico: los comensales escanean el QR de su mesa, ven la carta y piden desde el celular; el negocio gestiona pedidos, cocina (KDS), carta, equipo y métricas en tiempo real.

## Estado del proyecto

| | |
|---|---|
| **Fase actual** | Fase 4 — Evolución, Monetización & Analítica (2do cuatrimestre) |
| **Producción** | [fudd.com.ar](https://fudd.com.ar) · QA: [qa.fudd.com.ar](https://qa.fudd.com.ar) |
| **Backlog** | Tablero [Fudd APP](https://github.com/orgs/Fudd-app/projects/1) (fuente única de US y bugs) |

El detalle de sprints y reglas por fase está en [`AGENTS.md`](AGENTS.md). El roadmap de la Fase 4 se presenta en `docs/presentations/parte-2/index.html`.

## Estructura del monorepo

```
mesa-click/
├── repos/
│   ├── api/            ← backend Go (ver repos/api/README.md)
│   └── web/            ← frontend Next.js (ver repos/web/README.md)
├── docs/
│   ├── proceso/        ← flujo de trabajo: requerimiento → issue → PR → review
│   ├── deploy/         ← ambientes prod y qa
│   ├── flows/          ← happy paths admin y cliente
│   ├── product/        ← arquitectura del backend y propuestas
│   ├── presentations/  ← presentaciones y status reports (parte-1, parte-2)
│   ├── testing/        ← reportes de load testing, Lighthouse y demo
│   ├── legal/          ← compliance
│   └── diseño/         ← maqueta de referencia UX e identidad de marca
├── load-tests/         ← scripts k6
├── scripts/            ← scripts de desarrollo local (PowerShell)
├── docker-compose*.yml
├── LOCAL_DEV.md        ← cómo levantar todo localmente
└── AGENTS.md           ← reglas para el equipo y los agentes de IA
```

## Desarrollo local

Requisitos: Docker, Node.js 24+ y (opcional, para correr la API fuera de Docker) Go 1.25+.

```powershell
npm run setup   # primera vez: levanta PostgreSQL, API y web con Docker y siembra datos demo
npm run dev     # uso diario
```

Guía completa (seed, login de desarrollo, QR desde el celular, troubleshooting) en [`LOCAL_DEV.md`](LOCAL_DEV.md). Para la API, pedir el `.env` a **Pablo Aguirre** o partir de `repos/api/.env.example`.

Validación antes de cada PR:

```powershell
cd repos/api; go test ./...
cd repos/web; npm run build
```

## Stack tecnológico

| Capa | Tecnología |
|---|---|
| Frontend | Next.js 16 (App Router), React 19, Tailwind CSS v4, TypeScript |
| Backend | Go 1.25, `net/http` estándar, `slog`, `pgx/v5` |
| Base de datos | PostgreSQL (Neon en prod y qa) |
| Auth | Google Identity + magic link/contraseña heredados para admins; JWT en cookie HttpOnly |
| Tiempo real | Server-Sent Events (SSE) |
| Hosting | Vercel (web) · Render (api) — ver [`docs/deploy/README.md`](docs/deploy/README.md) |

## Flujo de trabajo

Todo requerimiento o bug se refina, se carga como issue `[web]` o `[api]` en el tablero, se estima y recién después se desarrolla en una rama creada desde `qa`, con PR y code review antes del merge. El proceso completo está en [`docs/proceso/flujo-de-trabajo.md`](docs/proceso/flujo-de-trabajo.md).

| Rama | Ambiente |
|---|---|
| `main` | Producción (`fudd.com.ar`) — nunca se commitea directo |
| `qa` | QA / Staging (`qa.fudd.com.ar`) — base de toda rama `feat/*` o `fix/*` |

## Equipo

| Alumno | Rol |
|---|---|
| Oviedo Martín | Desarrollo |
| Silvestrin Mateo | Desarrollo |
| Aguirre Pablo | Desarrollo |

Práctica Profesionalizante I — IRESM.

## Para agentes de IA

Leer [`AGENTS.md`](AGENTS.md) antes de cualquier tarea y seguir [`docs/proceso/flujo-de-trabajo.md`](docs/proceso/flujo-de-trabajo.md).
