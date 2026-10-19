package auth

import (
	"context"
	"errors"
	"time"
)

var (
	ErrUsuarioNoEncontrado = errors.New("usuario no encontrado")
	ErrNegocioNoEncontrado = errors.New("comercio no encontrado")
	ErrUsuarioBloqueado    = errors.New("usuario bloqueado")
)

type MagicToken struct {
	ID        string
	UsuarioID string
	Token     string
	ExpiresAt time.Time
	UsedAt    *time.Time
}

type UsuarioAuth struct {
	ID           string
	TenantID     string
	SucursalID   *string
	Username     *string
	Nombre       string
	Email        string
	Rol          string
	Estado       string
	GoogleSub    *string
	PasswordHash *string
}

type NegocioPublicoAuth struct {
	ID             string  `json:"id"`
	Slug           string  `json:"slug"`
	Nombre         string  `json:"nombre"`
	NombreFantasia string  `json:"nombre_fantasia,omitempty"`
	Rubro          string  `json:"rubro,omitempty"`
	LogoURL        *string `json:"logo_url,omitempty"`
	ColorPrimario  *string `json:"color_primario,omitempty"`
	EstiloVisual   *string `json:"estilo_visual,omitempty"`
}

type UsuarioStaffSesion struct {
	ID         string  `json:"id"`
	TenantID   string  `json:"tenant_id"`
	SucursalID *string `json:"sucursal_id,omitempty"`
	Username   string  `json:"username,omitempty"`
	Nombre     string  `json:"nombre"`
	Rol        string  `json:"rol"`
}

type LoginStaffResultado struct {
	Usuario    *UsuarioAuth
	Negocio    *NegocioPublicoAuth
	RedirectTo string
}

type Claims struct {
	UsuarioID  string  `json:"usuario_id"`
	TenantID   string  `json:"tenant_id"`
	SucursalID *string `json:"sucursal_id,omitempty"`
	Rol        string  `json:"rol"`
}

type ctxKey string

const claimsKey ctxKey = "auth_claims"

func ClaimsFromContext(ctx context.Context) *Claims {
	c, _ := ctx.Value(claimsKey).(*Claims)
	return c
}

func ContextConClaims(ctx context.Context, c *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, c)
}
