-- 034_add_ultimo_login_usuarios.sql
-- Agrega columna para auditar el último inicio de sesión de cada usuario del sistema.

ALTER TABLE usuarios
    ADD COLUMN IF NOT EXISTS ultimo_login_at TIMESTAMPTZ NULL;

COMMENT ON COLUMN usuarios.ultimo_login_at IS
    'Fecha y hora del último inicio de sesión exitoso del usuario (Google, contraseña o magic link).';
