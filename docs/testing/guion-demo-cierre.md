# Guion de Demo Interactiva en Vivo y Cierre de la Materia (US-78)

> **Historia de Usuario**: [US-78: Demo Interactiva en Vivo y Cierre de la Materia](../features/sprint-19/US-78-demo-interactiva-cierre.md)  
> **Integrantes del Equipo**: Martín Oviedo · Mateo Silvestrin · Pablo Aguirre  
> **Negocio Demo**: **Bistró San Telmo** (`/bistro-san-telmo` · Plan PRO)

---

## 1. Preparación Previa (1 Comando)

Para poblar la base de datos con el negocio gastronómico completo (categorías, fotos, variantes, 12 mesas con QR estables, métricas históricas y reseñas):

```bash
cd repos/api
go run ./cmd/seed-demo
```

- **Credenciales Admin Demo**: `demo@mesaclick.com` / `MesaClick2026!`
- **URL Comensal Mesa 1 (QR Jurado)**: `http://localhost:3000/mesa/demo-jurado-mesa-1` (o `https://mesa-click-web.onrender.com/mesa/demo-jurado-mesa-1`)
- **URL Dashboard Admin/Recepción**: `http://localhost:3000/dashboard`
- **URL Pantalla Cocina (KDS)**: `http://localhost:3000/kds`

---

## 2. Distribución de Roles en Vivo (3 Integrantes)

| Integrante | Rol en la Demo | Dispositivo / Pantalla |
|---|---|---|
| **Integrante 1 (Martín Oviedo)** | **Comensal (Jurado) & Reputación** | Celular escaneando QR de Mesa 1 (`/mesa/demo-jurado-mesa-1`) + Sección Reseñas & Planes en Dashboard |
| **Integrante 2 (Pablo Aguirre)** | **Cocina (KDS) & Recepción en Vivo** | Tablet / Monitor secundario en `/kds` y pestaña Recepcionista en `/dashboard` |
| **Integrante 3 (Mateo Silvestrin)** | **Admin de Negocio & Analítica** | Proyector principal en `/dashboard` (Carta, Disponibilidad 86, Franjas Horarias y KPIs) |

---

## 3. Guion Paso a Paso (Duración estimada: 8 minutos)

### Acto 1 — Escaneo QR y Pedido Colaborativo en Mesa (2 min)
1. **Martín** (y los docentes del jurado desde sus celulares) escanean el QR impreso de la **Mesa 1** (`demo-jurado-mesa-1`).
2. Ingresan su nombre/alias en la pantalla de bienvenida colaborativa y navegan la carta mobile con categorías sticky.
3. Seleccionan **Provoleta Ahumada** y **Bife de Chorizo Angus**, eligen variante (*Punto Jugoso*) y agregan nota libre (*"Sin sal agregada"*).
4. Confirman el pedido desde el bottom-sheet; instantáneamente la vista pasa a **Seguimiento en vivo (Recibido)**.

### Acto 2 — Cocina KDS y Notificación SSE en Tiempo Real (2 min)
1. En el proyector dividido, **Pablo** muestra cómo la comanda aparece al instante vía SSE en la pantalla dedicada **`/kds`** con su temporizador activo y opción de impresión térmica (58/80 mm).
2. **Pablo** toca el primer ítem para pasarlo a **"En preparación"**: el celular del comensal actualiza su barra de progreso a *Preparando* sin recargar la página.
3. **Pablo** marca todos los ítems como **"Listo"**: el pedido completo pasa a *Listo para servir* tanto en el salón como en el celular del comensal.

### Acto 3 — Calificación Post-Consumo (Smart Google Funnel), Pago y Cierre (2 min)
1. Al quedar el pedido en estado **Listo**, en el celular del comensal se habilita el botón de calificación post-consumo (**1 a 5★**).
2. **Martín** envía una reseña de **5★**: el **Smart Google Review Funnel** le ofrece el CTA directo con 1 tap para publicar la recomendación en Google Maps (Place ID).
3. El comensal solicita la cuenta o abona con **Mercado Pago Sandbox (Checkout Pro)**; **Pablo** verifica la alerta en Recepcionista y cierra/rota la cuenta de la mesa.

### Acto 4 — Administración Pro, Stock 86, Carga Masiva y KPIs (2 min)
1. **Mateo** muestra en `/dashboard`:
   - **Disponibilidad (86)**: marca un plato como agotado con 1 tap y muestra cómo se bloquea en vivo en los celulares conectados.
   - **Planes Free vs Pro**: muestra las barras de uso de cuota y el bloqueo elegante con `UpgradeModal`.
   - **Métricas (KPIs) y Reputación**: exhibe el total facturado del turno, ticket promedio, ranking de platos y el CSAT con alertas de servicio en tiempo real.

