#!/bin/bash
set -e

# =============================================================================
# migrate_and_seed.sh — Migraciones + seeds para wc-fixture
# Corre directamente contra el contenedor postgres en ejecución.
# Uso: bash deploy/postgres/migrate_and_seed.sh
#      (desde la raíz del monorepo wc-fixture/)
# =============================================================================

POSTGRES_CONTAINER="wc_postgres"
POSTGRES_USER="wc2026"

# Colores para output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

ok()   { echo -e "${GREEN}✅ $1${NC}"; }
info() { echo -e "${YELLOW}⏳ $1${NC}"; }
err()  { echo -e "${RED}❌ $1${NC}"; exit 1; }

# Verificar que el contenedor está corriendo
docker exec "$POSTGRES_CONTAINER" pg_isready -U "$POSTGRES_USER" > /dev/null 2>&1 \
  || err "El contenedor $POSTGRES_CONTAINER no está disponible. Correr: make up"

echo ""
echo "============================================="
echo " wc-fixture — Migraciones + Seeds"
echo "============================================="
echo ""

# ── Helper: ejecutar SQL en una base específica ────────────────────────────────
run_sql() {
  local db=$1
  local file=$2
  local desc=$3
  info "$desc"
  docker exec -i "$POSTGRES_CONTAINER" psql -U "$POSTGRES_USER" -d "$db" \
    -v ON_ERROR_STOP=1 < "$file" \
    && ok "$desc" \
    || err "Falló: $desc"
}

# =============================================================================
# 1. MIGRACIONES — team_db
# =============================================================================
echo "--- team-registry (team_db) ---"
run_sql team_db contexts/team-registry/migrations/001_create_teams.sql         "001 create teams"
run_sql team_db contexts/team-registry/migrations/002_create_confederations.sql "002 create confederations"
echo ""

# =============================================================================
# 2. MIGRACIONES — venue_db
# =============================================================================
echo "--- venue-geo (venue_db) ---"
run_sql venue_db contexts/venue-geo/migrations/001_create_venues.sql           "001 create venues"
run_sql venue_db contexts/venue-geo/migrations/002_create_venue_distances.sql  "002 create venue_distances"
run_sql venue_db contexts/venue-geo/migrations/003_postgis_indexes.sql         "003 postgis indexes + seed venues (migración)"
echo ""

# =============================================================================
# 3. MIGRACIONES — fixture_db
# =============================================================================
echo "--- fixture-core (fixture_db) ---"
run_sql fixture_db contexts/fixture-core/migrations/001_create_tournaments.sql   "001 create tournaments"
run_sql fixture_db contexts/fixture-core/migrations/002_create_groups.sql        "002 create groups"
run_sql fixture_db contexts/fixture-core/migrations/003_create_matches.sql       "003 create matches"
run_sql fixture_db contexts/fixture-core/migrations/004_create_match_results.sql "004 create match_results"
run_sql fixture_db contexts/fixture-core/migrations/005_create_group_standings.sql "005 create group_standings"
run_sql fixture_db contexts/fixture-core/migrations/006_create_fixture_events.sql "006 create fixture_events"
run_sql fixture_db contexts/fixture-core/migrations/007_create_indexes.sql       "007 create indexes"
echo ""

# =============================================================================
# 4. MIGRACIONES — ingestion_db
# =============================================================================
echo "--- result-ingestion (ingestion_db) ---"
run_sql ingestion_db contexts/result-ingestion/migrations/001_create_ingestion_log.sql "001 create ingestion_log"
echo ""

# =============================================================================
# 5. SEEDS
# =============================================================================
echo "--- Seeds de datos ---"
run_sql team_db  deploy/postgres/seed_teams.sql  "seed 48 selecciones → team_db"
run_sql venue_db deploy/postgres/seed_venues.sql "seed 16 estadios + distancias → venue_db"
run_sql fixture_db deploy/postgres/seed_fixture.sql "seed fixture: torneo + grupos + 72 partidos → fixture_db"
echo ""

echo "============================================="
ok "Todo listo. Migraciones y seeds completados."
echo "============================================="
echo ""
echo "Verificar:"
echo "  docker exec -i wc_postgres psql -U wc2026 -d team_db  -c 'SELECT confederation, COUNT(*) FROM teams GROUP BY 1 ORDER BY 1;'"
echo "  docker exec -i wc_postgres psql -U wc2026 -d venue_db -c 'SELECT country, COUNT(*) FROM venues GROUP BY 1 ORDER BY 1;'"
