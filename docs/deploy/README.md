# Documentación de Ambientes y Despliegue (Fudd / Mesa CLICK)

Este documento detalla la infraestructura, cuentas, dominios y configuración de los ambientes de **Producción (`prod`)** y **QA (`qa`)**.

---

## 1. Cuenta Base y Accesos Generales

Para centralizar el alta y administración de todos los servicios de infraestructura se creó una cuenta de Google dedicada:

- **Cuenta Google principal:** `adminmesaclick@gmail.com`
- **Uso:** Inicio de sesión (SSO con Google o usuario/contraseña homóloga a la cuenta de Google) en Cloudflare, GitHub, Vercel, Render y Neon.

---

## 2. Dominio y DNS

- **Dominio:** `fudd.com.ar`
- **Registrador:** NIC Argentina (registrado con el usuario de ARCA de **Pablo Aguirre**).
- **Fecha de alta:** 22/10/2026
- **Fecha de vencimiento:** 22/10/2027
- **Proveedor DNS:** Delegado a **Cloudflare**.
  - **Acceso a Cloudflare:** Usuario `adminmesaclick@gmail.com` (misma contraseña que la cuenta de Google).

---

## 3. Repositorio y Control de Versiones (GitHub)

- **Cuenta de usuario GitHub (vinculada con Google):** [https://github.com/adminmesaclic](https://github.com/adminmesaclic)
- **Organización:** [https://github.com/Fudd-app](https://github.com/Fudd-app)
- **Repositorio principal:** [https://github.com/Fudd-app/fudd-app](https://github.com/Fudd-app/fudd-app)

---

## 4. Frontend (Vercel)

El frontend (`Next.js`) se despliega en **Vercel**, autenticado mediante la cuenta de Google (`adminmesaclick@gmail.com`) y conectado al repositorio `Fudd-app/fudd-app`.

- **Organización en Vercel:** [https://vercel.com/fuddapp](https://vercel.com/fuddapp)

| Ambiente | URL | Redirecciones | Rama Git | Root Directory |
| :--- | :--- | :--- | :--- | :--- |
| **Producción (`prod`)** | `https://fudd.com.ar` | `www.fudd.com.ar` ➔ `fudd.com.ar` (Redirección permanente) | `main` | `/repos/web` |
| **QA (`qa`)** | `https://qa.fudd.com.ar` | Sin `www` | `qa` | `/repos/web` |

- **Node.js:** 24.x en ambos proyectos (Settings → General → Node.js Version). Debe coincidir con `engines.node` de `repos/web/package.json`; si se cambia el major, actualizar también Vercel, `ARG NODE_VERSION` del `Dockerfile` y el README.

---

## 5. Backend / API (Render)

El backend (`Go`) se despliega en **Render**, autenticado mediante la cuenta de Google (`adminmesaclick@gmail.com`). Ambos ambientes conviven dentro del **mismo proyecto y cuenta de Render**, configurados como **dos servicios distintos**:

| Ambiente | Service Name | Rama Git | Root Directory |
| :--- | :--- | :--- | :--- |
| **Producción (`prod`)** | `fudd-app-api-prod` | `main` | `/repos/api` |
| **QA (`qa`)** | `fidd-app-api-qa` | `qa` | `/repos/api` |

---

## 6. Bases de Datos (PostgreSQL)

### Ambiente de Producción (`prod`)
- **Proveedor:** [Neon](https://console.neon.tech)
- **Cuenta de acceso:** Google (`adminmesaclick@gmail.com`)
- **Connection String (formato):**
  ```text
  postgresql://user:clave@ep-long-feather-b5lxyy77-pooler.c-7.us-east-2.aws.neon.tech/neondb?sslmode=require&channel_binding=require
  ```

### Ambiente de QA (`qa`)
- **Proveedor:** [Neon](https://console.neon.tech)
- **Cuenta de acceso:** `aguirrepablo@iresm.edu.ar` (cuenta institucional IRESM de Pablo Aguirre).
  > **Pendiente:** Evaluar cómo migrar o generar un acceso compartido para todo el equipo.
- **Connection String (formato):**
  ```text
  postgresql://user:pass@ep-young-brook-b5y8igob-pooler.c-7.us-east-2.aws.neon.tech/neondb?sslmode=require&channel_binding=require
  ```
