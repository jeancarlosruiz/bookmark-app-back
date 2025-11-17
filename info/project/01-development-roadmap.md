# 🧩 Calendario de Desarrollo — Backend (Go)

**Duración:** 4 semanas — 2 horas diarias  
**Stack:** Go, GORM o Ent, PostgreSQL, JWT, Fiber o Gin  
**Repositorio:** `bookmark-manager-api`  
**Deploy:** Railway / Render / Fly.io

---

## 🧭 Semana 1 — Setup y estructura del backend

### Día 1

[x] Crear repositorio en GitHub.
[x] Iniciar proyecto con `go mod init`.
[x] Decidir ORM (GORM recomendado).
[x] Definir estructura de carpetas (`cmd/`, `internal/`, `pkg/`).

### Día 2

[x] Configurar conexión con PostgreSQL.
[x] Crear migraciones iniciales: `users`, `bookmarks`, `tags`.

### Día 3

[x] Crear modelo y repositorio de `Bookmark`.
[x] Endpoint básico `GET /bookmarks`, `POST /bookmarks`.

### Día 4

[x] Añadir middlewares.
[x] Probar endpoints con Postman.
[x] Crear endpoints de signup para que guarde UserID (de neo auth), nombre y demas informacion necesaria.

### Día 5

[x] Implementar `PUT /bookmarks/:id` y `DELETE /bookmarks/:id`. (Para actualizar deberia leer y buscar como seria la mejor forma de crear una validacion y como ponerla en los middleware quizas dividir mejor services, controller, middleware, validators)
[x] Validar campos y relaciones.

---

## ⚙️ Semana 2 — CRUD completo y búsqueda

### Día 6

[x] Crear seed inicial con bookmarks y tags.
[x] Endpoint `GET /bookmarks/:id`.

### Día 7

[x] Endpoint para creación de bookmark con tags relacionados.
[x] Relación many-to-many `bookmark_tags`.

### Día 8

[x] Endpoint `/bookmarks/search?q=title`.01-
[x] Endpoint `/bookmarks?tags=tag1,tag2`.

### Día 9

[] Añadir campos `view_count`, `last_visited_at`.
[] Endpoint para incrementar visitas.

### Día 10

[] Campos booleanos `is_pinned`, `is_archived`.
[] Endpoints de actualización rápida.

---

## 🔐 Semana 3 — Autenticación y metadata scraping

### Día 11

- Crear endpoints `/auth/register` y `/auth/login`.
- Generar tokens JWT.

### Día 12

- Middleware de autenticación.
- Asociar `user_id` en todos los bookmarks.

### Día 13

- Implementar scraping de metadatos (favicon, title, description).
- Usar `goquery` o `colly`.

### Día 14

- Validar URLs y evitar duplicados.
- Endpoint `/bookmarks/duplicate`.

### Día 15

- Implementar “sort” dinámico (`ORDER BY` en SQL).
- Mejorar rendimiento de consultas.

---

## 💎 Semana 4 — Extras y deploy

### Día 16

- Endpoint de estadísticas (`/stats`).
- Calcular bookmarks más visitados, más usados.

### Día 17

- Endpoint `/user/avatar` (Cloudinary o UploadThing).
- Subida y manejo de archivos.

### Día 18

- Optimización general (índices, caché opcional).
- Endpoint `/health` para monitoreo.

### Día 19

- Escribir tests unitarios (`testing` package).
- Test de integración para endpoints clave.

### Día 20

- Deploy en Railway o Render.
- Documentar API en Swagger.
- Crear README con diagrama de arquitectura.
