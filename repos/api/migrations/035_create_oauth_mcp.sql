-- 035_create_oauth_mcp.sql
-- Servidor de autorización OAuth 2.1 para clientes MCP (issue #55).

CREATE TABLE IF NOT EXISTS oauth_clientes (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id     TEXT NOT NULL UNIQUE,
    client_name   TEXT NOT NULL,
    redirect_uris TEXT[] NOT NULL,
    created_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS oauth_solicitudes (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id             TEXT NOT NULL REFERENCES oauth_clientes(client_id) ON DELETE CASCADE,
    redirect_uri          TEXT NOT NULL,
    code_challenge        TEXT NOT NULL,
    state                 TEXT NOT NULL DEFAULT '',
    scope                 TEXT NOT NULL,
    resource              TEXT NOT NULL,
    estado                TEXT NOT NULL DEFAULT 'pendiente'
                          CHECK (estado IN ('pendiente', 'aprobada', 'rechazada', 'canjeada')),
    usuario_id            UUID REFERENCES usuarios(id) ON DELETE CASCADE,
    codigo_hash           TEXT UNIQUE,
    codigo_expira_at      TIMESTAMP WITH TIME ZONE,
    expira_at             TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at            TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS oauth_conexiones (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id        TEXT NOT NULL REFERENCES oauth_clientes(client_id) ON DELETE CASCADE,
    usuario_id       UUID NOT NULL REFERENCES usuarios(id) ON DELETE CASCADE,
    tenant_id        UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    scope            TEXT NOT NULL,
    session_version  BIGINT NOT NULL,
    revocada_at      TIMESTAMP WITH TIME ZONE,
    ultimo_uso_at    TIMESTAMP WITH TIME ZONE,
    created_at       TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_oauth_conexiones_usuario ON oauth_conexiones(usuario_id) WHERE revocada_at IS NULL;

CREATE TABLE IF NOT EXISTS oauth_refresh_tokens (
    token_hash   TEXT PRIMARY KEY,
    conexion_id  UUID NOT NULL REFERENCES oauth_conexiones(id) ON DELETE CASCADE,
    usado_at     TIMESTAMP WITH TIME ZONE,
    expira_at    TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_oauth_refresh_conexion ON oauth_refresh_tokens(conexion_id);
