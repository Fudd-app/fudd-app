# Instrucciones de IA / Codex / GitHub Copilot — Mesa CLICK

Este repositorio utiliza **AGENTS.md** como la fuente principal de verdad y reglas para cualquier asistente de IA o desarrollador.

> Consultar siempre y seguir las pautas detalladas en [AGENTS.md](../AGENTS.md).
>
> Todo requerimiento, bug o PR sigue el [flujo de trabajo](../docs/proceso/flujo-de-trabajo.md): refinar → issue `[web]`/`[api]`/`[docs]` en el proyecto → estimar → rama desde `qa` → tests y regresión → PR → code review → merge. Aplica también a docs; la única excepción es un hotfix.

## Reglas clave indispensables:

1. **Git Workflow obligatorio**:
   - **NUNCA** commitear directo en `main`: solo recibe PRs de `release/*` y `hotfix/*`.
   - Toda nueva rama de feature (`feat/*`), corrección (`fix/*`) o docs (`docs/*`) **DEBE crearse a partir de `qa`** (`git checkout qa && git pull origin qa`).
   - Al finalizar el desarrollo y las pruebas, abrir Pull Request hacia `qa`.
   - El pase a producción es a demanda, con el [proceso de release](../docs/proceso/release.md): rama `release/vAAAA.MM.DD` desde `qa`, novedades en `repos/web/content/novedades.json`, PR a `main`, tag y GitHub Release.
   - Hotfix (bug en producción que bloquea la operación o expone datos): rama `hotfix/*` desde `main`, PR a `main` y de vuelta a `qa`.

2. **Entornos y Despliegue** (detalle en `docs/deploy/README.md`):
   - `main` → producción: web en Vercel (`fudd.com.ar`), API en Render (`fudd-app-api-prod`), DB en Neon.
   - `qa` → staging: web en Vercel (`qa.fudd.com.ar`), API en Render (`fidd-app-api-qa`), DB en Neon.

3. **Variables de entorno (`.env`)**:
   - Para la API backend, solicitar el archivo `.env` configurado directamente a **Pablo Aguirre**.

4. **Backend (Go)**:
   - Go 1.26+, PostgreSQL con driver `pgx/v5`, `net/http` estándar.
   - Usar `slog` para logs estructurados.
   - Mantener migraciones SQL idempotentes (no modificar archivos `.sql` existentes, agregar nuevos numerados).

5. **Frontend (Next.js)**:
   - Next.js App Router (v16+), React 19+, Tailwind CSS v4+, TypeScript estricto (no `any`).
   - Node.js 24 (fijado en `engines.node` de `repos/web/package.json`).
   - El cliente escanea mesas con código QR, por ende la vista de mesa es mobile-first.
