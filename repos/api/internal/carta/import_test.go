package carta

import (
	"archive/zip"
	"bytes"
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

// Una hoja con un número de fila desmedido debe rechazarse sin colgar el
// parseo (familia de GO-2026-6685; este input no reproduce ese CVE exacto).
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

func TestParsearXLSX_ZipBomb(t *testing.T) {
	// Más de limiteDescompresionXLSX descomprimido, pero muy chico comprimido.
	relleno := strings.Repeat(" ", limiteDescompresionXLSX+1)
	hoja := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">` +
		relleno + `<sheetData/></worksheet>`
	contenido := reemplazarHoja(t, xlsxValido(t), hoja)
	if len(contenido) > 5<<20 {
		t.Fatalf("el archivo de prueba debería pasar el límite de subida de 5 MB, pesa %d bytes", len(contenido))
	}

	filas, errores := parsearConTimeout(t, contenido)
	if len(filas) != 0 || len(errores) == 0 || !strings.Contains(errores[0].Motivo, "malformado") {
		t.Fatalf("se esperaba rechazo por tamaño descomprimido, se obtuvo filas=%+v errores=%+v", filas, errores)
	}
}
