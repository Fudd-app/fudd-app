package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const googleJWKSEndpoint = "https://www.googleapis.com/oauth2/v3/certs"

var (
	ErrGoogleNoConfigurado       = errors.New("inicio con Google no configurado")
	ErrCredencialGoogleInvalida  = errors.New("credencial de Google inválida")
	ErrEmailGoogleNoVerificado   = errors.New("Google no pudo verificar el correo")
	ErrIdentidadGoogleIncompleta = errors.New("la identidad de Google está incompleta")
)

// IdentidadGoogle contiene únicamente los datos de identidad necesarios para
// vincular una cuenta de Google con un usuario ya registrado en Mesa CLICK.
type IdentidadGoogle struct {
	Subject         string
	Email           string
	Nombre          string
	EmailVerificado bool
}

// VerificadorGoogle permite sustituir la validación real en pruebas unitarias.
type VerificadorGoogle interface {
	Verificar(ctx context.Context, credencial, audiencia string) (*IdentidadGoogle, error)
}

type googleIDTokenClaims struct {
	Email           string `json:"email"`
	Nombre          string `json:"name"`
	EmailVerificado bool   `json:"email_verified"`
	jwt.RegisteredClaims
}

type googleJWKSet struct {
	Keys []struct {
		KeyID     string `json:"kid"`
		KeyType   string `json:"kty"`
		Algorithm string `json:"alg"`
		Modulus   string `json:"n"`
		Exponent  string `json:"e"`
	} `json:"keys"`
}

type verificadorGoogleIDToken struct {
	mu         sync.Mutex
	keys       map[string]*rsa.PublicKey
	expiraEn   time.Time
	httpClient *http.Client
}

func nuevoVerificadorGoogleIDToken() *verificadorGoogleIDToken {
	return &verificadorGoogleIDToken{
		keys:       make(map[string]*rsa.PublicKey),
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

func (v *verificadorGoogleIDToken) Verificar(ctx context.Context, credencial, audiencia string) (*IdentidadGoogle, error) {
	claims := &googleIDTokenClaims{}
	token, err := jwt.ParseWithClaims(
		credencial,
		claims,
		func(token *jwt.Token) (any, error) {
			keyID, _ := token.Header["kid"].(string)
			if keyID == "" {
				return nil, ErrCredencialGoogleInvalida
			}
			return v.obtenerClave(ctx, keyID)
		},
		jwt.WithAudience(audiencia),
		jwt.WithExpirationRequired(),
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
	)
	if err != nil || !token.Valid {
		return nil, ErrCredencialGoogleInvalida
	}
	if claims.Issuer != "accounts.google.com" && claims.Issuer != "https://accounts.google.com" {
		return nil, ErrCredencialGoogleInvalida
	}

	identidad := &IdentidadGoogle{
		Subject:         strings.TrimSpace(claims.Subject),
		Email:           NormalizarEmail(claims.Email),
		Nombre:          strings.TrimSpace(claims.Nombre),
		EmailVerificado: claims.EmailVerificado,
	}
	if identidad.Subject == "" || identidad.Email == "" {
		return nil, ErrIdentidadGoogleIncompleta
	}
	return identidad, nil
}

func (v *verificadorGoogleIDToken) obtenerClave(ctx context.Context, keyID string) (*rsa.PublicKey, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if time.Now().Before(v.expiraEn) {
		if key := v.keys[keyID]; key != nil {
			return key, nil
		}
	}
	if err := v.actualizarClaves(ctx); err != nil {
		return nil, err
	}
	key := v.keys[keyID]
	if key == nil {
		return nil, ErrCredencialGoogleInvalida
	}
	return key, nil
}

func (v *verificadorGoogleIDToken) actualizarClaves(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, googleJWKSEndpoint, nil)
	if err != nil {
		return ErrCredencialGoogleInvalida
	}
	resp, err := v.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("no se pudieron obtener las claves públicas de Google: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Google devolvió HTTP %d al solicitar sus claves públicas", resp.StatusCode)
	}

	var jwks googleJWKSet
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("respuesta de claves de Google inválida: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey, len(jwks.Keys))
	for _, jwk := range jwks.Keys {
		if jwk.KeyID == "" || jwk.KeyType != "RSA" || jwk.Algorithm != "RS256" {
			continue
		}
		key, err := convertirJWK(jwk.Modulus, jwk.Exponent)
		if err != nil {
			continue
		}
		keys[jwk.KeyID] = key
	}
	if len(keys) == 0 {
		return errors.New("Google no devolvió claves públicas utilizables")
	}

	v.keys = keys
	v.expiraEn = time.Now().Add(duracionCache(resp.Header.Get("Cache-Control")))
	return nil
}

func convertirJWK(modulus, exponent string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(modulus)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(exponent)
	if err != nil {
		return nil, err
	}
	e := 0
	for _, b := range eBytes {
		e = e<<8 + int(b)
	}
	if e == 0 {
		return nil, errors.New("exponente RSA inválido")
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: e}, nil
}

func duracionCache(cacheControl string) time.Duration {
	for _, directiva := range strings.Split(cacheControl, ",") {
		valor, ok := strings.CutPrefix(strings.TrimSpace(directiva), "max-age=")
		if !ok {
			continue
		}
		segundos, err := strconv.Atoi(valor)
		if err == nil && segundos > 0 {
			return time.Duration(segundos) * time.Second
		}
	}
	return time.Hour
}
