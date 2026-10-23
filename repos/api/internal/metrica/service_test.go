package metrica

import (
	"context"
	"errors"
	"testing"
	"time"
)

const sucursalValida = "b7fcf135-c45f-454d-a7e0-eb7874be0561"

type storeFalso struct {
	pertenece    bool
	bases        []ResumenBase
	baseCalls    int
	platos       []PlatoEstrella
	porDia       []MetricaDiaria
	porTurno     []MetricaTurno
	porEstado    []MetricaEstado
	porCategoria []MetricaCategoria
	mapaCalor    []CeldaCalor
}

func (s *storeFalso) SucursalPerteneceATenant(context.Context, string, string) (bool, error) {
	return s.pertenece, nil
}

func (s *storeFalso) ObtenerResumenBase(context.Context, string, string, time.Time, time.Time) (ResumenBase, error) {
	if len(s.bases) == 0 {
		return ResumenBase{}, nil
	}
	base := s.bases[s.baseCalls%len(s.bases)]
	s.baseCalls++
	return base, nil
}

func (s *storeFalso) ListarPlatosEstrella(context.Context, string, string, time.Time, time.Time, int) ([]PlatoEstrella, error) {
	return s.platos, nil
}

func (s *storeFalso) ListarPorDia(context.Context, string, string, time.Time, time.Time, string) ([]MetricaDiaria, error) {
	return s.porDia, nil
}

func (s *storeFalso) ListarPorTurno(context.Context, string, string, time.Time, time.Time, string) ([]MetricaTurno, error) {
	return s.porTurno, nil
}

func (s *storeFalso) ListarPorEstado(context.Context, string, string, time.Time, time.Time) ([]MetricaEstado, error) {
	return s.porEstado, nil
}

func (s *storeFalso) ListarPorCategoria(context.Context, string, string, time.Time, time.Time) ([]MetricaCategoria, error) {
	return s.porCategoria, nil
}

func (s *storeFalso) ListarMapaCalor(context.Context, string, string, time.Time, time.Time, string) ([]CeldaCalor, error) {
	return s.mapaCalor, nil
}

func TestObtenerResumenCalculaKPIsYVariaciones(t *testing.T) {
	store := &storeFalso{
		pertenece: true,
		bases: []ResumenBase{
			{
				FacturacionTotal:              100000,
				TicketPromedio:                50000,
				PedidosTotales:                4,
				PedidosCerrados:               2,
				PedidosActivos:                2,
				TiempoPromedioDespachoMinutos: 12,
			},
			{
				FacturacionTotal:              80000,
				TicketPromedio:                40000,
				PedidosTotales:                2,
				PedidosCerrados:               2,
				TiempoPromedioDespachoMinutos: 15,
			},
		},
		platos:    []PlatoEstrella{{ArticuloID: "art-1", Nombre: "Milanesa", Unidades: 8, Monto: 64000}},
		porDia:    []MetricaDiaria{{Fecha: "2026-10-06", FacturacionTotal: 100000, PedidosTotales: 4, PedidosCerrados: 2}},
		porTurno:  []MetricaTurno{{Turno: "Noche", FacturacionTotal: 100000, PedidosTotales: 4, PedidosCerrados: 2}},
		porEstado: []MetricaEstado{{Estado: "cerrado", Cantidad: 2}},
		porCategoria: []MetricaCategoria{
			{CategoriaID: "cat-1", Nombre: "Platos Principales", Unidades: 8, Monto: 75000},
			{CategoriaID: "cat-2", Nombre: "Bebidas", Unidades: 10, Monto: 25000},
		},
		mapaCalor: []CeldaCalor{
			{DiaSemana: 5, Hora: 21, Pedidos: 4, Monto: 100000},
		},
	}
	svc := NuevoService(store)

	resumen, err := svc.ObtenerResumen(
		context.Background(),
		"tenant-1",
		sucursalValida,
		"2026-10-06",
		"2026-10-06",
	)
	if err != nil {
		t.Fatalf("ObtenerResumen devolvió error: %v", err)
	}

	if resumen.FacturacionTotal != 100000 || resumen.PedidosTotales != 4 || resumen.PedidosCerrados != 2 {
		t.Fatalf("resumen inesperado: %+v", resumen.ResumenBase)
	}
	if resumen.PlatoMasVendido == nil || resumen.PlatoMasVendido.Nombre != "Milanesa" {
		t.Fatalf("plato más vendido inesperado: %+v", resumen.PlatoMasVendido)
	}
	assertPorcentaje(t, resumen.Variaciones.FacturacionTotal, 25)
	assertPorcentaje(t, resumen.Variaciones.TicketPromedio, 25)
	assertPorcentaje(t, resumen.Variaciones.PedidosTotales, 100)
	assertPorcentaje(t, resumen.Variaciones.TiempoPromedioDespachoMinutos, -20)

	if len(resumen.PorCategoria) != 2 {
		t.Fatalf("se esperaban 2 categorías, obtuvo %d", len(resumen.PorCategoria))
	}
	if resumen.PorCategoria[0].Porcentaje != 75.0 {
		t.Errorf("porcentaje de categoría 0 esperado 75.0, obtuvo %f", resumen.PorCategoria[0].Porcentaje)
	}
	if resumen.PorCategoria[1].Porcentaje != 25.0 {
		t.Errorf("porcentaje de categoría 1 esperado 25.0, obtuvo %f", resumen.PorCategoria[1].Porcentaje)
	}

	if len(resumen.MapaCalor) != 1 || resumen.MapaCalor[0].DiaSemana != 5 || resumen.MapaCalor[0].Hora != 21 || resumen.MapaCalor[0].Pedidos != 4 {
		t.Fatalf("mapa de calor inesperado: %+v", resumen.MapaCalor)
	}

	duracion := resumen.Periodo.Hasta.Sub(resumen.Periodo.Desde)
	if duracion != 24*time.Hour {
		t.Fatalf("un filtro de un día debe cubrir 24h, obtuvo %v", duracion)
	}
	if got := resumen.Periodo.ComparadoHasta; !got.Equal(resumen.Periodo.Desde) {
		t.Fatalf("el período comparado debe terminar al comenzar el actual: %v", got)
	}
}

func TestObtenerResumenUsaHoyCuandoNoHayFechas(t *testing.T) {
	store := &storeFalso{}
	svc := NuevoService(store)
	svc.ahora = func() time.Time {
		return time.Date(2026, time.October, 6, 15, 30, 0, 0, time.UTC)
	}

	resumen, err := svc.ObtenerResumen(context.Background(), "tenant-1", "", "", "")
	if err != nil {
		t.Fatalf("ObtenerResumen devolvió error: %v", err)
	}
	if got := resumen.Periodo.Desde.In(svc.ubicacion).Format("2006-01-02 15:04"); got != "2026-10-06 00:00" {
		t.Fatalf("inicio del día inesperado: %s", got)
	}
	if resumen.Periodo.Hasta.Sub(resumen.Periodo.Desde) != 24*time.Hour {
		t.Fatalf("el período predeterminado debe abarcar el día completo")
	}
}

func TestObtenerResumenRechazaPeriodoInvalido(t *testing.T) {
	svc := NuevoService(&storeFalso{})
	_, err := svc.ObtenerResumen(context.Background(), "tenant-1", "", "2026-10-07", "2026-10-06")
	if err == nil || !errors.Is(err, ErrValidacion) {
		t.Fatalf("se esperaba ErrValidacion, obtuvo %v", err)
	}
}

func TestObtenerResumenRechazaSucursalAjena(t *testing.T) {
	svc := NuevoService(&storeFalso{pertenece: false})
	_, err := svc.ObtenerResumen(
		context.Background(),
		"tenant-1",
		sucursalValida,
		"2026-10-06",
		"2026-10-06",
	)
	if err != ErrSucursalNoEncontrada {
		t.Fatalf("se esperaba ErrSucursalNoEncontrada, obtuvo %v", err)
	}
}

func BenchmarkObtenerResumen(b *testing.B) {
	store := &storeFalso{
		bases:  []ResumenBase{{FacturacionTotal: 100000, PedidosTotales: 10}},
		platos: []PlatoEstrella{{ArticuloID: "art-1", Nombre: "Milanesa", Unidades: 8, Monto: 64000}},
	}
	svc := NuevoService(store)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := svc.ObtenerResumen(ctx, "tenant-1", "", "2026-10-06", "2026-10-06"); err != nil {
			b.Fatal(err)
		}
	}
}

func assertPorcentaje(t *testing.T, got *float64, want float64) {
	t.Helper()
	if got == nil || *got != want {
		t.Fatalf("porcentaje inesperado: got %v, want %.1f", got, want)
	}
}
