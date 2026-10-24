package carta

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// FilaImportacion representa una fila ya parseada y normalizada del archivo.
type FilaImportacion struct {
	Numero      int
	Categoria   string
	Nombre      string
	Precio      float64
	Descripcion string
	Disponible  bool
}

// ErrorFila describe un problema encontrado en una fila específica.
type ErrorFila struct {
	Fila   int    `json:"fila"`
	Motivo string `json:"motivo"`
}

// ResultadoImportacion es el reporte JSON que devuelve el endpoint.
type ResultadoImportacion struct {
	Creados  int         `json:"creados"`
	Omitidos int         `json:"omitidos"`
	Errores  []ErrorFila `json:"errores"`
}

// Columnas esperadas (case-insensitive, se aceptan variantes con tildes o sin).
var columnasEsperadas = []string{"categoria", "nombre", "precio", "descripcion", "disponible"}

// ParsearCSV lee un Reader con contenido CSV y devuelve las filas válidas
// junto con los errores encontrados por fila.
// La primera fila debe ser el encabezado.
func ParsearCSV(r io.Reader) ([]FilaImportacion, []ErrorFila) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1 // Acepta número variable de columnas

	registros, err := reader.ReadAll()
	if err != nil {
		return nil, []ErrorFila{{Fila: 0, Motivo: "archivo CSV malformado: " + err.Error()}}
	}
	if len(registros) == 0 {
		return nil, []ErrorFila{{Fila: 0, Motivo: "el archivo está vacío"}}
	}

	return parsearRegistros(registros)
}

// limiteDescompresionXLSX acota el tamaño descomprimido del XLSX. El handler
// limita la subida a 5 MB comprimidos, pero un zip bomb de ese tamaño puede
// expandirse a varios GB (el default de excelize es 16 GB).
// Es variable para que los tests usen un límite chico.
var limiteDescompresionXLSX int64 = 50 << 20 // 50 MB

const (
	// maxFilasXLSX acota las filas recorridas. excelize rellena cada fila hasta
	// su última celda: una celda en la columna XFD cuesta ~1 MB por fila aunque
	// el archivo pese pocos KB, y un número de fila enorme genera millones de
	// filas vacías. Una carta real no se acerca a este máximo.
	maxFilasXLSX = 2000
	// maxColumnasXLSX es cuántas columnas por fila se conservan; el resto se
	// descarta para no retener filas de 16384 celdas.
	maxColumnasXLSX = 26 // A–Z
)

// ParsearXLSX lee un Reader con contenido XLSX y devuelve las filas válidas.
// Lee la primera hoja del libro.
func ParsearXLSX(r io.Reader) ([]FilaImportacion, []ErrorFila) {
	f, err := excelize.OpenReader(r, excelize.Options{UnzipSizeLimit: limiteDescompresionXLSX})
	if err != nil {
		// excelize no exporta este error, solo se distingue por el mensaje.
		if strings.Contains(err.Error(), "unzip size exceeds") {
			return nil, []ErrorFila{{Fila: 0, Motivo: fmt.Sprintf(
				"el archivo Excel es demasiado grande: descomprimido supera los %d MB", limiteDescompresionXLSX>>20)}}
		}
		return nil, []ErrorFila{{Fila: 0, Motivo: "archivo Excel malformado o dañado"}}
	}
	defer f.Close()

	hojas := f.GetSheetList()
	if len(hojas) == 0 {
		return nil, []ErrorFila{{Fila: 0, Motivo: "el archivo Excel no tiene hojas"}}
	}

	filas, errFila := leerFilasXLSX(f, hojas[0])
	if errFila != nil {
		return nil, []ErrorFila{*errFila}
	}
	if len(filas) == 0 {
		return nil, []ErrorFila{{Fila: 0, Motivo: "la hoja de cálculo está vacía"}}
	}

	return parsearRegistros(filas)
}

// leerFilasXLSX recorre la hoja fila por fila aplicando maxFilasXLSX y
// maxColumnasXLSX. Reemplaza a f.GetRows, que no tiene ninguno de los dos.
func leerFilasXLSX(f *excelize.File, hoja string) ([][]string, *ErrorFila) {
	rows, err := f.Rows(hoja)
	if err != nil {
		return nil, &ErrorFila{Fila: 0, Motivo: "no se pudieron leer las filas del Excel"}
	}
	defer rows.Close()

	var filas [][]string
	for n := 1; rows.Next(); n++ {
		if n > maxFilasXLSX {
			return nil, &ErrorFila{Fila: 0, Motivo: fmt.Sprintf(
				"el archivo Excel supera el máximo de %d filas", maxFilasXLSX)}
		}
		columnas, err := rows.Columns()
		if err != nil {
			return nil, &ErrorFila{Fila: 0, Motivo: "no se pudieron leer las filas del Excel"}
		}
		if len(columnas) > maxColumnasXLSX {
			columnas = columnas[:maxColumnasXLSX]
		}
		// Copia para soltar el arreglo original, que puede tener 16384 celdas.
		filas = append(filas, append([]string(nil), columnas...))
	}
	if err := rows.Error(); err != nil {
		return nil, &ErrorFila{Fila: 0, Motivo: "no se pudieron leer las filas del Excel"}
	}

	return filas, nil
}

// parsearRegistros convierte la matriz de strings (cabecera + filas) en FilaImportacion.
// Escanea automáticamente para encontrar la fila de encabezados (la primera que contenga
// "categoria" y "nombre"), tolerando filas de título o instrucciones al inicio.
// Retorna las filas válidas y los errores encontrados por fila.
func parsearRegistros(registros [][]string) ([]FilaImportacion, []ErrorFila) {
	// ── Buscar la fila de encabezados ──────────────────────────────────────
	headerRowIdx := -1
	for i, fila := range registros {
		colIdx := mapearColumnas(fila)
		if colIdx["categoria"] >= 0 && colIdx["nombre"] >= 0 {
			headerRowIdx = i
			break
		}
	}

	if headerRowIdx < 0 {
		return nil, []ErrorFila{{
			Fila:   1,
			Motivo: "no se encontró la fila de encabezados. Asegurate de que el archivo tenga las columnas: categoria, nombre, precio",
		}}
	}

	// Trabajar solo desde la fila de encabezados en adelante
	registros = registros[headerRowIdx:]

	// Mapear columnas del encabezado (case-insensitive)
	encabezado := registros[0]
	colIdx := mapearColumnas(encabezado)

	idxCat, okCat := colIdx["categoria"]
	idxNom, okNom := colIdx["nombre"]
	idxPre, okPre := colIdx["precio"]
	idxDes := colIdx["descripcion"] // Opcional
	idxDis := colIdx["disponible"]  // Opcional

	if !okCat || !okNom || !okPre {
		faltantes := []string{}
		if !okCat {
			faltantes = append(faltantes, "categoria")
		}
		if !okNom {
			faltantes = append(faltantes, "nombre")
		}
		if !okPre {
			faltantes = append(faltantes, "precio")
		}
		return nil, []ErrorFila{{
			Fila:   1,
			Motivo: fmt.Sprintf("encabezado inválido: faltan columnas obligatorias: %s", strings.Join(faltantes, ", ")),
		}}
	}

	var filas []FilaImportacion
	var errores []ErrorFila

	for i, fila := range registros[1:] {
		numFila := i + 2 // Fila 2 en adelante (la 1 es el encabezado)

		// Saltar filas completamente vacías
		if estaVacia(fila) {
			continue
		}

		// Leer categoria
		categoria := obtenerCelda(fila, idxCat)
		if categoria == "" {
			errores = append(errores, ErrorFila{Fila: numFila, Motivo: "columna 'categoria' vacía"})
			continue
		}

		// Leer nombre
		nombre := obtenerCelda(fila, idxNom)
		if nombre == "" {
			errores = append(errores, ErrorFila{Fila: numFila, Motivo: "columna 'nombre' vacía"})
			continue
		}

		// Leer precio
		precioStr := strings.ReplaceAll(obtenerCelda(fila, idxPre), ",", ".")
		precio, err := strconv.ParseFloat(precioStr, 64)
		if err != nil || precio < 0 {
			errores = append(errores, ErrorFila{
				Fila:   numFila,
				Motivo: fmt.Sprintf("precio inválido: %q (debe ser un número >= 0)", obtenerCelda(fila, idxPre)),
			})
			continue
		}

		// Leer descripción (opcional)
		descripcion := ""
		if idxDes >= 0 {
			descripcion = obtenerCelda(fila, idxDes)
		}

		// Leer disponible (opcional, default: true)
		disponible := true
		if idxDis >= 0 {
			v := strings.ToLower(strings.TrimSpace(obtenerCelda(fila, idxDis)))
			if v == "no" || v == "false" || v == "0" {
				disponible = false
			}
		}

		filas = append(filas, FilaImportacion{
			Numero:      numFila,
			Categoria:   categoria,
			Nombre:      nombre,
			Precio:      precio,
			Descripcion: descripcion,
			Disponible:  disponible,
		})
	}

	return filas, errores
}

// mapearColumnas devuelve un mapa de nombre_normalizado → índice de columna.
// Índice -1 si la columna no existe.
func mapearColumnas(encabezado []string) map[string]int {
	m := map[string]int{
		"categoria":   -1,
		"nombre":      -1,
		"precio":      -1,
		"descripcion": -1,
		"disponible":  -1,
	}
	for i, h := range encabezado {
		normalizado := normalizarEncabezado(h)
		if _, existe := m[normalizado]; existe {
			m[normalizado] = i
		}
	}
	return m
}

// normalizarEncabezado limpia tildes, espacios y pasa a minúsculas.
func normalizarEncabezado(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	replacer := strings.NewReplacer(
		"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u",
		"à", "a", "è", "e", "ì", "i", "ò", "o", "ù", "u",
		"ä", "a", "ë", "e", "ï", "i", "ö", "o", "ü", "u",
		"ñ", "n",
	)
	return replacer.Replace(s)
}

// obtenerCelda retorna el valor de la celda o "" si está fuera de rango.
func obtenerCelda(fila []string, idx int) string {
	if idx < 0 || idx >= len(fila) {
		return ""
	}
	return strings.TrimSpace(fila[idx])
}

// estaVacia devuelve true si todos los campos de la fila son strings vacíos.
func estaVacia(fila []string) bool {
	for _, v := range fila {
		if strings.TrimSpace(v) != "" {
			return false
		}
	}
	return true
}

// ImportarCarta ejecuta la importación en la base de datos.
// Es llamado desde el Service para separar la lógica de parseo de la de persistencia.
func (svc *Service) ImportarCarta(ctx context.Context, tenantID string, filas []FilaImportacion) (ResultadoImportacion, error) {
	return svc.store.ImportarCartaAtomica(ctx, tenantID, filas)
}
