-- US-83/84/85: acceso del equipo por invitación Google, sesiones revocables
-- y trazabilidad de pedidos cargados por mozos.

ALTER TABLE usuarios
    ADD COLUMN IF NOT EXISTS sucursal_id UUID REFERENCES sucursales(id) ON DELETE RESTRICT,
    ADD COLUMN IF NOT EXISTS estado TEXT NOT NULL DEFAULT 'activo',
    ADD COLUMN IF NOT EXISTS session_version BIGINT NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

ALTER TABLE usuarios DROP CONSTRAINT IF EXISTS usuarios_rol_check;
ALTER TABLE usuarios
    ADD CONSTRAINT usuarios_rol_check CHECK (rol IN ('admin', 'encargado', 'mozo', 'cocina'));

ALTER TABLE usuarios DROP CONSTRAINT IF EXISTS usuarios_estado_check;
ALTER TABLE usuarios
    ADD CONSTRAINT usuarios_estado_check CHECK (estado IN ('activo', 'bloqueado'));

-- Un dueño no queda fijado a una sucursal; todo el staff sí.
ALTER TABLE usuarios DROP CONSTRAINT IF EXISTS usuarios_sucursal_por_rol_check;
ALTER TABLE usuarios
    ADD CONSTRAINT usuarios_sucursal_por_rol_check CHECK (
        (rol = 'admin' AND sucursal_id IS NULL)
        OR (rol IN ('encargado', 'mozo', 'cocina') AND sucursal_id IS NOT NULL)
    ) NOT VALID;

-- Compatibilidad con miembros creados por el flujo anterior: se asignan a la
-- primera sucursal del tenant antes de validar la nueva invariantes.
UPDATE usuarios u
SET sucursal_id = (
    SELECT s.id FROM sucursales s
    WHERE s.tenant_id = u.tenant_id
    ORDER BY s.created_at, s.id
    LIMIT 1
)
WHERE u.rol <> 'admin' AND u.sucursal_id IS NULL;
ALTER TABLE usuarios VALIDATE CONSTRAINT usuarios_sucursal_por_rol_check;

-- El mismo email/Google puede participar en negocios distintos. La identidad
-- sigue siendo única dentro de cada tenant.
ALTER TABLE usuarios DROP CONSTRAINT IF EXISTS usuarios_email_key;
DROP INDEX IF EXISTS idx_usuarios_google_sub;
CREATE UNIQUE INDEX IF NOT EXISTS idx_usuarios_tenant_email
    ON usuarios (tenant_id, lower(email));
CREATE UNIQUE INDEX IF NOT EXISTS idx_usuarios_tenant_google_sub
    ON usuarios (tenant_id, google_sub)
    WHERE google_sub IS NOT NULL;
-- El login global sigue reservado al dueño; el staff usa /negocios/{slug}.
CREATE UNIQUE INDEX IF NOT EXISTS idx_usuarios_admin_email
    ON usuarios (lower(email)) WHERE rol = 'admin';
CREATE UNIQUE INDEX IF NOT EXISTS idx_usuarios_admin_google_sub
    ON usuarios (google_sub) WHERE rol = 'admin' AND google_sub IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_usuarios_sucursal_activos
    ON usuarios (sucursal_id, rol)
    WHERE estado = 'activo';

CREATE TABLE invitaciones_equipo (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    sucursal_id          UUID NOT NULL REFERENCES sucursales(id) ON DELETE CASCADE,
    email                TEXT NOT NULL,
    rol                  TEXT NOT NULL CHECK (rol IN ('encargado', 'mozo', 'cocina')),
    token_hash           TEXT NOT NULL UNIQUE,
    estado               TEXT NOT NULL DEFAULT 'pendiente'
                         CHECK (estado IN ('pendiente', 'cancelada', 'usada', 'reemplazada')),
    invitado_por_id      UUID REFERENCES usuarios(id) ON DELETE SET NULL,
    invitado_por_nombre  TEXT NOT NULL,
    expires_at           TIMESTAMPTZ NOT NULL,
    used_at              TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_invitaciones_equipo_pendiente_tenant_email
    ON invitaciones_equipo (tenant_id, lower(email))
    WHERE estado = 'pendiente';
CREATE INDEX idx_invitaciones_equipo_tenant_sucursal
    ON invitaciones_equipo (tenant_id, sucursal_id, created_at DESC);

ALTER TABLE pedidos
    ADD COLUMN IF NOT EXISTS origen TEXT NOT NULL DEFAULT 'comensal',
    ADD COLUMN IF NOT EXISTS creado_por_usuario_id UUID REFERENCES usuarios(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS creado_por_nombre TEXT;

ALTER TABLE pedidos DROP CONSTRAINT IF EXISTS pedidos_origen_check;
ALTER TABLE pedidos
    ADD CONSTRAINT pedidos_origen_check CHECK (origen IN ('comensal', 'mozo'));

CREATE INDEX IF NOT EXISTS idx_pedidos_creado_por_usuario
    ON pedidos (creado_por_usuario_id)
    WHERE creado_por_usuario_id IS NOT NULL;
