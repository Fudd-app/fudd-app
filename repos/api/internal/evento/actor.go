package evento

import (
	"context"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
)

type ctxKey string

const actorKey ctxKey = "evento_actor"

// NombresUsuario resuelve el nombre de un usuario para completar el actor.
type NombresUsuario interface {
	NombreUsuario(ctx context.Context, usuarioID string) (string, error)
}

// ContextoConActor fija el actor explícito (comensal, webhook, sistema) y tiene
// prioridad sobre los claims del contexto.
func ContextoConActor(ctx context.Context, a Actor) context.Context {
	return context.WithValue(ctx, actorKey, a)
}

// ActorDesdeContexto arma el actor: el explícito si existe, si no el usuario
// autenticado de los claims, y si no hay ninguno, el sistema. usuarios puede ser
// nil: Registrar completa entonces el nombre desde la tabla de usuarios.
// Si el nombre no se puede resolver, el actor conserva solo el rol.
func ActorDesdeContexto(ctx context.Context, usuarios NombresUsuario) Actor {
	if a, ok := ctx.Value(actorKey).(Actor); ok {
		return a
	}
	claims := auth.ClaimsFromContext(ctx)
	if claims == nil || claims.UsuarioID == "" {
		return Actor{Tipo: ActorSistema}
	}
	a := Actor{Tipo: ActorUsuario, UsuarioID: claims.UsuarioID, Rol: claims.Rol}
	if usuarios != nil {
		if nombre, err := usuarios.NombreUsuario(ctx, claims.UsuarioID); err == nil {
			a.Nombre = nombre
		}
	}
	return a
}

// Comensal es el actor de los pedidos y solicitudes hechos por QR.
func Comensal(nombre string) Actor { return Actor{Tipo: ActorComensal, Nombre: nombre} }

// MercadoPago es el actor de los pagos confirmados por la pasarela.
func MercadoPago() Actor { return Actor{Tipo: ActorMercadoPago} }
