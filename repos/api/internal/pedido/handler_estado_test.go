package pedido_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/pedido"
)

func patchEstado(t *testing.T, store *mockStore, rol, body string) *httptest.ResponseRecorder {
	t.Helper()
	h := pedido.NuevosHandlers(pedido.NuevoService(store))
	req := httptest.NewRequest(http.MethodPatch, "/pedidos/p-1/estado", strings.NewReader(body))
	req.SetPathValue("id", "p-1")
	req = req.WithContext(auth.ContextConClaims(req.Context(), actorRol(rol)))
	rec := httptest.NewRecorder()
	h.CambiarEstado(rec, req)
	return rec
}

func codigoDe(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("respuesta no es JSON: %v (%s)", err, rec.Body.String())
	}
	return resp["codigo"]
}

func TestHandlerCambiarEstado_Errores(t *testing.T) {
	tests := []struct {
		nombre, rol, desde, body, codigo string
		status                           int
	}{
		{"entregado sin modo_entrega", auth.RolMozo, "listo", `{"estado":"entregado"}`, "MODO_ENTREGA_INVALIDO", http.StatusBadRequest},
		{"entregado con modo inválido", auth.RolMozo, "listo", `{"estado":"entregado","modo_entrega":"delivery"}`, "MODO_ENTREGA_INVALIDO", http.StatusBadRequest},
		{"cocina no confirma", auth.RolCocinero, "recibido", `{"estado":"confirmado"}`, "SIN_PERMISO", http.StatusForbidden},
		{"cocina no entrega", auth.RolCocinero, "listo", `{"estado":"entregado","modo_entrega":"mesa"}`, "SIN_PERMISO", http.StatusForbidden},
		{"retroceso", auth.RolDuenio, "listo", `{"estado":"preparando"}`, "TRANSICION_INVALIDA", http.StatusConflict},
		{"entregar sin estar listo", auth.RolDuenio, "preparando", `{"estado":"entregado","modo_entrega":"mesa"}`, "TRANSICION_INVALIDA", http.StatusConflict},
		{"preparando sin confirmar", auth.RolMozo, "recibido", `{"estado":"preparando"}`, "PEDIDO_SIN_CONFIRMAR", http.StatusConflict},
		{"desde cerrado", auth.RolDuenio, "cerrado", `{"estado":"confirmado"}`, "TRANSICION_INVALIDA", http.StatusConflict},
	}
	for _, tt := range tests {
		t.Run(tt.nombre, func(t *testing.T) {
			rec := patchEstado(t, &mockStore{estadoActual: tt.desde}, tt.rol, tt.body)
			if rec.Code != tt.status || codigoDe(t, rec) != tt.codigo {
				t.Fatalf("got %d %s, want %d %s", rec.Code, rec.Body.String(), tt.status, tt.codigo)
			}
		})
	}
}

func TestHandlerCambiarEstado_Exito(t *testing.T) {
	store := &mockStore{estadoActual: "listo"}
	rec := patchEstado(t, store, auth.RolMozo, `{"estado":"entregado","modo_entrega":"retiro"}`)
	if rec.Code != http.StatusOK || store.modoEntrega != "retiro" {
		t.Fatalf("got %d %s, modo=%q", rec.Code, rec.Body.String(), store.modoEntrega)
	}
}

func TestHandlerCambiarEstadoItem_PedidoSinConfirmarEs409(t *testing.T) {
	h := pedido.NuevosHandlers(pedido.NuevoService(&mockStore{errItem: pedido.ErrPedidoSinConfirmar}))
	req := httptest.NewRequest(http.MethodPatch, "/pedidos/items/i-1/estado", strings.NewReader(`{"estado":"preparando"}`))
	req.SetPathValue("id", "i-1")
	req = req.WithContext(auth.ContextConClaims(req.Context(), actorRol(auth.RolCocinero)))
	rec := httptest.NewRecorder()
	h.CambiarEstadoItem(rec, req)
	if rec.Code != http.StatusConflict || codigoDe(t, rec) != "PEDIDO_SIN_CONFIRMAR" {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
}
