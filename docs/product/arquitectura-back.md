# Arquitectura del Backend — Mesa CLICK

Documento de referencia para la Fase 2 del proyecto. Define el stack, las entidades, las relaciones, los servicios y la estrategia de autenticación.

---

## Stack tecnológico

| Capa | Tecnología | Justificación |
|---|---|---|
| Lenguaje | Go (Golang) | Performance, tipado fuerte, excelente soporte concurrente para tiempo real |
| Base de datos | PostgreSQL | Relacional, robusto, soporte nativo para JSONB y conexiones concurrentes |
| Logging | `slog` (stdlib) | Logs estructurados sin dependencias externas |
| Monitoreo de errores | Sentry | Captura de errores en producción con contexto |
| Auth | Magic link por email | Sin contraseña, menor fricción para el admin |
| Tiempo real | Server-Sent Events (SSE) | Más simple que WebSocket para flujo unidireccional (back → front) |

---

## Entidades y relaciones

### Diagrama de entidades

```
Tenant
  └── Sucursal (1:N)
        ├── Sector (1:N)
        │     └── Mesa (1:N)
        └── Pedido (1:N)
              └── PedidoItem (1:N)
                    └── PedidoItemVariante (1:N)

Tenant
  └── Usuario (1:N)
        └── MagicToken (1:N)

Tenant
  └── Articulo (1:N)
        └── Variante (1:N)

Sucursal ──── Articulo (opcional: carta por sucursal)
```

---

### Tablas

#### `tenants`
| Campo | Tipo | Descripción |
|---|---|---|
| `id` | UUID | PK |
| `nombre` | TEXT | Nombre del negocio |
| `nombre_fantasia` | TEXT | Nombre de fantasía / marca |
| `rubro` | TEXT | Ej: cafetería, restaurante, comida rápida |
| `descripcion` | TEXT | Bio / presentación del negocio |
| `redes_sociales` | JSONB | `{ instagram, facebook, ... }` |
| `slug` | TEXT UNIQUE | Identificador en URL |
| `created_at` | TIMESTAMPTZ | |

#### `usuarios`
| Campo | Tipo | Descripción |
|---|---|---|
| `id` | UUID | PK |
| `tenant_id` | UUID | FK → tenants |
| `email` | TEXT UNIQUE | |
| `nombre` | TEXT | |
| `rol` | TEXT | `admin`, `encargado`, `mozo` |
| `created_at` | TIMESTAMPTZ | |

#### `magic_tokens`
| Campo | Tipo | Descripción |
|---|---|---|
| `id` | UUID | PK |
| `usuario_id` | UUID | FK → usuarios |
| `token` | TEXT UNIQUE | UUID aleatorio |
| `expires_at` | TIMESTAMPTZ | 15 minutos desde emisión |
| `used_at` | TIMESTAMPTZ | NULL si no fue usado |

#### `sucursales`
| Campo | Tipo | Descripción |
|---|---|---|
| `id` | UUID | PK |
| `tenant_id` | UUID | FK → tenants |
| `nombre` | TEXT | Nombre o referencia de la sucursal |
| `whatsapp` | TEXT | |
| `email` | TEXT | |
| `telefono` | TEXT | |
| `horarios` | JSONB | `{ lunes: "09:00-22:00", ... }` |
| `created_at` | TIMESTAMPTZ | |

#### `sectores`
| Campo | Tipo | Descripción |
|---|---|---|
| `id` | UUID | PK |
| `sucursal_id` | UUID | FK → sucursales |
| `nombre` | TEXT | Ej: "Salón principal", "Terraza" |

#### `mesas`
| Campo | Tipo | Descripción |
|---|---|---|
| `id` | UUID | PK |
| `sucursal_id` | UUID | FK → sucursales |
| `sector_id` | UUID | FK → sectores (nullable) |
| `numero` | INT | Número visible de la mesa |
| `capacidad` | INT | Cantidad de sillas |
| `qr_token` | TEXT UNIQUE | Token para la URL pública del QR |
| `estado` | TEXT | `activa`, `inactiva` |

#### `articulos`
| Campo | Tipo | Descripción |
|---|---|---|
| `id` | UUID | PK |
| `tenant_id` | UUID | FK → tenants |
| `sucursal_id` | UUID | FK → sucursales (nullable, si es por sucursal) |
| `categoria_id` | UUID | FK → categorias |
| `nombre` | TEXT | |
| `descripcion` | TEXT | |
| `precio` | NUMERIC(10,2) | |
| `foto_url` | TEXT | URL de la imagen |
| `activo` | BOOLEAN | Default TRUE |

#### `categorias`
| Campo | Tipo | Descripción |
|---|---|---|
| `id` | UUID | PK |
| `tenant_id` | UUID | FK → tenants |
| `nombre` | TEXT | Ej: "Entradas", "Bebidas" |
| `orden` | INT | Para ordenar la carta |

#### `variantes`
| Campo | Tipo | Descripción |
|---|---|---|
| `id` | UUID | PK |
| `articulo_id` | UUID | FK → articulos |
| `nombre` | TEXT | Ej: "Grande", "Sin azúcar" |
| `precio_adicional` | NUMERIC(10,2) | Default 0 |

#### `pedidos`
| Campo | Tipo | Descripción |
|---|---|---|
| `id` | UUID | PK |
| `mesa_id` | UUID | FK → mesas |
| `sucursal_id` | UUID | FK → sucursales |
| `estado` | TEXT | `recibido`, `confirmado`, `preparando`, `listo`, `entregado`, `cerrado` |
| `created_at` | TIMESTAMPTZ | |
| `updated_at` | TIMESTAMPTZ | |

#### `pedido_items`
| Campo | Tipo | Descripción |
|---|---|---|
| `id` | UUID | PK |
| `pedido_id` | UUID | FK → pedidos |
| `articulo_id` | UUID | FK → articulos |
| `cantidad` | INT | |
| `precio_unitario` | NUMERIC(10,2) | Snapshot del precio al momento del pedido |
| `notas` | TEXT | Notas libres del cliente ("sin cebolla") |

#### `pedido_item_variantes`
| Campo | Tipo | Descripción |
|---|---|---|
| `id` | UUID | PK |
| `pedido_item_id` | UUID | FK → pedido_items |
| `variante_id` | UUID | FK → variantes |

---

## Servicios

| Servicio | Responsabilidad |
|---|---|
| `TenantService` | Crear y configurar el negocio (tenant + registro admin) |
| `SucursalService` | CRUD de sucursales y sectores |
| `UsuarioService` | CRUD de usuarios internos, invitaciones |
| `AuthService` | Generar y validar magic links, crear sesiones JWT |
| `MesaService` | CRUD de mesas, generación del `qr_token` |
| `CartaService` | CRUD de categorías, artículos y variantes |
| `PedidoService` | Crear pedido, actualizar estado, agregar ítems, solicitar cuenta |
| `NotificacionService` | Publicar eventos a los canales SSE por sucursal |

---

## Flujo de autenticación — Magic Link

```
1. Admin ingresa su email en el formulario de login
2. POST /auth/magic-link { email }
   → AuthService genera un UUID único como token
   → Guarda en magic_tokens con expires_at = now + 15 min
   → Envía email con URL: {app}/auth/verify?token={uuid}
3. Admin hace clic en el link del email
4. GET /auth/verify?token={uuid}
   → AuthService valida: existe + no usado + no expirado
   → Marca used_at = now
   → Crea sesión: genera JWT firmado con usuario_id + tenant_id + rol
   → Redirect al panel del negocio con el JWT en cookie httpOnly
5. Cada request al panel incluye el JWT → middleware lo valida y carga el contexto del usuario
```

---

## Tiempo real — SSE (Server-Sent Events)

```
- Cada sucursal tiene un canal SSE: GET /sucursales/{id}/eventos
- El dashboard del recepcionista se suscribe a este canal al cargar
- La vista del cliente se suscribe al canal de su pedido: GET /pedidos/{id}/eventos
- Cuando el estado de un pedido cambia → PedidoService llama a NotificacionService
  → NotificacionService publica el evento en el canal correspondiente
- No se usa WebSocket para mantener la implementación simple (SSE es unidireccional back→front)
```

---

## Estructura de carpetas del servicio (Go)

```
repos/api/
├── cmd/
│   └── server/main.go          ← entry point
├── internal/
│   ├── auth/                   ← magic link, JWT, middleware
│   ├── tenant/                 ← tenant + onboarding
│   ├── sucursal/               ← sucursales + sectores
│   ├── usuario/                ← usuarios internos + invitaciones
│   ├── mesa/                   ← mesas + QR token
│   ├── carta/                  ← artículos + categorías + variantes
│   ├── pedido/                 ← pedidos + ítems + estados
│   ├── notificacion/           ← SSE + canales por sucursal
│   └── db/                     ← conexión + migraciones
├── migrations/                 ← archivos SQL versionados
└── Dockerfile
```

---

## Decisiones pendientes a confirmar

- [ ] **CUIT/CUIL y razón social**: ¿se incluyen en el tenant o son campos opcionales del perfil?
- [ ] **Carta global vs. por sucursal**: ¿los artículos pueden diferir entre sucursales del mismo negocio?
- [ ] **Menú de precios**: ¿existe lista de precios separada o el precio va directo en el artículo?
- [ ] **Exportación (Excel/CSV)**: ¿se incluye en el MVP o queda para una iteración posterior?
- [ ] **Tiempo de preparación**: ¿se configura por artículo, por categoría o por sucursal?
