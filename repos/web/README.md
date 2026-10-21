# fudd — Frontend (Next.js)

Frontend web de fudd. Permite a los admins configurar su negocio y a los clientes hacer pedidos desde la mesa escaneando un QR.

Para el contexto completo del proyecto, ver el [README raíz](../../README.md) y [AGENTS.md](../../AGENTS.md).

## Estado actual

**Fase 2 completada** — Backend Go + PostgreSQL + SSE + Tests completo (US-19 a US-37).

**Fase 3 en curso — Sprint 7 (10/07 – 16/07/2026)**
La Fase 3 se enfoca en integrar el frontend Next.js con el backend real Go en `repos/api/` (reemplazando mocks por llamadas de API).

Ver las user stories del sprint en `../../docs/presentations/mesa-click-presentacion.html`.

## Levantar localmente

Requisitos: Node.js 20+.

```powershell
npm install
npm run dev
```

Disponible en [http://localhost:3000](http://localhost:3000).

## Stack

- **Framework**: Next.js App Router (v16+)
- **UI**: React 19+, Tailwind CSS v4+
- **Lenguaje**: TypeScript estricto

## Estructura de carpetas

```
app/
├── page.tsx              ← landing
├── login/page.tsx        ← login / registro
├── onboarding/page.tsx   ← onboarding admin (4 pasos)
└── dashboard/page.tsx    ← panel de administración

components/
├── landing/
└── onboarding/
```

## Scripts disponibles

| Comando | Descripción |
|---|---|
| `npm run dev` | Servidor de desarrollo con hot reload |
| `npm run build` | Build de producción |
| `npm run lint` | Verificar errores de ESLint |

## Convenciones y Despliegue

- **TypeScript**: No usar `any`
- **Componentes**: Funcionales con hooks — sin class components
- **Estilos**: Exclusivamente con Tailwind CSS
- **Mobile-first**: La vista del cliente es optimizada para celulares
- **Git Workflow y Despliegue**: Ramas (`feat/*`, `fix/*`) creadas siempre desde `qa`. Al validar en QA, mergear a `main` (desplegado en **Render** como `mesa-click-web` con Docker en región Ohio, servicio Free Web Service).
