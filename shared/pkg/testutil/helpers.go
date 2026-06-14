package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// ConnectPostgres abre una conexión a PostgreSQL y verifica que esté disponible.
// Reintenta hasta 10 veces con 500ms de pausa — necesario porque el contenedor
// puede tardar unos instantes en aceptar conexiones después de estar "ready".
func ConnectPostgres(t *testing.T, dsn string) *sql.DB {
	t.Helper()

	var db *sql.DB
	var err error

	for i := range 10 {
		db, err = sql.Open("postgres", dsn)
		if err == nil {
			if pingErr := db.PingContext(context.Background()); pingErr == nil {
				break
			}
			db.Close()
		}
		if i < 9 {
			time.Sleep(500 * time.Millisecond)
		}
	}
	if err != nil {
		t.Fatalf("testutil: no se pudo conectar a postgres: %v", err)
	}

	t.Cleanup(func() { db.Close() })
	return db
}

// RunMigrations ejecuta un conjunto de archivos SQL sobre la conexión dada.
// Útil para aplicar el schema de test antes de correr los tests de repositorio.
func RunMigrations(t *testing.T, db *sql.DB, sqls ...string) {
	t.Helper()
	for i, query := range sqls {
		if _, err := db.ExecContext(context.Background(), query); err != nil {
			t.Fatalf("testutil: error en migración %d: %v\nSQL: %s", i+1, err, query)
		}
	}
}

// MustUUID parsea un string UUID y falla el test si es inválido.
// Conveniente para declarar constantes de test en línea.
func MustUUID(t *testing.T, s string) [16]byte {
	t.Helper()
	var id [16]byte
	n, err := fmt.Sscanf(s,
		"%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		&id[0], &id[1], &id[2], &id[3],
		&id[4], &id[5],
		&id[6], &id[7],
		&id[8], &id[9],
		&id[10], &id[11], &id[12], &id[13], &id[14], &id[15],
	)
	if err != nil || n != 16 {
		t.Fatalf("testutil: UUID inválido %q: %v", s, err)
	}
	return id
}

// SkipIfShort omite el test si se está corriendo con -short.
// Usar en tests de integración que requieren contenedores.
func SkipIfShort(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("test de integración omitido en modo -short")
	}
}

// Ptr retorna un puntero al valor dado. Útil para campos opcionales en structs de test.
//
//	match.HomeGoals = testutil.Ptr(3)
func Ptr[T any](v T) *T {
	return &v
}
