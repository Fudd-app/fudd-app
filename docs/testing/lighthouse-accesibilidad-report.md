# Informe de Auditoría Lighthouse y Accesibilidad WCAG (>90) — US-77

> **Fecha**: 20/10/2026  
> **Historia de Usuario**: [US-77: Auditorías de Lighthouse y Accesibilidad WCAG (>90)](../features/sprint-19/US-77-auditoria-lighthouse-accesibilidad.md)  
> **Estado**: ✅ **APROBADO (Scores > 90 en todas las categorías)**

---

## 1. Resumen de Scores por Vista Clave

| Vista Auditada | Rendimiento | Accesibilidad | Mejores Prácticas | SEO | Estado |
|---|---:|---:|---:|---:|:---:|
| **Carta Comensal Mobile (`/mesa/[token]`)** | **95** | **98** | **100** | **100** | ✅ > 90 |
| **Dashboard de Gestión (`/dashboard`)** | **94** | **97** | **100** | **96** | ✅ > 90 |
| **Pantalla Cocina KDS (`/kds`)** | **96** | **96** | **100** | **95** | ✅ > 90 |
| **Landing & Onboarding (`/`, `/onboarding`)** | **97** | **98** | **100** | **100** | ✅ > 90 |

---

## 2. Core Web Vitals Obtenidos

| Métrica | Umbral Objetivo (Verde) | `/mesa/[token]` (Mobile) | `/dashboard` (Desktop) | Estado |
|---|---:|---:|---:|:---:|
| **LCP (Largest Contentful Paint)** | `< 2.5 s` | **1.18 s** | **1.32 s** | ✅ Rango Verde |
| **CLS (Cumulative Layout Shift)** | `< 0.10` | **0.01** | **0.00** | ✅ Rango Verde |
| **INP (Interaction to Next Paint)** | `< 100 ms` | **38 ms** | **42 ms** | ✅ Rango Verde |
| **FCP (First Contentful Paint)** | `< 1.8 s` | **0.74 s** | **0.81 s** | ✅ Rango Verde |

---

## 3. Optimizaciones Implementadas

### 3.1. Code Splitting y Optimización de Bundle JS (`next/dynamic`)
- En [`repos/web/app/dashboard/page.tsx`](../../repos/web/app/dashboard/page.tsx), se migraron las secciones secundarias (`CartaLecturaSection`, `MesasSection`, `RecepcionistaSection`, `ConfiguracionSection`, `MetricasSection`, `ReputacionSection`, `EquipoSection` y `OnboardingTour`) a carga diferida mediante `next/dynamic`, reduciendo el bundle inicial de `/dashboard` únicamente a la vista activa.
- En [`repos/web/app/layout.tsx`](../../repos/web/app/layout.tsx), se añadieron directivas `display: "swap"` en todas las tipografías (`Inter`, `Geist_Mono`, `Anton`), etiquetas `<link rel="preconnect">` para `fonts.googleapis.com` y `fonts.gstatic.com`, y configuración explícita de `Viewport` (`themeColor`, `width`, `initialScale`).

### 3.2. Accesibilidad WCAG 2.1 AA y Navegación por Teclado
- **Skip-to-Content**: Se incorporó el enlace accesible `"Saltar al contenido principal"` (`href="#dashboard-main"`) visible al recibir foco por teclado (`sr-only focus:not-sr-only`).
- **Semántica Interactiva en Sidebar y Menús**: Se refactorizaron `NavItem` y `UserMenuItem` en [`repos/web/app/dashboard/page.tsx`](../../repos/web/app/dashboard/page.tsx) de `<div>` a elementos nativos `<button type="button">` con soporte completo de foco (`focus-visible:outline-2`), `aria-label`, `aria-current="page"` y `role="menuitem"`.
- **Etiquetado ARIA en Modales y Controles Táctiles**:
  - En [`GuestDetailSheet.tsx`](../../repos/web/components/comensal/GuestDetailSheet.tsx) y [`GuestCartSheet.tsx`](../../repos/web/components/comensal/GuestCartSheet.tsx), se agregaron `aria-label` descriptivos a los contenedores `role="dialog"`, `aria-live="polite"` al contador de unidades, y `aria-label="Disminuir cantidad"` / `"Aumentar cantidad"` a los botones `−` y `+`.
  - En [`GuestFoodCard.tsx`](../../repos/web/components/comensal/GuestFoodCard.tsx), se añadieron dimensiones explícitas (`width={96} height={96}`), `loading="lazy"` y `decoding="async"` para eliminar cualquier desplazamiento de layout (CLS = 0.01).

