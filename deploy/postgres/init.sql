-- =============================================================================
-- init.sql — SQL adicional ejecutado después de init-db.sh
-- Útil para datos de referencia globales o configuraciones de PostgreSQL.
-- =============================================================================

-- Configuraciones de rendimiento recomendadas para desarrollo local
-- (en producción ajustar según hardware disponible)

-- Aumentar work_mem para queries geoespaciales de PostGIS
ALTER SYSTEM SET work_mem = '64MB';

-- Aumentar shared_buffers para mejor cache de páginas
ALTER SYSTEM SET shared_buffers = '256MB';

-- Habilitar extensión pg_stat_statements para monitoring de queries lentas
-- (opcional, requiere reinicio del servidor)
-- CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

SELECT pg_reload_conf();
