package usuario

import (
	"errors"
	"time"
)

var (
	ErrNotFound            = errors.New("usuario no encontrado")
	ErrValidation          = errors.New("error de validación")
	ErrEmailConflict       = errors.New("el email ya pertenece al equipo")
	ErrUsernameConflict    = errors.New("el nombre de usuario ya está en uso en este comercio")
	ErrForbidden           = errors.New("sin permisos para gestionar este miembro")
	ErrInvitacionNoValida  = errors.New("invitación inválida")
	ErrInvitacionExpirada  = errors.New("invitación expirada")
	ErrEmailGoogleDistinto = errors.New("el correo de Google no coincide con la invitación")
)

type Usuario struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenant_id"`
	SucursalID     *string   `json:"sucursal_id,omitempty"`
	SucursalNombre *string   `json:"sucursal_nombre,omitempty"`
	Username       *string   `json:"username,omitempty"`
	Email          string    `json:"email"`
	Nombre         string    `json:"nombre"`
	Rol            string    `json:"rol"`
	Estado         string    `json:"estado"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type CrearUsuarioStaffInput struct {
	Nombre       string  `json:"nombre"`
	Username     string  `json:"username"`
	Email        string  `json:"email,omitempty"`
	Rol          string  `json:"rol"`
	SucursalID   *string `json:"sucursal_id,omitempty"`
	Password     string  `json:"password"`
	PasswordHash string  `json:"-"`
}

type ResetPasswordInput struct {
	NuevaPassword string `json:"nueva_password"`
}

type CambiarEstadoUsuarioInput struct {
	Estado string `json:"estado"`
}

type Invitacion struct {
	ID             string     `json:"id"`
	TenantID       string     `json:"tenant_id,omitempty"`
	TenantNombre   string     `json:"negocio"`
	TenantSlug     string     `json:"slug"`
	SucursalID     string     `json:"sucursal_id"`
	SucursalNombre string     `json:"sucursal"`
	Email          string     `json:"email"`
	Rol            string     `json:"rol"`
	Estado         string     `json:"estado"`
	ExpiresAt      time.Time  `json:"expires_at"`
	UsedAt         *time.Time `json:"used_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

type InvitarInput struct {
	Email      string `json:"email"`
	Rol        string `json:"rol"`
	SucursalID string `json:"sucursal_id,omitempty"`
}

type InvitacionCreada struct {
	Invitacion       Invitacion `json:"invitacion"`
	Link             string     `json:"invitacion_link"`
	EmailEnviado     bool       `json:"email_enviado"`
	AdvertenciaEmail string     `json:"advertencia_email,omitempty"`
}

type AceptarInvitacionInput struct {
	CredencialGoogle string `json:"credential"`
	Nombre           string `json:"nombre"`
}

type Equipo struct {
	Usuarios     []Usuario    `json:"usuarios"`
	Invitaciones []Invitacion `json:"invitaciones"`
}

type ActualizarUsuarioInput struct {
	Nombre     *string `json:"nombre,omitempty"`
	Username   *string `json:"username,omitempty"`
	Email      *string `json:"email,omitempty"`
	Rol        *string `json:"rol,omitempty"`
	SucursalID *string `json:"sucursal_id,omitempty"`
	Estado     *string `json:"estado,omitempty"`
}
