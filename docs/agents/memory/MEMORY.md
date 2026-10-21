# Memory Index — Mesa CLICK

- [Mantener docs actualizados](feedback_mantener-docs-actualizados.md) — Soy el encargado de actualizar AGENTS.md, GEMINI.md, CLAUDE.md y READMEs ante cualquier cambio de fase, sprint o arquitectura
- [Probar antes de pushear](feedback_probar-antes-de-pushear.md) — Antes de hacer git push, siempre indicar los pasos para probar lo implementado y pedir confirmación explícita al usuario. NUNCA pushear sin confirmación.
- **EmailSender Abstraction:** Se implementó `auth.EmailSender` para usar `LogEmailSender` localmente y `ResendEmailSender` en producción, destrabando la configuración de magic link.
- **Sprint 4 Backend Completado:** Creados los paquetes y handlers para `internal/sucursal` (CRUD sucursales y sectores) e `internal/usuario` (CRUD equipo e invitaciones).
- **Sprint 5 & 6 Backend Completados (25/06/2026):**
  * Implementado el almacenamiento y flujo del servicio de pedidos en `internal/pedido`.
  * Diseñado un broker de Server-Sent Events (SSE) concurrente seguro en `internal/notificacion` y expuestos endpoints SSE (con y sin autenticación) en `rutas.go`.
  * Desarrollada una suite robusta de tests unitarios e integración en `internal/notificacion/broker_test.go` para validar suscripción, publicación, desuscripción y stream de red, asegurando la compilación exitosa (`go test ./...` pasa al 100%).
- **Fase 3: Integración y QA Completada (Sprints 7, 8 y 9) (30/07/2026):**
  * **Sprint 7:** Integración del flujo Admin (login magic link real, onboarding de negocio/sucursal y CRUD de carta/mesas conectado a PostgreSQL).
  * **Sprint 8:** Integración del flujo Cliente y Recepcionista en tiempo real (carta pública vía QR token, creación de pedidos reales y sincronización SSE en vivo).
  * **Sprint 9:** QA end-to-end de ambos happy paths, optimización responsive mobile, pipeline de CI/CD en GitHub Actions y deploy automático en Render (Docker).
- **Sprint 16 Freemium (Free vs Pro) Completado (02/10/2026):**
  * **US-68 (Backend cuotas):** Migración 024_plan_freemium_tenants.sql, middleware `tenant.RequerirCuota` fail-closed (401 si no hay claims, 500 en DB error, 403 con error estructurado `PLAN_LIMIT_REACHED`), endpoints `GET /tenants/me/plan` y `POST /tenants/me/upgrade`.
  * **US-69 (Planes & Suscripción):** Componente `PlanesSection.tsx` con barras de progreso monocromáticas `Progress.tsx`, comparativa canónica Free vs Pro, badge en navbar y CTA de upgrade a Pro.
  * **US-70 (Bloqueo elegante):** Interceptor de error 403 `PLAN_LIMIT_REACHED` en `lib/api.ts`, componente accesible `UpgradeModal.tsx`, y bloqueo preventivo con apertura automática de modal en `MesasSection.tsx` y `CartaSection.tsx`.
  * **US-71 (Personalización Pro & Marca de Agua):** Migración 025_personalizacion_pro_tenants.sql, fail-safe SQL en consulta pública de comensal, componente `MarcaAgua.tsx` ('Potenciado por Mesa CLICK'), variables de estilo `--mesa-accent` y `--mesa-font`, y controles en `AparienciaTab` con candados Pro.
- **Estrategia Git y Ambientes de Despliegue:**
  * Ramas principales: `main` (Producción) y `qa` (Testing).
  * Todo desarrollo de nueva feature o corrección de bug inicia a partir de `qa` (`feat/*`, `fix/*`).
  * Al validar en `qa`, se mergea hacia `main`.
  * `main` corre en **Render** (servicio Free, Docker, región Ohio) tanto para la API backend (`repos/api` → `mesa-click-api`) como para el frontend (`repos/web` → `mesa-click-web`).
