// repos/web/lib/api.ts
//
// Cliente HTTP centralizado para interactuar con la API REST de fudd (Go).
// Provee métodos tipados para autenticación, tenant/onboarding, carta, mesas, sucursales y equipo.

export function getApiBaseUrl(): string {
  if (typeof window !== 'undefined') {
    if (window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1') {
      return 'http://localhost:8080';
    }
    if (/^\d+\.\d+\.\d+\.\d+$/.test(window.location.hostname)) {
      return `http://${window.location.hostname}:8080`;
    }

    // En producción la sesión se transporta por una cookie HttpOnly. Consumir
    // la API mediante el mismo origen evita que el navegador la trate como una
    // cookie de terceros (web y API viven en servicios Render separados).
    return '/api-proxy';
  }
  return process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
}

export class ApiError extends Error {
  status: number;
  data: unknown;

  constructor(message: string, status: number, data?: unknown) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.data = data;
  }
}

export interface DetallePlanLimit {
  recurso: 'mesas' | 'productos' | 'sucursales' | 'carga_masiva';
  limite: number;
  uso: number;
  plan: 'free' | 'pro';
}

export function esPlanLimitReached(err: unknown): boolean {
  if (typeof err === 'object' && err !== null) {
    const apiErr = err as { status?: number; data?: unknown };
    if (apiErr.status === 403 && typeof apiErr.data === 'object' && apiErr.data !== null) {
      const data = apiErr.data as Record<string, unknown>;
      return data.codigo === 'PLAN_LIMIT_REACHED';
    }
  }
  return false;
}

export function codigoError(err: unknown): string | undefined {
  if (typeof err === 'object' && err !== null) {
    const apiErr = err as { data?: unknown };
    if (typeof apiErr.data === 'object' && apiErr.data !== null) {
      const codigo = (apiErr.data as Record<string, unknown>).codigo;
      if (typeof codigo === 'string') return codigo;
    }
  }
  return undefined;
}

export function detallePlanLimit(err: unknown): DetallePlanLimit | null {
  if (!esPlanLimitReached(err)) return null;
  const data = (err as { data: Record<string, unknown> }).data;
  if (typeof data.detalle === 'object' && data.detalle !== null) {
    return data.detalle as DetallePlanLimit;
  }
  return null;
}

function getApiErrorMessage(data: unknown, fallback: string) {
  if (typeof data === 'object' && data !== null) {
    const payload = data as Record<string, unknown>;
    if (payload.codigo === 'PLAN_LIMIT_REACHED') {
      if (typeof payload.error === 'string') return payload.error;
      if (typeof payload.mensaje === 'string') return payload.mensaje;
    }
    if (typeof payload.error === 'string') return payload.error;
    if (typeof payload.mensaje === 'string') return payload.mensaje;
  }
  return fallback;
}

export function getErrorMessage(error: unknown, fallback: string) {
  if (esPlanLimitReached(error)) {
    const err = error as ApiError;
    if (typeof err.message === 'string' && err.message) return err.message;
  }
  const message = error instanceof Error && error.message ? error.message : fallback;
  return toUserMessage(message, fallback);
}

function toUserMessage(message: string, fallback: string) {
  const normalized = message.trim().toLowerCase();

  if (!normalized) return fallback;
  if (normalized === 'failed to fetch' || normalized.includes('networkerror')) {
    return 'No pudimos conectar con el servidor. Revisá tu conexión e intentá nuevamente.';
  }
  if (normalized.includes('slug') || normalized.includes('nombre de url')) {
    return 'Ese nombre en URL ya está en uso. Probá con otro.';
  }
  if (normalized.includes('cuenta registrada con ese correo')) {
    return 'No encontramos una cuenta registrada con ese correo.';
  }
  if (
    normalized.includes('correo de acceso ya') ||
    normalized.includes('email ya') ||
    normalized.includes('email conflict') ||
    normalized.includes('correo ya')
  ) {
    return 'Ese correo de acceso ya está registrado. Iniciá sesión o usá otro correo.';
  }
  if (normalized === 'body inválido' || normalized === 'body invalido') {
    return 'No pudimos leer los datos enviados. Revisá el formulario e intentá nuevamente.';
  }
  if (normalized === 'email requerido') {
    return 'Ingresá un correo electrónico.';
  }
  if (normalized.includes('validar el correo')) {
    return 'No pudimos validar el correo de acceso. Intentá nuevamente.';
  }
  if (normalized.includes('tiempo de espera agotado')) {
    return 'El servidor tardó demasiado en responder. Intentá nuevamente.';
  }
  if (normalized.includes('correo o contraseña incorrectos')) {
    return 'Correo o contraseña incorrectos.';
  }
  if (normalized === 'error interno' || normalized.startsWith('error http 500')) {
    return 'Ocurrió un problema en el servidor. Intentá nuevamente en unos minutos.';
  }
  if (normalized.startsWith('error http 401') || normalized.includes('no autorizado')) {
    return 'Tu sesión no está activa. Volvé a iniciar sesión.';
  }
  if (normalized.startsWith('error http 403')) {
    return 'No tenés permisos para realizar esta acción.';
  }
  if (normalized.startsWith('error http 404')) {
    return 'No encontramos ese recurso en el servidor. Probá nuevamente en unos minutos.';
  }

  return message;
}

function parseJsonField(value: unknown) {
  if (typeof value !== 'string') return value;
  try {
    return JSON.parse(value);
  } catch {
    return value;
  }
}

// La sesión vive únicamente en la cookie HttpOnly emitida por la API. El
// frontend nunca lee ni persiste el JWT.
export async function cerrarSesion(): Promise<void> {
  await apiFetch<void>('/auth/logout', { method: 'POST' });
}

async function apiFetch<T>(
  endpoint: string,
  options: RequestInit = {},
  timeoutMs = 10000,
): Promise<T> {
  const baseUrl = getApiBaseUrl();
  const url = `${baseUrl}${endpoint}`;
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string>),
  };

  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), timeoutMs);

  try {
    const response = await fetch(url, {
      ...options,
      headers,
      credentials: 'include',
      signal: options.signal || controller.signal,
    });

    if (response.status === 204) {
      return null as unknown as T;
    }

    const contentType = response.headers.get('content-type');
    const isJson = contentType && contentType.includes('application/json');
    const data: unknown = isJson ? await response.json() : null;

    if (!response.ok) {
      const errorMsg = getApiErrorMessage(data, `Error HTTP ${response.status}`);
      throw new ApiError(errorMsg, response.status, data);
    }

    return data as T;
  } catch (err: unknown) {
    if (err instanceof Error && err.name === 'AbortError') {
      throw new ApiError('Tiempo de espera agotado al conectar con el servidor.', 408);
    }
    throw err;
  } finally {
    clearTimeout(timeoutId);
  }
}

// --- TIPOS ---

export type PlanTenant = 'free' | 'pro';

export interface EstadoPlan {
  plan: PlanTenant;
  plan_desde?: string | null;
  plan_hasta?: string | null;
  dias_restantes_pro?: number | null;
  en_prueba?: boolean;
  prueba_hasta?: string | null;
  dias_restantes_prueba?: number | null;
  precio_mensual?: number;
  moneda?: string;
  meses_prueba?: number;
  upgrade_solicitado_at?: string | null;
  upgrade_nota?: string | null;
  limites: Record<string, number>;
  uso: Record<string, number>;
  disponibles: Record<string, number>;
  alcanzado: Record<string, boolean>;
}

export interface Tenant {
  id: string;
  nombre: string;
  nombre_fantasia?: string;
  rubro?: string;
  descripcion?: string;
  email_contacto?: string;
  whatsapp?: string;
  logo_url?: string;
  color_primario?: string;
  estilo_visual?: string;
  datos_fiscales?: Record<string, unknown>;
  google_review_url?: string;
  mp_access_token?: string;
  mp_public_key?: string;
  mp_activo?: boolean;
  plan?: PlanTenant;
  plan_desde?: string | null;
  plan_hasta?: string | null;
  prueba_hasta?: string | null;
  upgrade_solicitado_at?: string | null;
  upgrade_nota?: string | null;
  mostrar_marca_agua?: boolean;
  color_secundario?: string;
  color_categoria?: string;
  color_accion?: string;
  tipo_fuente?: string;
  slug: string;
  activo?: boolean;
  created_at: string;
  updated_at?: string;
}

export interface ActualizarTenantInput {
  nombre?: string;
  nombre_fantasia?: string;
  rubro?: string;
  descripcion?: string;
  email_contacto?: string;
  whatsapp?: string;
  logo_url?: string;
  color_primario?: string;
  estilo_visual?: string;
  datos_fiscales?: Record<string, unknown>;
  google_review_url?: string;
  mp_access_token?: string;
  mp_public_key?: string;
  mp_activo?: boolean;
  mostrar_marca_agua?: boolean;
  color_secundario?: string;
  color_categoria?: string;
  color_accion?: string;
  tipo_fuente?: string;
}

export interface OnboardingInput {
  nombre: string;
  nombre_fantasia?: string;
  rubro?: string;
  slug: string;
  email_admin: string;
  nombre_admin: string;
  password?: string;
  // Alternativa a password: registro con la cuenta de Google del admin.
  google_credential?: string;
  sucursal_nombre?: string;
  email_sucursal?: string;
  direccion?: string;
  telefono?: string;
  whatsapp?: string;
  horarios?: unknown;
}

export interface Sucursal {
  id: string;
  tenant_id: string;
  nombre: string;
  direccion?: string;
  telefono?: string;
  email?: string;
  whatsapp?: string;
  horarios?: string;
  mp_access_token?: string | null;
  mp_public_key?: string | null;
  mp_activo?: boolean | null;
  activa: boolean;
  created_at: string;
  updated_at: string;
}

export interface Sector {
  id: string;
  sucursal_id: string;
  nombre: string;
  created_at: string;
}

export interface CategoriaAPI {
  id: string;
  tenant_id: string;
  nombre: string;
  orden: number;
  icono?: string;
  franja_horaria_id?: string;
  activa: boolean;
  created_at: string;
  updated_at: string;
}

export interface VariantePublica {
  id: string;
  articulo_id: string;
  nombre: string;
  precio_adicional: number;
  grupo?: string;
  seleccion_unica: boolean;
  orden: number;
}

export interface ArticuloAPI {
  id: string;
  tenant_id: string;
  categoria_id: string;
  nombre: string;
  descripcion: string;
  precio: number;
  foto_url?: string;
  activo: boolean;
  franja_horaria_id?: string;
  orden: number;
  created_at: string;
  updated_at: string;
  variantes?: VariantePublica[];
}

export type RedondeoAjustePrecios = 'ninguno' | '10' | '100';

export interface AjustePreciosInput {
  categoria_id?: string;
  porcentaje: number;
  redondeo: RedondeoAjustePrecios;
}

export interface AjustePreciosResultado {
  actualizados: number;
}

export interface MesaAPI {
  id: string;
  tenant_id: string;
  sucursal_id: string;
  sector_id?: string;
  numero: number;
  capacidad: number;
  qr_token: string;
  estado: 'activa' | 'inactiva';
  cuenta_solicitada: boolean;
  pago_habilitado: boolean;
  cuenta_version: number;
  created_at: string;
  updated_at: string;
}

export type EstadoUsuario = 'activo' | 'bloqueado' | 'archivado';

export interface UsuarioAPI {
  id: string;
  tenant_id: string;
  sucursal_id?: string;
  sucursal_nombre?: string;
  username?: string;
  nombre: string;
  email: string;
  rol: 'admin' | 'encargado' | 'mozo' | 'cocina';
  estado: EstadoUsuario;
  created_at: string;
  updated_at?: string;
}

export interface NegocioPublicoAuth {
  id: string;
  slug: string;
  nombre: string;
  nombre_fantasia?: string;
  rubro?: string;
  logo_url?: string | null;
  color_primario?: string | null;
  estilo_visual?: string | null;
}

export interface LoginStaffResponse {
  redirect_to?: string;
  usuario: SesionUsuario & { username?: string };
}

export interface SesionUsuario {
  id: string;
  tenant_id: string;
  email: string;
  nombre: string;
  rol: 'admin' | 'encargado' | 'mozo' | 'cocina';
  sucursal_id?: string | null;
  es_superadmin?: boolean;
}

export interface SolicitudMcp {
  id: string;
  client_name: string;
  dominio_retorno: string;
  negocio: string;
  permisos: string[];
  solo_lectura: boolean;
  mismo_navegador: boolean;
  expira_at: string;
}

export interface ConexionMcp {
  id: string;
  client_name: string;
  creado_at: string;
  ultimo_uso_at: string | null;
}

export interface InvitacionEquipoAPI {
  id: string;
  negocio: string;
  slug: string;
  sucursal_id: string;
  sucursal: string;
  email: string;
  rol: 'encargado' | 'mozo' | 'cocina';
  estado: 'pendiente' | 'cancelada' | 'usada' | 'reemplazada';
  expires_at: string;
  created_at: string;
}

export interface EquipoAPI {
  usuarios: UsuarioAPI[];
  invitaciones: InvitacionEquipoAPI[];
}

export interface MesaPublica {
  id: string;
  numero: number;
  sucursal_id: string;
  tenant_id: string;
  estado: 'activa' | 'inactiva';
  cuenta_solicitada: boolean;
  pago_habilitado: boolean;
  cuenta_version: number;
  nombre: string;
  logo_url?: string | null;
  color_primario?: string | null;
  estilo_visual?: 'claro' | 'oscuro' | null;
  mercadopago_habilitado?: boolean;
  plan?: PlanTenant;
  mostrar_marca_agua?: boolean;
  color_secundario?: string | null;
  color_categoria?: string | null;
  color_accion?: string | null;
  tipo_fuente?: string | null;
  google_review_url?: string | null;
  sector?: string;
  nombre_fantasia?: string;
}

export interface Resena {
  id: string;
  mesa_id: string;
  mesa_numero?: number;
  sucursal_id: string;
  sucursal_nombre?: string;
  cuenta_version: number;
  estrellas: number;
  comentario?: string | null;
  google_cliqueado: boolean;
  created_at: string;
}

export type ResenaItem = Resena;

export interface ResenaResumen {
  promedio: number;
  total: number;
  csat: number;
  distribucion: Record<number, number>;
  clics_google: number;
  resenas: Resena[];
}

export interface ArticuloPublico {
  id: string;
  nombre: string;
  descripcion?: string;
  precio: number;
  foto_url?: string;
  activo: boolean;
  variantes?: VariantePublica[];
}

export interface CategoriaPublica {
  id: string;
  nombre: string;
  orden: number;
  icono?: string;
  articulos: ArticuloPublico[];
  disponible: boolean;
  disponible_desde?: string;
}

export interface FranjaHorariaAPI {
  id: string;
  tenant_id: string;
  nombre: string;
  hora_inicio: string;
  hora_fin: string;
}

export interface CartaPublicaResponse {
  categorias: CategoriaPublica[];
}

export interface PedidoItemVarianteAPI {
  variante_id: string;
  nombre: string;
  precio_adicional: number;
}

export interface PedidoItemAPI {
  id: string;
  pedido_id: string;
  articulo_id: string;
  nombre_articulo?: string;
  cantidad: number;
  precio_unitario: number;
  notas?: string;
  comensal_id?: string;
  comensal_nombre?: string;
  estado?: 'pendiente' | 'preparando' | 'listo';
  variantes?: PedidoItemVarianteAPI[];
}

export interface PedidoAPI {
  id: string;
  mesa_id: string;
  sucursal_id: string;
  cuenta_version: number;
  estado: 'recibido' | 'preparando' | 'listo' | 'cerrado';
	origen?: 'comensal' | 'mozo';
	creado_por_usuario_id?: string;
	creado_por_nombre?: string;
  items?: PedidoItemAPI[];
  created_at: string;
  updated_at: string;
}

export interface PlatoEstrellaAPI {
  articulo_id: string;
  nombre: string;
  unidades: number;
  monto: number;
}

export interface MetricaDiariaAPI {
  fecha: string;
  facturacion_total: number;
  pedidos_totales: number;
  pedidos_cerrados: number;
}

export interface MetricaTurnoAPI {
  turno: string;
  facturacion_total: number;
  pedidos_totales: number;
  pedidos_cerrados: number;
}

export interface MetricaEstadoAPI {
  estado: 'recibido' | 'preparando' | 'listo' | 'cerrado';
  cantidad: number;
}

export interface MetricasResumenAPI {
  periodo: {
    desde: string;
    hasta: string;
    zona_horaria: string;
    sucursal_id?: string;
    comparado_desde: string;
    comparado_hasta: string;
  };
  facturacion_total: number;
  ticket_promedio: number;
  pedidos_totales: number;
  pedidos_cerrados: number;
  pedidos_activos: number;
  tiempo_promedio_despacho_minutos: number;
  plato_mas_vendido: PlatoEstrellaAPI | null;
  platos_estrella: PlatoEstrellaAPI[];
  variaciones: {
    facturacion_total: number | null;
    ticket_promedio: number | null;
    pedidos_totales: number | null;
    tiempo_promedio_despacho_minutos: number | null;
  };
  por_dia: MetricaDiariaAPI[];
  por_turno: MetricaTurnoAPI[];
  por_estado: MetricaEstadoAPI[];
}

export interface NuevoPedidoInput {
  mesa_id: string;
  items: Array<{
    articulo_id: string;
    cantidad: number;
    notas?: string;
    comensal_id: string;
    comensal_nombre: string;
    variantes?: string[];
  }>;
}

// --- API METHODS ---

export const api = {
  // Base URLs
  getBaseUrl: () => getApiBaseUrl(),

  // 1. Auth (US-38)
  // El backend busca al usuario por email, así que lo mandamos normalizado
  // (igual que como lo guarda el onboarding) para que siempre matchee.
  // `magic_link_dev` solo viene en entornos sin proveedor de email configurado.
  solicitarMagicLink: async (email: string) => {
    return apiFetch<{ mensaje: string; magic_link_dev?: string }>('/auth/magic-link', {
      method: 'POST',
      body: JSON.stringify({ email: email.trim().toLowerCase() }),
    });
  },

  autenticarConPassword: async (email: string, password: string) => {
		return apiFetch<{ usuario: SesionUsuario }>('/auth/password', {
      method: 'POST',
      body: JSON.stringify({ email: email.trim().toLowerCase(), password }),
    });
  },

  autenticarConGoogle: async (credential: string) => {
		return apiFetch<{ usuario: SesionUsuario }>('/auth/google', {
      method: 'POST',
      body: JSON.stringify({ credential }),
    });
  },

	 autenticarEquipoConGoogle: async (slug: string, credential: string) => {
		 return apiFetch<{ usuario: SesionUsuario }>(`/auth/negocios/${encodeURIComponent(slug)}/google`, {
			 method: 'POST',
			 body: JSON.stringify({ credential }),
		 });
	 },

  verificarToken: async (token: string) => {
		return apiFetch<{ usuario: SesionUsuario }>(`/auth/verify?token=${encodeURIComponent(token)}`);
  },

	 obtenerSesion: async () => apiFetch<SesionUsuario>('/auth/me'),

  // 2. Tenant / Onboarding (US-39)
  crearTenant: async (data: OnboardingInput) => {
    return apiFetch<Tenant>('/tenants', {
      method: 'POST',
      body: JSON.stringify({
        ...data,
        email_admin: data.email_admin.trim().toLowerCase(),
        horarios: parseJsonField(data.horarios),
      }),
    }, 45000);
  },

  verificarEmailAdminDisponible: async (email: string) => {
    return apiFetch<{ disponible: boolean }>(
      `/tenants/email-disponible?email=${encodeURIComponent(email.trim().toLowerCase())}`,
      {},
      30000,
    );
  },

  verificarSlugDisponible: async (slug: string) => {
    return apiFetch<{ disponible: boolean }>(
      `/tenants/slug-disponible?slug=${encodeURIComponent(slug.trim().toLowerCase())}`,
      {},
      30000,
    );
  },

  obtenerMiTenant: async () => {
    return apiFetch<Tenant>('/tenants/me');
  },

  actualizarMiTenant: async (data: ActualizarTenantInput) => {
    return apiFetch<Tenant>('/tenants/me', {
      method: 'PATCH',
      body: JSON.stringify(data),
    });
  },

  obtenerMiPlan: async () => {
    return apiFetch<EstadoPlan>('/tenants/me/plan');
  },

  solicitarUpgradePro: async (nota: string) => {
    return apiFetch<Tenant>('/tenants/me/upgrade', {
      method: 'POST',
      body: JSON.stringify({ nota }),
    });
  },

  // 3. Sucursales & Sectores
  listarSucursales: async () => {
    return apiFetch<Sucursal[]>('/sucursales');
  },

  crearSucursal: async (data: Partial<Sucursal>) => {
    return apiFetch<Sucursal>('/sucursales', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  actualizarSucursal: async (id: string, data: Partial<Sucursal>) => {
    return apiFetch<Sucursal>(`/sucursales/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    });
  },

  listarSectores: async (sucursalId: string) => {
    return apiFetch<Sector[]>(`/sucursales/${sucursalId}/sectores`);
  },

  crearSector: async (sucursalId: string, nombre: string) => {
    return apiFetch<Sector>(`/sucursales/${sucursalId}/sectores`, {
      method: 'POST',
      body: JSON.stringify({ nombre }),
    });
  },

  // 4. Carta (US-40)
  listarCategorias: async () => {
    return apiFetch<CategoriaAPI[]>('/carta/categorias');
  },

  crearCategoria: async (nombre: string, orden: number = 0) => {
    return apiFetch<CategoriaAPI>('/carta/categorias', {
      method: 'POST',
      body: JSON.stringify({ nombre, orden }),
    });
  },

  eliminarCategoria: async (id: string) => {
    return apiFetch<void>(`/carta/categorias/${id}`, {
      method: 'DELETE',
    });
  },

  asignarFranjaCategoria: async (id: string, franjaHorariaId: string | null) => {
    return apiFetch<CategoriaAPI>(`/carta/categorias/${id}/franja-horaria`, {
      method: 'PATCH',
      body: JSON.stringify({ franja_horaria_id: franjaHorariaId }),
    });
  },

  asignarIconoCategoria: async (id: string, icono: string | null) => {
    return apiFetch<CategoriaAPI>(`/carta/categorias/${id}/icono`, {
      method: 'PATCH',
      body: JSON.stringify({ icono }),
    });
  },

  listarArticulos: async () => {
    return apiFetch<ArticuloAPI[]>('/carta/articulos');
  },

  crearArticulo: async (data: {
    categoria_id: string;
    nombre: string;
    descripcion?: string;
    precio: number;
    foto_url?: string;
    activo?: boolean;
    orden?: number;
  }) => {
    return apiFetch<ArticuloAPI>('/carta/articulos', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  actualizarArticulo: async (id: string, data: Partial<ArticuloAPI>) => {
    return apiFetch<ArticuloAPI>(`/carta/articulos/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    });
  },

  asignarFranjaArticulo: async (id: string, franjaHorariaId: string | null) => {
    return apiFetch<ArticuloAPI>(`/carta/articulos/${id}/franja-horaria`, {
      method: 'PATCH',
      body: JSON.stringify({ franja_horaria_id: franjaHorariaId }),
    });
  },

  listarFranjasHorarias: async () => {
    return apiFetch<FranjaHorariaAPI[]>('/carta/franjas-horarias');
  },

  crearFranjaHoraria: async (data: { nombre: string; hora_inicio: string; hora_fin: string }) => {
    return apiFetch<FranjaHorariaAPI>('/carta/franjas-horarias', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  actualizarFranjaHoraria: async (id: string, data: { nombre: string; hora_inicio: string; hora_fin: string }) => {
    return apiFetch<FranjaHorariaAPI>(`/carta/franjas-horarias/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    });
  },

  eliminarFranjaHoraria: async (id: string) => {
    return apiFetch<void>(`/carta/franjas-horarias/${id}`, { method: 'DELETE' });
  },

  ajustarPrecios: async (data: AjustePreciosInput) => {
    return apiFetch<AjustePreciosResultado>('/carta/precios/ajuste-porcentual', {
      method: 'PATCH',
      body: JSON.stringify(data),
    });
  },

  eliminarArticulo: async (id: string) => {
    return apiFetch<void>(`/carta/articulos/${id}`, {
      method: 'DELETE',
    });
  },

  listarVariantes: async (articuloId: string) => {
    return apiFetch<VariantePublica[]>(`/carta/articulos/${articuloId}/variantes`);
  },

  crearVariante: async (articuloId: string, data: {
    nombre: string;
    precio_adicional: number;
    grupo?: string;
    seleccion_unica?: boolean;
    orden?: number;
  }) => {
    return apiFetch<VariantePublica>(`/carta/articulos/${articuloId}/variantes`, {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  actualizarVariante: async (id: string, data: {
    nombre?: string;
    precio_adicional?: number;
    grupo?: string;
    seleccion_unica?: boolean;
    orden?: number;
  }) => {
    return apiFetch<VariantePublica>(`/carta/variantes/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    });
  },

  eliminarVariante: async (id: string) => {
    return apiFetch<void>(`/carta/variantes/${id}`, {
      method: 'DELETE',
    });
  },

  // 5. Mesas & QR (US-40 & US-41)
  listarMesas: async () => {
    return apiFetch<MesaAPI[]>('/mesas');
  },

  crearMesa: async (data: {
    sucursal_id: string;
    sector_id?: string;
    numero: number;
    capacidad: number;
  }) => {
    return apiFetch<MesaAPI>('/mesas', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  actualizarMesa: async (id: string, data: Partial<MesaAPI>) => {
    return apiFetch<MesaAPI>(`/mesas/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    });
  },

  habilitarPagoMesa: async (id: string, habilitado = true) => {
    return apiFetch<MesaAPI>(`/mesas/${encodeURIComponent(id)}/habilitar-pago`, {
      method: 'POST',
      body: JSON.stringify({ habilitado }),
    });
  },

  cerrarCuenta: async (id: string) => {
    return apiFetch<MesaAPI>(`/mesas/${encodeURIComponent(id)}/cerrar-cuenta`, {
      method: 'POST',
    });
  },

  eliminarMesa: async (id: string) => {
    return apiFetch<void>(`/mesas/${id}`, {
      method: 'DELETE',
    });
  },

  // 6. Equipo / Usuarios (US-40)
  listarUsuarios: async () => {
		return apiFetch<EquipoAPI>('/usuarios');
  },

  invitarUsuario: async (data: {
    email: string;
		rol: 'encargado' | 'mozo' | 'cocina';
    sucursal_id?: string;
  }) => {
		return apiFetch<{ invitacion: InvitacionEquipoAPI; invitacion_link: string; email_enviado: boolean; advertencia_email?: string }>('/usuarios', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  eliminarUsuario: async (id: string) => {
    return apiFetch<void>(`/usuarios/${id}`, {
      method: 'DELETE',
    });
  },

  crearUsuarioStaff: async (data: {
    nombre: string;
    username: string;
    email?: string;
    rol: 'encargado' | 'mozo' | 'cocina';
    sucursal_id?: string;
    password: string;
  }) => {
    return apiFetch<UsuarioAPI>('/usuarios/staff', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  resetearPasswordUsuario: async (id: string, nuevaPassword: string) => {
    return apiFetch<{ ok: boolean }>(`/usuarios/${encodeURIComponent(id)}/password`, {
      method: 'PATCH',
      body: JSON.stringify({ nueva_password: nuevaPassword }),
    });
  },

  cambiarEstadoUsuario: async (id: string, estado: EstadoUsuario) => {
    return apiFetch<UsuarioAPI>(`/usuarios/${encodeURIComponent(id)}/estado`, {
      method: 'PATCH',
      body: JSON.stringify({ estado }),
    });
  },

  actualizarUsuario: async (
    id: string,
    data: {
      nombre?: string;
      username?: string;
      email?: string;
      rol?: 'admin' | 'encargado' | 'mozo' | 'cocina';
      sucursal_id?: string;
      estado?: EstadoUsuario;
    },
  ) => {
    return apiFetch<UsuarioAPI>(`/usuarios/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    });
  },

  obtenerNegocioPublico: async (slug: string) => {
    return apiFetch<NegocioPublicoAuth>(`/publica/negocio/${encodeURIComponent(slug.trim().toLowerCase())}`);
  },

  autenticarStaffNegocio: async (slug: string, username: string, password: string) => {
    const res = await apiFetch<LoginStaffResponse>(
      `/publica/negocio/${encodeURIComponent(slug.trim().toLowerCase())}/login`,
      {
        method: 'POST',
        body: JSON.stringify({
          username: username.trim().replace(/^@/, '').toLowerCase(),
          password,
        }),
      },
    );
    if (typeof window !== 'undefined' && res.usuario?.sucursal_id) {
      localStorage.setItem('kds_sucursal_id', res.usuario.sucursal_id);
    }
    return res;
  },

	cancelarInvitacion: async (id: string) => apiFetch<void>(`/usuarios/invitaciones/${id}`, { method: 'DELETE' }),
	reenviarInvitacion: async (id: string) => apiFetch<{ invitacion: InvitacionEquipoAPI; invitacion_link: string; email_enviado: boolean; advertencia_email?: string }>(`/usuarios/invitaciones/${id}/reenviar`, { method: 'POST' }),
	obtenerInvitacion: async (token: string) => apiFetch<InvitacionEquipoAPI>(`/publica/invitaciones/${encodeURIComponent(token)}`),
	aceptarInvitacion: async (token: string, credential: string, nombre: string) => apiFetch<{ usuario: SesionUsuario }>(`/publica/invitaciones/${encodeURIComponent(token)}/aceptar`, {
		method: 'POST',
		body: JSON.stringify({ credential, nombre }),
	}),

  // Asistentes de IA (MCP / OAuth)
  obtenerSolicitudMcp: async (id: string) => {
    return apiFetch<SolicitudMcp>(`/oauth/solicitudes/${encodeURIComponent(id)}`);
  },

  aprobarSolicitudMcp: async (id: string) => {
    return apiFetch<{ redirect_url: string }>(`/oauth/solicitudes/${encodeURIComponent(id)}/aprobar`, {
      method: 'POST',
    });
  },

  rechazarSolicitudMcp: async (id: string) => {
    return apiFetch<{ redirect_url: string }>(`/oauth/solicitudes/${encodeURIComponent(id)}/rechazar`, {
      method: 'POST',
    });
  },

  listarConexionesMcp: async () => {
    return apiFetch<ConexionMcp[]>('/oauth/conexiones');
  },

  revocarConexionMcp: async (id: string) => {
    return apiFetch<void>(`/oauth/conexiones/${encodeURIComponent(id)}`, { method: 'DELETE' });
  },

  // 7. Flujo Público Comensal (US-42, US-43, US-44)
  obtenerMesaPorQR: async (qrToken: string) => {
    return apiFetch<MesaPublica>(`/publica/mesas/${encodeURIComponent(qrToken)}`);
  },

  solicitarCuenta: async (qrToken: string) => {
    return apiFetch<MesaPublica>(`/publica/mesas/${encodeURIComponent(qrToken)}/cuenta`, {
      method: 'POST',
    });
  },

  obtenerCartaPublica: async (sucursalId: string) => {
    return apiFetch<CartaPublicaResponse>(`/publica/sucursales/${encodeURIComponent(sucursalId)}/carta`);
  },

  crearPedido: async (data: NuevoPedidoInput) => {
    return apiFetch<PedidoAPI>('/pedidos', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

	crearPedidoAsistido: async (data: { mesa_id: string; alias_cliente?: string; items: NuevoPedidoInput['items'] }) => apiFetch<PedidoAPI>('/pedidos/asistido', {
		method: 'POST',
		body: JSON.stringify(data),
	}),

  listarPedidosMesa: async (qrToken: string) => {
    return apiFetch<PedidoAPI[]>(`/publica/mesas/${encodeURIComponent(qrToken)}/pedidos`);
  },

  // 8. Recepcionista & SSE (US-44, US-45)
  listarPedidosActivos: async (sucursalId: string) => {
    return apiFetch<PedidoAPI[]>(`/pedidos?sucursal_id=${encodeURIComponent(sucursalId)}`);
  },

  cambiarEstadoPedido: async (pedidoId: string, estado: 'recibido' | 'preparando' | 'listo' | 'cerrado') => {
    return apiFetch<PedidoAPI>(`/pedidos/${encodeURIComponent(pedidoId)}/estado`, {
      method: 'PATCH',
      body: JSON.stringify({ estado }),
    });
  },

  cambiarEstadoItem: async (itemId: string, estado: 'pendiente' | 'preparando' | 'listo') => {
    return apiFetch<PedidoAPI>(`/pedidos/items/${encodeURIComponent(itemId)}/estado`, {
      method: 'PATCH',
      body: JSON.stringify({ estado }),
    });
  },

  obtenerEventosPedidoUrl: (pedidoId: string) => {
    return `${getApiBaseUrl()}/pedidos/${encodeURIComponent(pedidoId)}/eventos`;
  },

  obtenerEventosMesaUrl: (mesaId: string, tenantId?: string, sucursalId?: string) => {
    const params = new URLSearchParams();
    if (tenantId) params.set('tenant_id', tenantId);
    if (sucursalId) params.set('sucursal_id', sucursalId);
    const qs = params.toString();
    return `${getApiBaseUrl()}/publica/mesas/${encodeURIComponent(mesaId)}/eventos${qs ? `?${qs}` : ''}`;
  },

  obtenerEventosSucursalUrl: (sucursalId: string) => {
		return `${getApiBaseUrl()}/sucursales/${encodeURIComponent(sucursalId)}/eventos`;
  },

  obtenerEventosKDSUrl: (sucursalId: string) => {
		return `${getApiBaseUrl()}/kds/eventos?sucursal_id=${encodeURIComponent(sucursalId)}`;
  },

  // 9. Métricas y KPIs (US-72 / US-73)
  obtenerMetricasResumen: async (filtros?: { desde?: string; hasta?: string; sucursalId?: string }) => {
    const params = new URLSearchParams();
    if (filtros?.desde) params.set('desde', filtros.desde);
    if (filtros?.hasta) params.set('hasta', filtros.hasta);
    if (filtros?.sucursalId) params.set('sucursal_id', filtros.sucursalId);
    const query = params.toString();
    return apiFetch<MetricasResumenAPI>(`/metricas/resumen${query ? `?${query}` : ''}`, {}, 30000);
  },

  // 10. Carga masiva de catálogo (US-58 / US-59)
  importarCarta: async (archivo: File): Promise<ResultadoImportacion> => {
    const formData = new FormData();
    formData.append('archivo', archivo);

    const baseUrl = getApiBaseUrl();
    const headers: Record<string, string> = {};

    const res = await fetch(`${baseUrl}/carta/importar`, {
      method: 'POST',
      headers,
      body: formData,
      credentials: 'include',
    });

    const data = await res.json().catch(() => ({}));
    if (!res.ok) {
      const msg = (data as Record<string, string>).error || 'Error al importar la carta';
      throw new ApiError(msg, res.status, data);
    }
    return data as ResultadoImportacion;
  },

  // 11. Pagos con Mercado Pago (US-82)
  crearPreferenciaPagoMP: async (qrToken: string): Promise<PreferenciaPagoMP> => {
    return apiFetch<PreferenciaPagoMP>(`/publica/mesas/${encodeURIComponent(qrToken)}/pago/mercadopago`, {
      method: 'POST',
    });
  },

  confirmarPagoMP: async (qrToken: string, paymentId: string): Promise<RegistroPagoMP> => {
    return apiFetch<RegistroPagoMP>(`/publica/mesas/${encodeURIComponent(qrToken)}/pago/mercadopago/confirmar`, {
      method: 'POST',
      body: JSON.stringify({ payment_id: paymentId }),
    });
  },

  // 12. Reseñas y Reputación (US-79 / US-80)
  crearResena: async (qrToken: string, input: { estrellas: number; comentario?: string }) => {
    return apiFetch<Resena>(`/publica/mesas/${encodeURIComponent(qrToken)}/resena`, {
      method: 'POST',
      body: JSON.stringify(input),
    });
  },

  registrarClickGoogle: async (qrToken: string) => {
    return apiFetch<{ ok: boolean }>(`/publica/mesas/${encodeURIComponent(qrToken)}/resena/google-click`, {
      method: 'PATCH',
    });
  },

  obtenerResumenResenas: async (sucursalId?: string, desde?: string, hasta?: string) => {
    const params = new URLSearchParams();
    if (sucursalId) params.set('sucursal_id', sucursalId);
    if (desde) params.set('desde', desde);
    if (hasta) params.set('hasta', hasta);
    const query = params.toString();
    return apiFetch<ResenaResumen>(`/resenias/resumen${query ? `?${query}` : ''}`);
  },

  // 13. Telemetría y Superadministración de Plataforma (#62, #63)
  obtenerStatusPlataforma: async (): Promise<StatusPlataforma> => {
    return apiFetch<StatusPlataforma>('/api/platform/status');
  },

  listarTenantsPlataforma: async (): Promise<TenantResumenPlatform[]> => {
    return apiFetch<TenantResumenPlatform[]>('/api/platform/tenants');
  },

  obtenerDetalleActividadTenant: async (tenantId: string): Promise<TenantDetalleActividadPlatform> => {
    return apiFetch<TenantDetalleActividadPlatform>(`/api/platform/tenants/${encodeURIComponent(tenantId)}/actividad`);
  },
};

export interface PreferenciaPagoMP {
  preference_id: string;
  init_point: string;
  sandbox_init_point: string;
  monto_total: number;
}

export interface RegistroPagoMP {
  id: string;
  mesa_id: string;
  cuenta_version: number;
  proveedor: string;
  preferencia_id: string;
  pago_id: string;
  monto: number;
  moneda: string;
  estado: string;
  created_at: string;
}

export interface ResultadoImportacion {
  creados: number;
  omitidos: number;
  errores: ErrorFila[];
}

export interface ErrorFila {
  fila: number;
  motivo: string;
}
export interface StatusPlataforma {
  servicio: string;
  estado: string;
  database: string;
  timestamp: string;
  uptime_segundos: number;
  total_tenants: number;
  total_sucursales: number;
  total_mesas: number;
  total_usuarios: number;
  pedidos_hoy: number;
  total_ventas_hoy: number;
  usuarios_online: number;
  goroutines: number;
  heap_alloc_mb: number;
  sse_canales: number;
  sse_suscriptores: number;
  pool_stats?: {
    max_conns: number;
    total_conns: number;
    acquired_conns: number;
    idle_conns: number;
  };
}

export interface TenantResumenPlatform {
  id: string;
  nombre: string;
  nombre_fantasia?: string;
  slug: string;
  rubro?: string;
  plan: string;
  trial_hasta?: string;
  created_at: string;
  total_sucursales: number;
  total_mesas: number;
  mesas_activas: number;
  total_usuarios: number;
  usuarios_por_rol: Record<string, number>;
  total_pedidos: number;
  total_ventas: number;
  ultimo_pedido_at?: string;
}

export interface UsuarioPresenciaPlatform {
  id: string;
  tenant_id: string;
  sucursal_id?: string;
  sucursal_nombre?: string;
  nombre: string;
  email: string;
  username?: string;
  rol: string;
  estado: string;
  ultimo_login_at?: string;
  ultima_actividad_at?: string;
  estado_presencia: "online" | "inactivo" | "desconectado";
  created_at: string;
}

export interface MovimientoPlatform {
  id: string;
  tipo: "pedido" | "usuario" | "invitacion" | "resena" | string;
  descripcion: string;
  actor: string;
  fecha: string;
  detalle?: Record<string, any>;
}

export interface TenantDetalleActividadPlatform {
  tenant: TenantResumenPlatform;
  usuarios: UsuarioPresenciaPlatform[];
  ultimos_movimientos: MovimientoPlatform[];
}
