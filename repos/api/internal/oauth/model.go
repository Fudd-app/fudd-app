package oauth

import (
	"errors"
	"time"
)

type Cliente struct {
	ID           string
	ClientID     string
	ClientName   string
	RedirectURIs []string
	CreatedAt    time.Time
}

type Solicitud struct {
	ID, ClientID, RedirectURI, CodeChallenge, State, Scope, Resource, Estado string
	NavegadorHash                                                            string
	UsuarioID                                                                *string
	CodigoHash                                                               *string
	CodigoExpiraAt                                                           *time.Time
	ExpiraAt, CreatedAt                                                      time.Time
}

type Conexion struct {
	ID, ClientID, ClientName, UsuarioID, TenantID, Scope string
	SessionVersion                                       int64
	RevocadaAt, UltimoUsoAt                              *time.Time
	CreatedAt                                            time.Time
}

type RefreshToken struct {
	TokenHash, ConexionID string
	UsadoAt               *time.Time
	ExpiraAt              time.Time
}

// SolicitudVista es la respuesta de GET /oauth/solicitudes/{id}.
type SolicitudVista struct {
	ID             string    `json:"id"`
	ClientName     string    `json:"client_name"`
	DominioRetorno string    `json:"dominio_retorno"`
	Negocio        string    `json:"negocio"`
	Permisos       []string  `json:"permisos"`
	SoloLectura    bool      `json:"solo_lectura"`
	MismoNavegador bool      `json:"mismo_navegador"`
	ExpiraAt       time.Time `json:"expira_at"`
}

// ConexionVista es la respuesta de GET /oauth/conexiones.
type ConexionVista struct {
	ID          string     `json:"id"`
	ClientName  string     `json:"client_name"`
	CreadoAt    time.Time  `json:"creado_at"`
	UltimoUsoAt *time.Time `json:"ultimo_uso_at"`
}

type TokenRespuesta struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

var (
	ErrClienteInvalido       = errors.New("cliente inválido")
	ErrRedirectInvalido      = errors.New("redirect_uri inválido")
	ErrSolicitudInvalida     = errors.New("solicitud inválida")
	ErrOtroNavegador         = errors.New("la solicitud se inició en otro navegador")
	ErrScopeInvalido         = errors.New("scope inválido")
	ErrSolicitudNoEncontrada = errors.New("solicitud no encontrada")
	ErrSoloDuenio            = errors.New("solo el dueño puede conectar asistentes de IA")
	ErrPlanSinMCP            = errors.New("la conexión con asistentes de IA no está incluida en el plan")
	ErrGrantInvalido         = errors.New("invalid_grant")
	ErrConexionNoEncontrada  = errors.New("conexión no encontrada")
	ErrTokenInvalido         = errors.New("token inválido")
)
