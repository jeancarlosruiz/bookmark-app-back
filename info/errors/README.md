# 🐛 Error Analysis

Esta carpeta contiene análisis detallados de errores encontrados durante el desarrollo y sus soluciones.

## 📑 Documentos

### [01-foreign-key-constraint.md](./01-foreign-key-constraint.md)
**Error:** `insert or update on table "bookmarks" violates foreign key constraint "fk_bookmarks_user"`

**Temas:**
- Cross-schema foreign key constraints en PostgreSQL
- Configuración de search_path
- Validación de usuarios antes de operaciones
- GORM AutoMigrate con tablas externas

**Soluciones:**
- Configurar `SET search_path TO public, neon_auth`
- Agregar validación de usuario en seed
- Remover tabla User de AutoMigrate

---

### [02-many-to-many-join-table.md](./02-many-to-many-join-table.md)
**Error:** `column "bookmarks_id" of relation "bookmark_tags" does not exist`

**Temas:**
- Relaciones many-to-many en GORM
- Naming conventions vs. explicit configuration
- Join tables y foreign keys
- Tags `joinForeignKey` y `joinReferences`

**Soluciones:**
- Configurar explícitamente `joinForeignKey:BookmarkID`
- Configurar `joinReferences:TagID`
- Relación bidireccional entre Bookmarks y Tags

---

## 🎯 Propósito

Cada documento de error incluye:

1. **Resumen Ejecutivo** - Error, causa raíz, solución
2. **Análisis Detallado** - Por qué ocurrió el error
3. **Trace del Error** - Paso a paso del problema
4. **Solución Implementada** - Código antes/después
5. **Verificación** - Cómo probar que funciona
6. **Troubleshooting** - Problemas relacionados
7. **Prevención** - Cómo evitar en el futuro

## 🔍 Cómo Usar Esta Sección

**Si encuentras un error similar:**
1. Lee el análisis completo del error
2. Verifica si tu contexto es similar
3. Aplica la solución sugerida
4. Consulta la sección de troubleshooting si persiste

**Si encuentras un error nuevo:**
1. Documéntalo usando el mismo formato
2. Usa el siguiente número disponible (03-*)
3. Actualiza este README
4. Actualiza el README principal en `info/`

---

[⬅️ Volver al índice principal](../README.md)
