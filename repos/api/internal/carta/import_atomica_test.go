package carta_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/carta"
	"github.com/xuri/excelize/v2"
)

func TestImportarCartaPropagaFalloAtomicoSinReportarExito(t *testing.T) {
	errTransaccion := errors.New("falló la transacción")
	store := &mockStore{
		importarCartaFn: func(context.Context, string, []carta.FilaImportacion) (carta.ResultadoImportacion, error) {
			return carta.ResultadoImportacion{}, errTransaccion
		},
	}

	resultado, err := carta.NuevoService(store).ImportarCarta(
		context.Background(),
		"tenant-1",
		[]carta.FilaImportacion{{Numero: 2, Categoria: "Cafés", Nombre: "Latte", Precio: 2800}},
	)

	if !errors.Is(err, errTransaccion) {
		t.Fatalf("se esperaba propagar el error transaccional, obtenido %v", err)
	}
	if resultado.Creados != 0 {
		t.Fatalf("una transacción fallida no debe reportar creados: %+v", resultado)
	}
}

func TestImportarCartaIgnoraIdenticosYConservaVariantesDeDescripcion(t *testing.T) {
	store := &mockStore{
		categorias: []carta.Categoria{{ID: "cat-1", TenantID: "tenant-1", Nombre: "Cafés"}},
		articulos: []carta.Articulo{{
			ID: "existente", TenantID: "tenant-1", CategoriaID: "cat-1",
			Nombre: "Latte", Descripcion: "Con vainilla", Precio: 2800,
		}},
	}
	svc := carta.NuevoService(store)

	resultado, err := svc.ImportarCarta(context.Background(), "tenant-1", []carta.FilaImportacion{
		{Numero: 2, Categoria: "Cafés", Nombre: " latte ", Descripcion: "con VAINILLA", Precio: 3000},
		{Numero: 3, Categoria: "Cafés", Nombre: "Latte", Descripcion: "Con caramelo", Precio: 3100},
		{Numero: 4, Categoria: "Cafés", Nombre: "Latte", Descripcion: "Con caramelo", Precio: 3200},
	})

	if err != nil {
		t.Fatalf("importación inesperadamente fallida: %v", err)
	}
	if resultado.Creados != 1 || resultado.Omitidos != 2 || len(resultado.Errores) != 0 {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
	if len(store.articulos) != 2 || store.articulos[1].Descripcion != "Con caramelo" {
		t.Fatalf("la variante por descripción no fue creada correctamente: %+v", store.articulos)
	}
}

func TestCSVYXLSXCompartenElMismoParseo(t *testing.T) {
	csv := "categoria,nombre,precio,descripcion\nCafés,Latte,2800,Con vainilla\n"
	filasCSV, erroresCSV := carta.ParsearCSV(strings.NewReader(csv))
	if len(erroresCSV) != 0 || len(filasCSV) != 1 {
		t.Fatalf("CSV inesperado: filas=%+v errores=%+v", filasCSV, erroresCSV)
	}

	libro := excelize.NewFile()
	hoja := libro.GetSheetName(0)
	valores := [][]interface{}{
		{"categoria", "nombre", "precio", "descripcion"},
		{"Cafés", "Latte", 2800, "Con vainilla"},
	}
	for indice, fila := range valores {
		celda, err := excelize.CoordinatesToCellName(1, indice+1)
		if err != nil {
			t.Fatal(err)
		}
		if err := libro.SetSheetRow(hoja, celda, &fila); err != nil {
			t.Fatal(err)
		}
	}
	buffer, err := libro.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	filasXLSX, erroresXLSX := carta.ParsearXLSX(bytes.NewReader(buffer.Bytes()))
	if len(erroresXLSX) != 0 || len(filasXLSX) != 1 {
		t.Fatalf("XLSX inesperado: filas=%+v errores=%+v", filasXLSX, erroresXLSX)
	}

	if filasCSV[0].Categoria != filasXLSX[0].Categoria ||
		filasCSV[0].Nombre != filasXLSX[0].Nombre ||
		filasCSV[0].Descripcion != filasXLSX[0].Descripcion ||
		filasCSV[0].Precio != filasXLSX[0].Precio {
		t.Fatalf("CSV y XLSX no produjeron la misma fila: csv=%+v xlsx=%+v", filasCSV[0], filasXLSX[0])
	}
}
