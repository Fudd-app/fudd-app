package usuario_test

import (
	"context"
	"errors"
	"testing"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/usuario"
)

type mockStore struct {
	crearFn            func(ctx context.Context, tenantID string, input usuario.UsuarioInput) (*usuario.Usuario, error)
	crearStaffFn       func(ctx context.Context, tenantID string, input usuario.CrearUsuarioStaffInput) (*usuario.Usuario, error)
	listarFn           func(ctx context.Context, tenantID string) ([]usuario.Usuario, error)
	actualizarFn       func(ctx context.Context, id string, tenantID string, input usuario.ActualizarUsuarioInput) (*usuario.Usuario, error)
	resetearPasswordFn func(ctx context.Context, id string, tenantID string, passwordHash string) error
	cambiarEstadoFn    func(ctx context.Context, id string, tenantID string, nuevoEstado string) (*usuario.Usuario, error)
	eliminarFn         func(ctx context.Context, id string, tenantID string) error
	crearMagicTokenFn  func(ctx context.Context, usuarioID string) (string, error)
}

func (m *mockStore) Crear(ctx context.Context, tenantID string, input usuario.UsuarioInput) (*usuario.Usuario, error) {
	return m.crearFn(ctx, tenantID, input)
}
func (m *mockStore) CrearStaff(ctx context.Context, tenantID string, input usuario.CrearUsuarioStaffInput) (*usuario.Usuario, error) {
	if m.crearStaffFn != nil {
		return m.crearStaffFn(ctx, tenantID, input)
	}
	return nil, nil
}
func (m *mockStore) Listar(ctx context.Context, tenantID string) ([]usuario.Usuario, error) {
	return m.listarFn(ctx, tenantID)
}
func (m *mockStore) Actualizar(ctx context.Context, id string, tenantID string, input usuario.ActualizarUsuarioInput) (*usuario.Usuario, error) {
	if m.actualizarFn != nil {
		return m.actualizarFn(ctx, id, tenantID, input)
	}
	return nil, nil
}
func (m *mockStore) ResetearPassword(ctx context.Context, id string, tenantID string, passwordHash string) error {
	if m.resetearPasswordFn != nil {
		return m.resetearPasswordFn(ctx, id, tenantID, passwordHash)
	}
	return nil
}
func (m *mockStore) CambiarEstado(ctx context.Context, id string, tenantID string, nuevoEstado string) (*usuario.Usuario, error) {
	if m.cambiarEstadoFn != nil {
		return m.cambiarEstadoFn(ctx, id, tenantID, nuevoEstado)
	}
	return nil, nil
}
func (m *mockStore) Eliminar(ctx context.Context, id string, tenantID string) error {
	return m.eliminarFn(ctx, id, tenantID)
}
func (m *mockStore) CrearMagicToken(ctx context.Context, usuarioID string) (string, error) {
	return m.crearMagicTokenFn(ctx, usuarioID)
}

func TestInvitarUsuario_Exitoso(t *testing.T) {
	store := &mockStore{
		crearFn: func(ctx context.Context, tenantID string, input usuario.UsuarioInput) (*usuario.Usuario, error) {
			return &usuario.Usuario{ID: "u-1", TenantID: tenantID, Email: input.Email, Nombre: input.Nombre, Rol: input.Rol, Estado: "activo"}, nil
		},
		crearMagicTokenFn: func(ctx context.Context, usuarioID string) (string, error) {
			return "tok-12345", nil
		},
	}
	svc := usuario.NuevoService(store)
	result, err := svc.Invitar(context.Background(), "t-1", usuario.UsuarioInput{
		Nombre: "Juan Perez",
		Email:  "juan@mibar.com",
		Rol:    "mozo",
	})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if result.Usuario.Nombre != "Juan Perez" {
		t.Errorf("got %q, want %q", result.Usuario.Nombre, "Juan Perez")
	}
	if result.MagicLink == "" {
		t.Error("se esperaba link de invitacion generado")
	}
}

func TestInvitarUsuario_ValidationErrors(t *testing.T) {
	svc := usuario.NuevoService(&mockStore{})

	// 1. Nombre vacío
	_, err := svc.Invitar(context.Background(), "t-1", usuario.UsuarioInput{
		Email: "juan@mibar.com",
		Rol:   "mozo",
	})
	if err == nil || !errors.Is(err, usuario.ErrValidation) {
		t.Errorf("esperaba error ErrValidation por nombre vacío, got: %v", err)
	}

	// 2. Email vacío
	_, err = svc.Invitar(context.Background(), "t-1", usuario.UsuarioInput{
		Nombre: "Juan",
		Rol:    "mozo",
	})
	if err == nil || !errors.Is(err, usuario.ErrValidation) {
		t.Errorf("esperaba error ErrValidation por email vacío, got: %v", err)
	}

	// 3. Rol inválido
	_, err = svc.Invitar(context.Background(), "t-1", usuario.UsuarioInput{
		Nombre: "Juan",
		Email:  "juan@mibar.com",
		Rol:    "gerente_general",
	})
	if err == nil || !errors.Is(err, usuario.ErrValidation) {
		t.Errorf("esperaba error ErrValidation por rol inválido, got: %v", err)
	}
}

func TestActualizarUsuario_Exitoso(t *testing.T) {
	store := &mockStore{
		actualizarFn: func(ctx context.Context, id string, tenantID string, input usuario.ActualizarUsuarioInput) (*usuario.Usuario, error) {
			rol := "mozo"
			if input.Rol != nil {
				rol = *input.Rol
			}
			return &usuario.Usuario{ID: id, TenantID: tenantID, Rol: rol}, nil
		},
	}
	svc := usuario.NuevoService(store)
	nuevoRol := "encargado"
	res, err := svc.Actualizar(context.Background(), "u-1", "t-1", usuario.ActualizarUsuarioInput{
		Rol: &nuevoRol,
	})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if res.Rol != "encargado" {
		t.Errorf("got %s, want encargado", res.Rol)
	}
}

func TestCrearStaff_ExitosoYAislamientoPorTenant(t *testing.T) {
	creadosPorTenant := make(map[string]map[string]bool)

	store := &mockStore{
		crearStaffFn: func(ctx context.Context, tenantID string, input usuario.CrearUsuarioStaffInput) (*usuario.Usuario, error) {
			if input.PasswordHash == "" || input.PasswordHash == "Panda721!" {
				t.Fatalf("se esperaba hash bcrypt y nunca texto plano, got %q", input.PasswordHash)
			}
			if creadosPorTenant[tenantID] == nil {
				creadosPorTenant[tenantID] = make(map[string]bool)
			}
			if creadosPorTenant[tenantID][input.Username] {
				return nil, usuario.ErrUsernameConflict
			}
			creadosPorTenant[tenantID][input.Username] = true

			uname := input.Username
			return &usuario.Usuario{
				ID:       "u-staff-1",
				TenantID: tenantID,
				Username: &uname,
				Nombre:   input.Nombre,
				Rol:      input.Rol,
				Estado:   "activo",
			}, nil
		},
	}

	svc := usuario.NuevoService(store)

	// 1. Crear en tenant A
	u1, err := svc.CrearStaff(context.Background(), "tenant-A", usuario.CrearUsuarioStaffInput{
		Nombre:   "Juan Mozo",
		Username: "@Mozo_Juan",
		Rol:      "mozo",
		Password: "Panda721!",
	})
	if err != nil {
		t.Fatalf("error inesperado creando staff en tenant-A: %v", err)
	}
	if u1.Username == nil || *u1.Username != "mozo_juan" {
		t.Fatalf("username normalizado inesperado: %v", u1.Username)
	}

	// 2. Duplicado en el mismo tenant A debe fallar con ErrUsernameConflict
	_, err = svc.CrearStaff(context.Background(), "tenant-A", usuario.CrearUsuarioStaffInput{
		Nombre:   "Otro Juan",
		Username: "mozo_juan",
		Rol:      "mozo",
		Password: "Panda721!",
	})
	if !errors.Is(err, usuario.ErrUsernameConflict) {
		t.Fatalf("esperaba ErrUsernameConflict en el mismo tenant, got: %v", err)
	}

	// 3. Mismo username en tenant B está permitido
	_, err = svc.CrearStaff(context.Background(), "tenant-B", usuario.CrearUsuarioStaffInput{
		Nombre:   "Juan de Tenant B",
		Username: "mozo_juan",
		Rol:      "cocina",
		Password: "Panda721!",
	})
	if err != nil {
		t.Fatalf("se debía permitir mismo username en tenant-B, got: %v", err)
	}
}

func TestCambiarEstadoYResetearPassword(t *testing.T) {
	var hashRecibido string
	store := &mockStore{
		cambiarEstadoFn: func(ctx context.Context, id string, tenantID string, nuevoEstado string) (*usuario.Usuario, error) {
			return &usuario.Usuario{ID: id, TenantID: tenantID, Estado: nuevoEstado}, nil
		},
		resetearPasswordFn: func(ctx context.Context, id string, tenantID string, passwordHash string) error {
			hashRecibido = passwordHash
			return nil
		},
	}
	svc := usuario.NuevoService(store)

	u, err := svc.CambiarEstado(context.Background(), "u-1", "t-1", "bloqueado")
	if err != nil {
		t.Fatalf("error inesperado cambiando estado: %v", err)
	}
	if u.Estado != "bloqueado" {
		t.Fatalf("estado esperado bloqueado, got %q", u.Estado)
	}

	if err := svc.ResetearPassword(context.Background(), "u-1", "t-1", "NuevaClave88"); err != nil {
		t.Fatalf("error reseteando password: %v", err)
	}
	if hashRecibido == "" || hashRecibido == "NuevaClave88" {
		t.Fatal("se esperaba password hasheada con bcrypt")
	}
}
