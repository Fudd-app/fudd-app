package oauth

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/db"
)

type Store interface {
	CrearCliente(ctx context.Context, c *Cliente) error
	ObtenerCliente(ctx context.Context, clientID string) (*Cliente, error)
	CrearSolicitud(ctx context.Context, s *Solicitud) error
	ObtenerSolicitud(ctx context.Context, id string) (*Solicitud, error)
	AprobarSolicitud(ctx context.Context, id, usuarioID, codigoHash string, codigoExpira time.Time) error
	RechazarSolicitud(ctx context.Context, id string) error
	CanjearCodigo(ctx context.Context, codigoHash string) (*Solicitud, error)
	CrearConexion(ctx context.Context, c *Conexion, refresh RefreshToken) error
	ObtenerConexion(ctx context.Context, id string) (*Conexion, error)
	RotarRefresh(ctx context.Context, viejoHash string, nuevo RefreshToken) (*Conexion, error)
	RevocarPorRefresh(ctx context.Context, tokenHash string) error
	ListarConexiones(ctx context.Context, usuarioID string) ([]ConexionVista, error)
	RevocarConexion(ctx context.Context, id, usuarioID string) error
	MarcarUso(ctx context.Context, conexionID string) error
}

type pgStore struct{}

func NuevoStore() Store { return &pgStore{} }

var reUUID = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

const columnasSolicitud = `id, client_id, redirect_uri, code_challenge, state, scope, resource, estado,
	usuario_id, codigo_hash, codigo_expira_at, expira_at, created_at, navegador_hash`

func escanearSolicitud(row pgx.Row) (*Solicitud, error) {
	s := &Solicitud{}
	err := row.Scan(&s.ID, &s.ClientID, &s.RedirectURI, &s.CodeChallenge, &s.State, &s.Scope, &s.Resource,
		&s.Estado, &s.UsuarioID, &s.CodigoHash, &s.CodigoExpiraAt, &s.ExpiraAt, &s.CreatedAt, &s.NavegadorHash)
	return s, err
}

func (s *pgStore) CrearCliente(ctx context.Context, c *Cliente) error {
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO oauth_clientes (client_id, client_name, redirect_uris)
		 VALUES ($1, $2, $3) RETURNING id, created_at`,
		c.ClientID, c.ClientName, c.RedirectURIs,
	).Scan(&c.ID, &c.CreatedAt)
	if err != nil {
		return fmt.Errorf("creando cliente oauth: %w", err)
	}
	return nil
}

func (s *pgStore) ObtenerCliente(ctx context.Context, clientID string) (*Cliente, error) {
	c := &Cliente{}
	err := db.Pool.QueryRow(ctx,
		`SELECT id, client_id, client_name, redirect_uris, created_at
		   FROM oauth_clientes WHERE client_id = $1`, clientID,
	).Scan(&c.ID, &c.ClientID, &c.ClientName, &c.RedirectURIs, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrClienteInvalido
	}
	if err != nil {
		return nil, fmt.Errorf("obteniendo cliente oauth: %w", err)
	}
	return c, nil
}

func (s *pgStore) CrearSolicitud(ctx context.Context, sol *Solicitud) error {
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO oauth_solicitudes (client_id, redirect_uri, code_challenge, state, scope, resource, estado, expira_at, navegador_hash)
		 VALUES ($1, $2, $3, $4, $5, $6, 'pendiente', $7, $8) RETURNING id, created_at`,
		sol.ClientID, sol.RedirectURI, sol.CodeChallenge, sol.State, sol.Scope, sol.Resource, sol.ExpiraAt, sol.NavegadorHash,
	).Scan(&sol.ID, &sol.CreatedAt)
	if err != nil {
		return fmt.Errorf("creando solicitud oauth: %w", err)
	}
	sol.Estado = "pendiente"
	return nil
}

func (s *pgStore) ObtenerSolicitud(ctx context.Context, id string) (*Solicitud, error) {
	if !reUUID.MatchString(id) {
		return nil, ErrSolicitudNoEncontrada
	}
	sol, err := escanearSolicitud(db.Pool.QueryRow(ctx,
		`SELECT `+columnasSolicitud+` FROM oauth_solicitudes WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSolicitudNoEncontrada
	}
	if err != nil {
		return nil, fmt.Errorf("obteniendo solicitud oauth: %w", err)
	}
	return sol, nil
}

func (s *pgStore) AprobarSolicitud(ctx context.Context, id, usuarioID, codigoHash string, codigoExpira time.Time) error {
	if !reUUID.MatchString(id) {
		return ErrSolicitudNoEncontrada
	}
	tag, err := db.Pool.Exec(ctx,
		`UPDATE oauth_solicitudes
		    SET estado = 'aprobada', usuario_id = $2, codigo_hash = $3, codigo_expira_at = $4
		  WHERE id = $1 AND estado = 'pendiente' AND expira_at > now()`,
		id, usuarioID, codigoHash, codigoExpira)
	if err != nil {
		return fmt.Errorf("aprobando solicitud oauth: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrSolicitudNoEncontrada
	}
	return nil
}

func (s *pgStore) RechazarSolicitud(ctx context.Context, id string) error {
	if !reUUID.MatchString(id) {
		return ErrSolicitudNoEncontrada
	}
	tag, err := db.Pool.Exec(ctx,
		`UPDATE oauth_solicitudes SET estado = 'rechazada'
		  WHERE id = $1 AND estado = 'pendiente' AND expira_at > now()`, id)
	if err != nil {
		return fmt.Errorf("rechazando solicitud oauth: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrSolicitudNoEncontrada
	}
	return nil
}

func (s *pgStore) CanjearCodigo(ctx context.Context, codigoHash string) (*Solicitud, error) {
	sol, err := escanearSolicitud(db.Pool.QueryRow(ctx,
		`UPDATE oauth_solicitudes SET estado = 'canjeada'
		  WHERE codigo_hash = $1 AND estado = 'aprobada' AND codigo_expira_at > now()
		  RETURNING `+columnasSolicitud, codigoHash))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrGrantInvalido
	}
	if err != nil {
		return nil, fmt.Errorf("canjeando código oauth: %w", err)
	}
	return sol, nil
}

func (s *pgStore) CrearConexion(ctx context.Context, c *Conexion, refresh RefreshToken) error {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("iniciando transacción: %w", err)
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx,
		`INSERT INTO oauth_conexiones (client_id, usuario_id, tenant_id, scope, session_version)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at`,
		c.ClientID, c.UsuarioID, c.TenantID, c.Scope, c.SessionVersion,
	).Scan(&c.ID, &c.CreatedAt)
	if err != nil {
		return fmt.Errorf("creando conexión oauth: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO oauth_refresh_tokens (token_hash, conexion_id, expira_at) VALUES ($1, $2, $3)`,
		refresh.TokenHash, c.ID, refresh.ExpiraAt); err != nil {
		return fmt.Errorf("creando refresh token: %w", err)
	}
	return tx.Commit(ctx)
}

const seleccionConexion = `SELECT c.id, c.client_id, cl.client_name, c.usuario_id, c.tenant_id, c.scope,
	c.session_version, c.revocada_at, c.ultimo_uso_at, c.created_at
	FROM oauth_conexiones c JOIN oauth_clientes cl ON cl.client_id = c.client_id`

func escanearConexion(row pgx.Row) (*Conexion, error) {
	c := &Conexion{}
	err := row.Scan(&c.ID, &c.ClientID, &c.ClientName, &c.UsuarioID, &c.TenantID, &c.Scope,
		&c.SessionVersion, &c.RevocadaAt, &c.UltimoUsoAt, &c.CreatedAt)
	return c, err
}

func (s *pgStore) ObtenerConexion(ctx context.Context, id string) (*Conexion, error) {
	if !reUUID.MatchString(id) {
		return nil, ErrConexionNoEncontrada
	}
	c, err := escanearConexion(db.Pool.QueryRow(ctx, seleccionConexion+` WHERE c.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrConexionNoEncontrada
	}
	if err != nil {
		return nil, fmt.Errorf("obteniendo conexión oauth: %w", err)
	}
	return c, nil
}

func (s *pgStore) RotarRefresh(ctx context.Context, viejoHash string, nuevo RefreshToken) (*Conexion, error) {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("iniciando transacción: %w", err)
	}
	defer tx.Rollback(ctx)

	var conexionID string
	var usadoAt *time.Time
	var expiraAt time.Time
	err = tx.QueryRow(ctx,
		`SELECT conexion_id, usado_at, expira_at FROM oauth_refresh_tokens WHERE token_hash = $1 FOR UPDATE`,
		viejoHash).Scan(&conexionID, &usadoAt, &expiraAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrGrantInvalido
	}
	if err != nil {
		return nil, fmt.Errorf("leyendo refresh token: %w", err)
	}

	if usadoAt != nil {
		// Reuso de un refresh ya rotado: se asume robo y se revoca toda la conexión.
		if _, err := tx.Exec(ctx,
			`UPDATE oauth_conexiones SET revocada_at = now() WHERE id = $1 AND revocada_at IS NULL`,
			conexionID); err != nil {
			return nil, fmt.Errorf("revocando conexión por reuso: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("confirmando revocación: %w", err)
		}
		return nil, ErrGrantInvalido
	}
	if !expiraAt.After(time.Now()) {
		return nil, ErrGrantInvalido
	}

	conexion, err := escanearConexion(tx.QueryRow(ctx, seleccionConexion+` WHERE c.id = $1`, conexionID))
	if err != nil {
		return nil, fmt.Errorf("leyendo conexión: %w", err)
	}
	if conexion.RevocadaAt != nil {
		return nil, ErrGrantInvalido
	}

	if _, err := tx.Exec(ctx, `UPDATE oauth_refresh_tokens SET usado_at = now() WHERE token_hash = $1`, viejoHash); err != nil {
		return nil, fmt.Errorf("marcando refresh usado: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO oauth_refresh_tokens (token_hash, conexion_id, expira_at) VALUES ($1, $2, $3)`,
		nuevo.TokenHash, conexionID, nuevo.ExpiraAt); err != nil {
		return nil, fmt.Errorf("creando refresh token nuevo: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE oauth_conexiones SET ultimo_uso_at = now() WHERE id = $1`, conexionID); err != nil {
		return nil, fmt.Errorf("actualizando uso de conexión: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("confirmando rotación: %w", err)
	}
	return conexion, nil
}

func (s *pgStore) RevocarPorRefresh(ctx context.Context, tokenHash string) error {
	_, err := db.Pool.Exec(ctx,
		`UPDATE oauth_conexiones SET revocada_at = now()
		  WHERE revocada_at IS NULL
		    AND id = (SELECT conexion_id FROM oauth_refresh_tokens WHERE token_hash = $1)`, tokenHash)
	if err != nil {
		return fmt.Errorf("revocando conexión por refresh: %w", err)
	}
	return nil
}

func (s *pgStore) ListarConexiones(ctx context.Context, usuarioID string) ([]ConexionVista, error) {
	if !reUUID.MatchString(usuarioID) {
		return []ConexionVista{}, nil
	}
	rows, err := db.Pool.Query(ctx,
		`SELECT c.id, cl.client_name, c.created_at, c.ultimo_uso_at
		   FROM oauth_conexiones c JOIN oauth_clientes cl ON cl.client_id = c.client_id
		  WHERE c.usuario_id = $1 AND c.revocada_at IS NULL
		  ORDER BY c.created_at DESC`, usuarioID)
	if err != nil {
		return nil, fmt.Errorf("listando conexiones oauth: %w", err)
	}
	defer rows.Close()

	lista := []ConexionVista{}
	for rows.Next() {
		var v ConexionVista
		if err := rows.Scan(&v.ID, &v.ClientName, &v.CreadoAt, &v.UltimoUsoAt); err != nil {
			return nil, fmt.Errorf("leyendo conexión oauth: %w", err)
		}
		lista = append(lista, v)
	}
	return lista, rows.Err()
}

func (s *pgStore) RevocarConexion(ctx context.Context, id, usuarioID string) error {
	if !reUUID.MatchString(id) || !reUUID.MatchString(usuarioID) {
		return ErrConexionNoEncontrada
	}
	tag, err := db.Pool.Exec(ctx,
		`UPDATE oauth_conexiones SET revocada_at = now()
		  WHERE id = $1 AND usuario_id = $2 AND revocada_at IS NULL`, id, usuarioID)
	if err != nil {
		return fmt.Errorf("revocando conexión oauth: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrConexionNoEncontrada
	}
	return nil
}

func (s *pgStore) MarcarUso(ctx context.Context, conexionID string) error {
	if !reUUID.MatchString(conexionID) {
		return nil
	}
	_, err := db.Pool.Exec(ctx,
		`UPDATE oauth_conexiones SET ultimo_uso_at = now()
		  WHERE id = $1 AND (ultimo_uso_at IS NULL OR ultimo_uso_at < now() - interval '1 minute')`, conexionID)
	if err != nil {
		return fmt.Errorf("marcando uso de conexión: %w", err)
	}
	return nil
}
