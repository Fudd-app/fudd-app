package carta

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
)

type Handlers struct {
	svc *Service
}

func NuevosHandlers(svc *Service) *Handlers { return &Handlers{svc: svc} }

func (h *Handlers) ListarCategorias(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	cats, err := h.svc.ListarCategorias(r.Context(), claims.TenantID)
	if err != nil {
		slog.ErrorContext(r.Context(), "error listando categorías", "err", err)
		jsonError(w, "error listando categorías", http.StatusInternalServerError)
		return
	}
	jsonOK(w, cats)
}

func (h *Handlers) CrearCategoria(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	var input CategoriaInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, "body inválido", http.StatusBadRequest)
		return
	}
	cat, err := h.svc.CrearCategoria(r.Context(), claims.TenantID, input)
	if err != nil {
		if errors.Is(err, ErrValidation) {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		slog.ErrorContext(r.Context(), "error en store", "err", err)
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(cat)
}

func (h *Handlers) EliminarCategoria(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	id := r.PathValue("id")
	if err := h.svc.EliminarCategoria(r.Context(), id, claims.TenantID); err != nil {
		if errors.Is(err, ErrNotFound) {
			jsonError(w, "categoría no encontrada", http.StatusNotFound)
			return
		}
		slog.ErrorContext(r.Context(), "error eliminando categoría", "err", err)
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) ListarArticulos(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	arts, err := h.svc.ListarArticulos(r.Context(), claims.TenantID)
	if err != nil {
		slog.ErrorContext(r.Context(), "error listando artículos", "err", err)
		jsonError(w, "error listando artículos", http.StatusInternalServerError)
		return
	}
	jsonOK(w, arts)
}

func (h *Handlers) CrearArticulo(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	var input ArticuloInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, "body inválido", http.StatusBadRequest)
		return
	}
	art, err := h.svc.CrearArticulo(r.Context(), claims.TenantID, input)
	if err != nil {
		if errors.Is(err, ErrValidation) {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		slog.ErrorContext(r.Context(), "error en store", "err", err)
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(art)
}

func (h *Handlers) ActualizarArticulo(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	id := r.PathValue("id")
	var u ArticuloUpdate
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		jsonError(w, "body inválido", http.StatusBadRequest)
		return
	}
	art, err := h.svc.ActualizarArticulo(r.Context(), id, claims.TenantID, u)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			jsonError(w, "artículo no encontrado", http.StatusNotFound)
			return
		}
		if errors.Is(err, ErrValidation) {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		slog.ErrorContext(r.Context(), "error actualizando artículo", "err", err)
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}
	jsonOK(w, art)
}

func (h *Handlers) AjustarPrecios(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	var input AjustePreciosInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, "body inválido", http.StatusBadRequest)
		return
	}

	resultado, err := h.svc.AjustarPrecios(r.Context(), claims.TenantID, input)
	if err != nil {
		if errors.Is(err, ErrValidation) {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		slog.ErrorContext(r.Context(), "error ajustando precios", "err", err)
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}
	jsonOK(w, resultado)
}

func (h *Handlers) EliminarArticulo(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	id := r.PathValue("id")
	if err := h.svc.EliminarArticulo(r.Context(), id, claims.TenantID); err != nil {
		if errors.Is(err, ErrNotFound) {
			jsonError(w, "artículo no encontrado", http.StatusNotFound)
			return
		}
		slog.ErrorContext(r.Context(), "error eliminando artículo", "err", err)
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) CartaPublica(w http.ResponseWriter, r *http.Request) {
	sucursalID := r.PathValue("sucursal_id")
	c, err := h.svc.ObtenerCartaPublica(r.Context(), sucursalID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			jsonError(w, "carta no disponible", http.StatusNotFound)
			return
		}
		slog.ErrorContext(r.Context(), "error obteniendo carta pública", "err", err)
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}
	jsonOK(w, c)
}

func (h *Handlers) ListarFranjasHorarias(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	franjas, err := h.svc.ListarFranjasHorarias(r.Context(), claims.TenantID)
	if err != nil {
		slog.ErrorContext(r.Context(), "error listando franjas horarias", "err", err)
		jsonError(w, "error listando franjas horarias", http.StatusInternalServerError)
		return
	}
	jsonOK(w, franjas)
}

func (h *Handlers) CrearFranjaHoraria(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	var input FranjaHorariaInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, "body inválido", http.StatusBadRequest)
		return
	}
	franja, err := h.svc.CrearFranjaHoraria(r.Context(), claims.TenantID, input)
	if err != nil {
		h.responderErrorFranja(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(franja)
}

func (h *Handlers) ActualizarFranjaHoraria(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	var input FranjaHorariaInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, "body inválido", http.StatusBadRequest)
		return
	}
	franja, err := h.svc.ActualizarFranjaHoraria(r.Context(), r.PathValue("id"), claims.TenantID, input)
	if err != nil {
		h.responderErrorFranja(w, r, err)
		return
	}
	jsonOK(w, franja)
}

func (h *Handlers) EliminarFranjaHoraria(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	if err := h.svc.EliminarFranjaHoraria(r.Context(), r.PathValue("id"), claims.TenantID); err != nil {
		h.responderErrorFranja(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) AsignarFranjaCategoria(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	var input AsignarFranjaInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, "body inválido", http.StatusBadRequest)
		return
	}
	categoria, err := h.svc.AsignarFranjaCategoria(r.Context(), r.PathValue("id"), claims.TenantID, input.FranjaHorariaID)
	if err != nil {
		h.responderErrorFranja(w, r, err)
		return
	}
	jsonOK(w, categoria)
}

func (h *Handlers) AsignarIconoCategoria(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	var input AsignarIconoInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, "body inválido", http.StatusBadRequest)
		return
	}
	categoria, err := h.svc.AsignarIconoCategoria(r.Context(), r.PathValue("id"), claims.TenantID, input.Icono)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			jsonError(w, "categoría no encontrada", http.StatusNotFound)
			return
		}
		if errors.Is(err, ErrValidation) {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		slog.ErrorContext(r.Context(), "error actualizando ícono de categoría", "err", err)
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}
	jsonOK(w, categoria)
}

func (h *Handlers) AsignarFranjaArticulo(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	var input AsignarFranjaInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, "body inválido", http.StatusBadRequest)
		return
	}
	articulo, err := h.svc.AsignarFranjaArticulo(r.Context(), r.PathValue("id"), claims.TenantID, input.FranjaHorariaID)
	if err != nil {
		h.responderErrorFranja(w, r, err)
		return
	}
	jsonOK(w, articulo)
}

func (h *Handlers) responderErrorFranja(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrFranjaDuplicada):
		jsonError(w, ErrFranjaDuplicada.Error(), http.StatusConflict)
	case errors.Is(err, ErrValidation):
		jsonError(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, ErrNotFound):
		jsonError(w, "franja, categoría o artículo no encontrado", http.StatusNotFound)
	default:
		slog.ErrorContext(r.Context(), "error gestionando franja horaria", "err", err)
		jsonError(w, "error interno", http.StatusInternalServerError)
	}
}

func (h *Handlers) ListarVariantes(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	articuloID := r.PathValue("id")
	variantes, err := h.svc.ListarVariantes(r.Context(), articuloID, claims.TenantID)
	if err != nil {
		slog.ErrorContext(r.Context(), "error listando variantes", "err", err)
		jsonError(w, "error listando variantes", http.StatusInternalServerError)
		return
	}
	jsonOK(w, variantes)
}

func (h *Handlers) CrearVariante(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	articuloID := r.PathValue("id")
	var input CrearVarianteInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, "body inválido", http.StatusBadRequest)
		return
	}
	v, err := h.svc.CrearVariante(r.Context(), articuloID, claims.TenantID, input)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			jsonError(w, "artículo no encontrado", http.StatusNotFound)
			return
		}
		if errors.Is(err, ErrValidation) {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		slog.ErrorContext(r.Context(), "error creando variante", "err", err)
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(v)
}

func (h *Handlers) ActualizarVariante(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	id := r.PathValue("id")
	var input ActualizarVarianteInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, "body inválido", http.StatusBadRequest)
		return
	}
	v, err := h.svc.ActualizarVariante(r.Context(), id, claims.TenantID, input)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			jsonError(w, "variante no encontrada", http.StatusNotFound)
			return
		}
		if errors.Is(err, ErrValidation) {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		slog.ErrorContext(r.Context(), "error actualizando variante", "err", err)
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}
	jsonOK(w, v)
}

func (h *Handlers) EliminarVariante(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	id := r.PathValue("id")
	if err := h.svc.EliminarVariante(r.Context(), id, claims.TenantID); err != nil {
		if errors.Is(err, ErrNotFound) {
			jsonError(w, "variante no encontrada", http.StatusNotFound)
			return
		}
		slog.ErrorContext(r.Context(), "error eliminando variante", "err", err)
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ImportarCarta recibe un archivo CSV o XLSX vía multipart/form-data (campo "archivo"),
// lo parsea y crea en la base de datos las categorías y artículos contenidos.
// Retorna un JSON con el reporte de artículos creados, omitidos y filas con error.
func (h *Handlers) ImportarCarta(w http.ResponseWriter, r *http.Request) {
	const maxTamano = 5 << 20 // 5 MB
	r.Body = http.MaxBytesReader(w, r.Body, maxTamano)

	if err := r.ParseMultipartForm(maxTamano); err != nil {
		jsonError(w, "el archivo excede el límite de 5 MB", http.StatusBadRequest)
		return
	}

	archivo, cabecera, err := r.FormFile("archivo")
	if err != nil {
		jsonError(w, "campo 'archivo' requerido", http.StatusBadRequest)
		return
	}
	defer archivo.Close()

	ext := strings.ToLower(filepath.Ext(cabecera.Filename))
	if ext != ".csv" && ext != ".xlsx" {
		jsonError(w, "formato no soportado: solo se aceptan archivos .csv y .xlsx", http.StatusBadRequest)
		return
	}

	contenido, err := io.ReadAll(archivo)
	if err != nil {
		jsonError(w, "error leyendo el archivo", http.StatusInternalServerError)
		return
	}

	var filas []FilaImportacion
	var erroresParseo []ErrorFila

	if ext == ".csv" {
		filas, erroresParseo = ParsearCSV(strings.NewReader(string(contenido)))
	} else {
		filas, erroresParseo = ParsearXLSX(bytes.NewReader(contenido))
	}

	// Si el parseo completo falló (archivo malformado), retornamos de inmediato.
	if len(filas) == 0 && len(erroresParseo) > 0 && erroresParseo[0].Fila <= 1 {
		jsonError(w, erroresParseo[0].Motivo, http.StatusBadRequest)
		return
	}
	// La carga masiva es todo-o-nada: si alguna fila es inválida, se informa
	// el reporte completo sin persistir las filas válidas del mismo archivo.
	if len(erroresParseo) > 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ResultadoImportacion{
			Omitidos: len(filas) + len(erroresParseo),
			Errores:  erroresParseo,
		})
		return
	}

	claims := auth.ClaimsFromContext(r.Context())
	resultado, err := h.svc.ImportarCarta(r.Context(), claims.TenantID, filas)
	if err != nil {
		slog.ErrorContext(r.Context(), "error importando carta", "err", err)
		jsonError(w, "error interno durante la importación", http.StatusInternalServerError)
		return
	}

	slog.InfoContext(r.Context(), "importación completada",
		"tenant_id", claims.TenantID,
		"creados", resultado.Creados,
		"omitidos", resultado.Omitidos,
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resultado)
}

func (h *Handlers) ActualizarDisponibilidad(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		jsonError(w, "id requerido", http.StatusBadRequest)
		return
	}

	var input ActualizarDisponibilidadInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, "cuerpo de solicitud inválido", http.StatusBadRequest)
		return
	}

	claims := auth.ClaimsFromContext(r.Context())
	a, err := h.svc.ActualizarDisponibilidad(r.Context(), id, claims.TenantID, input.Disponible)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			jsonError(w, "artículo no encontrado", http.StatusNotFound)
			return
		}
		if errors.Is(err, ErrValidation) {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		jsonError(w, "error actualizando disponibilidad", http.StatusInternalServerError)
		return
	}
	jsonOK(w, a)
}

func (h *Handlers) ReponerTodos(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	res, err := h.svc.ReponerTodos(r.Context(), claims.TenantID)
	if err != nil {
		jsonError(w, "error reponiendo stock", http.StatusInternalServerError)
		return
	}
	jsonOK(w, res)
}

func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
