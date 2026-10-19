package sucursal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/db"
	"github.com/jackc/pgx/v5"
)

type Store interface {
	Crear(ctx context.Context, tenantID string, input SucursalInput) (*Sucursal, error)
	ObtenerPorID(ctx context.Context, id string, tenantID string) (*Sucursal, error)
	Listar(ctx context.Context, tenantID string) ([]Sucursal, error)
	Actualizar(ctx context.Context, id string, tenantID string, input SucursalInput) (*Sucursal, error)
	Eliminar(ctx context.Context, id string, tenantID string) error

	CrearSector(ctx context.Context, sucursalID string, tenantID string, input SectorInput) (*Sector, error)
	ListarSectores(ctx context.Context, sucursalID string, tenantID string) ([]Sector, error)
	EliminarSector(ctx context.Context, id, tenantID, sucursalAlcance string) error
}

type pgStore struct{}

func NuevoStore() Store { return &pgStore{} }

func (s *pgStore) Crear(ctx context.Context, tenantID string, input SucursalInput) (*Sucursal, error) {
	horariosJSON, err := json.Marshal(input.Horarios)
	if err != nil {
		return nil, fmt.Errorf("error serializando horarios: %w", err)
	}

	var suc Sucursal
	var horariosBytes []byte

	err = db.Pool.QueryRow(ctx,
		`INSERT INTO sucursales (tenant_id, nombre, whatsapp, email, telefono, horarios, mp_access_token, mp_public_key, mp_activo)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id, tenant_id, nombre, whatsapp, email, telefono, horarios, mp_access_token, mp_public_key, mp_activo, created_at`,
		tenantID, input.Nombre, input.Whatsapp, input.Email, input.Telefono, horariosJSON, input.MPAccessToken, input.MPPublicKey, input.MPActivo,
	).Scan(&suc.ID, &suc.TenantID, &suc.Nombre, &suc.Whatsapp, &suc.Email, &suc.Telefono, &horariosBytes, &suc.MPAccessToken, &suc.MPPublicKey, &suc.MPActivo, &suc.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("error insertando sucursal: %w", err)
	}

	if len(horariosBytes) > 0 {
		_ = json.Unmarshal(horariosBytes, &suc.Horarios)
	}

	return &suc, nil
}

func (s *pgStore) ObtenerPorID(ctx context.Context, id string, tenantID string) (*Sucursal, error) {
	var suc Sucursal
	var horariosBytes []byte

	err := db.Pool.QueryRow(ctx,
		`SELECT id, tenant_id, nombre, whatsapp, email, telefono, horarios, mp_access_token, mp_public_key, mp_activo, created_at
		 FROM sucursales
		 WHERE id = $1 AND tenant_id = $2`,
		id, tenantID,
	).Scan(&suc.ID, &suc.TenantID, &suc.Nombre, &suc.Whatsapp, &suc.Email, &suc.Telefono, &horariosBytes, &suc.MPAccessToken, &suc.MPPublicKey, &suc.MPActivo, &suc.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("error obteniendo sucursal: %w", err)
	}

	if len(horariosBytes) > 0 {
		_ = json.Unmarshal(horariosBytes, &suc.Horarios)
	}

	return &suc, nil
}

func (s *pgStore) Listar(ctx context.Context, tenantID string) ([]Sucursal, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT id, tenant_id, nombre, whatsapp, email, telefono, horarios, mp_access_token, mp_public_key, mp_activo, created_at
		 FROM sucursales
		 WHERE tenant_id = $1
		 ORDER BY created_at ASC`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("error listando sucursales: %w", err)
	}
	defer rows.Close()

	var sucursales []Sucursal
	for rows.Next() {
		var suc Sucursal
		var horariosBytes []byte
		err := rows.Scan(&suc.ID, &suc.TenantID, &suc.Nombre, &suc.Whatsapp, &suc.Email, &suc.Telefono, &horariosBytes, &suc.MPAccessToken, &suc.MPPublicKey, &suc.MPActivo, &suc.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("error leyendo sucursal: %w", err)
		}
		if len(horariosBytes) > 0 {
			_ = json.Unmarshal(horariosBytes, &suc.Horarios)
		}
		sucursales = append(sucursales, suc)
	}

	return sucursales, nil
}

func (s *pgStore) Actualizar(ctx context.Context, id string, tenantID string, input SucursalInput) (*Sucursal, error) {
	actual, err := s.ObtenerPorID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	nombre := actual.Nombre
	if strings.TrimSpace(input.Nombre) != "" {
		nombre = input.Nombre
	}

	whatsapp := actual.Whatsapp
	if input.Whatsapp != nil {
		whatsapp = input.Whatsapp
	}

	email := actual.Email
	if input.Email != nil {
		email = input.Email
	}

	telefono := actual.Telefono
	if input.Telefono != nil {
		telefono = input.Telefono
	}

	horarios := actual.Horarios
	if input.Horarios != nil {
		horarios = input.Horarios
	}
	horariosJSON, err := json.Marshal(horarios)
	if err != nil {
		return nil, fmt.Errorf("error serializando horarios: %w", err)
	}

	mpAccessToken := actual.MPAccessToken
	if input.MPAccessToken != nil {
		mpAccessToken = input.MPAccessToken
	}
	mpPublicKey := actual.MPPublicKey
	if input.MPPublicKey != nil {
		mpPublicKey = input.MPPublicKey
	}
	mpActivo := actual.MPActivo
	if input.MPActivo != nil {
		mpActivo = input.MPActivo
	}

	var suc Sucursal
	var horariosBytes []byte

	err = db.Pool.QueryRow(ctx,
		`UPDATE sucursales
		 SET nombre = $1, whatsapp = $2, email = $3, telefono = $4, horarios = $5,
		     mp_access_token = $6, mp_public_key = $7, mp_activo = $8
		 WHERE id = $9 AND tenant_id = $10
		 RETURNING id, tenant_id, nombre, whatsapp, email, telefono, horarios, mp_access_token, mp_public_key, mp_activo, created_at`,
		nombre, whatsapp, email, telefono, horariosJSON,
		mpAccessToken, mpPublicKey, mpActivo, id, tenantID,
	).Scan(&suc.ID, &suc.TenantID, &suc.Nombre, &suc.Whatsapp, &suc.Email, &suc.Telefono, &horariosBytes, &suc.MPAccessToken, &suc.MPPublicKey, &suc.MPActivo, &suc.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("error actualizando sucursal: %w", err)
	}

	if len(horariosBytes) > 0 {
		_ = json.Unmarshal(horariosBytes, &suc.Horarios)
	}

	return &suc, nil
}

func (s *pgStore) Eliminar(ctx context.Context, id string, tenantID string) error {
	cmdTag, err := db.Pool.Exec(ctx,
		`DELETE FROM sucursales WHERE id = $1 AND tenant_id = $2`,
		id, tenantID,
	)
	if err != nil {
		return fmt.Errorf("error eliminando sucursal: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *pgStore) CrearSector(ctx context.Context, sucursalID string, tenantID string, input SectorInput) (*Sector, error) {
	// Validar que la sucursal pertenezca al tenant antes de insertar el sector
	var exists bool
	err := db.Pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM sucursales WHERE id = $1 AND tenant_id = $2)`,
		sucursalID, tenantID,
	).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotFound
	}

	var sec Sector
	err = db.Pool.QueryRow(ctx,
		`INSERT INTO sectores (sucursal_id, nombre)
		 VALUES ($1, $2)
		 RETURNING id, sucursal_id, nombre`,
		sucursalID, input.Nombre,
	).Scan(&sec.ID, &sec.SucursalID, &sec.Nombre)

	if err != nil {
		return nil, fmt.Errorf("error insertando sector: %w", err)
	}

	return &sec, nil
}

func (s *pgStore) ListarSectores(ctx context.Context, sucursalID string, tenantID string) ([]Sector, error) {
	// Validar que la sucursal pertenezca al tenant
	var exists bool
	err := db.Pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM sucursales WHERE id = $1 AND tenant_id = $2)`,
		sucursalID, tenantID,
	).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotFound
	}

	rows, err := db.Pool.Query(ctx,
		`SELECT id, sucursal_id, nombre
		 FROM sectores
		 WHERE sucursal_id = $1
		 ORDER BY nombre ASC`,
		sucursalID,
	)
	if err != nil {
		return nil, fmt.Errorf("error listando sectores: %w", err)
	}
	defer rows.Close()

	var sectores []Sector
	for rows.Next() {
		var sec Sector
		err := rows.Scan(&sec.ID, &sec.SucursalID, &sec.Nombre)
		if err != nil {
			return nil, fmt.Errorf("error leyendo sector: %w", err)
		}
		sectores = append(sectores, sec)
	}

	return sectores, nil
}

func (s *pgStore) EliminarSector(ctx context.Context, id, tenantID, sucursalAlcance string) error {
	// El duenio puede operar sobre todo el tenant; el equipo queda limitado a
	// la sucursal incluida en su sesion.
	cmdTag, err := db.Pool.Exec(ctx,
		`DELETE FROM sectores
		 WHERE id = $1
		   AND sucursal_id IN (
			SELECT id
			FROM sucursales
			WHERE tenant_id = $2
			  AND ($3 = '' OR id = $3::uuid)
		   )`,
		id, tenantID, sucursalAlcance,
	)
	if err != nil {
		return fmt.Errorf("error eliminando sector: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return ErrSectorNotFound
	}

	return nil
}
