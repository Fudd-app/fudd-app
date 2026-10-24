# fudd — API (Backend)

Servicio backend en **Go** para la gestión de negocios gastronómicos: tenants y planes Free/Pro, sucursales, mesas y QR, carta, pedidos y cocina en tiempo real (SSE), equipo y sesiones, métricas, reseñas y pagos con Mercado Pago.

## Estado del desarrollo

Fase 4 — Evolución, Monetización & Analítica. El sprint en curso está en [`AGENTS.md`](../../AGENTS.md) y los issues de este repo (prefijo `[api]`) en el tablero [Fudd APP](https://github.com/orgs/Fudd-app/projects/1).

## Stack Tecnológico

- **Lenguaje:** Go 1.26
- **Base de Datos:** PostgreSQL (Neon en prod y qa)
- **Driver DB:** `pgx/v5`
- **Logging:** `slog` (estándar)
- **Ruteo:** `net/http` (Mux estándar)
- **Migraciones:** sistema propio basado en SQL puro (idempotente)

## Estructura del Repositorio

```
repos/api/
├── main.go              ← entry point: carga env, conecta DB, corre migraciones y levanta HTTP
├── rutas.go             ← definición centralizada de endpoints
├── internal/
│   ├── db/              ← pool de conexiones y migraciones
│   ├── auth/            ← Google Identity, magic link, JWT y sesiones
│   ├── tenant/          ← negocios, planes Free/Pro y middleware de cuotas
│   ├── sucursal/        ← sucursales, sectores y horarios
│   ├── mesa/            ← mesas y QR tokens
│   ├── carta/           ← categorías, artículos, variantes, disponibilidad y carga masiva
│   ├── pedido/          ← pedidos y estados por ítem
│   ├── notificacion/    ← broker SSE
│   ├── usuario/         ← equipo, invitaciones y ciclo de vida del staff
│   ├── metrica/         ← agregaciones para el dashboard de KPIs
│   ├── resenia/         ← reseñas y Smart Google Funnel
│   └── mercadopago/     ← pago de cuenta (Checkout Pro)
├── cmd/
│   ├── seed-demo/       ← siembra datos demo
│   └── loadtest/        ← generador de carga
├── migrations/          ← archivos .sql numerados (001 → 033)
├── .env.example         ← plantilla de configuración
└── Dockerfile           ← imagen para despliegue y desarrollo local
```

## Guía de Inicio Rápido

Lo recomendado es levantar todo el sistema con Docker desde la raíz (`npm run setup` / `npm run dev`, ver [`LOCAL_DEV.md`](../../LOCAL_DEV.md)). Para correr solo la API:

1. **Configuración:** pedir el `.env` a **Pablo Aguirre** o crearlo desde la plantilla:
   ```bash
   cp .env.example .env
   ```
2. **Ejecución:**
   ```bash
   go run .
   ```
   El servidor corre automáticamente las migraciones pendientes al iniciar.
3. **Verificación:** `GET /health` devuelve el estado del servidor y la base de datos.

## Convenciones de Desarrollo

1. **Logs:** usar `slog.Info`, `slog.Error`, etc. con atributos para logging estructurado.
2. **Errores:** patrón de retorno de errores estándar de Go. En handlers, responder con JSON estructurado.
3. **Migraciones:** no modificar archivos `.sql` existentes en `migrations/`. Todo cambio de esquema va en un archivo nuevo numerado.
4. **Contexto:** respetar los tiempos de expiración y el manejo de contextos en las queries.
5. **Tests:** `go test ./...` en verde antes de cada PR; todo bug se corrige con un test que lo reproduce.
6. **Flujo de trabajo:** issues y PRs con prefijo `[api]`, ramas desde `qa` y code review antes del merge ([`docs/proceso/flujo-de-trabajo.md`](../../docs/proceso/flujo-de-trabajo.md)). Si un cambio altera un contrato (ruta, payload, error, evento SSE), se crea el issue `[web]` para sus consumidores.
7. **Despliegue:** Render — `qa` → `fidd-app-api-qa`, `main` → `fudd-app-api-prod` ([`docs/deploy/README.md`](../../docs/deploy/README.md)).

---
fudd — Práctica Profesionalizante I.
