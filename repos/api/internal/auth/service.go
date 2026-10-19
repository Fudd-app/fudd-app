package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Service struct {
	store          Store
	email          EmailSender
	googleClientID string
	googleVerifier VerificadorGoogle
}

func NuevoService(s Store, e EmailSender) *Service {
	return NuevoServiceConGoogle(s, e, os.Getenv("GOOGLE_CLIENT_ID"), nuevoVerificadorGoogleIDToken())
}

func NuevoServiceConGoogle(s Store, e EmailSender, clientID string, verifier VerificadorGoogle) *Service {
	return &Service{
		store:          s,
		email:          e,
		googleClientID: strings.TrimSpace(clientID),
		googleVerifier: verifier,
	}
}

func RutaRedireccionPorRol(rol string) string {
	if strings.EqualFold(strings.TrimSpace(rol), "cocina") {
		return "/kds"
	}
	return "/dashboard"
}

func (svc *Service) ObtenerNegocioPublico(ctx context.Context, slug string) (*NegocioPublicoAuth, error) {
	slug = strings.ToLower(strings.TrimSpace(slug))
	if slug == "" {
		return nil, ErrNegocioNoEncontrado
	}
	return svc.store.ObtenerNegocioPorSlug(ctx, slug)
}

func (svc *Service) AutenticarStaffPorSlug(ctx context.Context, slug, username, password string) (*LoginStaffResultado, error) {
	slug = strings.ToLower(strings.TrimSpace(slug))
	username = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(username), "@")))

	if slug == "" {
		return nil, ErrNegocioNoEncontrado
	}

	negocio, err := svc.store.ObtenerNegocioPorSlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	if username == "" || password == "" {
		_ = CompararPassword(dummyPasswordHash, password)
		return nil, ErrCredencialesInvalidas
	}

	usuario, err := svc.store.ObtenerUsuarioPorTenantYUsername(ctx, negocio.ID, username)
	if err != nil {
		if errors.Is(err, ErrUsuarioNoEncontrado) {
			_ = CompararPassword(dummyPasswordHash, password)
			return nil, ErrCredencialesInvalidas
		}
		return nil, err
	}

	if strings.EqualFold(usuario.Estado, "archivado") {
		_ = CompararPassword(dummyPasswordHash, password)
		return nil, ErrCredencialesInvalidas
	}

	hash := dummyPasswordHash
	if usuario.PasswordHash != nil && strings.TrimSpace(*usuario.PasswordHash) != "" {
		hash = *usuario.PasswordHash
	}
	if err := CompararPassword(hash, password); err != nil || usuario.PasswordHash == nil {
		return nil, ErrCredencialesInvalidas
	}

	if strings.EqualFold(usuario.Estado, "bloqueado") {
		return nil, ErrUsuarioBloqueado
	}

	return &LoginStaffResultado{
		Usuario:    usuario,
		Negocio:    negocio,
		RedirectTo: RutaRedireccionPorRol(usuario.Rol),
	}, nil
}

func (svc *Service) AutenticarPassword(ctx context.Context, email, password string) (*UsuarioAuth, error) {
	email = NormalizarEmail(email)
	if email == "" || password == "" {
		_ = CompararPassword(dummyPasswordHash, password)
		return nil, ErrCredencialesInvalidas
	}

	usuario, err := svc.store.ObtenerUsuarioPorEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUsuarioNoEncontrado) {
			_ = CompararPassword(dummyPasswordHash, password)
			return nil, ErrCredencialesInvalidas
		}
		return nil, err
	}

	if strings.EqualFold(usuario.Estado, "archivado") {
		_ = CompararPassword(dummyPasswordHash, password)
		return nil, ErrCredencialesInvalidas
	}

	hash := dummyPasswordHash
	if usuario.PasswordHash != nil && strings.TrimSpace(*usuario.PasswordHash) != "" {
		hash = *usuario.PasswordHash
	}
	if err := CompararPassword(hash, password); err != nil || usuario.PasswordHash == nil {
		return nil, ErrCredencialesInvalidas
	}

	if strings.EqualFold(usuario.Estado, "bloqueado") {
		return nil, ErrUsuarioBloqueado
	}

	return usuario, nil
}

// VerificarIdentidadGoogle valida la credencial con Google y devuelve la
// identidad con email normalizado y verificado. La usa también el registro
// de negocios para crear el admin vinculado a su cuenta de Google.
func (svc *Service) VerificarIdentidadGoogle(ctx context.Context, credencial string) (*IdentidadGoogle, error) {
	if svc.googleClientID == "" || svc.googleVerifier == nil {
		return nil, ErrGoogleNoConfigurado
	}
	credencial = strings.TrimSpace(credencial)
	if credencial == "" {
		return nil, ErrCredencialGoogleInvalida
	}

	identidad, err := svc.googleVerifier.Verificar(ctx, credencial, svc.googleClientID)
	if err != nil {
		return nil, err
	}
	identidad.Subject = strings.TrimSpace(identidad.Subject)
	identidad.Email = NormalizarEmail(identidad.Email)
	if identidad.Subject == "" || identidad.Email == "" {
		return nil, ErrIdentidadGoogleIncompleta
	}
	if !identidad.EmailVerificado {
		return nil, ErrEmailGoogleNoVerificado
	}
	return identidad, nil
}

// AutenticarGoogle valida la credencial con Google y luego emite una sesión
// únicamente para usuarios que ya existen en Mesa CLICK. En el primer acceso
// vincula el `sub` estable de Google para no depender de cambios futuros de email.
func (svc *Service) AutenticarGoogle(ctx context.Context, credencial string) (*UsuarioAuth, error) {
	identidad, err := svc.VerificarIdentidadGoogle(ctx, credencial)
	if err != nil {
		return nil, err
	}

	usuario, err := svc.store.ObtenerUsuarioPorGoogleSub(ctx, identidad.Subject)
	if err == nil {
		if strings.EqualFold(usuario.Estado, "bloqueado") {
			return nil, ErrUsuarioBloqueado
		}
		if strings.EqualFold(usuario.Estado, "archivado") {
			return nil, ErrUsuarioNoEncontrado
		}
		return usuario, nil
	}
	if !errors.Is(err, ErrUsuarioNoEncontrado) {
		return nil, err
	}

	usuario, err = svc.store.ObtenerUsuarioPorEmail(ctx, identidad.Email)
	if err != nil {
		if errors.Is(err, ErrUsuarioNoEncontrado) {
			slog.WarnContext(ctx, "login de Google para correo no registrado", "email", identidad.Email)
		}
		return nil, err
	}
	if strings.EqualFold(usuario.Estado, "bloqueado") {
		return nil, ErrUsuarioBloqueado
	}
	if strings.EqualFold(usuario.Estado, "archivado") {
		return nil, ErrUsuarioNoEncontrado
	}
	if err := svc.store.VincularGoogleSub(ctx, usuario.ID, identidad.Subject); err != nil {
		return nil, err
	}
	usuario.GoogleSub = &identidad.Subject
	return usuario, nil
}

// SolicitarLink genera el magic link y lo envía por email.
// Devuelve el link generado para que el handler pueda exponerlo en entornos de
// desarrollo sin proveedor de email.
func (svc *Service) SolicitarLink(ctx context.Context, email string) (string, error) {
	email = NormalizarEmail(email)

	usuario, err := svc.store.ObtenerUsuarioPorEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUsuarioNoEncontrado) {
			slog.Warn("magic link solicitado para email no registrado", "email", email)
			return "", ErrUsuarioNoEncontrado
		}
		return "", fmt.Errorf("error buscando usuario: %w", err)
	}
	if strings.EqualFold(usuario.Estado, "bloqueado") {
		return "", ErrUsuarioBloqueado
	}
	if strings.EqualFold(usuario.Estado, "archivado") {
		return "", ErrUsuarioNoEncontrado
	}

	token, err := generarTokenAleatorio()
	if err != nil {
		return "", fmt.Errorf("error generando token: %w", err)
	}

	expiresAt := time.Now().Add(15 * time.Minute)
	if _, err := svc.store.GuardarToken(ctx, usuario.ID, token, expiresAt); err != nil {
		return "", err
	}

	link := ConstruirLinkVerificacion(token)

	if err := svc.email.EnviarMagicLink(ctx, email, link); err != nil {
		slog.ErrorContext(ctx, "error enviando magic link", "email", email, "err", err)
		return "", fmt.Errorf("error enviando email: %w", err)
	}

	return link, nil
}

// NormalizarEmail deja los emails en una forma canónica (minúsculas, sin espacios)
// para que "Admin@Bar.com" y "admin@bar.com " sean la misma cuenta.
func NormalizarEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func ConstruirLinkVerificacion(token string) string {
	appURL := os.Getenv("APP_URL")
	if appURL == "" {
		appURL = "http://localhost:3000"
	}
	return fmt.Sprintf("%s/auth/verify?token=%s", strings.TrimRight(appURL, "/"), token)
}

func (svc *Service) VerificarToken(ctx context.Context, token string) (*UsuarioAuth, error) {
	mt, err := svc.store.ObtenerToken(ctx, token)
	if err != nil {
		return nil, errors.New("token inválido")
	}
	if mt.UsedAt != nil {
		return nil, errors.New("token ya usado")
	}
	if time.Now().After(mt.ExpiresAt) {
		return nil, errors.New("token expirado")
	}
	if err := svc.store.MarcarTokenUsado(ctx, mt.ID); err != nil {
		return nil, err
	}
	usr, err := svc.store.ObtenerUsuarioPorID(ctx, mt.UsuarioID)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(usr.Estado, "bloqueado") || strings.EqualFold(usr.Estado, "archivado") {
		return nil, errors.New("cuenta no habilitada")
	}
	return usr, nil
}

func GenerarJWT(claims *Claims, secreto string) (string, error) {
	mc := jwt.MapClaims{
		"usuario_id": claims.UsuarioID,
		"tenant_id":  claims.TenantID,
		"rol":        claims.Rol,
		"exp":        time.Now().Add(24 * time.Hour * 30).Unix(),
	}
	if claims.SucursalID != nil && *claims.SucursalID != "" {
		mc["sucursal_id"] = *claims.SucursalID
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, mc)
	return t.SignedString([]byte(secreto))
}

func ValidarJWT(tokenStr, secreto string) (*Claims, error) {
	t, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("método de firma inesperado: %v", t.Header["alg"])
		}
		return []byte(secreto), nil
	})
	if err != nil || !t.Valid {
		return nil, errors.New("token inválido")
	}
	mc, ok := t.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("claims inválidos")
	}
	usuarioID, ok1 := mc["usuario_id"].(string)
	tenantID, ok2 := mc["tenant_id"].(string)
	rol, ok3 := mc["rol"].(string)
	if !ok1 || !ok2 || !ok3 {
		return nil, errors.New("claims inválidos o incompletos")
	}
	var sucursalID *string
	if sid, ok := mc["sucursal_id"].(string); ok && sid != "" {
		sucursalID = &sid
	}
	return &Claims{
		UsuarioID:  usuarioID,
		TenantID:   tenantID,
		SucursalID: sucursalID,
		Rol:        rol,
	}, nil
}

func generarTokenAleatorio() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
