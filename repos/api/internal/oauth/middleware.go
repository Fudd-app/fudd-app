package oauth

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
)

// RequerirTokenMCP protege el endpoint MCP. Acepta solo access tokens MCP por
// Authorization: Bearer; el JWT de sesión web y la cookie no sirven acá.
func RequerirTokenMCP(svc *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			desafio := `Bearer resource_metadata="` + svc.cfg.Issuer + `/.well-known/oauth-protected-resource/mcp"`

			cabecera := r.Header.Get("Authorization")
			token := ""
			if len(cabecera) >= 7 && strings.EqualFold(cabecera[:7], "bearer ") {
				token, _ = strings.CutPrefix(cabecera, cabecera[:7])
				token = strings.TrimSpace(token)
			}
			if token == "" {
				w.Header().Set("WWW-Authenticate", desafio)
				jsonError(w, "invalid_token", http.StatusUnauthorized)
				return
			}

			acceso, err := svc.ValidarAccessToken(r.Context(), token)
			if err != nil {
				if !errors.Is(err, ErrTokenInvalido) {
					slog.ErrorContext(r.Context(), "error validando token MCP", "error", err)
					jsonError(w, "server_error", http.StatusInternalServerError)
					return
				}
				w.Header().Set("WWW-Authenticate", desafio+`, error="invalid_token"`)
				jsonError(w, "invalid_token", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r.WithContext(auth.ContextConClaims(r.Context(), acceso.Claims)))
		})
	}
}
