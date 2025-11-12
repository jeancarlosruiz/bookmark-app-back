# 📋 Project Documentation

Esta carpeta contiene documentación general del proyecto, roadmaps, planificación y decisiones de arquitectura.

## 📑 Documentos

### [01-development-roadmap.md](./01-development-roadmap.md)
**Tema:** Roadmap de Desarrollo - Plan de 4 Semanas

**Contenido:**
- Planificación semanal de features
- Tareas organizadas por prioridad
- Estado de implementación actual
- Dependencies entre features
- Timeline estimado

**Estructura del roadmap:**
- **Semana 1:** CRUD completo + Validaciones
- **Semana 2:** Búsqueda, filtros y autenticación
- **Semana 3:** Features avanzados (metadata scraping, analytics)
- **Semana 4:** Testing, documentación y deployment

---

## 🎯 Propósito

Esta sección contiene:

1. **Roadmaps** - Planificación de desarrollo
2. **Architecture Decisions** - ADRs (Architecture Decision Records)
3. **Feature Specs** - Especificaciones de features
4. **Meeting Notes** - Notas de reuniones importantes
5. **Changelogs** - Registro de cambios importantes
6. **Release Notes** - Notas de versiones

## 📚 Próximos Documentos

- [ ] `02-architecture-decisions.md` - ADRs del proyecto
- [ ] `03-api-specification.md` - Especificación completa de la API
- [ ] `04-changelog.md` - Registro de cambios por versión
- [ ] `05-contributing.md` - Guía para contribuir al proyecto
- [ ] `06-deployment-guide.md` - Guía de deployment

## 🔍 Cómo Usar Esta Sección

**Para entender la dirección del proyecto:**
1. Lee el Development Roadmap
2. Consulta los ADRs para decisiones técnicas
3. Revisa el changelog para historial

**Para planificar trabajo:**
1. Consulta el roadmap para prioridades
2. Verifica dependencies entre tareas
3. Actualiza el estado conforme avances

**Para nuevos contributors:**
1. Lee el roadmap y ADRs primero
2. Consulta el contributing guide
3. Revisa el changelog para contexto

---

## 📊 Estado del Proyecto

### Features Completados ✅
- Models y database schema
- CRUD básico de bookmarks (READ, CREATE, DELETE)
- Relaciones many-to-many con tags
- User management (read-only, external auth)
- Database seeding para testing

### En Progreso 🚧
- UPDATE bookmark (función vacía)
- Validaciones de entrada
- Error handling mejorado

### Próximos Steps 📅
- Completar UPDATE operation
- Implementar search y filtering
- Agregar autenticación JWT
- Metadata scraping de URLs
- Visit tracking

### Deuda Técnica 🔧
- Tests unitarios e integración
- Logging estructurado
- API documentation (Swagger)
- Performance optimization
- Security hardening

---

[⬅️ Volver al índice principal](../README.md)
