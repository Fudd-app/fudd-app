package platform

import "time"

type StatusPlataforma struct {
	Servicio         string         `json:"servicio"`
	Estado           string         `json:"estado"`
	Database         string         `json:"database"`
	Timestamp        string         `json:"timestamp"`
	UptimeSegundos   int64          `json:"uptime_segundos"`
	TotalTenants     int            `json:"total_tenants"`
	TotalSucursales  int            `json:"total_sucursales"`
	TotalMesas       int            `json:"total_mesas"`
	TotalUsuarios    int            `json:"total_usuarios"`
	PedidosHoy       int            `json:"pedidos_hoy"`
	TotalVentasHoy   float64        `json:"total_ventas_hoy"`
	UsuariosOnline   int            `json:"usuarios_online"`
	Goroutines       int            `json:"goroutines"`
	HeapAllocMB      float64        `json:"heap_alloc_mb"`
	SSECanales       int            `json:"sse_canales"`
	SSESuscriptores  int            `json:"sse_suscriptores"`
	PoolStats        map[string]any `json:"pool_stats,omitempty"`
}

type TenantResumen struct {
	ID              string         `json:"id"`
	Nombre          string         `json:"nombre"`
	NombreFantasia  string         `json:"nombre_fantasia,omitempty"`
	Slug            string         `json:"slug"`
	Rubro           string         `json:"rubro,omitempty"`
	Plan            string         `json:"plan"`
	TrialHasta      *time.Time     `json:"trial_hasta,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	TotalSucursales int            `json:"total_sucursales"`
	TotalMesas      int            `json:"total_mesas"`
	MesasActivas    int            `json:"mesas_activas"`
	TotalUsuarios   int            `json:"total_usuarios"`
	UsuariosPorRol  map[string]int `json:"usuarios_por_rol"`
	TotalPedidos    int            `json:"total_pedidos"`
	TotalVentas     float64        `json:"total_ventas"`
	UltimoPedidoAt  *time.Time     `json:"ultimo_pedido_at,omitempty"`
}

type UsuarioPresencia struct {
	ID             string     `json:"id"`
	TenantID       string     `json:"tenant_id"`
	SucursalID     *string    `json:"sucursal_id,omitempty"`
	SucursalNombre *string    `json:"sucursal_nombre,omitempty"`
	Nombre         string     `json:"nombre"`
	Email          string     `json:"email"`
	Username       *string    `json:"username,omitempty"`
	Rol            string     `json:"rol"`
	Estado         string     `json:"estado"`
	UltimoLoginAt  *time.Time `json:"ultimo_login_at,omitempty"`
	UltimaActividadAt *time.Time `json:"ultima_actividad_at,omitempty"`
	EstadoPresencia string     `json:"estado_presencia"` // "online", "inactivo", "desconectado"
	CreatedAt      time.Time  `json:"created_at"`
}

type Movimiento struct {
	ID          string         `json:"id"`
	Tipo        string         `json:"tipo"` // "pedido", "usuario", "invitacion", "resena"
	Descripcion string         `json:"descripcion"`
	Actor       string         `json:"actor"`
	Fecha       time.Time      `json:"fecha"`
	Detalle     map[string]any `json:"detalle,omitempty"`
}

type TenantDetalleActividad struct {
	Tenant             TenantResumen      `json:"tenant"`
	Usuarios           []UsuarioPresencia `json:"usuarios"`
	UltimosMovimientos []Movimiento       `json:"ultimos_movimientos"`
}
