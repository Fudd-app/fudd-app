package ratelimit

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const maxEntradas = 10000

type ventanaConteo struct {
	inicio time.Time
	conteo int
}

// Limitador de ventana fija en memoria. Alcanza para una sola instancia (Render);
// si se escala horizontalmente hay que moverlo a un store compartido.
type Limitador struct {
	mu      sync.Mutex
	limite  int
	ventana time.Duration
	conteos map[string]*ventanaConteo
	ahora   func() time.Time
}

func Nuevo(limite int, ventana time.Duration) *Limitador {
	return &Limitador{
		limite:  limite,
		ventana: ventana,
		conteos: make(map[string]*ventanaConteo),
		ahora:   time.Now,
	}
}

// Permitir devuelve true si todavía hay cupo en la ventana actual de la clave.
func (l *Limitador) Permitir(clave string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	ahora := l.ahora()
	if len(l.conteos) > maxEntradas {
		for k, v := range l.conteos {
			if ahora.Sub(v.inicio) >= l.ventana {
				delete(l.conteos, k)
			}
		}
	}

	v, ok := l.conteos[clave]
	if !ok || ahora.Sub(v.inicio) >= l.ventana {
		l.conteos[clave] = &ventanaConteo{inicio: ahora, conteo: 1}
		return l.limite >= 1
	}
	if v.conteo >= l.limite {
		return false
	}
	v.conteo++
	return true
}

// IPCliente toma el primer valor de X-Forwarded-For (hay proxy delante) y, si
// no está, el host de RemoteAddr.
func IPCliente(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		primero, _, _ := strings.Cut(xff, ",")
		if ip := strings.TrimSpace(primero); ip != "" {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
