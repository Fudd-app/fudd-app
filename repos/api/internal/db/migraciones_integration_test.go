package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const carpetaMigraciones = "../../migrations"

// Dos procesos que migran la misma base vacía a la vez (p. ej. paquetes de tests en
// paralelo en CI, o dos instancias arrancando juntas) deben terminar sin error y dejar
// cada migración registrada una sola vez.
func TestEjecutarMigraciones_Concurrentes(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL no configurada, saltando test de integración")
	}
	if err := Conectar(); err != nil {
		t.Skipf("no se pudo conectar a la base de datos: %v", err)
	}
	t.Cleanup(Cerrar)
	ctx := context.Background()

	// Migrar primero el schema por defecto deja pgcrypto creada en public. Si no, en una
	// base nueva la extensión se crearía dentro del schema temporal y el DROP ... CASCADE
	// del cleanup la borraría junto con lo que dependa de ella.
	if err := EjecutarMigraciones(carpetaMigraciones); err != nil {
		t.Fatalf("migrando schema por defecto: %v", err)
	}

	// Un schema nuevo hace de "base vacía" sin tocar los datos de los demás tests.
	esquema := fmt.Sprintf("test_migraciones_%d", time.Now().UnixNano())
	if _, err := Pool.Exec(ctx, "CREATE SCHEMA "+esquema); err != nil {
		t.Fatalf("creando schema: %v", err)
	}
	poolOriginal := Pool
	t.Cleanup(func() {
		if _, err := poolOriginal.Exec(ctx, "DROP SCHEMA "+esquema+" CASCADE"); err != nil {
			t.Errorf("borrando schema %s: %v", esquema, err)
		}
	})

	cfg, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("parseando DATABASE_URL: %v", err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = esquema + ", public"
	poolEsquema, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("creando pool del schema: %v", err)
	}
	Pool = poolEsquema
	t.Cleanup(func() {
		Pool = poolOriginal
		poolEsquema.Close()
	})

	const procesos = 4
	var largada sync.WaitGroup
	largada.Add(1)
	errores := make(chan error, procesos)
	for range procesos {
		go func() {
			largada.Wait()
			errores <- EjecutarMigraciones(carpetaMigraciones)
		}()
	}
	largada.Done()
	for range procesos {
		if err := <-errores; err != nil {
			t.Errorf("migración concurrente falló: %v", err)
		}
	}

	archivos, err := filepath.Glob(filepath.Join(carpetaMigraciones, "*.sql"))
	if err != nil {
		t.Fatalf("listando migraciones: %v", err)
	}
	var registradas int
	if err := poolEsquema.QueryRow(ctx, "SELECT count(*) FROM _migraciones").Scan(&registradas); err != nil {
		t.Fatalf("contando migraciones registradas: %v", err)
	}
	if registradas != len(archivos) {
		t.Fatalf("se esperaban %d migraciones registradas, hay %d", len(archivos), registradas)
	}
}
