package tenant

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
)

type ErrorCuotaEnvelope struct {
	Error   string            `json:"error"`
	Codigo  string            `json:"codigo"`
	Detalle DetalleErrorCuota `json:"detalle"`
}

type DetalleErrorCuota struct {
	Recurso string `json:"recurso"`
	Nombre  string `json:"nombre"`
	Usado   int    `json:"usado"`
	Limite  int    `json:"limite"`
	Plan    string `json:"plan"`
	Accion  string `json:"accion"`
}

func RequerirCuota(store Store, rec Recurso) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := auth.ClaimsFromContext(r.Context())
			if claims == nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "no autenticado"})
				return
			}

			cuotas, err := store.ObtenerEstadoCuotas(r.Context(), claims.TenantID)
			if err != nil {
				slog.ErrorContext(r.Context(), "error verificando cuotas de plan", "tenant_id", claims.TenantID, "error", err)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "error interno verificando límites del plan"})
				return
			}

			now := time.Now()
			planEfectivo := cuotas.PlanEfectivo(now)

			// Si el recurso es Pro-only (carga_masiva) y el plan es Free -> 403
			if recursosPro[rec] {
				if planEfectivo != PlanPro {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusForbidden)
					_ = json.NewEncoder(w).Encode(ErrorCuotaEnvelope{
						Error:  fmt.Sprintf("La %s es una funcionalidad exclusiva del plan Pro.", NombreRecurso(rec)),
						Codigo: "PLAN_LIMIT_REACHED",
						Detalle: DetalleErrorCuota{
							Recurso: string(rec),
							Nombre:  NombreRecurso(rec),
							Usado:   0,
							Limite:  0,
							Plan:    planEfectivo,
							Accion:  "upgrade",
						},
					})
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			// Si es Pro y no es Pro-only
			if planEfectivo == PlanPro {
				next.ServeHTTP(w, r)
				return
			}

			// Plan Free para recursos estándar
			limite := limitesFree[rec]
			usado := 0
			if cuotas.Uso != nil {
				usado = cuotas.Uso[rec]
			}
			if usado >= limite {
				nombre := NombreRecurso(rec)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(ErrorCuotaEnvelope{
					Error:  fmt.Sprintf("Alcanzaste el límite de %d %s de tu plan Free.", limite, nombre),
					Codigo: "PLAN_LIMIT_REACHED",
					Detalle: DetalleErrorCuota{
						Recurso: string(rec),
						Nombre:  nombre,
						Usado:   usado,
						Limite:  limite,
						Plan:    planEfectivo,
						Accion:  "upgrade",
					},
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
