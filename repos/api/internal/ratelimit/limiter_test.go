package ratelimit

import (
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

func TestLimitador_MapaLlenoRechazaClavesNuevas(t *testing.T) {
	l := Nuevo(5, time.Minute)
	l.maxClaves = 2
	l.Permitir("a")
	l.Permitir("b")
	if l.Permitir("c") {
		t.Fatal("con el mapa lleno de claves vigentes no debería admitir una nueva")
	}
	if !l.Permitir("a") {
		t.Fatal("una clave existente debería seguir admitida")
	}
	if len(l.conteos) != 2 {
		t.Fatalf("el mapa no debería crecer: %d", len(l.conteos))
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
	r.Header.Set("X-Forwarded-For", "203.0.113.7")
	r.Header.Set("CF-Connecting-IP", " 198.51.100.9 ")
	if ip := IPCliente(r); ip != "198.51.100.9" {
		t.Fatalf("CF-Connecting-IP debe tener prioridad: %q", ip)
	}
	r.Header.Set("CF-Connecting-IP", "basura")
	if ip := IPCliente(r); ip != "203.0.113.7" {
		t.Fatalf("CF-Connecting-IP no IP se ignora: %q", ip)
	}
}
