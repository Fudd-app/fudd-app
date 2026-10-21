---
name: mantener-docs-actualizados
description: El agente es responsable de mantener siempre actualizados AGENTS.md, CLAUDE.md y todos los README del repo ante cualquier cambio de fase, sprint, US o decisión de arquitectura.
metadata:
  type: feedback
---

Soy el encargado de mantener actualizados los siguientes archivos ante cualquier cambio relevante en el proyecto:

- `AGENTS.md` (raíz) — reglas generales, fases, sprints, fechas (fuente de verdad principal)
- `repos/web/AGENTS.md` — reglas del frontend, sprint en curso
- `CLAUDE.md` / `CODEX.md` / `repos/web/CLAUDE.md` / `repos/web/CODEX.md` — apuntan a AGENTS.md
- `.github/copilot-instructions.md` — instrucciones para Copilot/Codex
- `README.md` (raíz y sub-repos) — documentación pública del proyecto

**Why:** El repo se comparte con múltiples personas y agentes de IA. Si estos archivos están desactualizados, los colaboradores arrancan con contexto incorrecto.

**How to apply:** Al final de cualquier tarea que cambie fase, sprint, US, stack, estructura de carpetas o decisión de arquitectura — actualizar los archivos afectados sin que el usuario lo pida. No esperar instrucción explícita.
