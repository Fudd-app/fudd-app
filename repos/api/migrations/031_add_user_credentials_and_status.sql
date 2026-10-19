-- 031_add_user_credentials_and_status.sql
-- US-83 & US-84: Credenciales de equipo (username + password_hash), sucursal asignada,
-- ciclo de vida (activo, bloqueado, archivado) y baja lógica (deleted_at).

-- 1. Hacer email opcional para miembros operativos del staff (mozos, cocina, encargados)
ALTER TABLE usuarios
    ALTER COLUMN email DROP NOT NULL;

ALTER TABLE usuarios
    DROP CONSTRAINT IF EXISTS usuarios_email_key;

-- Normalizar cadenas vacías a NULL para evitar colisiones en índices únicos
UPDATE usuarios SET email = NULL WHERE email IS NOT NULL AND btrim(email) = '';

-- 2. Agregar columnas de credenciales, sucursal y ciclo de vida
ALTER TABLE usuarios
    ADD COLUMN IF NOT EXISTS username VARCHAR(50) NULL,
    ADD COLUMN IF NOT EXISTS sucursal_id UUID NULL REFERENCES sucursales(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS estado VARCHAR(20) NOT NULL DEFAULT 'activo',
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ NULL;

ALTER TABLE usuarios
    DROP CONSTRAINT IF EXISTS usuarios_estado_check;

ALTER TABLE usuarios
    ADD CONSTRAINT usuarios_estado_check
    CHECK (estado IN ('activo', 'bloqueado', 'archivado'));

-- 3. Índices únicos parciales (respetando soft delete y aislamiento por tenant)
DROP INDEX IF EXISTS idx_usuarios_email_lower;
CREATE UNIQUE INDEX idx_usuarios_email_lower
    ON usuarios (lower(email))
    WHERE email IS NOT NULL AND deleted_at IS NULL;

DROP INDEX IF EXISTS idx_usuarios_tenant_username;
CREATE UNIQUE INDEX idx_usuarios_tenant_username
    ON usuarios (tenant_id, lower(username))
    WHERE username IS NOT NULL AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_usuarios_sucursal ON usuarios (sucursal_id);
