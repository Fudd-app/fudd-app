<!--
Título del PR: [web] o [api] + issue. Ej: "[api] US-88: Título" / "[web] BUG: Descripción".
Flujo completo: docs/proceso/flujo-de-trabajo.md — no se mergea sin code review.
-->

## 📌 Resumen

- **Issue:** Closes #
- **Repo:** `[web]` | `[api]`
- **Rama destino:** `qa`

---

## 🛠️ Qué incluye este PR

- 

---

## 🧪 Cómo probarlo

1. 

---

## 🔁 Regresión

Funcionalidades revisadas que podrían verse afectadas y cómo se verificaron:

- 

---

## ✅ Checklist

- [ ] Rama creada a partir de `qa` (`feat/US-XX-...` o `fix/<nro-issue>-...`).
- [ ] Solo modifica el repo del prefijo (`repos/web` o `repos/api`).
- [ ] Criterios de aceptación del issue cumplidos y marcados.
- [ ] Suite completa en verde: `cd repos/api && go test ./...` | `cd repos/web && npm run build`.
- [ ] Sin credenciales ni secretos expuestos en el código.
- [ ] Code review aprobado por otra persona u otro agente (no el autor).
