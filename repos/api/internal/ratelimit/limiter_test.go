package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLimitador_PermiteHastaElLimiteYBloquea(t *testing.T) {
	l := Nuevo(3, time.Minute)
	for i := 0; i < 3; i++ {
		if !l.Permitir("a") {
			t.Fatalf("la solicitud %d debería pasar", i+1)
		}
	}
	if l.Permitir("a") {
		t.Fatal("la cuarta solicitud debería bloquearse")
	}
}

func TestLimitador_ReiniciaAlPasarLaVentana(t *testing.T) {
	l := Nuevo(1, time.Minute)
	ahora := time.Now()
	l.ahora = func() time.Time { return ahora }
	l.Permitir("a")
	if l.Permitir("a") {
		t.Fatal("debería bloquear dentro de la ventana")
	}
	ahora = ahora.Add(time.Minute)
	if !l.Permitir("a") {
		t.Fatal("debería permitir tras vencer la ventana")
	}
}

func TestLimitador_ClavesIndependientes(t *testing.T) {
	l := Nuevo(1, time.Minute)
	l.Permitir("a")
	if !l.Permitir("b") {
		t.Fatal("otra clave no debería verse afectada")
	}
}

func TestLimitador_PurgaVencidasAlLlenarse(t *testing.T) {
	l := Nuevo(1, time.Minute)
	l.maxClaves = 3
	ahora := time.Now()
	l.ahora = func() time.Time { return ahora }
	for _, k := range []string{"a", "b", "c"} {
		l.Permitir(k)
	}
	ahora = ahora.Add(2 * time.Minute)
	if !l.Permitir("nueva") {
		t.Fatal("debería purgar las vencidas y admitir la clave nueva")
	}
	if len(l.conteos) != 1 {
		t.Fatalf("esperaba 1 entrada tras la purga, hay %d", len(l.conteos))
	}
}

func TestLimitador_MapaLlenoDesalojaLaMasViejaYAceptaLaNueva(t *testing.T) {
	l := Nuevo(5, time.Minute)
	l.maxClaves = 2
	ahora := time.Now()
	l.ahora = func() time.Time { return ahora }
	l.Permitir("a")
	ahora = ahora.Add(time.Second)
	l.Permitir("b")
	ahora = ahora.Add(time.Second)
	if !l.Permitir("c") {
		t.Fatal("con el mapa lleno debería admitir la clave nueva")
	}
	if len(l.conteos) != 2 {
		t.Fatalf("el mapa no debería crecer: %d", len(l.conteos))
	}
	if _, ok := l.conteos["a"]; ok {
		t.Fatal("debería haberse desalojado la entrada más vieja (a)")
	}
	if _, ok := l.conteos["b"]; !ok {
		t.Fatal("b debería conservarse")
	}
}

func TestIPCliente(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.1:1234"
	if ip := IPCliente(r); ip != "10.0.0.1" {
		t.Fatalf("sin XFF: %q", ip)
	}
	r.Header.Set("X-Forwarded-For", "  6.6.6.6 , 203.0.113.7")
	if ip := IPCliente(r); ip != "203.0.113.7" {
		t.Fatalf("con XFF múltiple debe tomar el último: %q", ip)
	}
	r.Header.Set("X-Forwarded-For", "203.0.113.7, ")
	if ip := IPCliente(r); ip != "203.0.113.7" {
		t.Fatalf("con último valor vacío: %q", ip)
	}
	r.Header.Set("X-Forwarded-For", "   ")
	if ip := IPCliente(r); ip != "10.0.0.1" {
		t.Fatalf("con XFF vacío: %q", ip)
	}
	r.Header.Set("X-Forwarded-For", "no-es-ip")
	if ip := IPCliente(r); ip != "10.0.0.1" {
		t.Fatalf("con XFF no IP: %q", ip)
	}
}

func TestIPCliente_CFConnectingIP(t *testing.T) {
	nuevo := func(secretoHeader string) *http.Request {
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = "10.0.0.1:1234"
		r.Header.Set("X-Forwarded-For", "6.6.6.6, 203.0.113.7")
		r.Header.Set("CF-Connecting-IP", " 198.51.100.9 ")
		if secretoHeader != "" {
			r.Header.Set("X-Fudd-Proxy-Secret", secretoHeader)
		}
		return r
	}

	t.Setenv("CF_PROXY_SECRET", "")
	if ip := IPCliente(nuevo("")); ip != "203.0.113.7" {
		t.Fatalf("sin env se ignora CF-Connecting-IP: %q", ip)
	}
	if ip := IPCliente(nuevo("x")); ip != "203.0.113.7" {
		t.Fatalf("sin env, un secreto vacío/ajeno no habilita el header: %q", ip)
	}

	t.Setenv("CF_PROXY_SECRET", "s3creto")
	if ip := IPCliente(nuevo("")); ip != "203.0.113.7" {
		t.Fatalf("sin header secreto se ignora: %q", ip)
	}
	if ip := IPCliente(nuevo("incorrecto")); ip != "203.0.113.7" {
		t.Fatalf("con secreto incorrecto se ignora: %q", ip)
	}
	if ip := IPCliente(nuevo("s3creto")); ip != "198.51.100.9" {
		t.Fatalf("con secreto correcto se usa CF-Connecting-IP: %q", ip)
	}
	r := nuevo("s3creto")
	r.Header.Set("CF-Connecting-IP", "basura")
	if ip := IPCliente(r); ip != "203.0.113.7" {
		t.Fatalf("CF-Connecting-IP no IP se ignora: %q", ip)
	}
}

func TestClaveCliente(t *testing.T) {
	clave := func(addr string) string {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = addr
		return ClaveCliente(req)
	}

	if a, b := clave("[2001:db8:1:2::1]:1234"), clave("[2001:db8:1:2:ffff::9]:1234"); a != b {
		t.Fatalf("mismo /64 debería dar la misma clave: %q vs %q", a, b)
	}
	if a, b := clave("[2001:db8:1:2::1]:1234"), clave("[2001:db8:1:3::1]:1234"); a == b {
		t.Fatalf("/64 distintos deberían dar claves distintas: %q", a)
	}
	if got := clave("203.0.113.7:1234"); got != "203.0.113.7" {
		t.Fatalf("IPv4 debería quedar igual, got %q", got)
	}
	if got := clave("[::ffff:203.0.113.7]:1234"); got != "::ffff:203.0.113.7" {
		t.Fatalf("IPv4-mapped debería quedar sin máscara /64, got %q", got)
	}
}
