package notificacion_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/notificacion"
)

func TestBroker_SuscripcionYPublicacion(t *testing.T) {
	broker := notificacion.NuevoBroker()
	canal := "test-canal"

	ch, desuscribir := broker.Suscribir(canal)
	defer desuscribir()

	mensaje := map[string]string{"mensaje": "hola mundo"}
	broker.Publicar(canal, "saludo", mensaje)

	select {
	case ev, ok := <-ch:
		if !ok {
			t.Fatal("el canal se cerró inesperadamente")
		}
		if ev.Nombre != "saludo" {
			t.Errorf("nombre de evento incorrecto: got %q, want %q", ev.Nombre, "saludo")
		}
		expectedData := `{"mensaje":"hola mundo"}`
		if ev.Data != expectedData {
			t.Errorf("datos incorrectos: got %q, want %q", ev.Data, expectedData)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout esperando el evento publicado")
	}
}

func TestBroker_Desuscripcion(t *testing.T) {
	broker := notificacion.NuevoBroker()
	canal := "test-canal-desuscripcion"

	ch, desuscribir := broker.Suscribir(canal)
	desuscribir() // Desuscribir inmediatamente

	// Intentar publicar en el canal desuscrito
	broker.Publicar(canal, "evento", "datos")

	// Verificar que el canal se cierra al desuscribir y no recibe el evento
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("el canal debería estar cerrado o no recibir datos tras la desuscripción")
		}
	case <-time.After(100 * time.Millisecond):
		// Ok, no bloqueó y el canal se cerró o no envió nada
	}
}

func TestHandlers_EventosSucursal_SSEConnect(t *testing.T) {
	handlers := notificacion.NuevosHandlers()

	// Crear request con un context cancelable para simular desconexión
	req := httptest.NewRequest("GET", "/sucursales/123/eventos", nil)
	req.SetPathValue("sucursal_id", "123")

	ctx, cancel := context.WithCancel(req.Context())
	ctx = auth.ContextConClaims(ctx, &auth.Claims{UsuarioID: "u-1", TenantID: "t-1", Rol: auth.RolEncargado, SucursalID: "123"})
	req = req.WithContext(ctx)

	// Usar httptest.NewRecorder que implementa http.Flusher de forma simulada en algunas versiones de go,
	// o usar un mock especial. En Go, httptest.ResponseRecorder no implementa http.Flusher por defecto en versiones antiguas,
	// pero en las modernas sí. Para estar seguros, implementaremos una respuesta corta y cancelaremos.
	w := httptest.NewRecorder()

	// Corremos el handler en una goroutine ya que es un endpoint bloqueante (keep-alive)
	go func() {
		handlers.EventosSucursal(w, req)
	}()

	body := esperarPing(t, w)
	cancel() // Desconectar al cliente

	if !strings.Contains(body, "event: ping") || !strings.Contains(body, "data: conectado") {
		t.Errorf("no se recibió el ping de conexión inicial. Body: %q", body)
	}
}

func TestHandlers_EventosPedido_SSEConnect(t *testing.T) {
	handlers := notificacion.NuevosHandlers()

	req := httptest.NewRequest("GET", "/pedidos/abc/eventos", nil)
	req.SetPathValue("id", "abc")

	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	go func() {
		handlers.EventosPedido(w, req)
	}()

	body := esperarPing(t, w)
	cancel()

	if !strings.Contains(body, "event: ping") || !strings.Contains(body, "data: conectado") {
		t.Errorf("no se recibió el ping de conexión inicial para pedido. Body: %q", body)
	}
}

func TestHandlers_EventosMesa_SSEConnect(t *testing.T) {
	handlers := notificacion.NuevosHandlers()

	req := httptest.NewRequest("GET", "/publica/mesas/mesa-123/eventos", nil)
	req.SetPathValue("mesa_id", "mesa-123")

	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	go func() {
		handlers.EventosMesa(w, req)
	}()

	body := esperarPing(t, w)
	cancel()

	if !strings.Contains(body, "event: ping") || !strings.Contains(body, "data: conectado") {
		t.Errorf("no se recibió el ping de conexión inicial para mesa. Body: %q", body)
	}
}

func esperarPing(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()

	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) {
		body := w.Body.String()
		if strings.Contains(body, "event: ping") && strings.Contains(body, "data: conectado") {
			return body
		}
		time.Sleep(10 * time.Millisecond)
	}
	return w.Body.String()
}
