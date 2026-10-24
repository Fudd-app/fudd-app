// internal/db/migraciones.go
//
// Lee los archivos de la carpeta migrations/ en orden numérico
// y los ejecuta contra la base de datos.
// Si una migración ya fue ejecutada, la saltea.
//
// Cada migración corre en una transacción junto con su registro, así que no puede
// usar sentencias que Postgres no permite dentro de una (p. ej. CREATE INDEX
// CONCURRENTLY o VACUUM).

package db

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// claveLockMigraciones identifica el advisory lock que serializa las migraciones.
// Es un número fijo arbitrario: lo único que importa es que todos los procesos
// que migran la misma base usen el mismo.
const claveLockMigraciones int64 = 7_262_873_421

// EjecutarMigraciones corre todos los archivos .sql de la carpeta migrations/.
//
// Toma un advisory lock antes de empezar: si dos procesos migran la misma base a la
// vez (tests de integración en paralelo, dos instancias arrancando juntas), el segundo
// espera al primero y después encuentra las migraciones ya registradas.
func EjecutarMigraciones(carpeta string) error {
	ctx := context.Background()

	// El lock es de sesión: hay que tomarlo, usarlo y soltarlo en la misma conexión.
	conn, err := Pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("error obteniendo conexión para migrar: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", claveLockMigraciones); err != nil {
		return fmt.Errorf("error tomando lock de migraciones: %w", err)
	}
	defer func() {
		if _, err := conn.Exec(ctx, "SELECT pg_advisory_unlock($1)", claveLockMigraciones); err != nil {
			// Si falla, Postgres libera el lock igual cuando se cierra la conexión.
			slog.Warn("no se pudo liberar el lock de migraciones", "err", err)
		}
	}()

	// Primero creamos la tabla que registra qué migraciones ya corrieron.
	// Va dentro del lock: dos CREATE TABLE IF NOT EXISTS simultáneos también chocan.
	err = crearTablaMigraciones(ctx, conn)
	if err != nil {
		return fmt.Errorf("error creando tabla de migraciones: %w", err)
	}

	// Leemos todos los archivos .sql de la carpeta
	archivos, err := filepath.Glob(filepath.Join(carpeta, "*.sql"))
	if err != nil {
		return fmt.Errorf("error leyendo carpeta de migraciones: %w", err)
	}

	// Los ordenamos por nombre (así respetamos el orden 001, 002, 003...)
	sort.Strings(archivos)

	for _, archivo := range archivos {
		nombre := filepath.Base(archivo)

		// Verificamos si esta migración ya fue ejecutada
		yaEjecutada, err := migracionYaEjecutada(ctx, conn, nombre)
		if err != nil {
			return err
		}
		if yaEjecutada {
			slog.Info("migración ya ejecutada, se saltea", "archivo", nombre)
			continue
		}

		// Leemos el contenido del archivo SQL
		contenido, err := os.ReadFile(archivo)
		if err != nil {
			return fmt.Errorf("error leyendo %s: %w", nombre, err)
		}

		err = ejecutarMigracion(ctx, conn, nombre, string(contenido))
		if err != nil {
			return err
		}

		slog.Info("migración ejecutada", "archivo", nombre)
	}

	return nil
}

// ejecutarMigracion corre el SQL y registra la migración en una misma transacción:
// si algo falla, no queda aplicada a medias ni registrada como hecha.
func ejecutarMigracion(ctx context.Context, conn *pgxpool.Conn, nombre, contenido string) error {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error iniciando transacción para %s: %w", nombre, err)
	}
	// Después de un Commit exitoso el Rollback no hace nada.
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, contenido); err != nil {
		return fmt.Errorf("error ejecutando migración %s: %w", nombre, err)
	}
	if _, err := tx.Exec(ctx, "INSERT INTO _migraciones (nombre) VALUES ($1)", nombre); err != nil {
		return fmt.Errorf("error registrando migración %s: %w", nombre, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("error confirmando migración %s: %w", nombre, err)
	}
	return nil
}

// crearTablaMigraciones crea la tabla que lleva el registro de migraciones.
// Si ya existe, no hace nada.
func crearTablaMigraciones(ctx context.Context, conn *pgxpool.Conn) error {
	sql := `
		CREATE TABLE IF NOT EXISTS _migraciones (
			nombre     TEXT        PRIMARY KEY,
			ejecutada_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`
	_, err := conn.Exec(ctx, strings.TrimSpace(sql))
	return err
}

// migracionYaEjecutada devuelve true si esa migración ya está registrada.
func migracionYaEjecutada(ctx context.Context, conn *pgxpool.Conn, nombre string) (bool, error) {
	var existe bool
	err := conn.QueryRow(
		ctx,
		"SELECT EXISTS(SELECT 1 FROM _migraciones WHERE nombre = $1)",
		nombre,
	).Scan(&existe)
	return existe, err
}
