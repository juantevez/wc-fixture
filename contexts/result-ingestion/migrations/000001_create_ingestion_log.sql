-- ============================================================================
-- 001_create_ingestion_log.sql
-- Tabla de idempotencia para result-ingestion.
-- ============================================================================

CREATE TABLE IF NOT EXISTS ingestion_log (
    id               BIGSERIAL   NOT NULL,
    idempotency_key  TEXT        NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT ingestion_log_pkey   PRIMARY KEY (id),
    CONSTRAINT ingestion_log_key_uk UNIQUE (idempotency_key)
);

CREATE INDEX IF NOT EXISTS idx_ingestion_log_key
    ON ingestion_log (idempotency_key);

COMMENT ON TABLE  ingestion_log               IS 'Registro de idempotencia — evita doble procesamiento de resultados';
COMMENT ON COLUMN ingestion_log.idempotency_key IS 'Clave única: {match_id}:{home_total}:{away_total}';
