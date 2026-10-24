package pedido

import "testing"

func TestEstadoTrasItems_NoRetrocede(t *testing.T) {
	tests := []struct {
		nombre    string
		actual    string
		total     int
		listos    int
		iniciados int
		esperado  string
	}{
		{nombre: "listo por el mozo con ítems pendientes", actual: "listo", total: 2, esperado: "listo"},
		{nombre: "deshacer listo -> preparando", actual: "listo", total: 2, listos: 1, iniciados: 2, esperado: "listo"},
		{nombre: "deshacer preparando -> pendiente", actual: "preparando", total: 2, esperado: "preparando"},
		{nombre: "confirmado avanza a preparando", actual: "confirmado", total: 2, iniciados: 1, esperado: "preparando"},
		{nombre: "preparando avanza a listo", actual: "preparando", total: 2, listos: 2, iniciados: 2, esperado: "listo"},
		{nombre: "confirmado sin ítems iniciados", actual: "confirmado", total: 2, esperado: "confirmado"},
	}

	for _, tt := range tests {
		t.Run(tt.nombre, func(t *testing.T) {
			if obtenido := estadoTrasItems(tt.actual, tt.total, tt.listos, tt.iniciados); obtenido != tt.esperado {
				t.Fatalf("estado incorrecto: got %q, want %q", obtenido, tt.esperado)
			}
		})
	}
}

func TestResolverEstadoPedido(t *testing.T) {
	tests := []struct {
		nombre    string
		total     int
		listos    int
		iniciados int
		esperado  string
	}{
		{nombre: "todos pendientes", total: 3, esperado: "confirmado"},
		{nombre: "uno preparando", total: 3, iniciados: 1, esperado: "preparando"},
		{nombre: "uno listo y faltan items", total: 3, listos: 1, iniciados: 1, esperado: "preparando"},
		{nombre: "todos listos", total: 3, listos: 3, iniciados: 3, esperado: "listo"},
	}

	for _, tt := range tests {
		t.Run(tt.nombre, func(t *testing.T) {
			if obtenido := resolverEstadoPedido(tt.total, tt.listos, tt.iniciados); obtenido != tt.esperado {
				t.Fatalf("estado incorrecto: got %q, want %q", obtenido, tt.esperado)
			}
		})
	}
}
