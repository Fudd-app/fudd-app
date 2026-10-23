package tenant

import (
	"context"
	"errors"
	"testing"
	"time"
)

type override struct {
	habilitado bool
	vence      *time.Time
}

// storeFuncionalidades replica la regla SQL de pgStore.FuncionalidadHabilitada.
type storeFuncionalidades struct {
	Store
	cuotas             *EstadoCuotas
	plan               map[string]bool
	override           *override
	consultas          int
	fallaCuotas        error
	fallaFuncionalidad error
}

func (s *storeFuncionalidades) ObtenerEstadoCuotas(context.Context, string) (*EstadoCuotas, error) {
	return s.cuotas, s.fallaCuotas
}

func (s *storeFuncionalidades) FuncionalidadHabilitada(_ context.Context, _ string, _ Recurso, plan string, ahora time.Time) (bool, error) {
	s.consultas++
	if s.fallaFuncionalidad != nil {
		return false, s.fallaFuncionalidad
	}
	if o := s.override; o != nil && (o.vence == nil || o.vence.After(ahora)) {
		return o.habilitado, nil
	}
	return s.plan[plan], nil
}

type relojFalso struct{ t time.Time }

func (r *relojFalso) ahora() time.Time { return r.t }

func nuevoServiceFuncionalidades(st *storeFuncionalidades) (*Service, *relojFalso) {
	reloj := &relojFalso{t: time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)}
	svc := NuevoService(st)
	svc.ahora = reloj.ahora
	return svc, reloj
}

func planesPorDefecto() map[string]bool { return map[string]bool{PlanPro: true, PlanFree: false} }

func TestAccesoFuncionalidad_Resolucion(t *testing.T) {
	ahora := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)
	futuro := ahora.Add(24 * time.Hour)
	pasado := ahora.Add(-24 * time.Hour)

	casos := []struct {
		nombre   string
		cuotas   EstadoCuotas
		override *override
		quiero   bool
	}{
		{"free sin excepción", EstadoCuotas{Plan: PlanFree}, nil, false},
		{"free con excepción vigente", EstadoCuotas{Plan: PlanFree}, &override{true, &futuro}, true},
		{"free con excepción sin vencimiento", EstadoCuotas{Plan: PlanFree}, &override{true, nil}, true},
		{"excepción vencida cae al plan", EstadoCuotas{Plan: PlanFree}, &override{true, &pasado}, false},
		{"pro", EstadoCuotas{Plan: PlanPro}, nil, true},
		{"pro con excepción en false", EstadoCuotas{Plan: PlanPro}, &override{false, nil}, false},
		{"prueba pro vigente", EstadoCuotas{Plan: PlanPro, PlanHasta: &futuro, TrialHasta: &futuro}, nil, true},
		{"prueba vencida", EstadoCuotas{Plan: PlanPro, PlanHasta: &pasado, TrialHasta: &pasado}, nil, false},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			cuotas := c.cuotas
			svc, _ := nuevoServiceFuncionalidades(&storeFuncionalidades{cuotas: &cuotas, plan: planesPorDefecto(), override: c.override})
			got, err := svc.AccesoFuncionalidad(context.Background(), "t-1", RecursoMCP)
			if err != nil {
				t.Fatalf("error inesperado: %v", err)
			}
			if got != c.quiero {
				t.Errorf("got %v, quiero %v", got, c.quiero)
			}
		})
	}
}

func TestAccesoFuncionalidad_SinFilaDeDefinicionEsFalse(t *testing.T) {
	svc, _ := nuevoServiceFuncionalidades(&storeFuncionalidades{cuotas: &EstadoCuotas{Plan: PlanPro}, plan: map[string]bool{}})
	got, err := svc.AccesoFuncionalidad(context.Background(), "t-1", RecursoMCP)
	if err != nil || got {
		t.Errorf("got %v, %v; quiero false sin error", got, err)
	}
}

func TestAccesoFuncionalidad_Cache(t *testing.T) {
	st := &storeFuncionalidades{cuotas: &EstadoCuotas{Plan: PlanFree}, plan: planesPorDefecto()}
	svc, reloj := nuevoServiceFuncionalidades(st)
	ctx := context.Background()

	if got, _ := svc.AccesoFuncionalidad(ctx, "t-1", RecursoMCP); got {
		t.Fatal("free debería estar sin acceso")
	}
	st.override = &override{habilitado: true}

	reloj.t = reloj.t.Add(59 * time.Second)
	if got, _ := svc.AccesoFuncionalidad(ctx, "t-1", RecursoMCP); got || st.consultas != 1 {
		t.Errorf("dentro del TTL debía usar el cache: got=%v consultas=%d", got, st.consultas)
	}
	if got, _ := svc.AccesoFuncionalidad(ctx, "t-2", RecursoMCP); !got || st.consultas != 2 {
		t.Errorf("otro tenant no comparte cache: got=%v consultas=%d", got, st.consultas)
	}

	reloj.t = reloj.t.Add(2 * time.Second)
	if got, _ := svc.AccesoFuncionalidad(ctx, "t-1", RecursoMCP); !got || st.consultas != 3 {
		t.Errorf("pasado el TTL debía reconsultar: got=%v consultas=%d", got, st.consultas)
	}
}

func TestAccesoFuncionalidad_ErrorNoSeCacheaNiSeTrataComoSinAcceso(t *testing.T) {
	boom := errors.New("db caída")
	st := &storeFuncionalidades{cuotas: &EstadoCuotas{Plan: PlanPro}, plan: planesPorDefecto(), fallaCuotas: boom}
	svc, _ := nuevoServiceFuncionalidades(st)
	if _, err := svc.AccesoFuncionalidad(context.Background(), "t-1", RecursoMCP); !errors.Is(err, boom) {
		t.Fatalf("esperaba el error de la DB, obtuve %v", err)
	}
	st.fallaCuotas = nil
	if got, err := svc.AccesoFuncionalidad(context.Background(), "t-1", RecursoMCP); err != nil || !got {
		t.Errorf("tras recuperarse debía resolver: %v %v", got, err)
	}
}

func TestAccesoFuncionalidad_ErrorSoloEnFuncionalidadHabilitadaNoSeCachea(t *testing.T) {
	boom := errors.New("db caída")
	st := &storeFuncionalidades{cuotas: &EstadoCuotas{Plan: PlanPro}, plan: planesPorDefecto(), fallaFuncionalidad: boom}
	svc, _ := nuevoServiceFuncionalidades(st)
	ctx := context.Background()
	if got, err := svc.AccesoFuncionalidad(ctx, "t-1", RecursoMCP); !errors.Is(err, boom) || got {
		t.Fatalf("esperaba el error de la DB y no acceso, obtuve %v, %v", got, err)
	}
	st.fallaFuncionalidad = nil
	if got, err := svc.AccesoFuncionalidad(ctx, "t-1", RecursoMCP); err != nil || !got {
		t.Errorf("el error no debía cachearse: %v %v", got, err)
	}
	if st.consultas != 2 {
		t.Errorf("consultas: got %d, quiero 2", st.consultas)
	}
}

func TestEstadoPlan_FalloDeFuncionalidadesNoRompeElEstado(t *testing.T) {
	st := &storeFuncionalidades{cuotas: &EstadoCuotas{Plan: PlanPro}, plan: planesPorDefecto(), fallaFuncionalidad: errors.New("db caída")}
	svc, _ := nuevoServiceFuncionalidades(st)
	dto, err := svc.EstadoPlan(context.Background(), "t-1")
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if dto.Funcionalidades != nil {
		t.Errorf("Funcionalidades debía ser nil, obtuve %v", dto.Funcionalidades)
	}
}
