package mercadopago

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/evento"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/mesa"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/notificacion"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/pedido"
)

type ClientFactory func(accessToken string) Client

type Service struct {
	client        Client
	clientFactory ClientFactory
	store         Store
	mesaSvc       *mesa.Service
	pedidoSvc     *pedido.Service
	appURL        string
	apiURL        string
}

func NuevoService(client Client, store Store, mesaSvc *mesa.Service, pedidoSvc *pedido.Service, appURL, apiURL string) *Service {
	if appURL == "" {
		appURL = "http://localhost:3000"
	}
	return &Service{
		client:        client,
		clientFactory: func(token string) Client { return NuevoCliente(token) },
		store:         store,
		mesaSvc:       mesaSvc,
		pedidoSvc:     pedidoSvc,
		appURL:        strings.TrimRight(appURL, "/"),
		apiURL:        strings.TrimRight(apiURL, "/"),
	}
}

func (s *Service) SetClientFactory(factory ClientFactory) {
	s.clientFactory = factory
}

func (s *Service) resolverClient(ctx context.Context, mesaID string) (Client, error) {
	token, activo, err := s.store.ObtenerCredencialesMesa(ctx, mesaID)
	if err != nil {
		return nil, err
	}
	if !activo {
		return nil, ErrMercadoPagoInactivo
	}
	if token != "" {
		if s.clientFactory != nil {
			return s.clientFactory(token), nil
		}
		return NuevoCliente(token), nil
	}
	if s.client != nil {
		return s.client, nil
	}
	return nil, ErrTokenNoConfigurado
}

func (s *Service) CrearPreferenciaMesa(ctx context.Context, qrToken string) (*PreferenciaDTO, error) {
	mp, err := s.mesaSvc.ObtenerPorQRToken(ctx, qrToken)
	if err != nil {
		return nil, err
	}
	if mp.Estado != "activa" {
		return nil, ErrMesaNoEncontrada
	}
	if !mp.PagoHabilitado {
		return nil, ErrPagoNoHabilitado
	}

	client, err := s.resolverClient(ctx, mp.ID)
	if err != nil {
		return nil, err
	}

	pedidos, err := s.pedidoSvc.ListarCuentaActualPorQR(ctx, qrToken)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo pedidos de la mesa: %w", err)
	}

	var total float64
	items := make([]PreferenciaItem, 0)
	for _, p := range pedidos {
		for _, item := range p.Items {
			subtotal := item.PrecioUnitario * float64(item.Cantidad)
			total += subtotal

			nombre := strings.TrimSpace(item.NombreArticulo)
			if nombre == "" {
				nombre = "Consumo"
			}
			items = append(items, PreferenciaItem{
				Title:      fmt.Sprintf("%s (x%d)", nombre, item.Cantidad),
				Quantity:   item.Cantidad,
				UnitPrice:  item.PrecioUnitario,
				CurrencyID: "ARS",
			})
		}
	}

	if total <= 0 || len(items) == 0 {
		return nil, ErrSinConsumos
	}

	externalRef := fmt.Sprintf("mesa:%s:v%d:token:%s", mp.ID, mp.CuentaVersion, qrToken)

	backURLs := PreferenciaBackURLs{
		Success: fmt.Sprintf("%s/mesa/%s?pago=exitoso", s.appURL, qrToken),
		Failure: fmt.Sprintf("%s/mesa/%s?pago=fallido", s.appURL, qrToken),
		Pending: fmt.Sprintf("%s/mesa/%s?pago=pendiente", s.appURL, qrToken),
	}

	prefReq := PreferenciaRequest{
		Items:             items,
		BackURLs:          backURLs,
		ExternalReference: externalRef,
	}

	if s.apiURL != "" && strings.HasPrefix(s.apiURL, "https://") {
		prefReq.NotificationURL = fmt.Sprintf("%s/publica/pago/mercadopago/webhook?mesa_id=%s", s.apiURL, mp.ID)
	}

	prefResp, err := client.CrearPreferencia(ctx, prefReq)
	if err != nil {
		return nil, fmt.Errorf("error creando preferencia en Mercado Pago: %w", err)
	}

	if err := s.store.GuardarPreferencia(ctx, mp.ID, mp.CuentaVersion, prefResp.ID, total); err != nil {
		slog.WarnContext(ctx, "no se pudo registrar la preferencia en la BD local", "err", err)
	}

	return &PreferenciaDTO{
		PreferenceID:     prefResp.ID,
		InitPoint:        prefResp.InitPoint,
		SandboxInitPoint: prefResp.SandboxInitPoint,
		MontoTotal:       total,
	}, nil
}

func (s *Service) ConfirmarPago(ctx context.Context, qrToken string, paymentID string) (*RegistroPago, error) {
	if strings.TrimSpace(paymentID) == "" {
		return nil, fmt.Errorf("payment_id requerido")
	}

	mp, err := s.mesaSvc.ObtenerPorQRToken(ctx, qrToken)
	if err != nil {
		return nil, err
	}

	client, err := s.resolverClient(ctx, mp.ID)
	if err != nil {
		return nil, err
	}

	pagoMP, err := client.ObtenerPago(ctx, paymentID)
	if err != nil {
		return nil, fmt.Errorf("error consultando pago en Mercado Pago: %w", err)
	}

	if pagoMP.Status != "approved" {
		return nil, fmt.Errorf("%w: estado actual %q", ErrPagoNoAprobado, pagoMP.Status)
	}

	// Validar que el pago corresponde a esta mesa
	if !strings.Contains(pagoMP.ExternalReference, mp.ID) {
		return nil, ErrPagoNoCorresponde
	}

	existe, err := s.store.ExistePago(ctx, paymentID)
	if err != nil {
		return nil, err
	}
	if existe {
		return &RegistroPago{
			MesaID: mp.ID,
			PagoID: paymentID,
			Monto:  pagoMP.TransactionAmount,
			Moneda: "ARS",
			Estado: "aprobado",
		}, nil
	}

	rp, err := s.store.RegistrarPagoAprobado(ctx, mp.ID, mp.CuentaVersion, paymentID, "", pagoMP.TransactionAmount, pagoMP)
	if err != nil {
		return nil, fmt.Errorf("error registrando pago aprobado: %w", err)
	}

	// Cerrar la cuenta de la mesa automáticamente
	if _, err := s.mesaSvc.CerrarCuenta(evento.ContextoConActor(ctx, evento.MercadoPago()), mp.ID, mp.TenantID); err != nil {
		slog.ErrorContext(ctx, "error cerrando cuenta de mesa tras pago", "mesa_id", mp.ID, "err", err)
	}

	// Notificar en tiempo real al dashboard y comensal
	notificacion.Instancia.Publicar(fmt.Sprintf("sucursal:%s", mp.SucursalID), "pago_recibido", map[string]any{
		"mesa_id": mp.ID,
		"numero":  mp.Numero,
		"monto":   pagoMP.TransactionAmount,
		"pago_id": paymentID,
	})
	notificacion.Instancia.Publicar(fmt.Sprintf("mesa:%s", mp.ID), "pago_aprobado", map[string]any{
		"monto":   pagoMP.TransactionAmount,
		"pago_id": paymentID,
	})

	return rp, nil
}

func (s *Service) ProcesarWebhook(ctx context.Context, query url.Values, dataID string, eventType string) error {
	id := dataID
	if id == "" {
		id = query.Get("data.id")
		if id == "" {
			id = query.Get("id")
		}
	}
	tipo := eventType
	if tipo == "" {
		tipo = query.Get("type")
		if tipo == "" {
			tipo = query.Get("topic")
		}
	}

	if id == "" || (tipo != "payment" && tipo != "") {
		slog.InfoContext(ctx, "webhook de Mercado Pago ignorado (no es payment)", "tipo", tipo, "id", id)
		return nil
	}

	client := s.client
	if mesaIDParam := query.Get("mesa_id"); mesaIDParam != "" {
		if c, err := s.resolverClient(ctx, mesaIDParam); err == nil && c != nil {
			client = c
		}
	}
	if client == nil {
		return fmt.Errorf("no hay cliente de Mercado Pago configurado para procesar webhook")
	}

	pagoMP, err := client.ObtenerPago(ctx, id)
	if err != nil {
		return fmt.Errorf("error obteniendo pago en webhook: %w", err)
	}

	if pagoMP.Status != "approved" {
		slog.InfoContext(ctx, "pago aún no aprobado en webhook", "status", pagoMP.Status, "id", id)
		return nil
	}

	// Parsear external_reference: "mesa:<mesaID>:v<cuentaVersion>:token:<token>"
	mesaID, cuentaVersion := parseExternalReference(pagoMP.ExternalReference)
	if mesaID == "" {
		slog.WarnContext(ctx, "external_reference no contiene mesa_id válido", "ref", pagoMP.ExternalReference)
		return nil
	}

	existe, err := s.store.ExistePago(ctx, id)
	if err != nil {
		return err
	}
	if existe {
		slog.InfoContext(ctx, "pago ya procesado anteriormente", "pago_id", id)
		return nil
	}

	tenantID, sucursalID, numero, err := s.store.ObtenerMesaIDYTenantPorPago(ctx, mesaID)
	if err != nil {
		return fmt.Errorf("error obteniendo tenant de la mesa: %w", err)
	}

	if _, err := s.store.RegistrarPagoAprobado(ctx, mesaID, cuentaVersion, id, "", pagoMP.TransactionAmount, pagoMP); err != nil {
		return fmt.Errorf("error registrando pago aprobado desde webhook: %w", err)
	}

	// Cerrar la cuenta de la mesa
	if _, err := s.mesaSvc.CerrarCuenta(evento.ContextoConActor(ctx, evento.MercadoPago()), mesaID, tenantID); err != nil {
		slog.ErrorContext(ctx, "error cerrando cuenta desde webhook", "mesa_id", mesaID, "err", err)
	}

	notificacion.Instancia.Publicar(fmt.Sprintf("sucursal:%s", sucursalID), "pago_recibido", map[string]any{
		"mesa_id": mesaID,
		"numero":  numero,
		"monto":   pagoMP.TransactionAmount,
		"pago_id": id,
	})
	notificacion.Instancia.Publicar(fmt.Sprintf("mesa:%s", mesaID), "pago_aprobado", map[string]any{
		"monto":   pagoMP.TransactionAmount,
		"pago_id": id,
	})

	return nil
}

func parseExternalReference(ref string) (mesaID string, cuentaVersion int) {
	partes := strings.Split(ref, ":")
	for i := 0; i < len(partes); i++ {
		if partes[i] == "mesa" && i+1 < len(partes) {
			mesaID = partes[i+1]
		}
		if strings.HasPrefix(partes[i], "v") {
			v, err := strconv.Atoi(strings.TrimPrefix(partes[i], "v"))
			if err == nil {
				cuentaVersion = v
			}
		}
	}
	if cuentaVersion <= 0 {
		cuentaVersion = 1
	}
	return mesaID, cuentaVersion
}
