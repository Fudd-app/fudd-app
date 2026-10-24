package platform

import (
	"sync"
	"time"
)

type Tracker struct {
	mu        sync.RWMutex
	actividad map[string]time.Time
}

var InstanciaTracker = NuevoTracker()

func NuevoTracker() *Tracker {
	return &Tracker{
		actividad: make(map[string]time.Time),
	}
}

func (t *Tracker) RegistrarActividad(usuarioID string) {
	if usuarioID == "" {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.actividad[usuarioID] = time.Now()
}

func (t *Tracker) EstadoPresencia(usuarioID string, ultimoLogin *time.Time) (estado string, ultima *time.Time) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	act, tieneActividad := t.actividad[usuarioID]
	if tieneActividad {
		diferencia := time.Since(act)
		if diferencia <= 5*time.Minute {
			return "online", &act
		}
		if diferencia <= 30*time.Minute {
			return "inactivo", &act
		}
		return "desconectado", &act
	}

	if ultimoLogin != nil {
		diferencia := time.Since(*ultimoLogin)
		if diferencia <= 5*time.Minute {
			return "online", ultimoLogin
		}
		if diferencia <= 30*time.Minute {
			return "inactivo", ultimoLogin
		}
		return "desconectado", ultimoLogin
	}

	return "desconectado", nil
}

func (t *Tracker) TotalUsuariosOnline() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	ahora := time.Now()
	online := 0
	for _, act := range t.actividad {
		if ahora.Sub(act) <= 5*time.Minute {
			online++
		}
	}
	return online
}
