# =============================================================================
# Makefile raíz — wc-fixture monorepo
# =============================================================================

.PHONY: up down down-v build ps logs migrate seed migrate-and-seed clean

COMPOSE = docker compose --env-file deploy/docker/.env -f deploy/docker/docker-compose.yml

# ── Stack ──────────────────────────────────────────────────────────────────────

up:
	$(COMPOSE) up --build -d

down:
	$(COMPOSE) down

down-v:
	$(COMPOSE) down -v

build:
	$(COMPOSE) build

ps:
	$(COMPOSE) ps

logs:
	$(COMPOSE) logs -f

logs-%:
	$(COMPOSE) logs -f $*

# ── Setup inicial (una sola vez) ───────────────────────────────────────────────
# Corre TODAS las migraciones y luego los seeds.
# Prerequisito: make up y esperar que todos los containers estén healthy.

migrate-and-seed:
	@bash deploy/postgres/migrate_and_seed.sh

# Alias por si se quiere correr por separado
migrate:
	@bash deploy/postgres/migrate_and_seed.sh

seed:
	@echo "🏳️  Seeding teams..."
	@docker exec -i wc_postgres psql -U wc2026 -d team_db  -v ON_ERROR_STOP=1 < deploy/postgres/seed_teams.sql
	@echo "🗺️  Seeding venues..."
	@docker exec -i wc_postgres psql -U wc2026 -d venue_db -v ON_ERROR_STOP=1 < deploy/postgres/seed_venues.sql
	@echo "✅ Seeds completados."

# ── Utilidades ─────────────────────────────────────────────────────────────────

clean:
	find . -name "bin" -type d -exec rm -rf {} + 2>/dev/null || true
	@echo "✅ Binarios eliminados."
