package tenant

import (
	"context"
	"sync"
	"time"
)

const ttlFuncionalidades = 60 * time.Second

type claveFuncionalidad struct {
	tenantID string
	recurso  Recurso
}

type entradaFuncionalidad struct {
	habilitado bool
	vence      time.Time
}

// cacheFuncionalidades evita consultar la DB en cada tool de MCP; el TTL corto
// hace que un cambio de plan o de excepción se aplique sin deploy.
type cacheFuncionalidades struct {
	mu       sync.Mutex
	entradas map[claveFuncionalidad]entradaFuncionalidad
}

func (c *cacheFuncionalidades) obtener(k claveFuncionalidad, ahora time.Time) (bool, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entradas[k]
	if !ok || !ahora.Before(e.vence) {
		return false, false
	}
	return e.habilitado, true
}

func (c *cacheFuncionalidades) guardar(k claveFuncionalidad, habilitado bool, ahora time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entradas == nil {
		c.entradas = make(map[claveFuncionalidad]entradaFuncionalidad)
	}
	c.entradas[k] = entradaFuncionalidad{habilitado: habilitado, vence: ahora.Add(ttlFuncionalidades)}
}

func (svc *Service) reloj() time.Time {
	if svc.ahora != nil {
		return svc.ahora()
	}
	return time.Now()
}

// AccesoFuncionalidad resuelve si el negocio puede usar el recurso: la
// excepción vigente del negocio manda; si no hay, el plan efectivo (que ya
// contempla la prueba Pro). Los errores no se cachean.
func (svc *Service) AccesoFuncionalidad(ctx context.Context, tenantID string, rec Recurso) (bool, error) {
	ahora := svc.reloj()
	clave := claveFuncionalidad{tenantID: tenantID, recurso: rec}
	if habilitado, ok := svc.funcionalidades.obtener(clave, ahora); ok {
		return habilitado, nil
	}
	cuotas, err := svc.store.ObtenerEstadoCuotas(ctx, tenantID)
	if err != nil {
		return false, err
	}
	habilitado, err := svc.store.FuncionalidadHabilitada(ctx, tenantID, rec, cuotas.PlanEfectivo(ahora), ahora)
	if err != nil {
		return false, err
	}
	svc.funcionalidades.guardar(clave, habilitado, ahora)
	return habilitado, nil
}

// AccesoMCP adapta AccesoFuncionalidad a las interfaces de oauth y mcp.
func (svc *Service) AccesoMCP(ctx context.Context, tenantID string) (bool, error) {
	return svc.AccesoFuncionalidad(ctx, tenantID, RecursoMCP)
}
