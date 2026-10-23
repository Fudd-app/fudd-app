package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"strings"
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

// AccesoPlan resuelve si el plan (o una excepción) del negocio incluye MCP.
// Se consulta en cada tool: un token vigente no sobrevive a una prueba vencida.
type AccesoPlan interface {
	AccesoMCP(ctx context.Context, tenantID string) (bool, error)
}

type Dependencias struct {
	Store    Store
	Metricas MetricasLector
	Acceso   AccesoPlan
	AppURL   string
	Ahora    func() time.Time
}

// NuevoHandler arma la cadena log → habilitado → token MCP → rate limit → SDK.
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

	cadena := autenticar(registrarUsuario(limitarPorUsuario(ratelimit.Nuevo(300, time.Minute), sdkHandler)))

	return registrarRequests(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if os.Getenv("MCP_HABILITADO") != "true" {
			escribirJSON(w, http.StatusNotFound, map[string]string{"error": "no encontrado"})
			return
		}
		cadena.ServeHTTP(w, r)
	}))
}

const maxBodyLog = 1 << 20

type claveUsuarioLog struct{}

// registrarRequests es la primera capa: loguea cada request a /mcp con lo que
// importa para diagnosticar clientes (método RPC, status, cabeceras), nunca el
// valor de Authorization ni los params.
func registrarRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inicio := time.Now()
		rpcMethod := ""
		if r.Method == http.MethodPost && esJSON(r.Header.Get("Content-Type")) && r.Body != nil {
			cuerpo, _ := io.ReadAll(io.LimitReader(r.Body, maxBodyLog))
			// Se restaura lo leído más lo que quede sin leer, para que el SDK vea el body intacto.
			r.Body = struct {
				io.Reader
				io.Closer
			}{io.MultiReader(io.NopCloser(bytes.NewReader(cuerpo)), r.Body), r.Body}
			rpcMethod = metodosRPC(cuerpo)
		}

		// El ID lo completa registrarUsuario, después de la autenticación.
		usuario := new(string)
		r = r.WithContext(context.WithValue(r.Context(), claveUsuarioLog{}, usuario))
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		var rw http.ResponseWriter = sw
		if _, ok := w.(http.Flusher); ok {
			rw = flushStatusWriter{sw}
		}
		next.ServeHTTP(rw, r)

		attrs := []any{
			"http_method", r.Method,
			"rpc_method", rpcMethod,
			"status", sw.status,
			"duracion_ms", time.Since(inicio).Milliseconds(),
			"tiene_authorization", r.Header.Get("Authorization") != "",
			"user_agent", r.UserAgent(),
			"mcp_protocol_version", r.Header.Get("MCP-Protocol-Version"),
			"accept", r.Header.Get("Accept"),
			"content_type", r.Header.Get("Content-Type"),
		}
		if *usuario != "" {
			attrs = append(attrs, "usuario_id", *usuario)
		}
		if sw.status >= http.StatusInternalServerError {
			slog.ErrorContext(r.Context(), "mcp request", attrs...)
			return
		}
		slog.InfoContext(r.Context(), "mcp request", attrs...)
	})
}

// registrarUsuario corre después de la auth y deja el ID del usuario para el log.
func registrarUsuario(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if usuario, ok := r.Context().Value(claveUsuarioLog{}).(*string); ok {
			if claims := auth.ClaimsFromContext(r.Context()); claims != nil {
				*usuario = claims.UsuarioID
			}
		}
		next.ServeHTTP(w, r)
	})
}

func esJSON(contentType string) bool {
	tipo, _, err := mime.ParseMediaType(contentType)
	return err == nil && (tipo == "application/json" || strings.HasSuffix(tipo, "+json"))
}

// metodosRPC extrae solo el campo method del JSON-RPC (o los de un batch, separados por coma).
func metodosRPC(cuerpo []byte) string {
	cuerpo = bytes.TrimSpace(cuerpo)
	if len(cuerpo) > 0 && cuerpo[0] == '[' {
		var batch []struct {
			Method string `json:"method"`
		}
		if json.Unmarshal(cuerpo, &batch) != nil {
			return "?"
		}
		metodos := make([]string, 0, len(batch))
		for _, m := range batch {
			metodos = append(metodos, m.Method)
		}
		return strings.Join(metodos, ",")
	}
	var msg struct {
		Method string `json:"method"`
	}
	if json.Unmarshal(cuerpo, &msg) != nil {
		return "?"
	}
	return msg.Method
}

type statusWriter struct {
	http.ResponseWriter
	status  int
	escrito bool
}

func (s *statusWriter) WriteHeader(status int) {
	if !s.escrito {
		s.status = status
		s.escrito = true
	}
	s.ResponseWriter.WriteHeader(status)
}

func (s *statusWriter) Write(b []byte) (int, error) {
	s.escrito = true
	return s.ResponseWriter.Write(b)
}

// Unwrap permite a http.ResponseController llegar al writer original.
func (s *statusWriter) Unwrap() http.ResponseWriter { return s.ResponseWriter }

// flushStatusWriter solo se usa si el writer original implementa http.Flusher.
type flushStatusWriter struct{ *statusWriter }

func (f flushStatusWriter) Flush() { f.ResponseWriter.(http.Flusher).Flush() }

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
	servidor.AddReceivingMiddleware(estructurarPlanSinMCP(deps.AppURL))
	return servidor
}

// estructurarPlanSinMCP agrega structured content al error de plan sin MCP. El wrapper
// tipado del SDK pisa StructuredContent con la salida del handler, así que solo se
// puede completar después, sobre el resultado de tools/call.
func estructurarPlanSinMCP(appURL string) sdk.Middleware {
	mensaje := mensajePlanSinMCP(appURL)
	return func(next sdk.MethodHandler) sdk.MethodHandler {
		return func(ctx context.Context, method string, req sdk.Request) (sdk.Result, error) {
			res, err := next(ctx, method, req)
			if err != nil || method != "tools/call" {
				return res, err
			}
			llamada, ok := res.(*sdk.CallToolResult)
			if !ok || !llamada.IsError || len(llamada.Content) == 0 {
				return res, err
			}
			if texto, ok := llamada.Content[0].(*sdk.TextContent); ok && texto.Text == mensaje {
				llamada.StructuredContent = map[string]string{
					"codigo":      "PLAN_LIMIT_REACHED",
					"recurso":     "mcp",
					"upgrade_url": appURL + "/dashboard",
				}
			}
			return res, err
		}
	}
}
