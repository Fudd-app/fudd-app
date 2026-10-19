package main_test

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/carta"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/db"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/mesa"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/pedido"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/sucursal"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/tenant"
	"github.com/joho/godotenv"
)

func TestIntegracion_FlujoCompletoPedido(t *testing.T) {
	// Intentar cargar .env local si existe
	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL no configurada, saltando test de integración")
	}

	err := db.Conectar()
	if err != nil {
		t.Skipf("no se pudo conectar a la base de datos (PostgreSQL probablemente apagado): %v. Saltando test de integración.", err)
		return
	}
	defer db.Cerrar()

	// Ejecutar migraciones por seguridad
	err = db.EjecutarMigraciones("migrations")
	if err != nil {
		t.Skipf("no se pudieron correr migraciones (DB en modo lectura o sin permisos DDL): %v. Saltando test de integración.", err)
		return
	}

	ctx := context.Background()

	// Inicializar los stores y services reales de la app
	tenantStore := tenant.NuevoStore()
	tenantSvc := tenant.NuevoService(tenantStore)

	sucStore := sucursal.NuevoStore()
	sucSvc := sucursal.NuevoService(sucStore)

	mesaStore := mesa.NuevoStore()
	mesaSvc := mesa.NuevoService(mesaStore)

	cartaStore := carta.NuevoStore()
	cartaSvc := carta.NuevoService(cartaStore)

	pedidoStore := pedido.NuevoStore()
	pedidoSvc := pedido.NuevoService(pedidoStore)

	// Generar datos aleatorios para evitar conflictos
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	suffix := rng.Intn(100000)
	slug := fmt.Sprintf("test-bar-%d", suffix)
	email := fmt.Sprintf("admin-%d@test.com", suffix)

	// A. CREAR TENANT (Registro / Onboarding)
	t.Log("Creando tenant de prueba...")
	createdTenant, err := tenantSvc.Crear(ctx, tenant.OnboardingInput{
		Nombre:      "Test Bar Integracion",
		Slug:        slug,
		EmailAdmin:  email,
		NombreAdmin: "Admin Integracion",
		Password:    "MesaClick2026",
	})
	if err != nil {
		t.Fatalf("error creando tenant: %v", err)
	}
	// Limpieza al terminar
	defer func() {
		t.Log("Limpiando datos del tenant de prueba...")
		// Al eliminar el tenant en cascada, PostgreSQL elimina usuarios, sucursales, sectores, mesas y pedidos.
		_, _ = db.Pool.Exec(ctx, "DELETE FROM tenants WHERE id = $1", createdTenant.ID)
	}()

	// B. OBTENER SUCURSAL DEFAULT
	t.Log("Obteniendo sucursales del tenant...")
	sucursales, err := sucSvc.Listar(ctx, createdTenant.ID)
	if err != nil {
		t.Fatalf("error listando sucursales: %v", err)
	}
	if len(sucursales) == 0 {
		t.Fatal("no se creó la sucursal predeterminada al crear el tenant")
	}
	sucursalDefault := sucursales[0]

	// C. CREAR SECTOR
	t.Log("Creando sector de prueba...")
	sector, err := sucSvc.CrearSector(ctx, sucursalDefault.ID, createdTenant.ID, sucursal.SectorInput{
		Nombre: "Terraza",
	})
	if err != nil {
		t.Fatalf("error creando sector: %v", err)
	}

	// D. CREAR MESA
	t.Log("Creando mesa de prueba...")
	createdMesa, err := mesaSvc.Crear(ctx, createdTenant.ID, mesa.MesaInput{
		SucursalID: sucursalDefault.ID,
		Numero:     15,
		Capacidad:  4,
	})
	if err != nil {
		t.Fatalf("error creando mesa: %v", err)
	}

	// Asignar el sector a la mesa (usando UPDATE directo en BD para validar integración)
	_, err = db.Pool.Exec(ctx, "UPDATE mesas SET sector_id = $1 WHERE id = $2", sector.ID, createdMesa.ID)
	if err != nil {
		t.Fatalf("error asignando sector a mesa: %v", err)
	}

	// E. CREAR CATEGORÍA Y ARTÍCULO
	t.Log("Creando categoría y artículo de prueba...")
	createdCat, err := cartaSvc.CrearCategoria(ctx, createdTenant.ID, carta.CategoriaInput{
		Nombre: "Bebidas",
		Orden:  0,
	})
	if err != nil {
		t.Fatalf("error creando categoría: %v", err)
	}

	createdArt, err := cartaSvc.CrearArticulo(ctx, createdTenant.ID, carta.ArticuloInput{
		CategoriaID: createdCat.ID,
		Nombre:      "Agua Mineral",
		Descripcion: "500ml",
		Precio:      250,
	})
	if err != nil {
		t.Fatalf("error creando artículo: %v", err)
	}

	// E.1 AJUSTAR PRECIOS DE LA CATEGORÍA (US-60)
	t.Log("Aplicando ajuste porcentual de precios...")
	ajuste, err := cartaSvc.AjustarPrecios(ctx, createdTenant.ID, carta.AjustePreciosInput{
		CategoriaID: createdCat.ID,
		Porcentaje:  10,
		Redondeo:    carta.Redondeo100,
	})
	if err != nil {
		t.Fatalf("error ajustando precios: %v", err)
	}
	if ajuste.Actualizados != 1 {
		t.Fatalf("artículos actualizados: got %d, want 1", ajuste.Actualizados)
	}

	articulos, err := cartaSvc.ListarArticulos(ctx, createdTenant.ID)
	if err != nil {
		t.Fatalf("error listando artículos luego del ajuste: %v", err)
	}
	if len(articulos) != 1 || articulos[0].ID != createdArt.ID || articulos[0].Precio != 300 {
		t.Fatalf("precio ajustado inesperado: %+v", articulos)
	}

	// F. CREAR PEDIDO (Simula la acción del cliente comensal)
	t.Log("Creando pedido desde el cliente...")
	itemsInput := []pedido.NuevoItemInput{
		{
			ArticuloID:     createdArt.ID,
			Cantidad:       1,
			Notas:          "sin gas",
			ComensalID:     "47dc8c9e-fb98-44d7-80a1-b598addc1e8a",
			ComensalNombre: "Mateo",
		},
		{
			ArticuloID:     createdArt.ID,
			Cantidad:       1,
			Notas:          "bien fría",
			ComensalID:     "5ef10747-503a-4d72-a433-94978a2447e6",
			ComensalNombre: "Juani",
		},
	}
	createdPedido, err := pedidoSvc.Crear(ctx, pedido.NuevoPedidoInput{
		MesaID: createdMesa.ID,
		Items:  itemsInput,
	})
	if err != nil {
		t.Fatalf("error creando pedido: %v", err)
	}

	if createdPedido.MesaID != createdMesa.ID {
		t.Errorf("mesa del pedido incorrecta: got %q, want %q", createdPedido.MesaID, createdMesa.ID)
	}
	if createdPedido.Estado != "recibido" {
		t.Errorf("estado inicial del pedido incorrecto: got %q, want %q", createdPedido.Estado, "recibido")
	}
	if len(createdPedido.Items) != 2 {
		t.Fatalf("cantidad de items incorrecta: got %d, want 2", len(createdPedido.Items))
	}
	for _, item := range createdPedido.Items {
		if item.Estado != "pendiente" {
			t.Errorf("estado inicial del item incorrecto: got %q, want %q", item.Estado, "pendiente")
		}
	}

	// F.1 CAMBIAR ESTADOS POR ÍTEM (Simula la interacción del KDS)
	t.Log("Cambiando el primer item a 'preparando'...")
	updatedPedido, err := pedidoSvc.CambiarEstadoItem(ctx, createdPedido.Items[0].ID, createdTenant.ID, "", "preparando")
	if err != nil {
		t.Fatalf("error actualizando estado del primer item: %v", err)
	}
	if updatedPedido.Estado != "preparando" {
		t.Errorf("el pedido debe avanzar a preparando: got %q, want %q", updatedPedido.Estado, "preparando")
	}

	t.Log("Marcando el primer item como 'listo'...")
	updatedPedido, err = pedidoSvc.CambiarEstadoItem(ctx, createdPedido.Items[0].ID, createdTenant.ID, "", "listo")
	if err != nil {
		t.Fatalf("error marcando listo el primer item: %v", err)
	}
	if updatedPedido.Estado != "preparando" {
		t.Errorf("el pedido no debe quedar listo mientras falten items: got %q", updatedPedido.Estado)
	}

	t.Log("Marcando el segundo item como 'listo'...")
	updatedPedido, err = pedidoSvc.CambiarEstadoItem(ctx, createdPedido.Items[1].ID, createdTenant.ID, "", "listo")
	if err != nil {
		t.Fatalf("error marcando listo el segundo item: %v", err)
	}
	if updatedPedido.Estado != "listo" {
		t.Errorf("todos los items listos deben avanzar el pedido a listo: got %q, want %q", updatedPedido.Estado, "listo")
	}

	// G. LISTAR PEDIDOS ACTIVOS DE LA SUCURSAL
	t.Log("Verificando listado de pedidos activos de la sucursal...")
	pedidosActivos, err := pedidoSvc.ListarActivos(ctx, sucursalDefault.ID, createdTenant.ID)
	if err != nil {
		t.Fatalf("error listando pedidos activos: %v", err)
	}

	encontrado := false
	for _, p := range pedidosActivos {
		if p.ID == createdPedido.ID {
			encontrado = true
			if p.Estado != "listo" {
				t.Errorf("el pedido en el listado de activos tiene estado incorrecto: got %q, want %q", p.Estado, "listo")
			}
			for _, item := range p.Items {
				if item.Estado != "listo" {
					t.Errorf("el item %s debería estar listo y está %q", item.ID, item.Estado)
				}
			}
			break
		}
	}
	if !encontrado {
		t.Errorf("el pedido creado %s no se encontró en la lista de activos de la sucursal", createdPedido.ID)
	}
}
