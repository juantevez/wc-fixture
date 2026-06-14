// Package idempotency contiene las implementaciones del puerto IdempotencyStore.
package idempotency

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wc-fixture/result-ingestion/internal/domain/result"
)

// postgresStore implementa ports.IdempotencyStore usando PostgreSQL.
// Persiste las claves de idempotencia en la tabla ingestion_log.
type postgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *postgresStore {
	return &postgresStore{pool: pool}
}

// Exists verifica si la clave de idempotencia ya fue registrada.
func (s *postgresStore) Exists(ctx context.Context, key string) (bool, error) {
	const q = `SELECT EXISTS(SELECT 1 FROM ingestion_log WHERE idempotency_key = $1)`

	var exists bool
	if err := s.pool.QueryRow(ctx, q, key).Scan(&exists); err != nil {
		return false, fmt.Errorf("idempotency_store: error verificando clave: %w", err)
	}
	return exists, nil
}

// Register registra la clave de idempotencia.
// Si la clave ya existe retorna ErrDuplicateIngestion (race condition).
func (s *postgresStore) Register(ctx context.Context, key string) error {
	const q = `
		INSERT INTO ingestion_log (idempotency_key)
		VALUES ($1)
		ON CONFLICT (idempotency_key) DO NOTHING`

	tag, err := s.pool.Exec(ctx, q, key)
	if err != nil {
		return fmt.Errorf("idempotency_store: error registrando clave: %w", err)
	}

	// RowsAffected = 0 significa que ya existía (ON CONFLICT DO NOTHING)
	if tag.RowsAffected() == 0 {
		return result.ErrDuplicateIngestion(key)
	}
	return nil
}

// ── Memory store para tests ───────────────────────────────────────────────────

// MemoryStore implementa IdempotencyStore en memoria para tests unitarios.
type MemoryStore struct {
	keys map[string]bool
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{keys: make(map[string]bool)}
}

func (s *MemoryStore) Exists(_ context.Context, key string) (bool, error) {
	return s.keys[key], nil
}

func (s *MemoryStore) Register(_ context.Context, key string) error {
	if s.keys[key] {
		return result.ErrDuplicateIngestion(key)
	}
	s.keys[key] = true
	return nil
}

// Verificar que ambas implementaciones satisfacen el contrato
var (
	_ interface {
		Exists(context.Context, string) (bool, error)
		Register(context.Context, string) error
	} = (*postgresStore)(nil)
	_ interface {
		Exists(context.Context, string) (bool, error)
		Register(context.Context, string) error
	} = (*MemoryStore)(nil)
)

// pgxErrNoRows es el centinela de pgx para fila no encontrada.
var _ = pgx.ErrNoRows
var _ = errors.New
