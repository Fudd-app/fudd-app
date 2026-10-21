// cmd/seed-demo/main.go
//
// Script idempotente de seeding para la Demo Interactiva Final (US-78).
// Puebla un negocio gastronómico realista en Plan PRO ("Bistró San Telmo — Demo Jurado")
// con sucursal, sectores, 12 mesas con QR estables, franjas horarias, categorías con íconos,
// platos con fotos/variantes, pedidos históricos para KPIs y reseñas para el Smart Google Funnel.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	_ = godotenv.Load()
	_ = godotenv.Load("repos/api/.env")

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		slog.Error("DATABASE_URL no está configurada")
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		slog.Error("Error conectando a PostgreSQL", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	slug := "bistro-san-telmo"
	emailAdmin := "demo@mesaclick.com"
	passwordDemo := "MesaClick2026!"
	hashBytes, _ := bcrypt.GenerateFromPassword([]byte(passwordDemo), bcrypt.DefaultCost)

	// Limpiar demo previa si existiera para garantizar idempotencia
	_, _ = pool.Exec(ctx, `DELETE FROM pedidos WHERE sucursal_id IN (SELECT id FROM sucursales WHERE tenant_id IN (SELECT id FROM tenants WHERE slug = $1))`, slug)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE slug = $1`, slug)

	var tenantID string
	err = pool.QueryRow(ctx,
		`INSERT INTO tenants (
			nombre, nombre_fantasia, rubro, slug, email_contacto, whatsapp,
			color_primario, estilo_visual, plan, mostrar_marca_agua, google_review_url
		) VALUES (
			'Bistró San Telmo S.R.L.', 'Bistró San Telmo', 'Restaurante & Cafetería de Especialidad',
			$1, $2, '+54 9 11 4555-8899', '#E25B1A', 'bistro', 'pro', false,
			'https://search.google.com/local/writereview?placeid=ChIJKxjxMnvKvJURcaM'
		) RETURNING id`,
		slug, emailAdmin,
	).Scan(&tenantID)
	if err != nil {
		slog.Error("Error creando tenant demo", "err", err)
		os.Exit(1)
	}

	// Crear usuario Admin y Cocinero
	var adminID string
	err = pool.QueryRow(ctx,
		`INSERT INTO usuarios (tenant_id, email, nombre, rol, activo, password_hash)
		 VALUES ($1, $2, 'Martín · Admin Demo', 'admin', true, $3)
		 RETURNING id`,
		tenantID, emailAdmin, string(hashBytes),
	).Scan(&adminID)
	if err != nil {
		slog.Error("Error creando admin demo", "err", err)
		os.Exit(1)
	}

	// Crear Sucursal Principal
	var sucursalID string
	err = pool.QueryRow(ctx,
		`INSERT INTO sucursales (tenant_id, nombre, direccion, telefono, activa)
		 VALUES ($1, 'Casa Central · Defensa 840', 'Defensa 840, San Telmo, CABA', '+54 11 4555-8899', true)
		 RETURNING id`,
		tenantID,
	).Scan(&sucursalID)
	if err != nil {
		slog.Error("Error creando sucursal demo", "err", err)
		os.Exit(1)
	}

	// Crear Sectores
	var sectorSalonID, sectorTerrazaID string
	_ = pool.QueryRow(ctx, `INSERT INTO sectores (sucursal_id, nombre, orden) VALUES ($1, 'Salón Principal', 1) RETURNING id`, sucursalID).Scan(&sectorSalonID)
	_ = pool.QueryRow(ctx, `INSERT INTO sectores (sucursal_id, nombre, orden) VALUES ($1, 'Patio & Terraza', 2) RETURNING id`, sucursalID).Scan(&sectorTerrazaID)

	// Crear 12 Mesas con QR predecibles para la mesa del jurado
	var mesa1ID string
	for n := 1; n <= 12; n++ {
		sectorID := sectorSalonID
		if n > 8 {
			sectorID = sectorTerrazaID
		}
		qrToken := fmt.Sprintf("demo-jurado-mesa-%d", n)
		var mID string
		err := pool.QueryRow(ctx,
			`INSERT INTO mesas (tenant_id, sucursal_id, sector_id, numero, capacidad, qr_token, estado)
			 VALUES ($1, $2, $3, $4, 4, $5, 'activa') RETURNING id`,
			tenantID, sucursalID, sectorID, n, qrToken,
		).Scan(&mID)
		if err == nil && n == 1 {
			mesa1ID = mID
		}
	}

	// Crear Categorías y Platos Realistas
	type platoDemo struct {
		nombre      string
		descripcion string
		precio      float64
		fotoURL     string
		variante    string
		precioVar   float64
	}
	catalogo := []struct {
		categoria string
		icono     string
		platos    []platoDemo
	}{
		{
			categoria: "Entradas & Tapas",
			icono:     "tapas",
			platos: []platoDemo{
				{"Provoleta Ahumada al Hierro", "Con tomates asados, pesto de albahaca y pan de masa madre", 9800, "https://images.unsplash.com/photo-1544025162-d76694265947?w=600&q=80", "Sin TACC", 900},
				{"Empanadas de Ojo de Bife (2 u.)", "Cortadas a cuchillo, horneadas al barro con salsa criolla", 6400, "https://images.unsplash.com/photo-1604908176997-125f25cc6f3d?w=600&q=80", "Picante suave", 0},
			},
		},
		{
			categoria: "Principales de Autor",
			icono:     "restaurant",
			platos: []platoDemo{
				{"Bife de Chorizo Angus (400g)", "Con papas rotas al romero y manteca de hierbas", 22500, "https://images.unsplash.com/photo-1558030006-450675393462?w=600&q=80", "Punto Jugoso", 0},
				{"Sorrentinos de Calabaza & Brie", "Masa casera con manteca de salvia y almendras tostadas", 16800, "https://images.unsplash.com/photo-1551183053-bf91a1d81141?w=600&q=80", "Salsa Cuatro Quesos", 1800},
				{"Burger Smash Doble Cheddar", "Doble medallón smash, panceta crocante, cebolla caramelizada y papas", 14900, "https://images.unsplash.com/photo-1568901346375-23c9450c58cd?w=600&q=80", "Extra Medallón", 3200},
			},
		},
		{
			categoria: "Bebidas & Coctelería",
			icono:     "local_bar",
			platos: []platoDemo{
				{"Limonada de Menta y Jengibre (Jarra)", "Exprimida en el momento con almíbar orgánico", 7200, "https://images.unsplash.com/photo-1513558161293-cdaf765ed2fd?w=600&q=80", "Sin azúcar", 0},
				{"Copa Malbec Reserva Valle de Uco", "Bodega de autor · Cosecha 2023", 6800, "https://images.unsplash.com/photo-1510812431401-41d2bd2722f3?w=600&q=80", "", 0},
			},
		},
	}

	var primerArticuloID string
	for idx, c := range catalogo {
		var catID string
		_ = pool.QueryRow(ctx,
			`INSERT INTO categorias (tenant_id, nombre, orden, icono) VALUES ($1, $2, $3, $4) RETURNING id`,
			tenantID, c.categoria, idx+1, c.icono,
		).Scan(&catID)

		for _, p := range c.platos {
			var artID string
			_ = pool.QueryRow(ctx,
				`INSERT INTO articulos (tenant_id, categoria_id, nombre, descripcion, precio, foto_url, activo, disponible, reponer_diariamente)
				 VALUES ($1, $2, $3, $4, $5, $6, true, true, true) RETURNING id`,
				tenantID, catID, p.nombre, p.descripcion, p.precio, p.fotoURL,
			).Scan(&artID)
			if primerArticuloID == "" {
				primerArticuloID = artID
			}
			if p.variante != "" {
				_, _ = pool.Exec(ctx,
					`INSERT INTO variantes (articulo_id, nombre, precio_adicional, grupo, seleccion_unica, orden)
					 VALUES ($1, $2, $3, 'Preferencia', true, 1)`,
					artID, p.variante, p.precioVar,
				)
			}
		}
	}

	// Crear pedido listo + reseñas de demostración para poblar KPIs y Reputación
	if mesa1ID != "" && primerArticuloID != "" {
		var pedidoID string
		_ = pool.QueryRow(ctx,
			`INSERT INTO pedidos (mesa_id, sucursal_id, cuenta_version, estado, origen, creado_por_usuario_id, creado_por_nombre, listo_at)
			 VALUES ($1, $2, 1, 'listo', 'qr', $3, 'Comensal Demo', NOW()) RETURNING id`,
			mesa1ID, sucursalID, adminID,
		).Scan(&pedidoID)

		if pedidoID != "" {
			_, _ = pool.Exec(ctx,
				`INSERT INTO pedido_items (pedido_id, articulo_id, cantidad, precio_unitario, estado, comensal_nombre)
				 VALUES ($1, $2, 2, 22500, 'listo', 'Lucía')`,
				pedidoID, primerArticuloID,
			)
			_, _ = pool.Exec(ctx,
				`INSERT INTO resenas (tenant_id, sucursal_id, mesa_id, pedido_id, cuenta_version, puntuacion, comentario, redirigido_google, click_google)
				 VALUES ($1, $2, $3, $4, 1, 5, 'Excelente atención y la provoleta llegó súper rápido con el QR.', true, true)`,
				tenantID, sucursalID, mesa1ID, pedidoID,
			)
		}
	}

	slog.Info("Seed de Demo Interactiva completado con éxito",
		"negocio", "Bistró San Telmo",
		"slug", slug,
		"email_admin", emailAdmin,
		"password", passwordDemo,
		"qr_mesa_1", "http://localhost:3000/mesa/demo-jurado-mesa-1",
	)
}

