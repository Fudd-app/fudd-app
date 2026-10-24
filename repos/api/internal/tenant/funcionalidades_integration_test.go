package tenant

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/db"
)

func TestIntegration_FuncionalidadHabilitada(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL no configurada, saltando test de integración")
	}
	if err := db.Conectar(); err != nil {
		t.Skipf("no se pudo conectar a la base de datos: %v", err)
	}
	t.Cleanup(db.Cerrar)
	if err := db.EjecutarMigraciones("../../migrations"); err != nil {
		t.Fatalf("no se pudieron correr migraciones: %v", err)
	}

	ctx := context.Background()
	sufijo := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(1_000_000_000)
	var tenantID string
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO tenants (nombre, nombre_fantasia, rubro, slug) VALUES ('Test Func', 'Test Func', 'bar', $1) RETURNING id`,
		fmt.Sprintf("test-func-%d", sufijo)).Scan(&tenantID)
	if err != nil {
		t.Fatalf("creando tenant: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Pool.Exec(ctx, "DELETE FROM tenants WHERE id = $1", tenantID) })

	store := NuevoStore()
	ahora := time.Now()
	verificar := func(nombre, plan string, quiero bool) {
		t.Helper()
		got, err := store.FuncionalidadHabilitada(ctx, tenantID, RecursoMCP, plan, ahora)
		if err != nil {
			t.Fatalf("%s: %v", nombre, err)
		}
		if got != quiero {
			t.Errorf("%s: got %v, quiero %v", nombre, got, quiero)
		}
	}
	insertar := func(habilitado bool, vence *time.Time) {
		t.Helper()
		_, err := db.Pool.Exec(ctx,
			`INSERT INTO tenant_funcionalidades (tenant_id, recurso, habilitado, motivo, vence_at)
			 VALUES ($1, 'mcp', $2, 'test', $3)
			 ON CONFLICT (tenant_id, recurso) DO UPDATE SET habilitado = $2, vence_at = $3`,
			tenantID, habilitado, vence)
		if err != nil {
			t.Fatalf("insertando excepción: %v", err)
		}
	}

	verificar("free sin excepción", PlanFree, false)
	verificar("pro sin excepción", PlanPro, true)
	verificar("plan desconocido", "otro", false)

	futuro := ahora.Add(time.Hour)
	insertar(true, &futuro)
	verificar("free con excepción vigente", PlanFree, true)

	pasado := ahora.Add(-time.Hour)
	insertar(true, &pasado)
	verificar("excepción vencida", PlanFree, false)

	insertar(false, nil)
	verificar("pro con excepción en false", PlanPro, false)
}
