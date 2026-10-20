# US-83: Gestión de Equipo de Trabajo (Credenciales Operativas + Invitación Google) y Control de Ciclo de Vida

> **Sprint**: Sprint 15 (05/10 – 11/10/2026)  
> **Épica**: Seguridad, Gestión de Personal & Multi-Tenancy  
> **Tipo**: `Fullstack / Integración`  
> **Estado**: ✅ **Resuelta (Unificada)**  
> **Integraciones**: US-84 (`/{slug}/login`) y US-85 (Google + sesión revocable)

---

## 1. Descripción de la Historia

**Como** dueño (`admin`) o encargado de un local gastronómico,  
**quiero** dar de alta a los integrantes de mi equipo de trabajo (mozos, cocineros, encargados) tanto mediante credenciales operativas (usuario y contraseña con autogeneración de claves) como mediante invitaciones por correo electrónico vinculadas a Google, y disponer de herramientas para blanquear contraseñas, bloquear accesos temporales, archivar o eliminar cuentas,  
**para** gestionar al personal operativo con flexibilidad y control total del acceso por rol y sucursal.

---

## 2. Matriz de Permisos y Alcance

| Rol | Alcance | Permisos |
|---|---|---|
| Dueño (`admin`) | Todo el tenant | Configuración global, plan/facturación, sucursales, equipo y operación completa. Crea/invita encargados, mozos y cocineros. |
| Encargado | Su sucursal | Administración y operación de la sede. Gestiona únicamente mozos y cocineros de su sucursal. |
| Mozo | Su sucursal | Recepción/pedidos (incluyendo pedidos asistidos), gestión de carta y consulta del estado de mesas. No crea, edita ni elimina mesas/QR. |
| Cocinero (`cocina`) | Su sucursal | Carta en solo lectura y KDS; solo cambia estados de ítems. |

El backend revalida usuario activo, tenant, sucursal, rol y `session_version` en cada request protegido.

---

## 3. Dos Modalidades de Alta Coexistentes

### A. Alta Directa con Credenciales Operativas (`POST /usuarios/staff`)
1. Formulario con **Nombre completo**, **Nombre de usuario (`@username`)** único dentro del tenant, **Rol**, **Sucursal**, **Email opcional** y **Contraseña** (con botón de autogeneración segura y copiado al portapapeles para WhatsApp).
2. Hash de contraseña mediante **`bcrypt`** (`HashPasswordStaff`, mínimo 8 caracteres con letras y números).
3. Permite blanquear/cambiar la contraseña en cualquier momento (`PATCH /usuarios/{id}/password`), incrementando `session_version` para revocar sesiones previas inmediatamente.

### B. Invitación por Correo y Aceptación con Google (`POST /usuarios`)
1. Envía enlace único (`/invitacion/{token}`) con vigencia de 48 horas y hash SHA-256 en `invitaciones_equipo`.
2. Permite vista previa no destructiva (`GET /publica/invitaciones/{token}`), reenvío (`POST /usuarios/invitaciones/{id}/reenviar`) y cancelación (`DELETE /usuarios/invitaciones/{id}`).
3. Al aceptar (`POST /publica/invitaciones/{token}/aceptar`), verifica que el correo de Google coincida con el email invitado y crea el usuario activo en una transacción atómica.

---

## 4. Ciclo de Vida y Pedido Asistido

- **Estados de miembro (`PATCH /usuarios/{id}/estado` y `PATCH /usuarios/{id}`)**:
  - `activo`: Acceso habilitado.
  - `bloqueado`: Suspensión inmediata con revocación de `session_version`.
  - `archivado`: Inactivo / fin de temporada (ocultable en el listado, sin acceso al sistema).
- **Baja (`DELETE /usuarios/{id}`)**: Elimina el acceso preservando el historial de pedidos (`creado_por_usuario_id ON DELETE SET NULL` y snapshot `creado_por_nombre`).
- **Pedido asistido (`POST /pedidos/asistido`)**: Dueño, encargado o mozo pueden cargar pedidos para una mesa de su alcance registrando `origen = 'mozo'`, `creado_por_usuario_id` y `creado_por_nombre`.

---

## 5. Persistencia y API

- Migraciones:
  - `031_staff_invitations_sessions_and_waiter_orders.sql`
  - `032_unify_staff_credentials_and_lifecycle.sql`
- Endpoints:
  - `GET /usuarios`, `POST /usuarios`, `POST /usuarios/staff`
  - `PATCH /usuarios/{id}`, `PATCH /usuarios/{id}/password`, `PATCH /usuarios/{id}/estado`, `DELETE /usuarios/{id}`
  - `POST /usuarios/invitaciones/{id}/reenviar`, `DELETE /usuarios/invitaciones/{id}`
  - `GET /publica/invitaciones/{token}`, `POST /publica/invitaciones/{token}/aceptar`
  - `POST /pedidos/asistido`
