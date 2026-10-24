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
	"strconv"
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
	}, &sdk.StreamableHTTPOptions{
		Stateless:    true,
		JSONResponse: true,
		// La protección anti DNS rebinding del SDK es para servidores locales sin auth. Detrás del proxy
		// de Render el tráfico llega por loopback con Host público y devolvía 403; acá exigimos un Bearer
		// OAuth que un ataque de rebinding no puede obtener.
		DisableLocalhostProtection: true,
	})

	cadena := autenticar(registrarUsuario(limitarPorUsuario(ratelimit.Nuevo(300, time.Minute), sdkHandler)))

	return registrarRequests(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if os.Getenv("MCP_HABILITADO") != "true" {
			escribirJSON(w, http.StatusNotFound, map[string]string{"error": "no encontrado"})
			return
		}
		cadena.ServeHTTP(w, r)
	}))
}

const (
	maxBodyLog     = 64 << 10
	maxCampoLog    = 120
	maxBatchLog    = 10
	metodoDesconoc = "otro"
)

// metodosConocidos es la lista blanca de lo que se loguea como rpc_method:
// el resto es controlado por el cliente y se registra como "otro".
var metodosConocidos = map[string]bool{
	"initialize": true, "ping": true, "tools/list": true, "tools/call": true,
	"resources/list": true, "resources/templates/list": true, "resources/read": true,
	"prompts/list": true, "prompts/get": true, "logging/setLevel": true,
	"completion/complete": true, "notifications/initialized": true,
	"notifications/cancelled": true, "notifications/progress": true,
	"notifications/roots/list_changed": true,
}

// acotar limita a maxCampoLog caracteres un valor que viene del cliente.
func acotar(valor string) string {
	runas := []rune(valor)
	if len(runas) <= maxCampoLog {
		return valor
	}
	return string(runas[:maxCampoLog])
}

type claveUsuarioLog struct{}

// registrarRequests es la primera capa: loguea cada request a /mcp con lo que
// importa para diagnosticar clientes (método RPC, status, cabeceras), nunca el
// valor de Authorization ni los params. Todo lo que viene del cliente se acota
// antes de loguearlo, y el body solo se inspecciona si el endpoint está habilitado.
func registrarRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inicio := time.Now()
		rpcMethod := ""
		if os.Getenv("MCP_HABILITADO") == "true" && r.Method == http.MethodPost && esJSON(r.Header.Get("Content-Type")) && r.Body != nil {
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

		attrs := []any{"http_method", r.Method}
		if sw.status == http.StatusNotFound && os.Getenv("MCP_HABILITADO") != "true" {
			slog.InfoContext(r.Context(), "mcp request", append(attrs, "status", sw.status)...)
			return
		}
		attrs = append(attrs,
			"rpc_method", rpcMethod,
			"status", sw.status,
			"duracion_ms", time.Since(inicio).Milliseconds(),
			"tiene_authorization", r.Header.Get("Authorization") != "",
			"user_agent", acotar(r.UserAgent()),
			"mcp_protocol_version", acotar(r.Header.Get("MCP-Protocol-Version")),
			"accept", acotar(r.Header.Get("Accept")),
			"content_type", acotar(r.Header.Get("Content-Type")),
		)
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

// nombreMetodo devuelve el método si es conocido y "otro" si no.
func nombreMetodo(metodo string) string {
	if metodosConocidos[metodo] {
		return metodo
	}
	return metodoDesconoc
}

// metodosRPC extrae solo el campo method del JSON-RPC (o los de un batch, separados por coma,
// hasta maxBatchLog entradas más "+N" con las restantes). Los valores salen de la lista blanca.
func metodosRPC(cuerpo []byte) string {
	cuerpo = bytes.TrimSpace(cuerpo)
	if len(cuerpo) > 0 && cuerpo[0] == '[' {
		var batch []struct {
			Method string `json:"method"`
		}
		if json.Unmarshal(cuerpo, &batch) != nil {
			return "?"
		}
		metodos := make([]string, 0, maxBatchLog+1)
		for i, m := range batch {
			if i == maxBatchLog {
				metodos = append(metodos, "+"+strconv.Itoa(len(batch)-maxBatchLog))
				break
			}
			metodos = append(metodos, nombreMetodo(m.Method))
		}
		return strings.Join(metodos, ",")
	}
	var msg struct {
		Method string `json:"method"`
	}
	if json.Unmarshal(cuerpo, &msg) != nil {
		return "?"
	}
	return nombreMetodo(msg.Method)
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
	servidor.AddReceivingMiddleware(estructurarResultados())
	return servidor
}

// estructurarResultados normaliza structuredContent en las respuestas de tools:
//   - éxito: el SDK vuelve a serializar la salida tipada (claves ordenadas alfabéticamente), mientras
//     que el texto conserva el orden de los campos; se copia el texto para que ambos sean idénticos.
//   - error: se elimina structuredContent. La spec MCP exige que cumpla el outputSchema y los clientes
//     lo validan incluso con isError, así que el JSON {"error","mensaje","detalle"} queda solo en el texto.
//
// Solo se puede hacer después, sobre el resultado de tools/call.
func estructurarResultados() sdk.Middleware {
	return func(next sdk.MethodHandler) sdk.MethodHandler {
		return func(ctx context.Context, method string, req sdk.Request) (sdk.Result, error) {
			res, err := next(ctx, method, req)
			if err != nil || method != "tools/call" {
				return res, err
			}
			llamada, ok := res.(*sdk.CallToolResult)
			if !ok || len(llamada.Content) == 0 {
				return res, err
			}
			if llamada.IsError {
				llamada.StructuredContent = nil
				return res, err
			}
			texto, ok := llamada.Content[0].(*sdk.TextContent)
			if !ok {
				return res, err
			}
			if llamada.StructuredContent != nil {
				llamada.StructuredContent = json.RawMessage(texto.Text)
			}
			return res, err
		}
	}
}
