package resenia_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/resenia"
)

func TestHandler_CrearResena(t *testing.T) {
	t.Run("exito retorna 201", func(t *testing.T) {
		store := &mockStore{
			mesaCtx: &resenia.MesaContexto{
				MesaID:        "mesa-1",
				MesaNumero:    1,
				SucursalID:    "suc-1",
				TenantID:      "ten-1",
				CuentaVersion: 1,
				Estado:        "activa",
			},
			tienePedidos: true,
			resenaExiste: false,
		}
		h := resenia.NuevosHandlers(resenia.NuevoService(store))

		body, _ := json.Marshal(resenia.CrearResenaInput{Estrellas: 5})
		req := httptest.NewRequest(http.MethodPost, "/publica/mesas/token-1/resena", bytes.NewReader(body))
		req.SetPathValue("qr_token", "token-1")
		rec := httptest.NewRecorder()

		h.Crear(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("esperaba código 201, obtuve %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("duplicado retorna 409", func(t *testing.T) {
		store := &mockStore{
			mesaCtx: &resenia.MesaContexto{
				MesaID:        "mesa-1",
				MesaNumero:    1,
				SucursalID:    "suc-1",
				TenantID:      "ten-1",
				CuentaVersion: 1,
				Estado:        "activa",
			},
			tienePedidos: true,
			resenaExiste: true,
		}
		h := resenia.NuevosHandlers(resenia.NuevoService(store))

		body, _ := json.Marshal(resenia.CrearResenaInput{Estrellas: 5})
		req := httptest.NewRequest(http.MethodPost, "/publica/mesas/token-1/resena", bytes.NewReader(body))
		req.SetPathValue("qr_token", "token-1")
		rec := httptest.NewRecorder()

		h.Crear(rec, req)

		if rec.Code != http.StatusConflict {
			t.Errorf("esperaba código 409, obtuve %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("sin pedidos retorna 400", func(t *testing.T) {
		store := &mockStore{
			mesaCtx: &resenia.MesaContexto{
				MesaID:        "mesa-1",
				MesaNumero:    1,
				SucursalID:    "suc-1",
				TenantID:      "ten-1",
				CuentaVersion: 1,
				Estado:        "activa",
			},
			tienePedidos: false,
		}
		h := resenia.NuevosHandlers(resenia.NuevoService(store))

		body, _ := json.Marshal(resenia.CrearResenaInput{Estrellas: 5})
		req := httptest.NewRequest(http.MethodPost, "/publica/mesas/token-1/resena", bytes.NewReader(body))
		req.SetPathValue("qr_token", "token-1")
		rec := httptest.NewRecorder()

		h.Crear(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("esperaba código 400, obtuve %d: %s", rec.Code, rec.Body.String())
		}
	})
}

func TestHandler_RegistrarClickGoogle(t *testing.T) {
	store := &mockStore{}
	h := resenia.NuevosHandlers(resenia.NuevoService(store))

	req := httptest.NewRequest(http.MethodPatch, "/publica/mesas/token-1/resena/google-click", nil)
	req.SetPathValue("qr_token", "token-1")
	rec := httptest.NewRecorder()

	h.RegistrarClickGoogle(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("esperaba 200, obtuve %d", rec.Code)
	}
}

func TestHandler_Resumen(t *testing.T) {
	store := &mockStore{
		resumen: &resenia.ResumenResenas{
			Promedio:     4.5,
			Total:        10,
			CSAT:         80.0,
			Distribucion: map[int]int{1: 0, 2: 0, 3: 2, 4: 3, 5: 5},
			ClicsGoogle:  4,
			Resenas:      []resenia.ResenaItem{},
		},
	}
	h := resenia.NuevosHandlers(resenia.NuevoService(store))

	// Sin claims -> 401
	req := httptest.NewRequest(http.MethodGet, "/resenias/resumen", nil)
	rec := httptest.NewRecorder()
	h.Resumen(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("esperaba 401 sin auth, obtuve %d", rec.Code)
	}

	// Con claims -> 200
	ctx := auth.ContextConClaims(context.Background(), &auth.Claims{
		TenantID:  "ten-1",
		UsuarioID: "usr-1",
		Rol:       "admin",
	})
	req = httptest.NewRequest(http.MethodGet, "/resenias/resumen", nil).WithContext(ctx)
	rec = httptest.NewRecorder()
	h.Resumen(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("esperaba 200 con auth, obtuve %d", rec.Code)
	}
}
