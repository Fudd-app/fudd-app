---
name: "📋 User Story (Historia de Usuario)"
about: "Crear una Historia de Usuario refinada para un solo repo ([web] o [api])"
title: "[web|api] Título descriptivo de la funcionalidad"
labels: ["enhancement"]
assignees: []
---

<!--
Flujo completo: docs/proceso/flujo-de-trabajo.md
- Título: [web] o [api] según el repo que toca. Si toca ambos, crear dos issues.
- Sin número en el título: el issue se identifica por el número que asigna GitHub (#N).
- Al crearlo: agregarlo al proyecto Fudd APP y completar Status, Estimación, Sprint, Tipo y Rama.
-->

> **Repo**: `[web] repos/web` | `[api] repos/api`  
> **Sprint**: Sprint XX (DD/MM – DD/MM/2026)  
> **Rama de trabajo**: `feat/<nro-issue>-descripcion-corta`  
> **Depende de**: #N | Ninguna  

---

## 1. Descripción de la Historia

**Como** [admin de negocio | encargado | mozo | comensal],  
**quiero** [acción o funcionalidad],  
**para** [beneficio o valor de negocio].

---

## 2. Criterios de Aceptación

- [ ] Criterio verificable 1
- [ ] Criterio verificable 2

---

## 3. Alcance

- **Incluye**: ...
- **Fuera de alcance**: ...

---

## 4. Impacto y riesgo de regresión

Funcionalidades existentes que toca o podría romper (pantallas, endpoints, tablas, eventos SSE, cuotas Free/Pro, contrato con el otro repo):

- ...

---

## 5. Checklist Técnico

- [ ] Tarea técnica 1
- [ ] Tests nuevos o actualizados (`[api]`) / flujo probado en navegador (`[web]`)
- [ ] Suite completa en verde: `cd repos/api && go test ./...` | `cd repos/web && npm run build`
- [ ] Regresión verificada sobre lo listado en la sección 4

---

## 6. Archivos Clave Involucrados

- `repos/...`
