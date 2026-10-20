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
	ErrSesionRevocada      = errors.New("sesión revocada")
)

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
	SucursalID     *string
	Username       *string
	Email          string
	Nombre         string
	Rol            string
	Estado         string
	SessionVersion int64
	GoogleSub      *string
	PasswordHash   *string
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

type LoginStaffResultado struct {
	Usuario    *UsuarioAuth
	Negocio    *NegocioPublicoAuth
	RedirectTo string
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
