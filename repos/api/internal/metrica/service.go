package metrica

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	zonaHorariaNegocio = "America/Argentina/Buenos_Aires"
	maximoPeriodo      = 366 * 24 * time.Hour
)

var (
	ErrValidacion           = errors.New("validación fallida")
	ErrSucursalNoEncontrada = errors.New("sucursal no encontrada")
)

type Service struct {
	store     Store
	ubicacion *time.Location
	ahora     func() time.Time
}

func NuevoService(store Store) *Service {
	ubicacion, err := time.LoadLocation(zonaHorariaNegocio)
	if err != nil {
		ubicacion = time.FixedZone("America/Argentina/Buenos_Aires", -3*60*60)
	}
	return &Service{store: store, ubicacion: ubicacion, ahora: time.Now}
}

func (svc *Service) ObtenerResumen(
	ctx context.Context,
	tenantID, sucursalID, desdeRaw, hastaRaw string,
) (*Resumen, error) {
	tenantID = strings.TrimSpace(tenantID)
	sucursalID = strings.TrimSpace(sucursalID)
	if tenantID == "" {
		return nil, fmt.Errorf("tenant requerido: %w", ErrValidacion)
	}
	if sucursalID != "" {
		if !uuidValido(sucursalID) {
			return nil, fmt.Errorf("sucursal_id inválido: %w", ErrValidacion)
		}
		pertenece, err := svc.store.SucursalPerteneceATenant(ctx, sucursalID, tenantID)
		if err != nil {
			return nil, err
		}
		if !pertenece {
			return nil, ErrSucursalNoEncontrada
		}
	}

	desde, hasta, err := svc.resolverPeriodo(desdeRaw, hastaRaw)
	if err != nil {
		return nil, err
	}
	duracion := hasta.Sub(desde)
	comparadoHasta := desde
	comparadoDesde := desde.Add(-duracion)

	actual, err := svc.store.ObtenerResumenBase(ctx, tenantID, sucursalID, desde, hasta)
	if err != nil {
		return nil, err
	}
	anterior, err := svc.store.ObtenerResumenBase(ctx, tenantID, sucursalID, comparadoDesde, comparadoHasta)
	if err != nil {
		return nil, err
	}
	platos, err := svc.store.ListarPlatosEstrella(ctx, tenantID, sucursalID, desde, hasta, 5)
	if err != nil {
		return nil, err
	}
	porDia, err := svc.store.ListarPorDia(ctx, tenantID, sucursalID, desde, hasta, zonaHorariaNegocio)
	if err != nil {
		return nil, err
	}
	porTurno, err := svc.store.ListarPorTurno(ctx, tenantID, sucursalID, desde, hasta, zonaHorariaNegocio)
	if err != nil {
		return nil, err
	}
	porEstado, err := svc.store.ListarPorEstado(ctx, tenantID, sucursalID, desde, hasta)
	if err != nil {
		return nil, err
	}
	porCategoria, err := svc.store.ListarPorCategoria(ctx, tenantID, sucursalID, desde, hasta)
	if err != nil {
		return nil, err
	}
	mapaCalor, err := svc.store.ListarMapaCalor(ctx, tenantID, sucursalID, desde, hasta, zonaHorariaNegocio)
	if err != nil {
		return nil, err
	}

	var totalVentasCategorias float64
	for _, c := range porCategoria {
		totalVentasCategorias += c.Monto
	}
	if totalVentasCategorias > 0 {
		for i := range porCategoria {
			porcentaje := (porCategoria[i].Monto / totalVentasCategorias) * 100
			porCategoria[i].Porcentaje = math.Round(porcentaje*10) / 10
		}
	}

	resumen := &Resumen{
		Periodo: Periodo{
			Desde:          desde,
			Hasta:          hasta,
			ZonaHoraria:    zonaHorariaNegocio,
			SucursalID:     sucursalID,
			ComparadoDesde: comparadoDesde,
			ComparadoHasta: comparadoHasta,
		},
		ResumenBase:    actual,
		PlatosEstrella: platos,
		Variaciones: Variaciones{
			FacturacionTotal:              variacionPorcentual(actual.FacturacionTotal, anterior.FacturacionTotal),
			TicketPromedio:                variacionPorcentual(actual.TicketPromedio, anterior.TicketPromedio),
			PedidosTotales:                variacionPorcentual(float64(actual.PedidosTotales), float64(anterior.PedidosTotales)),
			TiempoPromedioDespachoMinutos: variacionPorcentual(actual.TiempoPromedioDespachoMinutos, anterior.TiempoPromedioDespachoMinutos),
		},
		PorDia:       porDia,
		PorTurno:     porTurno,
		PorEstado:    porEstado,
		PorCategoria: porCategoria,
		MapaCalor:    mapaCalor,
	}
	if len(platos) > 0 {
		resumen.PlatoMasVendido = &resumen.PlatosEstrella[0]
	}
	return resumen, nil
}

func (svc *Service) resolverPeriodo(desdeRaw, hastaRaw string) (time.Time, time.Time, error) {
	desdeRaw = strings.TrimSpace(desdeRaw)
	hastaRaw = strings.TrimSpace(hastaRaw)
	if desdeRaw == "" && hastaRaw == "" {
		ahora := svc.ahora().In(svc.ubicacion)
		desde := time.Date(ahora.Year(), ahora.Month(), ahora.Day(), 0, 0, 0, 0, svc.ubicacion)
		return desde, desde.AddDate(0, 0, 1), nil
	}
	if desdeRaw == "" || hastaRaw == "" {
		return time.Time{}, time.Time{}, fmt.Errorf("desde y hasta deben enviarse juntos: %w", ErrValidacion)
	}

	desde, _, err := svc.parsearLimite(desdeRaw)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("desde inválido: %w", ErrValidacion)
	}
	hasta, fechaCompleta, err := svc.parsearLimite(hastaRaw)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("hasta inválido: %w", ErrValidacion)
	}
	if fechaCompleta {
		hasta = hasta.AddDate(0, 0, 1)
	}
	if !hasta.After(desde) {
		return time.Time{}, time.Time{}, fmt.Errorf("hasta debe ser posterior a desde: %w", ErrValidacion)
	}
	if hasta.Sub(desde) > maximoPeriodo {
		return time.Time{}, time.Time{}, fmt.Errorf("el período no puede superar 366 días: %w", ErrValidacion)
	}
	return desde, hasta, nil
}

func (svc *Service) parsearLimite(value string) (time.Time, bool, error) {
	if len(value) == len("2006-01-02") {
		fecha, err := time.ParseInLocation("2006-01-02", value, svc.ubicacion)
		return fecha, true, err
	}
	fecha, err := time.Parse(time.RFC3339Nano, value)
	return fecha, false, err
}

func variacionPorcentual(actual, anterior float64) *float64 {
	if anterior == 0 {
		return nil
	}
	valor := ((actual - anterior) / math.Abs(anterior)) * 100
	valor = math.Round(valor*10) / 10
	return &valor
}

func uuidValido(value string) bool {
	if len(value) != 36 || value[8] != '-' || value[13] != '-' || value[18] != '-' || value[23] != '-' {
		return false
	}
	compacto := strings.ReplaceAll(value, "-", "")
	_, err := hex.DecodeString(compacto)
	return err == nil
}
