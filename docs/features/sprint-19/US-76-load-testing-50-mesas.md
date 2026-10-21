# US-76: Pruebas de Carga Simultáneas con 50+ Mesas en Tiempo Real

> **Sprint**: Sprint 19 (02/11 – 08/11/2026)  
> **Épica**: QA E2E, Load Testing, Polish Final & Presentación  
> **Tipo**: `Fullstack / QA`  
> **Estado**: ✅ **Resuelta**  
> **Asignado a**: Martín Oviedo + Antigravity (AI Agent)  
> **Rama de trabajo**: `feat/US-76-load-testing`  

---

## 1. Descripción de la Historia

**Como** equipo, realizamos pruebas de carga simultáneas simulando 50+ mesas enviando pedidos y recibiendo eventos en tiempo real sin degradación de rendimiento.

---

## 2. Criterios de Aceptación (Definition of Done)

- [x] Script de simulación de tráfico concurrente con k6 o herramienta equivalente.
- [x] 50 clientes simulados abriendo carta, agregando ítems y confirmando pedidos en una ventana de 5 minutos.
- [x] Tiempos de respuesta p95 por debajo de 300ms en todos los endpoints clave.
- [x] Verificación de estabilidad en el Hub de SSE de Go sin fugas de memoria ni saturación de conexiones PostgreSQL.

---

## 3. Checklist de Tareas Técnicas

- [x] Escribir script de test de carga en `load-tests/` (`load-tests/k6-50-mesas.js` y `repos/api/cmd/loadtest/main.go`).
- [x] Ejecutar pruebas en ambiente de staging / local con métricas (`20s` verificación y `5m` corrida completa con 55 mesas y 59 streams SSE).
- [x] Optimizar pool de conexiones de base de datos (`pgxpool` `MaxConns=50`, `MinConns=10`) y eliminar consultas N+1 en `pedido/store.go`.
- [x] Documentar resultados y gráficos de latencia en informe técnico (`docs/testing/load-testing-report.md`).

---

## 4. Archivos Clave Involucrados

- `repos/api/cmd/loadtest/main.go`
- `load-tests/k6-50-mesas.js`
- `repos/api/internal/db/db.go`
- `repos/api/internal/notificacion/broker.go`
- `repos/api/internal/pedido/store.go`
- `docs/testing/load-testing-report.md`

---

## 5. Notas, Dependencias y Flujo de Trabajo

- **Dependencias**: Requiere todos los flujos de pedidos e ítems terminados.
- **Estrategia Git**:
  1. Crear rama siempre a partir de `qa`:  
     ```bash
     git checkout qa && git pull
     git checkout -b feat/US-76-load-testing
     ```
  2. Implementar cambios siguiendo las reglas del proyecto ([AGENTS.md](../../AGENTS.md)).
  3. Validar builds antes de mergear:
     - Backend: `cd repos/api && go test ./...`
     - Frontend: `cd repos/web && npm run build`
  4. Abrir PR o mergear a `qa` y marcar este archivo como `✅ Resuelta`.
