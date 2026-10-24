package carta

import (
	"context"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/notificacion"
)

type Service struct {
	store Store
}

const maxDescripcionArticuloRunes = 160

func NuevoService(s Store) *Service { return &Service{store: s} }

func (svc *Service) ListarCategorias(ctx context.Context, tenantID string) ([]Categoria, error) {
	return svc.store.ListarCategorias(ctx, tenantID)
}

func (svc *Service) CrearCategoria(ctx context.Context, tenantID string, input CategoriaInput) (*Categoria, error) {
	if input.Nombre == "" {
		return nil, fmt.Errorf("nombre requerido: %w", ErrValidation)
	}
	return svc.store.CrearCategoria(ctx, tenantID, input)
}

func (svc *Service) EliminarCategoria(ctx context.Context, id, tenantID string) error {
	return svc.store.EliminarCategoria(ctx, id, tenantID)
}

func (svc *Service) ListarArticulos(ctx context.Context, tenantID string) ([]Articulo, error) {
	return svc.store.ListarArticulos(ctx, tenantID)
}

func (svc *Service) CrearArticulo(ctx context.Context, tenantID string, input ArticuloInput) (*Articulo, error) {
	if input.Nombre == "" {
		return nil, fmt.Errorf("nombre requerido: %w", ErrValidation)
	}
	if input.Precio < 0 {
		return nil, fmt.Errorf("precio no puede ser negativo: %w", ErrValidation)
	}
	if input.CategoriaID == "" {
		return nil, fmt.Errorf("categoria_id requerido: %w", ErrValidation)
	}
	input.Nombre = strings.TrimSpace(input.Nombre)
	input.Descripcion = strings.TrimSpace(input.Descripcion)
	if err := validarDescripcionArticulo(input.Descripcion); err != nil {
		return nil, err
	}
	foto, err := normalizarFotoURL(input.FotoURL)
	if err != nil {
		return nil, err
	}
	input.FotoURL = foto
	return svc.store.CrearArticulo(ctx, tenantID, input)
}

func (svc *Service) ActualizarArticulo(ctx context.Context, id, tenantID string, u ArticuloUpdate) (*Articulo, error) {
	if u.Precio != nil && *u.Precio < 0 {
		return nil, fmt.Errorf("precio no puede ser negativo: %w", ErrValidation)
	}
	if u.FotoURL != nil {
		foto, err := normalizarFotoURL(*u.FotoURL)
		if err != nil {
			return nil, err
		}
		u.FotoURL = &foto
	}
	if u.Nombre != nil {
		nombre := strings.TrimSpace(*u.Nombre)
		if nombre == "" {
			return nil, fmt.Errorf("nombre requerido: %w", ErrValidation)
		}
		u.Nombre = &nombre
	}
	if u.Descripcion != nil {
		descripcion := strings.TrimSpace(*u.Descripcion)
		if err := validarDescripcionArticulo(descripcion); err != nil {
			return nil, err
		}
		u.Descripcion = &descripcion
	}
	return svc.store.ActualizarArticulo(ctx, id, tenantID, u)
}

func validarDescripcionArticulo(descripcion string) error {
	if utf8.RuneCountInString(descripcion) > maxDescripcionArticuloRunes {
		return fmt.Errorf("la descripción no puede superar los %d caracteres: %w", maxDescripcionArticuloRunes, ErrValidation)
	}
	return nil
}

const maxFotoURLBytes = 380_000

func normalizarFotoURL(valor string) (string, error) {
	valor = strings.TrimSpace(valor)
	if valor == "" {
		return "", nil
	}
	if len(valor) > maxFotoURLBytes {
		return "", fmt.Errorf("la imagen supera el tamaño permitido: %w", ErrValidation)
	}
	valido := strings.HasPrefix(valor, "data:image/jpeg;base64,") ||
		strings.HasPrefix(valor, "data:image/png;base64,") ||
		strings.HasPrefix(valor, "data:image/webp;base64,") ||
		strings.HasPrefix(valor, "https://") || strings.HasPrefix(valor, "http://")
	if !valido {
		return "", fmt.Errorf("foto_url debe ser una imagen PNG/JPG/WebP válida: %w", ErrValidation)
	}
	return valor, nil
}

var iconosCategoriaPermitidos = map[string]struct{}{
	"restaurant": {}, "lunch_dining": {}, "ramen_dining": {}, "local_pizza": {},
	"breakfast_dining": {}, "bakery_dining": {}, "tapas": {}, "soup_kitchen": {},
	"set_meal": {}, "cake": {}, "icecream": {}, "local_cafe": {},
	"local_bar": {}, "liquor": {}, "emoji_food_beverage": {},
}

func (svc *Service) AsignarIconoCategoria(ctx context.Context, id, tenantID string, icono *string) (*Categoria, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("id requerido: %w", ErrValidation)
	}
	if icono == nil || strings.TrimSpace(*icono) == "" {
		return svc.store.AsignarIconoCategoria(ctx, id, tenantID, nil)
	}
	normalizado := strings.TrimSpace(*icono)
	if _, ok := iconosCategoriaPermitidos[normalizado]; !ok {
		return nil, fmt.Errorf("icono de categoría no permitido: %w", ErrValidation)
	}
	return svc.store.AsignarIconoCategoria(ctx, id, tenantID, &normalizado)
}

func (svc *Service) ActualizarDisponibilidad(ctx context.Context, id, tenantID string, disponible bool) (*Articulo, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("id requerido: %w", ErrValidation)
	}
	art, err := svc.store.ActualizarDisponibilidad(ctx, id, tenantID, disponible)
	if err != nil {
		return nil, err
	}
	// Notificar en tiempo real por SSE
	notificacion.Instancia.Publicar(fmt.Sprintf("tenant:%s:carta", tenantID), "articulo_disponibilidad_cambiada", art)
	return art, nil
}

func (svc *Service) ReponerTodos(ctx context.Context, tenantID string) (*ReponerTodosResultado, error) {
	cant, err := svc.store.ReponerTodos(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	res := &ReponerTodosResultado{Repuestos: cant}
	notificacion.Instancia.Publicar(fmt.Sprintf("tenant:%s:carta", tenantID), "carta_repuesta", res)
	return res, nil
}

func (svc *Service) AjustarPrecios(ctx context.Context, tenantID string, input AjustePreciosInput) (*AjustePreciosResultado, error) {
	input.CategoriaID = strings.TrimSpace(input.CategoriaID)
	input.Redondeo = strings.TrimSpace(input.Redondeo)
	if input.Redondeo == "" {
		input.Redondeo = RedondeoNinguno
	}

	if math.IsNaN(input.Porcentaje) || math.IsInf(input.Porcentaje, 0) || input.Porcentaje == 0 {
		return nil, fmt.Errorf("el porcentaje debe ser distinto de cero: %w", ErrValidation)
	}
	if input.Porcentaje < -100 || input.Porcentaje > 1000 {
		return nil, fmt.Errorf("el porcentaje debe estar entre -100 y 1000: %w", ErrValidation)
	}
	if input.Redondeo != RedondeoNinguno && input.Redondeo != Redondeo10 && input.Redondeo != Redondeo100 {
		return nil, fmt.Errorf("opción de redondeo inválida: %w", ErrValidation)
	}

	resultado, err := svc.store.AjustarPrecios(ctx, tenantID, input)
	if err != nil {
		return nil, err
	}
	if resultado.Actualizados == 0 {
		return nil, fmt.Errorf("no hay productos para actualizar: %w", ErrValidation)
	}
	return resultado, nil
}

func (svc *Service) EliminarArticulo(ctx context.Context, id, tenantID string) error {
	return svc.store.EliminarArticulo(ctx, id, tenantID)
}

func (svc *Service) ObtenerCartaPublica(ctx context.Context, sucursalID string) (*CartaPublica, error) {
	carta, err := svc.store.ObtenerCartaPublica(ctx, sucursalID)
	if err != nil {
		return nil, err
	}

	zona := strings.TrimSpace(os.Getenv("APP_TIMEZONE"))
	if zona == "" {
		zona = "America/Argentina/Buenos_Aires"
	}
	ubicacion, err := time.LoadLocation(zona)
	if err != nil {
		return nil, fmt.Errorf("zona horaria %q inválida: %w", zona, err)
	}
	return ResolverCartaPorHora(carta, time.Now().In(ubicacion)), nil
}

// ResolverCartaPorHora filtra los artículos fuera de su franja efectiva. Los
// artículos sin franja permanecen disponibles durante todo el día.
func ResolverCartaPorHora(carta *CartaPublica, ahora time.Time) *CartaPublica {
	if carta == nil {
		return &CartaPublica{Categorias: []CategoriaConArticulos{}}
	}

	resultado := &CartaPublica{Categorias: make([]CategoriaConArticulos, 0, len(carta.Categorias))}
	minutosAhora := ahora.Hour()*60 + ahora.Minute()

	for _, categoria := range carta.Categorias {
		originales := categoria.Articulos
		categoria.Articulos = make([]Articulo, 0, len(originales))
		proximosInicios := make([]int, 0, len(originales))

		for _, articulo := range originales {
			if articulo.FranjaEfectiva == nil || franjaActiva(*articulo.FranjaEfectiva, minutosAhora) {
				categoria.Articulos = append(categoria.Articulos, articulo)
				continue
			}
			if inicio, ok := minutosDelDia(articulo.FranjaEfectiva.HoraInicio); ok {
				delta := inicio - minutosAhora
				if delta <= 0 {
					delta += 24 * 60
				}
				proximosInicios = append(proximosInicios, delta)
			}
		}

		categoria.Disponible = len(categoria.Articulos) > 0
		categoria.DisponibleDesde = ""
		if !categoria.Disponible && len(proximosInicios) > 0 {
			sort.Ints(proximosInicios)
			proximo := (minutosAhora + proximosInicios[0]) % (24 * 60)
			categoria.DisponibleDesde = fmt.Sprintf("%02d:%02d", proximo/60, proximo%60)
		}
		resultado.Categorias = append(resultado.Categorias, categoria)
	}

	return resultado
}

func franjaActiva(franja FranjaHoraria, minutosAhora int) bool {
	inicio, inicioOK := minutosDelDia(franja.HoraInicio)
	fin, finOK := minutosDelDia(franja.HoraFin)
	if !inicioOK || !finOK || inicio == fin {
		return false
	}
	if inicio < fin {
		return minutosAhora >= inicio && minutosAhora < fin
	}
	return minutosAhora >= inicio || minutosAhora < fin
}

func minutosDelDia(valor string) (int, bool) {
	hora, err := time.Parse("15:04", valor)
	if err != nil {
		return 0, false
	}
	return hora.Hour()*60 + hora.Minute(), true
}

func (svc *Service) ListarFranjasHorarias(ctx context.Context, tenantID string) ([]FranjaHoraria, error) {
	return svc.store.ListarFranjasHorarias(ctx, tenantID)
}

func (svc *Service) CrearFranjaHoraria(ctx context.Context, tenantID string, input FranjaHorariaInput) (*FranjaHoraria, error) {
	input = normalizarFranja(input)
	if err := validarFranja(input); err != nil {
		return nil, err
	}
	return svc.store.CrearFranjaHoraria(ctx, tenantID, input)
}

func (svc *Service) ActualizarFranjaHoraria(ctx context.Context, id, tenantID string, input FranjaHorariaInput) (*FranjaHoraria, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("id requerido: %w", ErrValidation)
	}
	input = normalizarFranja(input)
	if err := validarFranja(input); err != nil {
		return nil, err
	}
	return svc.store.ActualizarFranjaHoraria(ctx, id, tenantID, input)
}

func (svc *Service) EliminarFranjaHoraria(ctx context.Context, id, tenantID string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("id requerido: %w", ErrValidation)
	}
	return svc.store.EliminarFranjaHoraria(ctx, id, tenantID)
}

func (svc *Service) AsignarFranjaCategoria(ctx context.Context, id, tenantID string, franjaID *string) (*Categoria, error) {
	return svc.store.AsignarFranjaCategoria(ctx, id, tenantID, normalizarFranjaID(franjaID))
}

func (svc *Service) AsignarFranjaArticulo(ctx context.Context, id, tenantID string, franjaID *string) (*Articulo, error) {
	return svc.store.AsignarFranjaArticulo(ctx, id, tenantID, normalizarFranjaID(franjaID))
}

func normalizarFranja(input FranjaHorariaInput) FranjaHorariaInput {
	input.Nombre = strings.TrimSpace(input.Nombre)
	input.HoraInicio = strings.TrimSpace(input.HoraInicio)
	input.HoraFin = strings.TrimSpace(input.HoraFin)
	return input
}

func validarFranja(input FranjaHorariaInput) error {
	if input.Nombre == "" {
		return fmt.Errorf("nombre requerido: %w", ErrValidation)
	}
	if _, err := time.Parse("15:04", input.HoraInicio); err != nil {
		return fmt.Errorf("hora_inicio debe tener formato HH:MM: %w", ErrValidation)
	}
	if _, err := time.Parse("15:04", input.HoraFin); err != nil {
		return fmt.Errorf("hora_fin debe tener formato HH:MM: %w", ErrValidation)
	}
	if input.HoraInicio == input.HoraFin {
		return fmt.Errorf("hora_inicio y hora_fin deben ser distintas: %w", ErrValidation)
	}
	return nil
}

func normalizarFranjaID(franjaID *string) *string {
	if franjaID == nil {
		return nil
	}
	valor := strings.TrimSpace(*franjaID)
	if valor == "" {
		return nil
	}
	return &valor
}

func (svc *Service) ListarVariantes(ctx context.Context, articuloID, tenantID string) ([]Variante, error) {
	if articuloID == "" {
		return nil, fmt.Errorf("articulo_id requerido: %w", ErrValidation)
	}
	return svc.store.ListarVariantes(ctx, articuloID, tenantID)
}

func (svc *Service) CrearVariante(ctx context.Context, articuloID, tenantID string, input CrearVarianteInput) (*Variante, error) {
	if articuloID == "" {
		return nil, fmt.Errorf("articulo_id requerido: %w", ErrValidation)
	}
	if input.Nombre == "" {
		return nil, fmt.Errorf("nombre requerido: %w", ErrValidation)
	}
	if input.PrecioAdicional < 0 {
		return nil, fmt.Errorf("precio_adicional no puede ser negativo: %w", ErrValidation)
	}
	return svc.store.CrearVariante(ctx, articuloID, tenantID, input)
}

func (svc *Service) ActualizarVariante(ctx context.Context, id, tenantID string, input ActualizarVarianteInput) (*Variante, error) {
	if id == "" {
		return nil, fmt.Errorf("id requerido: %w", ErrValidation)
	}
	if input.PrecioAdicional != nil && *input.PrecioAdicional < 0 {
		return nil, fmt.Errorf("precio_adicional no puede ser negativo: %w", ErrValidation)
	}
	return svc.store.ActualizarVariante(ctx, id, tenantID, input)
}

func (svc *Service) EliminarVariante(ctx context.Context, id, tenantID string) error {
	if id == "" {
		return fmt.Errorf("id requerido: %w", ErrValidation)
	}
	return svc.store.EliminarVariante(ctx, id, tenantID)
}
