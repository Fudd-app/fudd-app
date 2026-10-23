<!-- BEGIN:nextjs-agent-rules -->
# This is NOT the Next.js you know

This version has breaking changes — APIs, conventions, and file structure may all differ from your training data. Read the relevant guide in `node_modules/next/dist/docs/` before writing any code. Heed deprecation notices.
<!-- END:nextjs-agent-rules -->

---

# Contexto del proyecto — Mesa CLICK (Frontend)

> **Leer antes de empezar**: las reglas generales están en `../../AGENTS.md` y el flujo obligatorio de trabajo en `../../docs/proceso/flujo-de-trabajo.md`.
> Las User Stories y bugs viven en el tablero de GitHub [Fudd APP](https://github.com/orgs/Fudd-app/projects/1); los de este repo llevan el prefijo `[web]`.
> Los happy paths están en `../../docs/flows/`.

---

## Fase actual: Fase 4 — Evolución, Monetización & Analítica (2do Cuatrimestre)

El sprint en curso y su estado están en `../../AGENTS.md` → "Estado del proyecto".

- **Control de Suscripciones**: El backend valida cuotas del plan (Free vs Pro) devolviendo HTTP 403 `PLAN_LIMIT_REACHED`. El frontend intercepta este error en `lib/api.ts` y despliega preventivamente `UpgradeModal`.
- **Personalización Pro & Marca de Agua**: Los clientes en plan Pro pueden ocultar la marca de agua 'Potenciado por Mesa CLICK' y configurar paleta y fuentes extendidas en `AparienciaTab`. En plan Free, estos controles se bloquean elegantemente con candados hacia `UpgradeModal`.

---

## Estructura del proyecto (`repos/web/`)

```
repos/web/
├── app/                        ← rutas y páginas (Next.js App Router)
│   ├── page.tsx                (landing)
│   ├── (alta)/                 (registro y onboarding del negocio)
│   ├── login/ · [slug]/login/  (login admin · login del staff por negocio)
│   ├── auth/verify/            (verificación de magic link)
│   ├── invitacion/[token]/     (aceptación de invitación al equipo)
│   ├── dashboard/              (panel: salón, carta, mesas, equipo, métricas, configuración, planes)
│   ├── kds/                    (pantalla de cocina)
│   ├── mesa/[token]/           (comensal: carta → carrito → seguimiento → cuenta)
│   └── privacidad/ · terminos/
├── components/                 (auth, brand, comensal, dashboard, kds, landing, legal, menu, onboarding, registro, ui)
├── lib/
│   ├── api.ts                  (cliente HTTP tipado, SSE e interceptor PLAN_LIMIT_REACHED)
│   └── ...                     (comensal, planes, desgloseCuenta, legal, product-image)
└── public/
```

---

## Stack y convenciones

- **Framework**: Next.js App Router (no Pages Router)
- **Estilos**: Tailwind CSS v4+ — no usar CSS modules ni styled-components
- **Lenguaje**: TypeScript estricto — no usar `any`
- **Componentes**: funcionales con hooks, sin class components
- **Estado global**: ninguno (todo local o props)
- **Datos**: todo dato real se consume de la API vía `lib/api.ts`; no duplicar lógica del backend

## Reglas de código

- **Git workflow**: ramas (`feat/US-XX-*`, `fix/<nro-issue>-*`) siempre desde `qa`, PR con título `[web] ...` y code review antes del merge. `qa` se despliega en Vercel como `qa.fudd.com.ar` y `main` como `fudd.com.ar`.
- Un issue `[web]` no modifica `repos/api`; si hace falta un cambio de backend, se crea el issue `[api]` correspondiente.
- Validar con `npm run build` antes de abrir el PR
- No instalar librerías sin justificación explícita en la US
- Mantener diseño responsive — la vista del cliente es mobile-first (el cliente usa su celular)
- Antes de agregar una pantalla nueva, verificar que existe el issue `[web]` correspondiente en el sprint actual
