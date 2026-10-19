package pedido

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/notificacion"
)

type Handlers struct {
	svc *Service
}

func NuevosHandlers(svc *Service) *Handlers { return &Handlers{svc: svc} }

func (h *Handlers) Crear(w http.ResponseWriter, r *http.Request) {
	var input NuevoPedidoInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, "body inválido", http.StatusBadRequest)
		return
	}
	p, err := h.svc.Crear(r.Context(), input)
	if err != nil {
		if errors.Is(err, ErrMesaCerrada) {
			jsonError(w, err.Error(), http.StatusConflict)
			return
		}
		if errors.Is(err, ErrCuentaSolicitada) {
			jsonError(w, err.Error(), http.StatusConflict)
			return
		}
		if errors.Is(err, ErrValidation) || errors.Is(err, ErrNotFound) {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		slog.ErrorContext(r.Context(), "error creando pedido", "error", err)
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(p)
}

func (h *Handlers) CrearAsistido(w http.ResponseWriter, r *http.Request) {
	var input PedidoAsistidoInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, "body inválido", http.StatusBadRequest)
		return
	}
	p, err := h.svc.CrearAsistido(r.Context(), auth.ClaimsFromContext(r.Context()), input)
	if err != nil {
		switch {
		case errors.Is(err, ErrMesaCerrada), errors.Is(err, ErrCuentaSolicitada):
			jsonError(w, err.Error(), http.StatusConflict)
		case errors.Is(err, ErrValidation), errors.Is(err, ErrNotFound):
			jsonError(w, err.Error(), http.StatusBadRequest)
		default:
			jsonError(w, err.Error(), http.StatusForbidden)
		}
		return
	}
	jsonCreado(w, p)
}

func (h *Handlers) ListarActivos(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	sucursalID, err := auth.ResolverSucursal(claims, r.URL.Query().Get("sucursal_id"))
	if err != nil {
		jsonError(w, err.Error(), http.StatusForbidden)
		return
	}
	pedidos, err := h.svc.ListarActivos(r.Context(), sucursalID, claims.TenantID)
	if err != nil {
		slog.ErrorContext(r.Context(), "error listando pedidos", "err", err)
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}
	jsonOK(w, pedidos)
}

func (h *Handlers) ListarCuentaActual(w http.ResponseWriter, r *http.Request) {
	qrToken := r.PathValue("qr_token")
	pedidos, err := h.svc.ListarCuentaActualPorQR(r.Context(), qrToken)
	if err != nil {
		if errors.Is(err, ErrValidation) {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		slog.ErrorContext(r.Context(), "error listando cuenta actual de mesa", "err", err)
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}
	jsonOK(w, pedidos)
}

func (h *Handlers) CambiarEstado(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	id := r.PathValue("id")
	var input CambiarEstadoInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, "body inválido", http.StatusBadRequest)
		return
	}
	p, err := h.svc.CambiarEstado(r.Context(), id, claims.TenantID, claims.SucursalID, input.Estado)
	if err != nil {
		if errors.Is(err, ErrValidation) {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, ErrNotFound) {
			jsonError(w, "pedido no encontrado", http.StatusNotFound)
			return
		}
		slog.ErrorContext(r.Context(), "error cambiando estado", "error", err)
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}
	jsonOK(w, p)
}

func (h *Handlers) CambiarEstadoItem(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		jsonError(w, "no autorizado", http.StatusUnauthorized)
		return
	}

	itemID := r.PathValue("id")
	var input CambiarEstadoItemInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, "body inválido", http.StatusBadRequest)
		return
	}

	p, err := h.svc.CambiarEstadoItem(r.Context(), itemID, claims.TenantID, claims.SucursalID, input.Estado)
	if err != nil {
		switch {
		case errors.Is(err, ErrValidation):
			jsonError(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrPedidoCerrado):
			jsonError(w, "el pedido ya está cerrado", http.StatusConflict)
		case errors.Is(err, ErrNotFound):
			jsonError(w, "item de pedido no encontrado", http.StatusNotFound)
		default:
			slog.ErrorContext(r.Context(), "error cambiando estado del item", "error", err, "item_id", itemID)
			jsonError(w, "error interno", http.StatusInternalServerError)
		}
		return
	}

	jsonOK(w, p)
}

func (h *Handlers) EventosKDS(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		jsonError(w, "no autorizado", http.StatusUnauthorized)
		return
	}

	sucursalID, err := auth.ResolverSucursal(claims, r.URL.Query().Get("sucursal_id"))
	if err != nil {
		jsonError(w, err.Error(), http.StatusForbidden)
		return
	}
	if err := h.svc.ValidarAccesoKDS(r.Context(), sucursalID, claims.TenantID); err != nil {
		if errors.Is(err, ErrValidation) {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, ErrNotFound) {
			jsonError(w, "sucursal no encontrada", http.StatusNotFound)
			return
		}
		slog.ErrorContext(r.Context(), "error validando acceso a eventos KDS", "error", err)
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		jsonError(w, "SSE no soportado", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	canal := fmt.Sprintf("kds:%s", sucursalID)
	ch, desuscribir := notificacion.Instancia.Suscribir(canal)
	defer desuscribir()
	keepAlive := time.NewTicker(25 * time.Second)
	defer keepAlive.Stop()

	slog.InfoContext(r.Context(), "cliente conectado a eventos de cocina", "sucursal_id", sucursalID)
	fmt.Fprint(w, "event: ping\ndata: conectado\n\n")
	flusher.Flush()

	for {
		select {
		case evento, abierto := <-ch:
			if !abierto {
				return
			}
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", evento.Nombre, evento.Data)
			flusher.Flush()
		case <-keepAlive.C:
			fmt.Fprint(w, ": keepalive\n\n")
			flusher.Flush()
		case <-r.Context().Done():
			slog.InfoContext(r.Context(), "cliente desconectado de eventos de cocina", "sucursal_id", sucursalID)
			return
		}
	}
}

func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func jsonCreado(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
