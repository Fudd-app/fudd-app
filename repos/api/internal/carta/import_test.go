package carta

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
)

// xlsxValido genera un XLSX real con encabezado y una fila de carta.
func xlsxValido(t *testing.T) []byte {
	t.Helper()
	f := excelize.NewFile()
	defer f.Close()
	hoja := f.GetSheetName(0)
	_ = f.SetSheetRow(hoja, "A1", &[]any{"categoria", "nombre", "precio"})
	_ = f.SetSheetRow(hoja, "A2", &[]any{"Bebidas", "Agua", 1500})
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatalf("generando xlsx: %v", err)
	}
	return buf.Bytes()
}

// reemplazarHoja devuelve una copia del XLSX con el XML de la primera hoja
// reemplazado, para simular archivos manipulados.
func reemplazarHoja(t *testing.T, original []byte, xmlHoja string) []byte {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(original), int64(len(original)))
	if err != nil {
		t.Fatalf("leyendo zip: %v", err)
	}
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	for _, entrada := range zr.File {
		w, err := zw.Create(entrada.Name)
		if err != nil {
			t.Fatalf("creando entrada: %v", err)
		}
		if entrada.Name == "xl/worksheets/sheet1.xml" {
			_, _ = io.WriteString(w, xmlHoja)
			continue
		}
		rc, err := entrada.Open()
		if err != nil {
			t.Fatalf("abriendo entrada: %v", err)
		}
		_, _ = io.Copy(w, rc)
		rc.Close()
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("cerrando zip: %v", err)
	}
	return out.Bytes()
}

// parsearConTimeout falla el test si el parseo se cuelga en vez de terminar.
func parsearConTimeout(t *testing.T, contenido []byte) ([]FilaImportacion, []ErrorFila) {
	t.Helper()
	type resultado struct {
		filas   []FilaImportacion
		errores []ErrorFila
	}
	listo := make(chan resultado, 1)
	go func() {
		filas, errores := ParsearXLSX(bytes.NewReader(contenido))
		listo <- resultado{filas, errores}
	}()
	select {
	case r := <-listo:
		return r.filas, r.errores
	case <-time.After(10 * time.Second):
		t.Fatal("ParsearXLSX no terminó: posible loop infinito con archivo malicioso")
		return nil, nil
	}
}

func TestParsearXLSX_Valido(t *testing.T) {
	filas, errores := parsearConTimeout(t, xlsxValido(t))
	if len(errores) != 0 {
		t.Fatalf("errores inesperados: %+v", errores)
	}
	if len(filas) != 1 || filas[0].Nombre != "Agua" || filas[0].Precio != 1500 {
		t.Fatalf("filas inesperadas: %+v", filas)
	}
}

func TestParsearXLSX_ArchivoCorrupto(t *testing.T) {
	_, errores := parsearConTimeout(t, []byte("esto no es un xlsx"))
	if len(errores) == 0 || !strings.Contains(errores[0].Motivo, "malformado") {
		t.Fatalf("se esperaba error de archivo malformado, se obtuvo: %+v", errores)
	}
}

func TestParsearCSV_RechazaDescripcionDeMasDe140Caracteres(t *testing.T) {
	contenido := "categoria,nombre,precio,descripcion\nEntradas,Plato,100," + strings.Repeat("á", 141)
	filas, errores := ParsearCSV(strings.NewReader(contenido))

	if len(filas) != 0 {
		t.Fatalf("no deben aceptarse filas con descripción excedida: %+v", filas)
	}
	if len(errores) != 1 || !strings.Contains(errores[0].Motivo, "140") {
		t.Fatalf("se esperaba error de límite de descripción, obtenido: %+v", errores)
	}
}

// hojaConFilas arma el XML de una hoja con una celda en `columna` por fila.
func hojaConFilas(cantidad int, columna string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`)
	for i := 1; i <= cantidad; i++ {
		fmt.Fprintf(&b, `<row r="%d"><c r="%s%d" t="inlineStr"><is><t>x</t></is></c></row>`, i, columna, i)
	}
	b.WriteString(`</sheetData></worksheet>`)
	return b.String()
}

// Una hoja con un número de fila fuera del rango de Excel debe rechazarse sin
// colgar el parseo. Es un test de robustez: no reproduce GO-2026-6685, que se
// cubre actualizando excelize.
func TestParsearXLSX_NumeroDeFilaDesmedido(t *testing.T) {
	hoja := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>` +
		`<row r="2147483647"><c r="A2147483647" t="inlineStr"><is><t>x</t></is></c></row>` +
		`</sheetData></worksheet>`
	contenido := reemplazarHoja(t, xlsxValido(t), hoja)

	filas, errores := parsearConTimeout(t, contenido)
	if len(filas) != 0 || len(errores) == 0 {
		t.Fatalf("se esperaba rechazo del archivo, se obtuvo filas=%+v errores=%+v", filas, errores)
	}
}

// Una celda en la columna XFD por fila cuesta ~1 MB por fila en excelize aunque
// el archivo pese pocos KB. Pasado el máximo de filas el archivo se rechaza.
func TestParsearXLSX_CeldasEnUltimaColumna(t *testing.T) {
	contenido := reemplazarHoja(t, xlsxValido(t), hojaConFilas(maxFilasXLSX+1, "XFD"))

	filas, errores := parsearConTimeout(t, contenido)
	if len(filas) != 0 || len(errores) == 0 || !strings.Contains(errores[0].Motivo, "máximo") {
		t.Fatalf("se esperaba rechazo por máximo de filas, se obtuvo filas=%+v errores=%+v", filas, errores)
	}
}

// Las columnas más allá de maxColumnasXLSX se descartan sin romper las filas válidas.
func TestLeerFilasXLSX_RecortaColumnas(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()
	hoja := f.GetSheetName(0)
	_ = f.SetSheetRow(hoja, "A1", &[]any{"categoria", "nombre", "precio"})
	_ = f.SetSheetRow(hoja, "A2", &[]any{"Bebidas", "Agua", 1500})
	_ = f.SetCellValue(hoja, "XFD2", "basura")

	filas, errFila := leerFilasXLSX(f, hoja)
	if errFila != nil {
		t.Fatalf("error inesperado: %+v", errFila)
	}
	if len(filas) != 2 || len(filas[1]) != maxColumnasXLSX || filas[1][1] != "Agua" {
		t.Fatalf("filas inesperadas: %+v", filas)
	}
}

func TestParsearXLSX_ZipBomb(t *testing.T) {
	limiteOriginal := limiteDescompresionXLSX
	limiteDescompresionXLSX = 1 << 20 // 1 MB, para no armar 50 MB en cada corrida
	t.Cleanup(func() { limiteDescompresionXLSX = limiteOriginal })

	// Más del límite descomprimido, pero muy chico comprimido.
	relleno := strings.Repeat(" ", int(limiteDescompresionXLSX)+1)
	hoja := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">` +
		relleno + `<sheetData/></worksheet>`
	contenido := reemplazarHoja(t, xlsxValido(t), hoja)

	filas, errores := parsearConTimeout(t, contenido)
	if len(filas) != 0 || len(errores) == 0 || !strings.Contains(errores[0].Motivo, "demasiado grande") {
		t.Fatalf("se esperaba rechazo por tamaño descomprimido, se obtuvo filas=%+v errores=%+v", filas, errores)
	}
}
