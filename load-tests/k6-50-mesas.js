/**
 * US-76: Pruebas de Carga Simultáneas con 50+ Mesas en Tiempo Real (k6)
 *
 * Simula 55 comensales concurrentes (1 por mesa) durante 5 minutos:
 *   1. GET /publica/mesas/{qr_token}               -> Escanear QR de la mesa (200 OK)
 *   2. GET /publica/sucursales/{id}/carta          -> Consultar carta digital (200 OK)
 *   3. POST /pedidos                               -> Enviar pedido colaborativo con comensal_id UUIDv4 (201 Created)
 *   4. GET /publica/mesas/{qr_token}/pedidos       -> Consultar cuenta actual (200 OK)
 *   5. PATCH /pedidos/items/{id}/estado            -> KDS cocina avanza ítems a "preparando" y "listo" (200 OK)
 *
 * Ejecución con k6:
 *   k6 run -e BASE_URL=http://localhost:8080 -e TOKEN=<jwt> -e SUCURSAL_ID=<uuid> load-tests/k6-50-mesas.js
 *
 * O usando el generador nativo en Go (que provisiona automáticamente el tenant Pro, 55 mesas y 59 streams SSE):
 *   cd repos/api && go run ./cmd/loadtest -tables 55 -duration 5m
 */

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend, Rate } from 'k6/metrics';

const qrLatency = new Trend('latency_mesa_qr', true);
const cartaLatency = new Trend('latency_carta_publica', true);
const crearPedidoLatency = new Trend('latency_crear_pedido', true);
const cuentaLatency = new Trend('latency_listar_cuenta', true);
const kdsItemLatency = new Trend('latency_kds_estado_item', true);
const errorRate = new Rate('http_error_rate');

export const options = {
  scenarios: {
    mesas_simultaneas: {
      executor: 'constant-vus',
      vus: 55,
      duration: __ENV.DURATION || '5m',
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<300'],
    latency_mesa_qr: ['p(95)<300'],
    latency_carta_publica: ['p(95)<300'],
    latency_crear_pedido: ['p(95)<300'],
    latency_listar_cuenta: ['p(95)<300'],
    latency_kds_estado_item: ['p(95)<300'],
    http_error_rate: ['rate==0'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

function uuidv4() {
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function (c) {
    const r = (Math.random() * 16) | 0;
    const v = c === 'x' ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}

export function setup() {
  const token = __ENV.TOKEN || '';
  const sucursalId = __ENV.SUCURSAL_ID || '';
  const headers = token
    ? { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }
    : { 'Content-Type': 'application/json' };

  // Obtener mesas activas de la sucursal
  const resMesas = http.get(`${BASE_URL}/mesas?sucursal_id=${sucursalId}`, { headers });
  const mesas = resMesas.status === 200 ? resMesas.json() : [];

  // Obtener carta pública para extraer IDs reales de artículos y variantes
  const resCarta = http.get(`${BASE_URL}/publica/sucursales/${sucursalId}/carta`);
  const carta = resCarta.status === 200 ? resCarta.json() : { categorias: [] };
  const articulos = [];
  for (const cat of carta.categorias || []) {
    for (const art of cat.articulos || []) {
      if (art.disponible) {
        articulos.push({
          id: art.id,
          varianteId: art.variantes && art.variantes.length > 0 ? art.variantes[0].id : null,
        });
      }
    }
  }

  return { token, sucursalId, mesas, articulos };
}

export default function (data) {
  if (!data.mesas.length || !data.articulos.length) {
    sleep(1);
    return;
  }

  const idx = (__VU - 1) % data.mesas.length;
  const mesa = data.mesas[idx];
  const comensalId = uuidv4();

  // 1. Escanear QR de la mesa
  const resQR = http.get(`${BASE_URL}/publica/mesas/${mesa.qr_token}`);
  qrLatency.add(resQR.timings.duration);
  const okQR = check(resQR, { 'GET /publica/mesas/{qr} 200': (r) => r.status === 200 });
  errorRate.add(!okQR);
  sleep(0.6);

  // 2. Consultar carta pública
  const resCarta = http.get(`${BASE_URL}/publica/sucursales/${data.sucursalId}/carta`);
  cartaLatency.add(resCarta.timings.duration);
  const okCarta = check(resCarta, { 'GET /publica/sucursales/{id}/carta 200': (r) => r.status === 200 });
  errorRate.add(!okCarta);
  sleep(0.6);

  // 3. Crear pedido con 1-2 ítems
  const art = data.articulos[Math.floor(Math.random() * data.articulos.length)];
  const item = {
    articulo_id: art.id,
    cantidad: 1,
    notas: 'sin sal',
    comensal_id: comensalId,
    comensal_nombre: `Comensal VU ${__VU}`,
  };
  if (art.varianteId) {
    item.variantes = [art.varianteId];
  }

  const orderPayload = JSON.stringify({
    mesa_id: mesa.id,
    items: [item],
  });

  const resOrder = http.post(`${BASE_URL}/pedidos`, orderPayload, {
    headers: { 'Content-Type': 'application/json' },
  });
  crearPedidoLatency.add(resOrder.timings.duration);
  const okOrder = check(resOrder, { 'POST /pedidos 201': (r) => r.status === 201 });
  errorRate.add(!okOrder);
  sleep(0.6);

  // 4. Consultar cuenta actual de la mesa
  const resCuenta = http.get(`${BASE_URL}/publica/mesas/${mesa.qr_token}/pedidos`);
  cuentaLatency.add(resCuenta.timings.duration);
  const okCuenta = check(resCuenta, { 'GET /publica/mesas/{qr}/pedidos 200': (r) => r.status === 200 });
  errorRate.add(!okCuenta);

  // 5. Avanzar ítem en KDS (si tenemos token de staff)
  if (okOrder && data.token) {
    const body = resOrder.json();
    if (body && body.items && body.items.length > 0) {
      const itemId = body.items[0].id;
      const resKds = http.patch(
        `${BASE_URL}/pedidos/items/${itemId}/estado`,
        JSON.stringify({ estado: 'listo' }),
        {
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${data.token}`,
          },
        }
      );
      kdsItemLatency.add(resKds.timings.duration);
      const okKds = check(resKds, { 'PATCH /pedidos/items/{id}/estado 200': (r) => r.status === 200 });
      errorRate.add(!okKds);
    }
  }

  sleep(0.6);
}

