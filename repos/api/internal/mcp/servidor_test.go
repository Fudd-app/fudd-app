package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/metrica"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/oauth"
)

type storeFalso struct {
	sucursales map[string][]SucursalResumen // por tenant
	mesas      map[string][]MesaEstado      // por sucursal
	pedidos    map[string][]PedidoCocina
	hora       *float64
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

func (s *storeFalso) DespachoPromedio(context.Context, string, string, time.Time) (*float64, error) {
	return s.hora, nil
}

func (s *storeFalso) ComensalesAtendidos(context.Context, string, string, time.Time, time.Time) (int, error) {
	return s.comensales, nil
}

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

func nuevoStoreFalso() *storeFalso {
	return &storeFalso{
		sucursales: map[string][]SucursalResumen{
			"t-a": {{ID: "s-a", Nombre: "Centro", MesasActivas: 2}},
			"t-b": {{ID: "s-b1", Nombre: "Norte", MesasActivas: 1}, {ID: "s-b2", Nombre: "Sur", MesasActivas: 3}},
			"t-c": {},
		},
		mesas: map[string][]MesaEstado{
			"s-a": {
				{Numero: 1, Estado: MesaOcupada, Comensales: 3, PedidosActivos: 1, TotalParcial: 100},
				{Numero: 2, Estado: MesaLibre},
			},
			"s-b1": {{Numero: 9, Estado: MesaCuentaSolicitada, Comensales: 2}},
		},
		pedidos: map[string][]PedidoCocina{
			"s-a": {
				{Mesa: 1, Estado: "recibido", Minutos: 4, ItemsPendientes: 2},
				{Mesa: 2, Estado: "preparando", Minutos: 12, ItemsPreparando: 1, ItemsListos: 1},
				{Mesa: 3, Estado: "listo", Minutos: 30, ItemsListos: 2},
			},
		},
		hora: ptr(6.0), comensales: 11,
	}
}

func nuevoHandlerDePrueba(t *testing.T, st *storeFalso) http.Handler {
	t.Helper()
	t.Setenv("MCP_HABILITADO", "true")
	return nuevoHandler(authDePrueba, Dependencias{
		Store:    st,
		Metricas: &metricasFalsas{},
		Ahora:    func() time.Time { return time.Date(2026, 10, 22, 15, 30, 0, 0, time.UTC) },
	})
}

type respuestaRPC struct {
	Result struct {
		Tools []struct {
			Name        string `json:"name"`
			Annotations struct {
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
	svc := oauth.NuevoService(nil, nil, nil, oauth.Config{Issuer: "https://mcp.test", Resource: "https://mcp.test/mcp"})
	h := NuevoHandler(svc, Dependencias{Store: nuevoStoreFalso()})
	r := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader("{}"))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Header().Get("WWW-Authenticate"), "resource_metadata") {
		t.Errorf("esperaba 401 con desafío, obtuve %d %q", rec.Code, rec.Header().Get("WWW-Authenticate"))
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
	if !resp.Result.IsError || len(resp.Result.Content) == 0 ||
		resp.Result.Content[0].Text != "Demasiadas consultas en el último minuto. Esperá unos segundos y volvé a intentar." {
		t.Errorf("tool 61: %+v", resp.Result)
	}
	if resp := llamarTool(t, h, "t-b", "listar_sucursales", nil); resp.Result.IsError {
		t.Errorf("otro usuario no debería estar limitado: %+v", resp.Result)
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
	if err == nil || !strings.Contains(err.Error(), "Norte (s-b1), Sur (s-b2)") || !strings.Contains(err.Error(), "sucursal_id") {
		t.Errorf("varias sin id: %v", err)
	}
	_, ajena := h.resolverSucursal(ctx, "t-a", "s-b1")
	_, inexistente := h.resolverSucursal(ctx, "t-a", "no-existe")
	if ajena == nil || ajena.Error() != "Sucursal no encontrada" || inexistente == nil || inexistente.Error() != ajena.Error() {
		t.Errorf("ajena=%v inexistente=%v", ajena, inexistente)
	}
	if _, err := h.resolverSucursal(ctx, "t-c", ""); err == nil || !strings.Contains(err.Error(), "todavía no tiene sucursales") {
		t.Errorf("sin sucursales: %v", err)
	}
}

func TestToolsCallConDatos(t *testing.T) {
	st := nuevoStoreFalso()
	metricas := &metricasFalsas{}
	t.Setenv("MCP_HABILITADO", "true")
	h := nuevoHandler(authDePrueba, Dependencias{
		Store: st, Metricas: metricas,
		Ahora: func() time.Time { return time.Date(2026, 10, 22, 15, 30, 0, 0, time.UTC) },
	})

	t.Run("listar_sucursales", func(t *testing.T) {
		resp := llamarTool(t, h, "t-b", "listar_sucursales", nil)
		var out SalidaSucursales
		_ = json.Unmarshal(resp.Result.StructuredContent, &out)
		if resp.Result.IsError || len(out.Sucursales) != 2 || out.Sucursales[1].MesasActivas != 3 {
			t.Errorf("%+v %s", out, resp.Result.StructuredContent)
		}
		if len(resp.Result.Content) == 0 || !strings.Contains(resp.Result.Content[0].Text, "2 sucursal") {
			t.Errorf("texto: %+v", resp.Result.Content)
		}
	})

	t.Run("estado_salon", func(t *testing.T) {
		resp := llamarTool(t, h, "t-a", "estado_salon", nil)
		var out SalidaSalon
		_ = json.Unmarshal(resp.Result.StructuredContent, &out)
		if resp.Result.IsError || out.Sucursal.ID != "s-a" || out.Totales != (TotalesSalon{Mesas: 2, Ocupadas: 1, Comensales: 3}) {
			t.Errorf("%+v", out)
		}
		if got := resp.Result.Content[0].Text; !strings.Contains(got, "1 de 2 mesas ocupadas, 3 comensales") {
			t.Errorf("texto: %q", got)
		}
	})

	t.Run("estado_cocina", func(t *testing.T) {
		resp := llamarTool(t, h, "t-a", "estado_cocina", nil)
		var out SalidaCocina
		_ = json.Unmarshal(resp.Result.StructuredContent, &out)
		r := out.Resumen
		if r.Recibidos != 1 || r.Preparando != 1 || r.Listos != 1 {
			t.Errorf("conteos: %+v", r)
		}
		if r.PedidoMasDemorado == nil || r.PedidoMasDemorado.Mesa != 2 || r.PedidoMasDemorado.Minutos != 12 {
			t.Errorf("más demorado (los listos no cuentan): %+v", r.PedidoMasDemorado)
		}
		if r.DespachoPromedioUltimaHora == nil || *r.DespachoPromedioUltimaHora != 6.0 {
			t.Errorf("despacho última hora: %+v", r)
		}
		if strings.Contains(string(resp.Result.StructuredContent), "despacho_promedio_hoy_min") {
			t.Errorf("estado_cocina no debe exponer el despacho de hoy: %s", resp.Result.StructuredContent)
		}
		if got := resp.Result.Content[0].Text; !strings.Contains(got, "1 recibidos, 1 preparando, 1 listos") {
			t.Errorf("texto: %q", got)
		}
	})

	t.Run("resumen_turno", func(t *testing.T) {
		resp := llamarTool(t, h, "t-a", "resumen_turno", nil)
		var out SalidaTurno
		_ = json.Unmarshal(resp.Result.StructuredContent, &out)
		if out.Pedidos != 4 || out.PedidosCerrados != 3 || out.ComensalesAtendidos != 11 || out.Facturacion != 1500 ||
			out.TicketPromedio != 500 || out.DespachoPromedioMin == nil || *out.DespachoPromedioMin != 7.5 {
			t.Errorf("%+v", out)
		}
		// 15:30 UTC = 12:30 en Buenos Aires: el día arranca 00:00 local.
		if !strings.HasPrefix(metricas.desde, "2026-10-22T00:00:00") || !strings.HasSuffix(metricas.desde, "-03:00") ||
			!strings.HasPrefix(metricas.hasta, "2026-10-22T12:30:00") || out.ZonaHoraria != zonaHorariaNegocio {
			t.Errorf("período: %s -> %s", metricas.desde, metricas.hasta)
		}
	})

	t.Run("sin sucursales o sucursal ajena es error de tool", func(t *testing.T) {
		resp := llamarTool(t, h, "t-b", "estado_salon", nil)
		if !resp.Result.IsError || !strings.Contains(resp.Result.Content[0].Text, "varias sucursales") {
			t.Errorf("%+v", resp.Result)
		}
		resp = llamarTool(t, h, "t-a", "estado_cocina", map[string]any{"sucursal_id": "s-b1"})
		if !resp.Result.IsError || resp.Result.Content[0].Text != "Sucursal no encontrada" {
			t.Errorf("%+v", resp.Result)
		}
	})

	t.Run("error interno no filtra detalles", func(t *testing.T) {
		st.falla = errors.New("pq: password authentication failed")
		defer func() { st.falla = nil }()
		resp := llamarTool(t, h, "t-a", "estado_salon", nil)
		texto := resp.Result.Content[0].Text
		if !resp.Result.IsError || texto != mensajeInterno {
			t.Errorf("texto: %q", texto)
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
			resp := llamarTool(t, h, caso.tenant, "estado_salon", args)
			var out SalidaSalon
			_ = json.Unmarshal(resp.Result.StructuredContent, &out)
			if out.Sucursal.ID != caso.suc || len(out.Mesas) != 1+boolInt(caso.tenant == "t-a") ||
				fmt.Sprint(out.Mesas[0].Numero) != caso.mesa {
				t.Fatalf("iteración %d tenant %s vio %+v", i, caso.tenant, out)
			}
		}
	}
	// Un tenant nunca ve la sucursal del otro, ni con el id correcto.
	resp := llamarTool(t, h, "t-a", "estado_salon", map[string]any{"sucursal_id": "s-b1"})
	if !resp.Result.IsError || resp.Result.Content[0].Text != "Sucursal no encontrada" {
		t.Errorf("cruce de tenants: %+v", resp.Result)
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
