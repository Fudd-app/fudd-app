package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/jsonschema-go/jsonschema"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/metrica"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/oauth"
)

type storeFalso struct {
	sucursales map[string][]SucursalResumen // por tenant
	mesas      map[string][]MesaEstado      // por sucursal
	pedidos    map[string][]PedidoCocina
	hora       UltimaHora
	porHora    []HoraTurno
	comensales int
	falla      error
}

func (s *storeFalso) SucursalDelTenant(_ context.Context, tenantID, sucursalID string) (*SucursalRef, error) {
	for _, su := range s.sucursales[tenantID] {
		if su.ID == sucursalID {
			return &SucursalRef{ID: su.ID, Nombre: su.Nombre}, nil
		}
	}
	return nil, ErrSucursalNoEncontrada
}

func (s *storeFalso) SucursalesDelTenant(_ context.Context, tenantID string) ([]SucursalResumen, error) {
	return s.sucursales[tenantID], s.falla
}

func (s *storeFalso) EstadoSalon(_ context.Context, _, sucursalID string, _ time.Time) ([]MesaEstado, error) {
	return s.mesas[sucursalID], s.falla
}

func (s *storeFalso) PedidosCocina(_ context.Context, _, sucursalID string, _ time.Time) ([]PedidoCocina, error) {
	return s.pedidos[sucursalID], s.falla
}

func (s *storeFalso) UltimaHoraCocina(context.Context, string, string, time.Time) (UltimaHora, error) {
	return s.hora, s.falla
}

func (s *storeFalso) ResumenPorHora(context.Context, string, string, time.Time, time.Time, string) ([]HoraTurno, error) {
	return s.porHora, s.falla
}

func (s *storeFalso) ComensalesAtendidos(context.Context, string, string, time.Time, time.Time) (int, error) {
	return s.comensales, nil
}

// accesoFalso decide el acceso a MCP por tenant; por defecto lo permite.
type accesoFalso struct {
	denegados map[string]bool
	err       error
}

func (a *accesoFalso) AccesoMCP(_ context.Context, tenantID string) (bool, error) {
	return !a.denegados[tenantID], a.err
}

func permiteTodo() *accesoFalso { return &accesoFalso{} }

type metricasFalsas struct{ desde, hasta string }

func (m *metricasFalsas) ObtenerResumen(_ context.Context, _, _, desde, hasta string) (*metrica.Resumen, error) {
	m.desde, m.hasta = desde, hasta
	return &metrica.Resumen{ResumenBase: metrica.ResumenBase{
		FacturacionTotal: 1500, TicketPromedio: 500, PedidosTotales: 4, PedidosCerrados: 3, TiempoPromedioDespachoMinutos: 7.5,
	}}, nil
}

func ptr[T any](v T) *T { return &v }

// authDePrueba inyecta claims desde cabeceras X-Tenant/X-Usuario en lugar de un token MCP real.
func authDePrueba(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := &auth.Claims{UsuarioID: r.Header.Get("X-Usuario"), TenantID: r.Header.Get("X-Tenant"), Rol: auth.RolDuenio}
		next.ServeHTTP(w, r.WithContext(auth.ContextConClaims(r.Context(), claims)))
	})
}

// horarioTodosLosDias arma sucursales.horarios con los mismos tramos todos los días, como el front.
func horarioTodosLosDias(tramos string) []byte {
	dias := make([]string, len(diasHorarios))
	for i, d := range diasHorarios {
		dias[i] = `"` + d + `":{"abierto":true,"tramos":` + tramos + `}`
	}
	return []byte("{" + strings.Join(dias, ",") + "}")
}

func nuevoStoreFalso() *storeFalso {
	return &storeFalso{
		sucursales: map[string][]SucursalResumen{
			"t-a": {{ID: "s-a", Nombre: "Centro", MesasTotales: 3, MesasActivas: 2, Horarios: horarioTodosLosDias(`[{"apertura":"08:00","cierre":"00:00"}]`)}},
			"t-b": {{ID: "s-b1", Nombre: "Norte", MesasTotales: 1, MesasActivas: 1}, {ID: "s-b2", Nombre: "Sur", MesasTotales: 3, MesasActivas: 3}},
			"t-c": {},
		},
		mesas: map[string][]MesaEstado{
			"s-a": {
				{Numero: 1, Estado: MesaOcupada, Comensales: 3, PedidosActivos: 1, TotalParcial: 100,
					Espera: ptr(EsperaComida), MinutosEnEspera: ptr(12), TomadoPor: []string{"Ana", TomadorQR}},
				{Numero: 2, Estado: MesaLibre, TomadoPor: []string{}},
				{Numero: 4, Estado: MesaCuentaSolicitada, Comensales: 2, PedidosActivos: 1, TotalParcial: 50,
					Espera: ptr(EsperaCobro), TomadoPor: []string{TomadorQR}},
			},
			"s-b1": {{Numero: 9, Estado: MesaCuentaSolicitada, Comensales: 2, Espera: ptr(EsperaCobro), TomadoPor: []string{TomadorQR}}},
		},
		pedidos: map[string][]PedidoCocina{
			"s-a": {
				{PedidoID: "p1", Mesa: 1, Estado: "recibido", MinutosDesdeIngreso: 4, Items: ItemsPedido{Pendientes: 2}},
				{PedidoID: "p2", Mesa: 2, Estado: "preparando", MinutosDesdeIngreso: 12, Items: ItemsPedido{Preparando: 1, Listos: 1}},
				{PedidoID: "p3", Mesa: 3, Estado: "listo", MinutosDesdeIngreso: 30, Items: ItemsPedido{Listos: 2}},
			},
		},
		hora: UltimaHora{Ingresados60Min: 5, Listos60Min: 3, DespachoPromedioMin: ptr(6.0), DespachoP90Min: ptr(11.5)},
		porHora: []HoraTurno{
			{Hora: "11:00", Pedidos: 1, Comensales: 2, Facturacion: 500, DespachoPromedioMin: ptr(7.0)},
			{Hora: "12:00", Pedidos: 3, Comensales: 4, Facturacion: 1000},
		},
		comensales: 11,
	}
}

func nuevoHandlerDePrueba(t *testing.T, st *storeFalso) http.Handler {
	t.Helper()
	t.Setenv("MCP_HABILITADO", "true")
	return nuevoHandler(authDePrueba, Dependencias{
		Store:    st,
		Metricas: &metricasFalsas{},
		Acceso:   permiteTodo(),
		Ahora:    func() time.Time { return time.Date(2026, 10, 22, 15, 30, 0, 0, time.UTC) },
	})
}

type respuestaRPC struct {
	Result struct {
		Tools []struct {
			Name         string          `json:"name"`
			OutputSchema json.RawMessage `json:"outputSchema"`
			Annotations  struct {
				ReadOnlyHint  *bool `json:"readOnlyHint"`
				OpenWorldHint *bool `json:"openWorldHint"`
			} `json:"annotations"`
		} `json:"tools"`
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		StructuredContent json.RawMessage `json:"structuredContent"`
		IsError           bool            `json:"isError"`
		ServerInfo        struct {
			Name string `json:"name"`
		} `json:"serverInfo"`
		Instructions string `json:"instructions"`
	} `json:"result"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func rpc(t *testing.T, h http.Handler, tenant, usuario, metodo string, params any) (*httptest.ResponseRecorder, respuestaRPC) {
	t.Helper()
	cuerpo, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": metodo, "params": params})
	r := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(string(cuerpo)))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json, text/event-stream")
	r.Header.Set("X-Tenant", tenant)
	r.Header.Set("X-Usuario", usuario)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	var resp respuestaRPC
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("respuesta no es JSON: %v: %s", err, rec.Body)
		}
	}
	return rec, resp
}

func llamarTool(t *testing.T, h http.Handler, tenant, nombre string, args map[string]any) respuestaRPC {
	t.Helper()
	rec, resp := rpc(t, h, tenant, "u-"+tenant, "tools/call", map[string]any{"name": nombre, "arguments": args})
	if rec.Code != http.StatusOK {
		t.Fatalf("%s: status %d: %s", nombre, rec.Code, rec.Body)
	}
	return resp
}

func TestHandlerDeshabilitadoDevuelve404(t *testing.T) {
	t.Setenv("MCP_HABILITADO", "")
	h := nuevoHandler(authDePrueba, Dependencias{Store: nuevoStoreFalso()})
	rec, _ := rpc(t, h, "t-a", "u-1", "tools/list", nil)
	if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), `"no encontrado"`) {
		t.Errorf("esperaba 404, obtuve %d %s", rec.Code, rec.Body)
	}
	t.Setenv("MCP_HABILITADO", "1")
	rec, _ = rpc(t, h, "t-a", "u-1", "tools/list", nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("solo \"true\" habilita, obtuve %d", rec.Code)
	}
}

func TestHandlerSinTokenDevuelve401(t *testing.T) {
	t.Setenv("MCP_HABILITADO", "true")
	svc := oauth.NuevoService(nil, nil, nil, nil, oauth.Config{Issuer: "https://mcp.test", Resource: "https://mcp.test/mcp"})
	h := NuevoHandler(svc, Dependencias{Store: nuevoStoreFalso()})
	r := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader("{}"))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Header().Get("WWW-Authenticate"), "resource_metadata") {
		t.Errorf("esperaba 401 con desafío, obtuve %d %q", rec.Code, rec.Header().Get("WWW-Authenticate"))
	}
}

func TestHandlerAceptaHostPublicoDetrasDeProxyLoopback(t *testing.T) {
	h := nuevoHandlerDePrueba(t, nuevoStoreFalso())
	cuerpo := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`
	r := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(cuerpo))
	r.Host = "apiqa.fudd.com.ar"
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json, text/event-stream")
	r.Header.Set("X-Tenant", "t-a")
	r.Header.Set("X-Usuario", "u-1")
	addr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8080}
	r = r.WithContext(context.WithValue(r.Context(), http.LocalAddrContextKey, addr))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	if rec.Code != http.StatusOK {
		t.Errorf("esperaba 200 con Host público sobre loopback, obtuve %d: %s", rec.Code, rec.Body)
	}
}

func TestInitializeYListadoDeTools(t *testing.T) {
	h := nuevoHandlerDePrueba(t, nuevoStoreFalso())

	rec, resp := rpc(t, h, "t-a", "u-1", "initialize", map[string]any{
		"protocolVersion": "2025-06-18", "capabilities": map[string]any{},
		"clientInfo": map[string]any{"name": "test", "version": "1"},
	})
	if rec.Code != http.StatusOK || resp.Result.ServerInfo.Name != "fudd" || !strings.Contains(resp.Result.Instructions, "listar_sucursales") {
		t.Fatalf("initialize: %d %s", rec.Code, rec.Body)
	}

	rec, resp = rpc(t, h, "t-a", "u-1", "tools/list", map[string]any{})
	if rec.Code != http.StatusOK {
		t.Fatalf("tools/list: %d %s", rec.Code, rec.Body)
	}
	esperadas := map[string]bool{"listar_sucursales": false, "estado_salon": false, "estado_cocina": false, "resumen_turno": false}
	for _, tool := range resp.Result.Tools {
		if _, ok := esperadas[tool.Name]; !ok {
			t.Errorf("tool inesperada %q", tool.Name)
		}
		esperadas[tool.Name] = true
		if tool.Annotations.ReadOnlyHint == nil || !*tool.Annotations.ReadOnlyHint {
			t.Errorf("%s sin readOnlyHint", tool.Name)
		}
		if tool.Annotations.OpenWorldHint == nil || *tool.Annotations.OpenWorldHint {
			t.Errorf("%s debería tener openWorldHint=false", tool.Name)
		}
	}
	for nombre, vista := range esperadas {
		if !vista {
			t.Errorf("falta la tool %s", nombre)
		}
	}
}

func TestRateLimitPorUsuario(t *testing.T) {
	h := nuevoHandlerDePrueba(t, nuevoStoreFalso())
	for i := 1; i <= 300; i++ {
		if rec, _ := rpc(t, h, "t-a", "u-1", "tools/list", map[string]any{}); rec.Code != http.StatusOK {
			t.Fatalf("llamada %d: %d", i, rec.Code)
		}
	}
	rec, _ := rpc(t, h, "t-a", "u-1", "tools/list", map[string]any{})
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") != "60" ||
		!strings.Contains(rec.Body.String(), "rate_limited") {
		t.Errorf("llamada 301: %d %q %s", rec.Code, rec.Header().Get("Retry-After"), rec.Body)
	}
	if rec, _ := rpc(t, h, "t-a", "u-2", "tools/list", map[string]any{}); rec.Code != http.StatusOK {
		t.Errorf("otro usuario no debería estar limitado: %d", rec.Code)
	}
}

func TestRateLimitDeToolsEsLegibleParaElModelo(t *testing.T) {
	h := nuevoHandlerDePrueba(t, nuevoStoreFalso())
	for i := 1; i <= 60; i++ {
		if resp := llamarTool(t, h, "t-a", "listar_sucursales", nil); resp.Result.IsError {
			t.Fatalf("tool %d: %+v", i, resp.Result)
		}
	}
	resp := llamarTool(t, h, "t-a", "listar_sucursales", nil)
	if !resp.Result.IsError || len(resp.Result.Content) == 0 {
		t.Fatalf("tool 61: %+v", resp.Result)
	}
	err := errorEstructurado(t, resp)
	if err.Error != CodigoRateLimited || err.Mensaje != mensajeRateLimit {
		t.Errorf("tool 61: %+v", err)
	}
	if resp := llamarTool(t, h, "t-b", "listar_sucursales", nil); resp.Result.IsError {
		t.Errorf("otro usuario no debería estar limitado: %+v", resp.Result)
	}
}

func TestToolsSinAccesoAMCP(t *testing.T) {
	t.Setenv("MCP_HABILITADO", "true")
	acceso := &accesoFalso{denegados: map[string]bool{"t-a": true}}
	h := nuevoHandler(authDePrueba, Dependencias{
		Store: nuevoStoreFalso(), Metricas: &metricasFalsas{}, Acceso: acceso, AppURL: "https://app.test",
	})

	esperado := "Tu plan actual no incluye la conexión con asistentes de IA (es exclusiva del Plan Pro). " +
		"Podés activarlo desde Configuración → Planes en https://app.test/dashboard."
	for _, tool := range []string{"listar_sucursales", "estado_salon", "estado_cocina", "resumen_turno"} {
		resp := llamarTool(t, h, "t-a", tool, nil)
		if !resp.Result.IsError || len(resp.Result.Content) == 0 {
			t.Errorf("%s sin acceso: %+v", tool, resp.Result)
			continue
		}
		err := errorEstructurado(t, resp)
		detalle, _ := err.Detalle.(map[string]any)
		if err.Error != CodigoPlanLimitReached || err.Mensaje != esperado ||
			detalle["recurso"] != "mcp" || detalle["upgrade_url"] != "https://app.test/dashboard" {
			t.Errorf("%s sin acceso: %+v", tool, err)
		}
	}
	if resp := llamarTool(t, h, "t-b", "listar_sucursales", nil); resp.Result.IsError {
		t.Errorf("otro tenant con acceso: %+v", resp.Result)
	}

	acceso.err = errors.New("db caída")
	resp := llamarTool(t, h, "t-b", "listar_sucursales", nil)
	if !resp.Result.IsError || len(resp.Result.Content) == 0 || errorEstructurado(t, resp).Error != CodigoErrorInterno {
		t.Errorf("error de DB debe ser interno, no sin acceso: %+v", resp.Result)
	}
}

func TestResolverSucursal(t *testing.T) {
	h := &herramientas{deps: Dependencias{Store: nuevoStoreFalso()}}
	ctx := context.Background()

	if s, err := h.resolverSucursal(ctx, "t-a", ""); err != nil || s.ID != "s-a" {
		t.Errorf("una sucursal: %v %v", s, err)
	}
	if s, err := h.resolverSucursal(ctx, "t-b", "s-b2"); err != nil || s.Nombre != "Sur" {
		t.Errorf("id propio: %v %v", s, err)
	}
	_, err := h.resolverSucursal(ctx, "t-b", "")
	var et *errTool
	if !errors.As(err, &et) || et.Codigo != CodigoSucursalRequerida || !strings.Contains(err.Error(), `"id":"s-b1","nombre":"Norte"`) ||
		!strings.Contains(err.Error(), `"id":"s-b2","nombre":"Sur"`) || !strings.Contains(err.Error(), "sucursal_id") {
		t.Errorf("varias sin id: %v", err)
	}
	_, ajena := h.resolverSucursal(ctx, "t-a", "s-b1")
	_, inexistente := h.resolverSucursal(ctx, "t-a", "no-existe")
	if !errors.As(ajena, &et) || et.Codigo != CodigoSucursalNoEncontrada || inexistente == nil || inexistente.Error() != ajena.Error() {
		t.Errorf("ajena=%v inexistente=%v", ajena, inexistente)
	}
	if strings.Contains(ajena.Error(), "s-b1") || !strings.Contains(ajena.Error(), `"id":"s-a"`) {
		t.Errorf("el error debe listar solo las sucursales propias: %v", ajena)
	}
	if _, err := h.resolverSucursal(ctx, "t-c", ""); err == nil || !strings.Contains(err.Error(), "todavía no tiene sucursales") {
		t.Errorf("sin sucursales: %v", err)
	}
}

// errorEstructurado exige que el error viaje como JSON en el texto y sin structuredContent.
func errorEstructurado(t *testing.T, resp respuestaRPC) errorTool {
	t.Helper()
	if !resp.Result.IsError || len(resp.Result.Content) == 0 {
		t.Fatalf("esperaba un error de tool: %+v", resp.Result)
	}
	var deTexto errorTool
	if err := json.Unmarshal([]byte(resp.Result.Content[0].Text), &deTexto); err != nil || deTexto.Error == "" {
		t.Fatalf("el texto del error no es JSON estructurado: %q", resp.Result.Content[0].Text)
	}
	if len(resp.Result.StructuredContent) != 0 {
		t.Fatalf("un error no debe traer structuredContent: %s", resp.Result.StructuredContent)
	}
	return deTexto
}

type errorTool struct {
	Error   string `json:"error"`
	Mensaje string `json:"mensaje"`
	Detalle any    `json:"detalle"`
}

// salidaTool exige que el TextContent sea el mismo JSON que structuredContent y lo decodifica en out.
func salidaTool(t *testing.T, resp respuestaRPC, out any) {
	t.Helper()
	if resp.Result.IsError || len(resp.Result.Content) == 0 {
		t.Fatalf("la tool falló: %+v", resp.Result)
	}
	texto := resp.Result.Content[0].Text
	if texto != string(resp.Result.StructuredContent) {
		t.Fatalf("TextContent distinto de structuredContent:\n%s\n%s", texto, resp.Result.StructuredContent)
	}
	if strings.Contains(texto, "\n") {
		t.Errorf("el JSON no debe estar indentado: %s", texto)
	}
	if err := json.Unmarshal([]byte(texto), out); err != nil {
		t.Fatalf("el texto no es JSON: %v", err)
	}
}

func TestToolsCallConDatos(t *testing.T) {
	st := nuevoStoreFalso()
	metricas := &metricasFalsas{}
	t.Setenv("MCP_HABILITADO", "true")
	h := nuevoHandler(authDePrueba, Dependencias{
		Store: st, Metricas: metricas, Acceso: permiteTodo(),
		Ahora: func() time.Time { return time.Date(2026, 10, 22, 15, 30, 0, 0, time.UTC) },
	})

	t.Run("listar_sucursales", func(t *testing.T) {
		var out SalidaSucursales
		salidaTool(t, llamarTool(t, h, "t-b", "listar_sucursales", nil), &out)
		if len(out.Sucursales) != 2 || out.Sucursales[1].MesasActivas != 3 || out.Sucursales[1].MesasTotales != 3 ||
			out.Sucursales[0].AbiertaAhora != nil {
			t.Errorf("%+v", out)
		}
		if out.Resumen != "2 sucursales en el negocio." || out.TS != "2026-10-22T12:30:00-03:00" {
			t.Errorf("resumen/ts: %q %q", out.Resumen, out.TS)
		}
		var propia SalidaSucursales
		salidaTool(t, llamarTool(t, h, "t-a", "listar_sucursales", nil), &propia)
		if propia.Resumen != "1 sucursal en el negocio." || propia.Sucursales[0].AbiertaAhora == nil || !*propia.Sucursales[0].AbiertaAhora {
			t.Errorf("12:30 con horario 08-00 debería estar abierta: %+v", propia)
		}
	})

	t.Run("estado_salon", func(t *testing.T) {
		resp := llamarTool(t, h, "t-a", "estado_salon", nil)
		var out SalidaSalon
		salidaTool(t, resp, &out)
		if out.SucursalID != "s-a" || out.SucursalNombre != "Centro" || out.Moneda != "ARS" || out.TS != "2026-10-22T12:30:00-03:00" {
			t.Errorf("cabecera: %+v", out)
		}
		esperado := TotalesSalon{MesasOcupadas: 2, MesasTotales: 3, Comensales: 5, MesasEsperando: Esperando{Comida: 1, Cobro: 1}}
		if out.Totales != esperado {
			t.Errorf("totales: %+v", out.Totales)
		}
		if len(out.Mesas) != 2 || out.Mesas[0].Numero != 1 || out.Mesas[1].Numero != 4 {
			t.Errorf("sin incluir_libres no debe listar la mesa libre: %+v", out.Mesas)
		}
		if len(out.PorTomador) != 2 || out.PorTomador[0].Tomador != TomadorQR || out.PorTomador[0].Mesas != 2 ||
			out.PorTomador[0].Comensales != 5 || out.PorTomador[0].MesasEsperando != (Esperando{Comida: 1, Cobro: 1}) ||
			out.PorTomador[1].Tomador != "Ana" || out.PorTomador[1].Mesas != 1 {
			t.Errorf("por_tomador: %+v", out.PorTomador)
		}
		if out.Resumen != "Centro: 2 de 3 mesas ocupadas, 5 comensales. Mesas esperando: comida 1, entrega 0, cobro 1." {
			t.Errorf("resumen: %q", out.Resumen)
		}
		// Valor no calculable: null, no 0.
		if !strings.Contains(string(resp.Result.StructuredContent), `"minutos_desde_primer_pedido":null`) ||
			!strings.Contains(string(resp.Result.StructuredContent), `"minutos_en_espera":null`) {
			t.Errorf("faltan nulls: %s", resp.Result.StructuredContent)
		}

		var conLibres SalidaSalon
		salidaTool(t, llamarTool(t, h, "t-a", "estado_salon", map[string]any{"incluir_libres": true}), &conLibres)
		if len(conLibres.Mesas) != 3 {
			t.Errorf("incluir_libres: %+v", conLibres.Mesas)
		}
	})

	t.Run("estado_cocina", func(t *testing.T) {
		var out SalidaCocina
		salidaTool(t, llamarTool(t, h, "t-a", "estado_cocina", nil), &out)
		if out.Cola != (ColaCocina{Recibido: 1, Preparando: 1, Listo: 1}) {
			t.Errorf("cola: %+v", out.Cola)
		}
		if len(out.Pedidos) != 3 || out.Pedidos[0].PedidoID != "p3" || out.Pedidos[2].PedidoID != "p1" {
			t.Errorf("orden por minutos desc: %+v", out.Pedidos)
		}
		u := out.UltimaHora
		if u.Ingresados60Min != 5 || u.Listos60Min != 3 || u.Balance60Min != 2 ||
			u.DespachoPromedioMin == nil || *u.DespachoPromedioMin != 6 || u.DespachoP90Min == nil || *u.DespachoP90Min != 11.5 {
			t.Errorf("ultima_hora: %+v", u)
		}
		if out.Resumen != "Centro: en cocina 1 pedido recibido, 1 preparando, 1 listo. Más demorado: mesa 2 (preparando) hace 12 minutos." {
			t.Errorf("resumen (el listo no cuenta como demorado): %q", out.Resumen)
		}
	})

	t.Run("estado_cocina sin despachos en la hora", func(t *testing.T) {
		original := st.hora
		st.hora = UltimaHora{Ingresados60Min: 1}
		defer func() { st.hora = original }()
		resp := llamarTool(t, h, "t-a", "estado_cocina", nil)
		var out SalidaCocina
		salidaTool(t, resp, &out)
		if out.UltimaHora.DespachoPromedioMin != nil || out.UltimaHora.DespachoP90Min != nil || out.UltimaHora.Balance60Min != 1 ||
			!strings.Contains(string(resp.Result.StructuredContent), `"despacho_p90_min":null`) {
			t.Errorf("sin datos debe ser null: %s", resp.Result.StructuredContent)
		}
	})

	t.Run("resumen_turno", func(t *testing.T) {
		resp := llamarTool(t, h, "t-a", "resumen_turno", nil)
		var out SalidaTurno
		salidaTool(t, resp, &out)
		if out.Pedidos != 4 || out.PedidosCerrados != 3 || out.Comensales != 11 || out.Facturacion != 1500 || out.Moneda != "ARS" ||
			out.TicketPromedio == nil || *out.TicketPromedio != 500 || out.DespachoPromedioMin == nil || *out.DespachoPromedioMin != 7.5 {
			t.Errorf("%+v", out)
		}
		if len(out.PorHora) != 2 || out.PorHora[0].Hora != "11:00" || out.PorHora[1].DespachoPromedioMin != nil {
			t.Errorf("por_hora: %+v", out.PorHora)
		}
		if out.Resumen != "Centro, hoy: 4 pedidos (3 cerrados), 11 comensales, facturación $1500.00 ARS, ticket promedio $500.00 ARS." {
			t.Errorf("resumen: %q", out.Resumen)
		}
		// 15:30 UTC = 12:30 en Buenos Aires: el día arranca 00:00 local.
		if !strings.HasPrefix(metricas.desde, "2026-10-22T00:00:00") || !strings.HasSuffix(metricas.desde, "-03:00") ||
			!strings.HasPrefix(metricas.hasta, "2026-10-22T12:30:00") || out.ZonaHoraria != zonaHorariaNegocio ||
			out.Desde != "2026-10-22T00:00:00-03:00" {
			t.Errorf("período: %s -> %s (%s)", metricas.desde, metricas.hasta, out.Desde)
		}
	})

	t.Run("errores de sucursal son estructurados", func(t *testing.T) {
		err := errorEstructurado(t, llamarTool(t, h, "t-b", "estado_salon", nil))
		detalle, _ := err.Detalle.(map[string]any)
		sucursales, _ := detalle["sucursales"].([]any)
		if err.Error != CodigoSucursalRequerida || len(sucursales) != 2 {
			t.Errorf("sucursal_requerida: %+v", err)
		}
		err = errorEstructurado(t, llamarTool(t, h, "t-a", "estado_cocina", map[string]any{"sucursal_id": "s-b1"}))
		if err.Error != CodigoSucursalNoEncontrada || err.Mensaje != "Sucursal no encontrada" {
			t.Errorf("sucursal_no_encontrada: %+v", err)
		}
	})

	t.Run("error interno no filtra detalles", func(t *testing.T) {
		st.falla = errors.New("pq: password authentication failed")
		defer func() { st.falla = nil }()
		resp := llamarTool(t, h, "t-a", "estado_salon", nil)
		err := errorEstructurado(t, resp)
		if err.Error != CodigoErrorInterno || err.Mensaje != mensajeInterno || strings.Contains(resp.Result.Content[0].Text, "password") {
			t.Errorf("%+v", err)
		}
	})
}

func TestServidorPorRequestAislaTenants(t *testing.T) {
	h := nuevoHandlerDePrueba(t, nuevoStoreFalso())
	for i := 0; i < 3; i++ {
		for _, caso := range []struct{ tenant, suc, mesa string }{{"t-a", "s-a", "1"}, {"t-b", "s-b1", "9"}} {
			args := map[string]any{}
			if caso.tenant == "t-b" {
				args["sucursal_id"] = caso.suc
			}
			var out SalidaSalon
			salidaTool(t, llamarTool(t, h, caso.tenant, "estado_salon", args), &out)
			if out.SucursalID != caso.suc || len(out.Mesas) != 1+boolInt(caso.tenant == "t-a") ||
				fmt.Sprint(out.Mesas[0].Numero) != caso.mesa {
				t.Fatalf("iteración %d tenant %s vio %+v", i, caso.tenant, out)
			}
		}
	}
	// Un tenant nunca ve la sucursal del otro, ni con el id correcto.
	err := errorEstructurado(t, llamarTool(t, h, "t-a", "estado_salon", map[string]any{"sucursal_id": "s-b1"}))
	if err.Error != CodigoSucursalNoEncontrada || strings.Contains(fmt.Sprint(err.Detalle), "s-b1") {
		t.Errorf("cruce de tenants: %+v", err)
	}
}

func TestDescripcionesDeLasTools(t *testing.T) {
	campos := map[string][]string{
		"listar_sucursales": {"abierta_ahora", "mesas_totales", "mesas_activas"},
		"estado_salon":      {"minutos_en_espera", "por_tomador", "mesas_esperando", "tomado_por", "incluir_libres", "no hay registro de personal", "no hay confirmación"},
		"estado_cocina":     {"balance_60min", "p90", "ultima_hora", "cola", "historial"},
		"resumen_turno":     {"por_hora", "ticket_promedio", "00:00", "zona horaria"},
	}
	descripciones := map[string]string{
		"listar_sucursales": descListarSucursales, "estado_salon": descEstadoSalon,
		"estado_cocina": descEstadoCocina, "resumen_turno": descResumenTurno,
	}
	for tool, esperados := range campos {
		for _, e := range esperados {
			if !strings.Contains(descripciones[tool], e) {
				t.Errorf("la descripción de %s no menciona %q", tool, e)
			}
		}
	}
}

func TestCantidad(t *testing.T) {
	casos := []struct {
		n        int
		esperado string
	}{{0, "0 mesas"}, {1, "1 mesa"}, {2, "2 mesas"}, {11, "11 mesas"}}
	for _, c := range casos {
		if got := cantidad(c.n, "mesa", "mesas"); got != c.esperado {
			t.Errorf("cantidad(%d) = %q, esperaba %q", c.n, got, c.esperado)
		}
	}
}

func TestDerivarEspera(t *testing.T) {
	casos := []struct {
		nombre          string
		estado          string
		enCocina, listo int
		minCocina, minL *int
		espera          string
		minutos         *int
	}{
		{"libre", MesaLibre, 0, 0, nil, nil, "", nil},
		{"comida", MesaOcupada, 2, 1, ptr(15), ptr(3), EsperaComida, ptr(15)},
		{"entrega", MesaOcupada, 0, 1, nil, ptr(3), EsperaEntrega, ptr(3)},
		{"entrega sin listo_at", MesaOcupada, 0, 1, nil, nil, EsperaEntrega, nil},
		{"cobro gana a comida", MesaCuentaSolicitada, 1, 0, ptr(9), nil, EsperaCobro, nil},
		{"todo entregado no espera", MesaOcupada, 0, 0, nil, nil, "", nil},
		{"entregado con cobro", MesaCuentaSolicitada, 0, 0, nil, nil, EsperaCobro, nil},
		{"pago habilitado", MesaPagoHabilitado, 0, 0, nil, nil, EsperaCobro, nil},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			espera, minutos := derivarEspera(c.estado, c.enCocina, c.listo, c.minCocina, c.minL)
			if (espera == nil) != (c.espera == "") || (espera != nil && *espera != c.espera) || !reflect.DeepEqual(minutos, c.minutos) {
				t.Errorf("obtuve %v/%v, esperaba %q/%v", espera, minutos, c.espera, c.minutos)
			}
		})
	}
}

func TestMinutosDesde(t *testing.T) {
	ahora := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
	hace := ahora.Add(-90*time.Minute - 30*time.Second)
	futuro := ahora.Add(time.Minute)
	if m := minutosDesde(ahora, &hace); m == nil || *m != 90 {
		t.Errorf("hace 90m30s: %v", m)
	}
	if m := minutosDesde(ahora, &futuro); m == nil || *m != 0 {
		t.Errorf("futuro no debe ser negativo: %v", m)
	}
	if minutosDesde(ahora, nil) != nil {
		t.Error("sin timestamp debe ser nil")
	}
}

func TestAbiertaAhora(t *testing.T) {
	zona := ubicacionNegocio()
	// 2026-10-23 es viernes; 2026-10-24 sábado.
	en := func(dia, hora, min int) time.Time { return time.Date(2026, 10, dia, hora, min, 0, 0, zona) }
	simple := horarioTodosLosDias(`[{"apertura":"08:00","cierre":"23:00"}]`)
	noche := horarioTodosLosDias(`[{"apertura":"20:00","cierre":"02:00"}]`)
	hastaMedianoche := horarioTodosLosDias(`[{"apertura":"08:00","cierre":"00:00"}]`)
	partido := horarioTodosLosDias(`[{"apertura":"12:00","cierre":"15:00"},{"apertura":"20:00","cierre":"00:00"}]`)
	soloViernes := []byte(`{"viernes":{"abierto":true,"tramos":[{"apertura":"20:00","cierre":"02:00"}]},"sabado":{"abierto":false,"tramos":[{"apertura":"20:00","cierre":"02:00"}]}}`)

	casos := []struct {
		nombre    string
		horarios  []byte
		ahora     time.Time
		esperaNil bool
		abierta   bool
	}{
		{"sin horarios", nil, en(23, 12, 0), true, false},
		{"objeto vacío (default de la base)", []byte(`{}`), en(23, 12, 0), true, false},
		{"JSON inválido", []byte(`no`), en(23, 12, 0), true, false},
		{"dentro del tramo", simple, en(23, 12, 0), false, true},
		{"antes de abrir", simple, en(23, 7, 59), false, false},
		{"en la apertura", simple, en(23, 8, 0), false, true},
		{"en el cierre exacto", simple, en(23, 23, 0), false, false},
		{"cierre 00:00 a las 23:59", hastaMedianoche, en(23, 23, 59), false, true},
		{"cierre 00:00 pasada la medianoche", hastaMedianoche, en(24, 0, 1), false, false},
		{"cruza medianoche: antes de medianoche", noche, en(23, 23, 0), false, true},
		{"cruza medianoche: después de medianoche usa el día anterior", noche, en(24, 1, 30), false, true},
		{"cruza medianoche: después del cierre", noche, en(24, 2, 0), false, false},
		{"cruza medianoche: de tarde, aún cerrado", noche, en(24, 18, 0), false, false},
		{"dos tramos: entre ambos", partido, en(23, 17, 0), false, false},
		{"dos tramos: segundo", partido, en(23, 21, 0), false, true},
		{"día anterior abierto cruza a hoy cerrado", soloViernes, en(24, 1, 0), false, true},
		{"hoy sábado cerrado no abre", soloViernes, en(24, 21, 0), false, false},
		{"domingo: el sábado cerrado no extiende", soloViernes, en(25, 1, 0), false, false},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			got := abiertaAhora(c.horarios, c.ahora)
			if c.esperaNil {
				if got != nil {
					t.Errorf("esperaba nil, obtuve %v", *got)
				}
				return
			}
			if got == nil || *got != c.abierta {
				t.Errorf("esperaba %v, obtuve %v", c.abierta, got)
			}
		})
	}
}

func TestResumirSalonPorTomador(t *testing.T) {
	mesas := []MesaEstado{
		{Numero: 1, Estado: MesaOcupada, Comensales: 2, Espera: ptr(EsperaComida), TomadoPor: []string{"Ana"}},
		{Numero: 2, Estado: MesaOcupada, Comensales: 4, Espera: ptr(EsperaEntrega), TomadoPor: []string{"Ana", TomadorQR}},
		{Numero: 3, Estado: MesaPagoHabilitado, Comensales: 1, Espera: ptr(EsperaCobro), TomadoPor: []string{TomadorQR}},
		{Numero: 4, Estado: MesaLibre, TomadoPor: []string{}},
	}
	totales, porTomador, visibles := resumirSalon(mesas, false)
	if totales.MesasOcupadas != 3 || totales.MesasTotales != 4 || totales.Comensales != 7 ||
		totales.MesasEsperando != (Esperando{Comida: 1, Entrega: 1, Cobro: 1}) || len(visibles) != 3 {
		t.Errorf("totales: %+v visibles=%d", totales, len(visibles))
	}
	esperado := []TomadorSalon{
		{Tomador: "Ana", Mesas: 2, Comensales: 6, MesasEsperando: Esperando{Comida: 1, Entrega: 1}},
		{Tomador: TomadorQR, Mesas: 2, Comensales: 5, MesasEsperando: Esperando{Entrega: 1, Cobro: 1}},
	}
	if !reflect.DeepEqual(porTomador, esperado) {
		t.Errorf("por_tomador: %+v", porTomador)
	}
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func TestDerivarMesa(t *testing.T) {
	casos := []struct {
		nombre         string
		d              datosMesa
		estado         string
		comensales     int
		sinIdentificar bool
	}{
		{"sin pedidos", datosMesa{}, MesaLibre, 0, false},
		{"sin pedidos ignora flags", datosMesa{CuentaSolicitada: true, PagoHabilitado: true}, MesaLibre, 0, false},
		{"ocupada", datosMesa{PedidosTotales: 2, PedidosActivos: 1, Comensales: 2}, MesaOcupada, 2, false},
		{"todos cerrados", datosMesa{PedidosTotales: 2, Comensales: 2}, MesaLibre, 2, false},
		{"cuenta solicitada", datosMesa{PedidosTotales: 1, PedidosActivos: 1, CuentaSolicitada: true}, MesaCuentaSolicitada, 0, false},
		{"pago gana a cuenta", datosMesa{PedidosTotales: 1, PedidosActivos: 1, CuentaSolicitada: true, PagoHabilitado: true}, MesaPagoHabilitado, 0, false},
		{"cuenta solicitada sin activos", datosMesa{PedidosTotales: 1, CuentaSolicitada: true}, MesaCuentaSolicitada, 0, false},
		{"solo sin identificar", datosMesa{PedidosTotales: 1, PedidosActivos: 1, SinIdentificar: true}, MesaOcupada, 1, true},
		{"mezcla identificados", datosMesa{PedidosTotales: 1, PedidosActivos: 1, Comensales: 3, SinIdentificar: true}, MesaOcupada, 3, true},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			estado, comensales, sin := derivarMesa(c.d)
			if estado != c.estado || comensales != c.comensales || sin != c.sinIdentificar {
				t.Errorf("obtuve %s/%d/%v, esperaba %s/%d/%v", estado, comensales, sin, c.estado, c.comensales, c.sinIdentificar)
			}
		})
	}
}

func capturarLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	original := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(original) })
	return &buf
}

func TestLogDeRequestsMCP(t *testing.T) {
	logs := capturarLogs(t)
	h := nuevoHandlerDePrueba(t, nuevoStoreFalso())

	rec, resp := rpc(t, h, "t-a", "u-1", "tools/list", nil)
	if rec.Code != http.StatusOK || len(resp.Result.Tools) == 0 {
		t.Fatalf("tools/list falló: %d %s", rec.Code, rec.Body)
	}
	salida := logs.String()
	for _, esperado := range []string{"rpc_method=tools/list", "status=200", "http_method=POST", "usuario_id=u-1", "tiene_authorization=false"} {
		if !strings.Contains(salida, esperado) {
			t.Errorf("el log no contiene %q: %s", esperado, salida)
		}
	}
}

func TestLogSinTokenNoFiltraSecretos(t *testing.T) {
	logs := capturarLogs(t)
	t.Setenv("MCP_HABILITADO", "true")
	svc := oauth.NuevoService(nil, nil, nil, nil, oauth.Config{Issuer: "https://mcp.test", Resource: "https://mcp.test/mcp"})
	h := NuevoHandler(svc, Dependencias{Store: nuevoStoreFalso()})
	const cuerpo = `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{"secreto":"PARAMS-SECRETOS"}}`

	sinToken := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(cuerpo))
	sinToken.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, sinToken)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("esperaba 401, obtuve %d", rec.Code)
	}
	salida := logs.String()
	if !strings.Contains(salida, "status=401") || !strings.Contains(salida, "tiene_authorization=false") {
		t.Errorf("log sin token incorrecto: %s", salida)
	}

	logs.Reset()
	conToken := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(cuerpo))
	conToken.Header.Set("Content-Type", "application/json")
	conToken.Header.Set("Authorization", "Bearer TOKEN-SECRETO")
	h.ServeHTTP(httptest.NewRecorder(), conToken)
	salida = logs.String()
	if !strings.Contains(salida, "tiene_authorization=true") {
		t.Errorf("esperaba tiene_authorization=true: %s", salida)
	}
	if strings.Contains(salida, "TOKEN-SECRETO") || strings.Contains(salida, "PARAMS-SECRETOS") {
		t.Errorf("el log filtra token o params: %s", salida)
	}
}

func TestLogNoConsumeElBodyDelSDK(t *testing.T) {
	logs := capturarLogs(t)
	h := nuevoHandlerDePrueba(t, nuevoStoreFalso())
	resp := llamarTool(t, h, "t-a", "listar_sucursales", nil)
	if resp.Result.IsError || len(resp.Result.Content) == 0 {
		t.Fatalf("tools/call no llegó completo al SDK: %+v", resp.Result)
	}
	if !strings.Contains(logs.String(), "rpc_method=tools/call") {
		t.Errorf("el log no contiene rpc_method=tools/call: %s", logs)
	}
}

func postLogueado(h http.Handler, cuerpo string, cabeceras map[string]string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(cuerpo))
	r.Header.Set("Content-Type", "application/json")
	for k, v := range cabeceras {
		r.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

func TestLogMetodoDesconocidoOLargoSeRegistraComoOtro(t *testing.T) {
	logs := capturarLogs(t)
	h := nuevoHandlerDePrueba(t, nuevoStoreFalso())
	postLogueado(h, `{"jsonrpc":"2.0","id":1,"method":"`+strings.Repeat("x", 5000)+`"}`, nil)
	salida := logs.String()
	if !strings.Contains(salida, "rpc_method=otro") || strings.Contains(salida, "xxxxx") {
		t.Errorf("esperaba rpc_method=otro: %s", salida)
	}
}

func TestLogBatchAcotaEntradas(t *testing.T) {
	logs := capturarLogs(t)
	h := nuevoHandlerDePrueba(t, nuevoStoreFalso())
	entradas := make([]string, 15)
	for i := range entradas {
		entradas[i] = `{"jsonrpc":"2.0","id":1,"method":"ping"}`
	}
	postLogueado(h, "["+strings.Join(entradas, ",")+"]", nil)
	esperado := "rpc_method=" + strings.TrimSuffix(strings.Repeat("ping,", 10), ",") + ",+5"
	if !strings.Contains(logs.String(), esperado) {
		t.Errorf("esperaba %q: %s", esperado, logs)
	}
}

func TestLogTruncaCabecerasDelCliente(t *testing.T) {
	logs := capturarLogs(t)
	h := nuevoHandlerDePrueba(t, nuevoStoreFalso())
	postLogueado(h, `{"jsonrpc":"2.0","id":1,"method":"ping"}`, map[string]string{"User-Agent": strings.Repeat("a", 1000)})
	salida := logs.String()
	if !strings.Contains(salida, "user_agent="+strings.Repeat("a", 120)+" ") {
		t.Errorf("user_agent no quedó en 120: %s", salida)
	}
}

type bodyEspia struct {
	leido bool
}

func (b *bodyEspia) Read([]byte) (int, error) { b.leido = true; return 0, io.EOF }
func (b *bodyEspia) Close() error             { return nil }

func TestLogDeshabilitadoNoLeeElBody(t *testing.T) {
	logs := capturarLogs(t)
	t.Setenv("MCP_HABILITADO", "")
	h := nuevoHandler(authDePrueba, Dependencias{Store: nuevoStoreFalso()})
	espia := &bodyEspia{}
	r := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	r.Body = espia
	r.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	if rec.Code != http.StatusNotFound || espia.leido {
		t.Errorf("esperaba 404 sin leer el body: code=%d leido=%v", rec.Code, espia.leido)
	}
	if salida := logs.String(); !strings.Contains(salida, "status=404") || strings.Contains(salida, "rpc_method") {
		t.Errorf("log incorrecto: %s", salida)
	}
}

// Los clientes MCP validan structuredContent contra el outputSchema de cada tool.
func TestStructuredContentCumpleOutputSchema(t *testing.T) {
	h := nuevoHandlerDePrueba(t, nuevoStoreFalso())
	_, lista := rpc(t, h, "t-a", "u-1", "tools/list", map[string]any{})
	if len(lista.Result.Tools) != 4 {
		t.Fatalf("esperaba 4 tools: %d", len(lista.Result.Tools))
	}
	for _, tool := range lista.Result.Tools {
		t.Run(tool.Name, func(t *testing.T) {
			var esquema jsonschema.Schema
			if err := json.Unmarshal(tool.OutputSchema, &esquema); err != nil {
				t.Fatalf("outputSchema inválido: %v %s", err, tool.OutputSchema)
			}
			resuelto, err := esquema.Resolve(nil)
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			resp := llamarTool(t, h, "t-a", tool.Name, nil)
			if resp.Result.IsError {
				t.Fatalf("la tool falló: %+v", resp.Result)
			}
			var instancia any
			if err := json.Unmarshal(resp.Result.StructuredContent, &instancia); err != nil {
				t.Fatalf("structuredContent no es JSON: %v", err)
			}
			if err := resuelto.Validate(instancia); err != nil {
				t.Errorf("structuredContent no cumple el outputSchema: %v\n%s", err, resp.Result.StructuredContent)
			}
		})
	}
}
