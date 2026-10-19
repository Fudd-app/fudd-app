package pedido_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/pedido"
)

func TestEventosKDS_AbreCanalSSEParaSucursalDelTenant(t *testing.T) {
	const secreto = "secreto-test-kds"
	t.Setenv("JWT_SECRET", secreto)

	store := &mockStore{sucursalPertenece: true}
	handler := pedido.NuevosHandlers(pedido.NuevoService(store))
	protegido := auth.Requerir(http.HandlerFunc(handler.EventosKDS))

	token, err := auth.GenerarJWT(&auth.Claims{
		UsuarioID:  "usuario-1",
		TenantID:   "tenant-1",
		Rol:        "encargado",
		SucursalID: "suc-1",
	}, secreto)
	if err != nil {
		t.Fatalf("no se pudo generar JWT de prueba: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/kds/eventos?sucursal_id=suc-1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	ctx, cancel := context.WithCancel(req.Context())
	defer cancel()
	req = req.WithContext(ctx)
	recorder := httptest.NewRecorder()

	finalizado := make(chan struct{})
	go func() {
		protegido.ServeHTTP(recorder, req)
		close(finalizado)
	}()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(recorder.Body.String(), "event: ping") {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if got := recorder.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("content type incorrecto: got %q", got)
	}
	if body := recorder.Body.String(); !strings.Contains(body, "event: ping") || !strings.Contains(body, "data: conectado") {
		t.Fatalf("no se recibió el ping inicial del KDS: %q", body)
	}

	cancel()
	select {
	case <-finalizado:
	case <-time.After(time.Second):
		t.Fatal("el canal KDS no cerró al cancelar el contexto")
	}
}

func TestEventosKDS_RechazaSucursalDeOtroTenant(t *testing.T) {
	const secreto = "secreto-test-kds-ajeno"
	t.Setenv("JWT_SECRET", secreto)

	store := &mockStore{sucursalPertenece: false}
	handler := pedido.NuevosHandlers(pedido.NuevoService(store))
	protegido := auth.Requerir(http.HandlerFunc(handler.EventosKDS))
	token, err := auth.GenerarJWT(&auth.Claims{
		UsuarioID:  "usuario-1",
		TenantID:   "tenant-1",
		Rol:        "encargado",
		SucursalID: "suc-ajena",
	}, secreto)
	if err != nil {
		t.Fatalf("no se pudo generar JWT de prueba: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/kds/eventos?sucursal_id=suc-ajena", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	protegido.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status incorrecto: got %d, want %d", recorder.Code, http.StatusNotFound)
	}
}
