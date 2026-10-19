package resenia

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Store interface {
	ObtenerContextoMesaPorQR(ctx context.Context, qrToken string) (*MesaContexto, error)
	VerificarPedidosEnCuenta(ctx context.Context, mesaID string, cuentaVersion int) (bool, error)
	VerificarResenaExiste(ctx context.Context, mesaID string, cuentaVersion int) (bool, error)
	CrearResena(ctx context.Context, mesaID, sucursalID, tenantID string, cuentaVersion, estrellas int, comentario *string) (*Resena, error)
	RegistrarClickGoogle(ctx context.Context, qrToken string) error
	SucursalPerteneceATenant(ctx context.Context, sucursalID, tenantID string) (bool, error)
	ObtenerResumen(ctx context.Context, tenantID, sucursalID string, desde, hasta *time.Time) (*ResumenResenas, error)
}

type pgStore struct{}

func NuevoStore() Store {
	return &pgStore{}
}

func (s *pgStore) ObtenerContextoMesaPorQR(ctx context.Context, qrToken string) (*MesaContexto, error) {
	ctxMesa := &MesaContexto{}
	err := db.Pool.QueryRow(ctx,
		`SELECT m.id, m.numero, m.sucursal_id, su.tenant_id, m.cuenta_version, m.estado
		 FROM mesas m
		 JOIN sucursales su ON su.id = m.sucursal_id
		 WHERE m.qr_token = $1`, qrToken,
	).Scan(&ctxMesa.MesaID, &ctxMesa.MesaNumero, &ctxMesa.SucursalID, &ctxMesa.TenantID, &ctxMesa.CuentaVersion, &ctxMesa.Estado)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("error obteniendo contexto de mesa por qr: %w", err)
	}
	return ctxMesa, nil
}

func (s *pgStore) VerificarPedidosEnCuenta(ctx context.Context, mesaID string, cuentaVersion int) (bool, error) {
	var existe bool
	err := db.Pool.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM pedidos
			WHERE mesa_id = $1 AND cuenta_version = $2
		)`, mesaID, cuentaVersion,
	).Scan(&existe)
	if err != nil {
		return false, fmt.Errorf("error verificando pedidos en cuenta: %w", err)
	}
	return existe, nil
}

func (s *pgStore) VerificarResenaExiste(ctx context.Context, mesaID string, cuentaVersion int) (bool, error) {
	var existe bool
	err := db.Pool.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM resenas
			WHERE mesa_id = $1 AND cuenta_version = $2
		)`, mesaID, cuentaVersion,
	).Scan(&existe)
	if err != nil {
		return false, fmt.Errorf("error verificando existencia de reseña: %w", err)
	}
	return existe, nil
}

func (s *pgStore) CrearResena(ctx context.Context, mesaID, sucursalID, tenantID string, cuentaVersion, estrellas int, comentario *string) (*Resena, error) {
	r := &Resena{}
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO resenas (mesa_id, sucursal_id, tenant_id, cuenta_version, estrellas, comentario)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, mesa_id, sucursal_id, tenant_id, cuenta_version, estrellas, comentario, google_cliqueado, created_at, updated_at`,
		mesaID, sucursalID, tenantID, cuentaVersion, estrellas, comentario,
	).Scan(&r.ID, &r.MesaID, &r.SucursalID, &r.TenantID, &r.CuentaVersion, &r.Estrellas, &r.Comentario, &r.GoogleCliqueado, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrResenaDuplicada
		}
		return nil, fmt.Errorf("error guardando reseña: %w", err)
	}
	return r, nil
}

func (s *pgStore) RegistrarClickGoogle(ctx context.Context, qrToken string) error {
	tag, err := db.Pool.Exec(ctx,
		`UPDATE resenas r
		 SET google_cliqueado = true, updated_at = now()
		 FROM mesas m
		 WHERE r.mesa_id = m.id
		   AND m.qr_token = $1
		   AND r.cuenta_version = m.cuenta_version`, qrToken,
	)
	if err != nil {
		return fmt.Errorf("error registrando clic de google: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *pgStore) SucursalPerteneceATenant(ctx context.Context, sucursalID, tenantID string) (bool, error) {
	var pertenece bool
	err := db.Pool.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM sucursales
			WHERE id = $1 AND tenant_id = $2
		)`, sucursalID, tenantID,
	).Scan(&pertenece)
	if err != nil {
		return false, fmt.Errorf("error verificando pertenencia de sucursal: %w", err)
	}
	return pertenece, nil
}

func (s *pgStore) ObtenerResumen(ctx context.Context, tenantID, sucursalID string, desde, hasta *time.Time) (*ResumenResenas, error) {
	distribucion := map[int]int{
		1: 0,
		2: 0,
		3: 0,
		4: 0,
		5: 0,
	}

	rowsDist, err := db.Pool.Query(ctx,
		`SELECT estrellas, COUNT(*)::int
		 FROM resenas r
		 WHERE r.tenant_id = $1
		   AND (NULLIF($2, '')::uuid IS NULL OR r.sucursal_id = NULLIF($2, '')::uuid)
		   AND ($3::timestamptz IS NULL OR r.created_at >= $3)
		   AND ($4::timestamptz IS NULL OR r.created_at <= $4)
		 GROUP BY estrellas`,
		tenantID, sucursalID, desde, hasta,
	)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo distribución de estrellas: %w", err)
	}
	defer rowsDist.Close()

	total := 0
	sumaPuntaje := 0
	promotores := 0 // estrellas 4 y 5 para CSAT

	for rowsDist.Next() {
		var estrellas, cantidad int
		if err := rowsDist.Scan(&estrellas, &cantidad); err != nil {
			return nil, fmt.Errorf("error leyendo distribución de estrellas: %w", err)
		}
		distribucion[estrellas] = cantidad
		total += cantidad
		sumaPuntaje += estrellas * cantidad
		if estrellas >= 4 {
			promotores += cantidad
		}
	}
	if err := rowsDist.Err(); err != nil {
		return nil, fmt.Errorf("error iterando distribución de estrellas: %w", err)
	}

	var clicsGoogle int
	err = db.Pool.QueryRow(ctx,
		`SELECT COUNT(*)::int
		 FROM resenas r
		 WHERE r.tenant_id = $1
		   AND (NULLIF($2, '')::uuid IS NULL OR r.sucursal_id = NULLIF($2, '')::uuid)
		   AND ($3::timestamptz IS NULL OR r.created_at >= $3)
		   AND ($4::timestamptz IS NULL OR r.created_at <= $4)
		   AND r.google_cliqueado = true`,
		tenantID, sucursalID, desde, hasta,
	).Scan(&clicsGoogle)
	if err != nil {
		return nil, fmt.Errorf("error contando clics de google: %w", err)
	}

	var promedio float64
	var csat float64
	if total > 0 {
		promedio = math.Round((float64(sumaPuntaje)/float64(total))*10) / 10
		csat = math.Round((float64(promotores)/float64(total))*1000) / 10
	}

	rowsList, err := db.Pool.Query(ctx,
		`SELECT r.id, r.mesa_id, m.numero, r.sucursal_id, su.nombre, r.cuenta_version,
		        r.estrellas, r.comentario, r.google_cliqueado, r.created_at
		 FROM resenas r
		 JOIN mesas m ON m.id = r.mesa_id
		 JOIN sucursales su ON su.id = r.sucursal_id
		 WHERE r.tenant_id = $1
		   AND (NULLIF($2, '')::uuid IS NULL OR r.sucursal_id = NULLIF($2, '')::uuid)
		   AND ($3::timestamptz IS NULL OR r.created_at >= $3)
		   AND ($4::timestamptz IS NULL OR r.created_at <= $4)
		 ORDER BY r.created_at DESC
		 LIMIT 100`,
		tenantID, sucursalID, desde, hasta,
	)
	if err != nil {
		return nil, fmt.Errorf("error listando reseñas: %w", err)
	}
	defer rowsList.Close()

	resenas := make([]ResenaItem, 0)
	for rowsList.Next() {
		var item ResenaItem
		if err := rowsList.Scan(
			&item.ID,
			&item.MesaID,
			&item.MesaNumero,
			&item.SucursalID,
			&item.SucursalNombre,
			&item.CuentaVersion,
			&item.Estrellas,
			&item.Comentario,
			&item.GoogleCliqueado,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("error leyendo item de reseña: %w", err)
		}
		resenas = append(resenas, item)
	}
	if err := rowsList.Err(); err != nil {
		return nil, fmt.Errorf("error iterando lista de reseñas: %w", err)
	}

	return &ResumenResenas{
		Promedio:     promedio,
		Total:        total,
		CSAT:         csat,
		Distribucion: distribucion,
		ClicsGoogle:  clicsGoogle,
		Resenas:      resenas,
	}, nil
}
