---
name: "🐛 Bug / Corrección (Fix)"
about: "Reportar un error en QA o Producción para un solo repo ([web] o [api])"
title: "[web|api] BUG: Descripción corta del problema"
labels: ["bug"]
assignees: []
---

<!--
Flujo completo: docs/proceso/flujo-de-trabajo.md
- Título: [web] o [api] según dónde está la falla. Si requiere cambios en ambos, crear dos issues.
- Al crearlo: agregarlo al proyecto Fudd APP y completar Status, Estimación, Sprint, Tipo (Bug / Fix) y Rama.
-->

> **Repo**: `[web] repos/web` | `[api] repos/api`  
> **Ambiente afectado**: `QA (qa.fudd.com.ar)` | `Producción (fudd.com.ar)`  
> **Severidad**: `Alta (Bloqueante)` | `Media` | `Baja (Visual/UI)`  
> **Rama de trabajo**: `fix/<nro-issue>-descripcion-corta`  

---

## 1. Descripción del Problema

Qué está fallando y en qué pantalla o endpoint ocurre.

---

## 2. Pasos para Reproducir

1. Ir a `...`
2. Hacer clic en `...`
3. Ver error `...`

---

## 3. Comportamiento Esperado vs. Actual

- **Esperado**: ...
- **Actual**: ...

---

## 4. Causa e Impacto

- **Causa (al refinar)**: ...
- **Funcionalidades que podría afectar la corrección**: ...

---

## 5. Criterios de Cierre

- [ ] Test que reproduce el bug (`[api]`) o pasos de reproducción verificados en navegador (`[web]`).
- [ ] Corrección en rama `fix/...` creada desde `qa`.
- [ ] Suite completa en verde: `cd repos/api && go test ./...` | `cd repos/web && npm run build`.
- [ ] Regresión verificada sobre lo listado en la sección 4.
- [ ] Validado en `qa.fudd.com.ar`.
