package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strings"
)

// EsSuperadmin comprueba si el correo pertenece a la lista de correos
// autorizados en la variable de entorno SUPERADMIN_EMAILS.
func EsSuperadmin(email string) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return false
	}
	superadmins := os.Getenv("SUPERADMIN_EMAILS")
	if superadmins == "" {
		return false
	}
	for _, autorizado := range strings.Split(superadmins, ",") {
		if strings.ToLower(strings.TrimSpace(autorizado)) == email {
			return true
		}
	}
	return false
}

// RequerirSuperadmin protege las rutas exclusivas de la plataforma verificando
// que el usuario tenga sesión activa y su email esté en la whitelist de socios.
func RequerirSuperadmin(store Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return Requerir(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromContext(r.Context())
			if claims == nil {
				jsonAuthError(w, "no autorizado", http.StatusUnauthorized)
				return
			}

			usuario, err := store.ObtenerUsuarioPorID(r.Context(), claims.UsuarioID)
			if err != nil || usuario == nil || usuario.Estado != "activo" {
				jsonAuthError(w, "sesión inválida o revocada", http.StatusUnauthorized)
				return
			}

			if !EsSuperadmin(usuario.Email) {
				slog.WarnContext(r.Context(), "intento de acceso denegado a plataforma superadmin",
					"usuario_id", usuario.ID,
					"email", usuario.Email,
				)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"error":  "acceso denegado: se requieren privilegios de superadministrador",
					"codigo": "SUPERADMIN_REQUIRED",
				})
				return
			}

			next.ServeHTTP(w, r)
		}))
	}
}
