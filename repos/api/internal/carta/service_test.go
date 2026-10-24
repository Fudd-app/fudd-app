package carta_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/carta"
)

type mockStore struct {
	categorias       []carta.Categoria
	articulos        []carta.Articulo
	importarCartaFn  func(ctx context.Context, tenantID string, filas []carta.FilaImportacion) (carta.ResultadoImportacion, error)
	ajustarPreciosFn func(ctx context.Context, tenantID string, input carta.AjustePreciosInput) (*carta.AjustePreciosResultado, error)
	actualizarDispFn func(ctx context.Context, id, tenantID string, disponible bool) (*carta.Articulo, error)
	reponerTodosFn   func(ctx context.Context, tenantID string) (int, error)
}

func (m *mockStore) ListarCategorias(ctx context.Context, tenantID string) ([]carta.Categoria, error) {
	var result []carta.Categoria
	for _, c := range m.categorias {
		if c.TenantID == tenantID {
			result = append(result, c)
		}
	}
	return result, nil
}
func (m *mockStore) CrearCategoria(ctx context.Context, tenantID string, input carta.CategoriaInput) (*carta.Categoria, error) {
	c := carta.Categoria{ID: "cat-1", TenantID: tenantID, Nombre: input.Nombre, Orden: input.Orden}
	m.categorias = append(m.categorias, c)
	return &c, nil
}
func (m *mockStore) EliminarCategoria(ctx context.Context, id, tenantID string) error { return nil }
func (m *mockStore) BuscarOCrearCategoria(ctx context.Context, tenantID, nombre string) (string, error) {
	for _, c := range m.categorias {
		if c.TenantID == tenantID && c.Nombre == nombre {
			return c.ID, nil
		}
	}
	c := carta.Categoria{ID: "cat-auto", TenantID: tenantID, Nombre: nombre}
	m.categorias = append(m.categorias, c)
	return c.ID, nil
}
func (m *mockStore) ListarArticulos(ctx context.Context, tenantID string) ([]carta.Articulo, error) {
	return m.articulos, nil
}
func (m *mockStore) ExisteArticuloImportado(ctx context.Context, tenantID, categoriaID, nombre, descripcion string) (bool, error) {
	for _, articulo := range m.articulos {
		if articulo.TenantID == tenantID && articulo.CategoriaID == categoriaID &&
			strings.EqualFold(strings.TrimSpace(articulo.Nombre), strings.TrimSpace(nombre)) &&
			strings.EqualFold(strings.TrimSpace(articulo.Descripcion), strings.TrimSpace(descripcion)) {
			return true, nil
		}
	}
	return false, nil
}
func (m *mockStore) ImportarCartaAtomica(ctx context.Context, tenantID string, filas []carta.FilaImportacion) (carta.ResultadoImportacion, error) {
	if m.importarCartaFn != nil {
		return m.importarCartaFn(ctx, tenantID, filas)
	}
	resultado := carta.ResultadoImportacion{Errores: []carta.ErrorFila{}}
	categoriasOriginales := append([]carta.Categoria(nil), m.categorias...)
	articulosOriginales := append([]carta.Articulo(nil), m.articulos...)

	for _, fila := range filas {
		catID, err := m.BuscarOCrearCategoria(ctx, tenantID, fila.Categoria)
		if err != nil {
			m.categorias = categoriasOriginales
			m.articulos = articulosOriginales
			return resultado, err
		}
		existe, err := m.ExisteArticuloImportado(ctx, tenantID, catID, fila.Nombre, fila.Descripcion)
		if err != nil {
			m.categorias = categoriasOriginales
			m.articulos = articulosOriginales
			return resultado, err
		}
		if existe {
			resultado.Omitidos++
			continue
		}
		_, err = m.CrearArticulo(ctx, tenantID, carta.ArticuloInput{
			CategoriaID: catID,
			Nombre:      strings.TrimSpace(fila.Nombre),
			Descripcion: strings.TrimSpace(fila.Descripcion),
			Precio:      fila.Precio,
		})
		if err != nil {
			m.categorias = categoriasOriginales
			m.articulos = articulosOriginales
			return resultado, err
		}
		resultado.Creados++
	}
	return resultado, nil
}
func (m *mockStore) CrearArticulo(ctx context.Context, tenantID string, input carta.ArticuloInput) (*carta.Articulo, error) {
	a := carta.Articulo{ID: "art-1", TenantID: tenantID, CategoriaID: input.CategoriaID, Nombre: input.Nombre, Descripcion: input.Descripcion, Precio: input.Precio, FotoURL: input.FotoURL, Activo: true}
	m.articulos = append(m.articulos, a)
	return &a, nil
}

func TestAsignarIconoCategoria_ValidaCatalogo(t *testing.T) {
	svc := carta.NuevoService(&mockStore{})
	valido := "local_cafe"
	cat, err := svc.AsignarIconoCategoria(context.Background(), "cat-1", "t-1", &valido)
	if err != nil || cat.Icono == nil || *cat.Icono != valido {
		t.Fatalf("se esperaba icono válido, categoría=%+v error=%v", cat, err)
	}

	invalido := "<svg>"
	_, err = svc.AsignarIconoCategoria(context.Background(), "cat-1", "t-1", &invalido)
	if !errors.Is(err, carta.ErrValidation) {
		t.Fatalf("se esperaba ErrValidation para icono arbitrario, obtenido %v", err)
	}
}

func TestCrearArticulo_RechazaFotoNoRaster(t *testing.T) {
	svc := carta.NuevoService(&mockStore{})
	_, err := svc.CrearArticulo(context.Background(), "t-1", carta.ArticuloInput{
		CategoriaID: "cat-1",
		Nombre:      "Producto",
		Precio:      100,
		FotoURL:     "data:image/svg+xml;base64,PHN2Zz4=",
	})
	if !errors.Is(err, carta.ErrValidation) {
		t.Fatalf("se esperaba ErrValidation para SVG, obtenido %v", err)
	}
}

func TestCrearYActualizarArticulo_ValidanLongitudDescripcion(t *testing.T) {
	svc := carta.NuevoService(&mockStore{})
	descripcionEnLimite := strings.Repeat("á", 160)
	descripcionExcedida := strings.Repeat("á", 161)

	if _, err := svc.CrearArticulo(context.Background(), "t-1", carta.ArticuloInput{
		CategoriaID: "cat-1",
		Nombre:      "Producto",
		Descripcion: " " + descripcionEnLimite + " ",
		Precio:      100,
	}); err != nil {
		t.Fatalf("la descripción en el límite debería aceptarse: %v", err)
	}

	if _, err := svc.CrearArticulo(context.Background(), "t-1", carta.ArticuloInput{
		CategoriaID: "cat-1",
		Nombre:      "Producto",
		Descripcion: descripcionExcedida,
		Precio:      100,
	}); !errors.Is(err, carta.ErrValidation) {
		t.Fatalf("se esperaba ErrValidation al crear con descripción excedida, obtenido %v", err)
	}

	if _, err := svc.ActualizarArticulo(context.Background(), "art-1", "t-1", carta.ArticuloUpdate{
		Descripcion: &descripcionExcedida,
	}); !errors.Is(err, carta.ErrValidation) {
		t.Fatalf("se esperaba ErrValidation al actualizar con descripción excedida, obtenido %v", err)
	}
}

func (m *mockStore) ActualizarArticulo(ctx context.Context, id, tenantID string, u carta.ArticuloUpdate) (*carta.Articulo, error) {
	return &carta.Articulo{ID: id}, nil
}
func (m *mockStore) ActualizarDisponibilidad(ctx context.Context, id, tenantID string, disponible bool) (*carta.Articulo, error) {
	if m.actualizarDispFn != nil {
		return m.actualizarDispFn(ctx, id, tenantID, disponible)
	}
	for i, a := range m.articulos {
		if a.ID == id {
			m.articulos[i].Disponible = disponible
			return &m.articulos[i], nil
		}
	}
	return &carta.Articulo{ID: id, Disponible: disponible}, nil
}
func (m *mockStore) ReponerTodos(ctx context.Context, tenantID string) (int, error) {
	if m.reponerTodosFn != nil {
		return m.reponerTodosFn(ctx, tenantID)
	}
	count := 0
	for i := range m.articulos {
		if !m.articulos[i].Disponible {
			m.articulos[i].Disponible = true
			count++
		}
	}
	return count, nil
}
func (m *mockStore) AjustarPrecios(ctx context.Context, tenantID string, input carta.AjustePreciosInput) (*carta.AjustePreciosResultado, error) {
	if m.ajustarPreciosFn != nil {
		return m.ajustarPreciosFn(ctx, tenantID, input)
	}
	return &carta.AjustePreciosResultado{Actualizados: len(m.articulos)}, nil
}
func (m *mockStore) EliminarArticulo(ctx context.Context, id, tenantID string) error { return nil }
func (m *mockStore) ObtenerCartaPublica(ctx context.Context, sucursalID string) (*carta.CartaPublica, error) {
	return &carta.CartaPublica{}, nil
}
func (m *mockStore) ListarFranjasHorarias(ctx context.Context, tenantID string) ([]carta.FranjaHoraria, error) {
	return []carta.FranjaHoraria{}, nil
}
func (m *mockStore) CrearFranjaHoraria(ctx context.Context, tenantID string, input carta.FranjaHorariaInput) (*carta.FranjaHoraria, error) {
	return &carta.FranjaHoraria{ID: "franja-1", TenantID: tenantID, Nombre: input.Nombre, HoraInicio: input.HoraInicio, HoraFin: input.HoraFin}, nil
}
func (m *mockStore) ActualizarFranjaHoraria(ctx context.Context, id, tenantID string, input carta.FranjaHorariaInput) (*carta.FranjaHoraria, error) {
	return &carta.FranjaHoraria{ID: id, TenantID: tenantID, Nombre: input.Nombre, HoraInicio: input.HoraInicio, HoraFin: input.HoraFin}, nil
}
func (m *mockStore) EliminarFranjaHoraria(ctx context.Context, id, tenantID string) error {
	return nil
}
func (m *mockStore) AsignarFranjaCategoria(ctx context.Context, id, tenantID string, franjaID *string) (*carta.Categoria, error) {
	return &carta.Categoria{ID: id, TenantID: tenantID, FranjaHorariaID: franjaID}, nil
}
func (m *mockStore) AsignarIconoCategoria(ctx context.Context, id, tenantID string, icono *string) (*carta.Categoria, error) {
	return &carta.Categoria{ID: id, TenantID: tenantID, Icono: icono}, nil
}
func (m *mockStore) AsignarFranjaArticulo(ctx context.Context, id, tenantID string, franjaID *string) (*carta.Articulo, error) {
	return &carta.Articulo{ID: id, TenantID: tenantID, FranjaHorariaID: franjaID}, nil
}
func (m *mockStore) ListarVariantes(ctx context.Context, articuloID, tenantID string) ([]carta.Variante, error) {
	return []carta.Variante{}, nil
}
func (m *mockStore) CrearVariante(ctx context.Context, articuloID, tenantID string, input carta.CrearVarianteInput) (*carta.Variante, error) {
	return &carta.Variante{ID: "v-1", ArticuloID: articuloID, Nombre: input.Nombre, PrecioAdicional: input.PrecioAdicional}, nil
}
func (m *mockStore) ActualizarVariante(ctx context.Context, id, tenantID string, input carta.ActualizarVarianteInput) (*carta.Variante, error) {
	return &carta.Variante{ID: id}, nil
}
func (m *mockStore) EliminarVariante(ctx context.Context, id, tenantID string) error { return nil }

func TestListarCategorias(t *testing.T) {
	store := &mockStore{
		categorias: []carta.Categoria{
			{ID: "1", TenantID: "t-1", Nombre: "Bebidas"},
			{ID: "2", TenantID: "t-2", Nombre: "Otras"},
		},
	}
	svc := carta.NuevoService(store)
	cats, err := svc.ListarCategorias(context.Background(), "t-1")
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if len(cats) != 1 {
		t.Errorf("got %d categorías, want 1", len(cats))
	}
	if cats[0].Nombre != "Bebidas" {
		t.Errorf("got %q, want %q", cats[0].Nombre, "Bebidas")
	}
}

func TestCrearArticulo_PrecioNegativo(t *testing.T) {
	svc := carta.NuevoService(&mockStore{})
	_, err := svc.CrearArticulo(context.Background(), "t-1", carta.ArticuloInput{
		Nombre: "Test",
		Precio: -5,
	})
	if err == nil {
		t.Fatal("esperaba error por precio negativo")
	}
}

func TestCrearVariante_Validaciones(t *testing.T) {
	svc := carta.NuevoService(&mockStore{})
	_, err := svc.CrearVariante(context.Background(), "", "t-1", carta.CrearVarianteInput{
		Nombre:          "Extra queso",
		PrecioAdicional: 100,
	})
	if err == nil {
		t.Fatal("esperaba error por articulo_id vacío")
	}

	_, err = svc.CrearVariante(context.Background(), "art-1", "t-1", carta.CrearVarianteInput{
		Nombre:          "",
		PrecioAdicional: 100,
	})
	if err == nil {
		t.Fatal("esperaba error por nombre vacío")
	}

	_, err = svc.CrearVariante(context.Background(), "art-1", "t-1", carta.CrearVarianteInput{
		Nombre:          "Extra queso",
		PrecioAdicional: -10,
	})
	if err == nil {
		t.Fatal("esperaba error por precio_adicional negativo")
	}
}

func TestAjustarPrecios_Exitoso(t *testing.T) {
	store := &mockStore{
		ajustarPreciosFn: func(ctx context.Context, tenantID string, input carta.AjustePreciosInput) (*carta.AjustePreciosResultado, error) {
			if tenantID != "t-1" {
				t.Fatalf("tenantID: got %q, want %q", tenantID, "t-1")
			}
			if input.CategoriaID != "cat-1" || input.Porcentaje != 12.5 || input.Redondeo != carta.Redondeo100 {
				t.Fatalf("input inesperado: %+v", input)
			}
			return &carta.AjustePreciosResultado{Actualizados: 3}, nil
		},
	}

	resultado, err := carta.NuevoService(store).AjustarPrecios(context.Background(), "t-1", carta.AjustePreciosInput{
		CategoriaID: " cat-1 ",
		Porcentaje:  12.5,
		Redondeo:    carta.Redondeo100,
	})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if resultado.Actualizados != 3 {
		t.Fatalf("actualizados: got %d, want 3", resultado.Actualizados)
	}
}

func TestAjustarPrecios_Validaciones(t *testing.T) {
	tests := []struct {
		nombre string
		input  carta.AjustePreciosInput
	}{
		{nombre: "porcentaje cero", input: carta.AjustePreciosInput{Porcentaje: 0}},
		{nombre: "descuento menor a menos cien", input: carta.AjustePreciosInput{Porcentaje: -100.01}},
		{nombre: "incremento excesivo", input: carta.AjustePreciosInput{Porcentaje: 1000.01}},
		{nombre: "redondeo inválido", input: carta.AjustePreciosInput{Porcentaje: 10, Redondeo: "50"}},
	}

	for _, tt := range tests {
		t.Run(tt.nombre, func(t *testing.T) {
			_, err := carta.NuevoService(&mockStore{}).AjustarPrecios(context.Background(), "t-1", tt.input)
			if !errors.Is(err, carta.ErrValidation) {
				t.Fatalf("se esperaba ErrValidation, obtenido: %v", err)
			}
		})
	}
}

func TestAjustarPrecios_SinProductos(t *testing.T) {
	store := &mockStore{
		ajustarPreciosFn: func(ctx context.Context, tenantID string, input carta.AjustePreciosInput) (*carta.AjustePreciosResultado, error) {
			return &carta.AjustePreciosResultado{Actualizados: 0}, nil
		},
	}

	_, err := carta.NuevoService(store).AjustarPrecios(context.Background(), "t-1", carta.AjustePreciosInput{Porcentaje: 10})
	if !errors.Is(err, carta.ErrValidation) {
		t.Fatalf("se esperaba ErrValidation, obtenido: %v", err)
	}
}

func TestCrearFranjaHoraria_Validaciones(t *testing.T) {
	svc := carta.NuevoService(&mockStore{})
	casos := []carta.FranjaHorariaInput{
		{Nombre: "", HoraInicio: "08:00", HoraFin: "12:00"},
		{Nombre: "Desayuno", HoraInicio: "8", HoraFin: "12:00"},
		{Nombre: "Desayuno", HoraInicio: "08:00", HoraFin: "08:00"},
	}
	for _, input := range casos {
		if _, err := svc.CrearFranjaHoraria(context.Background(), "t-1", input); !errors.Is(err, carta.ErrValidation) {
			t.Fatalf("se esperaba ErrValidation para %+v, obtenido: %v", input, err)
		}
	}
}

func TestResolverCartaPorHora_FiltraYConservaOpcionales(t *testing.T) {
	desayuno := &carta.FranjaHoraria{ID: "f-1", Nombre: "Desayuno", HoraInicio: "08:00", HoraFin: "12:00"}
	cartaOriginal := &carta.CartaPublica{Categorias: []carta.CategoriaConArticulos{
		{
			Categoria: carta.Categoria{ID: "cat-1", Nombre: "Desayunos"},
			Articulos: []carta.Articulo{
				{ID: "siempre", Nombre: "Café"},
				{ID: "desayuno", Nombre: "Tostadas", FranjaEfectiva: desayuno},
			},
		},
	}}

	resultado := carta.ResolverCartaPorHora(cartaOriginal, time.Date(2026, 9, 29, 15, 0, 0, 0, time.UTC))
	if len(resultado.Categorias) != 1 || len(resultado.Categorias[0].Articulos) != 1 {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
	if resultado.Categorias[0].Articulos[0].ID != "siempre" {
		t.Fatalf("el artículo sin franja debe seguir visible")
	}
	if !resultado.Categorias[0].Disponible {
		t.Fatalf("la categoría debe figurar disponible mientras tenga un artículo visible")
	}
}

func TestResolverCartaPorHora_InformaProximaDisponibilidad(t *testing.T) {
	merienda := &carta.FranjaHoraria{ID: "f-2", Nombre: "Merienda", HoraInicio: "16:00", HoraFin: "20:00"}
	cartaOriginal := &carta.CartaPublica{Categorias: []carta.CategoriaConArticulos{
		{
			Categoria: carta.Categoria{ID: "cat-2", Nombre: "Merienda"},
			Articulos: []carta.Articulo{{ID: "torta", Nombre: "Torta", FranjaEfectiva: merienda}},
		},
	}}

	resultado := carta.ResolverCartaPorHora(cartaOriginal, time.Date(2026, 9, 29, 14, 30, 0, 0, time.UTC))
	categoria := resultado.Categorias[0]
	if categoria.Disponible || len(categoria.Articulos) != 0 || categoria.DisponibleDesde != "16:00" {
		t.Fatalf("categoría inesperada: %+v", categoria)
	}
}

func TestResolverCartaPorHora_FranjaQueCruzaMedianoche(t *testing.T) {
	noche := &carta.FranjaHoraria{ID: "f-3", Nombre: "Noche", HoraInicio: "20:00", HoraFin: "02:00"}
	base := &carta.CartaPublica{Categorias: []carta.CategoriaConArticulos{{
		Categoria: carta.Categoria{ID: "cat-3", Nombre: "Noche"},
		Articulos: []carta.Articulo{{ID: "burger", Nombre: "Burger", FranjaEfectiva: noche}},
	}}}

	resultado := carta.ResolverCartaPorHora(base, time.Date(2026, 9, 29, 1, 0, 0, 0, time.UTC))
	if !resultado.Categorias[0].Disponible {
		t.Fatalf("la franja nocturna debe estar activa después de medianoche")
	}
}

func TestActualizarDisponibilidad(t *testing.T) {
	store := &mockStore{
		articulos: []carta.Articulo{
			{ID: "art-1", TenantID: "t-1", Nombre: "Hamburguesa", Disponible: true},
		},
	}
	svc := carta.NuevoService(store)

	// Marcar como 86 (agotado)
	art, err := svc.ActualizarDisponibilidad(context.Background(), "art-1", "t-1", false)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if art.Disponible {
		t.Fatalf("se esperaba disponible=false, obtenido: %v", art.Disponible)
	}

	// Marcar nuevamente como disponible
	art, err = svc.ActualizarDisponibilidad(context.Background(), "art-1", "t-1", true)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if !art.Disponible {
		t.Fatalf("se esperaba disponible=true, obtenido: %v", art.Disponible)
	}
}

func TestActualizarDisponibilidad_Validacion(t *testing.T) {
	svc := carta.NuevoService(&mockStore{})
	_, err := svc.ActualizarDisponibilidad(context.Background(), "   ", "t-1", false)
	if !errors.Is(err, carta.ErrValidation) {
		t.Fatalf("se esperaba ErrValidation por ID vacío, obtenido: %v", err)
	}
}

func TestReponerTodos(t *testing.T) {
	store := &mockStore{
		articulos: []carta.Articulo{
			{ID: "art-1", TenantID: "t-1", Nombre: "Hamburguesa", Disponible: false},
			{ID: "art-2", TenantID: "t-1", Nombre: "Cerveza", Disponible: false},
			{ID: "art-3", TenantID: "t-1", Nombre: "Papas", Disponible: true},
		},
	}
	svc := carta.NuevoService(store)

	res, err := svc.ReponerTodos(context.Background(), "t-1")
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if res.Repuestos != 2 {
		t.Fatalf("repuestos: se esperaba 2, obtenido: %d", res.Repuestos)
	}
	for _, a := range store.articulos {
		if !a.Disponible {
			t.Fatalf("el artículo %s debería estar disponible", a.Nombre)
		}
	}
}
