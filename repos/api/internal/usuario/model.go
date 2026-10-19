package usuario

import (
	"errors"
	"time"
)

var (
	ErrNotFound         = errors.New("usuario no encontrado")
	ErrValidation       = errors.New("error de validación")
	ErrEmailConflict    = errors.New("el email ya está registrado")
	ErrUsernameConflict = errors.New("el nombre de usuario ya está en uso en este comercio")
)

type Usuario struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenant_id"`
	SucursalID     *string   `json:"sucursal_id,omitempty"`
	SucursalNombre *string   `json:"sucursal_nombre,omitempty"`
	Username       *string   `json:"username,omitempty"`
	Email          string    `json:"email"`
	Nombre         string    `json:"nombre"`
	Rol            string    `json:"rol"`    // 'admin', 'encargado', 'mozo', 'cocina'
	Estado         string    `json:"estado"` // 'activo', 'bloqueado', 'archivado'
	CreatedAt      time.Time `json:"created_at"`
}

type UsuarioInput struct {
	Nombre     string  `json:"nombre"`
	Email      string  `json:"email"`
	Rol        string  `json:"rol"`
	SucursalID *string `json:"sucursal_id,omitempty"`
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
	Estado string `json:"estado"` // 'activo', 'bloqueado', 'archivado'
}

type UsuarioInvitacionResponse struct {
	Usuario   *Usuario `json:"usuario"`
	MagicLink string   `json:"magic_link,omitempty"`
}

type ActualizarUsuarioInput struct {
	Nombre     *string `json:"nombre,omitempty"`
	Username   *string `json:"username,omitempty"`
	Email      *string `json:"email,omitempty"`
	Rol        *string `json:"rol,omitempty"`
	SucursalID *string `json:"sucursal_id,omitempty"`
}
