package usuario

import (
	"errors"
	"time"
)

var (
	ErrNotFound            = errors.New("usuario no encontrado")
	ErrValidation          = errors.New("error de validación")
	ErrEmailConflict       = errors.New("el email ya pertenece al equipo")
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
	Email          string    `json:"email"`
	Nombre         string    `json:"nombre"`
	Rol            string    `json:"rol"`
	Estado         string    `json:"estado"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
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
	Nombre *string `json:"nombre,omitempty"`
	Estado *string `json:"estado,omitempty"`
}
