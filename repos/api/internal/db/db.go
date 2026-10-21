// internal/db/db.go
//
// Este paquete maneja la conexión a PostgreSQL.
// El resto del backend importa este paquete para hablar con la base de datos.

package db

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool es el pool de conexiones a la base de datos.
// Un pool mantiene varias conexiones abiertas y las reutiliza,
// en lugar de abrir y cerrar una conexión por cada request.
var Pool *pgxpool.Pool

// Conectar abre la conexión a PostgreSQL usando la variable DATABASE_URL.
// Se llama una sola vez al iniciar el servidor.
func Conectar() error {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return fmt.Errorf("DATABASE_URL no está configurada")
	}

	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return fmt.Errorf("no se pudo parsear DATABASE_URL: %w", err)
	}

	maxConns := int32(50)
	if v := os.Getenv("DB_MAX_CONNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxConns = int32(n)
		}
	}
	minConns := int32(10)
	if v := os.Getenv("DB_MIN_CONNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			minConns = int32(n)
		}
	}
	if minConns > maxConns {
		minConns = maxConns
	}

	cfg.MaxConns = maxConns
	cfg.MinConns = minConns
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		return fmt.Errorf("no se pudo crear el pool de conexiones: %w", err)
	}

	// Verificamos que la conexión funcione de verdad
	if err := pool.Ping(context.Background()); err != nil {
		return fmt.Errorf("no se pudo conectar a la base de datos: %w", err)
	}

	Pool = pool
	slog.Info("conexión a PostgreSQL establecida", "max_conns", cfg.MaxConns, "min_conns", cfg.MinConns)
	return nil
}

// Cerrar cierra todas las conexiones del pool.
// Se llama cuando el servidor se apaga.
func Cerrar() {
	if Pool != nil {
		Pool.Close()
		slog.Info("conexión a PostgreSQL cerrada")
	}
}
