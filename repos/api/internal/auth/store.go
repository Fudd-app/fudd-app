package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/db"
	"github.com/jackc/pgx/v5"
)

type Store interface {
	ObtenerUsuarioPorEmail(ctx context.Context, email string) (*UsuarioAuth, error)
	ObtenerUsuarioPorGoogleSub(ctx context.Context, googleSub string) (*UsuarioAuth, error)
	ObtenerUsuarioPorID(ctx context.Context, id string) (*UsuarioAuth, error)
	VincularGoogleSub(ctx context.Context, usuarioID, googleSub string) error
	GuardarToken(ctx context.Context, usuarioID, token string, expiresAt time.Time) (string, error)
	ObtenerToken(ctx context.Context, token string) (*MagicToken, error)
	MarcarTokenUsado(ctx context.Context, tokenID string) error
}

// StaffStore agrega las búsquedas necesariamente acotadas por negocio. Se
// mantiene separado de Store para que el onboarding y sus dobles de prueba no
// dependan de los endpoints de acceso del equipo.
type StaffStore interface {
	ObtenerStaffPorGoogleSub(ctx context.Context, slug, googleSub string) (*UsuarioAuth, error)
	ObtenerStaffPorEmail(ctx context.Context, slug, email string) (*UsuarioAuth, error)
}

type SessionStore interface {
	InvalidarSesiones(ctx context.Context, usuarioID string) error
}

type pgStore struct{}

func NuevoStore() Store { return &pgStore{} }

func (s *pgStore) ObtenerUsuarioPorEmail(ctx context.Context, email string) (*UsuarioAuth, error) {
	u := &UsuarioAuth{}
	err := db.Pool.QueryRow(ctx,
		// lower() para que emails cargados con mayúsculas sigan matcheando.
		`SELECT id, tenant_id, email, nombre, rol, sucursal_id, estado, session_version, google_sub, password_hash
		 FROM usuarios WHERE lower(email) = lower($1) AND rol = 'admin'`, email,
	).Scan(&u.ID, &u.TenantID, &u.Email, &u.Nombre, &u.Rol, &u.SucursalID, &u.Estado, &u.SessionVersion, &u.GoogleSub, &u.PasswordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUsuarioNoEncontrado
		}
		return nil, fmt.Errorf("error consultando usuario por email: %w", err)
	}
	return u, nil
}

func (s *pgStore) ObtenerUsuarioPorGoogleSub(ctx context.Context, googleSub string) (*UsuarioAuth, error) {
	u := &UsuarioAuth{}
	err := db.Pool.QueryRow(ctx,
		`SELECT id, tenant_id, email, nombre, rol, sucursal_id, estado, session_version, google_sub, password_hash
		 FROM usuarios WHERE google_sub = $1 AND rol = 'admin'`, googleSub,
	).Scan(&u.ID, &u.TenantID, &u.Email, &u.Nombre, &u.Rol, &u.SucursalID, &u.Estado, &u.SessionVersion, &u.GoogleSub, &u.PasswordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUsuarioNoEncontrado
		}
		return nil, fmt.Errorf("error consultando usuario por identidad de Google: %w", err)
	}
	return u, nil
}

func (s *pgStore) ObtenerUsuarioPorID(ctx context.Context, id string) (*UsuarioAuth, error) {
	u := &UsuarioAuth{}
	err := db.Pool.QueryRow(ctx,
		`SELECT id, tenant_id, email, nombre, rol, sucursal_id, estado, session_version, google_sub, password_hash
		 FROM usuarios WHERE id = $1`, id,
	).Scan(&u.ID, &u.TenantID, &u.Email, &u.Nombre, &u.Rol, &u.SucursalID, &u.Estado, &u.SessionVersion, &u.GoogleSub, &u.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("usuario no encontrado: %w", err)
	}
	return u, nil
}

func (s *pgStore) ObtenerStaffPorGoogleSub(ctx context.Context, slug, googleSub string) (*UsuarioAuth, error) {
	return s.obtenerStaff(ctx, slug, `u.google_sub = $2`, googleSub)
}

func (s *pgStore) ObtenerStaffPorEmail(ctx context.Context, slug, email string) (*UsuarioAuth, error) {
	return s.obtenerStaff(ctx, slug, `lower(u.email) = lower($2)`, email)
}

func (s *pgStore) obtenerStaff(ctx context.Context, slug, condicion, valor string) (*UsuarioAuth, error) {
	u := &UsuarioAuth{}
	err := db.Pool.QueryRow(ctx,
		`SELECT u.id, u.tenant_id, u.email, u.nombre, u.rol, u.sucursal_id,
		        u.estado, u.session_version, u.google_sub, u.password_hash
		 FROM usuarios u
		 JOIN tenants t ON t.id = u.tenant_id
		 WHERE t.slug = $1 AND `+condicion+` AND u.rol <> 'admin'`,
		slug, valor,
	).Scan(&u.ID, &u.TenantID, &u.Email, &u.Nombre, &u.Rol, &u.SucursalID, &u.Estado, &u.SessionVersion, &u.GoogleSub, &u.PasswordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUsuarioNoEncontrado
		}
		return nil, fmt.Errorf("error consultando acceso del equipo: %w", err)
	}
	return u, nil
}

func (s *pgStore) VincularGoogleSub(ctx context.Context, usuarioID, googleSub string) error {
	resultado, err := db.Pool.Exec(ctx,
		`UPDATE usuarios
		 SET google_sub = $2
		 WHERE id = $1 AND (google_sub IS NULL OR google_sub = $2)`,
		usuarioID, googleSub,
	)
	if err != nil {
		return fmt.Errorf("error vinculando identidad de Google: %w", err)
	}
	if resultado.RowsAffected() != 1 {
		return errors.New("el usuario ya está vinculado a otra cuenta de Google")
	}
	return nil
}

func (s *pgStore) InvalidarSesiones(ctx context.Context, usuarioID string) error {
	resultado, err := db.Pool.Exec(ctx,
		`UPDATE usuarios SET session_version = session_version + 1, updated_at = now() WHERE id = $1`,
		usuarioID,
	)
	if err != nil {
		return fmt.Errorf("error invalidando sesiones: %w", err)
	}
	if resultado.RowsAffected() != 1 {
		return ErrUsuarioNoEncontrado
	}
	return nil
}

func (s *pgStore) GuardarToken(ctx context.Context, usuarioID, token string, expiresAt time.Time) (string, error) {
	var id string
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO magic_tokens (usuario_id, token, expires_at)
		 VALUES ($1, $2, $3) RETURNING id`,
		usuarioID, token, expiresAt,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("error guardando token: %w", err)
	}
	return id, nil
}

func (s *pgStore) ObtenerToken(ctx context.Context, token string) (*MagicToken, error) {
	mt := &MagicToken{}
	err := db.Pool.QueryRow(ctx,
		`SELECT id, usuario_id, token, expires_at, used_at
		 FROM magic_tokens WHERE token = $1`,
		token,
	).Scan(&mt.ID, &mt.UsuarioID, &mt.Token, &mt.ExpiresAt, &mt.UsedAt)
	if err != nil {
		return nil, fmt.Errorf("token no encontrado: %w", err)
	}
	return mt, nil
}

func (s *pgStore) MarcarTokenUsado(ctx context.Context, tokenID string) error {
	_, err := db.Pool.Exec(ctx,
		`UPDATE magic_tokens SET used_at = now() WHERE id = $1`, tokenID,
	)
	return err
}
