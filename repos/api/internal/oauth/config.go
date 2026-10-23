package oauth

import (
	"crypto/sha256"
	"errors"
	"os"
	"strings"
	"time"
)

const (
	ScopeLectura        = "fudd.lectura"
	DuracionAccessToken = 15 * time.Minute
	DuracionRefresh     = 30 * 24 * time.Hour
	DuracionCodigo      = 60 * time.Second
	DuracionSolicitud   = 10 * time.Minute
)

type Config struct {
	Issuer   string // MCP_BASE_URL
	Resource string // Issuer + "/mcp"
	AppURL   string // APP_URL
	Secreto  []byte // clave derivada de JWT_SECRET

	// DominioCookie (MCP_COOKIE_DOMAIN) es el dominio de la cookie de solicitud; vacío = host-only.
	DominioCookie string
}

func ConfigDesdeEntorno() (Config, error) {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return Config{}, errors.New("JWT_SECRET no configurado")
	}
	issuer := strings.TrimRight(os.Getenv("MCP_BASE_URL"), "/")
	if issuer == "" {
		issuer = strings.TrimRight(os.Getenv("API_URL"), "/")
	}
	if issuer == "" {
		issuer = "http://localhost:8080"
	}
	appURL := strings.TrimRight(os.Getenv("APP_URL"), "/")
	if appURL == "" {
		appURL = "http://localhost:3000"
	}
	// Clave derivada: un token de sesión web no valida como token MCP y viceversa.
	clave := sha256.Sum256([]byte("fudd-mcp-access-token:" + jwtSecret))
	return Config{
		Issuer:   issuer,
		Resource: issuer + "/mcp",
		AppURL:   appURL,
		Secreto:  clave[:],

		DominioCookie: strings.TrimPrefix(strings.TrimSpace(os.Getenv("MCP_COOKIE_DOMAIN")), "."),
	}, nil
}
