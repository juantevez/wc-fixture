// Package testutil provee helpers para tests de integración que requieren
// infraestructura real (PostgreSQL con PostGIS, NATS JetStream).
// Usa testcontainers-go para levantar contenedores efímeros por test suite.
package testutil

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// PostgresContainer representa un contenedor PostgreSQL+PostGIS levantado para tests.
type PostgresContainer struct {
	testcontainers.Container
	DSN string
}

// NewPostgresContainer levanta un contenedor postgis/postgis:16-3.4 y retorna
// el DSN de conexión. El contenedor se termina automáticamente al finalizar el test.
//
//	func TestRepository(t *testing.T) {
//	    pg := testutil.NewPostgresContainer(t)
//	    db := testutil.ConnectPostgres(t, pg.DSN)
//	    ...
//	}
func NewPostgresContainer(t *testing.T) *PostgresContainer {
	t.Helper()

	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "postgis/postgis:16-3.4",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_DB":       "testdb",
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("testutil: no se pudo levantar contenedor postgres: %v", err)
	}

	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("testutil: error al terminar contenedor postgres: %v", err)
		}
	})

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("testutil: no se pudo obtener host del contenedor: %v", err)
	}
	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("testutil: no se pudo obtener puerto del contenedor: %v", err)
	}

	dsn := fmt.Sprintf("postgres://test:test@%s:%s/testdb?sslmode=disable", host, port.Port())
	return &PostgresContainer{Container: container, DSN: dsn}
}

// NATSContainer representa un contenedor NATS con JetStream habilitado para tests.
type NATSContainer struct {
	testcontainers.Container
	URL string
}

// NewNATSContainer levanta un contenedor nats:2.10-alpine con JetStream y retorna
// la URL de conexión. El contenedor se termina automáticamente al finalizar el test.
func NewNATSContainer(t *testing.T) *NATSContainer {
	t.Helper()

	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "nats:2.10-alpine",
		Cmd:          []string{"-js"},
		ExposedPorts: []string{"4222/tcp"},
		WaitingFor:   wait.ForLog("Server is ready").WithStartupTimeout(30 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("testutil: no se pudo levantar contenedor nats: %v", err)
	}

	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("testutil: error al terminar contenedor nats: %v", err)
		}
	})

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("testutil: no se pudo obtener host del contenedor nats: %v", err)
	}
	port, err := container.MappedPort(ctx, "4222")
	if err != nil {
		t.Fatalf("testutil: no se pudo obtener puerto del contenedor nats: %v", err)
	}

	return &NATSContainer{
		Container: container,
		URL:       fmt.Sprintf("nats://%s:%s", host, port.Port()),
	}
}
