package evento_test

import (
	"context"
	"errors"
	"testing"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/evento"
)

type nombresFalsos struct {
	nombre string
	err    error
}

func (n nombresFalsos) NombreUsuario(ctx context.Context, usuarioID string) (string, error) {
	return n.nombre, n.err
}

func ctxConClaims() context.Context {
	return auth.ContextConClaims(context.Background(), &auth.Claims{UsuarioID: "u-1", TenantID: "t-1", Rol: auth.RolMozo})
}

func TestActorDesdeContexto_UsuarioAutenticadoConNombre(t *testing.T) {
	a := evento.ActorDesdeContexto(ctxConClaims(), nombresFalsos{nombre: "Ana"})
	if a.Tipo != evento.ActorUsuario || a.UsuarioID != "u-1" || a.Nombre != "Ana" || a.Rol != auth.RolMozo {
		t.Fatalf("actor incorrecto: %+v", a)
	}
}

func TestActorDesdeContexto_SinNombreConservaRol(t *testing.T) {
	a := evento.ActorDesdeContexto(ctxConClaims(), nombresFalsos{err: errors.New("no existe")})
	if a.Tipo != evento.ActorUsuario || a.Nombre != "" || a.Rol != auth.RolMozo {
		t.Fatalf("actor incorrecto: %+v", a)
	}
	if b := evento.ActorDesdeContexto(ctxConClaims(), nil); b.Nombre != "" || b.Rol != auth.RolMozo {
		t.Fatalf("actor incorrecto sin resolver: %+v", b)
	}
}

func TestActorDesdeContexto_ExplicitoTienePrioridad(t *testing.T) {
	ctx := evento.ContextoConActor(ctxConClaims(), evento.MercadoPago())
	if a := evento.ActorDesdeContexto(ctx, nil); a.Tipo != evento.ActorMercadoPago || a.UsuarioID != "" {
		t.Fatalf("actor incorrecto: %+v", a)
	}
}

func TestActorDesdeContexto_SinClaimsEsSistema(t *testing.T) {
	if a := evento.ActorDesdeContexto(context.Background(), nil); a.Tipo != evento.ActorSistema {
		t.Fatalf("actor incorrecto: %+v", a)
	}
}
