package carta

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResponderErrorFranjaDuplicadaDevuelveConflict(t *testing.T) {
	h := &Handlers{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/carta/franjas", nil)

	h.responderErrorFranja(recorder, request, ErrFranjaDuplicada)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("status inesperado: got %d, want %d", recorder.Code, http.StatusConflict)
	}

	var respuesta map[string]any
	if err := json.NewDecoder(recorder.Body).Decode(&respuesta); err != nil {
		t.Fatalf("respuesta JSON inválida: %v", err)
	}
	if respuesta["error"] != ErrFranjaDuplicada.Error() {
		t.Fatalf("mensaje inesperado: %+v", respuesta)
	}
}
