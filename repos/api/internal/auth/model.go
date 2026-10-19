package auth

import (
	"context"
	"errors"
	"time"
)

var ErrUsuarioNoEncontrado = errors.New("usuario no encontrado")
var ErrUsuarioBloqueado = errors.New("usuario bloqueado")
var ErrSesionRevocada = errors.New("sesión revocada")

type MagicToken struct {
	ID        string
	UsuarioID string
	Token     string
	ExpiresAt time.Time
	UsedAt    *time.Time
}

type UsuarioAuth struct {
	ID             string
	TenantID       string
	Email          string
	Nombre         string
	Rol            string
	SucursalID     *string
	Estado         string
	SessionVersion int64
	GoogleSub      *string
	PasswordHash   *string
}

type Claims struct {
	UsuarioID      string `json:"usuario_id"`
	TenantID       string `json:"tenant_id"`
	Rol            string `json:"rol"`
	SucursalID     string `json:"sucursal_id,omitempty"`
	SessionVersion int64  `json:"session_version"`
}

const (
	RolDuenio    = "admin"
	RolEncargado = "encargado"
	RolMozo      = "mozo"
	RolCocinero  = "cocina"
)

type ctxKey string

const claimsKey ctxKey = "auth_claims"

func ClaimsFromContext(ctx context.Context) *Claims {
	c, _ := ctx.Value(claimsKey).(*Claims)
	return c
}

func ContextConClaims(ctx context.Context, c *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, c)
}
