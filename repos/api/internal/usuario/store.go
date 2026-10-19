package usuario

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Store interface {
	SucursalPertenece(context.Context, string, string) (bool, error)
	CrearInvitacion(context.Context, string, string, string, string, string, string, time.Time) (*Invitacion, error)
	ListarUsuarios(context.Context, string, string) ([]Usuario, error)
	ListarInvitaciones(context.Context, string, string) ([]Invitacion, error)
	ObtenerUsuario(context.Context, string, string) (*Usuario, error)
	ActualizarUsuario(context.Context, string, string, string, string) (*Usuario, error)
	EliminarUsuario(context.Context, string, string) error
	CancelarInvitacion(context.Context, string, string, string) error
	ReenviarInvitacion(context.Context, string, string, string, string, string, time.Time) (*Invitacion, error)
	ObtenerInvitacionPorToken(context.Context, string) (*Invitacion, error)
	AceptarInvitacion(context.Context, string, string, string, time.Time) (*Usuario, error)
}

type pgStore struct{}

func NuevoStore() Store { return &pgStore{} }

const selectUsuario = `SELECT u.id,u.tenant_id,u.sucursal_id,s.nombre,u.email,u.nombre,
 u.rol,u.estado,u.created_at,u.updated_at
 FROM usuarios u LEFT JOIN sucursales s ON s.id=u.sucursal_id`

const selectInvitacion = `SELECT i.id,i.tenant_id,t.nombre_fantasia,t.slug,i.sucursal_id,
 s.nombre,i.email,i.rol,i.estado,i.expires_at,i.used_at,i.created_at
 FROM invitaciones_equipo i JOIN tenants t ON t.id=i.tenant_id
 JOIN sucursales s ON s.id=i.sucursal_id`

func scanUsuario(row pgx.Row) (*Usuario, error) {
	var u Usuario
	err := row.Scan(&u.ID, &u.TenantID, &u.SucursalID, &u.SucursalNombre, &u.Email,
		&u.Nombre, &u.Rol, &u.Estado, &u.CreatedAt, &u.UpdatedAt)
	return &u, err
}

func scanInvitacion(row pgx.Row) (*Invitacion, error) {
	var i Invitacion
	err := row.Scan(&i.ID, &i.TenantID, &i.TenantNombre, &i.TenantSlug, &i.SucursalID,
		&i.SucursalNombre, &i.Email, &i.Rol, &i.Estado, &i.ExpiresAt, &i.UsedAt, &i.CreatedAt)
	return &i, err
}

func (s *pgStore) SucursalPertenece(ctx context.Context, tenantID, sucursalID string) (bool, error) {
	var existe bool
	err := db.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM sucursales WHERE id=$1 AND tenant_id=$2)`, sucursalID, tenantID).Scan(&existe)
	return existe, err
}

func (s *pgStore) CrearInvitacion(ctx context.Context, tenantID, sucursalID, email, rol, tokenHash, actorID string, expiresAt time.Time) (*Invitacion, error) {
	var id string
	err := db.Pool.QueryRow(ctx, `INSERT INTO invitaciones_equipo
	 (tenant_id,sucursal_id,email,rol,token_hash,invitado_por_id,invitado_por_nombre,expires_at)
	 SELECT $1,$2,$3,$4,$5,$6,COALESCE((SELECT nombre FROM usuarios WHERE id=$6),'Equipo'),$7
	 WHERE NOT EXISTS (SELECT 1 FROM usuarios WHERE tenant_id=$1 AND lower(email)=lower($3))
	 RETURNING id`, tenantID, sucursalID, email, rol, tokenHash, actorID, expiresAt).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrEmailConflict
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrEmailConflict
		}
		return nil, fmt.Errorf("crear invitación: %w", err)
	}
	return s.obtenerInvitacionID(ctx, tenantID, id)
}

func (s *pgStore) ListarUsuarios(ctx context.Context, tenantID, sucursalID string) ([]Usuario, error) {
	query, args := selectUsuario+` WHERE u.tenant_id=$1`, []any{tenantID}
	if sucursalID != "" {
		query += ` AND u.sucursal_id=$2`
		args = append(args, sucursalID)
	}
	rows, err := db.Pool.Query(ctx, query+` ORDER BY u.created_at`, args...)
	if err != nil {
		return nil, fmt.Errorf("listar usuarios: %w", err)
	}
	defer rows.Close()
	resultado := make([]Usuario, 0)
	for rows.Next() {
		u, err := scanUsuario(rows)
		if err != nil {
			return nil, err
		}
		resultado = append(resultado, *u)
	}
	return resultado, rows.Err()
}

func (s *pgStore) ListarInvitaciones(ctx context.Context, tenantID, sucursalID string) ([]Invitacion, error) {
	query, args := selectInvitacion+` WHERE i.tenant_id=$1 AND i.estado='pendiente'`, []any{tenantID}
	if sucursalID != "" {
		query += ` AND i.sucursal_id=$2`
		args = append(args, sucursalID)
	}
	rows, err := db.Pool.Query(ctx, query+` ORDER BY i.created_at DESC`, args...)
	if err != nil {
		return nil, fmt.Errorf("listar invitaciones: %w", err)
	}
	defer rows.Close()
	resultado := make([]Invitacion, 0)
	for rows.Next() {
		i, err := scanInvitacion(rows)
		if err != nil {
			return nil, err
		}
		resultado = append(resultado, *i)
	}
	return resultado, rows.Err()
}

func (s *pgStore) ObtenerUsuario(ctx context.Context, tenantID, id string) (*Usuario, error) {
	u, err := scanUsuario(db.Pool.QueryRow(ctx, selectUsuario+` WHERE u.tenant_id=$1 AND u.id=$2`, tenantID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return u, err
}

func (s *pgStore) ActualizarUsuario(ctx context.Context, tenantID, id, nombre, estado string) (*Usuario, error) {
	tag, err := db.Pool.Exec(ctx, `UPDATE usuarios SET nombre=$3,estado=$4,
	 session_version=session_version+CASE WHEN estado<>$4 THEN 1 ELSE 0 END,updated_at=now()
	 WHERE tenant_id=$1 AND id=$2`, tenantID, id, nombre, estado)
	if err != nil {
		return nil, fmt.Errorf("actualizar usuario: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return s.ObtenerUsuario(ctx, tenantID, id)
}

func (s *pgStore) EliminarUsuario(ctx context.Context, tenantID, id string) error {
	tag, err := db.Pool.Exec(ctx, `DELETE FROM usuarios WHERE tenant_id=$1 AND id=$2 AND rol<>'admin'`, tenantID, id)
	if err != nil {
		return fmt.Errorf("eliminar usuario: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *pgStore) CancelarInvitacion(ctx context.Context, tenantID, id, sucursalID string) error {
	query, args := `UPDATE invitaciones_equipo SET estado='cancelada',updated_at=now() WHERE tenant_id=$1 AND id=$2 AND estado='pendiente'`, []any{tenantID, id}
	if sucursalID != "" {
		query += ` AND sucursal_id=$3`
		args = append(args, sucursalID)
	}
	tag, err := db.Pool.Exec(ctx, query, args...)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrInvitacionNoValida
	}
	return nil
}

func (s *pgStore) ReenviarInvitacion(ctx context.Context, tenantID, id, sucursalAlcance, tokenHash, actorID string, expiresAt time.Time) (*Invitacion, error) {
	tx, err := db.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var sucursalID, email, rol string
	query, args := `UPDATE invitaciones_equipo SET estado='reemplazada',updated_at=now()
	 WHERE tenant_id=$1 AND id=$2 AND estado='pendiente'`, []any{tenantID, id}
	if sucursalAlcance != "" {
		query += ` AND sucursal_id=$3`
		args = append(args, sucursalAlcance)
	}
	query += ` RETURNING sucursal_id,email,rol`
	err = tx.QueryRow(ctx, query, args...).Scan(&sucursalID, &email, &rol)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvitacionNoValida
	}
	if err != nil {
		return nil, err
	}
	var nuevoID string
	err = tx.QueryRow(ctx, `INSERT INTO invitaciones_equipo
	 (tenant_id,sucursal_id,email,rol,token_hash,invitado_por_id,invitado_por_nombre,expires_at)
	 SELECT $1,$2,$3,$4,$5,$6,COALESCE((SELECT nombre FROM usuarios WHERE id=$6),'Equipo'),$7 RETURNING id`,
		tenantID, sucursalID, email, rol, tokenHash, actorID, expiresAt).Scan(&nuevoID)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.obtenerInvitacionID(ctx, tenantID, nuevoID)
}

func (s *pgStore) obtenerInvitacionID(ctx context.Context, tenantID, id string) (*Invitacion, error) {
	i, err := scanInvitacion(db.Pool.QueryRow(ctx, selectInvitacion+` WHERE i.tenant_id=$1 AND i.id=$2`, tenantID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvitacionNoValida
	}
	return i, err
}

func (s *pgStore) ObtenerInvitacionPorToken(ctx context.Context, tokenHash string) (*Invitacion, error) {
	i, err := scanInvitacion(db.Pool.QueryRow(ctx, selectInvitacion+` WHERE i.token_hash=$1`, tokenHash))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvitacionNoValida
	}
	return i, err
}

func (s *pgStore) AceptarInvitacion(ctx context.Context, tokenHash, googleSub, nombre string, ahora time.Time) (*Usuario, error) {
	tx, err := db.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var invitacionID, tenantID, sucursalID, email, rol, estado string
	var expiresAt time.Time
	err = tx.QueryRow(ctx, `SELECT id,tenant_id,sucursal_id,email,rol,estado,expires_at
	 FROM invitaciones_equipo WHERE token_hash=$1 FOR UPDATE`, tokenHash).
		Scan(&invitacionID, &tenantID, &sucursalID, &email, &rol, &estado, &expiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvitacionNoValida
	}
	if err != nil {
		return nil, err
	}
	if estado != "pendiente" {
		return nil, ErrInvitacionNoValida
	}
	if !expiresAt.After(ahora) {
		return nil, ErrInvitacionExpirada
	}
	var usuarioID string
	err = tx.QueryRow(ctx, `INSERT INTO usuarios (tenant_id,sucursal_id,email,nombre,rol,estado,google_sub)
	 VALUES ($1,$2,$3,$4,$5,'activo',$6) RETURNING id`, tenantID, sucursalID, email, nombre, rol, googleSub).Scan(&usuarioID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrEmailConflict
		}
		return nil, err
	}
	if _, err = tx.Exec(ctx, `UPDATE invitaciones_equipo SET estado='usada',used_at=$2,updated_at=$2 WHERE id=$1`, invitacionID, ahora); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.ObtenerUsuario(ctx, tenantID, usuarioID)
}
