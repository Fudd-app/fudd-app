# US-84: Login de staff acotado al negocio

> **Sprint**: 15
> **Estado**: Implementada junto con US-83
> **Ruta web**: `/{slug}/login`
## Historia

Como integrante invitado quiero volver a entrar al negocio con mi cuenta de Google, sin contraseña y sin poder elegir otro tenant o sucursal.

## Flujo definitivo

1. La URL incluye el slug del negocio.
2. El miembro elige su cuenta de Google.
3. La API busca `google_sub` o email verificado únicamente dentro del tenant identificado por ese slug.
4. Solo se autentican usuarios staff ya creados mediante una invitación aceptada y en estado `activo`.
5. El tenant, rol y sucursal se toman de la base; nunca del navegador.
6. La sesión redirige a Dashboard o KDS según rol.

Un mismo email puede trabajar en negocios distintos porque el slug desambigua el acceso. El login global `/login` permanece reservado a dueños/administradores.

## Seguridad

- Cookie `session` HttpOnly, Secure en producción, SameSite=Lax y vigencia de 30 días.
- No se guarda JWT en `localStorage` ni se agrega a URLs SSE.
- Las conexiones SSE usan credenciales de cookie.
- El backend revalida usuario activo, tenant, sucursal, rol y `session_version` en cada request.
- Bloquear, eliminar o cerrar sesión invalida inmediatamente JWT emitidos anteriormente.

## Criterios de aceptación

- [x] Login Google tenant-scoped en `POST /auth/negocios/{slug}/google`.
- [x] Usuario inexistente, bloqueado o perteneciente a otro tenant no obtiene sesión.
- [x] Staff no puede iniciar con contraseña o magic link.
- [x] Navegación responsive y acciones visibles según rol.
- [x] La API sigue siendo la autoridad aunque se manipule la UI.
