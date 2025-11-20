# Guía de Deploy en Render

Esta guía explica cómo hacer deploy de la aplicación bookmark API en Render.

## Problema Común: Error con .env

**Error típico:** La aplicación falla porque no encuentra el archivo `.env`

**Causa:** En Render (y otras plataformas de producción), NO se usa archivo `.env`. Las variables de entorno se configuran directamente en la plataforma.

**Solución:** El código ha sido modificado para que `godotenv.Load()` no cause panic en producción cuando el archivo `.env` no existe. En desarrollo local seguirá cargando el archivo `.env`, pero en Render usará las variables de entorno del sistema.

## Paso 1: Preparar el Proyecto

### 1.1 Verificar que el código esté listo

El archivo `cmd/api/main.go` ya no debe hacer panic si no encuentra `.env`. Debe permitir que `godotenv.Load()` falle silenciosamente en producción.

### 1.2 Crear un Dockerfile (Opcional pero recomendado)

Render puede detectar automáticamente aplicaciones Go, pero un Dockerfile da más control:

```dockerfile
# Build stage
FROM golang:1.25.2-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/api ./cmd/api/main.go

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/bin/api .

# Expose port (Render asignará el PORT automáticamente)
EXPOSE 8080

# Run the application
CMD ["./api"]
```

### 1.3 Asegurar que .env esté en .gitignore

```bash
# Verificar que .env NO esté en git
cat .gitignore | grep ".env"
```

El archivo `.env` nunca debe estar en el repositorio.

## Paso 2: Configurar PostgreSQL en Render

### 2.1 Crear base de datos PostgreSQL

1. Ve a tu dashboard de Render: https://dashboard.render.com/
2. Click en **"New +"** → **"PostgreSQL"**
3. Configura:
   - **Name:** `bookmark-db` (o el nombre que prefieras)
   - **Database:** `bookmarks`
   - **User:** (se genera automáticamente)
   - **Region:** Elige la región más cercana
   - **Plan:** Free (para empezar)
4. Click en **"Create Database"**
5. **IMPORTANTE:** Guarda la **Internal Database URL** (la usarás en el siguiente paso)

### 2.2 Configurar esquema neon_auth (si usas autenticación externa)

Si tu aplicación usa el esquema `neon_auth` para usuarios:

```sql
-- Conectarte a tu base de datos de Render y ejecutar:
CREATE SCHEMA IF NOT EXISTS neon_auth;

-- Crear tabla users_sync si no existe
CREATE TABLE IF NOT EXISTS neon_auth.users_sync (
    id TEXT PRIMARY KEY,
    email VARCHAR(255),
    raw_json JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);
```

## Paso 3: Crear Web Service en Render

### 3.1 Crear el servicio

1. En el dashboard de Render, click en **"New +"** → **"Web Service"**
2. Conecta tu repositorio de GitHub/GitLab
3. Selecciona el repositorio `go-bookmark`

### 3.2 Configurar el servicio

**Configuración básica:**
- **Name:** `bookmark-api` (o el nombre que prefieras)
- **Region:** Misma región que la base de datos
- **Branch:** `main`
- **Root Directory:** (dejar vacío)
- **Runtime:** `Go`

**Build & Deploy:**
- **Build Command:**
  ```bash
  go build -o bin/api cmd/api/main.go
  ```

- **Start Command:**
  ```bash
  ./bin/api
  ```

**Plan:** Free (para empezar)

### 3.3 Configurar Variables de Entorno

En la sección **"Environment Variables"**, añade:

| Key | Value |
|-----|-------|
| `DATABASE_URL` | (Copia la "Internal Database URL" de tu PostgreSQL de Render) |
| `PORT` | `8080` |

**Ejemplo de DATABASE_URL:**
```
postgresql://user:password@hostname.oregon-postgres.render.com/database
```

**IMPORTANTE:** Usa la **Internal Database URL** (no la External), es más rápida y segura.

### 3.4 Deploy

Click en **"Create Web Service"** - Render automáticamente:
1. Clonará tu repositorio
2. Ejecutará `go build`
3. Iniciará tu aplicación con `./bin/api`
4. Las migraciones se ejecutarán automáticamente al iniciar

## Paso 4: Verificar el Deploy

### 4.1 Revisar logs

En el dashboard de tu servicio en Render:
- Ve a la pestaña **"Logs"**
- Deberías ver:
  ```
  Server running on :8080
  ```

### 4.2 Probar la API

Tu API estará disponible en:
```
https://bookmark-api.onrender.com
```

**Prueba endpoint básico:**
```bash
curl https://bookmark-api.onrender.com/api/users
```

## Paso 5: Configuración Adicional (Opcional)

### 5.1 Health Check

Render hace health checks automáticamente en `/`. Si quieres un endpoint específico, añade en `internal/routes/routes.go`:

```go
router.GET("/health", func(c *gin.Context) {
    c.JSON(200, gin.H{"status": "ok"})
})
```

### 5.2 Auto-Deploy

Por defecto, Render hace auto-deploy cuando haces push a `main`. Para desactivarlo:
- Ve a **Settings** → **Build & Deploy**
- Desactiva **"Auto-Deploy"**

### 5.3 Variables de entorno adicionales

Si en el futuro necesitas más variables:
- Ve a **Environment** en tu servicio
- Click en **"Add Environment Variable"**
- Añade las variables necesarias (JWT_SECRET, etc.)

## Solución de Problemas Comunes

### Error: "Failed to load config"

**Causa:** El código hace panic cuando no encuentra `.env`

**Solución:** Asegúrate de que `config.Load()` no cause panic en producción. El código debe permitir que el error sea silencioso cuando el archivo no existe.

### Error: "Failed to connect to DB"

**Causas posibles:**
1. `DATABASE_URL` no está configurada correctamente
2. Estás usando External Database URL en lugar de Internal
3. La base de datos no está en la misma región

**Solución:**
- Verifica que `DATABASE_URL` sea la **Internal Database URL**
- Verifica que ambos servicios estén en la misma región
- Revisa los logs de PostgreSQL en Render

### Error: Build timeout

**Causa:** Build tarda más de lo esperado

**Solución:**
- Usa un plan de pago (más recursos de build)
- O optimiza las dependencias

### La aplicación se reinicia constantemente

**Causa:** Probablemente está haciendo panic al iniciar

**Solución:**
- Revisa los logs en detalle
- Verifica todas las variables de entorno
- Asegúrate de que la conexión a DB funciona

## Mantenimiento

### Ver logs en tiempo real

```bash
# Instalar Render CLI
npm install -g render-cli

# Login
render login

# Ver logs
render logs -s bookmark-api --tail
```

### Hacer rollback

En el dashboard:
1. Ve a **"Events"**
2. Encuentra el deploy anterior que funcionaba
3. Click en **"Rollback to this deploy"**

## Resumen del Proceso

1. ✅ Modificar código para que no requiera `.env` en producción
2. ✅ Crear base de datos PostgreSQL en Render
3. ✅ Crear Web Service conectado a tu repositorio
4. ✅ Configurar variables de entorno (`DATABASE_URL`, `PORT`)
5. ✅ Deploy automático
6. ✅ Verificar que funcione

## Recursos

- Dashboard de Render: https://dashboard.render.com/
- Documentación de Render: https://render.com/docs
- Render + Go: https://render.com/docs/deploy-go
- PostgreSQL en Render: https://render.com/docs/databases
