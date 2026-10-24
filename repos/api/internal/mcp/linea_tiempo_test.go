package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/evento"
)

var base = time.Date(2026, 10, 22, 14, 0, 0, 0, time.UTC)

func en(minutos float64) time.Time { return base.Add(time.Duration(minutos * float64(time.Minute))) }

func evt(mesaID string, version int, tipo, pedidoID, anterior, nuevo, actorTipo, nombre, rol string, ts time.Time) evento.Evento {
	e := evento.Evento{
		MesaID: mesaID, CuentaVersion: version, Tipo: tipo, EstadoAnterior: anterior, EstadoNuevo: nuevo,
		ActorTipo: actorTipo, ActorNombre: nombre, ActorRol: rol, TS: ts, Detalle: json.RawMessage(`{}`),
	}
	if pedidoID != "" {
		e.PedidoID = &pedidoID
	}
	return e
}

// eventosDePrueba: la mesa 1 recorre una cuenta completa entre las 14:00 y las 14:45 (UTC); la mesa 2 abre a las 15:00.
func eventosDePrueba() []evento.Evento {
	entregado := evt("m-1", 3, evento.TipoPedidoEstado, "p-1", "listo", "entregado", evento.ActorUsuario, "Laura", "mozo", en(28))
	entregado.Detalle = json.RawMessage(`{"modo_entrega":"retiro"}`)
	return []evento.Evento{
		evt("m-1", 3, evento.TipoCuentaAbierta, "p-1", "", "recibido", evento.ActorComensal, "Mateo", "", en(0)),
		evt("m-1", 3, evento.TipoPedidoCreado, "p-1", "", "recibido", evento.ActorComensal, "Mateo", "", en(0)),
		evt("m-1", 3, evento.TipoPedidoEstado, "p-1", "recibido", "confirmado", evento.ActorUsuario, "Laura", "mozo", en(2)),
		evt("m-1", 3, evento.TipoPedidoEstado, "p-1", "confirmado", "preparando", evento.ActorUsuario, "Carlos", "cocinero", en(10)),
		evt("m-1", 3, evento.TipoPedidoEstado, "p-1", "preparando", "listo", evento.ActorUsuario, "Carlos", "cocinero", en(25)),
		entregado,
		evt("m-1", 3, evento.TipoCuentaSolicitada, "", "abierta", "solicitada", evento.ActorComensal, "Mateo", "", en(40)),
		evt("m-1", 3, evento.TipoCuentaCerrada, "", "", "", evento.ActorUsuario, "Laura", "mozo", en(45)),
		evt("m-2", 1, evento.TipoCuentaAbierta, "p-2", "", "recibido", evento.ActorComensal, "Juani", "", en(60)),
		evt("m-2", 1, evento.TipoPedidoCreado, "p-2", "", "recibido", evento.ActorComensal, "Juani", "", en(60)),
	}
}

func (s *storeFalso) MesasSucursal(_ context.Context, _, sucursalID string) ([]MesaRef, error) {
	return s.mesasRef[sucursalID], s.falla
}

func (s *storeFalso) MesaPorNumero(_ context.Context, _, sucursalID string, numero int) (*MesaRef, error) {
	for _, m := range s.mesasRef[sucursalID] {
		if m.Numero == numero {
			return &m, nil
		}
	}
	return nil, ErrMesaNoEncontrada
}

func (s *storeFalso) PrimerEvento(_ context.Context, _, sucursalID string) (*time.Time, error) {
	var primero *time.Time
	for _, e := range s.eventos[sucursalID] {
		if ts := e.TS; primero == nil || ts.Before(*primero) {
			primero = &ts
		}
	}
	return primero, s.falla
}

func (s *storeFalso) EventosSucursal(_ context.Context, _, sucursalID string, desde, hasta time.Time) ([]evento.Evento, error) {
	resultado := []evento.Evento{}
	for _, e := range s.eventos[sucursalID] {
		if !e.TS.Before(desde) && e.TS.Before(hasta) {
			resultado = append(resultado, e)
		}
	}
	return resultado, s.falla
}

func (s *storeFalso) EventosMesa(ctx context.Context, tenantID, sucursalID, mesaID string, desde, hasta time.Time) ([]evento.Evento, error) {
	todos, err := s.EventosSucursal(ctx, tenantID, sucursalID, desde, hasta)
	resultado := []evento.Evento{}
	for _, e := range todos {
		if e.MesaID == mesaID {
			resultado = append(resultado, e)
		}
	}
	return resultado, err
}

func TestParsearMomento(t *testing.T) {
	ahora := time.Date(2026, 10, 22, 15, 30, 0, 0, time.UTC)
	validos := map[string]time.Time{
		"":                          ahora,
		"ahora":                     ahora,
		"now":                       ahora,
		"hace 60 min":               ahora.Add(-60 * time.Minute),
		"Hace 60 minutos":           ahora.Add(-60 * time.Minute),
		"hace 2 horas":              ahora.Add(-2 * time.Hour),
		"hace 1 hora":               ahora.Add(-time.Hour),
		"60m":                       ahora.Add(-60 * time.Minute),
		"2h":                        ahora.Add(-2 * time.Hour),
		" 90 min ":                  ahora.Add(-90 * time.Minute),
		"90 minutes ago":            ahora.Add(-90 * time.Minute),
		"2 hours ago":               ahora.Add(-2 * time.Hour),
		"hace 30 minutos atrás":     ahora.Add(-30 * time.Minute),
		"hace una hora":             ahora.Add(-time.Hour),
		"hace un minuto":            ahora.Add(-time.Minute),
		"hace 60 min.":              ahora.Add(-60 * time.Minute),
		"2026-10-22T11:00:00-03:00": time.Date(2026, 10, 22, 14, 0, 0, 0, time.UTC),
		"2026-10-22T14:00:00Z":      time.Date(2026, 10, 22, 14, 0, 0, 0, time.UTC),
		"2026-10-22T15:30:30Z":      ahora, // dentro de la tolerancia de reloj: se recorta a ahora
		"2026-10-22T14:00:00.250Z":  time.Date(2026, 10, 22, 14, 0, 0, 250_000_000, time.UTC),
	}
	for raw, esperado := range validos {
		got, err := parsearMomento(raw, ahora)
		if err != nil || !got.Equal(esperado) {
			t.Errorf("parsearMomento(%q) = %v, %v; quería %v", raw, got, err, esperado)
		}
	}

	invalidos := map[string]error{
		"ayer":                      ErrMomentoInvalido,
		"hace mucho":                ErrMomentoInvalido,
		"hace 5 años":               ErrMomentoInvalido,
		"hace 5 parsecs":            ErrMomentoInvalido,
		"-30m":                      ErrMomentoInvalido,
		"hace 99999999999999 h":     ErrMomentoInvalido,
		"hace 999999 horas":         ErrMomentoInvalido,
		"2026-10-22T11:00:00":       ErrMomentoInvalido, // sin zona
		"2026-10-22":                ErrMomentoInvalido,
		"22/10/2026 11:00":          ErrMomentoInvalido,
		"2026-10-22T20:00:00-03:00": ErrMomentoFuturo,
		"2027-01-01T00:00:00Z":      ErrMomentoFuturo,
	}
	for raw, esperado := range invalidos {
		if _, err := parsearMomento(raw, ahora); !errors.Is(err, esperado) {
			t.Errorf("parsearMomento(%q) err = %v; quería %v", raw, err, esperado)
		}
	}
}

func TestReconstruirMesas(t *testing.T) {
	mesas := []MesaRef{{ID: "m-2", Numero: 2}, {ID: "m-1", Numero: 1}, {ID: "m-3", Numero: 3}}
	eventos := eventosDePrueba()

	estado := func(momento time.Time, numero int) MesaEnMomento {
		t.Helper()
		for _, m := range reconstruirMesas(mesas, eventos, momento) {
			if m.Mesa == numero {
				return m
			}
		}
		t.Fatalf("falta la mesa %d", numero)
		return MesaEnMomento{}
	}

	// 14:12, el pedido lo tiene cocina desde las 14:10; la espera de comida cuenta desde que se creó (14:00), como estado_salon.
	m := estado(en(12), 1)
	if !m.CuentaAbierta || m.CuentaVersion == nil || *m.CuentaVersion != 3 || len(m.Pedidos) != 1 {
		t.Fatalf("t=14:12: %+v", m)
	}
	p := m.Pedidos[0]
	if p.PedidoID != "p-1" || p.Estado != "preparando" || p.MinutosEnEstado != 2 || p.QuienUltimoCambio != "Carlos" ||
		p.Rol == nil || *p.Rol != "cocinero" || p.DesdeTS != "2026-10-22T11:10:00-03:00" {
		t.Errorf("pedido a las 14:12: %+v", p)
	}
	if m.Espera == nil || *m.Espera != EsperaComida || m.MinutosEnEspera == nil || *m.MinutosEnEspera != 12 || m.CuentaSolicitada || m.PagoHabilitado {
		t.Errorf("espera a las 14:12: %+v", m)
	}

	// 14:01, recién creado por el comensal: sin rol.
	p = estado(en(1), 1).Pedidos[0]
	if p.Estado != "recibido" || p.QuienUltimoCambio != "Mateo" || p.Rol != nil || p.MinutosEnEstado != 1 {
		t.Errorf("pedido a las 14:01: %+v", p)
	}

	// 14:26, cocina terminó y falta marcar la entrega.
	m = estado(en(26), 1)
	if m.Espera == nil || *m.Espera != EsperaEntrega || *m.MinutosEnEspera != 1 || m.Pedidos[0].Estado != "listo" {
		t.Errorf("espera a las 14:26: %+v", m)
	}

	// 14:30, entregado: no genera espera.
	m = estado(en(30), 1)
	if m.Espera != nil || m.MinutosEnEspera != nil || m.Pedidos[0].Estado != "entregado" || m.Pedidos[0].QuienUltimoCambio != "Laura" {
		t.Errorf("a las 14:30: %+v", m)
	}

	// 14:41, pidió la cuenta: cobro gana y no tiene minutos.
	m = estado(en(41), 1)
	if !m.CuentaSolicitada || m.Espera == nil || *m.Espera != EsperaCobro || m.MinutosEnEspera != nil {
		t.Errorf("a las 14:41: %+v", m)
	}

	// 14:50, cuenta cerrada; antes del primer hito, sin cuenta.
	for _, momento := range []time.Time{en(50), en(-30)} {
		m = estado(momento, 1)
		if m.CuentaAbierta || m.CuentaVersion != nil || len(m.Pedidos) != 0 || m.Espera != nil || m.Pedidos == nil {
			t.Errorf("sin cuenta abierta en %v: %+v", momento, m)
		}
	}

	// El momento es inclusivo; la mesa sin eventos aparece cerrada y las mesas salen ordenadas por número.
	if m := estado(en(60), 2); !m.CuentaAbierta || m.Espera == nil || *m.Espera != EsperaComida {
		t.Errorf("mesa 2 a las 15:00: %+v", m)
	}
	if m := estado(en(60), 3); m.CuentaAbierta {
		t.Errorf("mesa 3 sin eventos: %+v", m)
	}
	if got := reconstruirMesas(mesas, eventos, en(60)); got[0].Mesa != 1 || got[1].Mesa != 2 || got[2].Mesa != 3 {
		t.Errorf("orden: %+v", got)
	}

	// Una cuenta nueva sobre la misma mesa reemplaza a la cerrada; un pago tardío de la anterior no la cierra ni la reabre.
	nueva := append([]evento.Evento{}, eventos...)
	nueva = append(nueva,
		evt("m-1", 4, evento.TipoCuentaAbierta, "p-9", "", "recibido", evento.ActorComensal, "Ana", "", en(50)),
		evt("m-1", 4, evento.TipoPedidoCreado, "p-9", "", "recibido", evento.ActorComensal, "Ana", "", en(50)),
		evt("m-1", 3, evento.TipoPagoAprobado, "", "", "aprobado", evento.ActorMercadoPago, "", "", en(51)),
	)
	m = reconstruirMesas(mesas[1:2], nueva, en(52))[0]
	if !m.CuentaAbierta || *m.CuentaVersion != 4 || len(m.Pedidos) != 1 || m.Pedidos[0].PedidoID != "p-9" {
		t.Errorf("cuenta nueva: %+v", m)
	}
	// Con solo el pago tardío de la cuenta 3 después del cierre, la mesa sigue cerrada.
	soloPago := append(append([]evento.Evento{}, eventos...), evt("m-1", 3, evento.TipoPagoAprobado, "", "", "aprobado", evento.ActorMercadoPago, "", "", en(51)))
	if m := reconstruirMesas(mesas[1:2], soloPago, en(52))[0]; m.CuentaAbierta {
		t.Errorf("pago tardío no debe reabrir: %+v", m)
	}

	// pago habilitado y luego deshabilitado.
	hab := append([]evento.Evento{}, eventos[:5]...)
	hab = append(hab, evt("m-1", 3, evento.TipoPagoHabilitado, "", "deshabilitado", "habilitado", evento.ActorUsuario, "Laura", "mozo", en(26)))
	if m := reconstruirMesas(mesas[1:2], hab, en(27))[0]; !m.PagoHabilitado || *m.Espera != EsperaCobro {
		t.Errorf("pago habilitado: %+v", m)
	}
	hab = append(hab, evt("m-1", 3, evento.TipoPagoHabilitado, "", "habilitado", "deshabilitado", evento.ActorUsuario, "Laura", "mozo", en(28)))
	if m := reconstruirMesas(mesas[1:2], hab, en(29))[0]; m.PagoHabilitado || *m.Espera != EsperaEntrega {
		t.Errorf("pago deshabilitado: %+v", m)
	}
}

// La cuenta abierta sigue la regla de estado_salon (derivarMesa): sin pedidos, o con todos
// cerrados y sin cuenta pedida ni pago habilitado, la mesa está libre.
func TestReconstruirMesas_MismaReglaQueSalon(t *testing.T) {
	mesas := []MesaRef{{ID: "m-1", Numero: 1}}

	// El mozo habilita el pago de una cuenta nueva que todavía no tiene pedidos.
	soloPago := append(eventosDePrueba()[:8:8],
		evt("m-1", 4, evento.TipoPagoHabilitado, "", "deshabilitado", "habilitado", evento.ActorUsuario, "Laura", "mozo", en(50)))
	if m := reconstruirMesas(mesas, soloPago, en(51))[0]; m.CuentaAbierta || m.Espera != nil || m.PagoHabilitado {
		t.Errorf("pago habilitado sin pedidos no abre cuenta: %+v", m)
	}

	// Todos los pedidos cerrados y sin cuenta pedida: libre.
	cerrado := append(eventosDePrueba()[:5:5],
		evt("m-1", 3, evento.TipoPedidoEstado, "p-1", "listo", "cerrado", evento.ActorUsuario, "Laura", "mozo", en(30)))
	if m := reconstruirMesas(mesas, cerrado, en(31))[0]; m.CuentaAbierta || m.CuentaVersion != nil || len(m.Pedidos) != 0 {
		t.Errorf("pedidos cerrados sin cuenta pedida: %+v", m)
	}
}

func TestMesasAReconstruir(t *testing.T) {
	mesas := []MesaRef{
		{ID: "m-1", Numero: 1, Activa: true},
		{ID: "m-baja-con-hitos", Numero: 5},
		{ID: "m-baja", Numero: 6},
	}
	eventos := []evento.Evento{evt("m-baja-con-hitos", 1, evento.TipoCuentaAbierta, "", "", "", evento.ActorComensal, "Ana", "", en(0))}
	got := mesasAReconstruir(mesas, eventos)
	if len(got) != 2 || got[0].ID != "m-1" || got[1].ID != "m-baja-con-hitos" {
		t.Errorf("se esperaban la activa y la dada de baja con hitos: %+v", got)
	}
}

func TestCalcularEtapas(t *testing.T) {
	etapas := calcularEtapas(eventosDePrueba())
	if len(etapas) != 2 || etapas[0].PedidoID != "p-1" || etapas[1].PedidoID != "p-2" {
		t.Fatalf("etapas: %+v", etapas)
	}
	f := func(v float64) *float64 { return &v }
	retiro := "retiro"
	esperada := EtapasPedido{
		PedidoID: "p-1", MinutosHastaConfirmacion: f(2), MinutosEsperaCocina: f(8), MinutosPreparacion: f(15),
		MinutosAEntrega: f(3), ModoEntrega: &retiro,
	}
	if !reflect.DeepEqual(etapas[0], esperada) {
		t.Errorf("p-1: %+v", etapas[0])
	}
	if !reflect.DeepEqual(etapas[1], EtapasPedido{PedidoID: "p-2"}) {
		t.Errorf("p-2 sin hitos posteriores debe quedar todo en null: %+v", etapas[1])
	}

	// Un rango que arranca después de la creación deja nulas solo las etapas que dependen de ella.
	recortado := calcularEtapas(eventosDePrueba()[3:6])
	if len(recortado) != 1 || recortado[0].MinutosHastaConfirmacion != nil || recortado[0].MinutosEsperaCocina != nil ||
		recortado[0].MinutosPreparacion == nil || *recortado[0].MinutosPreparacion != 15 {
		t.Errorf("recortado: %+v", recortado)
	}

	hitos := construirHitos(eventosDePrueba()[:4])
	if hitos[0].MinutosDesdeAnterior != nil || *hitos[2].MinutosDesdeAnterior != 2 || *hitos[3].MinutosDesdeAnterior != 8 ||
		hitos[3].Quien != "Carlos" || *hitos[3].Rol != "cocinero" || hitos[0].Rol != nil || hitos[0].TS != "2026-10-22T11:00:00-03:00" {
		t.Errorf("hitos: %+v", hitos)
	}
}

func TestToolsLineaDeTiempo(t *testing.T) {
	h := nuevoHandlerDePrueba(t, nuevoStoreFalso()) // ahora = 15:30 UTC (12:30 en el negocio)

	t.Run("estado_mesas sin momento es ahora", func(t *testing.T) {
		var out SalidaEstadoMesas
		salidaTool(t, llamarTool(t, h, "t-a", "estado_mesas", nil), &out)
		if out.Momento != out.TS || out.SucursalID != "s-a" || out.HistorialDesde != "2026-10-22T11:00:00-03:00" || len(out.Mesas) != 2 {
			t.Fatalf("%+v", out)
		}
		if out.Mesas[0].CuentaAbierta || !out.Mesas[1].CuentaAbierta || out.Mesas[1].Pedidos[0].PedidoID != "p-2" {
			t.Errorf("mesas: %+v", out.Mesas)
		}
		if out.Resumen != "Centro el 2026-10-22T12:30:00-03:00: 1 de 2 mesas con cuenta abierta. Mesas esperando: comida 1, entrega 0, cobro 0." {
			t.Errorf("resumen: %q", out.Resumen)
		}
	})

	t.Run("estado_mesas hace 90 min filtrando mesa", func(t *testing.T) {
		var out SalidaEstadoMesas
		salidaTool(t, llamarTool(t, h, "t-a", "estado_mesas", map[string]any{"momento": "hace 90 min", "mesa": 1}), &out)
		if len(out.Mesas) != 1 || out.Mesas[0].Mesa != 1 || !out.Mesas[0].CuentaAbierta || out.Mesas[0].Pedidos[0].Estado != "recibido" ||
			out.Momento != "2026-10-22T11:00:00-03:00" {
			t.Errorf("%+v", out)
		}
	})

	errores := []struct {
		nombre, tool, codigo string
		args                 map[string]any
		tenant               string
	}{
		{"momento inválido", "estado_mesas", CodigoParametroInvalido, map[string]any{"momento": "ayer"}, "t-a"},
		{"momento futuro", "estado_mesas", CodigoParametroInvalido, map[string]any{"momento": "2027-01-01T00:00:00Z"}, "t-a"},
		{"antes del primer hito", "estado_mesas", CodigoSinDatos, map[string]any{"momento": "hace 3 horas"}, "t-a"},
		{"mesa inexistente", "estado_mesas", CodigoMesaNoEncontrada, map[string]any{"mesa": 9}, "t-a"},
		{"historial mesa inexistente", "historial_mesa", CodigoMesaNoEncontrada, map[string]any{"mesa": 9}, "t-a"},
		{"historial desde inválido", "historial_mesa", CodigoParametroInvalido, map[string]any{"mesa": 1, "desde": "x"}, "t-a"},
		{"historial desde después de hasta", "historial_mesa", CodigoParametroInvalido, map[string]any{"mesa": 1, "desde": "hace 1 hora", "hasta": "hace 2 horas"}, "t-a"},
		{"sucursal ajena", "historial_mesa", CodigoSucursalNoEncontrada, map[string]any{"mesa": 9, "sucursal_id": "s-a"}, "t-b"},
		{"historial rango mayor a 7 días", "historial_mesa", CodigoParametroInvalido, map[string]any{"mesa": 1, "desde": "hace 200 horas"}, "t-a"},
		{"historial antes del primer hito", "historial_mesa", CodigoSinDatos, map[string]any{"mesa": 1, "desde": "hace 5 horas", "hasta": "hace 3 horas"}, "t-a"},
	}
	for _, c := range errores {
		t.Run(c.nombre, func(t *testing.T) {
			e := errorEstructurado(t, llamarTool(t, h, c.tenant, c.tool, c.args))
			if e.Error != c.codigo {
				t.Errorf("error %q, quería %q: %+v", e.Error, c.codigo, e)
			}
		})
	}

	t.Run("estado_mesas de una mesa dada de baja", func(t *testing.T) {
		var out SalidaEstadoMesas
		salidaTool(t, llamarTool(t, h, "t-a", "estado_mesas", map[string]any{"mesa": 7}), &out)
		if len(out.Mesas) != 1 || out.Mesas[0].Mesa != 7 || out.Mesas[0].CuentaAbierta {
			t.Errorf("%+v", out)
		}
	})

	t.Run("sin_datos indica desde cuándo hay historial", func(t *testing.T) {
		e := errorEstructurado(t, llamarTool(t, h, "t-a", "estado_mesas", map[string]any{"momento": "hace 3 horas"}))
		detalle, _ := e.Detalle.(map[string]any)
		if detalle["historial_desde"] != "2026-10-22T11:00:00-03:00" {
			t.Errorf("detalle: %+v", e.Detalle)
		}
	})

	t.Run("historial_mesa", func(t *testing.T) {
		var out SalidaHistorialMesa
		salidaTool(t, llamarTool(t, h, "t-a", "historial_mesa", map[string]any{"mesa": 1}), &out)
		if out.Mesa != 1 || len(out.Hitos) != 8 || len(out.Etapas) != 1 || out.Hasta != out.TS || out.Desde != "2026-10-22T00:30:00-03:00" {
			t.Fatalf("%+v", out)
		}
		ultimaEtapa := out.Etapas[0]
		if *ultimaEtapa.MinutosPreparacion != 15 || *ultimaEtapa.ModoEntrega != "retiro" {
			t.Errorf("etapas: %+v", ultimaEtapa)
		}
		entregado := out.Hitos[5]
		if entregado.EstadoNuevo == nil || *entregado.EstadoNuevo != "entregado" || *entregado.ModoEntrega != "retiro" || entregado.Quien != "Laura" {
			t.Errorf("hito entregado: %+v", entregado)
		}
		// Un rango más chico deja solo lo que cae adentro.
		var recorte SalidaHistorialMesa
		salidaTool(t, llamarTool(t, h, "t-a", "historial_mesa", map[string]any{"mesa": 1, "desde": "2026-10-22T14:30:00Z", "hasta": "2026-10-22T14:42:00Z"}), &recorte)
		if len(recorte.Hitos) != 1 || recorte.Hitos[0].Hito != evento.TipoCuentaSolicitada {
			t.Errorf("recorte: %+v", recorte.Hitos)
		}
	})
}
