package usuario

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/db"
	"github.com/jackc/pgx/v5/pgconn"
)

type Store interface {
	Crear(ctx context.Context, tenantID string, input UsuarioInput) (*Usuario, error)
	CrearStaff(ctx context.Context, tenantID string, input CrearUsuarioStaffInput) (*Usuario, error)
	Listar(ctx context.Context, tenantID string) ([]Usuario, error)
	Actualizar(ctx context.Context, id string, tenantID string, input ActualizarUsuarioInput) (*Usuario, error)
	ResetearPassword(ctx context.Context, id string, tenantID string, passwordHash string) error
	CambiarEstado(ctx context.Context, id string, tenantID string, nuevoEstado string) (*Usuario, error)
	Eliminar(ctx context.Context, id string, tenantID string) error
	CrearMagicToken(ctx context.Context, usuarioID string) (string, error)
}

type pgStore struct{}

func NuevoStore() Store { return &pgStore{} }

func (s *pgStore) obtenerUsuarioDetallado(ctx context.Context, id, tenantID string) (*Usuario, error) {
	var u Usuario
	err := db.Pool.QueryRow(ctx,
		`SELECT u.id, u.tenant_id, u.sucursal_id, su.nombre, u.username, COALESCE(u.email, ''),
		        u.nombre, u.rol, COALESCE(u.estado, 'activo'), u.created_at
		   FROM usuarios u
		   LEFT JOIN sucursales su ON su.id = u.sucursal_id
		  WHERE u.id = $1 AND u.tenant_id = $2 AND u.deleted_at IS NULL`,
		id, tenantID,
	).Scan(&u.ID, &u.TenantID, &u.SucursalID, &u.SucursalNombre, &u.Username, &u.Email,
		&u.Nombre, &u.Rol, &u.Estado, &u.CreatedAt)
	if err != nil {
		return nil, ErrNotFound
	}
	return &u, nil
}

func clasificarUniqueErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		if strings.Contains(pgErr.ConstraintName, "username") {
			return ErrUsernameConflict
		}
		return ErrEmailConflict
	}
	return nil
}

func (s *pgStore) Crear(ctx context.Context, tenantID string, input UsuarioInput) (*Usuario, error) {
	var id string
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO usuarios (tenant_id, email, nombre, rol, sucursal_id, estado)
		 VALUES ($1, NULLIF($2, ''), $3, $4, $5, 'activo')
		 RETURNING id`,
		tenantID, input.Email, input.Nombre, input.Rol, input.SucursalID,
	).Scan(&id)

	if err != nil {
		if conflictErr := clasificarUniqueErr(err); conflictErr != nil {
			return nil, conflictErr
		}
		return nil, fmt.Errorf("error creando usuario: %w", err)
	}

	return s.obtenerUsuarioDetallado(ctx, id, tenantID)
}

func (s *pgStore) CrearStaff(ctx context.Context, tenantID string, input CrearUsuarioStaffInput) (*Usuario, error) {
	var id string
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO usuarios (tenant_id, username, email, nombre, rol, sucursal_id, password_hash, estado)
		 VALUES ($1, $2, NULLIF($3, ''), $4, $5, $6, $7, 'activo')
		 RETURNING id`,
		tenantID, input.Username, input.Email, input.Nombre, input.Rol, input.SucursalID, input.PasswordHash,
	).Scan(&id)

	if err != nil {
		if conflictErr := clasificarUniqueErr(err); conflictErr != nil {
			return nil, conflictErr
		}
		return nil, fmt.Errorf("error creando miembro de staff: %w", err)
	}

	return s.obtenerUsuarioDetallado(ctx, id, tenantID)
}

func (s *pgStore) Listar(ctx context.Context, tenantID string) ([]Usuario, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT u.id, u.tenant_id, u.sucursal_id, su.nombre, u.username, COALESCE(u.email, ''),
		        u.nombre, u.rol, COALESCE(u.estado, 'activo'), u.created_at
		   FROM usuarios u
		   LEFT JOIN sucursales su ON su.id = u.sucursal_id
		  WHERE u.tenant_id = $1 AND u.deleted_at IS NULL
		  ORDER BY u.created_at ASC`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("error listando usuarios: %w", err)
	}
	defer rows.Close()

	var usuarios []Usuario
	for rows.Next() {
		var u Usuario
		err := rows.Scan(&u.ID, &u.TenantID, &u.SucursalID, &u.SucursalNombre, &u.Username, &u.Email,
			&u.Nombre, &u.Rol, &u.Estado, &u.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("error leyendo usuario: %w", err)
		}
		usuarios = append(usuarios, u)
	}

	return usuarios, nil
}

func (s *pgStore) Actualizar(ctx context.Context, id string, tenantID string, input ActualizarUsuarioInput) (*Usuario, error) {
	u, err := s.obtenerUsuarioDetallado(ctx, id, tenantID)
	if err != nil {
		return nil, ErrNotFound
	}

	nombre := u.Nombre
	if input.Nombre != nil && *input.Nombre != "" {
		nombre = *input.Nombre
	}
	rol := u.Rol
	if input.Rol != nil && *input.Rol != "" {
		rol = *input.Rol
	}
	username := u.Username
	if input.Username != nil {
		trimmed := strings.TrimSpace(*input.Username)
		if trimmed == "" {
			username = nil
		} else {
			username = &trimmed
		}
	}
	email := u.Email
	if input.Email != nil {
		email = strings.TrimSpace(*input.Email)
	}
	sucursalID := u.SucursalID
	if input.SucursalID != nil {
		trimmed := strings.TrimSpace(*input.SucursalID)
		if trimmed == "" {
			sucursalID = nil
		} else {
			sucursalID = &trimmed
		}
	}

	_, err = db.Pool.Exec(ctx,
		`UPDATE usuarios
		    SET nombre = $1,
		        rol = $2,
		        username = $3,
		        email = NULLIF($4, ''),
		        sucursal_id = $5
		  WHERE id = $6 AND tenant_id = $7 AND deleted_at IS NULL`,
		nombre, rol, username, email, sucursalID, id, tenantID,
	)
	if err != nil {
		if conflictErr := clasificarUniqueErr(err); conflictErr != nil {
			return nil, conflictErr
		}
		return nil, fmt.Errorf("error actualizando usuario: %w", err)
	}

	return s.obtenerUsuarioDetallado(ctx, id, tenantID)
}

func (s *pgStore) ResetearPassword(ctx context.Context, id string, tenantID string, passwordHash string) error {
	cmdTag, err := db.Pool.Exec(ctx,
		`UPDATE usuarios
		    SET password_hash = $1
		  WHERE id = $2 AND tenant_id = $3 AND deleted_at IS NULL`,
		passwordHash, id, tenantID,
	)
	if err != nil {
		return fmt.Errorf("error reseteando contraseña de usuario: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrNotFound
	}

	// Invalidar tokens mágicos pendientes del usuario al blanquear clave
	_, _ = db.Pool.Exec(ctx,
		`UPDATE magic_tokens SET used_at = NOW() WHERE usuario_id = $1 AND used_at IS NULL`,
		id,
	)

	return nil
}

func (s *pgStore) CambiarEstado(ctx context.Context, id string, tenantID string, nuevoEstado string) (*Usuario, error) {
	cmdTag, err := db.Pool.Exec(ctx,
		`UPDATE usuarios
		    SET estado = $1
		  WHERE id = $2 AND tenant_id = $3 AND deleted_at IS NULL`,
		nuevoEstado, id, tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("error cambiando estado de usuario: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}

	return s.obtenerUsuarioDetallado(ctx, id, tenantID)
}

func (s *pgStore) Eliminar(ctx context.Context, id string, tenantID string) error {
	// Soft delete (CA-5 US-83): preserva integridad referencial y libera username/email para el futuro
	cmdTag, err := db.Pool.Exec(ctx,
		`UPDATE usuarios
		    SET deleted_at = NOW(),
		        estado = 'archivado'
		  WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`,
		id, tenantID,
	)
	if err != nil {
		return fmt.Errorf("error eliminando usuario: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *pgStore) CrearMagicToken(ctx context.Context, usuarioID string) (string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}
	token := hex.EncodeToString(tokenBytes)
	expiresAt := time.Now().Add(15 * time.Minute)

	_, err := db.Pool.Exec(ctx,
		`INSERT INTO magic_tokens (usuario_id, token, expires_at)
		 VALUES ($1, $2, $3)`,
		usuarioID, token, expiresAt,
	)
	if err != nil {
		return "", fmt.Errorf("error guardando token de invitacion: %w", err)
	}

	return token, nil
}
