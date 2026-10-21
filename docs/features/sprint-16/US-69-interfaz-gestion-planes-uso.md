# US-69: Sección 'Planes y Suscripción' con Métricas de Uso de Cuota

> **Sprint**: Sprint 16 (12/10 – 18/10/2026)  
> **Épica**: Modelo Freemium (Free vs Pro) & Control de Suscripciones  
> **Tipo**: `Frontend`  
> **Estado**: ✅ **Resuelta**  
> **Asignado a**: Martín Oviedo + Antigravity (AI Agent)  
> **Rama de trabajo**: `feat/US-69-interfaz-planes`  

---

## 1. Descripción de la Historia

**Como** admin, veo una sección de 'Planes y Suscripción' con el uso actual de mis cuotas (mesas, productos, sucursales) y botones de upgrade a Pro.

---

## 2. Criterios de Aceptación (Definition of Done)

- [x] Pestaña o modal de suscripción en el dashboard (`PlanesSection.tsx` en `ConfiguracionSection.tsx`).
- [x] Barras de progreso visuales con uso actual (`Progress.tsx` con soporte monocromático y métricas de cuota).
- [x] Badge visible en la barra superior indicando si el comercio está en plan 'Free' o 'Pro' con acceso directo a planes.
- [x] Comparativo claro de características entre Free y Pro (canónico de `LandingPricing.tsx`).
- [x] Precio Pro visible y consistente: $30.000 ARS por mes.
- [x] Registro, onboarding y panel informan la prueba Pro inicial de 3 meses y su vencimiento.

---

## 3. Checklist de Tareas Técnicas

- [x] Crear componente `PlanesSection.tsx` e integrar pestaña `planes` en `ConfiguracionSection.tsx`.
- [x] Diseñar barras de progreso accesibles `Progress.tsx` y tabla comparativa de planes.
- [x] Conectar con endpoint `GET /tenants/me/plan` en `lib/api.ts`.
- [x] Botón interactivo de doble acción 'Solicitar Upgrade Pro' (`POST /tenants/me/upgrade` + apertura de cliente de correo/WhatsApp).

---

## 4. Archivos Clave Involucrados

- `repos/web/components/ui/Progress.tsx`
- `repos/web/components/dashboard/PlanesSection.tsx`
- `repos/web/components/dashboard/ConfiguracionSection.tsx`
- `repos/web/app/dashboard/page.tsx`
- `repos/web/lib/api.ts`

---

## 5. Notas, Dependencias y Flujo de Trabajo

- **Dependencias**: Depende de US-68.
- **Estrategia Git**:
  1. Crear rama siempre a partir de `qa`:  
     ```bash
     git checkout qa && git pull
     git checkout -b feat/US-69-interfaz-planes
     ```
  2. Implementar cambios siguiendo las reglas del proyecto ([AGENTS.md](../../AGENTS.md)).
  3. Validar builds antes de mergear:
     - Backend: `cd repos/api && go test ./...`
     - Frontend: `cd repos/web && npm run build`
  4. Abrir PR o mergear a `qa` y marcar este archivo como `✅ Resuelta`.
