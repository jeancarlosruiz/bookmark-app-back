# 📘 User Guides

Esta carpeta contiene guías paso a paso y tutoriales para usar las funcionalidades del proyecto.

## 📑 Documentos

### [01-database-seeding.md](./01-database-seeding.md)
**Tema:** Database Seeding para Testing

**Contenido:**
- Cómo usar el comando `seed` para poblar la base de datos
- Configuración y prerequisitos
- Opciones: seed básico, reset & seed, clear only
- Obtener user IDs válidos
- Personalizar datos de seed
- Troubleshooting común

**Uso rápido:**
```bash
# Seed básico
go run cmd/seed/main.go --user=your-user-id

# Reset y seed (borra todo primero)
go run cmd/seed/main.go --user=your-user-id --reset

# Solo limpiar
go run cmd/seed/main.go --clear
```

---

## 🎯 Propósito

Cada guía incluye:

1. **Overview** - Qué aprenderás
2. **Prerequisitos** - Requisitos previos
3. **Setup** - Configuración inicial
4. **Pasos Detallados** - Instrucciones paso a paso con comandos
5. **Ejemplos** - Output esperado
6. **Verificación** - Cómo confirmar que funcionó
7. **Troubleshooting** - Problemas comunes y soluciones
8. **Tips** - Mejores prácticas

## 📚 Próximas Guías

- [ ] `02-api-usage.md` - Guía completa de uso de la API REST
- [ ] `03-testing-setup.md` - Configuración de tests unitarios e integración
- [ ] `04-authentication.md` - Configurar y usar autenticación JWT
- [ ] `05-deployment.md` - Deploy a producción
- [ ] `06-database-migrations.md` - Gestión de migraciones

## 🔍 Cómo Usar Esta Sección

**Para aprender una funcionalidad:**
1. Lee la guía correspondiente completa
2. Sigue los pasos uno por uno
3. Verifica los resultados esperados
4. Consulta troubleshooting si hay problemas

**Para referencia rápida:**
1. Busca la sección específica que necesitas
2. Copia el comando o código necesario
3. Adapta a tu caso de uso

---

[⬅️ Volver al índice principal](../README.md)
