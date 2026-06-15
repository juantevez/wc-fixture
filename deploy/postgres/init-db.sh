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

# =============================================================================
# 3. Migraciones por bounded context
#    Los archivos están montados en /migrations/<context>/ desde el monorepo.
#    Se ejecutan en orden numérico (001_, 002_, ...).
# =============================================================================

run_migrations() {
    local db="$1"
    local dir="$2"
    echo "📦 Migraciones en ${db} desde ${dir}..."
    for f in "${dir}"/*.sql; do
        [ -f "$f" ] || continue   # directorio vacío o inexistente
        echo "  → $(basename "$f")"
        psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$db" -f "$f"
    done
}

run_migrations "fixture_db"   "/migrations/fixture-core"
run_migrations "venue_db"     "/migrations/venue-geo"
run_migrations "team_db"      "/migrations/team-registry"
run_migrations "ingestion_db" "/migrations/result-ingestion"

echo "✅ Migraciones aplicadas."

# =============================================================================
# 4. Señal de completado — los servicios esperan este archivo
#    El healthcheck de postgres verifica su existencia antes de reportar healthy
# =============================================================================
touch /var/lib/postgresql/data/.init_complete

echo ""
echo "🎉 wc-fixture PostgreSQL listo. Bases, extensiones y migraciones inicializadas."
