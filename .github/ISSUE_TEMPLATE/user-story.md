---
name: "📋 User Story (Historia de Usuario)"
about: "Crear una nueva Historia de Usuario siguiendo el estándar de Fudd / Mesa CLICK"
title: "US-XX: [Título descriptivo de la funcionalidad]"
labels: ["enhancement"]
assignees: []
---

> **Sprint**: Sprint XX (DD/MM – DD/MM/2026)  
> **Épica**: [Nombre de la Épica / Objetivo del Sprint]  
> **Tipo**: `Frontend` | `Backend` | `Integración` | `Fullstack`  
> **Estado**: 📋 **Pendiente**  
> **Rama de trabajo**: `feat/US-XX-descripcion-corta`  

---

## 1. Descripción de la Historia

**Como** [rol / actor: admin de negocio, encargado, mozo, comensal],  
**quiero** [acción o funcionalidad],  
**para** [beneficio o valor de negocio].

---

## 2. Criterios de Aceptación (Definition of Done)

- [ ] Criterio de aceptación 1
- [ ] Criterio de aceptación 2
- [ ] Criterio de aceptación 3

---

## 3. Checklist de Tareas Técnicas

- [ ] Tarea técnica 1
- [ ] Tarea técnica 2
- [ ] Validar compilación y tests (`go test ./...` en API / `npm run build` en Web)

---

## 4. Archivos Clave Involucrados

- `repos/api/...`
- `repos/web/...`

---

## 5. Notas, Dependencias y Flujo de Trabajo

- **Dependencias**: Ninguna
- **Estrategia Git**:
  1. Crear rama siempre a partir de `qa`:  
     ```bash
     git checkout qa && git pull
     git checkout -b feat/US-XX-descripcion-corta
     ```
  2. Implementar cambios siguiendo las reglas del proyecto (`AGENTS.md`).
  3. Validar builds antes de mergear:
     - Backend: `cd repos/api && go test ./...`
     - Frontend: `cd repos/web && npm run build`
  4. Abrir PR hacia `qa`, validar en `qa.fudd.com.ar` y luego promover a `main` (`fudd.com.ar`).
