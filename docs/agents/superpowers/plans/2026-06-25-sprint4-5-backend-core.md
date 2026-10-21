# Sprint 4–5 Backend Core Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the five core service modules (auth, tenant, carta, mesa, pedido) in the Go backend so Phase 3 (frontend integration) can begin on June 29.

**Architecture:** Each module lives in `internal/<name>/` and follows the pattern: `model.go` (types) → `store.go` (DB interface + pgx implementation) → `service.go` (business logic, depends on interface) → `handler.go` (HTTP, depends on service). Routes are registered in the root `rutas.go`. Tests target the service layer using mock stores.

**Tech Stack:** Go 1.22, pgx/v5, `golang-jwt/jwt/v5` (to add), standard `net/http`, `net/http/httptest` for tests.

## Global Constraints

- Module path: `github.com/aguirrepablo-iresm/mesa-click/api`
- DB pool available at `db.Pool` (`*pgxpool.Pool`) from `internal/db`
- Log with `slog` — no `fmt.Println` for server logs
- HTTP errors: always JSON `{"error": "mensaje"}` with appropriate status code
- All protected endpoints require JWT in `Authorization: Bearer <token>` header OR `session` httpOnly cookie
- No email provider yet — log magic links to stdout with `slog.Info("magic link", "url", link)`
- Run all commands from `repos/api/` directory

---

## File Map

```
repos/api/
├── go.mod                         ← modify: add golang-jwt/jwt/v5
├── go.sum                         ← auto-updated
├── rutas.go                       ← modify: wire all handlers
├── internal/
│   ├── db/db.go                   ← existing, no changes
│   ├── auth/
│   │   ├── model.go               ← create: Claims, UsuarioAuth, MagicToken, ctxKey
│   │   ├── store.go               ← create: Store interface + pgStore
│   │   ├── service.go             ← create: SolicitarLink, VerificarToken, GenerarJWT, ValidarJWT
│   │   ├── middleware.go          ← create: Requerir(next http.Handler) http.Handler
│   │   ├── handler.go             ← create: HandlerSolicitarLink, HandlerVerificarToken
│   │   └── service_test.go        ← create: tests for SolicitarLink, VerificarToken
│   ├── tenant/
│   │   ├── model.go               ← create: Tenant, OnboardingInput
│   │   ├── store.go               ← create: Store interface + pgStore
│   │   ├── service.go             ← create: Crear, ObtenerPorID
│   │   ├── handler.go             ← create: HandlerCrear, HandlerObtenerMe
│   │   └── service_test.go        ← create: tests for Crear
│   ├── carta/
│   │   ├── model.go               ← create: Categoria, Articulo
│   │   ├── store.go               ← create: Store interface + pgStore
│   │   ├── service.go             ← create: CRUD categorías + artículos
│   │   ├── handler.go             ← create: handlers + HandlerCartaPublica
│   │   └── service_test.go        ← create: tests for ListarCategorias, CrearArticulo
│   ├── mesa/
│   │   ├── model.go               ← create: Mesa
│   │   ├── store.go               ← create: Store interface + pgStore
│   │   ├── service.go             ← create: CRUD + GenerarQRToken
│   │   ├── handler.go             ← create: handlers + HandlerCartaPublicaPorToken
│   │   └── service_test.go        ← create: test for GenerarQRToken, Crear
│   └── pedido/
│       ├── model.go               ← create: Pedido, PedidoItem, NuevoPedidoInput
│       ├── store.go               ← create: Store interface + pgStore
│       ├── service.go             ← create: Crear, CambiarEstado, ListarActivos
│       ├── handler.go             ← create: HandlerCrear, HandlerCambiarEstado, HandlerListar
│       └── service_test.go        ← create: test for CambiarEstado validations
```

---

## Task 1: Auth — Magic Link + JWT + Middleware

**Files:**
- Create: `internal/auth/model.go`
- Create: `internal/auth/store.go`
- Create: `internal/auth/service.go`
- Create: `internal/auth/middleware.go`
- Create: `internal/auth/handler.go`
- Create: `internal/auth/service_test.go`
- Modify: `go.mod` and `go.sum`

**Interfaces:**
- Produces:
  - `auth.Requerir(next http.Handler) http.Handler` — middleware used in Tasks 2–5
  - `auth.ClaimsFromContext(ctx) *Claims` — extracts JWT claims from context in all handlers
  - `auth.HandlerSolicitarLink` — `http.HandlerFunc`
  - `auth.HandlerVerificarToken` — `http.HandlerFunc`

---

- [ ] **Step 1.1: Agregar dependencia JWT**

```bash
cd repos/api && go get github.com/golang-jwt/jwt/v5
```

Expected output: line added to go.mod and go.sum updated.

---

- [ ] **Step 1.2: Crear `internal/auth/model.go`**

```go
package auth

import (
	"context"
	"time"
)

type MagicToken struct {
	ID        string
	UsuarioID string
	Token     string
	ExpiresAt time.Time
	UsedAt    *time.Time
}

type UsuarioAuth struct {
	ID       string
	TenantID string
	Email    string
	Rol      string
}

type Claims struct {
	UsuarioID string `json:"usuario_id"`
	TenantID  string `json:"tenant_id"`
	Rol       string `json:"rol"`
}

type ctxKey string

const claimsKey ctxKey = "auth_claims"

func ClaimsFromContext(ctx context.Context) *Claims {
	c, _ := ctx.Value(claimsKey).(*Claims)
	return c
}
```

---

- [ ] **Step 1.3: Escribir test fallido para `GenerarJWT` / `ValidarJWT`**

Crear `internal/auth/service_test.go`:

```go
package auth_test

import (
	"testing"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
)

func TestGenerarYValidarJWT(t *testing.T) {
	secreto := "secreto-de-prueba"
	claims := &auth.Claims{
		UsuarioID: "user-123",
		TenantID:  "tenant-456",
		Rol:       "admin",
	}

	token, err := auth.GenerarJWT(claims, secreto)
	if err != nil {
		t.Fatalf("GenerarJWT error: %v", err)
	}
	if token == "" {
		t.Fatal("token vacío")
	}

	recuperadas, err := auth.ValidarJWT(token, secreto)
	if err != nil {
		t.Fatalf("ValidarJWT error: %v", err)
	}
	if recuperadas.UsuarioID != claims.UsuarioID {
		t.Errorf("UsuarioID: got %q, want %q", recuperadas.UsuarioID, claims.UsuarioID)
	}
	if recuperadas.TenantID != claims.TenantID {
		t.Errorf("TenantID: got %q, want %q", recuperadas.TenantID, claims.TenantID)
	}
	if recuperadas.Rol != claims.Rol {
		t.Errorf("Rol: got %q, want %q", recuperadas.Rol, claims.Rol)
	}
}

func TestValidarJWT_TokenInvalido(t *testing.T) {
	_, err := auth.ValidarJWT("token.invalido.firma", "secreto")
	if err == nil {
		t.Fatal("esperaba error con token inválido")
	}
}
```

---

- [ ] **Step 1.4: Ejecutar test para verificar que falla**

```bash
cd repos/api && go test ./internal/auth/... -v -run TestGenerarYValidarJWT
```

Expected: error de compilación — `auth.GenerarJWT` no existe aún.

---

- [ ] **Step 1.5: Crear `internal/auth/store.go`**

```go
package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/db"
)

type Store interface {
	ObtenerUsuarioPorEmail(ctx context.Context, email string) (*UsuarioAuth, error)
	GuardarToken(ctx context.Context, usuarioID, token string, expiresAt time.Time) (string, error)
	ObtenerToken(ctx context.Context, token string) (*MagicToken, error)
	MarcarTokenUsado(ctx context.Context, tokenID string) error
}

type pgStore struct{}

func NuevoStore() Store { return &pgStore{} }

func (s *pgStore) ObtenerUsuarioPorEmail(ctx context.Context, email string) (*UsuarioAuth, error) {
	u := &UsuarioAuth{}
	err := db.Pool.QueryRow(ctx,
		`SELECT id, tenant_id, email, rol FROM usuarios WHERE email = $1`, email,
	).Scan(&u.ID, &u.TenantID, &u.Email, &u.Rol)
	if err != nil {
		return nil, fmt.Errorf("usuario no encontrado: %w", err)
	}
	return u, nil
}

func (s *pgStore) GuardarToken(ctx context.Context, usuarioID, token string, expiresAt time.Time) (string, error) {
	var id string
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO magic_tokens (usuario_id, token, expires_at)
		 VALUES ($1, $2, $3) RETURNING id`,
		usuarioID, token, expiresAt,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("error guardando token: %w", err)
	}
	return id, nil
}

func (s *pgStore) ObtenerToken(ctx context.Context, token string) (*MagicToken, error) {
	mt := &MagicToken{}
	err := db.Pool.QueryRow(ctx,
		`SELECT id, usuario_id, token, expires_at, used_at
		 FROM magic_tokens WHERE token = $1`,
		token,
	).Scan(&mt.ID, &mt.UsuarioID, &mt.Token, &mt.ExpiresAt, &mt.UsedAt)
	if err != nil {
		return nil, fmt.Errorf("token no encontrado: %w", err)
	}
	return mt, nil
}

func (s *pgStore) MarcarTokenUsado(ctx context.Context, tokenID string) error {
	_, err := db.Pool.Exec(ctx,
		`UPDATE magic_tokens SET used_at = now() WHERE id = $1`, tokenID,
	)
	return err
}
```

---

- [ ] **Step 1.6: Crear `internal/auth/service.go`**

```go
package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Service struct {
	store Store
}

func NuevoService(s Store) *Service { return &Service{store: s} }

func (svc *Service) SolicitarLink(ctx context.Context, email string) error {
	usuario, err := svc.store.ObtenerUsuarioPorEmail(ctx, email)
	if err != nil {
		// No revelamos si el email existe o no — siempre respondemos OK
		slog.Warn("magic link solicitado para email no registrado", "email", email)
		return nil
	}

	token, err := generarTokenAleatorio()
	if err != nil {
		return fmt.Errorf("error generando token: %w", err)
	}

	expiresAt := time.Now().Add(15 * time.Minute)
	if _, err := svc.store.GuardarToken(ctx, usuario.ID, token, expiresAt); err != nil {
		return err
	}

	appURL := os.Getenv("APP_URL")
	if appURL == "" {
		appURL = "http://localhost:3000"
	}
	link := fmt.Sprintf("%s/auth/verify?token=%s", appURL, token)
	slog.Info("magic link generado", "email", email, "link", link)
	return nil
}

func (svc *Service) VerificarToken(ctx context.Context, token string) (*UsuarioAuth, error) {
	mt, err := svc.store.ObtenerToken(ctx, token)
	if err != nil {
		return nil, errors.New("token inválido")
	}
	if mt.UsedAt != nil {
		return nil, errors.New("token ya usado")
	}
	if time.Now().After(mt.ExpiresAt) {
		return nil, errors.New("token expirado")
	}
	if err := svc.store.MarcarTokenUsado(ctx, mt.ID); err != nil {
		return nil, err
	}
	return svc.store.ObtenerUsuarioPorEmail(ctx, mt.UsuarioID)
}

func GenerarJWT(claims *Claims, secreto string) (string, error) {
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"usuario_id": claims.UsuarioID,
		"tenant_id":  claims.TenantID,
		"rol":        claims.Rol,
		"exp":        time.Now().Add(24 * time.Hour * 30).Unix(),
	})
	return t.SignedString([]byte(secreto))
}

func ValidarJWT(tokenStr, secreto string) (*Claims, error) {
	t, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("método de firma inesperado: %v", t.Header["alg"])
		}
		return []byte(secreto), nil
	})
	if err != nil || !t.Valid {
		return nil, errors.New("token inválido")
	}
	mc, ok := t.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("claims inválidos")
	}
	return &Claims{
		UsuarioID: mc["usuario_id"].(string),
		TenantID:  mc["tenant_id"].(string),
		Rol:       mc["rol"].(string),
	}, nil
}

// VerificarToken necesita obtener el usuario por ID, no por email.
// Ajuste: el store busca por usuario_id en el segundo paso.
func (svc *Service) usuarioPorID(ctx context.Context, id string) (*UsuarioAuth, error) {
	// Reutilizamos la misma store — la query correcta está en store.go
	// Esta función es interna, se llama desde VerificarToken
	return nil, nil // se implementa en store (ver ajuste abajo)
}

func generarTokenAleatorio() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
```

**Nota:** `VerificarToken` llama a `ObtenerUsuarioPorEmail` pasando `mt.UsuarioID` (que es un UUID). Hay que agregar `ObtenerUsuarioPorID` al store. Actualizar store e interfaz:

En `store.go`, agregar al interface:
```go
ObtenerUsuarioPorID(ctx context.Context, id string) (*UsuarioAuth, error)
```

Y la implementación en `pgStore`:
```go
func (s *pgStore) ObtenerUsuarioPorID(ctx context.Context, id string) (*UsuarioAuth, error) {
	u := &UsuarioAuth{}
	err := db.Pool.QueryRow(ctx,
		`SELECT id, tenant_id, email, rol FROM usuarios WHERE id = $1`, id,
	).Scan(&u.ID, &u.TenantID, &u.Email, &u.Rol)
	if err != nil {
		return nil, fmt.Errorf("usuario no encontrado: %w", err)
	}
	return u, nil
}
```

Y en `service.go`, cambiar la última línea de `VerificarToken`:
```go
return svc.store.ObtenerUsuarioPorID(ctx, mt.UsuarioID)
```

(Eliminar la función `usuarioPorID` del service — era un placeholder.)

---

- [ ] **Step 1.7: Ejecutar tests — deben pasar**

```bash
cd repos/api && go test ./internal/auth/... -v -run TestGenerarYValidarJWT
cd repos/api && go test ./internal/auth/... -v -run TestValidarJWT_TokenInvalido
```

Expected: `PASS` para ambos.

---

- [ ] **Step 1.8: Crear `internal/auth/middleware.go`**

```go
package auth

import (
	"net/http"
	"os"
	"strings"
)

func Requerir(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		secreto := os.Getenv("JWT_SECRET")

		var tokenStr string

		// Primero intentamos Authorization header
		if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
			tokenStr = strings.TrimPrefix(h, "Bearer ")
		}

		// Fallback a cookie
		if tokenStr == "" {
			if c, err := r.Cookie("session"); err == nil {
				tokenStr = c.Value
			}
		}

		if tokenStr == "" {
			http.Error(w, `{"error":"no autorizado"}`, http.StatusUnauthorized)
			return
		}

		claims, err := ValidarJWT(tokenStr, secreto)
		if err != nil {
			http.Error(w, `{"error":"token inválido"}`, http.StatusUnauthorized)
			return
		}

		ctx := r.Context()
		ctx = setClaimsInContext(ctx, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func setClaimsInContext(ctx interface{ Value(any) any }, claims *Claims) interface{} {
	// Necesitamos context.WithValue — importar context
	return claims // placeholder
}
```

**Nota:** la función `setClaimsInContext` es un placeholder que no compila. Reemplazar `middleware.go` completo con la versión correcta:

```go
package auth

import (
	"context"
	"net/http"
	"os"
	"strings"
)

func Requerir(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		secreto := os.Getenv("JWT_SECRET")

		var tokenStr string

		if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
			tokenStr = strings.TrimPrefix(h, "Bearer ")
		}
		if tokenStr == "" {
			if c, err := r.Cookie("session"); err == nil {
				tokenStr = c.Value
			}
		}

		if tokenStr == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"no autorizado"}`))
			return
		}

		claims, err := ValidarJWT(tokenStr, secreto)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"token inválido"}`))
			return
		}

		ctx := context.WithValue(r.Context(), claimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
```

(Usar directamente esta versión final — ignorar el placeholder anterior.)

---

- [ ] **Step 1.9: Crear `internal/auth/handler.go`**

```go
package auth

import (
	"encoding/json"
	"net/http"
	"os"
)

type Handlers struct {
	svc *Service
}

func NuevosHandlers(svc *Service) *Handlers { return &Handlers{svc: svc} }

func (h *Handlers) SolicitarLink(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Email == "" {
		jsonError(w, "email requerido", http.StatusBadRequest)
		return
	}

	if err := h.svc.SolicitarLink(r.Context(), body.Email); err != nil {
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"mensaje": "si el email existe, recibirás un link"})
}

func (h *Handlers) VerificarToken(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		jsonError(w, "token requerido", http.StatusBadRequest)
		return
	}

	usuario, err := h.svc.VerificarToken(r.Context(), token)
	if err != nil {
		jsonError(w, err.Error(), http.StatusUnauthorized)
		return
	}

	secreto := os.Getenv("JWT_SECRET")
	jwt, err := GenerarJWT(&Claims{
		UsuarioID: usuario.ID,
		TenantID:  usuario.TenantID,
		Rol:       usuario.Rol,
	}, secreto)
	if err != nil {
		jsonError(w, "error generando sesión", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    jwt,
		HttpOnly: true,
		Path:     "/",
		MaxAge:   60 * 60 * 24 * 30,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": jwt})
}

func jsonError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
```

---

- [ ] **Step 1.10: Verificar que compila**

```bash
cd repos/api && go build ./...
```

Expected: sin errores.

---

- [ ] **Step 1.11: Commit**

```bash
cd repos/api && git add internal/auth/ go.mod go.sum
git commit -m "feat(auth): magic link + JWT + middleware"
```

---

## Task 2: Tenant — Onboarding del negocio

**Files:**
- Create: `internal/tenant/model.go`
- Create: `internal/tenant/store.go`
- Create: `internal/tenant/service.go`
- Create: `internal/tenant/handler.go`
- Create: `internal/tenant/service_test.go`
- Modify: `rutas.go` (agregar rutas de tenant)

**Interfaces:**
- Consumes: `auth.Requerir`, `auth.ClaimsFromContext`
- Produces:
  - `tenant.Handlers.Crear` — `http.HandlerFunc`, ruta `POST /tenants`
  - `tenant.Handlers.ObtenerMe` — `http.HandlerFunc`, ruta `GET /tenants/me`

---

- [ ] **Step 2.1: Crear `internal/tenant/model.go`**

```go
package tenant

type Tenant struct {
	ID             string `json:"id"`
	Nombre         string `json:"nombre"`
	NombreFantasia string `json:"nombre_fantasia,omitempty"`
	Rubro          string `json:"rubro"`
	Slug           string `json:"slug"`
	CreatedAt      string `json:"created_at"`
}

type OnboardingInput struct {
	Nombre         string `json:"nombre"`
	NombreFantasia string `json:"nombre_fantasia"`
	Rubro          string `json:"rubro"`
	Slug           string `json:"slug"`
	EmailAdmin     string `json:"email_admin"`
	NombreAdmin    string `json:"nombre_admin"`
}
```

---

- [ ] **Step 2.2: Escribir test fallido para `Crear`**

Crear `internal/tenant/service_test.go`:

```go
package tenant_test

import (
	"context"
	"errors"
	"testing"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/tenant"
)

type mockStore struct {
	crearFn func(ctx context.Context, input tenant.OnboardingInput) (*tenant.Tenant, error)
}

func (m *mockStore) Crear(ctx context.Context, input tenant.OnboardingInput) (*tenant.Tenant, error) {
	return m.crearFn(ctx, input)
}
func (m *mockStore) ObtenerPorID(ctx context.Context, id string) (*tenant.Tenant, error) {
	return nil, nil
}

func TestCrear_Exitoso(t *testing.T) {
	store := &mockStore{
		crearFn: func(ctx context.Context, input tenant.OnboardingInput) (*tenant.Tenant, error) {
			return &tenant.Tenant{ID: "t-1", Nombre: input.Nombre, Slug: input.Slug}, nil
		},
	}
	svc := tenant.NuevoService(store)
	result, err := svc.Crear(context.Background(), tenant.OnboardingInput{
		Nombre:      "Mi Bar",
		Slug:        "mi-bar",
		EmailAdmin:  "admin@mibar.com",
		NombreAdmin: "Carlos",
	})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if result.ID != "t-1" {
		t.Errorf("ID: got %q, want %q", result.ID, "t-1")
	}
}

func TestCrear_SlugVacio_Error(t *testing.T) {
	svc := tenant.NuevoService(&mockStore{
		crearFn: func(ctx context.Context, input tenant.OnboardingInput) (*tenant.Tenant, error) {
			return nil, errors.New("slug vacío")
		},
	})
	_, err := svc.Crear(context.Background(), tenant.OnboardingInput{
		Nombre:     "Sin Slug",
		EmailAdmin: "a@b.com",
	})
	if err == nil {
		t.Fatal("esperaba error por slug vacío")
	}
}
```

---

- [ ] **Step 2.3: Ejecutar test — debe fallar**

```bash
cd repos/api && go test ./internal/tenant/... -v
```

Expected: error de compilación — `tenant.NuevoService` no existe.

---

- [ ] **Step 2.4: Crear `internal/tenant/store.go`**

```go
package tenant

import (
	"context"
	"fmt"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/db"
)

type Store interface {
	Crear(ctx context.Context, input OnboardingInput) (*Tenant, error)
	ObtenerPorID(ctx context.Context, id string) (*Tenant, error)
}

type pgStore struct{}

func NuevoStore() Store { return &pgStore{} }

func (s *pgStore) Crear(ctx context.Context, input OnboardingInput) (*Tenant, error) {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var t Tenant
	err = tx.QueryRow(ctx,
		`INSERT INTO tenants (nombre, nombre_fantasia, rubro, slug)
		 VALUES ($1, $2, $3, $4) RETURNING id, nombre, nombre_fantasia, rubro, slug, created_at`,
		input.Nombre, input.NombreFantasia, input.Rubro, input.Slug,
	).Scan(&t.ID, &t.Nombre, &t.NombreFantasia, &t.Rubro, &t.Slug, &t.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("error creando tenant: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO usuarios (tenant_id, email, nombre, rol)
		 VALUES ($1, $2, $3, 'admin')`,
		t.ID, input.EmailAdmin, input.NombreAdmin,
	)
	if err != nil {
		return nil, fmt.Errorf("error creando usuario admin: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO sucursales (tenant_id, nombre)
		 VALUES ($1, 'Casa central')`,
		t.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("error creando sucursal default: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *pgStore) ObtenerPorID(ctx context.Context, id string) (*Tenant, error) {
	t := &Tenant{}
	err := db.Pool.QueryRow(ctx,
		`SELECT id, nombre, nombre_fantasia, rubro, slug, created_at
		 FROM tenants WHERE id = $1`, id,
	).Scan(&t.ID, &t.Nombre, &t.NombreFantasia, &t.Rubro, &t.Slug, &t.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("tenant no encontrado: %w", err)
	}
	return t, nil
}
```

---

- [ ] **Step 2.5: Crear `internal/tenant/service.go`**

```go
package tenant

import (
	"context"
	"errors"
)

type Service struct {
	store Store
}

func NuevoService(s Store) *Service { return &Service{store: s} }

func (svc *Service) Crear(ctx context.Context, input OnboardingInput) (*Tenant, error) {
	if input.Slug == "" {
		return nil, errors.New("slug requerido")
	}
	if input.EmailAdmin == "" {
		return nil, errors.New("email del admin requerido")
	}
	return svc.store.Crear(ctx, input)
}

func (svc *Service) ObtenerPorID(ctx context.Context, id string) (*Tenant, error) {
	return svc.store.ObtenerPorID(ctx, id)
}
```

---

- [ ] **Step 2.6: Ejecutar tests — deben pasar**

```bash
cd repos/api && go test ./internal/tenant/... -v
```

Expected: `PASS` para `TestCrear_Exitoso` y `TestCrear_SlugVacio_Error`.

---

- [ ] **Step 2.7: Crear `internal/tenant/handler.go`**

```go
package tenant

import (
	"encoding/json"
	"net/http"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
)

type Handlers struct {
	svc *Service
}

func NuevosHandlers(svc *Service) *Handlers { return &Handlers{svc: svc} }

func (h *Handlers) Crear(w http.ResponseWriter, r *http.Request) {
	var input OnboardingInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, "body inválido", http.StatusBadRequest)
		return
	}

	t, err := h.svc.Crear(r.Context(), input)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(t)
}

func (h *Handlers) ObtenerMe(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		jsonError(w, "no autorizado", http.StatusUnauthorized)
		return
	}

	t, err := h.svc.ObtenerPorID(r.Context(), claims.TenantID)
	if err != nil {
		jsonError(w, "tenant no encontrado", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)
}

func jsonError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
```

---

- [ ] **Step 2.8: Agregar rutas de tenant en `rutas.go`**

Reemplazar el bloque de comentarios del Sprint 4 — Tenants en `rutas.go`:

```go
// Al inicio del archivo, agregar imports:
import (
    "encoding/json"
    "net/http"
    "time"

    "github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
    "github.com/aguirrepablo-iresm/mesa-click/api/internal/db"
    "github.com/aguirrepablo-iresm/mesa-click/api/internal/tenant"
)

// En registrarRutas, después de /health:
tenantStore := tenant.NuevoStore()
tenantSvc := tenant.NuevoService(tenantStore)
tenantH := tenant.NuevosHandlers(tenantSvc)

authStore := auth.NuevoStore()
authSvc := auth.NuevoService(authStore)
authH := auth.NuevosHandlers(authSvc)

mux.HandleFunc("POST /auth/magic-link", authH.SolicitarLink)
mux.HandleFunc("GET /auth/verify", authH.VerificarToken)

mux.Handle("POST /tenants", http.HandlerFunc(tenantH.Crear))
mux.Handle("GET /tenants/me", auth.Requerir(http.HandlerFunc(tenantH.ObtenerMe)))
```

---

- [ ] **Step 2.9: Verificar que compila**

```bash
cd repos/api && go build ./...
```

Expected: sin errores.

---

- [ ] **Step 2.10: Commit**

```bash
git add internal/tenant/ rutas.go
git commit -m "feat(tenant): onboarding con tenant + usuario admin + sucursal default"
```

---

## Task 3: Carta — Categorías, Artículos y Carta Pública

**Files:**
- Create: `internal/carta/model.go`
- Create: `internal/carta/store.go`
- Create: `internal/carta/service.go`
- Create: `internal/carta/handler.go`
- Create: `internal/carta/service_test.go`
- Modify: `rutas.go`

**Interfaces:**
- Consumes: `auth.Requerir`, `auth.ClaimsFromContext` (para tenant_id)
- Produces:
  - `GET /carta/categorias` — lista categorías del tenant (protegido)
  - `POST /carta/categorias` — crea categoría (protegido)
  - `DELETE /carta/categorias/{id}` — elimina categoría (protegido)
  - `GET /carta/articulos` — lista artículos del tenant (protegido)
  - `POST /carta/articulos` — crea artículo (protegido)
  - `PATCH /carta/articulos/{id}` — actualiza activo/nombre/precio (protegido)
  - `DELETE /carta/articulos/{id}` — elimina artículo (protegido)
  - `GET /publica/{sucursal_id}/carta` — carta pública sin auth (para QR)

---

- [ ] **Step 3.1: Crear `internal/carta/model.go`**

```go
package carta

type Categoria struct {
	ID       string `json:"id"`
	TenantID string `json:"tenant_id,omitempty"`
	Nombre   string `json:"nombre"`
	Orden    int    `json:"orden"`
}

type Articulo struct {
	ID          string  `json:"id"`
	TenantID    string  `json:"tenant_id,omitempty"`
	CategoriaID string  `json:"categoria_id"`
	Nombre      string  `json:"nombre"`
	Descripcion string  `json:"descripcion,omitempty"`
	Precio      float64 `json:"precio"`
	FotoURL     string  `json:"foto_url,omitempty"`
	Activo      bool    `json:"activo"`
}

type CategoriaInput struct {
	Nombre string `json:"nombre"`
	Orden  int    `json:"orden"`
}

type ArticuloInput struct {
	CategoriaID string  `json:"categoria_id"`
	Nombre      string  `json:"nombre"`
	Descripcion string  `json:"descripcion"`
	Precio      float64 `json:"precio"`
	FotoURL     string  `json:"foto_url"`
}

type ArticuloUpdate struct {
	Nombre  *string  `json:"nombre"`
	Precio  *float64 `json:"precio"`
	Activo  *bool    `json:"activo"`
}

type CartaPublica struct {
	Categorias []CategoriaConArticulos `json:"categorias"`
}

type CategoriaConArticulos struct {
	Categoria
	Articulos []Articulo `json:"articulos"`
}
```

---

- [ ] **Step 3.2: Escribir tests fallidos**

Crear `internal/carta/service_test.go`:

```go
package carta_test

import (
	"context"
	"testing"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/carta"
)

type mockStore struct {
	categorias []carta.Categoria
	articulos  []carta.Articulo
}

func (m *mockStore) ListarCategorias(ctx context.Context, tenantID string) ([]carta.Categoria, error) {
	var result []carta.Categoria
	for _, c := range m.categorias {
		if c.TenantID == tenantID {
			result = append(result, c)
		}
	}
	return result, nil
}
func (m *mockStore) CrearCategoria(ctx context.Context, tenantID string, input carta.CategoriaInput) (*carta.Categoria, error) {
	c := carta.Categoria{ID: "cat-1", TenantID: tenantID, Nombre: input.Nombre, Orden: input.Orden}
	m.categorias = append(m.categorias, c)
	return &c, nil
}
func (m *mockStore) EliminarCategoria(ctx context.Context, id, tenantID string) error { return nil }
func (m *mockStore) ListarArticulos(ctx context.Context, tenantID string) ([]carta.Articulo, error) {
	return m.articulos, nil
}
func (m *mockStore) CrearArticulo(ctx context.Context, tenantID string, input carta.ArticuloInput) (*carta.Articulo, error) {
	a := carta.Articulo{ID: "art-1", TenantID: tenantID, CategoriaID: input.CategoriaID, Nombre: input.Nombre, Precio: input.Precio, Activo: true}
	return &a, nil
}
func (m *mockStore) ActualizarArticulo(ctx context.Context, id, tenantID string, u carta.ArticuloUpdate) (*carta.Articulo, error) {
	return &carta.Articulo{ID: id}, nil
}
func (m *mockStore) EliminarArticulo(ctx context.Context, id, tenantID string) error { return nil }
func (m *mockStore) ObtenerCartaPublica(ctx context.Context, sucursalID string) (*carta.CartaPublica, error) {
	return &carta.CartaPublica{}, nil
}

func TestListarCategorias(t *testing.T) {
	store := &mockStore{
		categorias: []carta.Categoria{
			{ID: "1", TenantID: "t-1", Nombre: "Bebidas"},
			{ID: "2", TenantID: "t-2", Nombre: "Otras"},
		},
	}
	svc := carta.NuevoService(store)
	cats, err := svc.ListarCategorias(context.Background(), "t-1")
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if len(cats) != 1 {
		t.Errorf("got %d categorías, want 1", len(cats))
	}
	if cats[0].Nombre != "Bebidas" {
		t.Errorf("got %q, want %q", cats[0].Nombre, "Bebidas")
	}
}

func TestCrearArticulo_PrecioNegativo(t *testing.T) {
	svc := carta.NuevoService(&mockStore{})
	_, err := svc.CrearArticulo(context.Background(), "t-1", carta.ArticuloInput{
		Nombre: "Test",
		Precio: -5,
	})
	if err == nil {
		t.Fatal("esperaba error por precio negativo")
	}
}
```

---

- [ ] **Step 3.3: Ejecutar tests — deben fallar**

```bash
cd repos/api && go test ./internal/carta/... -v
```

Expected: error de compilación — `carta.NuevoService` no existe.

---

- [ ] **Step 3.4: Crear `internal/carta/store.go`**

```go
package carta

import (
	"context"
	"fmt"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/db"
)

type Store interface {
	ListarCategorias(ctx context.Context, tenantID string) ([]Categoria, error)
	CrearCategoria(ctx context.Context, tenantID string, input CategoriaInput) (*Categoria, error)
	EliminarCategoria(ctx context.Context, id, tenantID string) error
	ListarArticulos(ctx context.Context, tenantID string) ([]Articulo, error)
	CrearArticulo(ctx context.Context, tenantID string, input ArticuloInput) (*Articulo, error)
	ActualizarArticulo(ctx context.Context, id, tenantID string, u ArticuloUpdate) (*Articulo, error)
	EliminarArticulo(ctx context.Context, id, tenantID string) error
	ObtenerCartaPublica(ctx context.Context, sucursalID string) (*CartaPublica, error)
}

type pgStore struct{}

func NuevoStore() Store { return &pgStore{} }

func (s *pgStore) ListarCategorias(ctx context.Context, tenantID string) ([]Categoria, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT id, tenant_id, nombre, orden FROM categorias WHERE tenant_id = $1 ORDER BY orden`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cats []Categoria
	for rows.Next() {
		var c Categoria
		if err := rows.Scan(&c.ID, &c.TenantID, &c.Nombre, &c.Orden); err != nil {
			return nil, err
		}
		cats = append(cats, c)
	}
	return cats, nil
}

func (s *pgStore) CrearCategoria(ctx context.Context, tenantID string, input CategoriaInput) (*Categoria, error) {
	c := &Categoria{}
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO categorias (tenant_id, nombre, orden) VALUES ($1, $2, $3)
		 RETURNING id, tenant_id, nombre, orden`,
		tenantID, input.Nombre, input.Orden,
	).Scan(&c.ID, &c.TenantID, &c.Nombre, &c.Orden)
	return c, err
}

func (s *pgStore) EliminarCategoria(ctx context.Context, id, tenantID string) error {
	tag, err := db.Pool.Exec(ctx,
		`DELETE FROM categorias WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("categoría no encontrada")
	}
	return nil
}

func (s *pgStore) ListarArticulos(ctx context.Context, tenantID string) ([]Articulo, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT id, tenant_id, categoria_id, nombre, COALESCE(descripcion,''), precio, COALESCE(foto_url,''), activo
		 FROM articulos WHERE tenant_id = $1 ORDER BY nombre`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var arts []Articulo
	for rows.Next() {
		var a Articulo
		if err := rows.Scan(&a.ID, &a.TenantID, &a.CategoriaID, &a.Nombre, &a.Descripcion, &a.Precio, &a.FotoURL, &a.Activo); err != nil {
			return nil, err
		}
		arts = append(arts, a)
	}
	return arts, nil
}

func (s *pgStore) CrearArticulo(ctx context.Context, tenantID string, input ArticuloInput) (*Articulo, error) {
	a := &Articulo{}
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO articulos (tenant_id, categoria_id, nombre, descripcion, precio, foto_url, activo)
		 VALUES ($1, $2, $3, $4, $5, $6, true)
		 RETURNING id, tenant_id, categoria_id, nombre, COALESCE(descripcion,''), precio, COALESCE(foto_url,''), activo`,
		tenantID, input.CategoriaID, input.Nombre, input.Descripcion, input.Precio, input.FotoURL,
	).Scan(&a.ID, &a.TenantID, &a.CategoriaID, &a.Nombre, &a.Descripcion, &a.Precio, &a.FotoURL, &a.Activo)
	return a, err
}

func (s *pgStore) ActualizarArticulo(ctx context.Context, id, tenantID string, u ArticuloUpdate) (*Articulo, error) {
	a := &Articulo{}
	err := db.Pool.QueryRow(ctx,
		`UPDATE articulos SET
		   nombre   = COALESCE($3, nombre),
		   precio   = COALESCE($4, precio),
		   activo   = COALESCE($5, activo)
		 WHERE id = $1 AND tenant_id = $2
		 RETURNING id, tenant_id, categoria_id, nombre, COALESCE(descripcion,''), precio, COALESCE(foto_url,''), activo`,
		id, tenantID, u.Nombre, u.Precio, u.Activo,
	).Scan(&a.ID, &a.TenantID, &a.CategoriaID, &a.Nombre, &a.Descripcion, &a.Precio, &a.FotoURL, &a.Activo)
	if err != nil {
		return nil, fmt.Errorf("artículo no encontrado: %w", err)
	}
	return a, nil
}

func (s *pgStore) EliminarArticulo(ctx context.Context, id, tenantID string) error {
	tag, err := db.Pool.Exec(ctx,
		`DELETE FROM articulos WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("artículo no encontrado")
	}
	return nil
}

func (s *pgStore) ObtenerCartaPublica(ctx context.Context, sucursalID string) (*CartaPublica, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT c.id, c.nombre, c.orden,
		        a.id, a.categoria_id, a.nombre, COALESCE(a.descripcion,''), a.precio, COALESCE(a.foto_url,'')
		 FROM categorias c
		 JOIN articulos a ON a.categoria_id = c.id
		 WHERE a.tenant_id = (SELECT tenant_id FROM sucursales WHERE id = $1)
		   AND a.activo = true
		 ORDER BY c.orden, a.nombre`,
		sucursalID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	catMap := map[string]*CategoriaConArticulos{}
	var orden []string

	for rows.Next() {
		var (
			catID, catNombre string
			catOrden         int
			art              Articulo
		)
		if err := rows.Scan(&catID, &catNombre, &catOrden,
			&art.ID, &art.CategoriaID, &art.Nombre, &art.Descripcion, &art.Precio, &art.FotoURL); err != nil {
			return nil, err
		}
		art.Activo = true
		if _, ok := catMap[catID]; !ok {
			catMap[catID] = &CategoriaConArticulos{
				Categoria: Categoria{ID: catID, Nombre: catNombre, Orden: catOrden},
			}
			orden = append(orden, catID)
		}
		catMap[catID].Articulos = append(catMap[catID].Articulos, art)
	}

	carta := &CartaPublica{}
	for _, id := range orden {
		carta.Categorias = append(carta.Categorias, *catMap[id])
	}
	return carta, nil
}
```

---

- [ ] **Step 3.5: Crear `internal/carta/service.go`**

```go
package carta

import (
	"context"
	"errors"
)

type Service struct {
	store Store
}

func NuevoService(s Store) *Service { return &Service{store: s} }

func (svc *Service) ListarCategorias(ctx context.Context, tenantID string) ([]Categoria, error) {
	return svc.store.ListarCategorias(ctx, tenantID)
}

func (svc *Service) CrearCategoria(ctx context.Context, tenantID string, input CategoriaInput) (*Categoria, error) {
	if input.Nombre == "" {
		return nil, errors.New("nombre requerido")
	}
	return svc.store.CrearCategoria(ctx, tenantID, input)
}

func (svc *Service) EliminarCategoria(ctx context.Context, id, tenantID string) error {
	return svc.store.EliminarCategoria(ctx, id, tenantID)
}

func (svc *Service) ListarArticulos(ctx context.Context, tenantID string) ([]Articulo, error) {
	return svc.store.ListarArticulos(ctx, tenantID)
}

func (svc *Service) CrearArticulo(ctx context.Context, tenantID string, input ArticuloInput) (*Articulo, error) {
	if input.Nombre == "" {
		return nil, errors.New("nombre requerido")
	}
	if input.Precio < 0 {
		return nil, errors.New("precio no puede ser negativo")
	}
	if input.CategoriaID == "" {
		return nil, errors.New("categoria_id requerido")
	}
	return svc.store.CrearArticulo(ctx, tenantID, input)
}

func (svc *Service) ActualizarArticulo(ctx context.Context, id, tenantID string, u ArticuloUpdate) (*Articulo, error) {
	if u.Precio != nil && *u.Precio < 0 {
		return nil, errors.New("precio no puede ser negativo")
	}
	return svc.store.ActualizarArticulo(ctx, id, tenantID, u)
}

func (svc *Service) EliminarArticulo(ctx context.Context, id, tenantID string) error {
	return svc.store.EliminarArticulo(ctx, id, tenantID)
}

func (svc *Service) ObtenerCartaPublica(ctx context.Context, sucursalID string) (*CartaPublica, error) {
	return svc.store.ObtenerCartaPublica(ctx, sucursalID)
}
```

---

- [ ] **Step 3.6: Ejecutar tests — deben pasar**

```bash
cd repos/api && go test ./internal/carta/... -v
```

Expected: `PASS` para `TestListarCategorias` y `TestCrearArticulo_PrecioNegativo`.

---

- [ ] **Step 3.7: Crear `internal/carta/handler.go`**

```go
package carta

import (
	"encoding/json"
	"net/http"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
)

type Handlers struct {
	svc *Service
}

func NuevosHandlers(svc *Service) *Handlers { return &Handlers{svc: svc} }

func (h *Handlers) ListarCategorias(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	cats, err := h.svc.ListarCategorias(r.Context(), claims.TenantID)
	if err != nil {
		jsonError(w, "error listando categorías", http.StatusInternalServerError)
		return
	}
	jsonOK(w, cats)
}

func (h *Handlers) CrearCategoria(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	var input CategoriaInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, "body inválido", http.StatusBadRequest)
		return
	}
	cat, err := h.svc.CrearCategoria(r.Context(), claims.TenantID, input)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(cat)
}

func (h *Handlers) EliminarCategoria(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	id := r.PathValue("id")
	if err := h.svc.EliminarCategoria(r.Context(), id, claims.TenantID); err != nil {
		jsonError(w, "categoría no encontrada", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) ListarArticulos(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	arts, err := h.svc.ListarArticulos(r.Context(), claims.TenantID)
	if err != nil {
		jsonError(w, "error listando artículos", http.StatusInternalServerError)
		return
	}
	jsonOK(w, arts)
}

func (h *Handlers) CrearArticulo(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	var input ArticuloInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, "body inválido", http.StatusBadRequest)
		return
	}
	art, err := h.svc.CrearArticulo(r.Context(), claims.TenantID, input)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(art)
}

func (h *Handlers) ActualizarArticulo(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	id := r.PathValue("id")
	var u ArticuloUpdate
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		jsonError(w, "body inválido", http.StatusBadRequest)
		return
	}
	art, err := h.svc.ActualizarArticulo(r.Context(), id, claims.TenantID, u)
	if err != nil {
		jsonError(w, "artículo no encontrado", http.StatusNotFound)
		return
	}
	jsonOK(w, art)
}

func (h *Handlers) EliminarArticulo(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	id := r.PathValue("id")
	if err := h.svc.EliminarArticulo(r.Context(), id, claims.TenantID); err != nil {
		jsonError(w, "artículo no encontrado", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) CartaPublica(w http.ResponseWriter, r *http.Request) {
	sucursalID := r.PathValue("sucursal_id")
	carta, err := h.svc.ObtenerCartaPublica(r.Context(), sucursalID)
	if err != nil {
		jsonError(w, "carta no disponible", http.StatusNotFound)
		return
	}
	jsonOK(w, carta)
}

func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
```

---

- [ ] **Step 3.8: Registrar rutas de carta en `rutas.go`**

Agregar después de las rutas de tenant:

```go
import "github.com/aguirrepablo-iresm/mesa-click/api/internal/carta"

// En registrarRutas:
cartaStore := carta.NuevoStore()
cartaSvc := carta.NuevoService(cartaStore)
cartaH := carta.NuevosHandlers(cartaSvc)

mux.Handle("GET /carta/categorias",      auth.Requerir(http.HandlerFunc(cartaH.ListarCategorias)))
mux.Handle("POST /carta/categorias",     auth.Requerir(http.HandlerFunc(cartaH.CrearCategoria)))
mux.Handle("DELETE /carta/categorias/{id}", auth.Requerir(http.HandlerFunc(cartaH.EliminarCategoria)))
mux.Handle("GET /carta/articulos",       auth.Requerir(http.HandlerFunc(cartaH.ListarArticulos)))
mux.Handle("POST /carta/articulos",      auth.Requerir(http.HandlerFunc(cartaH.CrearArticulo)))
mux.Handle("PATCH /carta/articulos/{id}", auth.Requerir(http.HandlerFunc(cartaH.ActualizarArticulo)))
mux.Handle("DELETE /carta/articulos/{id}", auth.Requerir(http.HandlerFunc(cartaH.EliminarArticulo)))
mux.HandleFunc("GET /publica/{sucursal_id}/carta", cartaH.CartaPublica)
```

---

- [ ] **Step 3.9: Verificar que compila**

```bash
cd repos/api && go build ./...
```

---

- [ ] **Step 3.10: Commit**

```bash
git add internal/carta/ rutas.go
git commit -m "feat(carta): CRUD categorías + artículos + carta pública"
```

---

## Task 4: Mesa — CRUD + QR Token

**Files:**
- Create: `internal/mesa/model.go`
- Create: `internal/mesa/store.go`
- Create: `internal/mesa/service.go`
- Create: `internal/mesa/handler.go`
- Create: `internal/mesa/service_test.go`
- Modify: `rutas.go`

**Interfaces:**
- Consumes: `auth.Requerir`, `auth.ClaimsFromContext`
- Produces:
  - `GET /mesas` — lista mesas del tenant (protegido)
  - `POST /mesas` — crea mesa (protegido)
  - `PATCH /mesas/{id}` — actualiza número/capacidad/estado (protegido)
  - `DELETE /mesas/{id}` — elimina mesa (protegido)
  - `GET /publica/mesa/{qr_token}` — info de mesa por QR (sin auth, para el cliente)

---

- [ ] **Step 4.1: Crear `internal/mesa/model.go`**

```go
package mesa

type Mesa struct {
	ID          string `json:"id"`
	SucursalID  string `json:"sucursal_id"`
	SectorID    string `json:"sector_id,omitempty"`
	Numero      int    `json:"numero"`
	Capacidad   int    `json:"capacidad"`
	QRToken     string `json:"qr_token"`
	Estado      string `json:"estado"` // "activa" | "inactiva"
}

type MesaInput struct {
	SucursalID string `json:"sucursal_id"`
	Numero     int    `json:"numero"`
	Capacidad  int    `json:"capacidad"`
}

type MesaUpdate struct {
	Numero    *int    `json:"numero"`
	Capacidad *int    `json:"capacidad"`
	Estado    *string `json:"estado"`
}

type MesaPublica struct {
	ID         string `json:"id"`
	Numero     int    `json:"numero"`
	SucursalID string `json:"sucursal_id"`
	TenantID   string `json:"tenant_id"`
}
```

---

- [ ] **Step 4.2: Escribir tests fallidos**

Crear `internal/mesa/service_test.go`:

```go
package mesa_test

import (
	"context"
	"testing"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/mesa"
)

type mockStore struct {
	mesas []mesa.Mesa
}

func (m *mockStore) Listar(ctx context.Context, tenantID string) ([]mesa.Mesa, error) {
	return m.mesas, nil
}
func (m *mockStore) Crear(ctx context.Context, input mesa.MesaInput, qrToken string) (*mesa.Mesa, error) {
	return &mesa.Mesa{ID: "m-1", Numero: input.Numero, QRToken: qrToken, Estado: "activa"}, nil
}
func (m *mockStore) Actualizar(ctx context.Context, id, tenantID string, u mesa.MesaUpdate) (*mesa.Mesa, error) {
	return &mesa.Mesa{ID: id}, nil
}
func (m *mockStore) Eliminar(ctx context.Context, id, tenantID string) error { return nil }
func (m *mockStore) ObtenerPorQRToken(ctx context.Context, token string) (*mesa.MesaPublica, error) {
	return &mesa.MesaPublica{ID: "m-1"}, nil
}

func TestCrear_QRTokenGenerado(t *testing.T) {
	svc := mesa.NuevoService(&mockStore{})
	m, err := svc.Crear(context.Background(), "tenant-1", mesa.MesaInput{
		SucursalID: "suc-1",
		Numero:     5,
		Capacidad:  4,
	})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if m.QRToken == "" {
		t.Error("QRToken vacío — debería generarse automáticamente")
	}
}

func TestCrear_NumeroInvalido(t *testing.T) {
	svc := mesa.NuevoService(&mockStore{})
	_, err := svc.Crear(context.Background(), "tenant-1", mesa.MesaInput{
		SucursalID: "suc-1",
		Numero:     0,
	})
	if err == nil {
		t.Fatal("esperaba error por número de mesa inválido")
	}
}
```

---

- [ ] **Step 4.3: Ejecutar tests — deben fallar**

```bash
cd repos/api && go test ./internal/mesa/... -v
```

Expected: error de compilación.

---

- [ ] **Step 4.4: Crear `internal/mesa/store.go`**

```go
package mesa

import (
	"context"
	"fmt"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/db"
)

type Store interface {
	Listar(ctx context.Context, tenantID string) ([]Mesa, error)
	Crear(ctx context.Context, input MesaInput, qrToken string) (*Mesa, error)
	Actualizar(ctx context.Context, id, tenantID string, u MesaUpdate) (*Mesa, error)
	Eliminar(ctx context.Context, id, tenantID string) error
	ObtenerPorQRToken(ctx context.Context, token string) (*MesaPublica, error)
}

type pgStore struct{}

func NuevoStore() Store { return &pgStore{} }

func (s *pgStore) Listar(ctx context.Context, tenantID string) ([]Mesa, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT m.id, m.sucursal_id, COALESCE(m.sector_id::text,''), m.numero, m.capacidad, m.qr_token, m.estado
		 FROM mesas m
		 JOIN sucursales su ON su.id = m.sucursal_id
		 WHERE su.tenant_id = $1
		 ORDER BY m.numero`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var mesas []Mesa
	for rows.Next() {
		var m Mesa
		if err := rows.Scan(&m.ID, &m.SucursalID, &m.SectorID, &m.Numero, &m.Capacidad, &m.QRToken, &m.Estado); err != nil {
			return nil, err
		}
		mesas = append(mesas, m)
	}
	return mesas, nil
}

func (s *pgStore) Crear(ctx context.Context, input MesaInput, qrToken string) (*Mesa, error) {
	m := &Mesa{}
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO mesas (sucursal_id, numero, capacidad, qr_token, estado)
		 VALUES ($1, $2, $3, $4, 'activa')
		 RETURNING id, sucursal_id, COALESCE(sector_id::text,''), numero, capacidad, qr_token, estado`,
		input.SucursalID, input.Numero, input.Capacidad, qrToken,
	).Scan(&m.ID, &m.SucursalID, &m.SectorID, &m.Numero, &m.Capacidad, &m.QRToken, &m.Estado)
	return m, err
}

func (s *pgStore) Actualizar(ctx context.Context, id, tenantID string, u MesaUpdate) (*Mesa, error) {
	m := &Mesa{}
	err := db.Pool.QueryRow(ctx,
		`UPDATE mesas SET
		   numero    = COALESCE($3, numero),
		   capacidad = COALESCE($4, capacidad),
		   estado    = COALESCE($5, estado)
		 WHERE id = $1
		   AND sucursal_id IN (SELECT id FROM sucursales WHERE tenant_id = $2)
		 RETURNING id, sucursal_id, COALESCE(sector_id::text,''), numero, capacidad, qr_token, estado`,
		id, tenantID, u.Numero, u.Capacidad, u.Estado,
	).Scan(&m.ID, &m.SucursalID, &m.SectorID, &m.Numero, &m.Capacidad, &m.QRToken, &m.Estado)
	if err != nil {
		return nil, fmt.Errorf("mesa no encontrada: %w", err)
	}
	return m, nil
}

func (s *pgStore) Eliminar(ctx context.Context, id, tenantID string) error {
	tag, err := db.Pool.Exec(ctx,
		`DELETE FROM mesas WHERE id = $1
		 AND sucursal_id IN (SELECT id FROM sucursales WHERE tenant_id = $2)`, id, tenantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("mesa no encontrada")
	}
	return nil
}

func (s *pgStore) ObtenerPorQRToken(ctx context.Context, token string) (*MesaPublica, error) {
	mp := &MesaPublica{}
	err := db.Pool.QueryRow(ctx,
		`SELECT m.id, m.numero, m.sucursal_id, su.tenant_id
		 FROM mesas m
		 JOIN sucursales su ON su.id = m.sucursal_id
		 WHERE m.qr_token = $1 AND m.estado = 'activa'`, token,
	).Scan(&mp.ID, &mp.Numero, &mp.SucursalID, &mp.TenantID)
	if err != nil {
		return nil, fmt.Errorf("mesa no encontrada: %w", err)
	}
	return mp, nil
}
```

---

- [ ] **Step 4.5: Crear `internal/mesa/service.go`**

```go
package mesa

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
)

type Service struct {
	store Store
}

func NuevoService(s Store) *Service { return &Service{store: s} }

func (svc *Service) Listar(ctx context.Context, tenantID string) ([]Mesa, error) {
	return svc.store.Listar(ctx, tenantID)
}

func (svc *Service) Crear(ctx context.Context, tenantID string, input MesaInput) (*Mesa, error) {
	if input.Numero <= 0 {
		return nil, errors.New("número de mesa debe ser mayor a 0")
	}
	if input.SucursalID == "" {
		return nil, errors.New("sucursal_id requerido")
	}
	qrToken, err := generarQRToken()
	if err != nil {
		return nil, fmt.Errorf("error generando QR token: %w", err)
	}
	return svc.store.Crear(ctx, input, qrToken)
}

func (svc *Service) Actualizar(ctx context.Context, id, tenantID string, u MesaUpdate) (*Mesa, error) {
	if u.Estado != nil && *u.Estado != "activa" && *u.Estado != "inactiva" {
		return nil, errors.New("estado inválido: debe ser 'activa' o 'inactiva'")
	}
	return svc.store.Actualizar(ctx, id, tenantID, u)
}

func (svc *Service) Eliminar(ctx context.Context, id, tenantID string) error {
	return svc.store.Eliminar(ctx, id, tenantID)
}

func (svc *Service) ObtenerPorQRToken(ctx context.Context, token string) (*MesaPublica, error) {
	return svc.store.ObtenerPorQRToken(ctx, token)
}

func generarQRToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
```

---

- [ ] **Step 4.6: Ejecutar tests — deben pasar**

```bash
cd repos/api && go test ./internal/mesa/... -v
```

Expected: `PASS` para `TestCrear_QRTokenGenerado` y `TestCrear_NumeroInvalido`.

---

- [ ] **Step 4.7: Crear `internal/mesa/handler.go`**

```go
package mesa

import (
	"encoding/json"
	"net/http"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
)

type Handlers struct {
	svc *Service
}

func NuevosHandlers(svc *Service) *Handlers { return &Handlers{svc: svc} }

func (h *Handlers) Listar(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	mesas, err := h.svc.Listar(r.Context(), claims.TenantID)
	if err != nil {
		jsonError(w, "error listando mesas", http.StatusInternalServerError)
		return
	}
	jsonOK(w, mesas)
}

func (h *Handlers) Crear(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	var input MesaInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, "body inválido", http.StatusBadRequest)
		return
	}
	m, err := h.svc.Crear(r.Context(), claims.TenantID, input)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(m)
}

func (h *Handlers) Actualizar(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	id := r.PathValue("id")
	var u MesaUpdate
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		jsonError(w, "body inválido", http.StatusBadRequest)
		return
	}
	m, err := h.svc.Actualizar(r.Context(), id, claims.TenantID, u)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	jsonOK(w, m)
}

func (h *Handlers) Eliminar(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	id := r.PathValue("id")
	if err := h.svc.Eliminar(r.Context(), id, claims.TenantID); err != nil {
		jsonError(w, "mesa no encontrada", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) MesaPorQR(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("qr_token")
	mp, err := h.svc.ObtenerPorQRToken(r.Context(), token)
	if err != nil {
		jsonError(w, "mesa no encontrada", http.StatusNotFound)
		return
	}
	jsonOK(w, mp)
}

func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
```

---

- [ ] **Step 4.8: Registrar rutas de mesa en `rutas.go`**

Agregar después de las rutas de carta:

```go
import "github.com/aguirrepablo-iresm/mesa-click/api/internal/mesa"

// En registrarRutas:
mesaStore := mesa.NuevoStore()
mesaSvc := mesa.NuevoService(mesaStore)
mesaH := mesa.NuevosHandlers(mesaSvc)

mux.Handle("GET /mesas",        auth.Requerir(http.HandlerFunc(mesaH.Listar)))
mux.Handle("POST /mesas",       auth.Requerir(http.HandlerFunc(mesaH.Crear)))
mux.Handle("PATCH /mesas/{id}", auth.Requerir(http.HandlerFunc(mesaH.Actualizar)))
mux.Handle("DELETE /mesas/{id}", auth.Requerir(http.HandlerFunc(mesaH.Eliminar)))
mux.HandleFunc("GET /publica/mesa/{qr_token}", mesaH.MesaPorQR)
```

---

- [ ] **Step 4.9: Verificar que compila**

```bash
cd repos/api && go build ./...
```

---

- [ ] **Step 4.10: Commit**

```bash
git add internal/mesa/ rutas.go
git commit -m "feat(mesa): CRUD mesas + QR token generado automáticamente"
```

---

## Task 5: Pedido — Crear, Listar y Cambiar Estado

**Files:**
- Create: `internal/pedido/model.go`
- Create: `internal/pedido/store.go`
- Create: `internal/pedido/service.go`
- Create: `internal/pedido/handler.go`
- Create: `internal/pedido/service_test.go`
- Modify: `rutas.go`

**Interfaces:**
- Consumes: `auth.Requerir`, `auth.ClaimsFromContext` (para listar y cambiar estado)
- Produces:
  - `POST /pedidos` — crea pedido (sin auth — lo llama el cliente desde su QR)
  - `GET /pedidos` — lista pedidos activos de la sucursal (protegido)
  - `PATCH /pedidos/{id}/estado` — avanza estado (protegido)

---

- [ ] **Step 5.1: Crear `internal/pedido/model.go`**

```go
package pedido

var EstadosValidos = []string{"recibido", "preparando", "listo", "cerrado"}

type Pedido struct {
	ID         string       `json:"id"`
	MesaID     string       `json:"mesa_id"`
	SucursalID string       `json:"sucursal_id"`
	Estado     string       `json:"estado"`
	Items      []PedidoItem `json:"items,omitempty"`
	CreatedAt  string       `json:"created_at"`
	UpdatedAt  string       `json:"updated_at"`
}

type PedidoItem struct {
	ID           string  `json:"id"`
	PedidoID     string  `json:"pedido_id"`
	ArticuloID   string  `json:"articulo_id"`
	NombreArticulo string `json:"nombre_articulo,omitempty"`
	Cantidad     int     `json:"cantidad"`
	PrecioUnitario float64 `json:"precio_unitario"`
	Notas        string  `json:"notas,omitempty"`
}

type NuevoPedidoInput struct {
	MesaID string          `json:"mesa_id"`
	Items  []NuevoItemInput `json:"items"`
}

type NuevoItemInput struct {
	ArticuloID string  `json:"articulo_id"`
	Cantidad   int     `json:"cantidad"`
	Notas      string  `json:"notas"`
}

type CambiarEstadoInput struct {
	Estado string `json:"estado"`
}
```

---

- [ ] **Step 5.2: Escribir tests fallidos**

Crear `internal/pedido/service_test.go`:

```go
package pedido_test

import (
	"context"
	"testing"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/pedido"
)

type mockStore struct{}

func (m *mockStore) Crear(ctx context.Context, input pedido.NuevoPedidoInput, sucursalID string) (*pedido.Pedido, error) {
	return &pedido.Pedido{ID: "p-1", MesaID: input.MesaID, Estado: "recibido"}, nil
}
func (m *mockStore) ListarActivos(ctx context.Context, sucursalID string) ([]pedido.Pedido, error) {
	return []pedido.Pedido{}, nil
}
func (m *mockStore) CambiarEstado(ctx context.Context, id, tenantID, nuevoEstado string) (*pedido.Pedido, error) {
	return &pedido.Pedido{ID: id, Estado: nuevoEstado}, nil
}
func (m *mockStore) ObtenerSucursalPorMesa(ctx context.Context, mesaID string) (string, error) {
	return "suc-1", nil
}

func TestCambiarEstado_EstadoInvalido(t *testing.T) {
	svc := pedido.NuevoService(&mockStore{})
	_, err := svc.CambiarEstado(context.Background(), "p-1", "tenant-1", "invalido")
	if err == nil {
		t.Fatal("esperaba error por estado inválido")
	}
}

func TestCambiarEstado_EstadosValidos(t *testing.T) {
	svc := pedido.NuevoService(&mockStore{})
	for _, estado := range pedido.EstadosValidos {
		_, err := svc.CambiarEstado(context.Background(), "p-1", "tenant-1", estado)
		if err != nil {
			t.Errorf("estado %q debería ser válido pero dio error: %v", estado, err)
		}
	}
}

func TestCrear_SinItems_Error(t *testing.T) {
	svc := pedido.NuevoService(&mockStore{})
	_, err := svc.Crear(context.Background(), pedido.NuevoPedidoInput{
		MesaID: "mesa-1",
		Items:  []pedido.NuevoItemInput{},
	})
	if err == nil {
		t.Fatal("esperaba error por pedido sin items")
	}
}
```

---

- [ ] **Step 5.3: Ejecutar tests — deben fallar**

```bash
cd repos/api && go test ./internal/pedido/... -v
```

Expected: error de compilación.

---

- [ ] **Step 5.4: Crear `internal/pedido/store.go`**

```go
package pedido

import (
	"context"
	"fmt"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/db"
)

type Store interface {
	Crear(ctx context.Context, input NuevoPedidoInput, sucursalID string) (*Pedido, error)
	ListarActivos(ctx context.Context, sucursalID string) ([]Pedido, error)
	CambiarEstado(ctx context.Context, id, tenantID, nuevoEstado string) (*Pedido, error)
	ObtenerSucursalPorMesa(ctx context.Context, mesaID string) (string, error)
}

type pgStore struct{}

func NuevoStore() Store { return &pgStore{} }

func (s *pgStore) ObtenerSucursalPorMesa(ctx context.Context, mesaID string) (string, error) {
	var sucursalID string
	err := db.Pool.QueryRow(ctx,
		`SELECT sucursal_id FROM mesas WHERE id = $1`, mesaID,
	).Scan(&sucursalID)
	if err != nil {
		return "", fmt.Errorf("mesa no encontrada: %w", err)
	}
	return sucursalID, nil
}

func (s *pgStore) Crear(ctx context.Context, input NuevoPedidoInput, sucursalID string) (*Pedido, error) {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	p := &Pedido{}
	err = tx.QueryRow(ctx,
		`INSERT INTO pedidos (mesa_id, sucursal_id, estado)
		 VALUES ($1, $2, 'recibido')
		 RETURNING id, mesa_id, sucursal_id, estado, created_at, updated_at`,
		input.MesaID, sucursalID,
	).Scan(&p.ID, &p.MesaID, &p.SucursalID, &p.Estado, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("error creando pedido: %w", err)
	}

	for _, item := range input.Items {
		var precioUnitario float64
		err := tx.QueryRow(ctx,
			`SELECT precio FROM articulos WHERE id = $1`, item.ArticuloID,
		).Scan(&precioUnitario)
		if err != nil {
			return nil, fmt.Errorf("artículo %s no encontrado: %w", item.ArticuloID, err)
		}

		var itemID string
		err = tx.QueryRow(ctx,
			`INSERT INTO pedido_items (pedido_id, articulo_id, cantidad, precio_unitario, notas)
			 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
			p.ID, item.ArticuloID, item.Cantidad, precioUnitario, item.Notas,
		).Scan(&itemID)
		if err != nil {
			return nil, fmt.Errorf("error insertando item: %w", err)
		}

		p.Items = append(p.Items, PedidoItem{
			ID:             itemID,
			PedidoID:       p.ID,
			ArticuloID:     item.ArticuloID,
			Cantidad:       item.Cantidad,
			PrecioUnitario: precioUnitario,
			Notas:          item.Notas,
		})
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *pgStore) ListarActivos(ctx context.Context, sucursalID string) ([]Pedido, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT id, mesa_id, sucursal_id, estado, created_at, updated_at
		 FROM pedidos
		 WHERE sucursal_id = $1 AND estado != 'cerrado'
		 ORDER BY created_at`, sucursalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var pedidos []Pedido
	for rows.Next() {
		var p Pedido
		if err := rows.Scan(&p.ID, &p.MesaID, &p.SucursalID, &p.Estado, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		pedidos = append(pedidos, p)
	}
	return pedidos, nil
}

func (s *pgStore) CambiarEstado(ctx context.Context, id, tenantID, nuevoEstado string) (*Pedido, error) {
	p := &Pedido{}
	err := db.Pool.QueryRow(ctx,
		`UPDATE pedidos SET estado = $1, updated_at = now()
		 WHERE id = $2
		   AND sucursal_id IN (SELECT id FROM sucursales WHERE tenant_id = $3)
		 RETURNING id, mesa_id, sucursal_id, estado, created_at, updated_at`,
		nuevoEstado, id, tenantID,
	).Scan(&p.ID, &p.MesaID, &p.SucursalID, &p.Estado, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("pedido no encontrado: %w", err)
	}
	return p, nil
}
```

---

- [ ] **Step 5.5: Crear `internal/pedido/service.go`**

```go
package pedido

import (
	"context"
	"errors"
	"fmt"
)

type Service struct {
	store Store
}

func NuevoService(s Store) *Service { return &Service{store: s} }

func (svc *Service) Crear(ctx context.Context, input NuevoPedidoInput) (*Pedido, error) {
	if input.MesaID == "" {
		return nil, errors.New("mesa_id requerido")
	}
	if len(input.Items) == 0 {
		return nil, errors.New("el pedido debe tener al menos un ítem")
	}
	for _, item := range input.Items {
		if item.Cantidad <= 0 {
			return nil, fmt.Errorf("cantidad inválida para artículo %s", item.ArticuloID)
		}
	}
	sucursalID, err := svc.store.ObtenerSucursalPorMesa(ctx, input.MesaID)
	if err != nil {
		return nil, errors.New("mesa no encontrada")
	}
	return svc.store.Crear(ctx, input, sucursalID)
}

func (svc *Service) ListarActivos(ctx context.Context, sucursalID string) ([]Pedido, error) {
	return svc.store.ListarActivos(ctx, sucursalID)
}

func (svc *Service) CambiarEstado(ctx context.Context, id, tenantID, nuevoEstado string) (*Pedido, error) {
	if !estadoValido(nuevoEstado) {
		return nil, fmt.Errorf("estado inválido: %q (válidos: recibido, preparando, listo, cerrado)", nuevoEstado)
	}
	return svc.store.CambiarEstado(ctx, id, tenantID, nuevoEstado)
}

func estadoValido(estado string) bool {
	for _, v := range EstadosValidos {
		if v == estado {
			return true
		}
	}
	return false
}
```

---

- [ ] **Step 5.6: Ejecutar tests — deben pasar**

```bash
cd repos/api && go test ./internal/pedido/... -v
```

Expected: `PASS` para los tres tests.

---

- [ ] **Step 5.7: Crear `internal/pedido/handler.go`**

```go
package pedido

import (
	"encoding/json"
	"net/http"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
)

type Handlers struct {
	svc *Service
}

func NuevosHandlers(svc *Service) *Handlers { return &Handlers{svc: svc} }

func (h *Handlers) Crear(w http.ResponseWriter, r *http.Request) {
	var input NuevoPedidoInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, "body inválido", http.StatusBadRequest)
		return
	}
	p, err := h.svc.Crear(r.Context(), input)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(p)
}

func (h *Handlers) ListarActivos(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	// Listar pedidos de la primera sucursal del tenant
	// (en una iteración futura: pasar sucursal_id como query param)
	sucursalID := r.URL.Query().Get("sucursal_id")
	if sucursalID == "" && claims != nil {
		jsonError(w, "sucursal_id requerido como query param", http.StatusBadRequest)
		return
	}
	pedidos, err := h.svc.ListarActivos(r.Context(), sucursalID)
	if err != nil {
		jsonError(w, "error listando pedidos", http.StatusInternalServerError)
		return
	}
	jsonOK(w, pedidos)
}

func (h *Handlers) CambiarEstado(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	id := r.PathValue("id")
	var input CambiarEstadoInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, "body inválido", http.StatusBadRequest)
		return
	}
	p, err := h.svc.CambiarEstado(r.Context(), id, claims.TenantID, input.Estado)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	jsonOK(w, p)
}

func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
```

---

- [ ] **Step 5.8: Registrar rutas de pedido en `rutas.go` y limpiar comentarios**

Reemplazar el contenido completo de `rutas.go`:

```go
package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/aguirrepablo-iresm/mesa-click/api/internal/auth"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/carta"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/mesa"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/pedido"
	"github.com/aguirrepablo-iresm/mesa-click/api/internal/tenant"
)

func registrarRutas(mux *http.ServeMux) {
	// Health check
	mux.HandleFunc("GET /health", handlerHealth)

	// Auth
	authStore := auth.NuevoStore()
	authSvc := auth.NuevoService(authStore)
	authH := auth.NuevosHandlers(authSvc)
	mux.HandleFunc("POST /auth/magic-link", authH.SolicitarLink)
	mux.HandleFunc("GET /auth/verify", authH.VerificarToken)

	// Tenant
	tenantStore := tenant.NuevoStore()
	tenantSvc := tenant.NuevoService(tenantStore)
	tenantH := tenant.NuevosHandlers(tenantSvc)
	mux.Handle("POST /tenants", http.HandlerFunc(tenantH.Crear))
	mux.Handle("GET /tenants/me", auth.Requerir(http.HandlerFunc(tenantH.ObtenerMe)))

	// Carta (admin — protegida)
	cartaStore := carta.NuevoStore()
	cartaSvc := carta.NuevoService(cartaStore)
	cartaH := carta.NuevosHandlers(cartaSvc)
	mux.Handle("GET /carta/categorias", auth.Requerir(http.HandlerFunc(cartaH.ListarCategorias)))
	mux.Handle("POST /carta/categorias", auth.Requerir(http.HandlerFunc(cartaH.CrearCategoria)))
	mux.Handle("DELETE /carta/categorias/{id}", auth.Requerir(http.HandlerFunc(cartaH.EliminarCategoria)))
	mux.Handle("GET /carta/articulos", auth.Requerir(http.HandlerFunc(cartaH.ListarArticulos)))
	mux.Handle("POST /carta/articulos", auth.Requerir(http.HandlerFunc(cartaH.CrearArticulo)))
	mux.Handle("PATCH /carta/articulos/{id}", auth.Requerir(http.HandlerFunc(cartaH.ActualizarArticulo)))
	mux.Handle("DELETE /carta/articulos/{id}", auth.Requerir(http.HandlerFunc(cartaH.EliminarArticulo)))

	// Mesas (admin — protegidas)
	mesaStore := mesa.NuevoStore()
	mesaSvc := mesa.NuevoService(mesaStore)
	mesaH := mesa.NuevosHandlers(mesaSvc)
	mux.Handle("GET /mesas", auth.Requerir(http.HandlerFunc(mesaH.Listar)))
	mux.Handle("POST /mesas", auth.Requerir(http.HandlerFunc(mesaH.Crear)))
	mux.Handle("PATCH /mesas/{id}", auth.Requerir(http.HandlerFunc(mesaH.Actualizar)))
	mux.Handle("DELETE /mesas/{id}", auth.Requerir(http.HandlerFunc(mesaH.Eliminar)))

	// Pedidos
	pedidoStore := pedido.NuevoStore()
	pedidoSvc := pedido.NuevoService(pedidoStore)
	pedidoH := pedido.NuevosHandlers(pedidoSvc)
	mux.HandleFunc("POST /pedidos", pedidoH.Crear)
	mux.Handle("GET /pedidos", auth.Requerir(http.HandlerFunc(pedidoH.ListarActivos)))
	mux.Handle("PATCH /pedidos/{id}/estado", auth.Requerir(http.HandlerFunc(pedidoH.CambiarEstado)))

	// Endpoints públicos (sin auth — para el cliente con QR)
	mux.HandleFunc("GET /publica/{sucursal_id}/carta", cartaH.CartaPublica)
	mux.HandleFunc("GET /publica/mesa/{qr_token}", mesaH.MesaPorQR)
}

func handlerHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"estado":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}
```

---

- [ ] **Step 5.9: Ejecutar todos los tests**

```bash
cd repos/api && go test ./... -v
```

Expected: todos los tests pasan.

---

- [ ] **Step 5.10: Verificar que compila y levanta**

```bash
cd repos/api && go build ./...
```

Si tenés PostgreSQL disponible:
```bash
cd repos/api && go run .
# → "servidor escuchando puerto=8080"
curl http://localhost:8080/health
# → {"estado":"ok","timestamp":"..."}
```

---

- [ ] **Step 5.11: Commit final**

```bash
git add internal/pedido/ rutas.go
git commit -m "feat(pedido): crear pedido + listar activos + cambiar estado

Completa el core del backend (Sprint 4+5):
- auth: magic link + JWT + middleware
- tenant: onboarding (tenant + admin + sucursal)
- carta: CRUD categorías + artículos + carta pública
- mesa: CRUD + QR token automático
- pedido: crear + listar activos + cambiar estado"
```

---

## Self-Review

### Cobertura del spec

| Requisito (arquitectura-back.md) | Tarea |
|---|---|
| AuthService: magic link + JWT | Task 1 |
| TenantService: crear tenant + usuario admin | Task 2 |
| SucursalService: crea sucursal default en onboarding | Task 2 (store.Crear) |
| UsuarioService: invitaciones | ⚠️ Fuera de scope (Sprint 6) |
| MesaService: CRUD + QR token | Task 4 |
| CartaService: CRUD categorías + artículos | Task 3 |
| PedidoService: crear + estado | Task 5 |
| NotificacionService (SSE) | ⚠️ Fuera de scope (Sprint 5 / posterior) |

### Elementos fuera de scope explicitados

- Variantes de artículos (YAGNI para MVP)
- Invitación de usuarios
- SSE / tiempo real
- Tests de integración con DB real

Todos los tests son unitarios (mock store) — suficiente para verificar la lógica de negocio antes de la integración.
