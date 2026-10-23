package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/metrica"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/oauth"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/ratelimit"
)

const instrucciones = "Fudd: datos en tiempo real del negocio gastronómico del usuario (salón, cocina y turno). " +
	"Solo lectura. Si el negocio tiene varias sucursales y el usuario no aclara cuál, llamá primero a listar_sucursales."

// MetricasLector es lo mínimo que se usa de metrica.Service.
type MetricasLector interface {
	ObtenerResumen(ctx context.Context, tenantID, sucursalID, desdeRaw, hastaRaw string) (*metrica.Resumen, error)
}

type Dependencias struct {
	Store    Store
	Metricas MetricasLector
	Ahora    func() time.Time
}

// NuevoHandler arma la cadena habilitado → token MCP → rate limit → SDK.
func NuevoHandler(oauthSvc *oauth.Service, deps Dependencias) http.Handler {
	return nuevoHandler(oauth.RequerirTokenMCP(oauthSvc), deps)
}

func nuevoHandler(autenticar func(http.Handler) http.Handler, deps Dependencias) http.Handler {
	if deps.Ahora == nil {
		deps.Ahora = time.Now
	}
	// Compartido entre requests: si se creara por request nunca limitaría.
	limiteTools := ratelimit.Nuevo(60, time.Minute)
	// Stateless: cada request crea su propio servidor con los claims de ese request.
	sdkHandler := sdk.NewStreamableHTTPHandler(func(r *http.Request) *sdk.Server {
		claims := auth.ClaimsFromContext(r.Context())
		if claims == nil {
			return nil
		}
		return construirServidor(claims, deps, limiteTools)
	}, &sdk.StreamableHTTPOptions{Stateless: true, JSONResponse: true})

	cadena := autenticar(limitarPorUsuario(ratelimit.Nuevo(300, time.Minute), sdkHandler))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if os.Getenv("MCP_HABILITADO") != "true" {
			escribirJSON(w, http.StatusNotFound, map[string]string{"error": "no encontrado"})
			return
		}
		cadena.ServeHTTP(w, r)
	})
}

func limitarPorUsuario(limitador *ratelimit.Limitador, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := auth.ClaimsFromContext(r.Context())
		if claims == nil || !limitador.Permitir(claims.UsuarioID) {
			w.Header().Set("Retry-After", "60")
			escribirJSON(w, http.StatusTooManyRequests, map[string]string{
				"error":             "rate_limited",
				"error_description": "Demasiadas consultas, reintentá en un minuto",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func escribirJSON(w http.ResponseWriter, status int, cuerpo any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(cuerpo)
}

func construirServidor(claims *auth.Claims, deps Dependencias, limitador *ratelimit.Limitador) *sdk.Server {
	servidor := sdk.NewServer(&sdk.Implementation{Name: "fudd", Version: "1.0.0"}, &sdk.ServerOptions{Instructions: instrucciones})
	h := &herramientas{deps: deps, claims: claims, limitador: limitador}
	h.registrar(servidor)
	return servidor
}
