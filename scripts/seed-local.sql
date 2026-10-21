BEGIN;

WITH tenant_upsert AS (
  INSERT INTO tenants (
    nombre,
    nombre_fantasia,
    rubro,
    slug,
    descripcion,
    email_contacto,
    whatsapp,
    logo_url,
    color_primario,
    estilo_visual,
    plan,
    plan_desde,
    datos_fiscales,
    google_review_url
  ) VALUES (
    'Mesa Click Demo SRL',
    'Mesa Click Demo',
    'restaurante',
    'mesa-click-demo',
    'Cuenta local de desarrollo para trabajar el dashboard sin depender de correo real.',
    'admin@mesaclick.local',
    '+5493510000000',
    NULL,
    '#155e75',
    'claro',
    'pro',
    NOW(),
    '{}'::jsonb,
    NULL
  )
  ON CONFLICT (slug) DO UPDATE SET
    nombre = EXCLUDED.nombre,
    nombre_fantasia = EXCLUDED.nombre_fantasia,
    rubro = EXCLUDED.rubro,
    descripcion = EXCLUDED.descripcion,
    email_contacto = EXCLUDED.email_contacto,
    whatsapp = EXCLUDED.whatsapp,
    color_primario = EXCLUDED.color_primario,
    estilo_visual = EXCLUDED.estilo_visual,
    plan = 'pro',
    plan_desde = COALESCE(tenants.plan_desde, NOW()),
    plan_hasta = NULL
  RETURNING id
), tenant_ref AS (
  SELECT id FROM tenant_upsert
  UNION ALL
  SELECT id FROM tenants WHERE slug = 'mesa-click-demo'
  LIMIT 1
), admin_upsert AS (
  INSERT INTO usuarios (tenant_id, email, nombre, rol, password_hash)
  SELECT id, 'admin@mesaclick.local', 'Admin Local', 'admin', crypt('MesaClick2026', gen_salt('bf', 10))
  FROM tenant_ref
  ON CONFLICT (tenant_id, lower(email))
    WHERE email IS NOT NULL AND deleted_at IS NULL
  DO UPDATE SET
    tenant_id = EXCLUDED.tenant_id,
    nombre = EXCLUDED.nombre,
    rol = EXCLUDED.rol,
    password_hash = EXCLUDED.password_hash
  RETURNING tenant_id
), sucursal_upsert AS (
  INSERT INTO sucursales (id, tenant_id, nombre, whatsapp, email, telefono, horarios)
  SELECT
    '10000000-0000-0000-0000-000000000001'::uuid,
    id,
    'Sucursal Centro',
    '+5493510000000',
    'admin@mesaclick.local',
    '3510000000',
    '{"lunes":"09:00-23:00","martes":"09:00-23:00","miercoles":"09:00-23:00","jueves":"09:00-23:00","viernes":"09:00-01:00","sabado":"10:00-01:00","domingo":"10:00-23:00"}'::jsonb
  FROM tenant_ref
  ON CONFLICT (id) DO UPDATE SET
    tenant_id = EXCLUDED.tenant_id,
    nombre = EXCLUDED.nombre,
    whatsapp = EXCLUDED.whatsapp,
    email = EXCLUDED.email,
    telefono = EXCLUDED.telefono,
    horarios = EXCLUDED.horarios
  RETURNING id, tenant_id
), sector_upsert AS (
  INSERT INTO sectores (id, sucursal_id, nombre)
  SELECT '20000000-0000-0000-0000-000000000001'::uuid, id, 'Salón principal'
  FROM sucursal_upsert
  ON CONFLICT (id) DO UPDATE SET
    sucursal_id = EXCLUDED.sucursal_id,
    nombre = EXCLUDED.nombre
  RETURNING id, sucursal_id
), mesas_seed AS (
  INSERT INTO mesas (sucursal_id, sector_id, numero, capacidad, qr_token, estado)
  SELECT s.sucursal_id, s.id, v.numero, v.capacidad, v.qr_token, 'activa'
  FROM sector_upsert s
  CROSS JOIN (VALUES
    (1, 4, 'mesa-demo-1'),
    (2, 2, 'mesa-demo-2'),
    (3, 6, 'mesa-demo-3'),
    (4, 4, 'mesa-demo-4')
  ) AS v(numero, capacidad, qr_token)
  ON CONFLICT (sucursal_id, numero) DO UPDATE SET
    sector_id = EXCLUDED.sector_id,
    capacidad = EXCLUDED.capacidad,
    qr_token = EXCLUDED.qr_token,
    estado = 'activa'
  RETURNING id
), categorias_seed AS (
  INSERT INTO categorias (id, tenant_id, nombre, orden, icono)
  SELECT v.id::uuid, t.id, v.nombre, v.orden, v.icono
  FROM tenant_ref t
  CROSS JOIN (VALUES
    ('30000000-0000-0000-0000-000000000001', 'Entradas', 10, 'tapas'),
    ('30000000-0000-0000-0000-000000000002', 'Principales', 20, 'lunch_dining'),
    ('30000000-0000-0000-0000-000000000003', 'Bebidas', 30, 'local_bar')
  ) AS v(id, nombre, orden, icono)
  ON CONFLICT (id) DO UPDATE SET
    tenant_id = EXCLUDED.tenant_id,
    nombre = EXCLUDED.nombre,
    orden = EXCLUDED.orden,
    icono = EXCLUDED.icono
  RETURNING id, tenant_id, nombre
)
INSERT INTO articulos (id, tenant_id, categoria_id, nombre, descripcion, precio, foto_url, activo)
SELECT v.id::uuid, c.tenant_id, c.id, v.nombre, v.descripcion, v.precio, NULL, TRUE
FROM categorias_seed c
JOIN (VALUES
  ('40000000-0000-0000-0000-000000000001', 'Entradas', 'Papas rústicas', 'Papas con alioli y verdeo.', 4200.00),
  ('40000000-0000-0000-0000-000000000002', 'Entradas', 'Empanadas criollas', 'Unidad horneada con carne cortada a cuchillo.', 1200.00),
  ('40000000-0000-0000-0000-000000000003', 'Principales', 'Burger Mesa Click', 'Medallón smash, cheddar, panceta y salsa de la casa.', 7800.00),
  ('40000000-0000-0000-0000-000000000004', 'Principales', 'Milanesa napolitana', 'Con papas fritas y ensalada simple.', 9200.00),
  ('40000000-0000-0000-0000-000000000005', 'Bebidas', 'Limonada', 'Jarra individual con menta y jengibre.', 2600.00),
  ('40000000-0000-0000-0000-000000000006', 'Bebidas', 'Agua sin gas', 'Botella 500 ml.', 1500.00)
) AS v(id, categoria, nombre, descripcion, precio) ON v.categoria = c.nombre
ON CONFLICT (id) DO UPDATE SET
  tenant_id = EXCLUDED.tenant_id,
  categoria_id = EXCLUDED.categoria_id,
  nombre = EXCLUDED.nombre,
  descripcion = EXCLUDED.descripcion,
  precio = EXCLUDED.precio,
  activo = EXCLUDED.activo;

-- Tablero local del recepcionista: cuatro mesas con situaciones distintas.
-- Los UUID fijos y los upserts mantienen el seed idempotente entre ejecuciones.
UPDATE mesas
SET
  cuenta_solicitada = (numero = 1),
  pago_habilitado = FALSE
WHERE sucursal_id = '10000000-0000-0000-0000-000000000001'::uuid
  AND numero BETWEEN 1 AND 4;

WITH pedidos_demo AS (
  SELECT *
  FROM (VALUES
    ('50000000-0000-0000-0000-000000000001', 1, 'listo',      24),
    ('50000000-0000-0000-0000-000000000002', 1, 'listo',      16),
    ('50000000-0000-0000-0000-000000000003', 2, 'preparando', 11),
    ('50000000-0000-0000-0000-000000000004', 2, 'preparando',  6),
    ('50000000-0000-0000-0000-000000000005', 3, 'listo',       4),
    ('50000000-0000-0000-0000-000000000006', 4, 'recibido',    2)
  ) AS v(id, mesa_numero, estado, minutos_atras)
), pedidos_resueltos AS (
  SELECT
    d.id::uuid AS id,
    m.id AS mesa_id,
    m.sucursal_id,
    m.cuenta_version,
    d.estado,
    NOW() - make_interval(mins => d.minutos_atras) AS creado
  FROM pedidos_demo d
  JOIN mesas m
    ON m.sucursal_id = '10000000-0000-0000-0000-000000000001'::uuid
   AND m.numero = d.mesa_numero
)
INSERT INTO pedidos (id, mesa_id, sucursal_id, cuenta_version, estado, created_at, updated_at)
SELECT id, mesa_id, sucursal_id, cuenta_version, estado, creado, creado
FROM pedidos_resueltos
ON CONFLICT (id) DO UPDATE SET
  mesa_id = EXCLUDED.mesa_id,
  sucursal_id = EXCLUDED.sucursal_id,
  cuenta_version = EXCLUDED.cuenta_version,
  estado = EXCLUDED.estado,
  created_at = EXCLUDED.created_at,
  updated_at = EXCLUDED.updated_at;

WITH items_demo AS (
  SELECT *
  FROM (VALUES
    ('60000000-0000-0000-0000-000000000001', '50000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000002', 2, 'Sin aceitunas',       '70000000-0000-0000-0000-000000000001', 'Mateo', 'listo'),
    ('60000000-0000-0000-0000-000000000002', '50000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000005', 1, NULL,                 '70000000-0000-0000-0000-000000000002', 'Sofi',   'listo'),
    ('60000000-0000-0000-0000-000000000003', '50000000-0000-0000-0000-000000000002', '40000000-0000-0000-0000-000000000004', 1, NULL,                 '70000000-0000-0000-0000-000000000001', 'Mateo', 'listo'),
    ('60000000-0000-0000-0000-000000000004', '50000000-0000-0000-0000-000000000003', '40000000-0000-0000-0000-000000000003', 2, 'Una sin panceta',      '70000000-0000-0000-0000-000000000003', 'Juli',   'preparando'),
    ('60000000-0000-0000-0000-000000000005', '50000000-0000-0000-0000-000000000003', '40000000-0000-0000-0000-000000000001', 1, NULL,                 '70000000-0000-0000-0000-000000000004', 'Nico',   'preparando'),
    ('60000000-0000-0000-0000-000000000006', '50000000-0000-0000-0000-000000000004', '40000000-0000-0000-0000-000000000006', 2, 'Bien frías',            '70000000-0000-0000-0000-000000000003', 'Juli',   'pendiente'),
    ('60000000-0000-0000-0000-000000000007', '50000000-0000-0000-0000-000000000005', '40000000-0000-0000-0000-000000000003', 1, NULL,                 '70000000-0000-0000-0000-000000000005', 'Luz',    'listo'),
    ('60000000-0000-0000-0000-000000000008', '50000000-0000-0000-0000-000000000005', '40000000-0000-0000-0000-000000000001', 1, NULL,                 '70000000-0000-0000-0000-000000000006', 'Tomás',  'listo'),
    ('60000000-0000-0000-0000-000000000009', '50000000-0000-0000-0000-000000000006', '40000000-0000-0000-0000-000000000002', 4, 'Dos suaves',            '70000000-0000-0000-0000-000000000007', 'Cami',   'pendiente'),
    ('60000000-0000-0000-0000-000000000010', '50000000-0000-0000-0000-000000000006', '40000000-0000-0000-0000-000000000005', 2, 'Sin jengibre',          '70000000-0000-0000-0000-000000000008', 'Fede',   'pendiente')
  ) AS v(id, pedido_id, articulo_id, cantidad, notas, comensal_id, comensal_nombre, estado)
)
INSERT INTO pedido_items (
  id,
  pedido_id,
  articulo_id,
  cantidad,
  precio_unitario,
  notas,
  comensal_id,
  comensal_nombre,
  estado
)
SELECT
  d.id::uuid,
  d.pedido_id::uuid,
  d.articulo_id::uuid,
  d.cantidad,
  a.precio,
  d.notas,
  d.comensal_id::uuid,
  d.comensal_nombre,
  d.estado
FROM items_demo d
JOIN articulos a ON a.id = d.articulo_id::uuid
ON CONFLICT (id) DO UPDATE SET
  pedido_id = EXCLUDED.pedido_id,
  articulo_id = EXCLUDED.articulo_id,
  cantidad = EXCLUDED.cantidad,
  precio_unitario = EXCLUDED.precio_unitario,
  notas = EXCLUDED.notas,
  comensal_id = EXCLUDED.comensal_id,
  comensal_nombre = EXCLUDED.comensal_nombre,
  estado = EXCLUDED.estado;

COMMIT;
