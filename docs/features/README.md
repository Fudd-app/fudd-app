# Tablero de Historias de Usuario (Fase 4) — Mesa CLICK

> Este directorio contiene el seguimiento individualizado **US por US** de la Fase 4 (Evolución, Monetización & Analítica).  
> Permite a los 3 integrantes del equipo distribuir tareas de forma paralela, asignar responsables y llevar el control de avance sin colisiones.

---

## 📊 Progreso General de la Fase 4

| Métrica | Valor |
|---|---|
| **Total de User Stories** | **38 US** (US-50 a US-87) |
| **Completadas** | **30** (79%) |
| **En Curso** | **1** |
| **Pendientes** | **7** |
| **Sprint Actual** | **Sprint 15 (05/10 – 11/10/2026)** · Sprints 16 y 17 completados por adelantado |

---

## 🛠️ Reglas para el Trabajo en Equipo (3 Desarrolladores)

1. **Tomar una US**:
   - Abrir el archivo correspondiente en su carpeta de sprint (ej. `sprint-11/US-53-rediseno-ui-base-accesibilidad.md`).
   - Cambiar **Asignado a** con tu nombre y **Estado** a `⚡ En Curso`.
   - Si trabajás en simultáneo con otro compañero, coordinen para que cada uno tome una US distinta del sprint.
2. **Creación de rama (siempre desde `qa`)**:
   ```bash
   git checkout qa && git pull
   git checkout -b feat/US-XX-descripcion-corta
   ```
3. **Validación obligatoria antes de integrar**:
   - En Backend Go: `go test ./...`
   - En Frontend Next.js: `npm run build`
4. **Cierre de US**:
   - Mergear hacia **`qa`**.
   - Marcar los checkboxes de tareas y criterios en el archivo de la US como `[x]` y el estado como `✅ Resuelta`.
   - Actualizar este tablero general.

---

## 📋 Matriz de User Stories por Sprint

### 📌 Sprint 10: Configuración Avanzada de Cuenta (31/08 – 06/09/2026)

| ID | Tipo | Título | Estado | Asignado | Archivo |
|---|---|---|---|---|---|
| **US-50** | `Frontend` | Gestión de Perfil de Negocio, Ajustes y Datos Fiscales | ✅ Resuelta | Equipo Mesa CLICK | [Ver detalle](sprint-10/US-50-perfil-negocio-ajustes.md) |
| **US-51** | `Backend` | Endpoints para Configuración de Negocio, Branding y Roles de Usuario | ✅ Resuelta | Equipo Mesa CLICK | [Ver detalle](sprint-10/US-51-backend-configuracion-tenant-usuario.md) |
| **US-52** | `Integración` | Configuración de Horarios y Turnos de Atención por Sucursal | ✅ Resuelta | Equipo Mesa CLICK | [Ver detalle](sprint-10/US-52-horarios-turnos-sucursal.md) |

### 📌 Sprint 11: Rediseño UI/UX Base & Onboarding Guiado (07/09 – 13/09/2026)

| ID | Tipo | Título | Estado | Asignado | Archivo |
|---|---|---|---|---|---|
| **US-53** | `Frontend / Fullstack` | Rediseño UI/UX Base, Sistema Accesible y Experiencia Mobile Comensal | ✅ Resuelta | Mateo Silvestrin / Equipo Mesa CLICK | [Ver detalle](sprint-11/US-53-rediseno-ui-base-accesibilidad.md) |
| **US-54** | `Frontend` | Estados Vacíos, Skeletons de Carga y Microinteracciones | ✅ Resuelta | Antigravity (AI Agent) | [Ver detalle](sprint-11/US-54-estados-vacio-carga-error.md) |
| **US-55** | `Integración` | Tour Interactivo de Onboarding para Administrador | ✅ Resuelta | Antigravity (AI Agent) | [Ver detalle](sprint-11/US-55-tour-onboarding-admin.md) |

### 📌 Sprint 12: Mobile-First Comensal (14/09 – 20/09/2026)

| ID | Tipo | Título | Estado | Asignado | Archivo |
|---|---|---|---|---|---|
| **US-56** | `Frontend` | Navegación Mobile Táctil con Categorías Sticky y Carrito Bottom-Sheet | ✅ Resuelta | Antigravity | [Ver detalle](sprint-12/US-56-navegacion-mobile-categorias-carrito.md) |
| **US-57** | `Frontend` | Personalización de Platos con Variantes, Opciones y Notas Libres | ✅ Resuelta | Antigravity | [Ver detalle](sprint-12/US-57-personalizacion-platos-variantes.md) |
| **US-81** | `Integración` | Pedidos Colaborativos en Mesa con Identificación de Comensal y División de Cuenta | ✅ Resuelta | Mateo Silvestrin | [Ver detalle](sprint-12/US-81-pedidos-colaborativos-mesa.md) |

### 📌 Sprint 13: Carga Masiva CSV/Excel & Ajuste de Precios (21/09 – 27/09/2026)

| ID | Tipo | Título | Estado | Asignado | Archivo |
|---|---|---|---|---|---|
| **US-58** | `Backend` | Procesamiento Atómico de Carga Masiva de Catálogo por CSV/Excel | ✅ Resuelta | Antigravity | [Ver detalle](sprint-13/US-58-backend-carga-masiva-csv.md) |
| **US-59** | `Integración` | Interfaz de Carga Masiva con Plantilla Oficial y Previsualización | ✅ Resuelta | Antigravity | [Ver detalle](sprint-13/US-59-interfaz-carga-masiva-errores.md) |
| **US-60** | `Integración` | Ajuste Porcentual Masivo de Precios por Categoría o Carta Completa | ✅ Resuelta | Mateo Silvestrin | [Ver detalle](sprint-13/US-60-ajuste-porcentual-precios.md) |
| **US-82** | `Integración` | Pago de Cuenta con Mercado Pago (Sandbox Checkout Pro) | ✅ Resuelta | Pablo Aguirre / Antigravity | [Ver detalle](sprint-13/US-82-pago-cuenta-mercadopago.md) |

### 📌 Sprint 14: Disponibilidad (86) & Menús por Franja Horaria (28/09 – 04/10/2026)

| ID | Tipo | Título | Estado | Asignado | Archivo |
|---|---|---|---|---|---|
| **US-61** | `Integración` | Marcado de Ítem Sin Stock (86) con Actualización en Tiempo Real | ✅ Resuelta | Antigravity (AI Agent) | [Ver detalle](sprint-14/US-61-marcado-item-sin-stock-86.md) |
| **US-62** | `Backend` | Persistencia de Disponibilidad y Reposición Automática | ✅ Resuelta | Antigravity (AI Agent) | [Ver detalle](sprint-14/US-62-backend-reposicion-disponibilidad.md) |
| **US-63** | `Integración` | Menús por Franja Horaria y Resolución Dinámica de Carta | ✅ Resuelta | Mateo Silvestrin | [Ver detalle](sprint-14/US-63-menus-franjas-horarias.md) |

### 📌 Sprint 15: KDS, Impresión de Comandas & Acceso del Equipo (05/10 – 11/10/2026)

| ID | Tipo | Título | Estado | Asignado | Archivo |
|---|---|---|---|---|---|
| **US-64** | `Backend` | Modelo de Estados por Ítem y Canal SSE de Cocina | ✅ Resuelta | Mateo Silvestrin | [Ver detalle](sprint-15/US-64-backend-estados-item-sse-cocina.md) |
| **US-65** | `Frontend` | Pantalla KDS Dedicada de Cocina con Temporizadores | ✅ Resuelta | Pablo Aguirre | [Ver detalle](sprint-15/US-65-pantalla-kds-cocina.md) |
| **US-66** | `Integración` | Interacción Táctil en KDS y Notificación al Salón | ✅ Resuelta | Pablo Aguirre | [Ver detalle](sprint-15/US-66-interaccion-kds-cambio-estados.md) |
| **US-67** | `Integración` | Impresión Térmica de Comandas (58/80 mm) y Reintento Manual | ✅ Resuelta | Pablo Aguirre + Antigravity | [Ver detalle](sprint-15/US-67-impresion-comandas-termicas.md) |
| **US-85** | `Fullstack` | Inicio de Sesión con Google para Administración | ⚡ En Curso | Mateo Silvestrin / Codex | [Ver detalle](sprint-15/US-85-login-google.md) |
| **US-86** | `Fullstack / Integración` | Imágenes de Productos, Íconos de Categoría y Carta Mobile Visual | ✅ Resuelta | Mateo Silvestrin / Codex | [Ver detalle](sprint-15/US-86-imagenes-iconos-carta-mobile.md) |
| **US-87** | `Fullstack / Frontend` | Editor Visual Pro de Apariencia de la Carta | ✅ Resuelta | Mateo Silvestrin / Codex | [Ver detalle](sprint-15/US-87-editor-visual-apariencia-carta.md) |
| **US-83** | `Fullstack` | Gestión de Equipo de Trabajo con Credenciales y Ciclo de Vida | ✅ Resuelta | Pablo Aguirre + Antigravity | [Ver detalle](sprint-15/US-83-gestion-equipo-credenciales-ciclo-vida.md) |
| **US-84** | `Fullstack` | Portal de Login Único por Negocio (Tenant-Scoped) con Branding | ✅ Resuelta | Pablo Aguirre + Antigravity | [Ver detalle](sprint-15/US-84-login-unico-tenant-branding-staff.md) |

### 📌 Sprint 16: Modelo Freemium (Free vs Pro) & Control de Suscripciones (12/10 – 18/10/2026)

| ID | Tipo | Título | Estado | Asignado | Archivo |
|---|---|---|---|---|---|
| **US-68** | `Backend` | Esquema de Suscripciones y Middleware de Control de Cuotas | ✅ Resuelta | Martín Oviedo + Antigravity (AI Agent) | [Ver detalle](sprint-16/US-68-backend-cuotas-middleware-planes.md) |
| **US-69** | `Frontend` | Sección 'Planes y Suscripción' con Métricas de Uso de Cuota | ✅ Resuelta | Martín Oviedo + Antigravity (AI Agent) | [Ver detalle](sprint-16/US-69-interfaz-gestion-planes-uso.md) |
| **US-70** | `Integración` | Bloqueo Elegante con Modal de Upgrade al Superar Límites | ✅ Resuelta | Martín Oviedo + Antigravity (AI Agent) | [Ver detalle](sprint-16/US-70-bloqueo-elegante-modal-upgrade.md) |
| **US-71** | `Frontend / Backend` | Personalización Exclusiva Pro y Retiro de Marca de Agua | ✅ Resuelta | Martín Oviedo + Antigravity (AI Agent) | [Ver detalle](sprint-16/US-71-personalizacion-pro-marca-agua.md) |

### 📌 Sprint 17: Dashboard con Métricas (KPIs) (19/10 – 25/10/2026)

| ID | Tipo | Título | Estado | Asignado | Archivo |
|---|---|---|---|---|---|
| **US-72** | `Backend` | Consultas SQL de Agregación Optimizadas para Métricas | ✅ Resuelta | Mateo Silvestrin / Codex | [Ver detalle](sprint-17/US-72-backend-consultas-agregacion-sql.md) |
| **US-73** | `Frontend` | Panel Visual con Tarjetas de KPIs y Métricas de Operación | ✅ Resuelta | Mateo Silvestrin / Codex | [Ver detalle](sprint-17/US-73-dashboard-kpis-metricas.md) |

### 📌 Sprint 18: Business Analytics, Reputación & Exportación (26/10 – 01/11/2026)

| ID | Tipo | Título | Estado | Asignado | Archivo |
|---|---|---|---|---|---|
| **US-74** | `Frontend` | Gráficos de Ventas por Categoría y Mapa de Calor por Horario | 📋 Pendiente | Por asignar | [Ver detalle](sprint-18/US-74-graficos-ventas-mapa-calor.md) |
| **US-75** | `Integración` | Filtros Temporales y Exportación de Reportes a PDF y Excel | 📋 Pendiente | Por asignar | [Ver detalle](sprint-18/US-75-filtros-exportacion-pdf-excel.md) |
| **US-79** | `Integración` | Calificación Post-Consumo y Smart Google Review Funnel | ✅ Resuelta | Martín Oviedo + Antigravity (AI Agent) | [Ver detalle](sprint-18/US-79-resenas-smart-google-funnel.md) |
| **US-80** | `Frontend` | Dashboard de Reseñas y Reputación con CSAT y Alertas de Servicio | ✅ Resuelta | Martín Oviedo + Antigravity (AI Agent) | [Ver detalle](sprint-18/US-80-dashboard-resenas-reputacion.md) |

### 📌 Sprint 19: QA E2E, Load Testing, Polish Final & Presentación (02/11 – 08/11/2026)

| ID | Tipo | Título | Estado | Asignado | Archivo |
|---|---|---|---|---|---|
| **US-76** | `Fullstack / QA` | Pruebas de Carga Simultáneas con 50+ Mesas en Tiempo Real | 📋 Pendiente | Por asignar | [Ver detalle](sprint-19/US-76-load-testing-50-mesas.md) |
| **US-77** | `Frontend` | Auditorías de Lighthouse y Accesibilidad WCAG (>90) | 📋 Pendiente | Por asignar | [Ver detalle](sprint-19/US-77-auditoria-lighthouse-accesibilidad.md) |
| **US-78** | `Todos` | Demo Interactiva en Vivo y Cierre de la Materia | 📋 Pendiente | Por asignar | [Ver detalle](sprint-19/US-78-demo-interactiva-cierre.md) |

---

## 💡 Documentos Complementarios y Propuestas
- [Propuesta de Reseñas al Local & Smart Google Funnel](reseñas-local.html) (Aporte de Martín para US-79 y US-80).
- [Presentación Interactiva de Fase 4](../presentations/parte-2/index.html).
- [Reglas y Arquitectura General](../../AGENTS.md).
