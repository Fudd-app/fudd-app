# Flujo de trabajo — Requerimientos, Bugs y Pull Requests

> Aplica a todo el equipo y a cualquier agente de IA (Claude, Codex, Gemini, Copilot, etc.).
> La **única fuente de verdad** del backlog es el tablero de GitHub **[Fudd APP](https://github.com/orgs/Fudd-app/projects/1)** (proyecto `1` de la org `Fudd-app`). No se versionan US ni bugs en `docs/`.

---

## Resumen

```
 PLANIFICACIÓN                                  DESARROLLO
 ─────────────                                  ──────────
 1. Cargar requerimiento / bug                  5. Tomar el ítem (En Curso)
 2. Refinar                                     6. Rama nueva desde qa
 3. Crear issue(s) [web] / [api] + proyecto     7. Desarrollar + testear + regresión
 4. Estimar                                     8. Commit + PR hacia qa (sigue En Curso)
                                                9. Code review del PR
                                               10. Merge a qa (En QA) → release (Resuelta)
```

Ningún desarrollo empieza sin un issue refinado, estimado y cargado en el proyecto. Esto aplica a **todo** cambio, también a docs y plantillas. La única excepción es un **hotfix** de producción, que sigue el camino rápido de [`release.md`](release.md#hotfix--única-excepción-al-flujo-completo).

El pase de `qa` a producción se hace con el [proceso de release](release.md).

---

## Convención de títulos: un ítem = un repo

Cada issue toca **un solo** repo interno. El repo va como prefijo del título:

| Prefijo | Repo | Validación obligatoria |
|---|---|---|
| `[web]` | `repos/web` (Next.js) | `cd repos/web && npm run build` |
| `[api]` | `repos/api` (Go) | `cd repos/api && go vet ./... && go test ./...` |
| `[docs]` | `docs/`, `.github/` y archivos de agentes (`AGENTS.md`, etc.), sin código | Links relativos verificados |

Formatos:

- User Story: `[web] Título descriptivo` · `[api] Título descriptivo`. Se identifica por el número que asigna GitHub (`#88`), no se agrega número en el título. Los issues anteriores conservan su `US-N` histórico.
- Bug: `[web] BUG: Descripción corta del problema` · `[api] BUG: ...`

Si un requerimiento necesita cambios en ambos repos, **se divide en dos issues** (uno `[api]` y uno `[web]`), cada uno con su propio número de issue, y el de `[web]` declara `Depende de #N` del de `[api]`. El contrato (endpoint, payload, códigos de error) se define en el issue `[api]` y el `[web]` lo referencia.

Los mismos prefijos se usan en el **título del PR**. En los commits se usa el scope de Conventional Commits: `feat(web): ...`, `fix(api): ...`.

---

## Fase 1 — Planificación

### 1. Cargar el requerimiento o bug

Quien recibe el pedido (persona o agente) lo registra tal como llegó, sin implementar nada. Si el pedido llega por chat a un agente, el agente **no empieza a codear**: arranca este flujo.

### 2. Refinar

Antes de crear el issue definitivo, responder:

- **¿Quién y para qué?** Actor (admin, encargado, mozo, comensal) y valor de negocio.
- **¿Qué repo(s)?** Si son ambos, dividir (ver arriba).
- **Criterios de aceptación** verificables (comportamiento observable, no tareas).
- **Alcance**: qué entra y qué queda explícitamente afuera.
- **Impacto**: qué funcionalidades existentes toca o podría romper (pantallas, endpoints, tablas, eventos SSE, cuotas Freemium).
- **Dependencias** con otros issues.
- **Bugs**: pasos para reproducir, esperado vs. actual, ambiente (QA / Producción) y severidad.

Para entender el impacto, el agente debe leer el código involucrado (y `docs/flows/` si toca un happy path). Lo que no se pueda resolver leyendo el código se pregunta al usuario: **una pregunta concreta a la vez**, no un cuestionario.

### 3. Crear el issue y cargarlo en el proyecto

Usar las plantillas de `.github/ISSUE_TEMPLATE/` (`user-story.md` o `bug-fix.md`).

El número del issue lo asigna GitHub al crearlo y es el que se usa en la rama, el PR y las referencias (`#88`).

Crear el issue y agregarlo al proyecto:

```bash
gh issue create --title "[api] Título descriptivo" --body-file cuerpo.md --label enhancement   # bug → --label bug
gh project item-add 1 --owner Fudd-app --url <URL-del-issue>
```

### 4. Estimar

Campos del proyecto a completar en el ítem:

| Campo | Valor |
|---|---|
| **Status** | Sin Status mientras falte información · `📋 Pendiente` cuando está refinado y estimado |
| **Estimación** | Story points Fibonacci: `1`, `2`, `3`, `5`, `8`. **Más de 8 → dividir el ítem** |
| **Sprint** | Sprint en el que se planifica. Vacío = backlog (todavía no planificado) |
| **Tipo** | `Frontend` para `[web]`, `Backend` para `[api]`, `Bug / Fix` para bugs |
| **Rama** | `feat/<nro-issue>-descripcion-corta`, `fix/<nro-issue>-descripcion-corta` o `docs/<nro-issue>-descripcion-corta` |

Guía de puntos: `1` cambio trivial y localizado · `2` cambio chico en 1–2 archivos · `3` feature chica con tests · `5` feature con varias piezas o migración · `8` feature grande o con incertidumbre.

Los agentes proponen la estimación con una línea de justificación; la persona que pidió el trabajo puede corregirla.

Editar campos por CLI (los IDs se obtienen con los dos primeros comandos):

```bash
gh project field-list 1 --owner Fudd-app --format json        # IDs de campos y opciones
gh project item-list 1 --owner Fudd-app --format json --limit 500 \
  --jq '.items[] | select(.content.number == 88) | .id'      # ID del ítem
gh project item-edit --project-id PVT_kwDOFJl_Ms4BoSi1 --id <item-id> \
  --field-id <field-id> --single-select-option-id <option-id> # Status / Sprint / Tipo
gh project item-edit --project-id PVT_kwDOFJl_Ms4BoSi1 --id <item-id> \
  --field-id <field-id> --number 3                            # Estimación
gh project item-edit --project-id PVT_kwDOFJl_Ms4BoSi1 --id <item-id> \
  --field-id <field-id> --text "feat/88-..."               # Rama
```

---

## Fase 2 — Desarrollo

### 5. Tomar el ítem

Solo ítems en `📋 Pendiente` del sprint en curso. Asignarse el issue (`gh issue edit <n> --add-assignee @me`) y mover el Status a `⚡ En Curso`.

### 6. Rama nueva desde `qa`

```bash
git checkout qa && git pull origin qa
git checkout -b feat/88-descripcion-corta         # o fix/<nro-issue>-descripcion-corta
```

Una rama por issue. Nunca trabajar sobre `qa` ni `main` directamente.

### 7. Desarrollar, testear y verificar que no se rompa nada

- Implementar **solo** lo que piden los criterios de aceptación del issue. Si aparece algo fuera de alcance, se anota como issue nuevo (vuelve a la Fase 1), no se resuelve en esta rama.
- Respetar el prefijo: un issue `[web]` no modifica `repos/api` y viceversa. Si hace falta, se frena y se crea el issue del otro repo.
- **Tests**:
  - `[api]`: agregar o actualizar tests de lo nuevo; un bug se corrige con un test que lo reproduce primero.
  - `[web]`: `npm run build` sin errores; probar el flujo en el navegador (`npm run dev`).
- **Regresión**, antes de commitear:
  1. Correr la suite **completa** del repo, no solo los tests nuevos.
  2. Buscar los usos de cada función, componente, endpoint, tipo o tabla modificados (`grep`) y verificar que siguen funcionando.
  3. Si el cambio en `[api]` altera un contrato (ruta, payload, código de error, evento SSE), revisar sus consumidores en `repos/web/lib/api.ts` y componentes, y dejarlo anotado en el issue.
  4. Recorrer el happy path afectado (`docs/flows/`) y las cuotas Free/Pro si aplica.
- Marcar en el issue los criterios cumplidos.

### 8. Commit y PR

Commits con Conventional Commits y scope del repo, explicando el **por qué**:

```bash
git commit -m "feat(api): validar cuota de sucursales antes de crear para respetar el plan Free"
git push -u origin feat/88-descripcion-corta
gh pr create --base qa --title "[api] Título" --body-file pr.md   # usa .github/pull_request_template.md
```

- El cuerpo del PR lleva `Closes #<nro-issue>`, el resultado de tests y regresión, y cómo probarlo.
- El ítem sigue en `⚡ En Curso` mientras el PR está abierto; el PR queda visible en el campo "Linked pull requests" del tablero.

### 9. Code review antes de mergear

Ningún PR se mergea sin review. Quien revisa (persona u otro agente, **nunca el mismo que lo escribió en la misma sesión**):

- Contrasta el diff con los criterios de aceptación del issue: que cumpla todo y no agregue alcance extra.
- Busca bugs, impacto en otras funcionalidades, seguridad (secretos, aislamiento por tenant, auth), y que los tests cubran lo nuevo.
- Verifica que el CI (`.github/workflows/ci.yml`) esté en verde.
- Deja los hallazgos como comentarios en el PR (`gh pr review <n> --comment|--request-changes|--approve`).

En Claude Code: `/code-review <nro-PR>` (agregar `--comment` para publicar los hallazgos en el PR).

Los hallazgos se corrigen en la misma rama y se vuelve a pedir review.

### 10. Merge y cierre

- PR aprobado + CI verde → merge a `qa` → Status `🧪 En QA`. Validar en `qa.fudd.com.ar`.
- Validado en QA → el ítem viaja en el próximo [release](release.md) → Status `✅ Resuelta` (el issue se cierra por el `Closes #` del PR de release; los PRs a `qa` no cierran issues porque `qa` no es la rama default).

---

## Estados del tablero

| Status | Significa | Quién lo mueve |
|---|---|---|
| Sin Status | Cargado, falta información o estimación | Quien lo carga |
| `📋 Pendiente` | Refinado y estimado, listo para tomar | Quien refina |
| `⚡ En Curso` | Asignado, rama creada | Quien lo toma |
| `🧪 En QA` | Mergeado en `qa`, en validación | Quien mergea |
| `✅ Resuelta` | En `main` / producción | Quien hace el release |
