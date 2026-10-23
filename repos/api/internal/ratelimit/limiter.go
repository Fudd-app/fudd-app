package ratelimit

import (
	"crypto/subtle"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const maxClaves = 10000

type ventanaConteo struct {
	inicio time.Time
	conteo int
}

// Limitador de ventana fija en memoria. Alcanza para una sola instancia (Render);
// si se escala horizontalmente hay que moverlo a un store compartido.
type Limitador struct {
	mu        sync.Mutex
	limite    int
	ventana   time.Duration
	conteos   map[string]*ventanaConteo
	ahora     func() time.Time
	maxClaves int
}

func Nuevo(limite int, ventana time.Duration) *Limitador {
	return &Limitador{
		limite:    limite,
		ventana:   ventana,
		conteos:   make(map[string]*ventanaConteo),
		ahora:     time.Now,
		maxClaves: maxClaves,
	}
}

// Permitir devuelve true si todavía hay cupo en la ventana actual de la clave.
func (l *Limitador) Permitir(clave string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	ahora := l.ahora()

	v, ok := l.conteos[clave]
	if !ok && len(l.conteos) >= l.maxClaves {
		l.purgarVencidas(ahora)
		if len(l.conteos) >= l.maxClaves {
			// Nunca se rechaza por mapa lleno (un atacante rotando IPs bloquearía a
			// los clientes nuevos): se desaloja la entrada más vieja.
			l.desalojarMasVieja()
		}
	}
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

func (l *Limitador) purgarVencidas(ahora time.Time) {
	for k, v := range l.conteos {
		if ahora.Sub(v.inicio) >= l.ventana {
			delete(l.conteos, k)
		}
	}
}

func (l *Limitador) desalojarMasVieja() {
	var claveVieja string
	var inicioViejo time.Time
	primera := true
	for k, v := range l.conteos {
		if primera || v.inicio.Before(inicioViejo) {
			claveVieja, inicioViejo, primera = k, v.inicio, false
		}
	}
	if !primera {
		delete(l.conteos, claveVieja)
	}
}

// IPCliente confía en CF-Connecting-IP solo si el request trae el secreto
// compartido en X-Fudd-Proxy-Secret (Cloudflare lo agrega con una Transform Rule):
// sin él, un atacante que le pegue directo al borde de Render puede falsificar el
// header. Si no, usa el último valor de X-Forwarded-For, que es el que agrega el
// proxy de Render (el primero lo controla el cliente y permitiría evadir el límite).
func IPCliente(r *http.Request) string {
	if secreto := os.Getenv("CF_PROXY_SECRET"); secreto != "" &&
		subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Fudd-Proxy-Secret")), []byte(secreto)) == 1 {
		if ip := strings.TrimSpace(r.Header.Get("CF-Connecting-IP")); net.ParseIP(ip) != nil {
			return ip
		}
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		valores := strings.Split(xff, ",")
		for i := len(valores) - 1; i >= 0; i-- {
			ip := strings.TrimSpace(valores[i])
			if ip == "" {
				continue
			}
			if net.ParseIP(ip) != nil {
				return ip
			}
			break
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
