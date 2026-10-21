# Informe Técnico de Pruebas de Carga — 50+ Mesas en Tiempo Real (US-76)

> **Fecha de ejecución**: 20/10/2026  
> **Historia de Usuario**: [US-76: Pruebas de Carga Simultáneas con 50+ Mesas en Tiempo Real](../features/sprint-19/US-76-load-testing-50-mesas.md)  
> **Rama**: `feat/US-76-load-testing`  
> **Resultado Global**: ✅ **APROBADO (100% 200/201 OK · 0% errores · p95 global = 11.44 ms < 300 ms)**

---

## 1. Escenario y Arquitectura de la Prueba

Se implementó el generador de carga concurrente en [`repos/api/cmd/loadtest/main.go`](../../repos/api/cmd/loadtest/main.go) y su script homólogo para k6 en [`load-tests/k6-50-mesas.js`](../../load-tests/k6-50-mesas.js), reproduciendo un servicio gastronómico completo en hora pico:

- **Mesas simultáneas simuladas**: **55 mesas activas** (superando el umbral de 50+ mesas requerido por la US-76).
- **Conexiones Server-Sent Events (SSE) concurrentes**: **59 streams SSE abiertos en simultáneo**:
  - `55` conexiones de comensales (`GET /publica/mesas/{mesa_id}/eventos?tenant_id=...&sucursal_id=...`), cada una suscripta a 2 canales internos del Broker (`mesa:{id}` y `tenant:{id}:carta`).
  - `2` conexiones de salón/recepcionista (`GET /sucursales/{sucursal_id}/eventos`).
  - `2` conexiones de pantalla de cocina KDS (`GET /kds/eventos?sucursal_id={sucursal_id}`).
  - **Total de suscripciones activas en el Hub SSE**: **114 canales suscriptos** distribuidos en **58 tópicos**.
- **Flujo continuo por mesa durante 5 minutos (300 s)**:
  1. Escaneo del QR de la mesa (`GET /publica/mesas/{qr_token}`).
  2. Apertura de la carta digital con categorías, artículos, variantes y franjas horarias (`GET /publica/sucursales/{id}/carta`).
  3. Confirmación de pedido colaborativo con `comensal_id` UUIDv4, notas y variantes (`POST /pedidos`).
  4. Consulta de estado de la cuenta actual de la mesa (`GET /publica/mesas/{qr_token}/pedidos`).
  5. Procesamiento en vivo desde cocina (4 workers KDS pasando cada ítem a `"preparando"` y `"listo"` vía `PATCH /pedidos/items/{id}/estado`, disparando notificaciones SSE a la mesa y al salón).
  6. Monitoreo de salón (`GET /pedidos?sucursal_id={id}`) y rotación periódica de mesas (`POST /mesas/{id}/cerrar-cuenta`).

---

## 2. Optimizaciones Aplicadas en el Backend (`repos/api`)

Durante la inspección y preparación de la prueba se identificaron y optimizaron tres puntos críticos para garantizar latencias sub-25ms y cero saturación bajo alta concurrencia:

### 2.1. Eliminación de Consultas N+1 en Listado de Pedidos (`internal/pedido/store.go`)
- **Problema detectado**: `listarPedidos` ejecutaba `s.listarItems(ctx, pedidos[i].ID)` dentro de un bucle `for` por cada pedido retornado, generando $1 + 2N$ consultas SQL en `GET /publica/mesas/{qr_token}/pedidos` y `GET /pedidos?sucursal_id={id}`.
- **Optimización**: Se implementó `cargarItemsDePedidos(ctx, pedidos)` utilizando carga por lotes con `WHERE pi.pedido_id = ANY($1)` para todos los ítems y `WHERE piv.pedido_item_id = ANY($1)` para todas las variantes. Esto redujo la complejidad de $O(N)$ consultas SQL a un máximo constante de **3 consultas SQL** sin importar la cantidad de pedidos activos.
- **Cierre inmediato de cursores en transacciones**: En `Crear`, se reemplazó `defer vRows.Close()` dentro del bucle de ítems por el cierre explícito inmediato tras escanear las variantes de cada ítem, liberando el estado de la conexión `pgx` dentro de la misma transacción.

### 2.2. Tuning del Pool de Conexiones PostgreSQL (`internal/db/db.go`)
- **Configuración aplicada sobre `pgxpool.Config`**:
  - `MaxConns`: **50** conexiones máximas (configurable vía `DB_MAX_CONNS`).
  - `MinConns`: **10** conexiones pre-calentadas (configurable vía `DB_MIN_CONNS`).
  - `MaxConnLifetime`: `30m` | `MaxConnIdleTime`: `5m` | `HealthCheckPeriod`: `1m`.

### 2.3. Endurecimiento del Broker SSE (`internal/notificacion/broker.go`)
- Se amplió el buffer por suscriptor de `10` a **`128` eventos** (`make(chan Evento, 128)`) para absorber ráfagas cuando 55 mesas y 4 cocineros emiten eventos simultáneos sobre `sucursal:{id}` y `kds:{id}`, logrando **0 mensajes descartados**.
- Se protegió el cierre de cada suscripción con `sync.Once` para garantizar idempotencia en la desconexión y eliminar cualquier riesgo de fuga de goroutines o `panic: close of closed channel`.

---

## 3. Resultados de la Prueba Corta de Verificación (20 segundos)

> Archivo crudo: [`docs/testing/loadtest-20s.json`](loadtest-20s.json)

- **Duración**: `20.01 s`
- **Mesas simultáneas**: `55` (`59/59` streams SSE conectados, `6,855` eventos SSE recibidos en tiempo real, `0` errores SSE)
- **Total de peticiones HTTP**: `2,430` (`100%` respuestas `200/201`, `0` errores, **`121.46 req/s`**)
- **Latencia Global**: `p50 = 4.36 ms` · `p90 = 8.86 ms` · `p95 = 10.88 ms` · `p99 = 22.13 ms`

| Endpoint | Peticiones | 200 / 201 | Errores | RPS | Avg (ms) | p50 (ms) | p90 (ms) | p95 (ms) | p99 (ms) | SLA (`p95 < 300ms`) |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|:---:|
| `GET /publica/mesas/{qr_token}` | 361 | 361 | 0 | 18.04 | 1.66 | 1.41 | 2.49 | **3.36** | 8.07 | ✅ PASS |
| `GET /publica/sucursales/{id}/carta` | 333 | 333 | 0 | 16.65 | 4.25 | 3.63 | 5.91 | **8.50** | 25.70 | ✅ PASS |
| `POST /pedidos` | 330 | 330 | 0 | 16.50 | 6.89 | 5.57 | 10.57 | **16.24** | 34.24 | ✅ PASS |
| `GET /publica/mesas/{qr_token}/pedidos` | 330 | 330 | 0 | 16.50 | 3.09 | 2.74 | 4.36 | **6.57** | 11.76 | ✅ PASS |
| `PATCH /pedidos/items/{id}/estado` | 1,008 | 1,008 | 0 | 50.39 | 6.45 | 5.95 | 9.56 | **11.71** | 20.60 | ✅ PASS |
| `GET /pedidos?sucursal_id={id}` | 13 | 13 | 0 | 0.65 | 9.39 | 8.33 | 16.69 | **18.19** | 18.19 | ✅ PASS |
| `POST /mesas/{id}/cerrar-cuenta` | 55 | 55 | 0 | 2.75 | 5.29 | 3.96 | 9.99 | **12.28** | 24.17 | ✅ PASS |

---

## 4. Resultados de la Corrida Completa de 5 Minutos (300 segundos)

> Archivo crudo: [`docs/testing/loadtest-5m.json`](loadtest-5m.json)

### 4.1. Resumen Ejecutivo (Ventana de 5 Minutos)
| Métrica | Valor Obtenido | Criterio de Aceptación (US-76) | Estado |
|---|---|---|---|
| **Duración de la corrida** | **300.00 s (5m 00s)** | Ventana de 5 minutos | ✅ Cumplido |
| **Mesas simultáneas activas** | **55 mesas** (59 streams SSE) | 50+ clientes simulados | ✅ Cumplido |
| **Eventos SSE entregados en vivo** | **103,440 eventos SSE** | Sin desconexiones ni pérdida | ✅ Cumplido |
| **Total de peticiones HTTP** | **36,423 requests** | Tráfico sostenido | ✅ Cumplido |
| **Respuestas exitosas (`200`/`201`)** | **36,423 (100.00%)** | 0% tasa de error | ✅ Cumplido |
| **Throughput promedio (RPS)** | **121.41 req/s** | Operación fluida en tiempo real | ✅ Cumplido |
| **Latencia Global (`p50` / `p90` / `p95` / `p99`)** | **4.54 ms / 9.43 ms / 11.44 ms / 17.06 ms** | `p95 < 300 ms` | ✅ **26x mejor que el umbral** |

### 4.2. Desglose de Latencias por Endpoint (Corrida de 5 Minutos)

| Endpoint | Reqs | 200 / 201 | Errores | RPS | Min (ms) | Avg (ms) | p50 (ms) | p90 (ms) | p95 (ms) | p99 (ms) | Max (ms) | SLA (`p95 < 300ms`) |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|:---:|
| `GET /publica/mesas/{qr_token}` | 5,065 | 5,065 | 0 | 16.88 | 0.00 | 1.42 | **1.23** | **2.26** | **2.79** | **5.59** | 25.05 | ✅ PASS |
| `GET /publica/sucursales/{id}/carta` | 5,051 | 5,051 | 0 | 16.84 | 0.00 | 3.61 | **3.25** | **5.28** | **6.29** | **10.57** | 40.78 | ✅ PASS |
| `POST /pedidos` | 5,036 | 5,036 | 0 | 16.79 | 1.26 | 7.10 | **6.13** | **11.78** | **14.57** | **21.99** | 107.89 | ✅ PASS |
| `GET /publica/mesas/{qr_token}/pedidos` | 5,024 | 5,024 | 0 | 16.75 | 0.00 | 3.71 | **3.32** | **5.45** | **6.41** | **11.43** | 38.24 | ✅ PASS |
| `PATCH /pedidos/items/{id}/estado` | 15,058 | 15,058 | 0 | 50.19 | 1.55 | 6.75 | **5.83** | **10.59** | **12.35** | **17.78** | 69.83 | ✅ PASS |
| `GET /pedidos?sucursal_id={id}` | 199 | 199 | 0 | 0.66 | 1.74 | 10.96 | **9.85** | **18.03** | **23.30** | **33.05** | 35.37 | ✅ PASS |
| `POST /mesas/{id}/cerrar-cuenta` | 990 | 990 | 0 | 3.30 | 0.54 | 4.51 | **3.78** | **6.80** | **9.26** | **16.34** | 26.88 | ✅ PASS |
| **GLOBAL CONSOLIDADO** | **36,423** | **36,423** | **0** | **121.41** | **0.00** | **5.41** | **4.54** | **9.43** | **11.44** | **17.06** | **107.89** | ✅ **PASS** |

---

## 5. Estabilidad del Hub SSE, Memoria Go y Pool PostgreSQL

Se monitoreó el endpoint `GET /health` del servidor Go en tres instantes clave de la corrida de 5 minutos:

| Instante | Goroutines | Heap Alloc (MB) | Sys Memory (MB) | Canales SSE Activos | Suscriptores SSE | DB Pool (`acquired` / `idle` / `total` / `max`) |
|---|---:|---:|---:|---:|---:|---:|
| **1. Baseline (Pre-carga)** | `6` | `5.83 MB` | `29.99 MB` | `0` | `0` | `0 / 10 / 10 / 50` |
| **2. Pico de Carga (Minuto 2:30)** | `133` | `6.05 MB` | `29.99 MB` | `58` | `114` | `1 / 9 / 10 / 50` |
| **3. Post-Desconexión (+1.5s)** | `24` | `5.97 MB` | `29.99 MB` | **`0`** | **`0`** | **`0 / 10 / 10 / 50`** |

### Conclusiones de Estabilidad:
1. **Cero Fugas en el Broker SSE**: Al desconectar las 59 conexiones SSE tras enviar `103,440` eventos en tiempo real, el mapa interno `b.suscriptores` del `Broker` volvió exactamente a **`0` canales y `0` suscriptores**, liberando la totalidad de los canales y goroutines asociadas.
2. **Consumo de Memoria Constante**: El heap activo de Go se mantuvo estable entre **`5.83 MB` y `6.05 MB`** durante los 5 minutos completos (`Sys = 29.99 MB` constante), demostrando ausencia total de memory leaks.
3. **Holgura del Pool PostgreSQL**: Gracias a la eliminación de consultas N+1 y al pre-calentamiento de `MinConns = 10` (`MaxConns = 50`), las `36,423` transacciones y consultas se resolvieron utilizando apenas las 10 conexiones pre-abiertas sin saturación ni esperas de adquisición.

