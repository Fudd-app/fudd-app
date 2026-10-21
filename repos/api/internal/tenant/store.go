package tenant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Store interface {
	Crear(ctx context.Context, input OnboardingInput) (*Tenant, error)
	ObtenerPorID(ctx context.Context, id string) (*Tenant, error)
	Actualizar(ctx context.Context, id string, input ActualizarTenantInput) (*Tenant, error)
	EmailAdminEnUso(ctx context.Context, email string) (bool, error)
	SlugEnUso(ctx context.Context, slug string) (bool, error)
	ObtenerEstadoCuotas(ctx context.Context, tenantID string) (*EstadoCuotas, error)
	RegistrarSolicitudUpgrade(ctx context.Context, tenantID, nota string) (*Tenant, error)
}

type pgStore struct{}

func NuevoStore() Store { return &pgStore{} }

func (s *pgStore) Crear(ctx context.Context, input OnboardingInput) (*Tenant, error) {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var t Tenant
	err = tx.QueryRow(ctx,
		`INSERT INTO tenants (nombre, nombre_fantasia, rubro, slug, plan, plan_desde, plan_hasta, trial_hasta)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id, nombre, nombre_fantasia, rubro, slug, plan, plan_desde, plan_hasta, trial_hasta, created_at`,
		input.Nombre, input.NombreFantasia, input.Rubro, input.Slug,
		input.Plan, input.PlanDesde, input.PlanHasta, input.TrialHasta,
	).Scan(&t.ID, &t.Nombre, &t.NombreFantasia, &t.Rubro, &t.Slug,
		&t.Plan, &t.PlanDesde, &t.PlanHasta, &t.TrialHasta, &t.CreatedAt)
	if err != nil {
		if isUniqueConstraint(err, "tenants_slug_key") {
			return nil, ErrSlugConflict
		}
		return nil, fmt.Errorf("error creando tenant: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO usuarios (tenant_id, email, nombre, rol, password_hash, google_sub)
		 VALUES ($1, $2, $3, 'admin', NULLIF($4, ''), NULLIF($5, ''))`,
		t.ID, input.EmailAdmin, input.NombreAdmin, input.PasswordHash, input.GoogleSub,
	)
	if err != nil {
		if isUniqueConstraint(err, "usuarios_email_key") || isUniqueConstraint(err, "idx_usuarios_email_lower") ||
			isUniqueConstraint(err, "idx_usuarios_admin_email") || isUniqueConstraint(err, "idx_usuarios_admin_google_sub") {
			return nil, ErrEmailAdminConflict
		}
		return nil, fmt.Errorf("error creando usuario admin: %w", err)
	}

	horariosJSON, err := json.Marshal(input.Horarios)
	if err != nil {
		return nil, fmt.Errorf("error serializando horarios de sucursal: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO sucursales (tenant_id, nombre, whatsapp, email, horarios)
		 VALUES ($1, $2, $3, $4, $5)`,
		t.ID, input.SucursalNombre, optionalString(input.Whatsapp), optionalString(input.EmailSucursal), horariosJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("error creando sucursal default: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &t, nil
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func isUniqueConstraint(err error, constraintName string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == constraintName
}

func (s *pgStore) ObtenerPorID(ctx context.Context, id string) (*Tenant, error) {
	t := &Tenant{}
	var datosFiscalesBytes []byte
	err := db.Pool.QueryRow(ctx,
		`SELECT id, nombre, nombre_fantasia, rubro, descripcion, email_contacto, whatsapp,
		        logo_url, color_primario, estilo_visual, datos_fiscales, google_review_url,
		        mp_access_token, mp_public_key, COALESCE(mp_activo, false),
		        COALESCE(plan, 'free'), plan_desde, plan_hasta, trial_hasta, upgrade_solicitado_at, upgrade_nota,
		        COALESCE(mostrar_marca_agua, true), color_secundario, color_categoria, color_accion, tipo_fuente,
		        slug, created_at
		 FROM tenants WHERE id = $1`, id,
	).Scan(&t.ID, &t.Nombre, &t.NombreFantasia, &t.Rubro, &t.Descripcion, &t.EmailContacto, &t.Whatsapp,
		&t.LogoURL, &t.ColorPrimario, &t.EstiloVisual, &datosFiscalesBytes, &t.GoogleReviewURL,
		&t.MPAccessToken, &t.MPPublicKey, &t.MPActivo,
		&t.Plan, &t.PlanDesde, &t.PlanHasta, &t.TrialHasta, &t.UpgradeSolicitadoAt, &t.UpgradeNota,
		&t.MostrarMarcaAgua, &t.ColorSecundario, &t.ColorCategoria, &t.ColorAccion, &t.TipoFuente,
		&t.Slug, &t.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("error obteniendo tenant: %w", err)
	}
	if len(datosFiscalesBytes) > 0 {
		_ = json.Unmarshal(datosFiscalesBytes, &t.DatosFiscales)
	}
	return t, nil
}

func (s *pgStore) Actualizar(ctx context.Context, id string, input ActualizarTenantInput) (*Tenant, error) {
	actual, err := s.ObtenerPorID(ctx, id)
	if err != nil {
		return nil, err
	}

	nombre := actual.Nombre
	if input.Nombre != nil && *input.Nombre != "" {
		nombre = *input.Nombre
	}

	nombreFantasia := actual.NombreFantasia
	if input.NombreFantasia != nil {
		nombreFantasia = *input.NombreFantasia
	}

	rubro := actual.Rubro
	if input.Rubro != nil && *input.Rubro != "" {
		rubro = *input.Rubro
	}

	descripcion := actual.Descripcion
	if input.Descripcion != nil {
		descripcion = input.Descripcion
	}

	emailContacto := actual.EmailContacto
	if input.EmailContacto != nil {
		emailContacto = input.EmailContacto
	}

	whatsapp := actual.Whatsapp
	if input.Whatsapp != nil {
		whatsapp = input.Whatsapp
	}

	logoURL := actual.LogoURL
	if input.LogoURL != nil {
		logoURL = input.LogoURL
	}

	colorPrimario := actual.ColorPrimario
	if input.ColorPrimario != nil {
		colorPrimario = input.ColorPrimario
	}

	estiloVisual := actual.EstiloVisual
	if input.EstiloVisual != nil {
		estiloVisual = input.EstiloVisual
	}

	datosFiscales := actual.DatosFiscales
	if input.DatosFiscales != nil {
		datosFiscales = input.DatosFiscales
	}
	if datosFiscales == nil {
		datosFiscales = make(map[string]any)
	}
	datosFiscalesJSON, err := json.Marshal(datosFiscales)
	if err != nil {
		return nil, fmt.Errorf("error serializando datos fiscales: %w", err)
	}

	googleReviewURL := actual.GoogleReviewURL
	if input.GoogleReviewURL != nil {
		googleReviewURL = input.GoogleReviewURL
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
		mpActivo = *input.MPActivo
	}

	mostrarMarcaAgua := actual.MostrarMarcaAgua
	if input.MostrarMarcaAgua != nil {
		mostrarMarcaAgua = *input.MostrarMarcaAgua
	}

	colorSecundario := actual.ColorSecundario
	if input.ColorSecundario != nil {
		colorSecundario = input.ColorSecundario
	}

	colorCategoria := actual.ColorCategoria
	if input.ColorCategoria != nil {
		colorCategoria = input.ColorCategoria
	}

	colorAccion := actual.ColorAccion
	if input.ColorAccion != nil {
		colorAccion = input.ColorAccion
	}

	tipoFuente := actual.TipoFuente
	if input.TipoFuente != nil {
		tipoFuente = input.TipoFuente
	}

	var t Tenant
	var datosFiscalesBytes []byte

	err = db.Pool.QueryRow(ctx,
		`UPDATE tenants
		 SET nombre = $1, nombre_fantasia = $2, rubro = $3, descripcion = $4,
		     email_contacto = $5, whatsapp = $6, logo_url = $7, color_primario = $8,
		     estilo_visual = $9, datos_fiscales = $10, google_review_url = $11,
		     mp_access_token = $12, mp_public_key = $13, mp_activo = $14,
		     mostrar_marca_agua = $15, color_secundario = $16, color_categoria = $17,
		     color_accion = $18, tipo_fuente = $19
		 WHERE id = $20
		 RETURNING id, nombre, nombre_fantasia, rubro, descripcion, email_contacto, whatsapp,
		           logo_url, color_primario, estilo_visual, datos_fiscales, google_review_url,
		           mp_access_token, mp_public_key, COALESCE(mp_activo, false),
		           COALESCE(plan, 'free'), plan_desde, plan_hasta, trial_hasta, upgrade_solicitado_at, upgrade_nota,
		           COALESCE(mostrar_marca_agua, true), color_secundario, color_categoria, color_accion, tipo_fuente,
		           slug, created_at`,
		nombre, nombreFantasia, rubro, descripcion,
		emailContacto, whatsapp, logoURL, colorPrimario,
		estiloVisual, datosFiscalesJSON, googleReviewURL,
		mpAccessToken, mpPublicKey, mpActivo,
		mostrarMarcaAgua, colorSecundario, colorCategoria, colorAccion, tipoFuente, id,
	).Scan(&t.ID, &t.Nombre, &t.NombreFantasia, &t.Rubro, &t.Descripcion, &t.EmailContacto, &t.Whatsapp,
		&t.LogoURL, &t.ColorPrimario, &t.EstiloVisual, &datosFiscalesBytes, &t.GoogleReviewURL,
		&t.MPAccessToken, &t.MPPublicKey, &t.MPActivo,
		&t.Plan, &t.PlanDesde, &t.PlanHasta, &t.TrialHasta, &t.UpgradeSolicitadoAt, &t.UpgradeNota,
		&t.MostrarMarcaAgua, &t.ColorSecundario, &t.ColorCategoria, &t.ColorAccion, &t.TipoFuente,
		&t.Slug, &t.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("error actualizando tenant: %w", err)
	}
	if len(datosFiscalesBytes) > 0 {
		_ = json.Unmarshal(datosFiscalesBytes, &t.DatosFiscales)
	}
	return &t, nil
}

func (s *pgStore) SlugEnUso(ctx context.Context, slug string) (bool, error) {
	var existe bool
	err := db.Pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM tenants WHERE slug = $1)`,
		slug,
	).Scan(&existe)
	if err != nil {
		return false, fmt.Errorf("error verificando slug: %w", err)
	}
	return existe, nil
}

func (s *pgStore) EmailAdminEnUso(ctx context.Context, email string) (bool, error) {
	var existe bool
	err := db.Pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM usuarios WHERE lower(email) = lower($1) AND rol = 'admin')`,
		email,
	).Scan(&existe)
	if err != nil {
		return false, fmt.Errorf("error verificando email admin: %w", err)
	}
	return existe, nil
}

func (s *pgStore) ObtenerEstadoCuotas(ctx context.Context, tenantID string) (*EstadoCuotas, error) {
	estado := &EstadoCuotas{
		Uso: make(map[Recurso]int),
	}
	var mesasActivas, productos, sucursales int
	query := `SELECT COALESCE(t.plan, 'free'), t.plan_desde, t.plan_hasta, t.trial_hasta,
	                 t.upgrade_solicitado_at, t.upgrade_nota,
	                 (SELECT COUNT(*) FROM mesas m
	                    JOIN sucursales su ON su.id = m.sucursal_id
	                   WHERE su.tenant_id = t.id AND m.estado = 'activa'),
	                 (SELECT COUNT(*) FROM articulos a WHERE a.tenant_id = t.id),
	                 (SELECT COUNT(*) FROM sucursales su WHERE su.tenant_id = t.id)
	            FROM tenants t WHERE t.id = $1`

	err := db.Pool.QueryRow(ctx, query, tenantID).Scan(
		&estado.Plan,
		&estado.PlanDesde,
		&estado.PlanHasta,
		&estado.TrialHasta,
		&estado.UpgradeSolicitadoAt,
		&estado.UpgradeNota,
		&mesasActivas,
		&productos,
		&sucursales,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("error obteniendo estado de cuotas: %w", err)
	}

	estado.Uso[RecursoMesas] = mesasActivas
	estado.Uso[RecursoProductos] = productos
	estado.Uso[RecursoSucursales] = sucursales
	return estado, nil
}

func (s *pgStore) RegistrarSolicitudUpgrade(ctx context.Context, tenantID, nota string) (*Tenant, error) {
	var t Tenant
	var datosFiscalesBytes []byte
	notaPtr := optionalString(nota)

	query := `UPDATE tenants
	             SET upgrade_solicitado_at = NOW(),
	                 upgrade_nota = $2
	           WHERE id = $1
	       RETURNING id, nombre, nombre_fantasia, rubro, descripcion, email_contacto, whatsapp,
	                 logo_url, color_primario, estilo_visual, datos_fiscales, google_review_url,
	                 mp_access_token, mp_public_key, COALESCE(mp_activo, false),
	                 COALESCE(plan, 'free'), plan_desde, plan_hasta, trial_hasta, upgrade_solicitado_at, upgrade_nota,
	                 slug, created_at`

	err := db.Pool.QueryRow(ctx, query, tenantID, notaPtr).Scan(
		&t.ID, &t.Nombre, &t.NombreFantasia, &t.Rubro, &t.Descripcion, &t.EmailContacto, &t.Whatsapp,
		&t.LogoURL, &t.ColorPrimario, &t.EstiloVisual, &datosFiscalesBytes, &t.GoogleReviewURL,
		&t.MPAccessToken, &t.MPPublicKey, &t.MPActivo,
		&t.Plan, &t.PlanDesde, &t.PlanHasta, &t.TrialHasta, &t.UpgradeSolicitadoAt, &t.UpgradeNota,
		&t.Slug, &t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("error registrando solicitud de upgrade: %w", err)
	}
	if len(datosFiscalesBytes) > 0 {
		_ = json.Unmarshal(datosFiscalesBytes, &t.DatosFiscales)
	}
	return &t, nil
}
