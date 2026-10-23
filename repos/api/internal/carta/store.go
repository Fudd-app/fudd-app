package carta

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Store define las operaciones de persistencia del módulo carta.
type Store interface {
	ListarCategorias(ctx context.Context, tenantID string) ([]Categoria, error)
	CrearCategoria(ctx context.Context, tenantID string, input CategoriaInput) (*Categoria, error)
	EliminarCategoria(ctx context.Context, id, tenantID string) error
	// BuscarOCrearCategoria retorna el ID de la categoría con ese nombre para ese tenant,
	// creándola si no existe. Usado en importaciones masivas.
	BuscarOCrearCategoria(ctx context.Context, tenantID, nombre string) (string, error)
	ListarArticulos(ctx context.Context, tenantID string) ([]Articulo, error)
	ExisteArticuloImportado(ctx context.Context, tenantID, categoriaID, nombre, descripcion string) (bool, error)
	ImportarCartaAtomica(ctx context.Context, tenantID string, filas []FilaImportacion) (ResultadoImportacion, error)
	CrearArticulo(ctx context.Context, tenantID string, input ArticuloInput) (*Articulo, error)
	ActualizarArticulo(ctx context.Context, id, tenantID string, u ArticuloUpdate) (*Articulo, error)
	ActualizarDisponibilidad(ctx context.Context, id, tenantID string, disponible bool) (*Articulo, error)
	ReponerTodos(ctx context.Context, tenantID string) (int, error)
	AjustarPrecios(ctx context.Context, tenantID string, input AjustePreciosInput) (*AjustePreciosResultado, error)
	EliminarArticulo(ctx context.Context, id, tenantID string) error
	ObtenerCartaPublica(ctx context.Context, sucursalID string) (*CartaPublica, error)
	ListarFranjasHorarias(ctx context.Context, tenantID string) ([]FranjaHoraria, error)
	CrearFranjaHoraria(ctx context.Context, tenantID string, input FranjaHorariaInput) (*FranjaHoraria, error)
	ActualizarFranjaHoraria(ctx context.Context, id, tenantID string, input FranjaHorariaInput) (*FranjaHoraria, error)
	EliminarFranjaHoraria(ctx context.Context, id, tenantID string) error
	AsignarFranjaCategoria(ctx context.Context, id, tenantID string, franjaID *string) (*Categoria, error)
	AsignarIconoCategoria(ctx context.Context, id, tenantID string, icono *string) (*Categoria, error)
	AsignarFranjaArticulo(ctx context.Context, id, tenantID string, franjaID *string) (*Articulo, error)

	ListarVariantes(ctx context.Context, articuloID, tenantID string) ([]Variante, error)
	CrearVariante(ctx context.Context, articuloID, tenantID string, input CrearVarianteInput) (*Variante, error)
	ActualizarVariante(ctx context.Context, id, tenantID string, input ActualizarVarianteInput) (*Variante, error)
	EliminarVariante(ctx context.Context, id, tenantID string) error
}

type pgStore struct{}

func NuevoStore() Store { return &pgStore{} }

func (s *pgStore) ListarCategorias(ctx context.Context, tenantID string) ([]Categoria, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT id, tenant_id, nombre, orden, icono, franja_horaria_id
		 FROM categorias WHERE tenant_id = $1 ORDER BY orden`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cats []Categoria
	for rows.Next() {
		var c Categoria
		var icono sql.NullString
		var franjaID sql.NullString
		if err := rows.Scan(&c.ID, &c.TenantID, &c.Nombre, &c.Orden, &icono, &franjaID); err != nil {
			return nil, err
		}
		c.Icono = nullStringPtr(icono)
		c.FranjaHorariaID = nullStringPtr(franjaID)
		cats = append(cats, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return cats, nil
}

func (s *pgStore) CrearCategoria(ctx context.Context, tenantID string, input CategoriaInput) (*Categoria, error) {
	c := &Categoria{}
	var icono sql.NullString
	var franjaID sql.NullString
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO categorias (tenant_id, nombre, orden) VALUES ($1, $2, $3)
		 RETURNING id, tenant_id, nombre, orden, icono, franja_horaria_id`,
		tenantID, input.Nombre, input.Orden,
	).Scan(&c.ID, &c.TenantID, &c.Nombre, &c.Orden, &icono, &franjaID)
	c.Icono = nullStringPtr(icono)
	c.FranjaHorariaID = nullStringPtr(franjaID)
	return c, err
}

func (s *pgStore) EliminarCategoria(ctx context.Context, id, tenantID string) error {
	tag, err := db.Pool.Exec(ctx,
		`DELETE FROM categorias WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// BuscarOCrearCategoria devuelve el ID de la categoría con ese nombre para el tenant dado.
// Si no existe la crea. La resolución de "nombre ya existente" usa ILIKE para ser
// insensible a mayúsculas/minúsculas y así evitar duplicados visuales.
func (s *pgStore) BuscarOCrearCategoria(ctx context.Context, tenantID, nombre string) (string, error) {
	var id string

	// Intentar encontrar primero (case-insensitive)
	err := db.Pool.QueryRow(ctx,
		`SELECT id FROM categorias WHERE tenant_id = $1 AND nombre ILIKE $2 LIMIT 1`,
		tenantID, strings.TrimSpace(nombre),
	).Scan(&id)
	if err == nil {
		return id, nil
	}

	// Si no existe, crearla. El orden (orden) se calcula como MAX(orden)+1.
	err = db.Pool.QueryRow(ctx,
		`INSERT INTO categorias (tenant_id, nombre, orden)
		 VALUES ($1, $2, COALESCE((SELECT MAX(orden)+1 FROM categorias WHERE tenant_id = $1), 1))
		 ON CONFLICT DO NOTHING
		 RETURNING id`,
		tenantID, strings.TrimSpace(nombre),
	).Scan(&id)
	if err != nil {
		// Puede ocurrir si dos goroutines insertan la misma categoría en paralelo;
		// en ese caso, la segunda obtiene conflict y retornamos la existente.
		err2 := db.Pool.QueryRow(ctx,
			`SELECT id FROM categorias WHERE tenant_id = $1 AND nombre ILIKE $2 LIMIT 1`,
			tenantID, strings.TrimSpace(nombre),
		).Scan(&id)
		if err2 != nil {
			return "", err2
		}
	}
	return id, nil
}

func (s *pgStore) ListarArticulos(ctx context.Context, tenantID string) ([]Articulo, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT id, tenant_id, categoria_id, nombre, COALESCE(descripcion,''), precio, COALESCE(foto_url,''), activo, disponible, reponer_diariamente, franja_horaria_id
		 FROM articulos WHERE tenant_id = $1 ORDER BY nombre`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var arts []Articulo
	for rows.Next() {
		var a Articulo
		var franjaID sql.NullString
		if err := rows.Scan(&a.ID, &a.TenantID, &a.CategoriaID, &a.Nombre, &a.Descripcion, &a.Precio, &a.FotoURL, &a.Activo, &a.Disponible, &a.ReponerDiariamente, &franjaID); err != nil {
			return nil, err
		}
		a.FranjaHorariaID = nullStringPtr(franjaID)
		arts = append(arts, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	artPtrs := make([]*Articulo, len(arts))
	for i := range arts {
		artPtrs[i] = &arts[i]
	}
	if err := s.cargarVariantesPunteros(ctx, artPtrs); err != nil {
		return nil, err
	}
	return arts, nil
}

// ExisteArticuloImportado aplica la identidad usada por la carga masiva:
// misma categoría, mismo nombre y misma descripción, sin distinguir mayúsculas
// ni espacios exteriores. El precio no forma parte de la identidad para no
// reemplazar silenciosamente un artículo existente durante una importación.
func (s *pgStore) ExisteArticuloImportado(
	ctx context.Context,
	tenantID, categoriaID, nombre, descripcion string,
) (bool, error) {
	var existe bool
	err := db.Pool.QueryRow(ctx,
		`SELECT EXISTS (
			SELECT 1
			FROM articulos
			WHERE tenant_id = $1
			  AND categoria_id = $2
			  AND LOWER(BTRIM(nombre)) = LOWER(BTRIM($3))
			  AND LOWER(BTRIM(COALESCE(descripcion, ''))) = LOWER(BTRIM($4))
		)`,
		tenantID, categoriaID, nombre, descripcion,
	).Scan(&existe)
	return existe, err
}

// ImportarCartaAtomica procesa toda la carga dentro de una única transacción.
// El advisory lock serializa las importaciones de un mismo tenant y evita que
// dos archivos concurrentes superen juntos la comprobación de duplicados.
func (s *pgStore) ImportarCartaAtomica(
	ctx context.Context,
	tenantID string,
	filas []FilaImportacion,
) (ResultadoImportacion, error) {
	resultado := ResultadoImportacion{Errores: []ErrorFila{}}
	if len(filas) == 0 {
		return resultado, nil
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return resultado, err
	}
	defer tx.Rollback(ctx) // no-op después de Commit

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, "carta-import:"+tenantID); err != nil {
		return resultado, err
	}

	catCache := make(map[string]string)
	for _, fila := range filas {
		claveCategoria := strings.ToLower(strings.TrimSpace(fila.Categoria))
		catID, ok := catCache[claveCategoria]
		if !ok {
			err = tx.QueryRow(ctx,
				`SELECT id
				 FROM categorias
				 WHERE tenant_id = $1
				   AND LOWER(BTRIM(nombre)) = LOWER(BTRIM($2))
				 ORDER BY orden, id
				 LIMIT 1`,
				tenantID, fila.Categoria,
			).Scan(&catID)
			if errors.Is(err, pgx.ErrNoRows) {
				err = tx.QueryRow(ctx,
					`INSERT INTO categorias (tenant_id, nombre, orden)
					 VALUES ($1, BTRIM($2), COALESCE((SELECT MAX(orden) + 1 FROM categorias WHERE tenant_id = $1), 1))
					 RETURNING id`,
					tenantID, fila.Categoria,
				).Scan(&catID)
			}
			if err != nil {
				return resultado, err
			}
			catCache[claveCategoria] = catID
		}

		var articuloID string
		err = tx.QueryRow(ctx,
			`INSERT INTO articulos (
				tenant_id, categoria_id, nombre, descripcion, precio,
				foto_url, activo, disponible, reponer_diariamente
			)
			SELECT $1, $2, BTRIM($3), BTRIM($4), $5, '', true, $6, true
			WHERE NOT EXISTS (
				SELECT 1
				FROM articulos
				WHERE tenant_id = $1
				  AND categoria_id = $2
				  AND LOWER(BTRIM(nombre)) = LOWER(BTRIM($3))
				  AND LOWER(BTRIM(COALESCE(descripcion, ''))) = LOWER(BTRIM($4))
			)
			RETURNING id`,
			tenantID, catID, fila.Nombre, fila.Descripcion, fila.Precio, fila.Disponible,
		).Scan(&articuloID)
		if errors.Is(err, pgx.ErrNoRows) {
			resultado.Omitidos++
			continue
		}
		if err != nil {
			return resultado, err
		}
		resultado.Creados++
	}

	if err := tx.Commit(ctx); err != nil {
		return resultado, err
	}
	return resultado, nil
}

func (s *pgStore) CrearArticulo(ctx context.Context, tenantID string, input ArticuloInput) (*Articulo, error) {
	a := &Articulo{}
	var franjaID sql.NullString
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO articulos (tenant_id, categoria_id, nombre, descripcion, precio, foto_url, activo, disponible, reponer_diariamente)
		 VALUES ($1, $2, $3, $4, $5, $6, true, true, true)
		 RETURNING id, tenant_id, categoria_id, nombre, COALESCE(descripcion,''), precio, COALESCE(foto_url,''), activo, disponible, reponer_diariamente, franja_horaria_id`,
		tenantID, input.CategoriaID, input.Nombre, input.Descripcion, input.Precio, input.FotoURL,
	).Scan(&a.ID, &a.TenantID, &a.CategoriaID, &a.Nombre, &a.Descripcion, &a.Precio, &a.FotoURL, &a.Activo, &a.Disponible, &a.ReponerDiariamente, &franjaID)
	a.FranjaHorariaID = nullStringPtr(franjaID)
	return a, err
}

func (s *pgStore) ActualizarArticulo(ctx context.Context, id, tenantID string, u ArticuloUpdate) (*Articulo, error) {
	a := &Articulo{}
	var franjaID sql.NullString
	err := db.Pool.QueryRow(ctx,
		`UPDATE articulos SET
		   categoria_id        = COALESCE((SELECT id FROM categorias WHERE id = $3::uuid AND tenant_id = $2), categoria_id),
		   nombre              = COALESCE($4, nombre),
		   descripcion         = COALESCE($5, descripcion),
		   precio              = COALESCE($6, precio),
		   foto_url            = COALESCE($7, foto_url),
		   activo              = COALESCE($8, activo),
		   disponible          = COALESCE($9, disponible),
		   reponer_diariamente = COALESCE($10, reponer_diariamente)
		 WHERE id = $1 AND tenant_id = $2
		 RETURNING id, tenant_id, categoria_id, nombre, COALESCE(descripcion,''), precio, COALESCE(foto_url,''), activo, disponible, reponer_diariamente, franja_horaria_id`,
		id, tenantID, u.CategoriaID, u.Nombre, u.Descripcion, u.Precio, u.FotoURL, u.Activo, u.Disponible, u.ReponerDiariamente,
	).Scan(&a.ID, &a.TenantID, &a.CategoriaID, &a.Nombre, &a.Descripcion, &a.Precio, &a.FotoURL, &a.Activo, &a.Disponible, &a.ReponerDiariamente, &franjaID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	a.FranjaHorariaID = nullStringPtr(franjaID)
	return a, nil
}

func (s *pgStore) ActualizarDisponibilidad(ctx context.Context, id, tenantID string, disponible bool) (*Articulo, error) {
	a := &Articulo{}
	err := db.Pool.QueryRow(ctx,
		`UPDATE articulos SET disponible = $3
		 WHERE id = $1 AND tenant_id = $2
		 RETURNING id, tenant_id, categoria_id, nombre, COALESCE(descripcion,''), precio, COALESCE(foto_url,''), activo, disponible, reponer_diariamente`,
		id, tenantID, disponible,
	).Scan(&a.ID, &a.TenantID, &a.CategoriaID, &a.Nombre, &a.Descripcion, &a.Precio, &a.FotoURL, &a.Activo, &a.Disponible, &a.ReponerDiariamente)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return a, nil
}

func (s *pgStore) ReponerTodos(ctx context.Context, tenantID string) (int, error) {
	tag, err := db.Pool.Exec(ctx,
		`UPDATE articulos
		 SET disponible = true
		 WHERE tenant_id = $1 AND reponer_diariamente = true AND disponible = false`,
		tenantID,
	)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

func (s *pgStore) AjustarPrecios(ctx context.Context, tenantID string, input AjustePreciosInput) (*AjustePreciosResultado, error) {
	resultado := &AjustePreciosResultado{}
	err := db.Pool.QueryRow(ctx,
		`WITH precios_actualizados AS (
			UPDATE articulos
			SET precio = CASE $4
				WHEN '10' THEN ROUND((precio * (1 + $3::numeric / 100)) / 10) * 10
				WHEN '100' THEN ROUND((precio * (1 + $3::numeric / 100)) / 100) * 100
				ELSE ROUND(precio * (1 + $3::numeric / 100), 2)
			END
			WHERE tenant_id = $1
			  AND ($2 = '' OR categoria_id::text = $2)
			RETURNING id
		)
		SELECT COUNT(*) FROM precios_actualizados`,
		tenantID, input.CategoriaID, input.Porcentaje, input.Redondeo,
	).Scan(&resultado.Actualizados)
	if err != nil {
		return nil, err
	}
	return resultado, nil
}

func (s *pgStore) EliminarArticulo(ctx context.Context, id, tenantID string) error {
	tag, err := db.Pool.Exec(ctx,
		`DELETE FROM articulos WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *pgStore) ObtenerCartaPublica(ctx context.Context, sucursalID string) (*CartaPublica, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT c.id, c.nombre, c.orden,
		        CASE WHEN COALESCE(t.plan, 'free') = 'pro' AND (t.plan_hasta IS NULL OR t.plan_hasta > NOW()) THEN c.icono END,
		        c.franja_horaria_id,
		        a.id, a.categoria_id, a.nombre, COALESCE(a.descripcion,''), a.precio, COALESCE(a.foto_url,''), a.disponible, a.franja_horaria_id,
		        f.id, f.nombre, TO_CHAR(f.hora_inicio, 'HH24:MI'), TO_CHAR(f.hora_fin, 'HH24:MI')
		 FROM categorias c
		 JOIN articulos a ON a.categoria_id = c.id
		 JOIN sucursales su ON su.id = $1 AND su.tenant_id = c.tenant_id
		 JOIN tenants t ON t.id = c.tenant_id
		 LEFT JOIN franjas_horarias f ON f.id = COALESCE(a.franja_horaria_id, c.franja_horaria_id)
		 WHERE a.tenant_id = t.id
		   AND a.activo = true
		 ORDER BY c.orden, a.nombre`,
		sucursalID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	catMap := map[string]*CategoriaConArticulos{}
	var orden []string

	for rows.Next() {
		var (
			catID, catNombre string
			catOrden         int
			art              Articulo
			catIcono         sql.NullString
			catFranjaID      sql.NullString
			artFranjaID      sql.NullString
			franjaID         sql.NullString
			franjaNombre     sql.NullString
			horaInicio       sql.NullString
			horaFin          sql.NullString
		)
		if err := rows.Scan(&catID, &catNombre, &catOrden, &catIcono, &catFranjaID,
			&art.ID, &art.CategoriaID, &art.Nombre, &art.Descripcion, &art.Precio, &art.FotoURL, &art.Disponible, &artFranjaID,
			&franjaID, &franjaNombre, &horaInicio, &horaFin); err != nil {
			return nil, err
		}
		art.Activo = true
		art.FranjaHorariaID = nullStringPtr(artFranjaID)
		if franjaID.Valid {
			art.FranjaEfectiva = &FranjaHoraria{
				ID:         franjaID.String,
				Nombre:     franjaNombre.String,
				HoraInicio: horaInicio.String,
				HoraFin:    horaFin.String,
			}
		}
		if _, ok := catMap[catID]; !ok {
			catMap[catID] = &CategoriaConArticulos{
				Categoria: Categoria{ID: catID, Nombre: catNombre, Orden: catOrden, Icono: nullStringPtr(catIcono), FranjaHorariaID: nullStringPtr(catFranjaID)},
			}
			orden = append(orden, catID)
		}
		catMap[catID].Articulos = append(catMap[catID].Articulos, art)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	var allArts []*Articulo
	for _, cat := range catMap {
		for i := range cat.Articulos {
			allArts = append(allArts, &cat.Articulos[i])
		}
	}
	if err := s.cargarVariantesPunteros(ctx, allArts); err != nil {
		return nil, err
	}

	resultado := &CartaPublica{}
	for _, id := range orden {
		resultado.Categorias = append(resultado.Categorias, *catMap[id])
	}
	return resultado, nil
}

func nullStringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	result := value.String
	return &result
}

func (s *pgStore) ListarFranjasHorarias(ctx context.Context, tenantID string) ([]FranjaHoraria, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT id, tenant_id, nombre, TO_CHAR(hora_inicio, 'HH24:MI'), TO_CHAR(hora_fin, 'HH24:MI')
		 FROM franjas_horarias WHERE tenant_id = $1 ORDER BY hora_inicio, nombre`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	franjas := make([]FranjaHoraria, 0)
	for rows.Next() {
		var franja FranjaHoraria
		if err := rows.Scan(&franja.ID, &franja.TenantID, &franja.Nombre, &franja.HoraInicio, &franja.HoraFin); err != nil {
			return nil, err
		}
		franjas = append(franjas, franja)
	}
	return franjas, rows.Err()
}

func (s *pgStore) CrearFranjaHoraria(ctx context.Context, tenantID string, input FranjaHorariaInput) (*FranjaHoraria, error) {
	franja := &FranjaHoraria{}
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO franjas_horarias (tenant_id, nombre, hora_inicio, hora_fin)
		 VALUES ($1, $2, $3::time, $4::time)
		 RETURNING id, tenant_id, nombre, TO_CHAR(hora_inicio, 'HH24:MI'), TO_CHAR(hora_fin, 'HH24:MI')`,
		tenantID, input.Nombre, input.HoraInicio, input.HoraFin,
	).Scan(&franja.ID, &franja.TenantID, &franja.Nombre, &franja.HoraInicio, &franja.HoraFin)
	if esConstraintUnica(err, "franjas_horarias_nombre_tenant_unique") {
		return nil, ErrFranjaDuplicada
	}
	return franja, err
}

func (s *pgStore) ActualizarFranjaHoraria(ctx context.Context, id, tenantID string, input FranjaHorariaInput) (*FranjaHoraria, error) {
	franja := &FranjaHoraria{}
	err := db.Pool.QueryRow(ctx,
		`UPDATE franjas_horarias
		 SET nombre = $3, hora_inicio = $4::time, hora_fin = $5::time, updated_at = NOW()
		 WHERE id = $1 AND tenant_id = $2
		 RETURNING id, tenant_id, nombre, TO_CHAR(hora_inicio, 'HH24:MI'), TO_CHAR(hora_fin, 'HH24:MI')`,
		id, tenantID, input.Nombre, input.HoraInicio, input.HoraFin,
	).Scan(&franja.ID, &franja.TenantID, &franja.Nombre, &franja.HoraInicio, &franja.HoraFin)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if esConstraintUnica(err, "franjas_horarias_nombre_tenant_unique") {
		return nil, ErrFranjaDuplicada
	}
	return franja, err
}

func esConstraintUnica(err error, nombre string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == nombre
}

func (s *pgStore) EliminarFranjaHoraria(ctx context.Context, id, tenantID string) error {
	tag, err := db.Pool.Exec(ctx, `DELETE FROM franjas_horarias WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *pgStore) AsignarFranjaCategoria(ctx context.Context, id, tenantID string, franjaID *string) (*Categoria, error) {
	categoria := &Categoria{}
	var iconoAsignado sql.NullString
	var franjaAsignada sql.NullString
	err := db.Pool.QueryRow(ctx,
		`UPDATE categorias c
		 SET franja_horaria_id = $3::uuid
		 WHERE c.id = $1 AND c.tenant_id = $2
		   AND ($3::uuid IS NULL OR EXISTS (
		       SELECT 1 FROM franjas_horarias f WHERE f.id = $3::uuid AND f.tenant_id = $2
		   ))
		 RETURNING c.id, c.tenant_id, c.nombre, c.orden, c.icono, c.franja_horaria_id`,
		id, tenantID, franjaID,
	).Scan(&categoria.ID, &categoria.TenantID, &categoria.Nombre, &categoria.Orden, &iconoAsignado, &franjaAsignada)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	categoria.Icono = nullStringPtr(iconoAsignado)
	categoria.FranjaHorariaID = nullStringPtr(franjaAsignada)
	return categoria, err
}

func (s *pgStore) AsignarIconoCategoria(ctx context.Context, id, tenantID string, icono *string) (*Categoria, error) {
	categoria := &Categoria{}
	var iconoAsignado sql.NullString
	var franjaAsignada sql.NullString
	err := db.Pool.QueryRow(ctx,
		`UPDATE categorias
		 SET icono = $3
		 WHERE id = $1 AND tenant_id = $2
		 RETURNING id, tenant_id, nombre, orden, icono, franja_horaria_id`,
		id, tenantID, icono,
	).Scan(&categoria.ID, &categoria.TenantID, &categoria.Nombre, &categoria.Orden, &iconoAsignado, &franjaAsignada)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	categoria.Icono = nullStringPtr(iconoAsignado)
	categoria.FranjaHorariaID = nullStringPtr(franjaAsignada)
	return categoria, err
}

func (s *pgStore) AsignarFranjaArticulo(ctx context.Context, id, tenantID string, franjaID *string) (*Articulo, error) {
	articulo := &Articulo{}
	var franjaAsignada sql.NullString
	err := db.Pool.QueryRow(ctx,
		`UPDATE articulos a
		 SET franja_horaria_id = $3::uuid
		 WHERE a.id = $1 AND a.tenant_id = $2
		   AND ($3::uuid IS NULL OR EXISTS (
		       SELECT 1 FROM franjas_horarias f WHERE f.id = $3::uuid AND f.tenant_id = $2
		   ))
		 RETURNING a.id, a.tenant_id, a.categoria_id, a.nombre, COALESCE(a.descripcion,''),
		           a.precio, COALESCE(a.foto_url,''), a.activo, a.franja_horaria_id`,
		id, tenantID, franjaID,
	).Scan(&articulo.ID, &articulo.TenantID, &articulo.CategoriaID, &articulo.Nombre, &articulo.Descripcion,
		&articulo.Precio, &articulo.FotoURL, &articulo.Activo, &franjaAsignada)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	articulo.FranjaHorariaID = nullStringPtr(franjaAsignada)
	return articulo, err
}

func (s *pgStore) cargarVariantesPunteros(ctx context.Context, articulos []*Articulo) error {
	if len(articulos) == 0 {
		return nil
	}
	ids := make([]string, len(articulos))
	artMap := make(map[string]*Articulo, len(articulos))
	for i, a := range articulos {
		ids[i] = a.ID
		artMap[a.ID] = a
	}

	rows, err := db.Pool.Query(ctx,
		`SELECT id, articulo_id, nombre, precio_adicional, grupo, seleccion_unica, orden
		 FROM variantes
		 WHERE articulo_id = ANY($1)
		 ORDER BY grupo NULLS LAST, orden, nombre`,
		ids,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var v Variante
		if err := rows.Scan(&v.ID, &v.ArticuloID, &v.Nombre, &v.PrecioAdicional, &v.Grupo, &v.SeleccionUnica, &v.Orden); err != nil {
			return err
		}
		if a, ok := artMap[v.ArticuloID]; ok {
			a.Variantes = append(a.Variantes, v)
		}
	}
	return rows.Err()
}

func (s *pgStore) ListarVariantes(ctx context.Context, articuloID, tenantID string) ([]Variante, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT v.id, v.articulo_id, v.nombre, v.precio_adicional, v.grupo, v.seleccion_unica, v.orden
		 FROM variantes v
		 JOIN articulos a ON a.id = v.articulo_id
		 WHERE v.articulo_id = $1 AND a.tenant_id = $2
		 ORDER BY v.grupo NULLS LAST, v.orden, v.nombre`,
		articuloID, tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	variantes := make([]Variante, 0)
	for rows.Next() {
		var v Variante
		if err := rows.Scan(&v.ID, &v.ArticuloID, &v.Nombre, &v.PrecioAdicional, &v.Grupo, &v.SeleccionUnica, &v.Orden); err != nil {
			return nil, err
		}
		variantes = append(variantes, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return variantes, nil
}

func (s *pgStore) CrearVariante(ctx context.Context, articuloID, tenantID string, input CrearVarianteInput) (*Variante, error) {
	v := &Variante{}
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO variantes (articulo_id, nombre, precio_adicional, grupo, seleccion_unica, orden)
		 SELECT $1, $2, $3, $4, $5, $6
		 WHERE EXISTS (SELECT 1 FROM articulos WHERE id = $1 AND tenant_id = $7)
		 RETURNING id, articulo_id, nombre, precio_adicional, grupo, seleccion_unica, orden`,
		articuloID, input.Nombre, input.PrecioAdicional, input.Grupo, input.SeleccionUnica, input.Orden, tenantID,
	).Scan(&v.ID, &v.ArticuloID, &v.Nombre, &v.PrecioAdicional, &v.Grupo, &v.SeleccionUnica, &v.Orden)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return v, nil
}

func (s *pgStore) ActualizarVariante(ctx context.Context, id, tenantID string, input ActualizarVarianteInput) (*Variante, error) {
	v := &Variante{}
	err := db.Pool.QueryRow(ctx,
		`UPDATE variantes v
		 SET nombre = COALESCE($3, v.nombre),
		     precio_adicional = COALESCE($4, v.precio_adicional),
		     grupo = COALESCE($5, v.grupo),
		     seleccion_unica = COALESCE($6, v.seleccion_unica),
		     orden = COALESCE($7, v.orden)
		 FROM articulos a
		 WHERE v.id = $1 AND v.articulo_id = a.id AND a.tenant_id = $2
		 RETURNING v.id, v.articulo_id, v.nombre, v.precio_adicional, v.grupo, v.seleccion_unica, v.orden`,
		id, tenantID, input.Nombre, input.PrecioAdicional, input.Grupo, input.SeleccionUnica, input.Orden,
	).Scan(&v.ID, &v.ArticuloID, &v.Nombre, &v.PrecioAdicional, &v.Grupo, &v.SeleccionUnica, &v.Orden)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return v, nil
}

func (s *pgStore) EliminarVariante(ctx context.Context, id, tenantID string) error {
	tag, err := db.Pool.Exec(ctx,
		`DELETE FROM variantes v
		 USING articulos a
		 WHERE v.id = $1 AND v.articulo_id = a.id AND a.tenant_id = $2`,
		id, tenantID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
