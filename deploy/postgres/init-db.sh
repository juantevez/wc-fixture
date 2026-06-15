#!/bin/bash
set -e

# =============================================================================
# init-db.sh — Inicialización de PostgreSQL para wc-fixture
# Se ejecuta automáticamente al primer arranque del contenedor postgres.
# =============================================================================

echo "🐳 Iniciando setup de bases de datos wc-fixture..."

# =============================================================================
# 1. Crear bases de datos por bounded context
# =============================================================================
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    CREATE DATABASE fixture_db;
    GRANT ALL PRIVILEGES ON DATABASE fixture_db TO $POSTGRES_USER;

    CREATE DATABASE venue_db;
    GRANT ALL PRIVILEGES ON DATABASE venue_db TO $POSTGRES_USER;

    CREATE DATABASE team_db;
    GRANT ALL PRIVILEGES ON DATABASE team_db TO $POSTGRES_USER;

    CREATE DATABASE ingestion_db;
    GRANT ALL PRIVILEGES ON DATABASE ingestion_db TO $POSTGRES_USER;
EOSQL

echo "✅ Bases de datos creadas: fixture_db, venue_db, team_db, ingestion_db"

# =============================================================================
# 2. Extensiones
# =============================================================================

echo "🗺️  Habilitando PostGIS en venue_db..."
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "venue_db" <<-EOSQL
    CREATE EXTENSION IF NOT EXISTS postgis;
    CREATE EXTENSION IF NOT EXISTS postgis_topology;
    CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
EOSQL

echo "🔑 Habilitando uuid-ossp en demás bases..."
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "fixture_db" <<-EOSQL
    CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
EOSQL
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "team_db" <<-EOSQL
    CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
EOSQL
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "ingestion_db" <<-EOSQL
    CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
EOSQL

echo "✅ Extensiones habilitadas correctamente."

# =============================================================================
# 3. Seeds de datos iniciales
#    Nota: los seeds dependen de las migraciones (tablas creadas).
#    Las migraciones las corre golang-migrate al arrancar cada servicio.
#    Acá solo dejamos los scripts disponibles para correr manualmente
#    o desde el Makefile después del primer up.
#
#    Para ejecutar los seeds una vez que los servicios estén arriba:
#      make seed  (desde la raíz del monorepo)
#    O manualmente:
#      psql -U wc2026 -d team_db  -f deploy/postgres/seed_teams.sql
#      psql -U wc2026 -d venue_db -f deploy/postgres/seed_venues.sql
# =============================================================================

echo ""
echo "📋 Bases de datos disponibles:"
psql --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" \
    -c "\l" | grep -E "fixture_db|venue_db|team_db|ingestion_db"

# Señal de completado para el healthcheck de docker-compose
touch /var/lib/postgresql/data/.init_complete

echo ""
echo "🎉 wc-fixture PostgreSQL listo."
echo "   Próximo paso: correr migraciones (automático al arrancar servicios)"
echo "   Luego seeds: make seed"
