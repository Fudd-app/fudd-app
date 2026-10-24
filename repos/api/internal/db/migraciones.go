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

	"github.com/jackc/pgx/v5"
)

// claveLockMigraciones identifica el advisory lock que serializa las migraciones.
// Es un número fijo arbitrario: lo único que importa es que todos los procesos
// que migran la misma base usen el mismo.
const claveLockMigraciones int64 = 7_262_873_421

// esperaMaximaLock acota cuánto espera un proceso a que otro termine de migrar.
// Mejor fallar el arranque con un error claro que colgarse sin logs.
const esperaMaximaLock = "2min"

// EjecutarMigraciones corre todos los archivos .sql de la carpeta migrations/.
//
// Si dos procesos migran la misma base a la vez (tests de integración en paralelo,
// dos instancias arrancando juntas), cada migración toma un advisory lock de
// transacción: el segundo proceso espera al primero y después la encuentra registrada.
// Se usa el lock de transacción y no el de sesión porque prod y QA se conectan por el
// pooler de Neon (PgBouncer en modo transacción), donde los locks de sesión no sirven:
// el lock y el unlock pueden caer en conexiones distintas del servidor.
func EjecutarMigraciones(carpeta string) error {
	ctx := context.Background()

	// Primero creamos la tabla que registra qué migraciones ya corrieron.
	// Va con el lock tomado: dos CREATE TABLE IF NOT EXISTS simultáneos también chocan.
	err := enTransaccionConLock(ctx, func(tx pgx.Tx) error {
		return crearTablaMigraciones(ctx, tx)
	})
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

	// Una sola lectura para saltear rápido las ya ejecutadas (el caso normal al arrancar).
	// Las pendientes se vuelven a chequear con el lock tomado.
	ejecutadas, err := migracionesEjecutadas(ctx)
	if err != nil {
		return fmt.Errorf("error leyendo migraciones ejecutadas: %w", err)
	}

	for _, archivo := range archivos {
		nombre := filepath.Base(archivo)

		if ejecutadas[nombre] {
			slog.Info("migración ya ejecutada, se saltea", "archivo", nombre)
			continue
		}

		// Leemos el contenido del archivo SQL
		contenido, err := os.ReadFile(archivo)
		if err != nil {
			return fmt.Errorf("error leyendo %s: %w", nombre, err)
		}

		var ejecutada bool
		err = enTransaccionConLock(ctx, func(tx pgx.Tx) error {
			var errMig error
			ejecutada, errMig = ejecutarMigracion(ctx, tx, nombre, string(contenido))
			return errMig
		})
		if err != nil {
			return err
		}

		if ejecutada {
			slog.Info("migración ejecutada", "archivo", nombre)
		} else {
			slog.Info("migración ejecutada por otro proceso, se saltea", "archivo", nombre)
		}
	}

	return nil
}

// enTransaccionConLock corre fn en una transacción que primero toma el lock de
// migraciones. El lock se libera solo con el COMMIT o el ROLLBACK.
func enTransaccionConLock(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error iniciando transacción: %w", err)
	}
	// Después de un Commit exitoso el Rollback no hace nada.
	defer func() { _ = tx.Rollback(ctx) }()

	// lock_timeout también acota la espera de advisory locks. Se restablece después
	// para no limitar los locks que tomen las propias migraciones.
	if _, err := tx.Exec(ctx, "SET LOCAL lock_timeout = '"+esperaMaximaLock+"'"); err != nil {
		return fmt.Errorf("error configurando lock_timeout: %w", err)
	}
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", claveLockMigraciones); err != nil {
		return fmt.Errorf("error tomando lock de migraciones (otro proceso lleva más de %s migrando?): %w", esperaMaximaLock, err)
	}
	if _, err := tx.Exec(ctx, "SET LOCAL lock_timeout = DEFAULT"); err != nil {
		return fmt.Errorf("error restableciendo lock_timeout: %w", err)
	}

	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("error confirmando transacción: %w", err)
	}
	return nil
}

// ejecutarMigracion corre el SQL y registra la migración en la transacción recibida:
// si algo falla, no queda aplicada a medias ni registrada como hecha. Devuelve false
// si otro proceso ya la había registrado mientras se esperaba el lock.
func ejecutarMigracion(ctx context.Context, tx pgx.Tx, nombre, contenido string) (bool, error) {
	var yaEjecutada bool
	err := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM _migraciones WHERE nombre = $1)", nombre).Scan(&yaEjecutada)
	if err != nil {
		return false, fmt.Errorf("error verificando migración %s: %w", nombre, err)
	}
	if yaEjecutada {
		return false, nil
	}

	if _, err := tx.Exec(ctx, contenido); err != nil {
		return false, fmt.Errorf("error ejecutando migración %s: %w", nombre, err)
	}
	if _, err := tx.Exec(ctx, "INSERT INTO _migraciones (nombre) VALUES ($1)", nombre); err != nil {
		return false, fmt.Errorf("error registrando migración %s: %w", nombre, err)
	}
	return true, nil
}

// crearTablaMigraciones crea la tabla que lleva el registro de migraciones.
// Si ya existe, no hace nada.
func crearTablaMigraciones(ctx context.Context, tx pgx.Tx) error {
	sql := `
		CREATE TABLE IF NOT EXISTS _migraciones (
			nombre     TEXT        PRIMARY KEY,
			ejecutada_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`
	_, err := tx.Exec(ctx, strings.TrimSpace(sql))
	return err
}

// migracionesEjecutadas devuelve el conjunto de migraciones ya registradas.
func migracionesEjecutadas(ctx context.Context) (map[string]bool, error) {
	filas, err := Pool.Query(ctx, "SELECT nombre FROM _migraciones")
	if err != nil {
		return nil, err
	}
	nombres, err := pgx.CollectRows(filas, pgx.RowTo[string])
	if err != nil {
		return nil, err
	}
	ejecutadas := make(map[string]bool, len(nombres))
	for _, nombre := range nombres {
		ejecutadas[nombre] = true
	}
	return ejecutadas, nil
}
