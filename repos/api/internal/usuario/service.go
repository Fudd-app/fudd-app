package usuario

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
)

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_.-]{3,50}$`)

type Service struct {
	store Store
	auth  *auth.Service
	email auth.EmailSender
}

func NuevoService(s Store, authSvc *auth.Service, email auth.EmailSender) *Service {
	return &Service{store: s, auth: authSvc, email: email}
}

func NormalizarUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(username), "@")))
}

func (svc *Service) CrearStaff(ctx context.Context, actor *auth.Claims, input CrearUsuarioStaffInput) (*Usuario, error) {
	if actor == nil || (actor.Rol != auth.RolDuenio && actor.Rol != auth.RolEncargado) {
		return nil, ErrForbidden
	}

	input.Nombre = strings.TrimSpace(input.Nombre)
	if input.Nombre == "" {
		return nil, fmt.Errorf("%w: el nombre completo es requerido", ErrValidation)
	}

	input.Username = NormalizarUsername(input.Username)
	if input.Username == "" || !usernameRegex.MatchString(input.Username) {
		return nil, fmt.Errorf("%w: el nombre de usuario debe tener entre 3 y 50 caracteres alfanuméricos (letras, números, guion o punto)", ErrValidation)
	}

	input.Rol = strings.ToLower(strings.TrimSpace(input.Rol))
	if input.Rol != auth.RolEncargado && input.Rol != auth.RolMozo && input.Rol != auth.RolCocinero {
		return nil, fmt.Errorf("%w: rol operativo inválido (debe ser encargado, mozo o cocina)", ErrValidation)
	}

	if actor.Rol == auth.RolEncargado {
		if input.Rol == auth.RolEncargado {
			return nil, ErrForbidden
		}
		suc := actor.SucursalID
		input.SucursalID = &suc
	}

	input.Email = auth.NormalizarEmail(input.Email)
	if input.Email != "" && !strings.Contains(input.Email, "@") {
		return nil, fmt.Errorf("%w: el email ingresado no es válido", ErrValidation)
	}

	credStore, ok := svc.store.(StaffCredentialStore)
	if !ok {
		return nil, errors.New("store sin soporte para alta por credenciales")
	}

	if input.SucursalID != nil {
		trimmed := strings.TrimSpace(*input.SucursalID)
		if trimmed == "" {
			input.SucursalID = nil
		} else {
			input.SucursalID = &trimmed
		}
	}
	if input.SucursalID == nil {
		if defID, err := credStore.ObtenerPrimeraSucursalID(ctx, actor.TenantID); err == nil && defID != "" {
			input.SucursalID = &defID
		}
	}
	if input.SucursalID != nil && *input.SucursalID != "" {
		pertenece, err := svc.store.SucursalPertenece(ctx, actor.TenantID, *input.SucursalID)
		if err != nil {
			return nil, err
		}
		if !pertenece {
			return nil, ErrForbidden
		}
	}

	hash, err := auth.HashPasswordStaff(input.Password)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrValidation, err.Error())
	}
	input.PasswordHash = hash
	input.Password = ""

	return credStore.CrearStaff(ctx, actor.TenantID, input)
}

func (svc *Service) Invitar(ctx context.Context, actor *auth.Claims, input InvitarInput) (*InvitacionCreada, error) {
	if actor == nil || (actor.Rol != auth.RolDuenio && actor.Rol != auth.RolEncargado) {
		return nil, ErrForbidden
	}
	input.Email = auth.NormalizarEmail(input.Email)
	input.Rol = strings.ToLower(strings.TrimSpace(input.Rol))
	if input.Email == "" || !strings.Contains(input.Email, "@") {
		return nil, fmt.Errorf("%w: email inválido", ErrValidation)
	}
	if input.Rol != auth.RolEncargado && input.Rol != auth.RolMozo && input.Rol != auth.RolCocinero {
		return nil, fmt.Errorf("%w: rol inválido", ErrValidation)
	}

	if actor.Rol == auth.RolEncargado {
		if input.Rol == auth.RolEncargado {
			return nil, ErrForbidden
		}
		input.SucursalID = actor.SucursalID
	}
	if input.SucursalID == "" {
		return nil, fmt.Errorf("%w: sucursal requerida", ErrValidation)
	}
	pertenece, err := svc.store.SucursalPertenece(ctx, actor.TenantID, input.SucursalID)
	if err != nil {
		return nil, err
	}
	if !pertenece {
		return nil, ErrForbidden
	}

	token, tokenHash, err := nuevoToken()
	if err != nil {
		return nil, err
	}
	i, err := svc.store.CrearInvitacion(ctx, actor.TenantID, input.SucursalID, input.Email, input.Rol, tokenHash, actor.UsuarioID, time.Now().Add(48*time.Hour))
	if err != nil {
		return nil, err
	}
	link := linkInvitacion(token)
	emailEnviado, advertencia := svc.enviarInvitacion(ctx, input.Email, link)
	respuesta := &InvitacionCreada{
		Invitacion:       *i,
		Link:             link,
		EmailEnviado:     emailEnviado,
		AdvertenciaEmail: advertencia,
	}
	return respuesta, nil
}

func (svc *Service) Listar(ctx context.Context, actor *auth.Claims) (*Equipo, error) {
	if actor == nil || (actor.Rol != auth.RolDuenio && actor.Rol != auth.RolEncargado) {
		return nil, ErrForbidden
	}
	alcance := ""
	if actor.Rol == auth.RolEncargado {
		alcance = actor.SucursalID
	}
	usuarios, err := svc.store.ListarUsuarios(ctx, actor.TenantID, alcance)
	if err != nil {
		return nil, err
	}
	invitaciones, err := svc.store.ListarInvitaciones(ctx, actor.TenantID, alcance)
	if err != nil {
		return nil, err
	}
	return &Equipo{Usuarios: usuarios, Invitaciones: invitaciones}, nil
}

func (svc *Service) Actualizar(ctx context.Context, actor *auth.Claims, id string, input ActualizarUsuarioInput) (*Usuario, error) {
	target, err := svc.targetAutorizado(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	nombre, estado := target.Nombre, target.Estado
	if input.Nombre != nil {
		nombre = strings.TrimSpace(*input.Nombre)
		if nombre == "" {
			return nil, fmt.Errorf("%w: nombre vacío", ErrValidation)
		}
		input.Nombre = &nombre
	}
	if input.Estado != nil {
		estado = strings.ToLower(strings.TrimSpace(*input.Estado))
		if estado != "activo" && estado != "bloqueado" && estado != "archivado" {
			return nil, fmt.Errorf("%w: estado inválido", ErrValidation)
		}
		input.Estado = &estado
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
		if rol != auth.RolEncargado && rol != auth.RolMozo && rol != auth.RolCocinero {
			return nil, fmt.Errorf("%w: rol inválido", ErrValidation)
		}
		if actor.Rol == auth.RolEncargado && rol == auth.RolEncargado {
			return nil, ErrForbidden
		}
		input.Rol = &rol
	}
	if input.SucursalID != nil {
		trimmed := strings.TrimSpace(*input.SucursalID)
		if trimmed != "" {
			if actor.Rol == auth.RolEncargado && trimmed != actor.SucursalID {
				return nil, ErrForbidden
			}
			pertenece, err := svc.store.SucursalPertenece(ctx, actor.TenantID, trimmed)
			if err != nil {
				return nil, err
			}
			if !pertenece {
				return nil, ErrForbidden
			}
		}
		input.SucursalID = &trimmed
	}

	if credStore, ok := svc.store.(StaffCredentialStore); ok {
		return credStore.ActualizarUsuarioDetalle(ctx, actor.TenantID, id, input)
	}
	return svc.store.ActualizarUsuario(ctx, actor.TenantID, id, nombre, estado)
}

func (svc *Service) ResetearPassword(ctx context.Context, actor *auth.Claims, id string, nuevaPassword string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: id de usuario requerido", ErrValidation)
	}
	if _, err := svc.targetAutorizado(ctx, actor, id); err != nil {
		return err
	}
	hash, err := auth.HashPasswordStaff(nuevaPassword)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrValidation, err.Error())
	}
	credStore, ok := svc.store.(StaffCredentialStore)
	if !ok {
		return errors.New("store sin soporte para reseteo de contraseña")
	}
	return credStore.ResetearPassword(ctx, actor.TenantID, id, hash)
}

func (svc *Service) CambiarEstado(ctx context.Context, actor *auth.Claims, id string, nuevoEstado string) (*Usuario, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: id de usuario requerido", ErrValidation)
	}
	if actor == nil || id == actor.UsuarioID {
		return nil, ErrForbidden
	}
	target, err := svc.targetAutorizado(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	estado := strings.ToLower(strings.TrimSpace(nuevoEstado))
	if estado != "activo" && estado != "bloqueado" && estado != "archivado" {
		return nil, fmt.Errorf("%w: estado inválido (debe ser activo, bloqueado o archivado)", ErrValidation)
	}
	if credStore, ok := svc.store.(StaffCredentialStore); ok {
		return credStore.CambiarEstado(ctx, actor.TenantID, id, estado)
	}
	return svc.store.ActualizarUsuario(ctx, actor.TenantID, id, target.Nombre, estado)
}

func (svc *Service) Eliminar(ctx context.Context, actor *auth.Claims, id string) error {
	if actor == nil || id == actor.UsuarioID {
		return ErrForbidden
	}
	if _, err := svc.targetAutorizado(ctx, actor, id); err != nil {
		return err
	}
	return svc.store.EliminarUsuario(ctx, actor.TenantID, id)
}

func (svc *Service) targetAutorizado(ctx context.Context, actor *auth.Claims, id string) (*Usuario, error) {
	if actor == nil || (actor.Rol != auth.RolDuenio && actor.Rol != auth.RolEncargado) {
		return nil, ErrForbidden
	}
	target, err := svc.store.ObtenerUsuario(ctx, actor.TenantID, id)
	if err != nil {
		return nil, err
	}
	if target.Rol == auth.RolDuenio {
		return nil, ErrForbidden
	}
	if actor.Rol == auth.RolEncargado {
		if target.SucursalID == nil || *target.SucursalID != actor.SucursalID || (target.Rol != auth.RolMozo && target.Rol != auth.RolCocinero) {
			return nil, ErrForbidden
		}
	}
	return target, nil
}

func (svc *Service) Cancelar(ctx context.Context, actor *auth.Claims, id string) error {
	if actor == nil || (actor.Rol != auth.RolDuenio && actor.Rol != auth.RolEncargado) {
		return ErrForbidden
	}
	alcance := ""
	if actor.Rol == auth.RolEncargado {
		alcance = actor.SucursalID
	}
	return svc.store.CancelarInvitacion(ctx, actor.TenantID, id, alcance)
}

func (svc *Service) Reenviar(ctx context.Context, actor *auth.Claims, id string) (*InvitacionCreada, error) {
	if actor == nil || (actor.Rol != auth.RolDuenio && actor.Rol != auth.RolEncargado) {
		return nil, ErrForbidden
	}
	token, tokenHash, err := nuevoToken()
	if err != nil {
		return nil, err
	}
	alcance := ""
	if actor.Rol == auth.RolEncargado {
		alcance = actor.SucursalID
	}
	i, err := svc.store.ReenviarInvitacion(ctx, actor.TenantID, id, alcance, tokenHash, actor.UsuarioID, time.Now().Add(48*time.Hour))
	if err != nil {
		return nil, err
	}
	link := linkInvitacion(token)
	emailEnviado, advertencia := svc.enviarInvitacion(ctx, i.Email, link)
	respuesta := &InvitacionCreada{
		Invitacion:       *i,
		Link:             link,
		EmailEnviado:     emailEnviado,
		AdvertenciaEmail: advertencia,
	}
	return respuesta, nil
}

func (svc *Service) enviarInvitacion(ctx context.Context, email, link string) (bool, string) {
	if svc.email == nil {
		return false, "La invitación se creó, pero no hay un proveedor de email configurado. Compartí el enlace manualmente."
	}
	if _, esSimulado := svc.email.(*auth.LogEmailSender); esSimulado {
		_ = auth.EnviarInvitacion(ctx, svc.email, email, link)
		return false, "La invitación se creó, pero el envío de email está en modo local. Compartí el enlace manualmente."
	}
	if err := auth.EnviarInvitacion(ctx, svc.email, email, link); err != nil {
		slog.ErrorContext(ctx, "no se pudo enviar invitación", "email", email, "err", err)
		return false, "La invitación se creó, pero no pudimos enviar el email. Podés copiar el enlace y compartirlo manualmente."
	}
	return true, ""
}

// VistaPrevia no cambia el estado ni consume el token.
func (svc *Service) VistaPrevia(ctx context.Context, token string) (*Invitacion, error) {
	if strings.TrimSpace(token) == "" {
		return nil, ErrInvitacionNoValida
	}
	i, err := svc.store.ObtenerInvitacionPorToken(ctx, hashToken(token))
	if err != nil {
		return nil, err
	}
	if i.Estado != "pendiente" {
		return nil, ErrInvitacionNoValida
	}
	if !i.ExpiresAt.After(time.Now()) {
		return nil, ErrInvitacionExpirada
	}
	return i, nil
}

func (svc *Service) Aceptar(ctx context.Context, token string, input AceptarInvitacionInput) (*Usuario, error) {
	i, err := svc.VistaPrevia(ctx, token)
	if err != nil {
		return nil, err
	}
	if svc.auth == nil {
		return nil, errors.New("autenticación Google no disponible")
	}
	identidad, err := svc.auth.VerificarIdentidadGoogle(ctx, input.CredencialGoogle)
	if err != nil {
		return nil, err
	}
	if auth.NormalizarEmail(identidad.Email) != auth.NormalizarEmail(i.Email) {
		return nil, ErrEmailGoogleDistinto
	}
	nombre := strings.TrimSpace(input.Nombre)
	if nombre == "" {
		nombre = strings.TrimSpace(identidad.Nombre)
	}
	if nombre == "" {
		nombre = strings.Split(i.Email, "@")[0]
	}
	return svc.store.AceptarInvitacion(ctx, hashToken(token), identidad.Subject, nombre, time.Now())
}

func nuevoToken() (string, string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	token := hex.EncodeToString(b)
	return token, hashToken(token), nil
}

func hashToken(token string) string {
	suma := sha256.Sum256([]byte(token))
	return hex.EncodeToString(suma[:])
}

func linkInvitacion(token string) string {
	base := strings.TrimRight(os.Getenv("APP_URL"), "/")
	if base == "" {
		base = "http://localhost:3000"
	}
	return base + "/invitacion/" + token
}
