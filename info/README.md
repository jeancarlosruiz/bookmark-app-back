# 📚 Documentación del Proyecto - Go Bookmark API

Este directorio contiene toda la documentación organizada del proyecto de gestión de bookmarks.

## 📖 Estructura de Documentación

### 🐛 [Errors](./errors/) - Análisis de Errores y Soluciones

Documentación detallada de errores encontrados durante el desarrollo y sus soluciones.

- **[01-foreign-key-constraint.md](./errors/01-foreign-key-constraint.md)**
  - Error de foreign key constraint cross-schema
  - Problema: `fk_bookmarks_user` violando constraint al insertar bookmarks
  - Solución: Configuración de search_path y validación de usuarios
  - Temas: PostgreSQL schemas, GORM foreign keys, cross-schema references

- **[02-many-to-many-join-table.md](./errors/02-many-to-many-join-table.md)**
  - Error en relación many-to-many entre Bookmarks y Tags
  - Problema: `column "bookmarks_id" does not exist` en tabla bookmark_tags
  - Solución: Configuración explícita de joinForeignKey y joinReferences
  - Temas: GORM many-to-many, join tables, naming conventions

### 📘 [Guides](./guides/) - Guías de Uso

Tutoriales y guías paso a paso para usar las funcionalidades del proyecto.

- **[01-database-seeding.md](./guides/01-database-seeding.md)**
  - Guía completa de database seeding para testing
  - Cómo usar el comando `seed` con datos de prueba
  - Opciones: seed básico, reset & seed, clear only
  - Troubleshooting común y mejores prácticas

### 🔍 [Analysis](./analysis/) - Análisis del Proyecto

Análisis técnicos, auditorías y diagnósticos del estado del proyecto.

- **[01-project-diagnosis.md](./analysis/01-project-diagnosis.md)**
  - Diagnóstico completo del proyecto
  - Estado actual de la arquitectura
  - Problemas identificados y recomendaciones
  - Análisis de code smells y mejoras sugeridas

### 📋 [Project](./project/) - Documentación del Proyecto

Planificación, roadmaps y documentación general del proyecto.

- **[01-development-roadmap.md](./project/01-development-roadmap.md)**
  - Roadmap de desarrollo del proyecto
  - Tareas pendientes organizadas por semana
  - Features planificadas y prioridades
  - Estado de implementación actual

---

## 🗂️ Estructura de Carpetas

```
info/
├── README.md                          # Este archivo - índice de navegación
├── errors/                            # Análisis de errores y soluciones
│   ├── 01-foreign-key-constraint.md
│   └── 02-many-to-many-join-table.md
├── guides/                            # Guías de uso y tutoriales
│   └── 01-database-seeding.md
├── analysis/                          # Análisis técnicos del proyecto
│   └── 01-project-diagnosis.md
└── project/                           # Documentación del proyecto
    └── 01-development-roadmap.md
```

## 🎯 Cómo Navegar Esta Documentación

### Si eres nuevo en el proyecto:
1. Lee el [CLAUDE.md](../CLAUDE.md) en la raíz del proyecto para entender la arquitectura
2. Revisa el [Project Diagnosis](./analysis/01-project-diagnosis.md) para conocer el estado actual
3. Consulta el [Development Roadmap](./project/01-development-roadmap.md) para ver qué sigue

### Si estás debugeando:
1. Revisa la carpeta [errors/](./errors/) para ver si tu error ya fue documentado
2. Los análisis de errores incluyen:
   - Descripción detallada del problema
   - Causa raíz con diagramas
   - Solución paso a paso
   - Código corregido
   - Troubleshooting

### Si estás implementando features:
1. Consulta las [guides/](./guides/) para herramientas disponibles
2. Revisa el [Development Roadmap](./project/01-development-roadmap.md) para prioridades
3. Lee análisis relevantes en [analysis/](./analysis/)

### Si estás haciendo code review:
1. Revisa [Project Diagnosis](./analysis/01-project-diagnosis.md) para conocer issues conocidos
2. Consulta los análisis de errores para patrones comunes a evitar
3. Verifica que las guías estén actualizadas

---

## 📝 Convenciones de Documentación

### Nomenclatura de Archivos

Todos los archivos siguen el formato: `[número]-[nombre-descriptivo].md`

- **Número**: Prefijo de dos dígitos (01, 02, etc.) para ordenamiento
- **Nombre**: Descriptivo, en kebab-case, en inglés
- **Extensión**: `.md` (Markdown)

**Ejemplos:**
- ✅ `01-foreign-key-constraint.md`
- ✅ `02-authentication-setup.md`
- ✅ `03-api-documentation.md`
- ❌ `foreign_key.md` (sin número, underscore)
- ❌ `02 Many to Many.md` (espacios)

### Estructura de Documentos

Cada documento debe incluir:

1. **Título principal** (H1) - Descriptivo y claro
2. **Resumen ejecutivo** - Breve descripción del contenido
3. **Secciones organizadas** - Con headers apropiados (H2, H3)
4. **Ejemplos de código** - Con syntax highlighting
5. **Diagramas** - Cuando aplique, usando ASCII art o markdown
6. **Referencias** - Links a recursos externos o internos

### Emojis en Headers

Usamos emojis para mejorar la navegación visual:

- 📋 Resumen/Overview
- 🔍 Análisis/Investigación
- ❌ Problema/Error
- ✅ Solución/Corrección
- 🎯 Objetivo/Meta
- 🛠️ Herramientas/Setup
- 🚨 Advertencia/Precaución
- 💡 Tip/Consejo
- 📚 Recursos/Referencias
- 🔧 Configuración
- 🧪 Testing/Pruebas
- 📖 Documentación
- 🐛 Bug/Error
- ✨ Feature/Mejora

---

## 🔄 Actualizar Esta Documentación

### Agregar un Nuevo Documento

1. **Determinar la categoría apropiada:**
   - Error solucionado → `errors/`
   - Guía de uso → `guides/`
   - Análisis técnico → `analysis/`
   - Planificación/roadmap → `project/`

2. **Nombrar el archivo:**
   ```bash
   # Obtener el siguiente número disponible
   ls info/errors/ | sort -V | tail -1
   # Si el último es 02-*, el nuevo será 03-*

   # Crear el archivo
   touch info/errors/03-authentication-error.md
   ```

3. **Actualizar este README:**
   - Agregar entrada en la sección correspondiente
   - Incluir breve descripción del contenido
   - Mantener orden numérico

4. **Seguir la estructura estándar:**
   ```markdown
   # Título del Documento

   ## 📋 Resumen Ejecutivo
   [Breve descripción]

   ## 🔍 Análisis/Contenido
   [Contenido principal]

   ## ✅ Solución/Conclusión
   [Conclusión o solución]

   ## 📚 Referencias
   [Enlaces y recursos]
   ```

### Reorganizar Documentos

Si necesitas reorganizar:

1. Actualizar referencias en otros documentos
2. Actualizar este README
3. Considerar mantener redirects o notas
4. Comunicar cambios al equipo

---

## 🎓 Tipos de Documentación

### 1. Error Analysis (errors/)

**Propósito:** Documentar errores encontrados y solucionados para referencia futura.

**Contenido debe incluir:**
- Mensaje de error exacto
- Contexto en que ocurrió
- Causa raíz del problema
- Análisis técnico detallado
- Solución implementada
- Código antes/después
- Cómo prevenir en el futuro
- Troubleshooting relacionado

**Ejemplo de estructura:**
```markdown
# Error: [Descripción breve]

## 📋 Resumen Ejecutivo
- Error original
- Causa raíz
- Solución

## 🔍 Análisis Detallado
- ¿Por qué ocurrió?
- Trace del error
- Código problemático

## ✅ Solución Implementada
- Cambios realizados
- Código corregido
- Archivos modificados

## 🛠️ Verificación
- Cómo probar que funciona
- Tests a ejecutar

## 🚨 Troubleshooting
- Errores relacionados
- Soluciones alternativas
```

### 2. Guides (guides/)

**Propósito:** Tutoriales paso a paso para usar funcionalidades del proyecto.

**Contenido debe incluir:**
- Objetivo de la guía
- Prerequisitos
- Pasos detallados con comandos
- Ejemplos de salida esperada
- Troubleshooting común
- Tips y mejores prácticas

**Ejemplo de estructura:**
```markdown
# Guía: [Funcionalidad]

## 📋 Overview
- Qué aprenderás
- Prerequisitos
- Tiempo estimado

## 🛠️ Setup
- Instalación/configuración inicial

## 📖 Pasos
1. Paso uno
2. Paso dos
3. Paso tres

## 🧪 Verificación
- Cómo verificar que funcionó

## 🚨 Troubleshooting
- Problemas comunes
- Soluciones

## 💡 Tips
- Mejores prácticas
- Optimizaciones
```

### 3. Analysis (analysis/)

**Propósito:** Análisis técnicos profundos del proyecto, auditorías, y diagnósticos.

**Contenido debe incluir:**
- Objetivo del análisis
- Metodología
- Hallazgos detallados
- Recomendaciones priorizadas
- Impacto estimado
- Plan de acción

**Ejemplo de estructura:**
```markdown
# Análisis: [Tema]

## 📋 Executive Summary
- Objetivo
- Metodología
- Hallazgos clave

## 🔍 Análisis Detallado
- Observaciones
- Datos y métricas
- Issues identificados

## 💡 Recomendaciones
- Mejoras sugeridas (priorizadas)
- Impacto estimado
- Esfuerzo requerido

## 📖 Plan de Acción
- Pasos siguientes
- Timeline sugerido
```

### 4. Project Documentation (project/)

**Propósito:** Documentación general del proyecto, roadmaps, planificación.

**Contenido debe incluir:**
- Roadmaps y planificación
- Decisiones de arquitectura
- Changelog importante
- Documentación de features
- Notas de release

---

## 📊 Estado de la Documentación

### Documentos Actuales

| Categoría | Cantidad | Última Actualización |
|-----------|----------|---------------------|
| Errors    | 2        | 2024-11-12          |
| Guides    | 1        | 2024-11-12          |
| Analysis  | 1        | 2024-10-24          |
| Project   | 1        | 2024-10-24          |
| **Total** | **5**    | -                   |

### Próximos Documentos Planeados

- [ ] `guides/02-api-usage.md` - Guía de uso de la API REST
- [ ] `guides/03-testing-setup.md` - Configuración de tests
- [ ] `errors/03-authentication-issues.md` - Cuando se implemente auth
- [ ] `analysis/02-performance-analysis.md` - Análisis de performance
- [ ] `project/02-architecture-decisions.md` - ADR (Architecture Decision Records)

### Documentos que Necesitan Actualización

- [ ] `analysis/01-project-diagnosis.md` - Actualizar después de fixes recientes
- [ ] `project/01-development-roadmap.md` - Marcar tareas completadas

---

## 🤝 Contribuir a la Documentación

### Principios

1. **Claridad sobre brevedad** - Prefiere explicaciones claras aunque sean más largas
2. **Ejemplos prácticos** - Incluye siempre ejemplos de código y comandos
3. **Actualizada** - La documentación debe reflejar el estado actual del código
4. **Accesible** - Escribe para diferentes niveles de experiencia
5. **Visual** - Usa diagramas, tablas, y formatting para mejorar comprensión

### Checklist para Nuevo Documento

- [ ] Título claro y descriptivo
- [ ] Resumen ejecutivo al inicio
- [ ] Secciones bien organizadas con headers
- [ ] Ejemplos de código con syntax highlighting
- [ ] Comandos reproducibles con output esperado
- [ ] Troubleshooting section
- [ ] Referencias a recursos externos
- [ ] Actualizado este README con el nuevo documento
- [ ] Revisión de ortografía y gramática
- [ ] Links internos funcionando

---

## 📞 Contacto y Soporte

- **Issues del Proyecto:** Ver GitHub issues (si aplica)
- **Preguntas sobre Documentación:** Crear issue con tag `documentation`
- **Sugerencias de Mejora:** Pull requests bienvenidos

---

**Última actualización:** 2024-11-12
**Mantenido por:** Proyecto Go Bookmark Team
