-- Amplía el flujo del pedido: recibido → confirmado → preparando → listo → entregado → cerrado.
-- Los pedidos existentes conservan su estado; los "recibido" anteriores siguen requiriendo confirmación.

ALTER TABLE pedidos DROP CONSTRAINT pedidos_estado_check;
ALTER TABLE pedidos ADD CONSTRAINT pedidos_estado_check
    CHECK (estado IN ('recibido', 'confirmado', 'preparando', 'listo', 'entregado', 'cerrado'));
