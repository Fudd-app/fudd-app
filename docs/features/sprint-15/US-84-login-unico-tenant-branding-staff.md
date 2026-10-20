# US-84: Portal de Login Único por Negocio con Branding del Comercio y Acceso Dual de Staff

> **Sprint**: Sprint 15 (05/10 – 11/10/2026)  
> **Épica**: Seguridad, Gestión de Personal & Multi-Tenancy  
> **Tipo**: `Fullstack / Integración`  
> **Estado**: ✅ **Resuelta (Unificada)**  
> **Ruta web**: `/{slug}/login`

---

## 1. Descripción de la Historia

**Como** integrante del equipo de un local gastronómico (mozo, cocinero o encargado),  
**quiero** contar con una página de inicio de sesión propia de mi lugar de trabajo (ej: `mesaclick.com/mi-restaurante/login`) que muestre el logo, nombre y colores del comercio y me permita ingresar tanto con mi usuario y contraseña asignados como con mi cuenta de Google,  
**para** entrar rápidamente a mi turno de trabajo y ser redirigido automáticamente a mi pantalla operativa (KDS de Cocina o Salón/Dashboard).

---

## 2. Flujo y Criterios de Aceptación

### CA-1: Cabecera Pública de Branding (`GET /publica/negocio/{slug}`)
- Devuelve `id`, `slug`, `nombre`, `nombre_fantasia`, `rubro`, `logo_url`, `color_primario` y `estilo_visual` del comercio.
- Si el `slug` no existe, `/{slug}/login` muestra una pantalla clara de *"Comercio no encontrado"*.

### CA-2: Acceso Dual Acotado al Tenant
1. **Ingreso con Usuario y Clave (`POST /publica/negocio/{slug}/login` / `POST /auth/negocios/{slug}/password`)**:
   - Autentica por `username` (con o sin `@`) o `email` + `password` exclusivamente dentro del `tenant_id` correspondiente al `slug`.
   - Un mismo nombre de usuario (ej. `cocina1` o `mozo_juan`) puede existir en distintos comercios sin colisión.
2. **Ingreso con Google (`POST /auth/negocios/{slug}/google`)**:
   - Autentica mediante `google_sub` o email verificado de Google exclusivamente dentro del `tenant_id` correspondiente al `slug`.

### CA-3: Seguridad de Sesión y Redirección por Rol
- Emite cookie `session` `HttpOnly`, `Secure` (en HTTPS/producción), `SameSite` adaptativo y vigencia de 30 días, con `session_version` revalidada en cada request.
- Usuarios con estado `bloqueado` reciben HTTP 403 con mensaje explícito de suspensión temporal.
- Usuarios con estado `archivado` o inexistentes no obtienen sesión.
- Redirección automática según rol:
  - Rol **`cocina`** → `/kds`
  - Rol **`mozo`** / **`encargado`** → `/dashboard`
