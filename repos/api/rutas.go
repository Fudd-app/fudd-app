package main

import (
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
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
			senderName = "Mesa CLICK"
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

	authStore := auth.NuevoStore()
	authSvc := auth.NuevoService(authStore, emailSender)
	// El acceso rápido se conserva siempre en desarrollo local, incluso si el
	// equipo tiene un proveedor de correo configurado. Nunca se expone en QA o producción.
	authH := auth.NuevosHandlers(authSvc, esDesarrolloLocal)
	mux.HandleFunc("POST /auth/password", authH.AutenticarPassword)
	mux.HandleFunc("POST /auth/google", authH.AutenticarGoogle)
	mux.HandleFunc("POST /auth/magic-link", authH.SolicitarLink)
	mux.HandleFunc("GET /auth/verify", authH.VerificarToken)
	mux.HandleFunc("GET /publica/negocio/{slug}", authH.ObtenerNegocioPublico)
	mux.HandleFunc("POST /publica/negocio/{slug}/login", authH.AutenticarStaffNegocio)

	// Tenant
	tenantStore := tenant.NuevoStore()
	tenantSvc := tenant.NuevoServiceConGoogle(tenantStore, authSvc)
	tenantH := tenant.NuevosHandlers(tenantSvc)
	mux.Handle("POST /tenants", http.HandlerFunc(tenantH.Crear))
	mux.Handle("GET /tenants/email-disponible", http.HandlerFunc(tenantH.EmailAdminDisponible))
	mux.Handle("GET /tenants/slug-disponible", http.HandlerFunc(tenantH.SlugDisponible))
	mux.Handle("GET /tenants/me", auth.Requerir(http.HandlerFunc(tenantH.ObtenerMe)))
	mux.Handle("PATCH /tenants/me", auth.Requerir(http.HandlerFunc(tenantH.ActualizarMe)))
	mux.Handle("GET /tenants/me/plan", auth.Requerir(http.HandlerFunc(tenantH.ObtenerMiPlan)))
	mux.Handle("POST /tenants/me/upgrade", auth.Requerir(http.HandlerFunc(tenantH.SolicitarUpgrade)))

	protegido := func(rec tenant.Recurso, h http.HandlerFunc) http.Handler {
		return auth.Requerir(tenant.RequerirCuota(tenantStore, rec)(h))
	}

	// Sucursales y Sectores (admin — protegidas)
	sucursalStore := sucursal.NuevoStore()
	sucursalSvc := sucursal.NuevoService(sucursalStore)
	sucursalH := sucursal.NuevosHandlers(sucursalSvc)
	mux.Handle("GET /sucursales", auth.Requerir(http.HandlerFunc(sucursalH.Listar)))
	mux.Handle("GET /sucursales/{id}", auth.Requerir(http.HandlerFunc(sucursalH.ObtenerPorID)))
	mux.Handle("POST /sucursales", protegido(tenant.RecursoSucursales, sucursalH.Crear))
	mux.Handle("PATCH /sucursales/{id}", auth.Requerir(http.HandlerFunc(sucursalH.Actualizar)))
	mux.Handle("DELETE /sucursales/{id}", auth.Requerir(http.HandlerFunc(sucursalH.Eliminar)))

	mux.Handle("POST /sucursales/{sucursal_id}/sectores", auth.Requerir(http.HandlerFunc(sucursalH.CrearSector)))
	mux.Handle("GET /sucursales/{sucursal_id}/sectores", auth.Requerir(http.HandlerFunc(sucursalH.ListarSectores)))
	mux.Handle("DELETE /sectores/{id}", auth.Requerir(http.HandlerFunc(sucursalH.EliminarSector)))

	// Equipo / Usuarios (admin — protegidas)
	usuarioStore := usuario.NuevoStore()
	usuarioSvc := usuario.NuevoService(usuarioStore)
	usuarioH := usuario.NuevosHandlers(usuarioSvc)
	mux.Handle("GET /usuarios", auth.Requerir(http.HandlerFunc(usuarioH.Listar)))
	mux.Handle("POST /usuarios", auth.Requerir(http.HandlerFunc(usuarioH.Invitar)))
	mux.Handle("POST /usuarios/staff", auth.Requerir(http.HandlerFunc(usuarioH.CrearStaff)))
	mux.Handle("PATCH /usuarios/{id}", auth.Requerir(http.HandlerFunc(usuarioH.Actualizar)))
	mux.Handle("PATCH /usuarios/{id}/password", auth.Requerir(http.HandlerFunc(usuarioH.ResetearPassword)))
	mux.Handle("PATCH /usuarios/{id}/estado", auth.Requerir(http.HandlerFunc(usuarioH.CambiarEstado)))
	mux.Handle("DELETE /usuarios/{id}", auth.Requerir(http.HandlerFunc(usuarioH.Eliminar)))

	// Carta (admin — protegida)
	cartaStore := carta.NuevoStore()
	cartaSvc := carta.NuevoService(cartaStore)
	cartaH := carta.NuevosHandlers(cartaSvc)
	mux.Handle("GET /carta/categorias", auth.Requerir(http.HandlerFunc(cartaH.ListarCategorias)))
	mux.Handle("POST /carta/categorias", auth.Requerir(http.HandlerFunc(cartaH.CrearCategoria)))
	mux.Handle("PATCH /carta/categorias/{id}/franja-horaria", auth.Requerir(http.HandlerFunc(cartaH.AsignarFranjaCategoria)))
	mux.Handle("PATCH /carta/categorias/{id}/icono", protegido(tenant.RecursoPersonalizacion, cartaH.AsignarIconoCategoria))
	mux.Handle("DELETE /carta/categorias/{id}", auth.Requerir(http.HandlerFunc(cartaH.EliminarCategoria)))
	mux.Handle("GET /carta/articulos", auth.Requerir(http.HandlerFunc(cartaH.ListarArticulos)))
	mux.Handle("POST /carta/articulos", protegido(tenant.RecursoProductos, cartaH.CrearArticulo))
	mux.Handle("PATCH /carta/articulos/{id}", auth.Requerir(http.HandlerFunc(cartaH.ActualizarArticulo)))
	mux.Handle("PATCH /carta/articulos/{id}/franja-horaria", auth.Requerir(http.HandlerFunc(cartaH.AsignarFranjaArticulo)))
	mux.Handle("PATCH /carta/precios/ajuste-porcentual", auth.Requerir(http.HandlerFunc(cartaH.AjustarPrecios)))
	mux.Handle("GET /carta/franjas-horarias", auth.Requerir(http.HandlerFunc(cartaH.ListarFranjasHorarias)))
	mux.Handle("POST /carta/franjas-horarias", auth.Requerir(http.HandlerFunc(cartaH.CrearFranjaHoraria)))
	mux.Handle("PATCH /carta/franjas-horarias/{id}", auth.Requerir(http.HandlerFunc(cartaH.ActualizarFranjaHoraria)))
	mux.Handle("DELETE /carta/franjas-horarias/{id}", auth.Requerir(http.HandlerFunc(cartaH.EliminarFranjaHoraria)))
	mux.Handle("GET /carta/articulos/{id}/variantes", auth.Requerir(http.HandlerFunc(cartaH.ListarVariantes)))
	mux.Handle("POST /carta/articulos/{id}/variantes", auth.Requerir(http.HandlerFunc(cartaH.CrearVariante)))
	mux.Handle("PATCH /carta/variantes/{id}", auth.Requerir(http.HandlerFunc(cartaH.ActualizarVariante)))
	mux.Handle("DELETE /carta/variantes/{id}", auth.Requerir(http.HandlerFunc(cartaH.EliminarVariante)))
	mux.Handle("DELETE /carta/articulos/{id}", auth.Requerir(http.HandlerFunc(cartaH.EliminarArticulo)))
	mux.Handle("PATCH /carta/articulos/{id}/disponibilidad", auth.Requerir(http.HandlerFunc(cartaH.ActualizarDisponibilidad)))
	mux.Handle("POST /carta/reponer-todos", auth.Requerir(http.HandlerFunc(cartaH.ReponerTodos)))
	mux.Handle("POST /carta/importar", protegido(tenant.RecursoCargaMasiva, cartaH.ImportarCarta))

	// Mesas (admin — protegidas)
	mesaStore := mesa.NuevoStore()
	mesaSvc := mesa.NuevoService(mesaStore)
	mesaH := mesa.NuevosHandlers(mesaSvc)
	mux.Handle("GET /mesas", auth.Requerir(http.HandlerFunc(mesaH.Listar)))
	mux.Handle("POST /mesas", protegido(tenant.RecursoMesas, mesaH.Crear))
	mux.Handle("PATCH /mesas/{id}", auth.Requerir(http.HandlerFunc(mesaH.Actualizar)))
	mux.Handle("POST /mesas/{id}/habilitar-pago", auth.Requerir(http.HandlerFunc(mesaH.HabilitarPago)))
	mux.Handle("POST /mesas/{id}/cerrar-cuenta", auth.Requerir(http.HandlerFunc(mesaH.CerrarCuenta)))
	// Alias temporal para clientes anteriores: ahora cierra la cuenta sin inhabilitar la mesa.
	mux.Handle("POST /mesas/{id}/cerrar", auth.Requerir(http.HandlerFunc(mesaH.CerrarCuenta)))
	mux.Handle("DELETE /mesas/{id}", auth.Requerir(http.HandlerFunc(mesaH.Eliminar)))

	// Pedidos
	pedidoStore := pedido.NuevoStore()
	pedidoSvc := pedido.NuevoService(pedidoStore)
	pedidoH := pedido.NuevosHandlers(pedidoSvc)
	mux.HandleFunc("POST /pedidos", pedidoH.Crear)
	mux.Handle("GET /pedidos", auth.Requerir(http.HandlerFunc(pedidoH.ListarActivos)))
	mux.Handle("PATCH /pedidos/{id}/estado", auth.Requerir(http.HandlerFunc(pedidoH.CambiarEstado)))
	mux.Handle("PATCH /pedidos/items/{id}/estado", auth.Requerir(http.HandlerFunc(pedidoH.CambiarEstadoItem)))
	mux.Handle("GET /kds/eventos", auth.Requerir(http.HandlerFunc(pedidoH.EventosKDS)))

	// Métricas operativas (admin — protegidas y aisladas por tenant)
	metricaStore := metrica.NuevoStore()
	metricaSvc := metrica.NuevoService(metricaStore)
	metricaH := metrica.NuevosHandlers(metricaSvc)
	mux.Handle("GET /metricas/resumen", auth.Requerir(http.HandlerFunc(metricaH.Resumen)))

	// Reseñas y reputación (admin — protegidas y comensal — públicas)
	reseniaStore := resenia.NuevoStore()
	reseniaSvc := resenia.NuevoService(reseniaStore)
	reseniaH := resenia.NuevosHandlers(reseniaSvc)
	mux.Handle("GET /resenias/resumen", auth.Requerir(http.HandlerFunc(reseniaH.Resumen)))

	// Notificaciones / SSE
	notificacionH := notificacion.NuevosHandlers()
	mux.Handle("GET /sucursales/{sucursal_id}/eventos", auth.Requerir(http.HandlerFunc(notificacionH.EventosSucursal)))
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
}

func handlerHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	dbEstado := "ok"
	status := http.StatusOK

	if db.Pool != nil {
		if err := db.Pool.Ping(r.Context()); err != nil {
			dbEstado = "error: " + err.Error()
			status = http.StatusServiceUnavailable
		}
	} else {
		dbEstado = "desconectado"
	}

	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"servicio":  "mesa-click-api",
		"estado":    "ok",
		"database":  dbEstado,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}
