package main

import (
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/carta"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/db"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/mercadopago"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/mesa"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/metrica"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/notificacion"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/pedido"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/platform"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/resenia"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/sucursal"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/tenant"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/usuario"
)

func esEntornoDesarrolloLocal(entorno, appURL string) bool {
	switch strings.ToLower(strings.TrimSpace(entorno)) {
	case "development", "local":
		return true
	case "":
		destino, err := url.Parse(strings.TrimSpace(appURL))
		if err != nil {
			return false
		}
		host := destino.Hostname()
		if strings.EqualFold(host, "localhost") {
			return true
		}
		ip := net.ParseIP(host)
		return ip != nil && ip.IsLoopback()
	default:
		return false
	}
}

func registrarRutas(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", handlerHealth)

	// Auth & Email Provider
	// proveedorReal indica si hay un canal de email de verdad configurado.
	// Si no lo hay, el magic link solo se escribe en el log. En desarrollo local
	// también se devuelve en la respuesta para poder probar el acceso rápidamente.
	proveedorReal := false
	var emailSender auth.EmailSender = &auth.LogEmailSender{}

	if brevoAPIKey := os.Getenv("BREVO_API_KEY"); brevoAPIKey != "" {
		senderEmail := os.Getenv("BREVO_FROM_EMAIL")
		if senderEmail == "" {
			senderEmail = os.Getenv("FROM_EMAIL")
		}
		senderName := os.Getenv("BREVO_SENDER_NAME")
		if senderName == "" {
			senderName = "fudd"
		}
		emailSender = auth.NuevoBrevoEmailSender(brevoAPIKey, senderName, senderEmail)
		proveedorReal = true
	} else if smtpHost := os.Getenv("SMTP_HOST"); smtpHost != "" {
		smtpPort := os.Getenv("SMTP_PORT")
		smtpUser := os.Getenv("SMTP_USER")
		smtpPass := os.Getenv("SMTP_PASS")
		if smtpPass == "" {
			smtpPass = os.Getenv("SMTP_PASSWORD")
		}
		smtpFrom := os.Getenv("SMTP_FROM")
		if smtpFrom == "" {
			smtpFrom = os.Getenv("FROM_EMAIL")
		}
		emailSender = auth.NuevoSMTPEmailSender(smtpHost, smtpPort, smtpUser, smtpPass, smtpFrom)
		proveedorReal = true
	} else if resendAPIKey := os.Getenv("RESEND_API_KEY"); resendAPIKey != "" {
		fromEmail := os.Getenv("FROM_EMAIL")
		if fromEmail == "" {
			fromEmail = "onboarding@resend.dev"
		}
		emailSender = auth.NuevoResendEmailSender(resendAPIKey, fromEmail)
		proveedorReal = true
	}

	entorno := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
	esProduccion := entorno == "production"
	esDesarrolloLocal := esEntornoDesarrolloLocal(entorno, os.Getenv("APP_URL"))
	if !proveedorReal {
		slog.Warn("sin proveedor de email configurado: el magic link solo se escribe en el log",
			"produccion", esProduccion)
	}

	auth.RegistrarHookActividad(platform.InstanciaTracker.RegistrarActividad)

	authStore := auth.NuevoStore()
	authSvc := auth.NuevoService(authStore, emailSender)
	// El acceso rápido se conserva siempre en desarrollo local, incluso si el
	// equipo tiene un proveedor de correo configurado. Nunca se expone en QA o producción.
	authH := auth.NuevosHandlers(authSvc, esDesarrolloLocal)
	mux.HandleFunc("POST /auth/password", authH.AutenticarPassword)
	mux.HandleFunc("POST /auth/google", authH.AutenticarGoogle)
	mux.HandleFunc("POST /auth/negocios/{slug}/google", authH.AutenticarStaffGoogle)
	mux.HandleFunc("POST /auth/negocios/{slug}/password", authH.AutenticarStaffNegocio)
	mux.HandleFunc("GET /publica/negocio/{slug}", authH.ObtenerNegocioPublico)
	mux.HandleFunc("POST /publica/negocio/{slug}/login", authH.AutenticarStaffNegocio)
	mux.HandleFunc("POST /auth/magic-link", authH.SolicitarLink)
	mux.HandleFunc("GET /auth/verify", authH.VerificarToken)
	requerirSesion := auth.RequerirSesion(authStore)
	conRoles := func(h http.HandlerFunc, roles ...string) http.Handler {
		return requerirSesion(auth.RequerirRoles(roles...)(h))
	}
	mux.Handle("GET /auth/me", requerirSesion(http.HandlerFunc(authH.MiSesion)))
	mux.Handle("POST /auth/logout", requerirSesion(http.HandlerFunc(authH.CerrarSesion)))

	// Tenant
	tenantStore := tenant.NuevoStore()
	tenantSvc := tenant.NuevoServiceConGoogle(tenantStore, authSvc)
	tenantH := tenant.NuevosHandlers(tenantSvc)
	mux.Handle("POST /tenants", http.HandlerFunc(tenantH.Crear))
	mux.Handle("GET /tenants/email-disponible", http.HandlerFunc(tenantH.EmailAdminDisponible))
	mux.Handle("GET /tenants/slug-disponible", http.HandlerFunc(tenantH.SlugDisponible))
	mux.Handle("GET /tenants/me", requerirSesion(http.HandlerFunc(tenantH.ObtenerMe)))
	mux.Handle("PATCH /tenants/me", conRoles(tenantH.ActualizarMe, auth.RolDuenio))
	mux.Handle("GET /tenants/me/plan", conRoles(tenantH.ObtenerMiPlan, auth.RolDuenio))
	mux.Handle("POST /tenants/me/upgrade", conRoles(tenantH.SolicitarUpgrade, auth.RolDuenio))

	protegido := func(rec tenant.Recurso, h http.HandlerFunc, roles ...string) http.Handler {
		return requerirSesion(auth.RequerirRoles(roles...)(tenant.RequerirCuota(tenantStore, rec)(h)))
	}

	// Sucursales y Sectores (admin — protegidas)
	sucursalStore := sucursal.NuevoStore()
	sucursalSvc := sucursal.NuevoService(sucursalStore)
	sucursalH := sucursal.NuevosHandlers(sucursalSvc)
	mux.Handle("GET /sucursales", requerirSesion(http.HandlerFunc(sucursalH.Listar)))
	mux.Handle("GET /sucursales/{id}", requerirSesion(http.HandlerFunc(sucursalH.ObtenerPorID)))
	mux.Handle("POST /sucursales", protegido(tenant.RecursoSucursales, sucursalH.Crear, auth.RolDuenio))
	mux.Handle("PATCH /sucursales/{id}", conRoles(sucursalH.Actualizar, auth.RolDuenio, auth.RolEncargado))
	mux.Handle("DELETE /sucursales/{id}", conRoles(sucursalH.Eliminar, auth.RolDuenio))

	mux.Handle("POST /sucursales/{sucursal_id}/sectores", conRoles(sucursalH.CrearSector, auth.RolDuenio, auth.RolEncargado))
	mux.Handle("GET /sucursales/{sucursal_id}/sectores", requerirSesion(http.HandlerFunc(sucursalH.ListarSectores)))
	mux.Handle("DELETE /sectores/{id}", conRoles(sucursalH.EliminarSector, auth.RolDuenio, auth.RolEncargado))

	// Equipo / Usuarios (admin — protegidas)
	usuarioStore := usuario.NuevoStore()
	usuarioSvc := usuario.NuevoService(usuarioStore, authSvc, emailSender)
	usuarioH := usuario.NuevosHandlers(usuarioSvc)
	mux.Handle("GET /usuarios", conRoles(usuarioH.Listar, auth.RolDuenio, auth.RolEncargado))
	mux.Handle("POST /usuarios", conRoles(usuarioH.Invitar, auth.RolDuenio, auth.RolEncargado))
	mux.Handle("POST /usuarios/staff", conRoles(usuarioH.CrearStaff, auth.RolDuenio, auth.RolEncargado))
	mux.Handle("PATCH /usuarios/{id}", conRoles(usuarioH.Actualizar, auth.RolDuenio, auth.RolEncargado))
	mux.Handle("PATCH /usuarios/{id}/password", conRoles(usuarioH.ResetearPassword, auth.RolDuenio, auth.RolEncargado))
	mux.Handle("PATCH /usuarios/{id}/estado", conRoles(usuarioH.CambiarEstado, auth.RolDuenio, auth.RolEncargado))
	mux.Handle("DELETE /usuarios/{id}", conRoles(usuarioH.Eliminar, auth.RolDuenio, auth.RolEncargado))
	mux.Handle("POST /usuarios/invitaciones/{id}/reenviar", conRoles(usuarioH.Reenviar, auth.RolDuenio, auth.RolEncargado))
	mux.Handle("DELETE /usuarios/invitaciones/{id}", conRoles(usuarioH.Cancelar, auth.RolDuenio, auth.RolEncargado))
	mux.HandleFunc("GET /publica/invitaciones/{token}", usuarioH.VistaPrevia)
	mux.HandleFunc("POST /publica/invitaciones/{token}/aceptar", usuarioH.Aceptar)

	// Carta (admin — protegida)
	cartaStore := carta.NuevoStore()
	cartaSvc := carta.NuevoService(cartaStore)
	cartaH := carta.NuevosHandlers(cartaSvc)
	mux.Handle("GET /carta/categorias", requerirSesion(http.HandlerFunc(cartaH.ListarCategorias)))
	mux.Handle("POST /carta/categorias", conRoles(cartaH.CrearCategoria, auth.RolDuenio, auth.RolEncargado, auth.RolMozo))
	mux.Handle("PATCH /carta/categorias/{id}/franja-horaria", conRoles(cartaH.AsignarFranjaCategoria, auth.RolDuenio, auth.RolEncargado, auth.RolMozo))
	mux.Handle("PATCH /carta/categorias/{id}/icono", protegido(tenant.RecursoPersonalizacion, cartaH.AsignarIconoCategoria, auth.RolDuenio, auth.RolEncargado, auth.RolMozo))
	mux.Handle("DELETE /carta/categorias/{id}", conRoles(cartaH.EliminarCategoria, auth.RolDuenio, auth.RolEncargado, auth.RolMozo))
	mux.Handle("GET /carta/articulos", requerirSesion(http.HandlerFunc(cartaH.ListarArticulos)))
	mux.Handle("POST /carta/articulos", protegido(tenant.RecursoProductos, cartaH.CrearArticulo, auth.RolDuenio, auth.RolEncargado, auth.RolMozo))
	mux.Handle("PATCH /carta/articulos/{id}", conRoles(cartaH.ActualizarArticulo, auth.RolDuenio, auth.RolEncargado, auth.RolMozo))
	mux.Handle("PATCH /carta/articulos/{id}/franja-horaria", conRoles(cartaH.AsignarFranjaArticulo, auth.RolDuenio, auth.RolEncargado, auth.RolMozo))
	mux.Handle("PATCH /carta/precios/ajuste-porcentual", conRoles(cartaH.AjustarPrecios, auth.RolDuenio, auth.RolEncargado, auth.RolMozo))
	mux.Handle("GET /carta/franjas-horarias", requerirSesion(http.HandlerFunc(cartaH.ListarFranjasHorarias)))
	mux.Handle("POST /carta/franjas-horarias", conRoles(cartaH.CrearFranjaHoraria, auth.RolDuenio, auth.RolEncargado, auth.RolMozo))
	mux.Handle("PATCH /carta/franjas-horarias/{id}", conRoles(cartaH.ActualizarFranjaHoraria, auth.RolDuenio, auth.RolEncargado, auth.RolMozo))
	mux.Handle("DELETE /carta/franjas-horarias/{id}", conRoles(cartaH.EliminarFranjaHoraria, auth.RolDuenio, auth.RolEncargado, auth.RolMozo))
	mux.Handle("GET /carta/articulos/{id}/variantes", requerirSesion(http.HandlerFunc(cartaH.ListarVariantes)))
	mux.Handle("POST /carta/articulos/{id}/variantes", conRoles(cartaH.CrearVariante, auth.RolDuenio, auth.RolEncargado, auth.RolMozo))
	mux.Handle("PATCH /carta/variantes/{id}", conRoles(cartaH.ActualizarVariante, auth.RolDuenio, auth.RolEncargado, auth.RolMozo))
	mux.Handle("DELETE /carta/variantes/{id}", conRoles(cartaH.EliminarVariante, auth.RolDuenio, auth.RolEncargado, auth.RolMozo))
	mux.Handle("DELETE /carta/articulos/{id}", conRoles(cartaH.EliminarArticulo, auth.RolDuenio, auth.RolEncargado, auth.RolMozo))
	mux.Handle("PATCH /carta/articulos/{id}/disponibilidad", conRoles(cartaH.ActualizarDisponibilidad, auth.RolDuenio, auth.RolEncargado, auth.RolMozo))
	mux.Handle("POST /carta/reponer-todos", conRoles(cartaH.ReponerTodos, auth.RolDuenio, auth.RolEncargado, auth.RolMozo))
	mux.Handle("POST /carta/importar", protegido(tenant.RecursoCargaMasiva, cartaH.ImportarCarta, auth.RolDuenio, auth.RolEncargado, auth.RolMozo))

	// Mesas (admin — protegidas)
	mesaStore := mesa.NuevoStore()
	mesaSvc := mesa.NuevoService(mesaStore)
	mesaH := mesa.NuevosHandlers(mesaSvc)
	mux.Handle("GET /mesas", conRoles(mesaH.Listar, auth.RolDuenio, auth.RolEncargado, auth.RolMozo, auth.RolCocinero))
	mux.Handle("POST /mesas", protegido(tenant.RecursoMesas, mesaH.Crear, auth.RolDuenio, auth.RolEncargado))
	mux.Handle("PATCH /mesas/{id}", conRoles(mesaH.Actualizar, auth.RolDuenio, auth.RolEncargado))
	mux.Handle("POST /mesas/{id}/habilitar-pago", conRoles(mesaH.HabilitarPago, auth.RolDuenio, auth.RolEncargado, auth.RolMozo))
	mux.Handle("POST /mesas/{id}/cerrar-cuenta", conRoles(mesaH.CerrarCuenta, auth.RolDuenio, auth.RolEncargado, auth.RolMozo))
	// Alias temporal para clientes anteriores: ahora cierra la cuenta sin inhabilitar la mesa.
	mux.Handle("POST /mesas/{id}/cerrar", conRoles(mesaH.CerrarCuenta, auth.RolDuenio, auth.RolEncargado, auth.RolMozo))
	mux.Handle("DELETE /mesas/{id}", conRoles(mesaH.Eliminar, auth.RolDuenio, auth.RolEncargado))

	// Pedidos
	pedidoStore := pedido.NuevoStore()
	pedidoSvc := pedido.NuevoService(pedidoStore)
	pedidoH := pedido.NuevosHandlers(pedidoSvc)
	mux.HandleFunc("POST /pedidos", pedidoH.Crear)
	mux.Handle("POST /pedidos/asistido", conRoles(pedidoH.CrearAsistido, auth.RolDuenio, auth.RolEncargado, auth.RolMozo))
	mux.Handle("GET /pedidos", conRoles(pedidoH.ListarActivos, auth.RolDuenio, auth.RolEncargado, auth.RolMozo))
	mux.Handle("PATCH /pedidos/{id}/estado", conRoles(pedidoH.CambiarEstado, auth.RolDuenio, auth.RolEncargado, auth.RolMozo))
	mux.Handle("PATCH /pedidos/items/{id}/estado", conRoles(pedidoH.CambiarEstadoItem, auth.RolDuenio, auth.RolEncargado, auth.RolCocinero))
	mux.Handle("GET /kds/eventos", conRoles(pedidoH.EventosKDS, auth.RolDuenio, auth.RolEncargado, auth.RolCocinero))

	// Métricas operativas (admin — protegidas y aisladas por tenant)
	metricaStore := metrica.NuevoStore()
	metricaSvc := metrica.NuevoService(metricaStore)
	metricaH := metrica.NuevosHandlers(metricaSvc)
	mux.Handle("GET /metricas/resumen", conRoles(metricaH.Resumen, auth.RolDuenio, auth.RolEncargado))

	// Reseñas y reputación (admin — protegidas y comensal — públicas)
	reseniaStore := resenia.NuevoStore()
	reseniaSvc := resenia.NuevoService(reseniaStore)
	reseniaH := resenia.NuevosHandlers(reseniaSvc)
	mux.Handle("GET /resenias/resumen", conRoles(reseniaH.Resumen, auth.RolDuenio, auth.RolEncargado))

	// Notificaciones / SSE
	notificacionH := notificacion.NuevosHandlers()
	mux.Handle("GET /sucursales/{sucursal_id}/eventos", conRoles(notificacionH.EventosSucursal, auth.RolDuenio, auth.RolEncargado, auth.RolMozo))
	mux.HandleFunc("GET /pedidos/{id}/eventos", notificacionH.EventosPedido)
	mux.HandleFunc("GET /publica/mesas/{mesa_id}/eventos", notificacionH.EventosMesa)

	// Públicos (sin auth — cliente con QR)
	mux.HandleFunc("GET /publica/sucursales/{sucursal_id}/carta", cartaH.CartaPublica)
	mux.HandleFunc("GET /publica/mesas/{qr_token}", mesaH.MesaPorQR)
	mux.HandleFunc("GET /publica/mesas/{qr_token}/pedidos", pedidoH.ListarCuentaActual)
	mux.HandleFunc("POST /publica/mesas/{qr_token}/cuenta", mesaH.SolicitarCuenta)
	mux.HandleFunc("POST /publica/mesas/{qr_token}/resena", reseniaH.Crear)
	mux.HandleFunc("PATCH /publica/mesas/{qr_token}/resena/google-click", reseniaH.RegistrarClickGoogle)

	// Mercado Pago (Sandbox / Producción)
	mpAccessToken := os.Getenv("MERCADOPAGO_ACCESS_TOKEN")
	appURL := os.Getenv("APP_URL")
	apiURL := os.Getenv("API_URL")
	mpClient := mercadopago.NuevoCliente(mpAccessToken)
	mpStore := mercadopago.NuevoStore()
	mpSvc := mercadopago.NuevoService(mpClient, mpStore, mesaSvc, pedidoSvc, appURL, apiURL)
	mpH := mercadopago.NuevosHandlers(mpSvc)

	mux.HandleFunc("POST /publica/mesas/{qr_token}/pago/mercadopago", mpH.CrearPreferencia)
	mux.HandleFunc("POST /publica/mesas/{qr_token}/pago/mercadopago/confirmar", mpH.ConfirmarPago)
	mux.HandleFunc("POST /publica/pago/mercadopago/webhook", mpH.Webhook)

	// Superadmin / Telemetría y Control de Plataforma
	platformStore := platform.NuevoStore()
	platformSvc := platform.NuevoService(platformStore, platform.InstanciaTracker)
	platformH := platform.NuevosHandlers(platformSvc)
	requerirSuper := auth.RequerirSuperadmin(authStore)

	mux.Handle("GET /api/platform/status", requerirSuper(http.HandlerFunc(platformH.Status)))
	mux.Handle("GET /api/platform/tenants", requerirSuper(http.HandlerFunc(platformH.ListarTenants)))
	mux.Handle("GET /api/platform/tenants/{id}/actividad", requerirSuper(http.HandlerFunc(platformH.DetalleActividad)))
}

func handlerHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	dbEstado := "ok"
	status := http.StatusOK

	var poolStats map[string]any
	if db.Pool != nil {
		if err := db.Pool.Ping(r.Context()); err != nil {
			dbEstado = "error: " + err.Error()
			status = http.StatusServiceUnavailable
		}
		st := db.Pool.Stat()
		poolStats = map[string]any{
			"max_conns":           st.MaxConns(),
			"total_conns":         st.TotalConns(),
			"acquired_conns":      st.AcquiredConns(),
			"idle_conns":          st.IdleConns(),
			"empty_acquire_count": st.EmptyAcquireCount(),
		}
	} else {
		dbEstado = "desconectado"
	}

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	canalesSSE, suscriptoresSSE := notificacion.Instancia.Estadisticas()

	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"servicio":      "mesa-click-api",
		"estado":        "ok",
		"database":      dbEstado,
		"timestamp":     time.Now().Format(time.RFC3339),
		"goroutines":    runtime.NumGoroutine(),
		"heap_alloc_mb": float64(mem.HeapAlloc) / (1024 * 1024),
		"sys_mb":        float64(mem.Sys) / (1024 * 1024),
		"sse": map[string]any{
			"canales":      canalesSSE,
			"suscriptores": suscriptoresSSE,
		},
		"db_pool": poolStats,
	})
}
