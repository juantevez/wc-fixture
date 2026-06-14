// Package postgres contiene los adaptadores de infraestructura para PostgreSQL.
// Implementa los puertos definidos en domain/ports y application/queries.
//
// Decisiones de implementación:
//   - pgx/v5 como driver directo (sin ORM) para máximo control sobre queries
//   - pgxpool para connection pooling
//   - Event sourcing: fixture_events como fuente de verdad del aggregate
//   - Read models: tablas desnormalizadas (matches, group_standings) para queries
//   - Transacciones explícitas en Save() para garantizar atomicidad
package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Config contiene los parámetros de conexión a PostgreSQL.
type Config struct {
	DSN             string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

// DefaultConfig retorna una configuración con valores razonables para producción.
func DefaultConfig(dsn string) Config {
	return Config{
		DSN:             dsn,
		MaxConns:        25,
		MinConns:        5,
		MaxConnLifetime: 30 * time.Minute,
		MaxConnIdleTime: 5 * time.Minute,
	}
}

// NewPool crea y verifica un pool de conexiones pgx.
// Falla si no puede conectar dentro del contexto dado.
func NewPool(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("postgres: config inválida: %w", err)
	}

	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("postgres: error al crear pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: ping fallido: %w", err)
	}

	return pool, nil
}
