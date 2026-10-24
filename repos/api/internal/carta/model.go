package carta

import "errors"

// Sentinel errors
var (
	ErrNotFound        = errors.New("recurso no encontrado")
	ErrValidation      = errors.New("datos inválidos")
	ErrFranjaDuplicada = errors.New("ya existe una franja horaria con ese nombre")
)

type Categoria struct {
	ID              string  `json:"id"`
	TenantID        string  `json:"tenant_id,omitempty"`
	Nombre          string  `json:"nombre"`
	Orden           int     `json:"orden"`
	Icono           *string `json:"icono,omitempty"`
	FranjaHorariaID *string `json:"franja_horaria_id,omitempty"`
}

type Articulo struct {
	ID                 string         `json:"id"`
	TenantID           string         `json:"tenant_id,omitempty"`
	CategoriaID        string         `json:"categoria_id"`
	Nombre             string         `json:"nombre"`
	Descripcion        string         `json:"descripcion,omitempty"`
	Precio             float64        `json:"precio"`
	FotoURL            string         `json:"foto_url,omitempty"`
	Activo             bool           `json:"activo"`
	Disponible         bool           `json:"disponible"`
	ReponerDiariamente bool           `json:"reponer_diariamente"`
	FranjaHorariaID    *string        `json:"franja_horaria_id,omitempty"`
	Variantes          []Variante     `json:"variantes,omitempty"`
	FranjaEfectiva     *FranjaHoraria `json:"-"`
}

// FranjaHoraria define una ventana diaria. Si el horario final es menor que el
// inicial, la franja cruza medianoche (por ejemplo, 20:00 a 02:00).
type FranjaHoraria struct {
	ID         string `json:"id"`
	TenantID   string `json:"tenant_id,omitempty"`
	Nombre     string `json:"nombre"`
	HoraInicio string `json:"hora_inicio"`
	HoraFin    string `json:"hora_fin"`
}

type FranjaHorariaInput struct {
	Nombre     string `json:"nombre"`
	HoraInicio string `json:"hora_inicio"`
	HoraFin    string `json:"hora_fin"`
}

type AsignarFranjaInput struct {
	FranjaHorariaID *string `json:"franja_horaria_id"`
}

type AsignarIconoInput struct {
	Icono *string `json:"icono"`
}

type Variante struct {
	ID              string  `json:"id"`
	ArticuloID      string  `json:"articulo_id"`
	Nombre          string  `json:"nombre"`
	PrecioAdicional float64 `json:"precio_adicional"`
	Grupo           *string `json:"grupo,omitempty"`
	SeleccionUnica  bool    `json:"seleccion_unica"`
	Orden           int     `json:"orden"`
}

type CrearVarianteInput struct {
	Nombre          string  `json:"nombre"`
	PrecioAdicional float64 `json:"precio_adicional"`
	Grupo           *string `json:"grupo,omitempty"`
	SeleccionUnica  bool    `json:"seleccion_unica"`
	Orden           int     `json:"orden"`
}

type ActualizarVarianteInput struct {
	Nombre          *string  `json:"nombre,omitempty"`
	PrecioAdicional *float64 `json:"precio_adicional,omitempty"`
	Grupo           *string  `json:"grupo,omitempty"`
	SeleccionUnica  *bool    `json:"seleccion_unica,omitempty"`
	Orden           *int     `json:"orden,omitempty"`
}

type CategoriaInput struct {
	Nombre string `json:"nombre"`
	Orden  int    `json:"orden"`
}

type ArticuloInput struct {
	CategoriaID string  `json:"categoria_id"`
	Nombre      string  `json:"nombre"`
	Descripcion string  `json:"descripcion"`
	Precio      float64 `json:"precio"`
	FotoURL     string  `json:"foto_url"`
}

type ArticuloUpdate struct {
	CategoriaID        *string  `json:"categoria_id"`
	Nombre             *string  `json:"nombre"`
	Descripcion        *string  `json:"descripcion"`
	Precio             *float64 `json:"precio"`
	FotoURL            *string  `json:"foto_url"`
	Activo             *bool    `json:"activo"`
	Disponible         *bool    `json:"disponible"`
	ReponerDiariamente *bool    `json:"reponer_diariamente"`
}

type ActualizarDisponibilidadInput struct {
	Disponible bool `json:"disponible"`
}

type ReponerTodosResultado struct {
	Repuestos int `json:"repuestos"`
}

const (
	RedondeoNinguno = "ninguno"
	Redondeo10      = "10"
	Redondeo100     = "100"
)

type AjustePreciosInput struct {
	CategoriaID string  `json:"categoria_id,omitempty"`
	Porcentaje  float64 `json:"porcentaje"`
	Redondeo    string  `json:"redondeo"`
}

type AjustePreciosResultado struct {
	Actualizados int `json:"actualizados"`
}

type CartaPublica struct {
	Categorias []CategoriaConArticulos `json:"categorias"`
}

type CategoriaConArticulos struct {
	Categoria
	Articulos       []Articulo `json:"articulos"`
	Disponible      bool       `json:"disponible"`
	DisponibleDesde string     `json:"disponible_desde,omitempty"`
}
