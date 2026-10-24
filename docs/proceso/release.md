# Proceso de release — Pase de `qa` a producción

> Aplica a todo el equipo y a cualquier agente de IA. Complementa el [flujo de trabajo](flujo-de-trabajo.md): un issue llega a producción **solo** a través de un release o de un hotfix.

---

## Resumen

```
 1. Decidir el corte         5. PR release → main (review + CI)
 2. Rama release/ desde qa   6. Merge → deploy automático
 3. Novedades + versión      7. Smoke test en producción
 4. Pre-flight de prod       8. Tag, GitHub Release, tablero y merge a qa
```

- **A demanda**: no hay release por sprint. Se corta cuando el equipo decide que lo que está en `qa` está listo.
- **Sin freeze de `qa`**: la rama de release fija el contenido; `qa` sigue recibiendo PRs.
- **Versión por fecha del corte**: `vAAAA.MM.DD` (ej. `v2026.10.24`). Un segundo corte el mismo día es `vAAAA.MM.DD.1`.

---

## 1. Decidir el corte

Entra todo lo que está en `qa` al momento del corte. Antes de cortar, revisar los ítems en `🧪 En QA` del tablero:

```bash
gh project item-list 1 --owner Fudd-app --format json --limit 500 \
  --jq '.items[] | select(.status == "🧪 En QA") | "#\(.content.number) \(.content.title)"'
```

Cada ítem tiene que estar **validado en `qa.fudd.com.ar`**. Si alguno no lo está: se valida, se revierte en la rama de release, o queda apagado por flag (ej. `MCP_HABILITADO`).

## 2. Crear la rama de release

```bash
git checkout qa && git pull origin qa
git checkout -b release/v2026.10.24
```

Si `main` tiene commits que no están en `qa` (no debería: ver "Protección de `main`"), primero se mergea `main` en la rama de release.

## 3. Novedades y versión

El primer commit de la rama agrega la entrada de la versión al **inicio** de `repos/web/content/novedades.json`. Esa entrada es la versión que ve el usuario en la campana del dashboard.

```json
{
  "version": "2026.10.24",
  "fecha": "2026-10-24",
  "items": [
    { "tipo": "nuevo", "titulo": "...", "descripcion": "...", "roles": ["admin", "encargado"] }
  ]
}
```

**Qué se comunica al usuario**: solo lo que suma valor o cambia algo que ya usaba.

| Tipo | Cuándo |
|---|---|
| `nuevo` | Funcionalidad que el usuario no tenía |
| `cambio` | Algo que ya usaba ahora funciona distinto (incluye un bug que el usuario notaba) |

No se comunican: dependencias, migraciones, CI, runtime, refactors, herramientas internas (Control Tower). Los textos salen del campo **"Novedad para el usuario"** de cada issue; se escriben en segunda persona, con las mismas etiquetas que se ven en pantalla, y con los `roles` a los que les importa (`admin`, `encargado`, `mozo`, `cocina`).

Si el release no tiene nada visible para el usuario, igual se agrega la entrada con `items: []`, así la versión mostrada coincide con la de producción.

```bash
git commit -m "chore(web): novedades de la versión 2026.10.24"
```

## 4. Pre-flight de producción

Antes de abrir el PR, completar y pegar en el cuerpo del PR:

- [ ] **Variables de entorno**: comparar las que lee el código en la rama contra `main` y cargar las nuevas en Render (`fudd-app-api-prod`) y Vercel (proyecto prod) **antes** del merge.
  ```bash
  for r in origin/main HEAD; do git grep -h -o -E 'Getenv\("[A-Z_]+"\)|process\.env\.[A-Z_]+' $r -- repos/api repos/web ':!*_test.go' | sort -u > env_${r//\//_}.txt; done
  diff env_origin_main.txt env_HEAD.txt
  ```
- [ ] **Migraciones**: listar las nuevas (`git diff --name-only origin/main -- repos/api/migrations`). Corren solas al arrancar la API y no tienen down. Verificar que el código **anterior** tolere el schema nuevo durante los minutos en que Vercel y Render despliegan a destiempo.
- [ ] **Backup de la base**: crear un branch en Neon prod llamado `pre-release-vAAAA-MM-DD` (instantáneo).
- [ ] **Ventana**: hacer el merge fuera del horario de servicio de los locales. Si un cambio altera la operación (ej. un estado nuevo de pedido), avisar antes a los negocios.
- [ ] **Rollback**: anotar qué deploy de Vercel y de Render es el actual, por si hay que volver.

## 5. PR de release

```bash
git push -u origin release/v2026.10.24
gh pr create --base main --title "Release v2026.10.24" --body-file release.md
```

Cuerpo del PR:

```markdown
## Release v2026.10.24

### Issues
Closes #84
Closes #85
...

### Novedades para el usuario
(copiar los títulos de novedades.json)

### Pre-flight
(checklist del paso 4 completa: env vars, migraciones, backup, ventana, rollback)
```

- Un `Closes #N` **por línea** por cada issue del release: los PRs a `qa` no cierran issues (`qa` no es la rama default); este PR sí.
- Code review y CI en verde, igual que cualquier PR.
- **Merge commit, nunca squash ni rebase**: con squash, `main` y `qa` divergen y el próximo release arrastra conflictos.

## 6. Deploy

El merge despliega solo: Vercel (`fudd.com.ar`) y Render (`fudd-app-api-prod`). En los logs de Render, confirmar que las migraciones nuevas aplicaron y que la API arrancó.

## 7. Smoke test en producción

Recorrer en `fudd.com.ar`, con un negocio de prueba:

- [ ] Login de admin y de staff.
- [ ] Happy path admin ([`docs/flows/happy-path-admin-negocio.md`](../flows/happy-path-admin-negocio.md)): carta, mesas y QR.
- [ ] Happy path comensal ([`docs/flows/happy-path-cliente.md`](../flows/happy-path-cliente.md)): escanear QR, pedir, seguimiento y cuenta.
- [ ] Dashboard en vivo y KDS reciben el pedido por SSE.
- [ ] Las funcionalidades del release.
- [ ] La campana muestra la versión nueva.

Si algo crítico falla: **rollback** (ver abajo) y se abre un bug.

## 8. Cierre

```bash
git checkout main && git pull origin main
git tag v2026.10.24 && git push origin v2026.10.24
gh release create v2026.10.24 --title "v2026.10.24" --notes-file notas.md
gh pr create --base qa --head release/v2026.10.24 --title "Release v2026.10.24 → qa"
```

- **GitHub Release**: novedades para el usuario + sección técnica (migraciones, env vars, issues).
- **Tablero**: los ítems del release pasan a `✅ Resuelta`.
- **PR de vuelta a `qa`**: lleva a `qa` los commits propios de la rama de release (novedades, fixes del pase). No requiere un nuevo review, porque su contenido ya se revisó en el PR de release. Merge commit.
- Borrar la rama `release/...` después de ese merge.

---

## Rollback

| Capa | Cómo |
|---|---|
| Web | Vercel → Deployments → deploy anterior → **Instant Rollback** |
| API | Render → `fudd-app-api-prod` → Events → deploy anterior → **Rollback** |
| Base | Las migraciones no se revierten. Si el código anterior no tolera el schema nuevo, se corrige hacia adelante con un hotfix. Restaurar el branch de backup de Neon es el **último recurso**: se pierde todo lo escrito después del deploy |

Después de un rollback, `main` sigue teniendo el código nuevo: el arreglo se hace como hotfix.

---

## Hotfix — única excepción al flujo completo

Un hotfix es un bug **en producción** que bloquea la operación de los locales (no se pueden tomar o cobrar pedidos, login caído) o expone datos. Todo lo demás espera al próximo release.

Lo que se saltea: el refinamiento, la estimación y la validación en `qa` **antes** de desarrollar. Lo que no se saltea: issue, PR, code review y CI.

1. Crear el issue `[web] BUG: ...` o `[api] BUG: ...` en el momento, con el síntoma y la severidad (se completa después), y cargarlo en el tablero en `⚡ En Curso`.
2. Rama desde **`main`**: `git checkout main && git pull && git checkout -b hotfix/<nro-issue>-descripcion`.
3. El cambio mínimo que resuelve el problema, con su test si es `[api]`.
4. PR hacia **`main`** con `Closes #N`, review (puede ser de otro agente) y CI en verde. Merge commit.
5. Verificar en producción.
6. Tag `vAAAA.MM.DD.N` y, si el usuario notaba el problema, entrada en `novedades.json` como `cambio`.
7. **Llevar el fix a `qa`** en el mismo día: PR `hotfix/<nro-issue>-descripcion` → `qa` (sin nuevo review, ya se revisó), merge commit.
8. Completar el issue (causa, impacto) y moverlo a `✅ Resuelta`.

---

## Protección de `main`

`main` solo recibe merges de `release/*` y `hotfix/*` por PR. Se recomienda activar en GitHub (Settings → Branches) una regla para `main` que exija PR, CI en verde (`backend-test`, `frontend-build`) y al menos una aprobación.
