---
name: "🐛 Bug / Corrección (Fix)"
about: "Reportar un error en QA o Producción para seguimiento y corrección"
title: "FIX: [Descripción corta del problema]"
labels: ["bug"]
assignees: []
---

> **Ambiente afectado**: `QA (qa.fudd.com.ar)` | `Producción (fudd.com.ar)`  
> **Capa**: `Frontend` | `Backend` | `Base de Datos`  
> **Severidad**: `Alta (Bloqueante)` | `Media` | `Baja (Visual/UI)`  
> **Rama de trabajo**: `fix/descripcion-corta`  

---

## 1. Descripción del Problema

Descripción clara de qué está fallando y en qué pantalla o endpoint ocurre.

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

## 4. Criterios de Cierre

- [ ] Bug corregido en rama `fix/...` (creada desde `qa`).
- [ ] Tests / Build pasando (`go test ./...` / `npm run build`).
- [ ] Validado en `qa.fudd.com.ar`.
