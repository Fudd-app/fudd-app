# Instrucciones de IA / Codex / GitHub Copilot — Mesa CLICK

Este repositorio utiliza **AGENTS.md** como la fuente principal de verdad y reglas para cualquier asistente de IA o desarrollador.

> Consultar siempre y seguir las pautas detalladas en [AGENTS.md](../AGENTS.md).
>
> Todo requerimiento, bug o PR sigue el [flujo de trabajo](../docs/proceso/flujo-de-trabajo.md): refinar → issue `[web]`/`[api]` en el proyecto → estimar → rama desde `qa` → tests y regresión → PR → code review → merge.

## Reglas clave indispensables:

1. **Git Workflow obligatorio**:
   - **NUNCA** commitear directo en `main`.
   - Toda nueva rama de feature (`feat/*`) o corrección (`fix/*`) **DEBE crearse a partir de `qa`** (`git checkout qa && git pull origin qa`).
   - Al finalizar el desarrollo y las pruebas, abrir Pull Request hacia `qa`.
   - Una vez testeado y validado en `qa`, se mergea hacia `main`.

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
