package auth

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strings"
)

func Requerir(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		secreto := os.Getenv("JWT_SECRET")
		if secreto == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"servidor mal configurado"}`))
			slog.Error("JWT_SECRET no configurado")
			return
		}

		var tokenStr string

		if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
			tokenStr = strings.TrimPrefix(h, "Bearer ")
		}
		if tokenStr == "" {
			if c, err := r.Cookie("session"); err == nil {
				tokenStr = c.Value
			}
		}
		if tokenStr == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"no autorizado"}`))
			return
		}

		claims, err := ValidarJWT(tokenStr, secreto)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"token inválido"}`))
			return
		}

		ctx := ContextConClaims(r.Context(), claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

var onActividadUsuario func(usuarioID string)

// RegistrarHookActividad permite conectar la telemetría de presencia de usuarios
// sin acoplar el paquete de autenticación a módulos de plataforma.
func RegistrarHookActividad(fn func(usuarioID string)) {
	onActividadUsuario = fn
}

// RequerirSesion valida tanto la firma del JWT como el estado actual del
// usuario en cada request. De esta forma bloquear/eliminar un miembro o cerrar
// sesión revoca inmediatamente todos los tokens ya emitidos.
func RequerirSesion(store Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return Requerir(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromContext(r.Context())
			usuario, err := store.ObtenerUsuarioPorID(r.Context(), claims.UsuarioID)
			if err != nil || usuario.Estado != "activo" || usuario.TenantID != claims.TenantID ||
				usuario.Rol != claims.Rol || usuario.SessionVersion != claims.SessionVersion ||
				sucursalUsuario(usuario) != claims.SucursalID {
				jsonAuthError(w, "sesión revocada", http.StatusUnauthorized)
				return
			}
			if onActividadUsuario != nil {
				onActividadUsuario(usuario.ID)
			}

			actuales := &Claims{
				UsuarioID: usuario.ID, TenantID: usuario.TenantID, Rol: usuario.Rol,
				SucursalID: sucursalUsuario(usuario), SessionVersion: usuario.SessionVersion,
			}
			next.ServeHTTP(w, r.WithContext(ContextConClaims(r.Context(), actuales)))
		}))
	}
}

func sucursalUsuario(usuario *UsuarioAuth) string {
	if usuario != nil && usuario.SucursalID != nil {
		return *usuario.SucursalID
	}
	return ""
}

func RequerirRoles(roles ...string) func(http.Handler) http.Handler {
	permitidos := make(map[string]struct{}, len(roles))
	for _, rol := range roles {
		permitidos[rol] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromContext(r.Context())
			if claims == nil {
				jsonAuthError(w, "no autorizado", http.StatusUnauthorized)
				return
			}
			if _, ok := permitidos[claims.Rol]; !ok {
				jsonAuthError(w, "sin permisos para esta operación", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ResolverSucursal impide que el staff seleccione otra sucursal mediante
// query params o payloads. El dueño conserva acceso transversal al tenant.
func ResolverSucursal(claims *Claims, solicitada string) (string, error) {
	if claims == nil {
		return "", ErrSesionRevocada
	}
	if claims.Rol == RolDuenio {
		if solicitada == "" {
			return "", errors.New("sucursal requerida")
		}
		return solicitada, nil
	}
	if claims.SucursalID == "" {
		return "", ErrSesionRevocada
	}
	if solicitada != "" && solicitada != claims.SucursalID {
		return "", errors.New("sucursal fuera de alcance")
	}
	return claims.SucursalID, nil
}

func jsonAuthError(w http.ResponseWriter, mensaje string, estado int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(estado)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": mensaje})
}
