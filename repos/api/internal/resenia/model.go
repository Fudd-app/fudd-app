package resenia

import (
	"errors"
	"time"
)

// Errores del dominio
var (
	ErrNotFound         = errors.New("recurso no encontrado")
	ErrValidation       = errors.New("datos inválidos")
	ErrMesaInactiva     = errors.New("la mesa no está activa")
	ErrSinPedidos       = errors.New("no hay pedidos en la cuenta actual")
	ErrResenaDuplicada  = errors.New("ya existe una reseña para esta cuenta")
	ErrSucursalInvalida = errors.New("sucursal no encontrada")
)

type Resena struct {
	ID              string    `json:"id"`
	MesaID          string    `json:"mesa_id"`
	SucursalID      string    `json:"sucursal_id"`
	TenantID        string    `json:"tenant_id"`
	CuentaVersion   int       `json:"cuenta_version"`
	Estrellas       int       `json:"estrellas"`
	Comentario      *string   `json:"comentario"`
	GoogleCliqueado bool      `json:"google_cliqueado"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type ResenaItem struct {
	ID              string    `json:"id"`
	MesaID          string    `json:"mesa_id"`
	MesaNumero      int       `json:"mesa_numero"`
	SucursalID      string    `json:"sucursal_id"`
	SucursalNombre  string    `json:"sucursal_nombre"`
	CuentaVersion   int       `json:"cuenta_version"`
	Estrellas       int       `json:"estrellas"`
	Comentario      *string   `json:"comentario"`
	GoogleCliqueado bool      `json:"google_cliqueado"`
	CreatedAt       time.Time `json:"created_at"`
}

type CrearResenaInput struct {
	Estrellas  int     `json:"estrellas"`
	Comentario *string `json:"comentario,omitempty"`
}

type ResenaNotificacionPayload struct {
	ID         string    `json:"id"`
	Estrellas  int       `json:"estrellas"`
	MesaNumero int       `json:"mesa_numero"`
	SucursalID string    `json:"sucursal_id"`
	Comentario *string   `json:"comentario"`
	CreatedAt  time.Time `json:"created_at"`
}

type ResumenResenas struct {
	Promedio     float64      `json:"promedio"`
	Total        int          `json:"total"`
	CSAT         float64      `json:"csat"`
	Distribucion map[int]int  `json:"distribucion"`
	ClicsGoogle  int          `json:"clics_google"`
	Resenas      []ResenaItem `json:"resenas"`
}

type MesaContexto struct {
	MesaID        string
	MesaNumero    int
	SucursalID    string
	TenantID      string
	CuentaVersion int
	Estado        string
}
