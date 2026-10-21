# US-68: Esquema de Suscripciones y Middleware de Control de Cuotas

> **Sprint**: Sprint 16 (12/10 – 18/10/2026)  
> **Épica**: Modelo Freemium (Free vs Pro) & Control de Suscripciones  
> **Tipo**: `Backend`  
> **Estado**: ✅ **Resuelta**  
> **Asignado a**: Martín Oviedo + Antigravity (AI Agent)  
> **Rama de trabajo**: `feat/US-68-backend-planes-cuotas`  

---

## 1. Descripción de la Historia

**Como** sistema, modelo el esquema de suscripciones (Free/Pro) en BD y valido límites (10 mesas, 30 productos, 1 sucursal) en middleware antes de permitir creaciones.

---

## 2. Criterios de Aceptación (Definition of Done)

- [x] Estructura de datos con plan del tenant (`plan: 'free' | 'pro'`) y fechas de validez.
- [x] Todo tenant nuevo inicia con Plan Pro de prueba por 3 meses calendario y luego degrada a Free si no contrata.
- [x] Catálogo comercial del backend: Plan Pro a $30.000 ARS mensuales.
- [x] Límites plan Free: máx 10 mesas activas, máx 30 productos en carta, máx 1 sucursal.
- [x] Middleware en Go que intercepta creación de mesas, artículos y sucursales.
- [x] Retorno de error HTTP 403 Forbidden con código estructurado `PLAN_LIMIT_REACHED` y detalle de cuota.

---

## 3. Checklist de Tareas Técnicas

- [x] Migración de base de datos para plan en tabla `tenants` (`024_plan_freemium_tenants.sql`).
- [x] Migración de vigencia de prueba inicial (`033_add_trial_freemium_tenants.sql`).
- [x] Middleware de verificación de límites de cuota en Go (`internal/tenant/middleware.go`).
- [x] Pruebas unitarias de rechazo cuando se supera el límite en plan Free (`internal/tenant/middleware_test.go`, `plan_test.go`).
- [x] Endpoint `GET /tenants/me/plan` con estado de cuotas y `POST /tenants/me/upgrade` para registrar solicitudes.

---

## 4. Archivos Clave Involucrados

- `repos/api/internal/tenant/model.go`
- `repos/api/internal/tenant/middleware.go`
- `repos/api/internal/tenant/service.go`

---

## 5. Notas, Dependencias y Flujo de Trabajo

- **Dependencias**: Habilita US-69 y US-70.
- **Estrategia Git**:
  1. Crear rama siempre a partir de `qa`:  
     ```bash
     git checkout qa && git pull
     git checkout -b feat/US-68-backend-planes-cuotas
     ```
  2. Implementar cambios siguiendo las reglas del proyecto ([AGENTS.md](../../AGENTS.md)).
  3. Validar builds antes de mergear:
     - Backend: `cd repos/api && go test ./...`
     - Frontend: `cd repos/web && npm run build`
  4. Abrir PR o mergear a `qa` y marcar este archivo como `✅ Resuelta`.
