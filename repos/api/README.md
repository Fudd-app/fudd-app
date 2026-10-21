# fudd — API (Backend)

Servicio backend desarrollado en **Go** para la gestión de pedidos, carta digital y administración de negocios gastronómicos.

## Estado del desarrollo: Fase 2 Completada

La **Fase 2 (Backend)** está 100% completada y testeada (Sprints 3, 4, 5 y 6). El backend utiliza una arquitectura basada en servicios internos, comunicación en tiempo real mediante SSE y cuenta con su suite de pruebas unitarias e integración en CI.

## Stack Tecnológico

- **Lenguaje:** Go 1.22+
- **Base de Datos:** PostgreSQL 16+
- **Driver DB:** `pgx/v5`
- **Logging:** `slog` (estándar)
- **Ruteo:** `net/http` (Mux estándar de Go 1.22)
- **Migraciones:** Sistema propio basado en SQL puro (idempotente)

## Estructura del Repositorio

```
repos/api/
├── main.go            ← Entry point: carga env, conecta DB y corre migraciones
├── rutas.go           ← Definición centralizada de endpoints
├── internal/
│   ├── db/            ← Gestión del pool de conexiones y lógica de migraciones
│   ├── auth/          ← Lógica de Magic Link y JWT
│   ├── tenant/        ← CRUD de negocios
│   ├── sucursal/      ← CRUD de sucursales y sectores
│   ├── mesa/          ← CRUD de mesas y generación de QR tokens
│   ├── carta/         ← CRUD de categorías, artículos y variantes
│   ├── pedido/        ← Gestión de pedidos y estados
│   └── notificacion/  ← Broker SSE y comunicación en tiempo real
├── migrations/        ← Archivos .sql numerados (001 → 012)
├── .env.example       ← Plantilla de configuración
└── Dockerfile         ← Imagen para despliegue y desarrollo local
```

## Guía de Inicio Rápido

### 1. Requisitos
- Go instalado.
- Instancia de PostgreSQL activa.

### 2. Configuración
> 💡 **Para el equipo (Mateo y Martín):** Pedir el archivo `.env` configurado directamente a **Pablo Aguirre**, o crearlo copiando la plantilla:

```bash
cp .env.example .env
```

### 3. Ejecución
```bash
go run .
```
El servidor correrá automáticamente las migraciones pendientes al iniciar.

## Endpoints de Verificación

- **Health Check:** `GET /health`
  Retorna el estado del servidor y la base de datos.

## Convenciones de Desarrollo y Git Workflow

1. **Logs:** Usar `slog.Info`, `slog.Error`, etc. con atributos para logging estructurado.
2. **Errores:** Seguir el patrón de retorno de errores estándar de Go. En handlers, usar `http.Error` o responder con JSON estructurado.
3. **Migraciones:** No modificar archivos `.sql` existentes en `migrations/`. Si se requiere un cambio en el esquema, crear un nuevo archivo numerado.
4. **Contexto:** Respetar los tiempos de expiración y el manejo de contextos en las queries a la base de datos.
5. **Git Workflow Obligatorio:**
   - Todo trabajo arranca desde `qa`: `git checkout qa && git pull origin qa`
   - Crear rama de trabajo: `git checkout -b feat/US-XX-descripcion` o `git checkout -b fix/descripcion`
   - Probar y verificar tests (`go test ./...`)
   - PR y validación en `qa`
   - Una vez aprobado en `qa`, merge a `main` (despliegue automático en **Render** como `mesa-click-api` con Docker en Ohio, servicio Free Web Service).

---
fudd — Práctica Profesionalizante I.
