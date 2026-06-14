// Package nats contiene los adaptadores de infraestructura para NATS JetStream.
// Implementa ports.EventPublisher (salida) y el consumer de eventos entrantes
// desde result-ingestion (entrada).
//
// Streams y consumers:
//   - Stream FIXTURE_EVENTS agrupa todos los subjects fixture.>
//   - Producer: fixture-core publica eventos de dominio hacia notification y otros
//   - Consumer: fixture-core consume fixture.result.ingested desde result-ingestion
package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	sharedevents "github.com/wc-fixture/shared/pkg/events"
)

// Config contiene los parámetros de conexión y configuración de NATS JetStream.
type Config struct {
	URL            string
	StreamName     string
	StreamSubjects []string
	MaxAge         time.Duration // retención de mensajes en el stream
	Replicas       int           // réplicas para HA (1 en dev, 3 en prod)
}

// DefaultConfig retorna una configuración razonable para desarrollo local.
func DefaultConfig(url string) Config {
	return Config{
		URL:            url,
		StreamName:     sharedevents.Stream,
		StreamSubjects: []string{string(sharedevents.StreamSubjectsFilter)},
		MaxAge:         720 * time.Hour, // 30 días
		Replicas:       1,
	}
}

// Connect establece la conexión a NATS y verifica que JetStream esté disponible.
func Connect(ctx context.Context, cfg Config) (*nats.Conn, jetstream.JetStream, error) {
	nc, err := nats.Connect(cfg.URL,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(10),
		nats.ReconnectWait(2*time.Second),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				fmt.Printf("nats: desconectado: %v\n", err)
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			fmt.Printf("nats: reconectado a %s\n", nc.ConnectedUrl())
		}),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("nats: error conectando a %s: %w", cfg.URL, err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, nil, fmt.Errorf("nats: error inicializando JetStream: %w", err)
	}

	if err := ensureStream(ctx, js, cfg); err != nil {
		nc.Close()
		return nil, nil, err
	}

	return nc, js, nil
}

// ensureStream crea el stream FIXTURE_EVENTS si no existe, o lo actualiza
// si la configuración cambió. Es idempotente — seguro de llamar en cada startup.
func ensureStream(ctx context.Context, js jetstream.JetStream, cfg Config) error {
	streamCfg := jetstream.StreamConfig{
		Name:      cfg.StreamName,
		Subjects:  cfg.StreamSubjects,
		Storage:   jetstream.FileStorage,
		Retention: jetstream.LimitsPolicy,
		MaxAge:    cfg.MaxAge,
		Replicas:  cfg.Replicas,
		// Garantías de entrega
		Discard:    jetstream.DiscardOld,
		MaxMsgSize: 1 << 20, // 1 MB max por mensaje
	}

	_, err := js.CreateOrUpdateStream(ctx, streamCfg)
	if err != nil {
		return fmt.Errorf("nats: error configurando stream %q: %w", cfg.StreamName, err)
	}

	return nil
}
