ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS trial_hasta TIMESTAMPTZ;

COMMENT ON COLUMN tenants.trial_hasta IS
    'Fin de la prueba Pro inicial. NULL para tenants sin prueba o planes Pro contratados directamente.';
