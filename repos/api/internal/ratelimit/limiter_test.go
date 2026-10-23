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

func TestLimitador_LimpiezaPerezosa(t *testing.T) {
	l := Nuevo(1, time.Minute)
	ahora := time.Now()
	l.ahora = func() time.Time { return ahora }
	for i := 0; i <= maxEntradas+1; i++ {
		l.conteos[string(rune(i))+"x"] = &ventanaConteo{inicio: ahora, conteo: 1}
	}
	ahora = ahora.Add(2 * time.Minute)
	l.Permitir("nueva")
	if len(l.conteos) != 1 {
		t.Fatalf("esperaba 1 entrada tras la limpieza, hay %d", len(l.conteos))
	}
}

func TestIPCliente(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.1:1234"
	if ip := IPCliente(r); ip != "10.0.0.1" {
		t.Fatalf("sin XFF: %q", ip)
	}
	r.Header.Set("X-Forwarded-For", "  203.0.113.7 , 10.1.1.1")
	if ip := IPCliente(r); ip != "203.0.113.7" {
		t.Fatalf("con XFF múltiple: %q", ip)
	}
	r.Header.Set("X-Forwarded-For", "   ")
	if ip := IPCliente(r); ip != "10.0.0.1" {
		t.Fatalf("con XFF vacío: %q", ip)
	}
}
