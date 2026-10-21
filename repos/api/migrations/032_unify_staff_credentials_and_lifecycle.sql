-- 032_unify_staff_credentials_and_lifecycle.sql
-- US-83 & US-84: Unificación de acceso del equipo por invitación Google y por
-- credenciales operativas (username + password_hash), con soporte de estado
-- 'archivado' y baja lógica (deleted_at).

-- 1. Hacer email opcional para miembros operativos creados con usuario y clave
ALTER TABLE usuarios
    ALTER COLUMN email DROP NOT NULL;

UPDATE usuarios SET email = NULL WHERE email IS NOT NULL AND btrim(email) = '';

-- 2. Columnas de credenciales operativas y baja lógica
ALTER TABLE usuarios
    ADD COLUMN IF NOT EXISTS username VARCHAR(50) NULL,
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ NULL;

-- 3. Permitir estado 'archivado' además de 'activo' y 'bloqueado'
ALTER TABLE usuarios
    DROP CONSTRAINT IF EXISTS usuarios_estado_check;

ALTER TABLE usuarios
    ADD CONSTRAINT usuarios_estado_check
    CHECK (estado IN ('activo', 'bloqueado', 'archivado'));

-- 4. Índices únicos parciales compatibles con email opcional y soft delete
DROP INDEX IF EXISTS idx_usuarios_email_lower;
DROP INDEX IF EXISTS idx_usuarios_tenant_email;
CREATE UNIQUE INDEX IF NOT EXISTS idx_usuarios_tenant_email
    ON usuarios (tenant_id, lower(email))
    WHERE email IS NOT NULL AND deleted_at IS NULL;

DROP INDEX IF EXISTS idx_usuarios_tenant_google_sub;
CREATE UNIQUE INDEX IF NOT EXISTS idx_usuarios_tenant_google_sub
    ON usuarios (tenant_id, google_sub)
    WHERE google_sub IS NOT NULL AND deleted_at IS NULL;

DROP INDEX IF EXISTS idx_usuarios_admin_email;
CREATE UNIQUE INDEX IF NOT EXISTS idx_usuarios_admin_email
    ON usuarios (lower(email))
    WHERE rol = 'admin' AND email IS NOT NULL AND deleted_at IS NULL;

DROP INDEX IF EXISTS idx_usuarios_admin_google_sub;
CREATE UNIQUE INDEX IF NOT EXISTS idx_usuarios_admin_google_sub
    ON usuarios (google_sub)
    WHERE rol = 'admin' AND google_sub IS NOT NULL AND deleted_at IS NULL;

DROP INDEX IF EXISTS idx_usuarios_tenant_username;
CREATE UNIQUE INDEX IF NOT EXISTS idx_usuarios_tenant_username
    ON usuarios (tenant_id, lower(username))
    WHERE username IS NOT NULL AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_usuarios_sucursal ON usuarios (sucursal_id);
