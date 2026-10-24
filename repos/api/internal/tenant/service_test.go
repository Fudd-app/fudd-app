package tenant_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/tenant"
)

func TestVistaOperativaNoExponeConfiguracionGlobal(t *testing.T) {
	secreto := "credencial-privada"
	vista := (&tenant.Tenant{
		ID: "t-1", Nombre: "Café", Slug: "cafe", Plan: "pro",
		DatosFiscales: map[string]any{"cuit": "secret"}, MPAccessToken: &secreto,
	}).VistaOperativa()

	contenido, err := json.Marshal(vista)
	if err != nil {
		t.Fatal(err)
	}
	jsonTexto := string(contenido)
	for _, campoSensible := range []string{"datos_fiscales", "mp_access_token", "plan", "credencial-privada", "secret"} {
		if strings.Contains(jsonTexto, campoSensible) {
			t.Fatalf("la vista operativa expuso %q: %s", campoSensible, jsonTexto)
		}
	}
}

type mockStore struct {
	crearFn                     func(ctx context.Context, input tenant.OnboardingInput) (*tenant.Tenant, error)
	actualizarFn                func(ctx context.Context, id string, input tenant.ActualizarTenantInput) (*tenant.Tenant, error)
	obtenerEstadoCuotasFn       func(ctx context.Context, tenantID string) (*tenant.EstadoCuotas, error)
	registrarSolicitudUpgradeFn func(ctx context.Context, tenantID, nota string) (*tenant.Tenant, error)
	slugEnUsoFn                 func(ctx context.Context, slug string) (bool, error)
	funcionalidadHabilitadaFn   func(ctx context.Context, tenantID string, rec tenant.Recurso, plan string, ahora time.Time) (bool, error)
}

func (m *mockStore) Crear(ctx context.Context, input tenant.OnboardingInput) (*tenant.Tenant, error) {
	return m.crearFn(ctx, input)
}
func (m *mockStore) ObtenerPorID(ctx context.Context, id string) (*tenant.Tenant, error) {
	return nil, nil
}
func (m *mockStore) Actualizar(ctx context.Context, id string, input tenant.ActualizarTenantInput) (*tenant.Tenant, error) {
	if m.actualizarFn != nil {
		return m.actualizarFn(ctx, id, input)
	}
	return nil, nil
}
func (m *mockStore) EmailAdminEnUso(ctx context.Context, email string) (bool, error) {
	return false, nil
}
func (m *mockStore) SlugEnUso(ctx context.Context, slug string) (bool, error) {
	if m.slugEnUsoFn != nil {
		return m.slugEnUsoFn(ctx, slug)
	}
	return false, nil
}
func (m *mockStore) ObtenerEstadoCuotas(ctx context.Context, tenantID string) (*tenant.EstadoCuotas, error) {
	if m.obtenerEstadoCuotasFn != nil {
		return m.obtenerEstadoCuotasFn(ctx, tenantID)
	}
	return nil, nil
}
func (m *mockStore) RegistrarSolicitudUpgrade(ctx context.Context, tenantID, nota string) (*tenant.Tenant, error) {
	if m.registrarSolicitudUpgradeFn != nil {
		return m.registrarSolicitudUpgradeFn(ctx, tenantID, nota)
	}
	return nil, nil
}

func (m *mockStore) FuncionalidadHabilitada(ctx context.Context, tenantID string, rec tenant.Recurso, plan string, ahora time.Time) (bool, error) {
	if m.funcionalidadHabilitadaFn != nil {
		return m.funcionalidadHabilitadaFn(ctx, tenantID, rec, plan, ahora)
	}
	return plan == tenant.PlanPro, nil
}

func TestCrear_Exitoso(t *testing.T) {
	var inputGuardado tenant.OnboardingInput
	store := &mockStore{
		crearFn: func(ctx context.Context, input tenant.OnboardingInput) (*tenant.Tenant, error) {
			inputGuardado = input
			return &tenant.Tenant{ID: "t-1", Nombre: input.Nombre, Slug: input.Slug}, nil
		},
	}
	svc := tenant.NuevoService(store)
	result, err := svc.Crear(context.Background(), tenant.OnboardingInput{
		Nombre:      "Mi Bar",
		Slug:        "mi-bar",
		EmailAdmin:  "admin@mibar.com",
		NombreAdmin: "Carlos",
		Password:    "MesaClick2026",
	})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if result.ID != "t-1" {
		t.Errorf("ID: got %q, want %q", result.ID, "t-1")
	}
	if inputGuardado.Password != "" {
		t.Fatal("la contraseña en texto plano no debe llegar al store")
	}
	if inputGuardado.PasswordHash == "" || inputGuardado.PasswordHash == "MesaClick2026" {
		t.Fatal("se esperaba un hash seguro de la contraseña")
	}
	if inputGuardado.Plan != tenant.PlanPro || inputGuardado.PlanDesde == nil || inputGuardado.PlanHasta == nil || inputGuardado.TrialHasta == nil {
		t.Fatalf("el negocio nuevo debe iniciar una prueba Pro: %+v", inputGuardado)
	}
	esperadoHasta := inputGuardado.PlanDesde.AddDate(0, tenant.MesesPruebaInicial, 0)
	if !inputGuardado.PlanHasta.Equal(esperadoHasta) || !inputGuardado.TrialHasta.Equal(esperadoHasta) {
		t.Fatalf("la prueba debe durar %d meses: desde=%v plan_hasta=%v prueba_hasta=%v", tenant.MesesPruebaInicial, inputGuardado.PlanDesde, inputGuardado.PlanHasta, inputGuardado.TrialHasta)
	}
	if inputGuardado.PlanDesde.After(time.Now()) {
		t.Fatal("el inicio de la prueba no puede quedar en el futuro")
	}
}

func TestCrear_SinPassword_MagicLink(t *testing.T) {
	var inputGuardado tenant.OnboardingInput
	store := &mockStore{
		crearFn: func(ctx context.Context, input tenant.OnboardingInput) (*tenant.Tenant, error) {
			inputGuardado = input
			return &tenant.Tenant{ID: "t-2", Nombre: input.Nombre, Slug: input.Slug}, nil
		},
	}
	svc := tenant.NuevoService(store)
	result, err := svc.Crear(context.Background(), tenant.OnboardingInput{
		Nombre:      "Mi Bar Magic",
		Slug:        "mi-bar-magic",
		EmailAdmin:  "admin@magic.com",
		NombreAdmin: "Carlos",
	})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if result.ID != "t-2" {
		t.Errorf("ID: got %q, want %q", result.ID, "t-2")
	}
	if inputGuardado.PasswordHash != "" {
		t.Fatalf("se esperaba PasswordHash vacío para registro magic link, got: %q", inputGuardado.PasswordHash)
	}
}

func TestCrear_PasswordDebil_Error(t *testing.T) {
	svc := tenant.NuevoService(&mockStore{})
	_, err := svc.Crear(context.Background(), tenant.OnboardingInput{
		Nombre:     "Mi Bar",
		Slug:       "mi-bar",
		EmailAdmin: "admin@mibar.com",
		Password:   "solo-letras",
	})
	if !errors.Is(err, tenant.ErrValidation) {
		t.Fatalf("se esperaba ErrValidation, obtenido: %v", err)
	}
}

func TestCrear_SlugVacio_Error(t *testing.T) {
	svc := tenant.NuevoService(&mockStore{
		crearFn: func(ctx context.Context, input tenant.OnboardingInput) (*tenant.Tenant, error) {
			return nil, errors.New("slug vacío")
		},
	})
	_, err := svc.Crear(context.Background(), tenant.OnboardingInput{
		Nombre:     "Sin Slug",
		EmailAdmin: "a@b.com",
	})
	if err == nil {
		t.Fatal("esperaba error por slug vacío")
	}
}

func TestCrear_WhatsappConLetras_Error(t *testing.T) {
	svc := tenant.NuevoService(&mockStore{})
	_, err := svc.Crear(context.Background(), tenant.OnboardingInput{
		Nombre:      "Mi Bar",
		Slug:        "mi-bar",
		EmailAdmin:  "admin@mibar.com",
		NombreAdmin: "Carlos",
		Whatsapp:    "+54 contacto",
	})
	if !errors.Is(err, tenant.ErrValidation) {
		t.Fatalf("se esperaba ErrValidation para WhatsApp inválido, obtenido: %v", err)
	}
}

func TestActualizar_Exitoso(t *testing.T) {
	store := &mockStore{
		actualizarFn: func(ctx context.Context, id string, input tenant.ActualizarTenantInput) (*tenant.Tenant, error) {
			nombre := "Default"
			if input.Nombre != nil {
				nombre = *input.Nombre
			}
			return &tenant.Tenant{ID: id, Nombre: nombre}, nil
		},
	}
	svc := tenant.NuevoService(store)
	nombre := "Bar Nuevo"
	res, err := svc.Actualizar(context.Background(), "t-1", tenant.ActualizarTenantInput{
		Nombre: &nombre,
	})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if res.Nombre != "Bar Nuevo" {
		t.Errorf("got %s, want Bar Nuevo", res.Nombre)
	}
}

func TestActualizar_PersonalizacionAvanzadaRequierePro(t *testing.T) {
	color := "#12AB34"
	store := &mockStore{
		obtenerEstadoCuotasFn: func(ctx context.Context, tenantID string) (*tenant.EstadoCuotas, error) {
			return &tenant.EstadoCuotas{Plan: tenant.PlanFree}, nil
		},
	}
	svc := tenant.NuevoService(store)
	_, err := svc.Actualizar(context.Background(), "t-1", tenant.ActualizarTenantInput{ColorCategoria: &color})
	if !errors.Is(err, tenant.ErrPlanRequired) {
		t.Fatalf("se esperaba ErrPlanRequired, obtenido %v", err)
	}
}

func TestActualizar_PersonalizacionAvanzadaPro(t *testing.T) {
	color := "#12AB34"
	store := &mockStore{
		obtenerEstadoCuotasFn: func(ctx context.Context, tenantID string) (*tenant.EstadoCuotas, error) {
			return &tenant.EstadoCuotas{Plan: tenant.PlanPro}, nil
		},
		actualizarFn: func(ctx context.Context, id string, input tenant.ActualizarTenantInput) (*tenant.Tenant, error) {
			return &tenant.Tenant{ID: id, ColorCategoria: input.ColorCategoria}, nil
		},
	}
	svc := tenant.NuevoService(store)
	resultado, err := svc.Actualizar(context.Background(), "t-1", tenant.ActualizarTenantInput{ColorCategoria: &color})
	if err != nil || resultado.ColorCategoria == nil || *resultado.ColorCategoria != color {
		t.Fatalf("se esperaba personalización Pro persistida, resultado=%+v error=%v", resultado, err)
	}
}

type verificadorGoogleFake struct {
	identidad *auth.IdentidadGoogle
	err       error
}

func (v verificadorGoogleFake) VerificarIdentidadGoogle(ctx context.Context, credencial string) (*auth.IdentidadGoogle, error) {
	return v.identidad, v.err
}

func TestCrear_ConGoogle_UsaIdentidadVerificada(t *testing.T) {
	var inputGuardado tenant.OnboardingInput
	store := &mockStore{
		crearFn: func(ctx context.Context, input tenant.OnboardingInput) (*tenant.Tenant, error) {
			inputGuardado = input
			return &tenant.Tenant{ID: "t-1"}, nil
		},
	}
	svc := tenant.NuevoServiceConGoogle(store, verificadorGoogleFake{
		identidad: &auth.IdentidadGoogle{Subject: "google-sub-1", Email: "admin@mibar.com", EmailVerificado: true},
	})

	_, err := svc.Crear(context.Background(), tenant.OnboardingInput{
		Nombre:           "Mi Bar",
		Slug:             "mi-bar",
		EmailAdmin:       "otro@correo.com",
		NombreAdmin:      "Carlos",
		GoogleCredential: "credencial-google",
	})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if inputGuardado.EmailAdmin != "admin@mibar.com" {
		t.Errorf("EmailAdmin: got %q, want el email verificado por Google", inputGuardado.EmailAdmin)
	}
	if inputGuardado.GoogleSub != "google-sub-1" {
		t.Errorf("GoogleSub: got %q, want %q", inputGuardado.GoogleSub, "google-sub-1")
	}
	if inputGuardado.PasswordHash != "" || inputGuardado.GoogleCredential != "" {
		t.Fatal("el registro con Google no debe guardar contraseña ni la credencial")
	}
}

func TestCrear_ConGoogle_CredencialInvalida_Error(t *testing.T) {
	svc := tenant.NuevoServiceConGoogle(&mockStore{}, verificadorGoogleFake{err: auth.ErrCredencialGoogleInvalida})
	_, err := svc.Crear(context.Background(), tenant.OnboardingInput{
		Nombre:           "Mi Bar",
		Slug:             "mi-bar",
		GoogleCredential: "credencial-vencida",
	})
	if !errors.Is(err, tenant.ErrValidation) {
		t.Fatalf("se esperaba ErrValidation, obtenido: %v", err)
	}
}

func TestCrear_ConGoogle_SinVerificador_Error(t *testing.T) {
	svc := tenant.NuevoService(&mockStore{})
	_, err := svc.Crear(context.Background(), tenant.OnboardingInput{
		Nombre:           "Mi Bar",
		Slug:             "mi-bar",
		GoogleCredential: "credencial-google",
	})
	if !errors.Is(err, tenant.ErrValidation) {
		t.Fatalf("se esperaba ErrValidation, obtenido: %v", err)
	}
}

func TestSlugDisponible(t *testing.T) {
	var consultado string
	svc := tenant.NuevoService(&mockStore{
		slugEnUsoFn: func(ctx context.Context, slug string) (bool, error) {
			consultado = slug
			return slug == "bajo-limonero", nil
		},
	})

	disponible, err := svc.SlugDisponible(context.Background(), "  Bajo-Limonero ")
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if consultado != "bajo-limonero" {
		t.Errorf("slug consultado: got %q, want %q", consultado, "bajo-limonero")
	}
	if disponible {
		t.Error("un slug en uso no debe figurar como disponible")
	}

	disponible, err = svc.SlugDisponible(context.Background(), "casa-clara")
	if err != nil || !disponible {
		t.Fatalf("se esperaba disponible, obtenido disponible=%v err=%v", disponible, err)
	}
}

func TestSlugDisponible_FormatoInvalido(t *testing.T) {
	svc := tenant.NuevoService(&mockStore{})
	for _, slug := range []string{"", "con espacios", "-guion", "acentuado-ñ"} {
		if _, err := svc.SlugDisponible(context.Background(), slug); !errors.Is(err, tenant.ErrValidation) {
			t.Errorf("slug %q: se esperaba ErrValidation, obtenido %v", slug, err)
		}
	}
}
func TestSlugReservado_NoDisponible(t *testing.T) {
	svc := tenant.NuevoService(&mockStore{})
	for _, reservado := range []string{"superadmin", "ADMIN", "platform", "kds", "mesa", "dashboard", "api", "login"} {
		disponible, err := svc.SlugDisponible(context.Background(), reservado)
		if err != nil {
			t.Errorf("slug reservado %q dio error: %v", reservado, err)
		}
		if disponible {
			t.Errorf("slug reservado %q no debe figurar disponible", reservado)
		}
	}
}

func TestCrear_SlugReservado_Error(t *testing.T) {
	svc := tenant.NuevoService(&mockStore{})
	_, err := svc.Crear(context.Background(), tenant.OnboardingInput{
		Nombre:     "Super Admin Bar",
		Slug:       "superadmin",
		EmailAdmin: "admin@bar.com",
		Password:   "clave1234",
	})
	if !errors.Is(err, tenant.ErrValidation) {
		t.Fatalf("se esperaba ErrValidation por slug reservado, obtenido: %v", err)
	}
}
