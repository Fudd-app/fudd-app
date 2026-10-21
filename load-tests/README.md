# Pruebas de Carga Simultáneas con 50+ Mesas (US-76)

Este directorio contiene los scripts de simulación de tráfico concurrente para validar el rendimiento del backend Go, el Hub SSE y el pool de conexiones de PostgreSQL bajo 50+ mesas simultáneas.

## 1. Ejecución con el Generador Nativo en Go (Recomendado)

El comando `repos/api/cmd/loadtest/main.go` provisiona automáticamente un tenant en Plan Pro, una sucursal, 4 categorías, 16 artículos con variantes y **55 mesas activas**, abre **59 conexiones SSE concurrentes** (55 mesas + 2 recepcionistas + 2 pantallas KDS de cocina) y ejecuta el flujo completo de comensal y cocina.

### Prueba corta de verificación (20 segundos)
```bash
cd repos/api
go run ./cmd/loadtest -tables 55 -duration 20s -json-out ../../docs/testing/loadtest-20s.json
```

### Corrida completa de 5 minutos (SLA p95 < 300ms)
```bash
cd repos/api
go run ./cmd/loadtest -tables 55 -duration 5m -json-out ../../docs/testing/loadtest-5m.json
```

## 2. Ejecución con k6

Si tenés `k6` instalado en el sistema:

```bash
k6 run -e BASE_URL=http://localhost:8080 -e TOKEN=<jwt_token> -e SUCURSAL_ID=<sucursal_uuid> load-tests/k6-50-mesas.js
```

## 3. Informe Técnico de Resultados

Los resultados detallados (percentiles p50, p90, p95, p99, RPS, estabilidad de memoria del Hub SSE y configuración del pool PostgreSQL) se encuentran documentados en:
- [`docs/testing/load-testing-report.md`](../docs/testing/load-testing-report.md)

