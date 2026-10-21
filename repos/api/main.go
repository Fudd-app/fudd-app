// cmd/server/main.go
//
// Punto de entrada del servidor fudd.
// Cuando corrés "go run ./cmd/server" esto es lo primero que ejecuta Go.
//
// Responsabilidades:
//   1. Cargar variables de entorno (.env)
//   2. Conectar a la base de datos
//   3. Ejecutar migraciones pendientes
//   4. Registrar las rutas de la API
//   5. Arrancar el servidor HTTP

package main

import (
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/joho/godotenv"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/db"
)

func main() {
	// --- 1. Cargar variables de entorno ---
	// godotenv lee el archivo .env y carga las variables.
	// En producción no habrá .env (las variables vienen del servidor),
	// por eso no fallamos si el archivo no existe.
	if err := godotenv.Load(); err != nil {
		slog.Info(".env no encontrado, usando variables del sistema")
	}

	// --- 2. Conectar a la base de datos ---
	if err := db.Conectar(); err != nil {
		slog.Error("error conectando a la base de datos", "error", err)
		os.Exit(1)
	}
	defer db.Cerrar()

	// --- 3. Ejecutar migraciones ---
	if err := db.EjecutarMigraciones("migrations"); err != nil {
		slog.Error("error ejecutando migraciones", "error", err)
		os.Exit(1)
	}

	// --- 4. Registrar rutas ---
	mux := http.NewServeMux()
	registrarRutas(mux)

	// --- 5. Arrancar el servidor ---
	puerto := os.Getenv("PORT")
	if puerto == "" {
		puerto = "8080"
	}

	servidor := &http.Server{
		Addr:    ":" + puerto,
		Handler: corsMiddleware(mux),
	}

	// Escuchamos señales del sistema operativo para apagado limpio.
	// Cuando apretás Ctrl+C, el servidor cierra las conexiones antes de salir.
	go func() {
		señales := make(chan os.Signal, 1)
		signal.Notify(señales, syscall.SIGINT, syscall.SIGTERM)
		<-señales
		slog.Info("apagando el servidor...")
		servidor.Close()
	}()

	slog.Info("servidor escuchando", "puerto", puerto)
	if err := servidor.ListenAndServe(); err != http.ErrServerClosed {
		slog.Error("error en el servidor", "error", err)
		os.Exit(1)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		permitido := origin == "" || origenCORSPermitido(origin)
		if origin != "" && permitido {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization, X-CSRF-Token")

		if r.Method == http.MethodOptions {
			if !permitido {
				http.Error(w, "origen no permitido", http.StatusForbidden)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func origenCORSPermitido(origin string) bool {
	origin = strings.TrimRight(strings.TrimSpace(origin), "/")
	permitidos := []string{strings.TrimRight(strings.TrimSpace(os.Getenv("APP_URL")), "/")}
	permitidos = append(permitidos, strings.Split(os.Getenv("CORS_ALLOWED_ORIGINS"), ",")...)
	for _, permitido := range permitidos {
		if strings.EqualFold(origin, strings.TrimRight(strings.TrimSpace(permitido), "/")) && permitido != "" {
			return true
		}
	}

	if strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production") {
		return false
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	host := u.Hostname()
	return strings.EqualFold(host, "localhost") || (net.ParseIP(host) != nil && net.ParseIP(host).IsLoopback())
}
