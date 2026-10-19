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

	hash := dummyPasswordHash
	if usuario.PasswordHash != nil && strings.TrimSpace(*usuario.PasswordHash) != "" {
		hash = *usuario.PasswordHash
	}
	if err := CompararPassword(hash, password); err != nil || usuario.PasswordHash == nil {
		return nil, ErrCredencialesInvalidas
	}
	if err := validarUsuarioActivo(usuario); err != nil {
		return nil, err
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
		return usuario, validarUsuarioActivo(usuario)
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
	if err := svc.store.VincularGoogleSub(ctx, usuario.ID, identidad.Subject); err != nil {
		return nil, err
	}
	usuario.GoogleSub = &identidad.Subject
	return usuario, validarUsuarioActivo(usuario)
}

// AutenticarStaffGoogle exige el slug del negocio para desambiguar a una misma
// persona que trabaja en más de un tenant. El alta solo puede existir por una
// invitación previamente aceptada; nunca crea usuarios durante el login.
func (svc *Service) AutenticarStaffGoogle(ctx context.Context, slug, credencial string) (*UsuarioAuth, error) {
	identidad, err := svc.VerificarIdentidadGoogle(ctx, credencial)
	if err != nil {
		return nil, err
	}
	staffStore, ok := svc.store.(StaffStore)
	if !ok {
		return nil, errors.New("store sin soporte para acceso del equipo")
	}
	slug = strings.TrimSpace(strings.ToLower(slug))
	usuario, err := staffStore.ObtenerStaffPorGoogleSub(ctx, slug, identidad.Subject)
	if err == nil {
		return usuario, validarUsuarioActivo(usuario)
	}
	if !errors.Is(err, ErrUsuarioNoEncontrado) {
		return nil, err
	}
	usuario, err = staffStore.ObtenerStaffPorEmail(ctx, slug, identidad.Email)
	if err != nil {
		return nil, err
	}
	if err := svc.store.VincularGoogleSub(ctx, usuario.ID, identidad.Subject); err != nil {
		return nil, err
	}
	usuario.GoogleSub = &identidad.Subject
	return usuario, validarUsuarioActivo(usuario)
}

func validarUsuarioActivo(usuario *UsuarioAuth) error {
	if usuario == nil {
		return ErrUsuarioNoEncontrado
	}
	// El vacío solo existe en dobles de prueba anteriores a session_version.
	if usuario.Estado != "" && usuario.Estado != "activo" {
		return ErrUsuarioBloqueado
	}
	return nil
}

func (svc *Service) CerrarSesion(ctx context.Context, usuarioID string) error {
	sessionStore, ok := svc.store.(SessionStore)
	if !ok {
		return nil
	}
	return sessionStore.InvalidarSesiones(ctx, usuarioID)
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
	usuario, err := svc.store.ObtenerUsuarioPorID(ctx, mt.UsuarioID)
	if err != nil {
		return nil, err
	}
	if err := validarUsuarioActivo(usuario); err != nil {
		return nil, err
	}
	return usuario, nil
}

func GenerarJWT(claims *Claims, secreto string) (string, error) {
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"usuario_id":      claims.UsuarioID,
		"tenant_id":       claims.TenantID,
		"rol":             claims.Rol,
		"sucursal_id":     claims.SucursalID,
		"session_version": claims.SessionVersion,
		"exp":             time.Now().Add(24 * time.Hour * 30).Unix(),
	})
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
	sucursalID, _ := mc["sucursal_id"].(string)
	var sessionVersion int64
	if raw, ok := mc["session_version"].(float64); ok {
		sessionVersion = int64(raw)
	}
	return &Claims{
		UsuarioID:      usuarioID,
		TenantID:       tenantID,
		Rol:            rol,
		SucursalID:     sucursalID,
		SessionVersion: sessionVersion,
	}, nil
}

func generarTokenAleatorio() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
