# US-67: Impresión Térmica de Comandas (58/80 mm) y Reintento Manual

> **Sprint**: Sprint 15 (05/10 – 11/10/2026)  
> **Épica**: Kitchen Display System (KDS) & Impresión de Comandas  
> **Tipo**: `Integración`  
> **Estado**: ✅ **Resuelta**  
> **Asignado a**: Pablo Aguirre + Antigravity (AI Agent)  
> **Rama de trabajo**: `feat/US-67-impresion-termica`  

---

## 1. Descripción de la Historia

**Como** cocina, imprimo la comanda en una impresora térmica (ticket 58/80 mm) al confirmarse el pedido, con opción de reintento manual si falla la impresión.

---

## 2. Criterios de Aceptación (Definition of Done)

- [x] Diseño de comanda térmica limpio, monoespaciado y de alto contraste en 58mm y 80mm.
- [x] Detalle de ticket: Mesa, fecha/hora, número de pedido, cantidad, ítem, modificadores destacados y notas.
- [x] Botón 'Imprimir ticket' en cada comanda del KDS y dashboard de recepción.
- [x] Disparo opcional de impresión automática al entrar un nuevo pedido.

---

## 3. Checklist de Tareas Técnicas

- [x] Crear vista/layout de impresión térmica con `@media print` en CSS.
- [x] Lógica de formateo compacto para 58mm y 80mm.
- [x] Botón de reimpresión manual accesible.
- [x] Documentar integración con impresoras térmicas USB/Red.

---

## 4. Archivos Clave Involucrados

- `repos/web/components/kds/TicketTermico.tsx`
- `repos/web/components/dashboard/RecepcionistaSection.tsx`

---

## 5. Notas, Dependencias y Flujo de Trabajo

- **Dependencias**: Complementaria a US-65.
- **Estrategia Git**:
  1. Crear rama siempre a partir de `qa`:  
     ```bash
     git checkout qa && git pull
     git checkout -b feat/US-67-impresion-termica
     ```
  2. Implementar cambios siguiendo las reglas del proyecto ([AGENTS.md](../../AGENTS.md)).
  3. Validar builds antes de mergear:
     - Backend: `cd repos/api && go test ./...`
     - Frontend: `cd repos/web && npm run build`
  4. Abrir PR o mergear a `qa` y marcar este archivo como `✅ Resuelta`.
