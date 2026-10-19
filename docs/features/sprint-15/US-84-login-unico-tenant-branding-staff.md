# US-84: Portal de Login Único por Negocio (Tenant-Scoped) con Identidad de Marca y Acceso de Staff

> **Sprint**: Sprint 15 (05/10 – 11/10/2026)  
> **Épica**: Seguridad, Gestión de Personal & Multi-Tenancy  
> **Tipo**: `Fullstack / Integración`  
> **Estado**: ✅ **Resuelta**  
> **Asignado a**: Pablo Aguirre + Antigravity (AI Agent)  
> **Rama de trabajo**: `feat/US-67-impresion-termica`  

---

## 1. Descripción de la Historia

**Como** miembro del equipo operativo (mozo, cocinero o encargado de sucursal),  
**quiero** ingresar al sistema a través de un portal de login exclusivo y personalizado de mi negocio (`/[slug]/login`), visualizando el logo y nombre de fantasía del local, autenticándome con mi nombre de usuario y contraseña,  
**para** acceder de manera directa a mi estación de trabajo (KDS en cocina o panel de salón), mientras el dueño o administrador continúa ingresando de forma independiente desde el portal maestro de la plataforma vía Magic Link.

---

## 2. Criterios de Aceptación (Definition of Done)

### CA-1: Ruta Pública de Login por Negocio (`/[slug]/login`)
- Cada comercio registrado dispone de una URL dedicada para su personal:  
  `https://mesaclick.com/[slug]/login` (ej: `/pizzeria-roma/login`, `/cafe-central/login`).
- Si el slug no existe o está inactivo, se muestra una página de error amigable indicando "Comercio no encontrado" con enlace a la landing de Mesa CLICK.

### CA-2: Aislamiento Estricto por Tenant (Scope de Seguridad)
- La autenticación se resuelve **estrictamente dentro del ámbito del `tenant_id`** asociado al slug recibido.
- Permite que múltiples locales utilicen nombres de usuario convencionales (ej: `cajero`, `cocina`, `mozo1`) sin colisión entre diferentes comercios de la plataforma.
- Si un usuario ingresa credenciales válidas correspondientes a otro restaurante, el sistema rechaza el intento ("Usuario no registrado en este comercio").

### CA-3: Branding y Personalización del Negocio
- La pantalla de login del negocio muestra:
  - **Logotipo oficial del restaurante** en alta calidad.
  - **Nombre de fantasía** del comercio.
  - Aplicación sutil del **color primario corporativo** del local en botones y focos.
  - Leyenda clara: *"Acceso de Equipo · [Nombre del Negocio]"*.
  - Enlace secundario discreto: *"¿Sos el dueño o administrador del negocio? Ingresar al panel general"*, que redirige al login maestro (`/login`).

### CA-4: Doble Flujo de Autenticación Coexistente
- **Login Maestro General (`/login`)**:
  - Exclusivo para el creador / dueño del negocio.
  - Mantiene el flujo de **Magic Link por email** (sin contraseña) y token JWT con permisos de Tenant Admin.
- **Login de Personal (`/[slug]/login`)**:
  - Exclusivo para el staff operativo.
  - Formulario con campos: **Nombre de Usuario** y **Contraseña**.
  - No requiere email ni magic link; genera una sesión JWT firmada con el `usuario_id`, `tenant_id`, `sucursal_id` y `rol` correspondientes.

### CA-5: Redirección Automática Inteligente por Rol
- Al autenticarse correctamente, el sistema redirige al usuario según su responsabilidad operativa:
  - Rol **`cocina`**: Redirige directamente a la pantalla de cocina **`/kds`**.
  - Rol **`mozo`**: Redirige a **`/dashboard`** con vista inicial en la sección **Recepcionista / Salón en Vivo**.
  - Rol **`encargado`**: Redirige a **`/dashboard`**.

### CA-6: Control de Estados de Usuario y Mensajes Claros
- Si un usuario con credenciales correctas tiene estado `bloqueado`, el sistema deniega el acceso con el mensaje:  
  *"Tu usuario ha sido suspendido temporalmente. Por favor, consultá con el encargado del local."*
- Si tiene estado `archivado` o ha sido eliminado, responde *"Credenciales inválidas"*.
- Feedback visual de carga durante la autenticación para evitar pulsaciones múltiples.

---

## 3. Checklist de Tareas Técnicas

### 🔹 Backend (`repos/api` - Go)
- [x] **Endpoint Público de Información de Tenant (`GET /publica/negocio/{slug}`)**:
  - Retornar nombre de fantasía, logo URL, color primario y estado activo del negocio para armar la cabecera visual del login.
- [x] **Endpoint de Login de Staff (`POST /publica/negocio/{slug}/login`)**:
  - [x] Buscar el tenant por `slug`.
  - [x] Buscar el usuario por `tenant_id` y `username` (case-insensitive) donde `deleted_at IS NULL`.
  - [x] Verificar que el usuario no esté en estado `bloqueado`.
  - [x] Comparar contraseña mediante `bcrypt.CompareHashAndPassword`.
  - [x] Generar token JWT con claims estándar: `user_id`, `tenant_id`, `sucursal_id`, `rol`, expiración.
  - [x] Retornar el token JWT y los datos del usuario + ruta sugerida de redirección (`/kds` o `/dashboard`).
- [x] **Tests de Backend**:
  - [x] Test unitario de autenticación con contraseña válida.
  - [x] Test de rechazo por contraseña incorrecta.
  - [x] Test de rechazo cuando el usuario está en estado `bloqueado`.
  - [x] Test de aislamiento: usuario de tenant A no puede loguearse en tenant B aunque use mismo username y clave.

### 🔹 Frontend (`repos/web` - Next.js)
- [x] **Ruta Dinámica `app/[slug]/login/page.tsx`**:
  - [x] Cargar datos públicos del negocio a partir del slug (SSG / SSR con revalidación o fetch inicial).
  - [x] Renderizar tarjeta de login con el branding del comercio (logo, nombre, colores).
  - [x] Formulario accesible con inputs:
    - `username` (autofocus, teclado texto/alfanumérico).
    - `password` (con toggle de mostrar/ocultar clave 👁️).
  - [x] Manejo de errores con banners explicativos (credenciales incorrectas, usuario bloqueado, error de red).
  - [x] Almacenar token en cookie/localStorage mediante `guardarToken()`.
  - [x] Redirección automática según el rol recibido (`/kds` o `/dashboard`).
- [x] **Ajuste en Login Maestro (`app/login/page.tsx`)**:
  - [x] Agregar aclaración informativa: *"¿Sos mozo o cocinero? Ingresá a través del portal de tu local (ej: mesaclick.com/tu-local/login)"*.

---

## 4. Archivos Clave Involucrados

- `repos/api/internal/auth/handler.go`
- `repos/api/internal/auth/service.go`
- `repos/api/internal/usuario/store.go`
- `repos/web/app/[slug]/login/page.tsx`
- `repos/web/app/login/page.tsx`
- `repos/web/lib/api.ts`
- `repos/web/components/brand/Logo.tsx`

---

## 5. Notas, Dependencias y Flujo de Trabajo

- **Dependencias**: Depende de **US-83** (que crea las columnas `username`, `password_hash` y `estado`).
- **Estrategia Git**:
  1. Crear rama siempre a partir de `qa`:  
     ```bash
     git checkout qa && git pull
     git checkout -b feat/US-84-login-tenant-branding
     ```
  2. Implementar cambios siguiendo las reglas del proyecto ([AGENTS.md](../../AGENTS.md)).
  3. Validar builds antes de mergear:
     - Backend: `cd repos/api && go test ./...`
     - Frontend: `cd repos/web && npm run build`
  4. Abrir PR o mergear a `qa` y marcar este archivo como `✅ Resuelta`.
