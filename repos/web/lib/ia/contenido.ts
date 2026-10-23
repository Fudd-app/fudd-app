// Fuente única del contenido de la página /ia y de su versión Markdown (/ia.md).
// Los nombres de tools y campos salen de repos/api/internal/mcp (tools.go, model.go).

const MCP_URL_POR_DEFECTO = "https://api.fudd.com.ar/mcp";

export function urlMcp(): string {
  return process.env.NEXT_PUBLIC_MCP_URL || MCP_URL_POR_DEFECTO;
}

export interface ParametroTool {
  nombre: string;
  tipo: string;
  obligatorio: boolean;
  descripcion: string;
}

export interface ToolMcp {
  nombre: string;
  titulo: string;
  descripcion: string;
  parametros: ParametroTool[];
  ejemploRespuesta: unknown;
}

const PARAM_SUCURSAL: ParametroTool = {
  nombre: "sucursal_id",
  tipo: "string",
  obligatorio: false,
  descripcion: "ID de la sucursal; opcional si el negocio tiene una sola.",
};

export const TOOLS: ToolMcp[] = [
  {
    nombre: "listar_sucursales",
    titulo: "Listar sucursales",
    descripcion: "Lista las sucursales del negocio con su cantidad de mesas activas.",
    parametros: [],
    ejemploRespuesta: {
      sucursales: [{ id: "b1c2d3e4-0000-0000-0000-000000000001", nombre: "Centro", mesas_activas: 12 }],
    },
  },
  {
    nombre: "estado_salon",
    titulo: "Estado del salón",
    descripcion:
      "Estado actual de cada mesa activa: libre, ocupada, cuenta solicitada o pago habilitado, con comensales, pedidos activos, tiempo y total parcial.",
    parametros: [PARAM_SUCURSAL],
    ejemploRespuesta: {
      sucursal: { id: "b1c2d3e4-0000-0000-0000-000000000001", nombre: "Centro" },
      mesas: [
        {
          numero: 4,
          sector: "Salón",
          capacidad: 4,
          estado: "ocupada",
          comensales: 3,
          comensales_sin_identificar: false,
          pedidos_activos: 2,
          minutos_desde_primer_pedido: 35,
          total_parcial: 18500,
        },
      ],
      totales: { mesas: 12, ocupadas: 5, comensales: 14 },
    },
  },
  {
    nombre: "estado_cocina",
    titulo: "Estado de cocina",
    descripcion:
      "Pedidos en cocina (recibido, preparando, listo) con su demora y el despacho promedio de la última hora: tiempo promedio desde que entra un pedido hasta que cocina lo marca listo.",
    parametros: [PARAM_SUCURSAL],
    ejemploRespuesta: {
      sucursal: { id: "b1c2d3e4-0000-0000-0000-000000000001", nombre: "Centro" },
      pedidos: [
        { mesa: 4, estado: "preparando", minutos: 12, items_pendientes: 1, items_preparando: 2, items_listos: 0 },
      ],
      resumen: {
        recibidos: 1,
        preparando: 2,
        listos: 1,
        pedido_mas_demorado: { mesa: 4, estado: "preparando", minutos: 12 },
        despacho_promedio_ultima_hora_min: 14.5,
      },
    },
  },
  {
    nombre: "resumen_turno",
    titulo: "Resumen del turno",
    descripcion:
      "Resumen de hoy hasta ahora: pedidos, comensales atendidos, facturación, ticket promedio y despacho promedio (promedio del día sobre pedidos cerrados).",
    parametros: [PARAM_SUCURSAL],
    ejemploRespuesta: {
      sucursal: { id: "b1c2d3e4-0000-0000-0000-000000000001", nombre: "Centro" },
      desde: "2026-10-23T00:00:00-03:00",
      hasta: "2026-10-23T14:30:00-03:00",
      zona_horaria: "America/Argentina/Buenos_Aires",
      pedidos: 48,
      pedidos_cerrados: 41,
      comensales_atendidos: 96,
      despacho_promedio_min: 13.2,
      facturacion: 612000,
      ticket_promedio: 14926.83,
    },
  },
];

export interface PasoCliente {
  id: string;
  nombre: string;
  disponibilidad?: string;
  pasos: string[];
  /** Comando para copiar (usa {{URL}} como marcador). */
  comando?: string;
  notas?: string[];
}

export const CLIENTES: PasoCliente[] = [
  {
    id: "claude",
    nombre: "Claude",
    disponibilidad: "claude.ai o app de escritorio",
    pasos: [
      "Abrí Configuración → Conectores.",
      "Elegí Agregar conector personalizado.",
      "Poné el nombre «Fudd» y pegá la URL del conector.",
      "Tocá Agregar y después Conectar: se abre Fudd para que inicies sesión y autorices.",
    ],
    notas: ["En planes Team y Enterprise, primero lo tiene que agregar un administrador desde la configuración de la organización."],
  },
  {
    id: "chatgpt",
    nombre: "ChatGPT",
    disponibilidad: "Disponible en planes pagos",
    pasos: [
      "Abrí Configuración → Apps (antes «Conectores») → Avanzado.",
      "Activá el Modo desarrollador.",
      "Tocá Crear, poné el nombre «Fudd» y pegá la URL del conector.",
      "En autenticación elegí OAuth y aceptá el aviso.",
      "Tocá Crear: se abre Fudd para que inicies sesión y autorices.",
    ],
  },
  {
    id: "gemini",
    nombre: "Gemini",
    disponibilidad: "Por ahora, desde Gemini CLI",
    pasos: [
      "En tu terminal, agregá el servidor con el comando de abajo.",
      "Dentro de Gemini CLI, ejecutá /mcp auth fudd: se abre el navegador para que autorices en Fudd.",
    ],
    comando: "gemini mcp add --transport http fudd {{URL}}",
  },
  {
    id: "perplexity",
    nombre: "Perplexity",
    disponibilidad: "Web o escritorio, en planes pagos",
    pasos: [
      "Abrí Configuración → Conectores.",
      "Elegí Agregar conector personalizado → Remoto.",
      "Poné el nombre «Fudd» y pegá la URL del conector.",
      "Aceptá el aviso y tocá Agregar. Después autorizá en Fudd.",
    ],
    notas: ["En el celular puede no aparecer la opción."],
  },
];

export const NOTA_MENUS =
  "Los menús de cada asistente cambian seguido. Si no encontrás la opción, buscá «conector personalizado», «custom connector» o «MCP» en su configuración.";

export const EJEMPLOS_PREGUNTAS = [
  "¿Cuántas mesas ocupadas hay?",
  "¿Cómo viene la cocina?, ¿hay algún pedido demorado?",
  "¿Cuántos comensales atendimos hoy y cuánto facturamos?",
];

export const DATOS_VISIBLES = [
  { titulo: "Salón", detalle: "mesas, comensales y cuentas." },
  { titulo: "Cocina", detalle: "pedidos y tiempos." },
  { titulo: "Turno", detalle: "pedidos, comensales y facturación." },
];

export const REQUISITOS = [
  "Ser el dueño de la cuenta (rol administrador).",
  "Tener el Plan Pro, incluido en la prueba gratuita de 3 meses.",
];

export const AL_AUTORIZAR =
  "Cuando conectes el asistente, Fudd te muestra una pantalla que indica qué aplicación pide acceso a tu negocio. Aprobá solo si la conexión la iniciaste vos.";

export const COMO_DESCONECTAR = "Entrá a Configuración → Asistentes de IA en tu panel y tocá Revocar.";

export interface ErrorMcp {
  cuando: string;
  mensaje: string;
  queHacer: string;
}

export const ERRORES: ErrorMcp[] = [
  {
    cuando: "HTTP 401: token ausente o vencido",
    mensaje: "WWW-Authenticate con resource_metadata",
    queHacer: "El cliente MCP debe reautorizar al usuario.",
  },
  {
    cuando: "La sucursal no pertenece al negocio",
    mensaje: "Sucursal no encontrada",
    queHacer: "Usar un sucursal_id devuelto por listar_sucursales.",
  },
  {
    cuando: "El negocio tiene varias sucursales y no se indicó cuál",
    mensaje: "El negocio tiene varias sucursales: <nombre (id)>, .... Indicá sucursal_id.",
    queHacer: "Repetir la consulta con sucursal_id.",
  },
  {
    cuando: "Límite de consultas por usuario",
    mensaje: "Demasiadas consultas en el último minuto. Esperá unos segundos y volvé a intentar.",
    queHacer: "Esperar unos segundos y reintentar.",
  },
  {
    cuando: "El negocio no tiene Plan Pro",
    mensaje: "Tu plan actual no incluye la conexión con asistentes de IA (es exclusiva del Plan Pro). Podés activarlo desde Configuración → Planes.",
    queHacer: "Contratar el Plan Pro o usar la prueba gratuita de 3 meses.",
  },
  {
    cuando: "Falla inesperada del servidor",
    mensaje: "Error interno, reintentá en unos segundos",
    queHacer: "Reintentar más tarde.",
  },
];

export function origenMcp(): string {
  return new URL(urlMcp()).origin;
}

export function comandoConUrl(cliente: PasoCliente): string | undefined {
  return cliente.comando?.replaceAll("{{URL}}", urlMcp());
}

export function jsonEjemplo(valor: unknown): string {
  return JSON.stringify(valor, null, 2);
}

export function generarMarkdown(): string {
  const url = urlMcp();
  const base = origenMcp();
  const lineas: string[] = [
    "# Fudd — Servidor MCP",
    "",
    `Endpoint: ${url} · Transporte: Streamable HTTP (stateless) · Solo lectura`,
    "",
    "Fudd es una plataforma para locales gastronómicos. Este servidor MCP permite consultar en tiempo real el salón, la cocina y el turno de un negocio. No permite tomar pedidos ni modificar nada.",
    "",
    "## Requisitos",
    "",
    ...REQUISITOS.map((r) => `- ${r}`),
    "",
    "## Autorización (OAuth 2.1)",
    "",
    `- Sin token, el endpoint responde 401 con \`WWW-Authenticate\` y \`resource_metadata\` en ${base}/.well-known/oauth-protected-resource/mcp`,
    `- Metadata del servidor de autorización: ${base}/.well-known/oauth-authorization-server`,
    `- Registro dinámico de clientes (RFC 7591), PKCE S256 obligatorio, parámetro \`resource\` = ${url}`,
    "- El usuario (dueño del negocio) inicia sesión en Fudd y aprueba la conexión.",
    "",
    "## Tools",
    "",
  ];
  for (const t of TOOLS) {
    lineas.push(`### ${t.nombre}`, "", t.descripcion, "");
    if (t.parametros.length === 0) {
      lineas.push("Parámetros: ninguno.", "");
    } else {
      lineas.push("Parámetros:", "");
      for (const p of t.parametros) {
        lineas.push(`- \`${p.nombre}\` (${p.tipo}, ${p.obligatorio ? "obligatorio" : "opcional"}): ${p.descripcion}`);
      }
      lineas.push("");
    }
    lineas.push("Ejemplo de respuesta:", "", "```json", jsonEjemplo(t.ejemploRespuesta), "```", "");
  }
  lineas.push("## Errores", "");
  for (const e of ERRORES) {
    lineas.push(`- ${e.cuando}: \`${e.mensaje}\`. ${e.queHacer}`);
  }
  lineas.push("", "## Pasos por cliente", "");
  for (const c of CLIENTES) {
    lineas.push(`### ${c.nombre}`, "");
    if (c.disponibilidad) lineas.push(`${c.disponibilidad}.`, "");
    c.pasos.forEach((p, i) => lineas.push(`${i + 1}. ${p}`));
    const comando = comandoConUrl(c);
    if (comando) lineas.push("", "```bash", comando, "```");
    for (const n of c.notas ?? []) lineas.push("", n);
    lineas.push("");
  }
  lineas.push(NOTA_MENUS, "", "## Desconectar", "", COMO_DESCONECTAR, "");
  return lineas.join("\n");
}
