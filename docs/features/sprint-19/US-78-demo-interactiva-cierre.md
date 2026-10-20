# US-78: Demo Interactiva en Vivo y Cierre de la Materia

> **Sprint**: Sprint 19 (02/11 – 08/11/2026)  
> **Épica**: QA E2E, Load Testing, Polish Final & Presentación  
> **Tipo**: `Todos`  
> **Estado**: ✅ **Resuelta**  
> **Asignado a**: Equipo Mesa CLICK (Martín Oviedo, Mateo Silvestrin, Pablo Aguirre)  
> **Rama de trabajo**: `feat/US-76-load-testing`  

---

## 1. Descripción de la Historia

**Como** equipo, preparamos la demo interactiva en vivo, datos de prueba limpios y la documentación final para la entrega de la materia.

---

## 2. Criterios de Aceptación (Definition of Done)

- [x] Base de datos de producción/demo poblada con datos realistas (negocio gastronómico completo, categorías, fotos, horarios y mesas con QR generados).
- [x] Guion de demo sincronizado entre los 3 integrantes mostrando: Cliente (QR y pedido) -> Cocina (KDS) -> Recepción/Mozo (Salón y cobro) -> Admin (Métricas, Suscripción y Reseñas Google).
- [x] Presentación de diapositivas (`docs/presentations/parte-2/index.html`) actualizada con estado final 100% completado.
- [x] Verificación de despliegue estable en Render.

---

## 3. Checklist de Tareas Técnicas

- [x] Script de seeding de datos de demo gastronómica completa (`repos/api/cmd/seed-demo/main.go`).
- [x] Ensayo general del flujo integral en vivo (`docs/testing/guion-demo-cierre.md`).
- [x] Chequeo final de enlaces, QR impresos para la mesa del jurado y backup (`demo-jurado-mesa-1` a `demo-jurado-mesa-12`).
- [x] Entrega final de repositorio y documentación.

---

## 4. Archivos Clave Involucrados

- `repos/api/cmd/seed-demo/main.go`
- `docs/testing/guion-demo-cierre.md`
- `docs/presentations/parte-2/index.html`
- `docs/features/README.md`

---

## 5. Notas, Dependencias y Flujo de Trabajo

- **Dependencias**: Culminación de todos los sprints de la Fase 4.
- **Estrategia Git**:
  1. Crear rama siempre a partir de `qa`:  
     ```bash
     git checkout qa && git pull
     git checkout -b feat/US-78-demo-cierre
     ```
  2. Implementar cambios siguiendo las reglas del proyecto ([AGENTS.md](../../AGENTS.md)).
  3. Validar builds antes de mergear:
     - Backend: `cd repos/api && go test ./...`
     - Frontend: `cd repos/web && npm run build`
  4. Abrir PR o mergear a `qa` y marcar este archivo como `✅ Resuelta`.
