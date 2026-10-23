// repos/web/lib/export.ts
//
// Utilidades de exportación para US-75:
// - Exportación a Excel / CSV con UTF-8 BOM para apertura nativa en Microsoft Excel.
// - Generación de Informe Ejecutivo imprimible en formato A4 (PDF).

import { MetricasResumenAPI } from "./api";

function escaparCSV(valor: string | number | null | undefined): string {
  if (valor === null || valor === undefined) return '""';
  const str = String(valor);
  if (str.includes(",") || str.includes('"') || str.includes("\n") || str.includes(";")) {
    return `"${str.replace(/"/g, '""')}"`;
  }
  return `"${str}"`;
}

export function descargarArchivo(contenido: string, nombreArchivo: string, mimeType = "text/csv;charset=utf-8;") {
  const blob = new Blob([contenido], { type: mimeType });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.setAttribute("href", url);
  link.setAttribute("download", nombreArchivo);
  link.style.visibility = "hidden";
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}

export function exportarExcelReporte(
  resumen: MetricasResumenAPI,
  nombreNegocio = "Mesa CLICK",
  sucursalNombre = "Todas las sucursales"
) {
  const formatoMoneda = (val: number) =>
    `$ ${Math.round(val).toLocaleString("es-AR")}`;

  const filas: string[] = [];

  // UTF-8 BOM para que Excel abra sin problemas de codificación (tildes, ñ)
  filas.push("\uFEFF");

  // 1. Encabezado del reporte
  filas.push(`${escaparCSV("REPORTE EJECUTIVO DE RENDIMIENTO & ANALÍTICA - MESA CLICK")},,,,,`);
  filas.push(`${escaparCSV("Negocio:")},${escaparCSV(nombreNegocio)},,,,`);
  filas.push(`${escaparCSV("Sucursal:")},${escaparCSV(sucursalNombre)},,,,`);
  filas.push(
    `${escaparCSV("Período analizado:")},${escaparCSV(
      `${resumen.periodo.desde.slice(0, 10)} al ${resumen.periodo.hasta.slice(0, 10)}`
    )},,,,`
  );
  filas.push(
    `${escaparCSV("Fecha de emisión:")},${escaparCSV(
      new Date().toLocaleString("es-AR", { timeZone: "America/Argentina/Buenos_Aires" })
    )},,,,`
  );
  filas.push("");

  // 2. KPIs Consolidados
  filas.push(`${escaparCSV("1. INDICADORES PRINCIPALES (KPIS)")},,,,,`);
  filas.push(
    `${escaparCSV("Facturación Total")},${escaparCSV("Ticket Promedio")},${escaparCSV(
      "Pedidos Totales"
    )},${escaparCSV("Pedidos Cerrados")},${escaparCSV("Pedidos Activos")},${escaparCSV(
      "Tiempo Medio Despacho (min)"
    )}`
  );
  filas.push(
    `${escaparCSV(formatoMoneda(resumen.facturacion_total))},${escaparCSV(
      formatoMoneda(resumen.ticket_promedio)
    )},${escaparCSV(resumen.pedidos_totales)},${escaparCSV(resumen.pedidos_cerrados)},${escaparCSV(
      resumen.pedidos_activos
    )},${escaparCSV(Math.round(resumen.tiempo_promedio_despacho_minutos))}`
  );
  filas.push("");

  // 3. Ventas por Categoría
  filas.push(`${escaparCSV("2. VENTAS POR CATEGORÍA DE PRODUCTOS")},,,,`);
  filas.push(
    `${escaparCSV("Categoría")},${escaparCSV("Unidades Vendidas")},${escaparCSV(
      "Facturación Total"
    )},${escaparCSV("Participación (%)")}`
  );
  if (resumen.por_categoria && resumen.por_categoria.length > 0) {
    for (const cat of resumen.por_categoria) {
      filas.push(
        `${escaparCSV(cat.nombre)},${escaparCSV(cat.unidades)},${escaparCSV(
          formatoMoneda(cat.monto)
        )},${escaparCSV(`${cat.porcentaje.toFixed(1)}%`)}`
      );
    }
  } else {
    filas.push(`${escaparCSV("Sin ventas por categoría en el período")},,,`);
  }
  filas.push("");

  // 4. Platos Estrella (Ranking)
  filas.push(`${escaparCSV("3. RANKING DE PLATOS MÁS VENDIDOS (TOP)")},,,,`);
  filas.push(
    `${escaparCSV("Posición")},${escaparCSV("Artículo / Plato")},${escaparCSV(
      "Unidades Vendidas"
    )},${escaparCSV("Facturación Total")}`
  );
  if (resumen.platos_estrella.length > 0) {
    resumen.platos_estrella.forEach((plato, idx) => {
      filas.push(
        `${escaparCSV(idx + 1)},${escaparCSV(plato.nombre)},${escaparCSV(
          plato.unidades
        )},${escaparCSV(formatoMoneda(plato.monto))}`
      );
    });
  } else {
    filas.push(`${escaparCSV("Sin platos vendidos en el período")},,,`);
  }
  filas.push("");

  // 5. Rendimiento por Turno
  filas.push(`${escaparCSV("4. RENDIMIENTO POR TURNO HORARIO")},,,,,`);
  filas.push(
    `${escaparCSV("Turno")},${escaparCSV("Pedidos Totales")},${escaparCSV(
      "Pedidos Cerrados"
    )},${escaparCSV("Facturación Total")},${escaparCSV("Ticket Promedio")}`
  );
  if (resumen.por_turno.length > 0) {
    for (const turno of resumen.por_turno) {
      const ticketProm =
        turno.pedidos_cerrados > 0
          ? turno.facturacion_total / turno.pedidos_cerrados
          : 0;
      filas.push(
        `${escaparCSV(turno.turno)},${escaparCSV(turno.pedidos_totales)},${escaparCSV(
          turno.pedidos_cerrados
        )},${escaparCSV(formatoMoneda(turno.facturacion_total))},${escaparCSV(
          formatoMoneda(ticketProm)
        )}`
      );
    }
  } else {
    filas.push(`${escaparCSV("Sin datos por turno en el período")},,,,`);
  }
  filas.push("");

  // 6. Estado de los Pedidos
  filas.push(`${escaparCSV("5. DISTRIBUCIÓN POR ESTADO DE PEDIDOS")},,`);
  filas.push(`${escaparCSV("Estado")},${escaparCSV("Cantidad de Pedidos")}`);
  for (const st of resumen.por_estado) {
    filas.push(`${escaparCSV(st.estado)},${escaparCSV(st.cantidad)}`);
  }

  const csvContent = filas.join("\r\n");
  const fechaDesde = resumen.periodo.desde.slice(0, 10);
  const fechaHasta = resumen.periodo.hasta.slice(0, 10);
  const nombreArchivo = `reporte_metricas_${fechaDesde}_al_${fechaHasta}.csv`;

  descargarArchivo(csvContent, nombreArchivo, "text/csv;charset=utf-8;");
}

export function imprimirInformePDF(
  resumen: MetricasResumenAPI,
  nombreNegocio = "Mesa CLICK",
  sucursalNombre = "Todas las sucursales"
) {
  const formatoMoneda = (val: number) =>
    `$ ${Math.round(val).toLocaleString("es-AR")}`;

  const fechaDesde = resumen.periodo.desde.slice(0, 10);
  const fechaHasta = resumen.periodo.hasta.slice(0, 10);
  const fechaEmision = new Date().toLocaleString("es-AR", {
    timeZone: "America/Argentina/Buenos_Aires",
    dateStyle: "medium",
    timeStyle: "short",
  });

  const ventanaImpresion = window.open("", "_blank");
  if (!ventanaImpresion) {
    alert("Por favor habilitá las ventanas emergentes en tu navegador para ver el informe imprimible.");
    return;
  }

  const filasCategorias = (resumen.por_categoria ?? [])
    .map(
      (c) => `
      <tr>
        <td style="padding: 6px 10px; border-bottom: 1px solid #e5e7eb;">${c.nombre}</td>
        <td style="padding: 6px 10px; border-bottom: 1px solid #e5e7eb; text-align: right;">${c.unidades}</td>
        <td style="padding: 6px 10px; border-bottom: 1px solid #e5e7eb; text-align: right; font-weight: 600;">${formatoMoneda(
          c.monto
        )}</td>
        <td style="padding: 6px 10px; border-bottom: 1px solid #e5e7eb; text-align: right; font-weight: 700;">${c.porcentaje.toFixed(
          1
        )}%</td>
      </tr>`
    )
    .join("");

  const filasPlatos = resumen.platos_estrella
    .map(
      (p, i) => `
      <tr>
        <td style="padding: 6px 10px; border-bottom: 1px solid #e5e7eb; font-weight: 700;">#${i + 1}</td>
        <td style="padding: 6px 10px; border-bottom: 1px solid #e5e7eb;">${p.nombre}</td>
        <td style="padding: 6px 10px; border-bottom: 1px solid #e5e7eb; text-align: right;">${p.unidades}</td>
        <td style="padding: 6px 10px; border-bottom: 1px solid #e5e7eb; text-align: right; font-weight: 600;">${formatoMoneda(
          p.monto
        )}</td>
      </tr>`
    )
    .join("");

  const filasTurnos = resumen.por_turno
    .map((t) => {
      const ticketProm = t.pedidos_cerrados > 0 ? t.facturacion_total / t.pedidos_cerrados : 0;
      return `
      <tr>
        <td style="padding: 6px 10px; border-bottom: 1px solid #e5e7eb; font-weight: 600;">${t.turno}</td>
        <td style="padding: 6px 10px; border-bottom: 1px solid #e5e7eb; text-align: right;">${t.pedidos_totales}</td>
        <td style="padding: 6px 10px; border-bottom: 1px solid #e5e7eb; text-align: right; font-weight: 600;">${formatoMoneda(
          t.facturacion_total
        )}</td>
        <td style="padding: 6px 10px; border-bottom: 1px solid #e5e7eb; text-align: right;">${formatoMoneda(
          ticketProm
        )}</td>
      </tr>`;
    })
    .join("");

  const html = `<!DOCTYPE html>
<html lang="es">
<head>
  <meta charset="utf-8" />
  <title>Informe Ejecutivo - ${nombreNegocio} (${fechaDesde} al ${fechaHasta})</title>
  <style>
    @page {
      size: A4 portrait;
      margin: 12mm 15mm;
    }
    * { box-sizing: border-box; }
    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
      color: #111827;
      background: #ffffff;
      margin: 0;
      padding: 0;
      font-size: 11px;
      line-height: 1.45;
    }
    .header {
      border-bottom: 2px solid #111827;
      padding-bottom: 12px;
      margin-bottom: 16px;
      display: flex;
      justify-content: space-between;
      align-items: flex-end;
    }
    .brand { font-size: 20px; font-weight: 800; letter-spacing: -0.03em; color: #111827; }
    .brand span { color: #287CC1; }
    .subtitle { font-size: 12px; color: #4B5563; margin-top: 3px; font-weight: 500; }
    .meta { text-align: right; font-size: 10px; color: #4B5563; }
    .kpi-grid {
      display: grid;
      grid-template-columns: repeat(4, 1fr);
      gap: 10px;
      margin-bottom: 18px;
    }
    .kpi-card {
      border: 1px solid #d1d5db;
      border-radius: 6px;
      padding: 10px 12px;
      background: #f9fafb;
    }
    .kpi-label { font-size: 9px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.05em; color: #6b7280; }
    .kpi-value { font-size: 18px; font-weight: 800; color: #111827; margin: 4px 0 2px; }
    .kpi-sub { font-size: 9px; color: #6b7280; }
    .section-title {
      font-size: 12px;
      font-weight: 700;
      color: #111827;
      border-bottom: 1px solid #e5e7eb;
      padding-bottom: 4px;
      margin: 16px 0 8px;
      text-transform: uppercase;
      letter-spacing: 0.05em;
    }
    .two-cols {
      display: grid;
      grid-template-columns: 1.1fr 0.9fr;
      gap: 16px;
      margin-bottom: 14px;
    }
    table { width: 100%; border-collapse: collapse; font-size: 10px; }
    th {
      background: #f3f4f6;
      text-align: left;
      padding: 6px 10px;
      font-weight: 700;
      color: #374151;
      border-bottom: 1px solid #d1d5db;
      font-size: 9px;
      text-transform: uppercase;
      letter-spacing: 0.04em;
    }
    .footer {
      margin-top: 24px;
      border-top: 1px solid #e5e7eb;
      padding-top: 8px;
      display: flex;
      justify-content: space-between;
      font-size: 9px;
      color: #9ca3af;
    }
    @media print {
      body { -webkit-print-color-adjust: exact; print-color-adjust: exact; }
    }
  </style>
</head>
<body>
  <div class="header">
    <div>
      <div class="brand">Mesa<span>CLICK</span> · Analytics</div>
      <div class="subtitle">${nombreNegocio} · ${sucursalNombre}</div>
    </div>
    <div class="meta">
      <div><strong>Período:</strong> ${fechaDesde} al ${fechaHasta}</div>
      <div><strong>Emisión:</strong> ${fechaEmision}</div>
    </div>
  </div>

  <div class="kpi-grid">
    <div class="kpi-card">
      <div class="kpi-label">Facturación Período</div>
      <div class="kpi-value">${formatoMoneda(resumen.facturacion_total)}</div>
      <div class="kpi-sub">${resumen.pedidos_cerrados} pedidos cerrados</div>
    </div>
    <div class="kpi-card">
      <div class="kpi-label">Ticket Promedio</div>
      <div class="kpi-value">${formatoMoneda(resumen.ticket_promedio)}</div>
      <div class="kpi-sub">Por pedido cerrado</div>
    </div>
    <div class="kpi-card">
      <div class="kpi-label">Pedidos Totales</div>
      <div class="kpi-value">${resumen.pedidos_totales}</div>
      <div class="kpi-sub">${resumen.pedidos_activos} activos · ${resumen.pedidos_cerrados} cerrados</div>
    </div>
    <div class="kpi-card">
      <div class="kpi-label">Tiempo Despacho</div>
      <div class="kpi-value">${Math.round(resumen.tiempo_promedio_despacho_minutos)} min</div>
      <div class="kpi-sub">Desde pedido hasta listo</div>
    </div>
  </div>

  <div class="two-cols">
    <div>
      <div class="section-title">Ventas por Categoría</div>
      <table>
        <thead>
          <tr>
            <th>Categoría</th>
            <th style="text-align: right;">Unid.</th>
            <th style="text-align: right;">Monto</th>
            <th style="text-align: right;">% Part.</th>
          </tr>
        </thead>
        <tbody>
          ${filasCategorias || '<tr><td colspan="4" style="text-align:center; padding:12px;">Sin ventas</td></tr>'}
        </tbody>
      </table>
    </div>

    <div>
      <div class="section-title">Platos Estrella (Top Ranking)</div>
      <table>
        <thead>
          <tr>
            <th>#</th>
            <th>Artículo</th>
            <th style="text-align: right;">Unid.</th>
            <th style="text-align: right;">Facturado</th>
          </tr>
        </thead>
        <tbody>
          ${filasPlatos || '<tr><td colspan="4" style="text-align:center; padding:12px;">Sin datos</td></tr>'}
        </tbody>
      </table>
    </div>
  </div>

  <div class="section-title">Rendimiento por Turno Horario</div>
  <table>
    <thead>
      <tr>
        <th>Turno</th>
        <th style="text-align: right;">Pedidos Totales</th>
        <th style="text-align: right;">Facturación Total</th>
        <th style="text-align: right;">Ticket Promedio</th>
      </tr>
    </thead>
    <tbody>
      ${filasTurnos || '<tr><td colspan="4" style="text-align:center; padding:12px;">Sin turnos registrados</td></tr>'}
    </tbody>
  </table>

  <div class="footer">
    <span>Mesa CLICK · Sistema de Gestión Gastronómica Digital</span>
    <span>Documento confidencial para uso administrativo y contable</span>
  </div>

  <script>
    window.addEventListener("load", () => {
      setTimeout(() => {
        window.print();
      }, 350);
    });
  </script>
</body>
</html>`;

  ventanaImpresion.document.open();
  ventanaImpresion.document.write(html);
  ventanaImpresion.document.close();
}
