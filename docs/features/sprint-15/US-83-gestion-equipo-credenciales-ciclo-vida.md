# US-83: Acceso del equipo por invitación, rol y sucursal

> **Sprint**: 15
> **Estado**: Implementada en `feat/US-83-acceso-equipo`
> **Integraciones obligatorias**: US-84 (login por negocio) y US-85 (Google + sesión segura)

## Historia

Como dueño o encargado quiero invitar personas al equipo y asignarles un rol y una sucursal, para que cada integrante acceda solo a las funciones operativas que le corresponden.

## Decisiones definitivas

- `admin` representa al dueño. No tiene sucursal fija y puede operar todo el tenant.
- `encargado`, `mozo` y `cocina` pertenecen exactamente a una sucursal.
- No existen usuario/contraseña para staff ni estado “archivado”.
- Los estados son: invitación `pendiente` (con trazabilidad interna `usada`, `cancelada` o `reemplazada`) y usuario `activo` o `bloqueado`.
- Eliminar un usuario es una baja física. Pedidos y auditoría conservan referencias anulables y snapshots de nombre.

## Matriz de permisos

| Rol | Alcance | Permisos |
|---|---|---|
| Dueño (`admin`) | Todo el tenant | Configuración global, plan/facturación, sucursales, equipo y operación completa. Invita encargados, mozos y cocineros. |
| Encargado | Su sucursal | Administración y operación de la sede. Gestiona únicamente mozos y cocineros de su sucursal. |
| Mozo | Su sucursal | Recepción/pedidos, gestión de carta y consulta del estado de mesas. No crea, edita ni elimina mesas/QR. |
| Cocinero (`cocina`) | Su sucursal | Carta en solo lectura y KDS; solo cambia estados de ítems. |

El backend vuelve a validar usuario activo, tenant, sucursal, rol y versión de sesión en cada request protegido. Ocultar una acción en el frontend no reemplaza este control.

## Invitaciones

1. Dueño: elige email, rol y sucursal.
2. Encargado: elige email y rol `mozo`/`cocina`; su sucursal se asigna automáticamente.
3. La invitación no crea un usuario activo. Se guarda un hash del token, vence a las 48 horas y es de un solo uso.
4. Solo puede existir una invitación pendiente para el mismo email dentro del tenant. El email puede participar en tenants distintos.
5. Reenviar reemplaza el token anterior; cancelar invalida la invitación pendiente.
6. El enlace se envía al email invitado y también queda visible para que el dueño o encargado pueda copiarlo. La UI diferencia entrega confirmada de modo local o fallo del proveedor.
7. `GET /publica/invitaciones/{token}` muestra negocio, sucursal, rol y email sin consumir el token.
8. Al aceptar, Google debe devolver exactamente el email invitado. El nombre sugerido por Google puede editarse.
9. La creación del usuario, vinculación de `google_sub`, consumo de la invitación y primera sesión ocurren como una única operación lógica; rol y sucursal nunca provienen del cliente que acepta.

## Pedido asistido

Dueño, encargado o mozo pueden cargar un pedido real para una mesa de su alcance desde Recepcionista. Usa artículos, variantes, disponibilidad, notas y alias opcional de la carta real; publica los mismos eventos SSE/KDS que un pedido del comensal y registra:

- `origen = mozo`
- `creado_por_usuario_id`
- `creado_por_nombre` (snapshot histórico)

## Criterios de aceptación

- [x] Invitación pendiente, preview no destructiva, aceptación Google, expiración, cancelación y reenvío.
- [x] Email repetible entre tenants pero no duplicado dentro del mismo tenant.
- [x] Bloqueo y baja revocan acceso inmediatamente.
- [x] Dueño y encargado respetan la matriz de gestión de equipo.
- [x] Mozo puede cargar pedidos asistidos; cocinero no.
- [x] Historial de pedidos conserva el origen y creador aunque el usuario se elimine.
- [x] UI responsive para gestión del equipo, invitación y pedido asistido.
- [x] Envío al email indicado con enlace manual disponible y estado de entrega explícito.

## Persistencia y API

- Migración: `031_staff_invitations_sessions_and_waiter_orders.sql`.
- Equipo: `GET/POST /usuarios`, `PATCH/DELETE /usuarios/{id}`.
- Invitaciones: `POST /usuarios/invitaciones/{id}/reenviar`, `DELETE /usuarios/invitaciones/{id}`.
- Aceptación pública: `GET/POST /publica/invitaciones/{token}`.
- Pedido asistido: `POST /pedidos/asistido`.
