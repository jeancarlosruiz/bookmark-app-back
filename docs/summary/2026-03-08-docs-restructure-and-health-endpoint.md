---
title: Restructuración de docs/ e implementación de health endpoint
date: 2026-03-08
---

# Resumen de cambios — 2026-03-08

## Health Endpoint

Se implementó `GET /health`

### Archivos creados/modificados

- **`internal/controllers/health_controller.go`** (nuevo) — Controller con checks de PostgreSQL y Redis, lógica de 3 niveles de status (healthy/degraded/unhealthy), y cálculo de uptime.
- **`internal/routes/routes.go`** (modificado) — Registrada ruta `/health` fuera de los grupos protegidos.

### Decisiones técnicas

- Acceso directo a `database.DB` y `database.RedisClient` sin service/repository (no hay lógica de negocio).
- `context.WithTimeout` de 3s por check para evitar bloqueos.
- `*string` para el campo `error` en JSON (serializa como `null` cuando no hay error).
- Redis caído = degraded (200), DB caída = unhealthy (503).

---

## Restructuración de docs/

Se organizaron los 7 archivos que estaban sueltos en `docs/` dentro de subdirectorios con convención clara.

### Estructura final

```
docs/
├── architecture/
│   ├── 01-overview.md          (nuevo)
│   ├── 02-stack.md             (nuevo)
│   ├── 04-system-design.md     (antes PORTFOLIO_SYSTEM_DESIGN.md)
│   └── 05-diagrams.md          (antes DIAGRAMS.md)
├── decisions/
│   └── _template.md            (nuevo — template para ADRs)
├── plans/
│   ├── 2025-01-05-migration-implementation.md  (antes BACKEND_MIGRATION_IMPLEMENTATION.md)
│   ├── 2026-03-07-health-endpoint-plan.md      (antes plan-health.md)
│   └── 2026-03-07-health-endpoint-prompt.md    (antes health-prompt.md)
├── reference/
│   ├── backend-system-design-notes.md          (antes backend-system-design.md)
│   └── portfolio-page-content.mdx              (antes backend.mdx)
└── summary/
    └── 2026-03-08-docs-restructure-and-health-endpoint.md  (este archivo)
```

### Documentos de arquitectura creados

- **01-overview.md** — Visión general del proyecto, estructura de directorios, arquitectura de 3 capas, ciclo de vida de requests, modelo de autenticación.
- **02-stack.md** — Stack tecnológico completo: dependencias de `go.mod`, configuración de DB/Redis, infraestructura de producción, variables de entorno.

### Otros cambios

- Eliminado `cambios.md` (contenido migrado a la estructura de docs).
- Todos los archivos reubicados recibieron headers YAML con metadata (`title`, `created`, `updated`).
