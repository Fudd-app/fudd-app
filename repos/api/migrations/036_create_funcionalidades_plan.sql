-- Acceso a funcionalidades por plan, con excepción por negocio (issue #59).

CREATE TABLE IF NOT EXISTS plan_funcionalidades (
    plan        TEXT NOT NULL,
    recurso     TEXT NOT NULL,
    habilitado  BOOLEAN NOT NULL,
    PRIMARY KEY (plan, recurso)
);

INSERT INTO plan_funcionalidades (plan, recurso, habilitado) VALUES
    ('pro', 'mcp', true),
    ('free', 'mcp', false)
ON CONFLICT (plan, recurso) DO NOTHING;

CREATE TABLE IF NOT EXISTS tenant_funcionalidades (
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    recurso     TEXT NOT NULL,
    habilitado  BOOLEAN NOT NULL,
    motivo      TEXT NOT NULL DEFAULT '',
    vence_at    TIMESTAMP WITH TIME ZONE,
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, recurso)
);
