# Guía de Implementación: Backend Go con Better Auth JWT

Esta guía contiene todas las instrucciones necesarias para modificar tu backend de Go para que funcione con los tokens JWT EdDSA generados por Better Auth.

---

## 📋 Resumen de Cambios

Tu backend Go actualmente usa HMAC (HS256) con una clave simétrica compartida. Lo vamos a cambiar para que use **EdDSA (Ed25519)** con verificación de clave pública vía JWKS (JSON Web Key Set).

**Ventaja principal:** Si tu backend Go es comprometido, los atacantes NO podrán crear tokens falsos porque solo tienen la clave pública, no la privada (que permanece segura en Next.js).

---

## 🔧 Paso 1: Instalar Dependencias

Ejecuta estos comandos en el directorio de tu backend Go:

```bash
go get github.com/MicahParks/keyfunc/v3
go get github.com/golang-jwt/jwt/v5
```

Verifica que `jwt/v5` sea versión >= 5.0.0 (necesaria para soporte EdDSA):

```bash
go list -m github.com/golang-jwt/jwt/v5
```

---

## 🔧 Paso 2: Reemplazar `middleware/protected.go`

Reemplaza **completamente** el contenido de tu archivo `middleware/protected.go` con este código:

```go
package middleware

import (
    "context"
    "log"
    "net/http"
    "os"
    "strings"
    "time"

    "github.com/MicahParks/keyfunc/v3"
    "github.com/golang-jwt/jwt/v5"
)

var (
    jwks    keyfunc.Keyfunc
    jwksURL string = getEnv("JWKS_URL", "http://localhost:3000/api/auth/jwks")
)

// Inicializar JWKS al arrancar la aplicación
func init() {
    var err error

    // Crear cliente JWKS con auto-refresh
    jwks, err = keyfunc.NewDefaultCtx(
        context.Background(),
        []string{jwksURL},
        keyfunc.Options{
            RefreshInterval:  1 * time.Hour,      // Refrescar claves cada hora
            RefreshRateLimit: 5 * time.Minute,    // Límite de tasa para prevenir abuso
            RefreshTimeout:   10 * time.Second,   // Timeout de red
        },
    )

    if err != nil {
        log.Fatalf("❌ Error al inicializar JWKS desde %s: %v", jwksURL, err)
    }

    log.Printf("✅ JWKS inicializado correctamente desde %s", jwksURL)
}

// Claims personalizados que coinciden con el payload de Better Auth
type BetterAuthClaims struct {
    UserID    string `json:"user_id"`     // Better Auth usa user_id, no sub
    Email     string `json:"email"`
    Name      string `json:"name"`
    SessionID string `json:"session_id"`
    jwt.RegisteredClaims
}

func Protected(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // 1. Extraer token del header Authorization
        authHeader := r.Header.Get("Authorization")
        if authHeader == "" {
            http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
            return
        }

        tokenString := strings.TrimPrefix(authHeader, "Bearer ")
        if tokenString == authHeader {
            http.Error(w, "Invalid Authorization format (expected: Bearer <token>)", http.StatusUnauthorized)
            return
        }

        // 2. Parsear y verificar JWT con JWKS (clave pública)
        token, err := jwt.ParseWithClaims(
            tokenString,
            &BetterAuthClaims{},
            jwks.Keyfunc, // Usa la clave pública obtenida del JWKS endpoint
            jwt.WithValidMethods([]string{"EdDSA"}), // Solo permitir EdDSA
            jwt.WithExpirationRequired(),             // Requiere campo exp
        )

        if err != nil {
            log.Printf("❌ Error al validar JWT: %v", err)
            http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
            return
        }

        if !token.Valid {
            http.Error(w, "Invalid token", http.StatusUnauthorized)
            return
        }

        // 3. Extraer claims personalizados
        claims, ok := token.Claims.(*BetterAuthClaims)
        if !ok {
            http.Error(w, "Invalid token claims", http.StatusUnauthorized)
            return
        }

        // 4. Agregar información del usuario al contexto para handlers posteriores
        ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
        ctx = context.WithValue(ctx, "email", claims.Email)
        ctx = context.WithValue(ctx, "name", claims.Name)

        log.Printf("✅ Usuario autenticado: %s (%s)", claims.Name, claims.UserID)

        // 5. Continuar con el siguiente handler
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// Limpieza al cerrar la aplicación (opcional, llamar en shutdown)
func CleanupJWKS() {
    if jwks != nil {
        jwks.EndBackground()
        log.Println("🛑 JWKS background refresh detenido")
    }
}

// Helper para obtener variables de entorno con fallback
func getEnv(key, fallback string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return fallback
}
```

---

## 🔧 Paso 3: Configurar Variables de Entorno

Actualiza tu archivo `.env` (o donde gestiones las variables de entorno):

```env
# NUEVA - URL del endpoint JWKS de Next.js
JWKS_URL=http://localhost:3000/api/auth/jwks  # Desarrollo local
# JWKS_URL=https://tudominio.com/api/auth/jwks  # Producción

# ELIMINAR - Ya no se usa con EdDSA
# JWT_SECRET=...
```

**Importante:**

- En **desarrollo**: `http://localhost:3000/api/auth/jwks`
- En **producción**: `https://tudominio.com/api/auth/jwks` (HTTPS obligatorio)

---

## 🔧 Paso 4: Actualizar Uso del Middleware

Si usas **Gin** (framework Go), el middleware se usa así:

```go
package main

import (
    "yourproject/middleware"
    "github.com/gin-gonic/gin"
)

func main() {
    r := gin.Default()

    // Rutas públicas (sin autenticación)
    r.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{"status": "ok"})
    })

    // Rutas protegidas (requieren JWT)
    protected := r.Group("/api")
    protected.Use(middleware.Protected)
    {
        protected.GET("/bookmark/user/:userId", getBookmarksHandler)
        protected.POST("/bookmark", createBookmarkHandler)
        protected.PUT("/bookmark/:id", updateBookmarkHandler)
        protected.DELETE("/bookmark/:id", deleteBookmarkHandler)
    }

    r.Run(":8080")
}

// Ejemplo de handler que usa el user_id del contexto
func getBookmarksHandler(c *gin.Context) {
    // Obtener user_id del contexto (agregado por el middleware)
    userID, exists := c.Get("user_id")
    if !exists {
        c.JSON(500, gin.H{"error": "user_id not found in context"})
        return
    }

    // Usar el userID autenticado
    log.Printf("Fetching bookmarks for user: %s", userID)

    // Tu lógica aquí...
    c.JSON(200, gin.H{"bookmarks": []string{}})
}
```

Si usas **net/http estándar**:

```go
package main

import (
    "net/http"
    "yourproject/middleware"
)

func main() {
    mux := http.NewServeMux()

    // Rutas públicas
    mux.HandleFunc("/health", healthHandler)

    // Rutas protegidas
    mux.Handle("/api/bookmark/", middleware.Protected(http.HandlerFunc(bookmarksHandler)))

    http.ListenAndServe(":8080", mux)
}
```

---

## ✅ Verificación y Pruebas

### Test 1: Verificar que JWKS está funcionando

1. **Inicia tu backend Go:**

```bash
go run main.go
```

2. **Verifica en los logs:**

Deberías ver:

```
✅ JWKS inicializado correctamente desde http://localhost:3000/api/auth/jwks
```

Si ves un error:

```
❌ Error al inicializar JWKS desde http://localhost:3000/api/auth/jwks: ...
```

**Solución:** Asegúrate de que Next.js esté corriendo en `http://localhost:3000`.

---

### Test 2: Probar autenticación con JWT

1. **Inicia sesión en Next.js** (http://localhost:3000)

2. **Obtén tu JWT** desde la consola del navegador (DevTools):

```javascript
// Pega esto en la consola del navegador
const { data } = await authClient.token();
console.log(data.token);
```

3. **Copia el token** (empieza con `eyJ...`)

4. **Haz una petición al backend Go:**

```bash
# Reemplaza <TU_JWT_AQUI> con el token copiado
# Reemplaza <USER_ID> con tu ID de usuario

curl -H "Authorization: Bearer <TU_JWT_AQUI>" \
     http://localhost:8080/api/bookmark/user/<USER_ID>
```

**Resultado esperado:**

- **200 OK** con datos de bookmarks
- Log en Go: `✅ Usuario autenticado: TuNombre (tu-user-id)`

**Si obtienes 401:**

- Verifica que el token no haya expirado (7 días por defecto)
- Verifica que JWKS esté funcionando (revisa logs de Go)
- Decodifica el token en https://jwt.io para verificar que tiene `user_id`

---

### Test 3: Verificar el payload del JWT

Visita https://jwt.io y pega tu token. Deberías ver:

**Header:**

```json
{
  "alg": "EdDSA",
  "typ": "JWT"
}
```

**Payload:**

```json
{
  "user_id": "tu-user-id-aqui",
  "email": "tu@email.com",
  "name": "Tu Nombre",
  "session_id": "session-id-aqui",
  "iat": 1234567890,
  "exp": 1234567890
}
```

**IMPORTANTE:** Debe tener el campo `user_id` (no `sub`).

---

## 🚨 Troubleshooting

### Error: "Failed to create JWKS"

**Causa:** El backend Go no puede conectarse al endpoint JWKS de Next.js.

**Soluciones:**

1. Verifica que Next.js esté corriendo: `curl http://localhost:3000/api/auth/jwks`
2. Verifica la variable `JWKS_URL` en tu `.env`
3. Si estás en Docker, usa el hostname correcto (no localhost)

---

### Error: "Invalid or expired token"

**Causas posibles:**

1. **Token expirado:** Los tokens tienen 7 días de validez. Genera uno nuevo iniciando sesión de nuevo.

2. **Reloj desincronizado:** Los servidores Next.js y Go tienen diferente hora.

   ```bash
   # Sincronizar hora en Linux/Mac
   sudo ntpdate -s time.nist.gov
   ```

3. **Token malformado:** Verifica en jwt.io que el token sea válido.

---

### Error: "Invalid token claims"

**Causa:** El token no contiene el payload esperado.

**Solución:** Verifica en `lib/auth/better-auth.ts` que `definePayload` incluya `user_id`:

```typescript
definePayload: (user, session) => ({
  user_id: user.id,  // ← Debe estar presente
  email: user.email,
  name: user.name,
  session_id: session.id,
}),
```

---

### Error: "JWKS endpoint returns 404"

**Causa:** El plugin JWT no está configurado correctamente en Next.js.

**Solución:**

1. Verifica que `lib/auth/better-auth.ts` tenga `jwt()` en plugins
2. Reinicia Next.js: `npm run dev`
3. Prueba el endpoint: `curl http://localhost:3000/api/auth/jwks`
   - Debe retornar JSON con claves públicas

---

### El middleware funciona pero el context no tiene user_id

**Causa:** El middleware está usando Gin pero accedes al contexto como `http.Request`.

**Solución para Gin:**

```go
func myHandler(c *gin.Context) {
    userID, _ := c.Get("user_id")  // ✅ Correcto con Gin
}
```

**Solución para net/http:**

```go
func myHandler(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("user_id")  // ✅ Correcto con net/http
}
```

---

## 🔒 Consideraciones de Seguridad

### Para Producción

1. **HTTPS Obligatorio:**
   - Tanto Next.js como Go backend deben usar HTTPS
   - Los tokens JWT se envían en headers, HTTPS previene intercepción

2. **CORS en JWKS:**
   - Si Go y Next.js están en dominios diferentes
   - Configura CORS en `/api/auth/jwks` para permitir solo tu dominio Go

3. **Rotación de Claves:**
   - Better Auth rota claves automáticamente cada 30 días
   - El grace period de 7 días evita que tokens válidos fallen durante la rotación

4. **Monitoreo:**
   - Log todos los fallos de validación JWT
   - Alerta si hay muchos tokens inválidos (posible ataque)

5. **Rate Limiting:**
   - Limita peticiones al endpoint JWKS (previene DoS)
   - Usa `RefreshRateLimit` en la config de keyfunc

---

## 📊 Comparación: Antes vs Después

| Aspecto                | HMAC (Antes)                   | EdDSA (Ahora)                |
| ---------------------- | ------------------------------ | ---------------------------- |
| **Tipo de clave**      | Simétrica (compartida)         | Asimétrica (pública/privada) |
| **Si Go es hackeado**  | Pueden crear tokens falsos     | NO pueden crear tokens       |
| **Rotación de claves** | Manual en ambos servicios      | Automática vía JWKS          |
| **Escalabilidad**      | Difícil con múltiples backends | Fácil, todos usan JWKS       |
| **Seguridad**          | Buena                          | Excelente                    |
| **Complejidad**        | Baja                           | Media                        |

---

## 📚 Referencias Técnicas

- [Better Auth JWT Plugin](https://www.better-auth.com/docs/plugins/jwt)
- [keyfunc JWKS Library](https://pkg.go.dev/github.com/MicahParks/keyfunc/v3)
- [golang-jwt EdDSA Support](https://pkg.go.dev/github.com/golang-jwt/jwt/v5)
- [RFC 8037 - EdDSA](https://datatracker.ietf.org/doc/html/rfc8037)
- [JWKS Specification](https://datatracker.ietf.org/doc/html/rfc7517)

---

## ✅ Checklist Final

Antes de desplegar a producción, verifica:

- [ ] JWKS se inicializa correctamente (log: `✅ JWKS inicializado`)
- [ ] Tokens válidos retornan 200 OK
- [ ] Tokens inválidos retornan 401 Unauthorized
- [ ] Variable `JWKS_URL` apunta a producción (HTTPS)
- [ ] Logs muestran usuarios autenticados correctamente
- [ ] CORS configurado si Next.js y Go están en dominios diferentes
- [ ] HTTPS habilitado en ambos servicios
- [ ] Rate limiting configurado en endpoints protegidos
- [ ] Monitoreo de errores JWT implementado

---

## 🆘 ¿Necesitas Ayuda?

Si encuentras problemas:

1. **Verifica los logs** de ambos servicios (Next.js y Go)
2. **Decodifica el JWT** en https://jwt.io para inspeccionar el payload
3. **Prueba el endpoint JWKS** manualmente: `curl http://localhost:3000/api/auth/jwks`
4. **Verifica las versiones** de las dependencias (`go.mod`)

**Comando útil para debugging:**

```bash
# Ver contenido del JWKS
curl -s http://localhost:3000/api/auth/jwks | jq .

# Ver payload del JWT (requiere jq)
echo "TU_JWT" | cut -d. -f2 | base64 -d | jq .
```

---

## 🎯 Próximos Pasos Recomendados

1. **Implementar refresh tokens** para expiración más corta de JWT (15 min en lugar de 7 días)
2. **Agregar logging estructurado** para auditoría de seguridad
3. **Implementar rate limiting** en rutas protegidas
4. **Configurar alertas** para fallos de autenticación masivos
5. **Documentar** tus endpoints protegidos con ejemplos de JWT

---

¡Todo listo! Con estos cambios, tu backend Go ahora verifica tokens JWT EdDSA de forma segura usando JWKS. 🚀
