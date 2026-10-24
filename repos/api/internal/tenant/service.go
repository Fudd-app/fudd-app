package tenant

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
)

// VerificadorGoogle valida la credencial de Google usada para registrarse.
type VerificadorGoogle interface {
	VerificarIdentidadGoogle(ctx context.Context, credencial string) (*auth.IdentidadGoogle, error)
}

type Service struct {
	store           Store
	google          VerificadorGoogle
	ahora           func() time.Time
	funcionalidades cacheFuncionalidades
}

func NuevoService(s Store) *Service { return &Service{store: s} }

var colorHexPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

var slugsReservados = map[string]struct{}{
	"admin":         {},
	"superadmin":    {},
	"platform":      {},
	"dashboard":     {},
	"login":         {},
	"logout":        {},
	"auth":          {},
	"kds":           {},
	"cocina":        {},
	"mesa":          {},
	"mesas":         {},
	"api":           {},
	"public":        {},
	"publica":       {},
	"static":        {},
	"pedidos":       {},
	"pedido":        {},
	"carta":         {},
	"menu":          {},
	"onboarding":    {},
	"terminos":      {},
	"privacidad":    {},
	"ayuda":         {},
	"status":        {},
	"health":        {},
	"configuracion": {},
}

func EsSlugReservado(slug string) bool {
	_, reservado := slugsReservados[strings.ToLower(strings.TrimSpace(slug))]
	return reservado
}

func NuevoServiceConGoogle(s Store, google VerificadorGoogle) *Service {
	return &Service{store: s, google: google}
}

func (svc *Service) Crear(ctx context.Context, input OnboardingInput) (*Tenant, error) {
	if input.Slug == "" {
		return nil, fmt.Errorf("%w: slug requerido", ErrValidation)
	}
	if EsSlugReservado(input.Slug) {
		return nil, fmt.Errorf("%w: el nombre en URL está reservado por el sistema", ErrValidation)
	}
	if strings.TrimSpace(input.GoogleCredential) != "" {
		if err := svc.aplicarIdentidadGoogle(ctx, &input); err != nil {
			return nil, err
		}
	} else if strings.TrimSpace(input.Password) != "" {
		if input.EmailAdmin == "" {
			return nil, fmt.Errorf("%w: email del admin requerido", ErrValidation)
		}
		passwordHash, err := auth.HashPassword(input.Password)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrValidation, err)
		}
		input.PasswordHash = passwordHash
	} else {
		if input.EmailAdmin == "" {
			return nil, fmt.Errorf("%w: email del admin requerido", ErrValidation)
		}
		input.PasswordHash = ""
	}
	input.Password = ""
	input.GoogleCredential = ""
	input.Rubro = normalizarRubro(input.Rubro)
	input.Descripcion = strings.TrimSpace(input.Descripcion)
	// Los métodos de acceso buscan al usuario por email; guardarlo siempre en
	// minúsculas evita que un registro con mayúsculas quede inaccesible.
	input.EmailAdmin = auth.NormalizarEmail(input.EmailAdmin)
	input.EmailSucursal = auth.NormalizarEmail(input.EmailSucursal)
	input.SucursalNombre = strings.TrimSpace(input.SucursalNombre)
	input.Whatsapp = strings.TrimSpace(input.Whatsapp)
	if input.Whatsapp != "" && !whatsappValido(input.Whatsapp) {
		return nil, fmt.Errorf("%w: el WhatsApp solo puede contener números, espacios y un + inicial", ErrValidation)
	}
	if input.SucursalNombre == "" {
		input.SucursalNombre = "Casa central"
	}
	if input.Horarios == nil {
		input.Horarios = make(map[string]any)
	}
	inicioPrueba, finPrueba := NuevaPruebaInicial(time.Now())
	input.Plan = PlanPro
	input.PlanDesde = &inicioPrueba
	input.PlanHasta = &finPrueba
	input.TrialHasta = &finPrueba
	return svc.store.Crear(ctx, input)
}

// aplicarIdentidadGoogle reemplaza el email del admin por el verificado por
// Google y vincula su `sub`; el admin queda sin contraseña local.
func (svc *Service) aplicarIdentidadGoogle(ctx context.Context, input *OnboardingInput) error {
	if svc.google == nil {
		return fmt.Errorf("%w: el registro con Google no está disponible", ErrValidation)
	}
	identidad, err := svc.google.VerificarIdentidadGoogle(ctx, input.GoogleCredential)
	if err != nil {
		if errors.Is(err, auth.ErrGoogleNoConfigurado) ||
			errors.Is(err, auth.ErrCredencialGoogleInvalida) ||
			errors.Is(err, auth.ErrEmailGoogleNoVerificado) ||
			errors.Is(err, auth.ErrIdentidadGoogleIncompleta) {
			return fmt.Errorf("%w: no pudimos validar tu cuenta de Google, volvé a intentarlo", ErrValidation)
		}
		return err
	}
	input.EmailAdmin = identidad.Email
	input.GoogleSub = identidad.Subject
	input.PasswordHash = ""
	return nil
}

func (svc *Service) ObtenerPorID(ctx context.Context, id string) (*Tenant, error) {
	return svc.store.ObtenerPorID(ctx, id)
}

func (svc *Service) Actualizar(ctx context.Context, id string, input ActualizarTenantInput) (*Tenant, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: id de tenant requerido", ErrValidation)
	}
	if input.Nombre != nil {
		trimmed := strings.TrimSpace(*input.Nombre)
		if trimmed == "" {
			return nil, fmt.Errorf("%w: el nombre no puede estar vacío", ErrValidation)
		}
		input.Nombre = &trimmed
	}
	if input.NombreFantasia != nil {
		trimmed := strings.TrimSpace(*input.NombreFantasia)
		input.NombreFantasia = &trimmed
	}
	if input.Rubro != nil {
		normalizado := normalizarRubro(*input.Rubro)
		input.Rubro = &normalizado
	}
	if input.EmailContacto != nil {
		normalizado := auth.NormalizarEmail(*input.EmailContacto)
		input.EmailContacto = &normalizado
	}
	if input.Whatsapp != nil {
		trimmed := strings.TrimSpace(*input.Whatsapp)
		input.Whatsapp = &trimmed
	}
	if input.Descripcion != nil {
		trimmed := strings.TrimSpace(*input.Descripcion)
		input.Descripcion = &trimmed
	}
	if input.GoogleReviewURL != nil {
		trimmed := strings.TrimSpace(*input.GoogleReviewURL)
		input.GoogleReviewURL = &trimmed
	}
	if input.LogoURL != nil {
		trimmed := strings.TrimSpace(*input.LogoURL)
		input.LogoURL = &trimmed
	}
	if input.ColorPrimario != nil {
		trimmed := strings.TrimSpace(*input.ColorPrimario)
		if !colorHexPattern.MatchString(trimmed) {
			return nil, fmt.Errorf("%w: color_primario debe usar formato #RRGGBB", ErrValidation)
		}
		input.ColorPrimario = &trimmed
	}
	coloresAvanzados := []*string{input.ColorSecundario, input.ColorCategoria, input.ColorAccion}
	for _, color := range coloresAvanzados {
		if color != nil && !colorHexPattern.MatchString(strings.TrimSpace(*color)) {
			return nil, fmt.Errorf("%w: los colores deben usar formato #RRGGBB", ErrValidation)
		}
	}
	if input.ColorSecundario != nil {
		valor := strings.TrimSpace(*input.ColorSecundario)
		input.ColorSecundario = &valor
	}
	if input.ColorCategoria != nil {
		valor := strings.TrimSpace(*input.ColorCategoria)
		input.ColorCategoria = &valor
	}
	if input.ColorAccion != nil {
		valor := strings.TrimSpace(*input.ColorAccion)
		input.ColorAccion = &valor
	}
	personalizacionPro := input.ColorSecundario != nil || input.ColorCategoria != nil || input.ColorAccion != nil ||
		input.TipoFuente != nil || input.MostrarMarcaAgua != nil
	if personalizacionPro {
		cuotas, err := svc.store.ObtenerEstadoCuotas(ctx, id)
		if err != nil {
			return nil, err
		}
		if cuotas.PlanEfectivo(time.Now()) != PlanPro {
			return nil, ErrPlanRequired
		}
	}
	if input.EstiloVisual != nil {
		trimmed := strings.ToLower(strings.TrimSpace(*input.EstiloVisual))
		if trimmed != "claro" && trimmed != "oscuro" {
			trimmed = "oscuro"
		}
		input.EstiloVisual = &trimmed
	}

	return svc.store.Actualizar(ctx, id, input)
}

// SlugDisponible indica si el nombre en URL todavía no lo usa otro negocio.
// La creación vuelve a validarlo por la restricción única de la base.
func (svc *Service) SlugDisponible(ctx context.Context, slug string) (bool, error) {
	slug = strings.ToLower(strings.TrimSpace(slug))
	if !slugPattern.MatchString(slug) {
		return false, fmt.Errorf("%w: nombre en URL inválido", ErrValidation)
	}
	if EsSlugReservado(slug) {
		return false, nil
	}

	enUso, err := svc.store.SlugEnUso(ctx, slug)
	if err != nil {
		return false, err
	}
	return !enUso, nil
}

func (svc *Service) EmailAdminDisponible(ctx context.Context, email string) (bool, error) {
	email = auth.NormalizarEmail(email)
	if email == "" {
		return false, fmt.Errorf("%w: email del admin requerido", ErrValidation)
	}

	enUso, err := svc.store.EmailAdminEnUso(ctx, email)
	if err != nil {
		return false, err
	}
	return !enUso, nil
}

func normalizarRubro(r string) string {
	r = strings.ToLower(strings.TrimSpace(r))
	// Reemplazar acentos
	r = strings.ReplaceAll(r, "í", "i")
	r = strings.ReplaceAll(r, "á", "a")
	r = strings.ReplaceAll(r, "é", "e")
	r = strings.ReplaceAll(r, "ó", "o")
	r = strings.ReplaceAll(r, "ú", "u")

	switch r {
	case "restaurante", "restaurant":
		return "restaurante"
	case "cafeteria", "cafe":
		return "cafeteria"
	case "comida_rapida", "comida rapida", "fast_food", "fast food":
		return "comida_rapida"
	case "cerveceria":
		return "cerveceria"
	case "pizzeria":
		return "pizzeria"
	case "bar", "pub", "bar / pub":
		return "bar"
	case "otro":
		return "otro"
	default:
		return "otro"
	}
}

func whatsappValido(value string) bool {
	for index, char := range value {
		if (char >= '0' && char <= '9') || char == ' ' || (char == '+' && index == 0) {
			continue
		}
		return false
	}
	return true
}

type EstadoPlanDTO struct {
	Plan                string          `json:"plan"`
	PlanDesde           *time.Time      `json:"plan_desde"`
	PlanHasta           *time.Time      `json:"plan_hasta"`
	DiasRestantesPro    *int            `json:"dias_restantes_pro"`
	EnPrueba            bool            `json:"en_prueba"`
	PruebaHasta         *time.Time      `json:"prueba_hasta"`
	DiasRestantesPrueba *int            `json:"dias_restantes_prueba"`
	PrecioMensual       int             `json:"precio_mensual"`
	Moneda              string          `json:"moneda"`
	MesesPrueba         int             `json:"meses_prueba"`
	UpgradeSolicitadoAt *time.Time      `json:"upgrade_solicitado_at"`
	UpgradeNota         *string         `json:"upgrade_nota"`
	Limites             map[string]int  `json:"limites"`
	Uso                 map[string]int  `json:"uso"`
	Disponibles         map[string]int  `json:"disponibles"`
	Alcanzado           map[string]bool `json:"alcanzado"`
	Funcionalidades     map[string]bool `json:"funcionalidades,omitempty"`
}

func (svc *Service) EstadoPlan(ctx context.Context, tenantID string) (*EstadoPlanDTO, error) {
	cuotas, err := svc.store.ObtenerEstadoCuotas(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	planEfectivo := cuotas.PlanEfectivo(now)
	enPrueba := cuotas.EnPrueba(now)

	var diasRestantes *int
	if planEfectivo == PlanPro && cuotas.PlanHasta != nil {
		dias := int(cuotas.PlanHasta.Sub(now).Hours() / 24)
		if dias < 0 {
			dias = 0
		}
		diasRestantes = &dias
	}

	var diasRestantesPrueba *int
	if enPrueba && cuotas.TrialHasta != nil {
		dias := int(cuotas.TrialHasta.Sub(now).Hours() / 24)
		if dias < 0 {
			dias = 0
		}
		diasRestantesPrueba = &dias
	}

	// Es información adicional: si falla, el resto del estado del plan sigue siendo válido.
	var funcionalidades map[string]bool
	accesoMCP, err := svc.AccesoFuncionalidad(ctx, tenantID, RecursoMCP)
	if err != nil {
		slog.ErrorContext(ctx, "error consultando funcionalidades del plan", "tenant_id", tenantID, "error", err)
	} else {
		funcionalidades = map[string]bool{string(RecursoMCP): accesoMCP}
	}

	usoMesas := cuotas.Uso[RecursoMesas]
	usoProductos := cuotas.Uso[RecursoProductos]
	usoSucursales := cuotas.Uso[RecursoSucursales]

	dto := &EstadoPlanDTO{
		Plan:                planEfectivo,
		PlanDesde:           cuotas.PlanDesde,
		PlanHasta:           cuotas.PlanHasta,
		DiasRestantesPro:    diasRestantes,
		EnPrueba:            enPrueba,
		PruebaHasta:         cuotas.TrialHasta,
		DiasRestantesPrueba: diasRestantesPrueba,
		PrecioMensual:       PrecioMensualProARS,
		Moneda:              MonedaPlanPro,
		MesesPrueba:         MesesPruebaInicial,
		UpgradeSolicitadoAt: cuotas.UpgradeSolicitadoAt,
		UpgradeNota:         cuotas.UpgradeNota,
		Funcionalidades:     funcionalidades,
		Uso: map[string]int{
			string(RecursoMesas):      usoMesas,
			string(RecursoProductos):  usoProductos,
			string(RecursoSucursales): usoSucursales,
		},
	}

	if planEfectivo == PlanPro {
		dto.Limites = map[string]int{
			string(RecursoMesas):      Ilimitado,
			string(RecursoProductos):  Ilimitado,
			string(RecursoSucursales): Ilimitado,
		}
		dto.Disponibles = map[string]int{
			string(RecursoMesas):      Ilimitado,
			string(RecursoProductos):  Ilimitado,
			string(RecursoSucursales): Ilimitado,
		}
		dto.Alcanzado = map[string]bool{
			string(RecursoMesas):      false,
			string(RecursoProductos):  false,
			string(RecursoSucursales): false,
		}
	} else {
		dto.Limites = map[string]int{
			string(RecursoMesas):      limitesFree[RecursoMesas],
			string(RecursoProductos):  limitesFree[RecursoProductos],
			string(RecursoSucursales): limitesFree[RecursoSucursales],
		}

		dispMesas := limitesFree[RecursoMesas] - usoMesas
		if dispMesas < 0 {
			dispMesas = 0
		}
		dispProductos := limitesFree[RecursoProductos] - usoProductos
		if dispProductos < 0 {
			dispProductos = 0
		}
		dispSucursales := limitesFree[RecursoSucursales] - usoSucursales
		if dispSucursales < 0 {
			dispSucursales = 0
		}

		dto.Disponibles = map[string]int{
			string(RecursoMesas):      dispMesas,
			string(RecursoProductos):  dispProductos,
			string(RecursoSucursales): dispSucursales,
		}

		dto.Alcanzado = map[string]bool{
			string(RecursoMesas):      usoMesas >= limitesFree[RecursoMesas],
			string(RecursoProductos):  usoProductos >= limitesFree[RecursoProductos],
			string(RecursoSucursales): usoSucursales >= limitesFree[RecursoSucursales],
		}
	}

	return dto, nil
}

func (svc *Service) SolicitarUpgrade(ctx context.Context, tenantID, nota string) (*Tenant, error) {
	if strings.TrimSpace(tenantID) == "" {
		return nil, fmt.Errorf("%w: tenant_id requerido", ErrValidation)
	}
	return svc.store.RegistrarSolicitudUpgrade(ctx, tenantID, strings.TrimSpace(nota))
}
