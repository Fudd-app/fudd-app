-- Registro append-only de hitos operativos de mesas y pedidos (quién, cuándo, qué cambió).

CREATE TABLE IF NOT EXISTS eventos_operacion (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sucursal_id UUID NOT NULL REFERENCES sucursales(id) ON DELETE CASCADE,
    mesa_id UUID NOT NULL REFERENCES mesas(id) ON DELETE CASCADE,
    cuenta_version INTEGER NOT NULL,
    pedido_id UUID REFERENCES pedidos(id) ON DELETE CASCADE,
    pedido_item_id UUID REFERENCES pedido_items(id) ON DELETE CASCADE,
    tipo TEXT NOT NULL,
    estado_anterior TEXT,
    estado_nuevo TEXT,
    actor_tipo TEXT NOT NULL CHECK (actor_tipo IN ('usuario', 'comensal', 'sistema', 'mercadopago')),
    usuario_id UUID, -- sin FK: el historial sobrevive al borrado del usuario
    actor_nombre TEXT,
    actor_rol TEXT,
    detalle JSONB NOT NULL DEFAULT '{}'::jsonb,
    ts TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_eventos_operacion_sucursal_ts ON eventos_operacion (sucursal_id, ts);
CREATE INDEX IF NOT EXISTS idx_eventos_operacion_mesa_ts ON eventos_operacion (mesa_id, ts);
CREATE INDEX IF NOT EXISTS idx_eventos_operacion_pedido_ts ON eventos_operacion (pedido_id, ts) WHERE pedido_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_eventos_operacion_item ON eventos_operacion (pedido_item_id) WHERE pedido_item_id IS NOT NULL;

-- Solo inserción: el historial no se corrige. Los borrados llegan únicamente por cascada.
CREATE OR REPLACE FUNCTION eventos_operacion_sin_update() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'eventos_operacion es de solo inserción';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_eventos_operacion_sin_update ON eventos_operacion;
CREATE TRIGGER trg_eventos_operacion_sin_update
    BEFORE UPDATE ON eventos_operacion
    FOR EACH ROW EXECUTE FUNCTION eventos_operacion_sin_update();
