package platform

import (
	"context"
	"runtime"
	"time"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/db"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/notificacion"
)

var inicioServidor = time.Now()

type Service struct {
	store   Store
	tracker *Tracker
}

func NuevoService(store Store, tracker *Tracker) *Service {
	if tracker == nil {
		tracker = InstanciaTracker
	}
	return &Service{
		store:   store,
		tracker: tracker,
	}
}

func (svc *Service) ObtenerStatus(ctx context.Context) (*StatusPlataforma, error) {
	st, err := svc.store.ObtenerMetricasGlobales(ctx)
	if err != nil {
		return nil, err
	}

	st.UptimeSegundos = int64(time.Since(inicioServidor).Seconds())
	st.Timestamp = time.Now().Format(time.RFC3339)
	st.UsuariosOnline = svc.tracker.TotalUsuariosOnline()

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	st.Goroutines = runtime.NumGoroutine()
	st.HeapAllocMB = float64(mem.HeapAlloc) / (1024 * 1024)

	canales, subs := notificacion.Instancia.Estadisticas()
	st.SSECanales = canales
	st.SSESuscriptores = subs

	if db.Pool != nil {
		pStat := db.Pool.Stat()
		st.PoolStats = map[string]any{
			"max_conns":      pStat.MaxConns(),
			"total_conns":    pStat.TotalConns(),
			"acquired_conns": pStat.AcquiredConns(),
			"idle_conns":     pStat.IdleConns(),
		}
	}

	return st, nil
}

func (svc *Service) ListarTenants(ctx context.Context) ([]TenantResumen, error) {
	return svc.store.ListarTenantsConsolidado(ctx)
}

func (svc *Service) ObtenerActividadTenant(ctx context.Context, tenantID string) (*TenantDetalleActividad, error) {
	tenant, usuarios, movimientos, err := svc.store.ObtenerDetalleTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	for i := range usuarios {
		estado, act := svc.tracker.EstadoPresencia(usuarios[i].ID, usuarios[i].UltimoLoginAt)
		usuarios[i].EstadoPresencia = estado
		usuarios[i].UltimaActividadAt = act
	}

	return &TenantDetalleActividad{
		Tenant:             *tenant,
		Usuarios:           usuarios,
		UltimosMovimientos: movimientos,
	}, nil
}
