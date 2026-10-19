package usuario

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
)

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_.-]{3,50}$`)

type Service struct {
	store Store
}

func NuevoService(s Store) *Service { return &Service{store: s} }

func validarRolStaff(rol string) bool {
	return rol == "admin" || rol == "encargado" || rol == "mozo" || rol == "cocina"
}

func NormalizarUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(username), "@")))
}

func (svc *Service) Invitar(ctx context.Context, tenantID string, input UsuarioInput) (*UsuarioInvitacionResponse, error) {
	input.Nombre = strings.TrimSpace(input.Nombre)
	if input.Nombre == "" {
		return nil, fmt.Errorf("%w: nombre requerido", ErrValidation)
	}
	if strings.TrimSpace(input.Email) == "" {
		return nil, fmt.Errorf("%w: email requerido", ErrValidation)
	}
	input.Rol = strings.ToLower(strings.TrimSpace(input.Rol))
	if !validarRolStaff(input.Rol) {
		return nil, fmt.Errorf("%w: rol inválido (debe ser admin, encargado, mozo o cocina)", ErrValidation)
	}

	// Mismo criterio que en onboarding: el email es la clave de login.
	input.Email = auth.NormalizarEmail(input.Email)

	u, err := svc.store.Crear(ctx, tenantID, input)
	if err != nil {
		return nil, err
	}

	token, err := svc.store.CrearMagicToken(ctx, u.ID)
	if err != nil {
		slog.ErrorContext(ctx, "error creando token para invitacion", "usuario_id", u.ID, "err", err)
		// No fallamos toda la operacion si falla la creacion del token de invitacion
		return &UsuarioInvitacionResponse{Usuario: u}, nil
	}

	link := auth.ConstruirLinkVerificacion(token)
	slog.InfoContext(ctx, "magic link de invitacion generado", "usuario", u.Email, "url", link)

	return &UsuarioInvitacionResponse{
		Usuario:   u,
		MagicLink: link,
	}, nil
}

func (svc *Service) CrearStaff(ctx context.Context, tenantID string, input CrearUsuarioStaffInput) (*Usuario, error) {
	input.Nombre = strings.TrimSpace(input.Nombre)
	if input.Nombre == "" {
		return nil, fmt.Errorf("%w: el nombre completo es requerido", ErrValidation)
	}

	input.Username = NormalizarUsername(input.Username)
	if input.Username == "" || !usernameRegex.MatchString(input.Username) {
		return nil, fmt.Errorf("%w: el nombre de usuario debe tener entre 3 y 50 caracteres alfanuméricos (letras, números, guion o punto)", ErrValidation)
	}

	input.Rol = strings.ToLower(strings.TrimSpace(input.Rol))
	if input.Rol != "encargado" && input.Rol != "mozo" && input.Rol != "cocina" {
		return nil, fmt.Errorf("%w: rol operativo inválido (debe ser encargado, mozo o cocina)", ErrValidation)
	}

	input.Email = auth.NormalizarEmail(input.Email)
	if input.Email != "" && !strings.Contains(input.Email, "@") {
		return nil, fmt.Errorf("%w: el email ingresado no es válido", ErrValidation)
	}

	if input.SucursalID != nil {
		trimmed := strings.TrimSpace(*input.SucursalID)
		if trimmed == "" {
			input.SucursalID = nil
		} else {
			input.SucursalID = &trimmed
		}
	}

	hash, err := auth.HashPasswordStaff(input.Password)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrValidation, err.Error())
	}
	input.PasswordHash = hash
	input.Password = ""

	return svc.store.CrearStaff(ctx, tenantID, input)
}

func (svc *Service) Listar(ctx context.Context, tenantID string) ([]Usuario, error) {
	return svc.store.Listar(ctx, tenantID)
}

func (svc *Service) Actualizar(ctx context.Context, id string, tenantID string, input ActualizarUsuarioInput) (*Usuario, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: id de usuario requerido", ErrValidation)
	}
	if input.Nombre != nil {
		trimmed := strings.TrimSpace(*input.Nombre)
		if trimmed == "" {
			return nil, fmt.Errorf("%w: el nombre no puede estar vacío", ErrValidation)
		}
		input.Nombre = &trimmed
	}
	if input.Username != nil {
		norm := NormalizarUsername(*input.Username)
		if norm != "" && !usernameRegex.MatchString(norm) {
			return nil, fmt.Errorf("%w: nombre de usuario inválido", ErrValidation)
		}
		input.Username = &norm
	}
	if input.Email != nil {
		normEmail := auth.NormalizarEmail(*input.Email)
		if normEmail != "" && !strings.Contains(normEmail, "@") {
			return nil, fmt.Errorf("%w: email inválido", ErrValidation)
		}
		input.Email = &normEmail
	}
	if input.Rol != nil {
		rol := strings.ToLower(strings.TrimSpace(*input.Rol))
		if !validarRolStaff(rol) {
			return nil, fmt.Errorf("%w: rol inválido (debe ser admin, encargado, mozo o cocina)", ErrValidation)
		}
		input.Rol = &rol
	}

	return svc.store.Actualizar(ctx, id, tenantID, input)
}

func (svc *Service) ResetearPassword(ctx context.Context, id string, tenantID string, nuevaPassword string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: id de usuario requerido", ErrValidation)
	}
	hash, err := auth.HashPasswordStaff(nuevaPassword)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrValidation, err.Error())
	}
	return svc.store.ResetearPassword(ctx, id, tenantID, hash)
}

func (svc *Service) CambiarEstado(ctx context.Context, id string, tenantID string, nuevoEstado string) (*Usuario, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: id de usuario requerido", ErrValidation)
	}
	estado := strings.ToLower(strings.TrimSpace(nuevoEstado))
	if estado != "activo" && estado != "bloqueado" && estado != "archivado" {
		return nil, fmt.Errorf("%w: estado inválido (debe ser activo, bloqueado o archivado)", ErrValidation)
	}
	return svc.store.CambiarEstado(ctx, id, tenantID, estado)
}

func (svc *Service) Eliminar(ctx context.Context, id string, tenantID string) error {
	if id == "" {
		return fmt.Errorf("%w: id de usuario requerido", ErrValidation)
	}
	return svc.store.Eliminar(ctx, id, tenantID)
}
