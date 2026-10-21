package tenant

import (
	"time"
)

type Recurso string

const (
	RecursoMesas           Recurso = "mesas"
	RecursoProductos       Recurso = "productos"
	RecursoSucursales      Recurso = "sucursales"
	RecursoCargaMasiva     Recurso = "carga_masiva"    // Pro only
	RecursoPersonalizacion Recurso = "personalizacion" // Pro only
)

const (
	PlanFree = "free"
	PlanPro  = "pro"

	PrecioMensualProARS = 30000
	MonedaPlanPro       = "ARS"
	MesesPruebaInicial  = 3
)

const Ilimitado = -1

// Límites del plan Free. En Pro, todos valen Ilimitado.
var limitesFree = map[Recurso]int{
	RecursoMesas:      10,
	RecursoProductos:  30,
	RecursoSucursales: 1,
}

var recursosPro = map[Recurso]bool{
	RecursoCargaMasiva:     true,
	RecursoPersonalizacion: true,
}

type EstadoCuotas struct {
	Plan                string
	PlanDesde           *time.Time
	PlanHasta           *time.Time
	TrialHasta          *time.Time
	UpgradeSolicitadoAt *time.Time
	UpgradeNota         *string
	Uso                 map[Recurso]int
}

// NuevaPruebaInicial devuelve el período Pro que recibe un negocio al darse de alta.
// Se usan meses calendario para que la prueba venza el mismo día tres meses después.
func NuevaPruebaInicial(now time.Time) (time.Time, time.Time) {
	desde := now.UTC()
	return desde, desde.AddDate(0, MesesPruebaInicial, 0)
}

// PlanEfectivo: 'pro' solo si plan=='pro' Y (plan_hasta IS NULL OR > now).
// Un Pro vencido degrada a Free. Fail-safe de negocio.
func (e EstadoCuotas) PlanEfectivo(now time.Time) string {
	if e.Plan != PlanPro {
		return PlanFree
	}
	if e.PlanHasta != nil && !e.PlanHasta.After(now) {
		return PlanFree
	}
	return PlanPro
}

func (e EstadoCuotas) EnPrueba(now time.Time) bool {
	return e.PlanEfectivo(now) == PlanPro && e.TrialHasta != nil && e.TrialHasta.After(now)
}

func (e EstadoCuotas) Limite(rec Recurso) int {
	return e.LimiteConTiempo(rec, time.Now())
}

func (e EstadoCuotas) LimiteConTiempo(rec Recurso, now time.Time) int {
	if e.PlanEfectivo(now) == PlanPro {
		return Ilimitado
	}
	if lim, ok := limitesFree[rec]; ok {
		return lim
	}
	if recursosPro[rec] {
		return 0
	}
	return Ilimitado
}

func (e EstadoCuotas) Permitido(rec Recurso) bool {
	return e.PermitidoConTiempo(rec, time.Now())
}

func (e EstadoCuotas) PermitidoConTiempo(rec Recurso, now time.Time) bool {
	plan := e.PlanEfectivo(now)
	if plan == PlanPro {
		return true
	}
	if recursosPro[rec] {
		return false
	}
	lim, ok := limitesFree[rec]
	if !ok {
		return true
	}
	uso := 0
	if e.Uso != nil {
		uso = e.Uso[rec]
	}
	return uso < lim
}

func (e EstadoCuotas) Alcanzado(rec Recurso) bool {
	return e.AlcanzadoConTiempo(rec, time.Now())
}

func (e EstadoCuotas) AlcanzadoConTiempo(rec Recurso, now time.Time) bool {
	return !e.PermitidoConTiempo(rec, now)
}

// NombreRecurso: Etiqueta en plural para el mensaje de error: "mesas", "productos", ...
func NombreRecurso(rec Recurso) string {
	switch rec {
	case RecursoMesas:
		return "mesas"
	case RecursoProductos:
		return "productos"
	case RecursoSucursales:
		return "sucursales"
	case RecursoCargaMasiva:
		return "carga masiva"
	case RecursoPersonalizacion:
		return "personalización visual"
	default:
		return string(rec)
	}
}
