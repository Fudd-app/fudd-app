package resenia_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/notificacion"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/resenia"
)

type mockStore struct {
	mesaCtx                 *resenia.MesaContexto
	errMesa                 error
	tienePedidos            bool
	errPedidos              error
	resenaExiste            bool
	errResenaExiste         error
	resenaCreada            *resenia.Resena
	errCrearResena          error
	errGoogleClick          error
	sucursalPertenece       bool
	errSucursalPertenece    error
	resumen                 *resenia.ResumenResenas
	errResumen              error
	ultimoPayloadNotificado *resenia.ResenaNotificacionPayload
}

func (m *mockStore) ObtenerContextoMesaPorQR(ctx context.Context, qrToken string) (*resenia.MesaContexto, error) {
	if m.errMesa != nil {
		return nil, m.errMesa
	}
	return m.mesaCtx, nil
}

func (m *mockStore) VerificarPedidosEnCuenta(ctx context.Context, mesaID string, cuentaVersion int) (bool, error) {
	if m.errPedidos != nil {
		return false, m.errPedidos
	}
	return m.tienePedidos, nil
}

func (m *mockStore) VerificarResenaExiste(ctx context.Context, mesaID string, cuentaVersion int) (bool, error) {
	if m.errResenaExiste != nil {
		return false, m.errResenaExiste
	}
	return m.resenaExiste, nil
}

func (m *mockStore) CrearResena(ctx context.Context, mesaID, sucursalID, tenantID string, cuentaVersion, estrellas int, comentario *string) (*resenia.Resena, error) {
	if m.errCrearResena != nil {
		return nil, m.errCrearResena
	}
	if m.resenaCreada != nil {
		return m.resenaCreada, nil
	}
	return &resenia.Resena{
		ID:            "res-1",
		MesaID:        mesaID,
		SucursalID:    sucursalID,
		TenantID:      tenantID,
		CuentaVersion: cuentaVersion,
		Estrellas:     estrellas,
		Comentario:    comentario,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}, nil
}

func (m *mockStore) RegistrarClickGoogle(ctx context.Context, qrToken string) error {
	return m.errGoogleClick
}

func (m *mockStore) SucursalPerteneceATenant(ctx context.Context, sucursalID, tenantID string) (bool, error) {
	if m.errSucursalPertenece != nil {
		return false, m.errSucursalPertenece
	}
	return m.sucursalPertenece, nil
}

func (m *mockStore) ObtenerResumen(ctx context.Context, tenantID, sucursalID string, desde, hasta *time.Time) (*resenia.ResumenResenas, error) {
	if m.errResumen != nil {
		return nil, m.errResumen
	}
	return m.resumen, nil
}

func TestCrearResena_Validaciones(t *testing.T) {
	store := &mockStore{}
	svc := resenia.NuevoService(store)

	// QR token vacío
	_, err := svc.CrearResena(context.Background(), "", resenia.CrearResenaInput{Estrellas: 5})
	if !errors.Is(err, resenia.ErrValidation) {
		t.Errorf("esperaba ErrValidation para token vacío, obtuve: %v", err)
	}

	// Estrellas menores a 1
	_, err = svc.CrearResena(context.Background(), "token-1", resenia.CrearResenaInput{Estrellas: 0})
	if !errors.Is(err, resenia.ErrValidation) {
		t.Errorf("esperaba ErrValidation para 0 estrellas, obtuve: %v", err)
	}

	// Estrellas mayores a 5
	_, err = svc.CrearResena(context.Background(), "token-1", resenia.CrearResenaInput{Estrellas: 6})
	if !errors.Is(err, resenia.ErrValidation) {
		t.Errorf("esperaba ErrValidation para 6 estrellas, obtuve: %v", err)
	}

	// Comentario demasiado largo (>500 caracteres)
	comentarioLargo := strings.Repeat("a", 501)
	_, err = svc.CrearResena(context.Background(), "token-1", resenia.CrearResenaInput{
		Estrellas:  5,
		Comentario: &comentarioLargo,
	})
	if !errors.Is(err, resenia.ErrValidation) {
		t.Errorf("esperaba ErrValidation para comentario > 500 chars, obtuve: %v", err)
	}
}

func TestCrearResena_MesaNoEncontradaOInactiva(t *testing.T) {
	store := &mockStore{errMesa: resenia.ErrNotFound}
	svc := resenia.NuevoService(store)

	_, err := svc.CrearResena(context.Background(), "token-inexistente", resenia.CrearResenaInput{Estrellas: 5})
	if !errors.Is(err, resenia.ErrNotFound) {
		t.Errorf("esperaba ErrNotFound, obtuve: %v", err)
	}

	// Mesa inactiva
	store.errMesa = nil
	store.mesaCtx = &resenia.MesaContexto{
		MesaID: "mesa-1",
		Estado: "inactiva",
	}
	_, err = svc.CrearResena(context.Background(), "token-inactivo", resenia.CrearResenaInput{Estrellas: 5})
	if !errors.Is(err, resenia.ErrMesaInactiva) {
		t.Errorf("esperaba ErrMesaInactiva, obtuve: %v", err)
	}
}

func TestCrearResena_SinPedidosEnCuenta(t *testing.T) {
	store := &mockStore{
		mesaCtx: &resenia.MesaContexto{
			MesaID:        "mesa-1",
			MesaNumero:    3,
			SucursalID:    "suc-1",
			TenantID:      "ten-1",
			CuentaVersion: 1,
			Estado:        "activa",
		},
		tienePedidos: false,
	}
	svc := resenia.NuevoService(store)

	_, err := svc.CrearResena(context.Background(), "token-1", resenia.CrearResenaInput{Estrellas: 5})
	if !errors.Is(err, resenia.ErrSinPedidos) {
		t.Errorf("esperaba ErrSinPedidos, obtuve: %v", err)
	}
}

func TestCrearResena_DuplicadaDevuelveConflicto(t *testing.T) {
	store := &mockStore{
		mesaCtx: &resenia.MesaContexto{
			MesaID:        "mesa-1",
			MesaNumero:    3,
			SucursalID:    "suc-1",
			TenantID:      "ten-1",
			CuentaVersion: 1,
			Estado:        "activa",
		},
		tienePedidos: true,
		resenaExiste: true,
	}
	svc := resenia.NuevoService(store)

	_, err := svc.CrearResena(context.Background(), "token-1", resenia.CrearResenaInput{Estrellas: 4})
	if !errors.Is(err, resenia.ErrResenaDuplicada) {
		t.Errorf("esperaba ErrResenaDuplicada, obtuve: %v", err)
	}
}

func TestCrearResena_Exito(t *testing.T) {
	sucursalID := "suc-1"
	ch, desuscribir := notificacion.Instancia.Suscribir("sucursal:" + sucursalID)
	defer desuscribir()

	comentario := "Excelente atención y comida"
	store := &mockStore{
		mesaCtx: &resenia.MesaContexto{
			MesaID:        "mesa-1",
			MesaNumero:    5,
			SucursalID:    sucursalID,
			TenantID:      "ten-1",
			CuentaVersion: 2,
			Estado:        "activa",
		},
		tienePedidos: true,
		resenaExiste: false,
	}
	svc := resenia.NuevoService(store)

	r, err := svc.CrearResena(context.Background(), "token-valido", resenia.CrearResenaInput{
		Estrellas:  5,
		Comentario: &comentario,
	})
	if err != nil {
		t.Fatalf("error inesperado creando reseña: %v", err)
	}
	if r.Estrellas != 5 {
		t.Errorf("esperaba 5 estrellas, obtuve %d", r.Estrellas)
	}
	if r.Comentario == nil || *r.Comentario != comentario {
		t.Errorf("esperaba comentario %q, obtuve %v", comentario, r.Comentario)
	}

	// Verificar notificación SSE recibida
	select {
	case ev := <-ch:
		if ev.Nombre != "resena_creada" {
			t.Errorf("esperaba evento 'resena_creada', obtuve %s", ev.Nombre)
		}
	case <-time.After(1 * time.Second):
		t.Error("timeout esperando notificación SSE de resena_creada")
	}
}

func TestRegistrarClickGoogle(t *testing.T) {
	store := &mockStore{}
	svc := resenia.NuevoService(store)

	err := svc.RegistrarClickGoogle(context.Background(), "token-1")
	if err != nil {
		t.Errorf("error inesperado registrando clic: %v", err)
	}

	store.errGoogleClick = resenia.ErrNotFound
	err = svc.RegistrarClickGoogle(context.Background(), "token-inexistente")
	if !errors.Is(err, resenia.ErrNotFound) {
		t.Errorf("esperaba ErrNotFound, obtuve: %v", err)
	}
}

func TestObtenerResumen_Validaciones(t *testing.T) {
	store := &mockStore{
		sucursalPertenece: false,
	}
	svc := resenia.NuevoService(store)

	// Tenant vacío
	_, err := svc.ObtenerResumen(context.Background(), "", "", "", "")
	if !errors.Is(err, resenia.ErrValidation) {
		t.Errorf("esperaba ErrValidation para tenant vacío, obtuve: %v", err)
	}

	// Sucursal no pertenece
	_, err = svc.ObtenerResumen(context.Background(), "ten-1", "suc-ajena", "", "")
	if !errors.Is(err, resenia.ErrSucursalInvalida) {
		t.Errorf("esperaba ErrSucursalInvalida, obtuve: %v", err)
	}

	// Formato de fecha inválido
	store.sucursalPertenece = true
	_, err = svc.ObtenerResumen(context.Background(), "ten-1", "", "fecha-invalida", "")
	if !errors.Is(err, resenia.ErrValidation) {
		t.Errorf("esperaba ErrValidation para fecha inválida, obtuve: %v", err)
	}
}
