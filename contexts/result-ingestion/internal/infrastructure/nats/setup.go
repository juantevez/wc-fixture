// Package nats contiene los adaptadores de infraestructura NATS JetStream
// para el bounded context result-ingestion.
package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	sharedevents "github.com/wc-fixture/shared/pkg/events"
)

// Config contiene los parámetros de conexión a NATS.
type Config struct {
	URL        string
	StreamName string
}

func DefaultConfig(url string) Config {
	return Config{
		URL:        url,
		StreamName: sharedevents.Stream,
	}
}

// Connect establece la conexión a NATS y verifica JetStream.
// El stream FIXTURE_EVENTS debe haber sido creado por fixture-core
// antes de que result-ingestion publique en él.
func Connect(ctx context.Context, cfg Config) (*nats.Conn, jetstream.JetStream, error) {
	nc, err := nats.Connect(cfg.URL,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(10),
		nats.ReconnectWait(2*time.Second),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				fmt.Printf("result-ingestion/nats: desconectado: %v\n", err)
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			fmt.Printf("result-ingestion/nats: reconectado a %s\n", nc.ConnectedUrl())
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

	return nc, js, nil
}
