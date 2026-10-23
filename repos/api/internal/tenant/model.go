package tenant

import (
	"errors"
	"time"
)

var (
	ErrNotFound           = errors.New("tenant no encontrado")
	ErrValidation         = errors.New("error de validación")
	ErrSlugConflict       = errors.New("nombre de url ya utilizado")
	ErrEmailAdminConflict = errors.New("correo de acceso ya registrado")
	ErrPlanRequired       = errors.New("función exclusiva del plan Pro")
)

type Tenant struct {
	ID                  string         `json:"id"`
	Nombre              string         `json:"nombre"`
	NombreFantasia      string         `json:"nombre_fantasia,omitempty"`
	Rubro               string         `json:"rubro"`
	Descripcion         *string        `json:"descripcion,omitempty"`
	EmailContacto       *string        `json:"email_contacto,omitempty"`
	Whatsapp            *string        `json:"whatsapp,omitempty"`
	LogoURL             *string        `json:"logo_url,omitempty"`
	ColorPrimario       *string        `json:"color_primario,omitempty"`
	EstiloVisual        *string        `json:"estilo_visual,omitempty"`
	DatosFiscales       map[string]any `json:"datos_fiscales,omitempty"`
	GoogleReviewURL     *string        `json:"google_review_url,omitempty"`
	MPAccessToken       *string        `json:"mp_access_token,omitempty"`
	MPPublicKey         *string        `json:"mp_public_key,omitempty"`
	MPActivo            bool           `json:"mp_activo"`
	Plan                string         `json:"plan"`
	PlanDesde           *time.Time     `json:"plan_desde,omitempty"`
	PlanHasta           *time.Time     `json:"plan_hasta,omitempty"`
	TrialHasta          *time.Time     `json:"prueba_hasta,omitempty"`
	UpgradeSolicitadoAt *time.Time     `json:"upgrade_solicitado_at,omitempty"`
	UpgradeNota         *string        `json:"upgrade_nota,omitempty"`
	MostrarMarcaAgua    bool           `json:"mostrar_marca_agua"`
	ColorSecundario     *string        `json:"color_secundario,omitempty"`
	ColorCategoria      *string        `json:"color_categoria,omitempty"`
	ColorAccion         *string        `json:"color_accion,omitempty"`
	TipoFuente          *string        `json:"tipo_fuente,omitempty"`
	Slug                string         `json:"slug"`
	CreatedAt           time.Time      `json:"created_at"`
}

// TenantOperativo contiene únicamente la identidad necesaria para presentar
// el panel al staff. Excluye datos fiscales, plan, facturación y credenciales.
type TenantOperativo struct {
	ID               string  `json:"id"`
	Nombre           string  `json:"nombre"`
	NombreFantasia   string  `json:"nombre_fantasia,omitempty"`
	Rubro            string  `json:"rubro"`
	LogoURL          *string `json:"logo_url,omitempty"`
	ColorPrimario    *string `json:"color_primario,omitempty"`
	EstiloVisual     *string `json:"estilo_visual,omitempty"`
	MostrarMarcaAgua bool    `json:"mostrar_marca_agua"`
	ColorSecundario  *string `json:"color_secundario,omitempty"`
	ColorCategoria   *string `json:"color_categoria,omitempty"`
	ColorAccion      *string `json:"color_accion,omitempty"`
	TipoFuente       *string `json:"tipo_fuente,omitempty"`
	Slug             string  `json:"slug"`
}

func (t *Tenant) VistaOperativa() TenantOperativo {
	return TenantOperativo{
		ID: t.ID, Nombre: t.Nombre, NombreFantasia: t.NombreFantasia, Rubro: t.Rubro,
		LogoURL: t.LogoURL, ColorPrimario: t.ColorPrimario, EstiloVisual: t.EstiloVisual,
		MostrarMarcaAgua: t.MostrarMarcaAgua, ColorSecundario: t.ColorSecundario,
		ColorCategoria: t.ColorCategoria, ColorAccion: t.ColorAccion, TipoFuente: t.TipoFuente,
		Slug: t.Slug,
	}
}

type OnboardingInput struct {
	Nombre         string `json:"nombre"`
	NombreFantasia string `json:"nombre_fantasia"`
	Rubro          string `json:"rubro"`
	Descripcion    string `json:"descripcion"`
	Slug           string `json:"slug"`
	EmailAdmin     string `json:"email_admin"`
	NombreAdmin    string `json:"nombre_admin"`
	Password       string `json:"password"`
	PasswordHash   string `json:"-"`
	// Alternativa a Password: registro con la cuenta de Google del admin.
	GoogleCredential string         `json:"google_credential"`
	GoogleSub        string         `json:"-"`
	SucursalNombre   string         `json:"sucursal_nombre"`
	Whatsapp         string         `json:"whatsapp"`
	EmailSucursal    string         `json:"email_sucursal"`
	Horarios         map[string]any `json:"horarios"`
	Plan             string         `json:"-"`
	PlanDesde        *time.Time     `json:"-"`
	PlanHasta        *time.Time     `json:"-"`
	TrialHasta       *time.Time     `json:"-"`
}

type ActualizarTenantInput struct {
	Nombre           *string        `json:"nombre,omitempty"`
	NombreFantasia   *string        `json:"nombre_fantasia,omitempty"`
	Rubro            *string        `json:"rubro,omitempty"`
	Descripcion      *string        `json:"descripcion,omitempty"`
	EmailContacto    *string        `json:"email_contacto,omitempty"`
	Whatsapp         *string        `json:"whatsapp,omitempty"`
	LogoURL          *string        `json:"logo_url,omitempty"`
	ColorPrimario    *string        `json:"color_primario,omitempty"`
	EstiloVisual     *string        `json:"estilo_visual,omitempty"`
	DatosFiscales    map[string]any `json:"datos_fiscales,omitempty"`
	GoogleReviewURL  *string        `json:"google_review_url,omitempty"`
	MPAccessToken    *string        `json:"mp_access_token,omitempty"`
	MPPublicKey      *string        `json:"mp_public_key,omitempty"`
	MPActivo         *bool          `json:"mp_activo,omitempty"`
	MostrarMarcaAgua *bool          `json:"mostrar_marca_agua,omitempty"`
	ColorSecundario  *string        `json:"color_secundario,omitempty"`
	ColorCategoria   *string        `json:"color_categoria,omitempty"`
	ColorAccion      *string        `json:"color_accion,omitempty"`
	TipoFuente       *string        `json:"tipo_fuente,omitempty"`
}
