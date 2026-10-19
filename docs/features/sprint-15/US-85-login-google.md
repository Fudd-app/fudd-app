# US-85: Identidad Google y sesiones revocables

> **Sprint**: 15
> **Estado**: Implementada e integrada con US-83/84

## Alcance

Google Identity Services valida la identidad tanto del dueño como del staff. La autenticación del dueño continúa disponible en `/auth/google`; el staff siempre usa el endpoint tenant-scoped de US-84 y solo puede existir después de aceptar una invitación.

La contraseña y el magic link heredados se conservan únicamente para cuentas administrativas existentes. Nunca son mecanismos de acceso del staff.

## Sesión

- JWT firmado con `usuario_id`, `tenant_id`, `rol`, `sucursal_id` y `session_version`.
- Entrega exclusiva por cookie HttpOnly; el body no expone el token.
- Vigencia de 30 días, Secure en producción, SameSite=Lax y Path `/`.
- `GET /auth/me` devuelve el DTO mínimo de la sesión vigente.
- `POST /auth/logout` incrementa `session_version` y elimina la cookie.
- Cada endpoint protegido consulta el usuario actual. Cambio de rol/sucursal, bloqueo, baja o versión distinta rechaza la sesión.

## Integración con invitaciones

- La vista previa no consume tokens.
- La credencial Google debe tener email verificado y coincidir exactamente (normalizado) con la invitación.
- Se usa el `sub` estable de Google; el nombre de Google se presenta como sugerencia editable.
- La aceptación crea el usuario activo y la primera cookie de sesión; no solicita contraseña.

## Criterios de aceptación

- [x] Login y aceptación validan ID tokens con audiencia, firma, issuer, expiración y email verificado.
- [x] Google `sub` y email son únicos dentro del tenant; el staff puede repetir identidad en otros tenants.
- [x] Cookie segura y ausencia de JWT en `localStorage`/query params.
- [x] Revocación inmediata comprobada por `session_version`.
- [x] Login posterior del staff se realiza con Google en el slug del negocio.
- [x] Tests cubren aislamiento por slug, email Google incorrecto y sesión revocada.
