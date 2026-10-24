package carta

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestImportarCartaConFilaInvalidaCancelaArchivoCompleto(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	archivo, err := writer.CreateFormFile("archivo", "carta.csv")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := archivo.Write([]byte("categoria,nombre,precio\nCafés,Latte,2800\nCafés,Té,no-es-un-precio\n")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/carta/importar", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	(&Handlers{}).ImportarCarta(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status inesperado: got %d, want %d", recorder.Code, http.StatusOK)
	}
	var resultado ResultadoImportacion
	if err := json.NewDecoder(recorder.Body).Decode(&resultado); err != nil {
		t.Fatalf("respuesta JSON inválida: %v", err)
	}
	if resultado.Creados != 0 || resultado.Omitidos != 2 || len(resultado.Errores) != 1 {
		t.Fatalf("la carga debía cancelarse completa: %+v", resultado)
	}
}

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
