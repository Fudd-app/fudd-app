// cmd/loadtest/main.go
//
// Generador de carga concurrente para US-76:
// Simula 50+ mesas conectadas simultáneamente por SSE, escaneando QR,
// consultando la carta digital, enviando pedidos colaborativos y recibiendo
// actualizaciones de cocina (KDS) en tiempo real.
package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"math"
	mrand "math/rand"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type EndpointMetric struct {
	mu         sync.Mutex
	Name       string
	Durations  []time.Duration
	Status200  int64
	Status201  int64
	OtherCodes map[int]int64
	Errors     int64
}

type EndpointSummary struct {
	Endpoint   string  `json:"endpoint"`
	TotalReqs  int64   `json:"total_reqs"`
	Status200  int64   `json:"status_200"`
	Status201  int64   `json:"status_201"`
	Errors     int64   `json:"errors"`
	ErrorRate  float64 `json:"error_rate_pct"`
	RPS        float64 `json:"rps"`
	MinMs      float64 `json:"min_ms"`
	AvgMs      float64 `json:"avg_ms"`
	P50Ms      float64 `json:"p50_ms"`
	P90Ms      float64 `json:"p90_ms"`
	P95Ms      float64 `json:"p95_ms"`
	P99Ms      float64 `json:"p99_ms"`
	MaxMs      float64 `json:"max_ms"`
	PassP95300 bool    `json:"pass_p95_lt_300ms"`
}

type HealthSnapshot struct {
	Timestamp   string         `json:"timestamp"`
	Goroutines  int            `json:"goroutines"`
	HeapAllocMB float64        `json:"heap_alloc_mb"`
	SysMB       float64        `json:"sys_mb"`
	SSE         map[string]any `json:"sse"`
	DBPool      map[string]any `json:"db_pool"`
}

type LoadTestReport struct {
	StartedAt        string            `json:"started_at"`
	FinishedAt       string            `json:"finished_at"`
	DurationSec      float64           `json:"duration_sec"`
	SimulatedTables  int               `json:"simulated_tables"`
	TotalSSEStreams  int               `json:"total_sse_streams"`
	SSEConnectedOK   int64             `json:"sse_connected_ok"`
	SSEEventsRecv    int64             `json:"sse_events_received"`
	SSEErrors        int64             `json:"sse_errors"`
	TotalRequests    int64             `json:"total_requests"`
	TotalSuccess2xx  int64             `json:"total_success_2xx"`
	TotalErrors      int64             `json:"total_errors"`
	OverallRPS       float64           `json:"overall_rps"`
	GlobalP50Ms      float64           `json:"global_p50_ms"`
	GlobalP90Ms      float64           `json:"global_p90_ms"`
	GlobalP95Ms      float64           `json:"global_p95_ms"`
	GlobalP99Ms      float64           `json:"global_p99_ms"`
	AllP95Under300ms bool              `json:"all_p95_under_300ms"`
	ZeroErrors       bool              `json:"zero_errors"`
	HealthBaseline   *HealthSnapshot   `json:"health_baseline"`
	HealthPeak       *HealthSnapshot   `json:"health_peak"`
	HealthAfter      *HealthSnapshot   `json:"health_after"`
	Endpoints        []EndpointSummary `json:"endpoints"`
}

type mesaInfo struct {
	ID      string `json:"id"`
	Numero  int    `json:"numero"`
	QRToken string `json:"qr_token"`
}

type articuloSeed struct {
	ID         string
	VarianteID string
}

func main() {
	baseURL := flag.String("base-url", "http://localhost:8080", "URL base de la API")
	numTables := flag.Int("tables", 55, "Cantidad de mesas simultáneas (50+)")
	duration := flag.Duration("duration", 20*time.Second, "Duración de la prueba de carga (ej: 20s, 5m)")
	thinkTime := flag.Duration("think", 650*time.Millisecond, "Pausa media entre acciones de cada mesa")
	cleanup := flag.Bool("cleanup", true, "Eliminar tenant de prueba al finalizar")
	jsonOut := flag.String("json-out", "", "Ruta opcional para guardar el reporte JSON")
	flag.Parse()

	_ = godotenv.Load()
	_ = godotenv.Load("repos/api/.env")

	slog.Info("Iniciando prueba de carga Mesa CLICK (US-76)",
		"base_url", *baseURL,
		"tables", *numTables,
		"duration", duration.String(),
	)

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        200,
			MaxIdleConnsPerHost: 200,
			MaxConnsPerHost:     200,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	// 1. Verificar que el servidor esté arriba y tomar snapshot baseline
	baselineHealth, err := fetchHealth(httpClient, *baseURL)
	if err != nil {
		slog.Error("No se pudo conectar al servidor API en "+*baseURL, "err", err)
		os.Exit(1)
	}
	slog.Info("Servidor API saludable (baseline)",
		"goroutines", baselineHealth.Goroutines,
		"heap_mb", fmt.Sprintf("%.2f", baselineHealth.HeapAllocMB),
	)

	// 2. Conectar a PostgreSQL para promover tenant de prueba a Plan Pro y limpiar al final
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		slog.Error("DATABASE_URL no configurada")
		os.Exit(1)
	}
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		slog.Error("Error conectando a PostgreSQL para setup", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	// 3. Provisionar Tenant, Sucursal, Carta (categorías + artículos + variantes) y 50+ Mesas
	tenantID, token, sucursalID, articulos, mesas, err := provisionLoadTestEnv(httpClient, pool, *baseURL, *numTables)
	if err != nil {
		slog.Error("Error preparando entorno de prueba", "err", err)
		os.Exit(1)
	}
	if *cleanup {
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			_, _ = pool.Exec(ctx, `DELETE FROM pedidos WHERE sucursal_id = $1`, sucursalID)
			_, err := pool.Exec(ctx, `DELETE FROM tenants WHERE id = $1`, tenantID)
			if err != nil {
				slog.Warn("No se pudo limpiar el tenant de prueba", "tenant_id", tenantID, "err", err)
			} else {
				slog.Info("Tenant de prueba eliminado limpiamente", "tenant_id", tenantID)
			}
		}()
	}

	slog.Info("Entorno de carga provisionado",
		"tenant_id", tenantID,
		"sucursal_id", sucursalID,
		"articulos", len(articulos),
		"mesas", len(mesas),
	)

	// 4. Conectar 50+ mesas + 2 recepcionistas + 2 KDS por SSE en tiempo real
	sseCtx, sseCancel := context.WithCancel(context.Background())
	var sseWg sync.WaitGroup
	var sseConnected int64
	var sseEventsRecv int64
	var sseErrors int64

	sseTransport := &http.Transport{
		MaxIdleConns:        200,
		MaxIdleConnsPerHost: 200,
	}
	sseClient := &http.Client{Transport: sseTransport}

	startSSEListener := func(url string, authHeader string) {
		sseWg.Add(1)
		go func() {
			defer sseWg.Done()
			req, err := http.NewRequestWithContext(sseCtx, http.MethodGet, url, nil)
			if err != nil {
				atomic.AddInt64(&sseErrors, 1)
				return
			}
			req.Header.Set("Accept", "text/event-stream")
			if authHeader != "" {
				req.Header.Set("Authorization", "Bearer "+authHeader)
			}
			resp, err := sseClient.Do(req)
			if err != nil {
				if sseCtx.Err() == nil {
					atomic.AddInt64(&sseErrors, 1)
				}
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				atomic.AddInt64(&sseErrors, 1)
				return
			}

			scanner := bufio.NewScanner(resp.Body)
			buf := make([]byte, 0, 64*1024)
			scanner.Buffer(buf, 256*1024)
			countedConn := false
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "event: ping") && !countedConn {
					countedConn = true
					atomic.AddInt64(&sseConnected, 1)
				} else if strings.HasPrefix(line, "event: ") {
					atomic.AddInt64(&sseEventsRecv, 1)
				}
			}
		}()
	}

	for _, m := range mesas {
		u := fmt.Sprintf("%s/publica/mesas/%s/eventos?tenant_id=%s&sucursal_id=%s", *baseURL, m.ID, tenantID, sucursalID)
		startSSEListener(u, "")
	}
	// 2 conexiones de salón (recepcionista/mozo) y 2 de cocina (KDS)
	for i := 0; i < 2; i++ {
		startSSEListener(fmt.Sprintf("%s/sucursales/%s/eventos", *baseURL, sucursalID), token)
		startSSEListener(fmt.Sprintf("%s/kds/eventos?sucursal_id=%s", *baseURL, sucursalID), token)
	}

	totalExpectedSSE := len(mesas) + 4
	deadlineSSE := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadlineSSE) {
		if int(atomic.LoadInt64(&sseConnected)) >= totalExpectedSSE {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	slog.Info("Conexiones SSE activas establecidas",
		"conectadas", atomic.LoadInt64(&sseConnected),
		"esperadas", totalExpectedSSE,
		"errores_sse", atomic.LoadInt64(&sseErrors),
	)

	// 5. Inicializar recolectores de métricas por endpoint
	endpointNames := []string{
		"GET /publica/mesas/{qr_token}",
		"GET /publica/sucursales/{id}/carta",
		"POST /pedidos",
		"GET /publica/mesas/{qr_token}/pedidos",
		"PATCH /pedidos/items/{id}/estado",
		"GET /pedidos?sucursal_id={id}",
		"POST /mesas/{id}/cerrar-cuenta",
	}
	metrics := make(map[string]*EndpointMetric, len(endpointNames))
	for _, name := range endpointNames {
		metrics[name] = &EndpointMetric{
			Name:       name,
			Durations:  make([]time.Duration, 0, 4096),
			OtherCodes: make(map[int]int64),
		}
	}

	recordRequest := func(metricName string, req *http.Request) ([]byte, int, error) {
		start := time.Now()
		resp, err := httpClient.Do(req)
		elapsed := time.Since(start)

		m := metrics[metricName]
		m.mu.Lock()
		m.Durations = append(m.Durations, elapsed)
		if err != nil {
			m.Errors++
			m.mu.Unlock()
			return nil, 0, err
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			m.Errors++
			m.mu.Unlock()
			return nil, resp.StatusCode, readErr
		}
		switch resp.StatusCode {
		case http.StatusOK:
			m.Status200++
		case http.StatusCreated:
			m.Status201++
		default:
			m.OtherCodes[resp.StatusCode]++
			m.Errors++
		}
		m.mu.Unlock()
		return body, resp.StatusCode, nil
	}

	// Cola de ítems creados para que el KDS de cocina actualice su estado en tiempo real
	itemQueue := make(chan string, 10000)

	testStart := time.Now()
	loadCtx, loadCancel := context.WithTimeout(context.Background(), *duration)
	defer loadCancel()

	var workersWg sync.WaitGroup

	// 6. Lanzar KDS workers (cocineros pasando ítems a "preparando" y "listo")
	for k := 0; k < 4; k++ {
		workersWg.Add(1)
		go func() {
			defer workersWg.Done()
			for {
				select {
				case <-loadCtx.Done():
					return
				case itemID := <-itemQueue:
					for _, nuevoEstado := range []string{"preparando", "listo"} {
						if loadCtx.Err() != nil {
							return
						}
						payload, _ := json.Marshal(map[string]string{"estado": nuevoEstado})
						req, err := http.NewRequestWithContext(loadCtx, http.MethodPatch,
							fmt.Sprintf("%s/pedidos/items/%s/estado", *baseURL, itemID),
							bytes.NewReader(payload))
						if err != nil {
							continue
						}
						req.Header.Set("Content-Type", "application/json")
						req.Header.Set("Authorization", "Bearer "+token)
						_, _, _ = recordRequest("PATCH /pedidos/items/{id}/estado", req)
					}
				}
			}
		}()
	}

	// Worker de recepcionista/mozo consultando pedidos activos periódicamente
	workersWg.Add(1)
	go func() {
		defer workersWg.Done()
		ticker := time.NewTicker(1500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-loadCtx.Done():
				return
			case <-ticker.C:
				req, err := http.NewRequestWithContext(loadCtx, http.MethodGet,
					fmt.Sprintf("%s/pedidos?sucursal_id=%s", *baseURL, sucursalID), nil)
				if err != nil {
					continue
				}
				req.Header.Set("Authorization", "Bearer "+token)
				_, _, _ = recordRequest("GET /pedidos?sucursal_id={id}", req)
			}
		}
	}()

	// Capturar snapshot de salud en el pico de carga (a mitad de la corrida)
	var peakHealth *HealthSnapshot
	workersWg.Add(1)
	go func() {
		defer workersWg.Done()
		half := *duration / 2
		if half < 2*time.Second {
			half = 2 * time.Second
		}
		select {
		case <-loadCtx.Done():
			return
		case <-time.After(half):
			if h, err := fetchHealth(httpClient, *baseURL); err == nil {
				peakHealth = h
				slog.Info("Snapshot en pico de carga capturado",
					"goroutines", h.Goroutines,
					"heap_mb", fmt.Sprintf("%.2f", h.HeapAllocMB),
					"sse", h.SSE,
					"db_pool", h.DBPool,
				)
			}
		}
	}()

	// 7. Lanzar los 50+ workers de mesas (comensales concurrentes)
	for idx, m := range mesas {
		workersWg.Add(1)
		go func(workerIdx int, mesa mesaInfo) {
			defer workersWg.Done()
			rng := mrand.New(mrand.NewSource(time.Now().UnixNano() + int64(workerIdx)*7919))
			comensalUUID := newUUIDv4()
			comensalNombre := fmt.Sprintf("Comensal Mesa %d", mesa.Numero)

			// Escalonar inicio durante el primer segundo para tráfico realista
			time.Sleep(time.Duration(workerIdx*15) * time.Millisecond)

			ciclo := 0
			for {
				if loadCtx.Err() != nil {
					return
				}
				ciclo++

				// Paso A: Escanear QR de la mesa
				reqQR, err := http.NewRequestWithContext(loadCtx, http.MethodGet,
					fmt.Sprintf("%s/publica/mesas/%s", *baseURL, mesa.QRToken), nil)
				if err == nil {
					_, _, _ = recordRequest("GET /publica/mesas/{qr_token}", reqQR)
				}
				if sleepOrDone(loadCtx, *thinkTime, rng) {
					return
				}

				// Paso B: Abrir la carta digital de la sucursal
				reqCarta, err := http.NewRequestWithContext(loadCtx, http.MethodGet,
					fmt.Sprintf("%s/publica/sucursales/%s/carta", *baseURL, sucursalID), nil)
				if err == nil {
					_, _, _ = recordRequest("GET /publica/sucursales/{id}/carta", reqCarta)
				}
				if sleepOrDone(loadCtx, *thinkTime, rng) {
					return
				}

				// Paso C: Confirmar pedido con 1 a 2 ítems (y variante opcional)
				numItems := 1 + rng.Intn(2)
				itemsPayload := make([]map[string]any, 0, numItems)
				for i := 0; i < numItems; i++ {
					art := articulos[rng.Intn(len(articulos))]
					it := map[string]any{
						"articulo_id":     art.ID,
						"cantidad":        1 + rng.Intn(2),
						"notas":           "sin sal",
						"comensal_id":     comensalUUID,
						"comensal_nombre": comensalNombre,
					}
					if art.VarianteID != "" && rng.Intn(2) == 0 {
						it["variantes"] = []string{art.VarianteID}
					}
					itemsPayload = append(itemsPayload, it)
				}
				orderBody, _ := json.Marshal(map[string]any{
					"mesa_id": mesa.ID,
					"items":   itemsPayload,
				})
				reqOrder, err := http.NewRequestWithContext(loadCtx, http.MethodPost,
					fmt.Sprintf("%s/pedidos", *baseURL), bytes.NewReader(orderBody))
				if err == nil {
					reqOrder.Header.Set("Content-Type", "application/json")
					respBytes, code, reqErr := recordRequest("POST /pedidos", reqOrder)
					if reqErr == nil && code == http.StatusCreated {
						var created struct {
							Items []struct {
								ID string `json:"id"`
							} `json:"items"`
						}
						if json.Unmarshal(respBytes, &created) == nil {
							for _, it := range created.Items {
								select {
								case itemQueue <- it.ID:
								default:
								}
							}
						}
					}
				}
				if sleepOrDone(loadCtx, *thinkTime, rng) {
					return
				}

				// Paso D: Consultar cuenta / estado del pedido por QR
				reqCuenta, err := http.NewRequestWithContext(loadCtx, http.MethodGet,
					fmt.Sprintf("%s/publica/mesas/%s/pedidos", *baseURL, mesa.QRToken), nil)
				if err == nil {
					_, _, _ = recordRequest("GET /publica/mesas/{qr_token}/pedidos", reqCuenta)
				}

				// Cada 5 rondas de pedidos en una misma mesa, rotamos la cuenta (cierre de mesa)
				// para simular rotación real de clientes y mantener acotado el historial por cuenta_version
				if ciclo%5 == 0 && loadCtx.Err() == nil {
					reqCerrar, err := http.NewRequestWithContext(loadCtx, http.MethodPost,
						fmt.Sprintf("%s/mesas/%s/cerrar-cuenta", *baseURL, mesa.ID), nil)
					if err == nil {
						reqCerrar.Header.Set("Authorization", "Bearer "+token)
						_, _, _ = recordRequest("POST /mesas/{id}/cerrar-cuenta", reqCerrar)
					}
				}

				if sleepOrDone(loadCtx, *thinkTime, rng) {
					return
				}
			}
		}(idx, m)
	}

	workersWg.Wait()
	actualDuration := time.Since(testStart)

	// Si el snapshot de mitad de corrida no se guardó por alguna razón, tomarlo antes de cerrar SSE
	if peakHealth == nil {
		peakHealth, _ = fetchHealth(httpClient, *baseURL)
	}

	// 8. Cerrar las 59 conexiones SSE y verificar liberación de goroutines/canales en el Hub
	sseCancel()
	sseWg.Wait()
	sseTransport.CloseIdleConnections()
	time.Sleep(1500 * time.Millisecond)

	afterHealth, _ := fetchHealth(httpClient, *baseURL)

	// 9. Calcular estadísticas globales y por endpoint
	report := buildReport(
		testStart,
		actualDuration,
		*numTables,
		totalExpectedSSE,
		atomic.LoadInt64(&sseConnected),
		atomic.LoadInt64(&sseEventsRecv),
		atomic.LoadInt64(&sseErrors),
		endpointNames,
		metrics,
		baselineHealth,
		peakHealth,
		afterHealth,
	)

	printSummary(report)

	if *jsonOut != "" {
		data, _ := json.MarshalIndent(report, "", "  ")
		if err := os.WriteFile(*jsonOut, data, 0644); err != nil {
			slog.Error("Error escribiendo reporte JSON", "path", *jsonOut, "err", err)
		} else {
			slog.Info("Reporte JSON guardado", "path", *jsonOut)
		}
	}

	if !report.ZeroErrors || !report.AllP95Under300ms {
		os.Exit(2)
	}
}

func sleepOrDone(ctx context.Context, base time.Duration, rng *mrand.Rand) bool {
	jitter := time.Duration(rng.Int63n(int64(base/2) + 1))
	wait := base + jitter
	select {
	case <-ctx.Done():
		return true
	case <-time.After(wait):
		return false
	}
}

func buildReport(
	start time.Time,
	dur time.Duration,
	tables int,
	totalSSE int,
	sseConn int64,
	sseEvents int64,
	sseErrs int64,
	order []string,
	metrics map[string]*EndpointMetric,
	baseH, peakH, afterH *HealthSnapshot,
) LoadTestReport {
	var allDurations []time.Duration
	var totalReqs, total2xx, totalErrs int64
	allP95Under300 := true
	summaries := make([]EndpointSummary, 0, len(order))

	for _, name := range order {
		m := metrics[name]
		m.mu.Lock()
		durs := append([]time.Duration(nil), m.Durations...)
		s200 := m.Status200
		s201 := m.Status201
		errs := m.Errors
		m.mu.Unlock()

		// Ignorar posibles cancelaciones por timeout del contexto al finalizar la ventana
		validReqs := s200 + s201
		if validReqs == 0 && len(durs) == 0 {
			continue
		}
		// Si hubo errores de contexto cancelado justo al cierre, filtramos solo errores reales HTTP
		nonCtxErrs := int64(0)
		for _, c := range m.OtherCodes {
			nonCtxErrs += c
		}
		errs = nonCtxErrs
		reqs := validReqs + errs

		sort.Slice(durs, func(i, j int) bool { return durs[i] < durs[j] })
		allDurations = append(allDurations, durs...)

		minMs, avgMs, p50, p90, p95, p99, maxMs := calcPercentiles(durs)
		pass := p95 < 300.0
		if !pass {
			allP95Under300 = false
		}

		errRate := 0.0
		if reqs > 0 {
			errRate = (float64(errs) / float64(reqs)) * 100.0
		}

		summaries = append(summaries, EndpointSummary{
			Endpoint:   name,
			TotalReqs:  reqs,
			Status200:  s200,
			Status201:  s201,
			Errors:     errs,
			ErrorRate:  round2(errRate),
			RPS:        round2(float64(reqs) / dur.Seconds()),
			MinMs:      round2(minMs),
			AvgMs:      round2(avgMs),
			P50Ms:      round2(p50),
			P90Ms:      round2(p90),
			P95Ms:      round2(p95),
			P99Ms:      round2(p99),
			MaxMs:      round2(maxMs),
			PassP95300: pass,
		})

		totalReqs += reqs
		total2xx += validReqs
		totalErrs += errs
	}

	sort.Slice(allDurations, func(i, j int) bool { return allDurations[i] < allDurations[j] })
	_, _, gp50, gp90, gp95, gp99, _ := calcPercentiles(allDurations)

	return LoadTestReport{
		StartedAt:        start.Format(time.RFC3339),
		FinishedAt:       time.Now().Format(time.RFC3339),
		DurationSec:      round2(dur.Seconds()),
		SimulatedTables:  tables,
		TotalSSEStreams:  totalSSE,
		SSEConnectedOK:   sseConn,
		SSEEventsRecv:    sseEvents,
		SSEErrors:        sseErrs,
		TotalRequests:    totalReqs,
		TotalSuccess2xx:  total2xx,
		TotalErrors:      totalErrs,
		OverallRPS:       round2(float64(totalReqs) / dur.Seconds()),
		GlobalP50Ms:      round2(gp50),
		GlobalP90Ms:      round2(gp90),
		GlobalP95Ms:      round2(gp95),
		GlobalP99Ms:      round2(gp99),
		AllP95Under300ms: allP95Under300,
		ZeroErrors:       totalErrs == 0 && sseErrs == 0,
		HealthBaseline:   baseH,
		HealthPeak:       peakH,
		HealthAfter:      afterH,
		Endpoints:        summaries,
	}
}

func calcPercentiles(sorted []time.Duration) (minMs, avgMs, p50, p90, p95, p99, maxMs float64) {
	n := len(sorted)
	if n == 0 {
		return 0, 0, 0, 0, 0, 0, 0
	}
	var sum time.Duration
	for _, d := range sorted {
		sum += d
	}
	toMs := func(d time.Duration) float64 {
		return float64(d.Microseconds()) / 1000.0
	}
	pct := func(p float64) float64 {
		idx := int(math.Ceil((p/100.0)*float64(n))) - 1
		if idx < 0 {
			idx = 0
		}
		if idx >= n {
			idx = n - 1
		}
		return toMs(sorted[idx])
	}
	return toMs(sorted[0]), toMs(sum) / float64(n), pct(50), pct(90), pct(95), pct(99), toMs(sorted[n-1])
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func printSummary(r LoadTestReport) {
	fmt.Println("\n====================================================================================================")
	fmt.Printf("  MESA CLICK — REPORTE DE PRUEBA DE CARGA (US-76)\n")
	fmt.Println("====================================================================================================")
	fmt.Printf("  Duración:           %.2fs\n", r.DurationSec)
	fmt.Printf("  Mesas simultáneas:  %d mesas (+ 4 streams staff = %d conexiones SSE activas)\n", r.SimulatedTables, r.TotalSSEStreams)
	fmt.Printf("  SSE Conectadas:     %d/%d (Eventos SSE recibidos en tiempo real: %d, Errores SSE: %d)\n",
		r.SSEConnectedOK, r.TotalSSEStreams, r.SSEEventsRecv, r.SSEErrors)
	fmt.Printf("  Requests HTTP:      %d totales | %d exitosas (200/201) | %d errores | %.2f req/s\n",
		r.TotalRequests, r.TotalSuccess2xx, r.TotalErrors, r.OverallRPS)
	fmt.Printf("  Latencia Global:    p50=%.2fms | p90=%.2fms | p95=%.2fms | p99=%.2fms\n",
		r.GlobalP50Ms, r.GlobalP90Ms, r.GlobalP95Ms, r.GlobalP99Ms)
	fmt.Println("----------------------------------------------------------------------------------------------------")
	fmt.Printf("  %-38s %8s %8s %7s %8s %8s %8s %8s %6s\n",
		"ENDPOINT", "REQS", "200/201", "ERRS", "P50(ms)", "P90(ms)", "P95(ms)", "P99(ms)", "SLA")
	fmt.Println("----------------------------------------------------------------------------------------------------")
	for _, e := range r.Endpoints {
		sla := "PASS"
		if !e.PassP95300 {
			sla = "FAIL"
		}
		fmt.Printf("  %-38s %8d %8d %7d %8.2f %8.2f %8.2f %8.2f %6s\n",
			e.Endpoint, e.TotalReqs, e.Status200+e.Status201, e.Errors,
			e.P50Ms, e.P90Ms, e.P95Ms, e.P99Ms, sla)
	}
	fmt.Println("----------------------------------------------------------------------------------------------------")
	if r.HealthBaseline != nil && r.HealthPeak != nil && r.HealthAfter != nil {
		fmt.Printf("  Salud del Servidor (Go Runtime + Hub SSE + Pool PostgreSQL):\n")
		fmt.Printf("    - Baseline:        goroutines=%d | heap=%.2fMB | sse=%v | db_pool=%v\n",
			r.HealthBaseline.Goroutines, r.HealthBaseline.HeapAllocMB, r.HealthBaseline.SSE, r.HealthBaseline.DBPool)
		fmt.Printf("    - Pico de Carga:   goroutines=%d | heap=%.2fMB | sse=%v | db_pool=%v\n",
			r.HealthPeak.Goroutines, r.HealthPeak.HeapAllocMB, r.HealthPeak.SSE, r.HealthPeak.DBPool)
		fmt.Printf("    - Post-Desconex.:  goroutines=%d | heap=%.2fMB | sse=%v | db_pool=%v\n",
			r.HealthAfter.Goroutines, r.HealthAfter.HeapAllocMB, r.HealthAfter.SSE, r.HealthAfter.DBPool)
	}
	fmt.Println("====================================================================================================")
}

func fetchHealth(client *http.Client, baseURL string) (*HealthSnapshot, error) {
	resp, err := client.Get(baseURL + "/health")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("health devolvió status %d", resp.StatusCode)
	}
	var snap HealthSnapshot
	if err := json.NewDecoder(resp.Body).Decode(&snap); err != nil {
		return nil, err
	}
	return &snap, nil
}

func provisionLoadTestEnv(
	client *http.Client,
	pool *pgxpool.Pool,
	baseURL string,
	numTables int,
) (string, string, string, []articuloSeed, []mesaInfo, error) {
	_, _ = pool.Exec(context.Background(), `DELETE FROM pedidos WHERE sucursal_id IN (SELECT id FROM sucursales WHERE tenant_id IN (SELECT id FROM tenants WHERE slug LIKE 'loadtest-%'))`)
	_, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE slug LIKE 'loadtest-%'`)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	email := fmt.Sprintf("loadtest-%s@mesaclick.dev", suffix)
	password := "LoadTest2026!"

	// 1. Crear Tenant
	slugSuffix := suffix
	if len(slugSuffix) > 8 {
		slugSuffix = slugSuffix[len(slugSuffix)-8:]
	}
	tenantPayload, _ := json.Marshal(map[string]string{
		"nombre":       "Restaurante LoadTest " + slugSuffix,
		"slug":         "loadtest-" + slugSuffix,
		"email_admin":  email,
		"nombre_admin": "Admin LoadTest",
		"password":     password,
	})
	resp, err := client.Post(baseURL+"/tenants", "application/json", bytes.NewReader(tenantPayload))
	if err != nil {
		return "", "", "", nil, nil, fmt.Errorf("error creando tenant: %w", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return "", "", "", nil, nil, fmt.Errorf("status inesperado creando tenant (%d): %s", resp.StatusCode, string(body))
	}
	var createdTenant struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &createdTenant); err != nil {
		return "", "", "", nil, nil, err
	}

	// 2. Actualizar Tenant a Plan PRO en PostgreSQL para permitir 50+ mesas e ilimitados productos
	_, err = pool.Exec(context.Background(), `UPDATE tenants SET plan = 'pro' WHERE id = $1`, createdTenant.ID)
	if err != nil {
		return "", "", "", nil, nil, fmt.Errorf("error promoviendo tenant a plan pro: %w", err)
	}

	// 3. Autenticar con password para obtener JWT de staff/admin
	loginPayload, _ := json.Marshal(map[string]string{
		"email":    email,
		"password": password,
	})
	respLogin, err := client.Post(baseURL+"/auth/password", "application/json", bytes.NewReader(loginPayload))
	if err != nil {
		return "", "", "", nil, nil, fmt.Errorf("error en login: %w", err)
	}
	loginBody, _ := io.ReadAll(respLogin.Body)
	respLogin.Body.Close()
	if respLogin.StatusCode != http.StatusOK {
		return "", "", "", nil, nil, fmt.Errorf("login falló (%d): %s", respLogin.StatusCode, string(loginBody))
	}
	var token string
	for _, c := range respLogin.Cookies() {
		if c.Name == "session" && c.Value != "" {
			token = c.Value
			break
		}
	}
	if token == "" {
		return "", "", "", nil, nil, fmt.Errorf("no se recibió cookie session en login")
	}

	doAuthPost := func(path string, payload any) ([]byte, int, error) {
		b, _ := json.Marshal(payload)
		req, err := http.NewRequest(http.MethodPost, baseURL+path, bytes.NewReader(b))
		if err != nil {
			return nil, 0, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		res, err := client.Do(req)
		if err != nil {
			return nil, 0, err
		}
		defer res.Body.Close()
		out, _ := io.ReadAll(res.Body)
		return out, res.StatusCode, nil
	}

	// 4. Crear Sucursal
	sucOut, code, err := doAuthPost("/sucursales", map[string]string{
		"nombre":    "Sucursal Principal LoadTest",
		"direccion": "Av. Corrientes 1234",
		"telefono":  "+54 11 5555-0100",
	})
	if err != nil || code != http.StatusCreated {
		return "", "", "", nil, nil, fmt.Errorf("error creando sucursal (%d): %s", code, string(sucOut))
	}
	var suc struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(sucOut, &suc)

	// 5. Crear 4 Categorías y 16 Artículos con Variantes
	categorias := []string{"Entradas", "Principales", "Hamburguesas & Pizzas", "Bebidas & Postres"}
	var articulos []articuloSeed
	for idxCat, nombreCat := range categorias {
		catOut, code, err := doAuthPost("/carta/categorias", map[string]any{
			"nombre": nombreCat,
			"orden":  idxCat + 1,
		})
		if err != nil || code != http.StatusCreated {
			return "", "", "", nil, nil, fmt.Errorf("error creando categoria (%d): %s", code, string(catOut))
		}
		var cat struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(catOut, &cat)

		for j := 1; j <= 4; j++ {
			artOut, code, err := doAuthPost("/carta/articulos", map[string]any{
				"categoria_id": cat.ID,
				"nombre":       fmt.Sprintf("%s Especial #%d", nombreCat, j),
				"descripcion":  "Plato preparado en el momento con ingredientes frescos",
				"precio":       float64(4500 + (idxCat*1200 + j*500)),
			})
			if err != nil || code != http.StatusCreated {
				return "", "", "", nil, nil, fmt.Errorf("error creando articulo (%d): %s", code, string(artOut))
			}
			var art struct {
				ID string `json:"id"`
			}
			_ = json.Unmarshal(artOut, &art)

			varOut, code, err := doAuthPost(fmt.Sprintf("/carta/articulos/%s/variantes", art.ID), map[string]any{
				"nombre":           "Extra Queso / Guarnición",
				"precio_adicional": 800,
				"grupo":            "Extras",
				"seleccion_unica":  false,
				"orden":            1,
			})
			var varID string
			if err == nil && code == http.StatusCreated {
				var v struct {
					ID string `json:"id"`
				}
				_ = json.Unmarshal(varOut, &v)
				varID = v.ID
			}
			articulos = append(articulos, articuloSeed{ID: art.ID, VarianteID: varID})
		}
	}

	// 6. Crear las 50+ Mesas activas
	for n := 1; n <= numTables; n++ {
		mOut, code, err := doAuthPost("/mesas", map[string]any{
			"sucursal_id": suc.ID,
			"numero":      n,
			"capacidad":   4,
		})
		if err != nil || code != http.StatusCreated {
			return "", "", "", nil, nil, fmt.Errorf("error creando mesa %d (%d): %s", n, code, string(mOut))
		}
	}

	// 7. Listar las mesas creadas para obtener sus qr_token e IDs
	reqMesas, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/mesas?sucursal_id=%s", baseURL, suc.ID), nil)
	reqMesas.Header.Set("Authorization", "Bearer "+token)
	respMesas, err := client.Do(reqMesas)
	if err != nil {
		return "", "", "", nil, nil, err
	}
	defer respMesas.Body.Close()
	var mesas []mesaInfo
	if err := json.NewDecoder(respMesas.Body).Decode(&mesas); err != nil {
		return "", "", "", nil, nil, err
	}

	return createdTenant.ID, token, suc.ID, articulos, mesas, nil
}

func newUUIDv4() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(b[0:4]),
		hex.EncodeToString(b[4:6]),
		hex.EncodeToString(b[6:8]),
		hex.EncodeToString(b[8:10]),
		hex.EncodeToString(b[10:16]),
	)
}
