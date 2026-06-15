package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/wc-fixture/notification/internal/application/handlers"
	sharedevents "github.com/wc-fixture/shared/pkg/events"
	"github.com/wc-fixture/shared/pkg/logger"
)

const (
	consumerName    = "notification-updates"
	maxDeliver      = 5
	ackWaitDuration = 30 * time.Second
)

// FixtureConsumer consume todos los eventos del stream FIXTURE_EVENTS
// y los despacha al EventDispatcher para su procesamiento y notificación.
type FixtureConsumer struct {
	js         jetstream.JetStream
	dispatcher *handlers.EventDispatcher
	consumer   jetstream.Consumer
}

// NewFixtureConsumer crea el consumer durable de JetStream para notification.
// Consume todos los subjects del stream (fixture.>) con DeliverNew —
// solo eventos nuevos desde el arranque del servicio.
func NewFixtureConsumer(
	ctx context.Context,
	js jetstream.JetStream,
	dispatcher *handlers.EventDispatcher,
	streamName string,
) (*FixtureConsumer, error) {

	consumerCfg := jetstream.ConsumerConfig{
		Name:          consumerName,
		Durable:       consumerName,
		FilterSubject: string(sharedevents.StreamSubjectsFilter), // fixture.>
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       ackWaitDuration,
		MaxDeliver:    maxDeliver,
		// DeliverNew: notification solo necesita eventos en tiempo real,
		// no replay histórico (fixture-core ya tiene el estado completo).
		DeliverPolicy: jetstream.DeliverNewPolicy,
		BackOff: []time.Duration{
			1 * time.Second,
			5 * time.Second,
			15 * time.Second,
			30 * time.Second,
			60 * time.Second,
		},
	}

	consumer, err := js.CreateOrUpdateConsumer(ctx, streamName, consumerCfg)
	if err != nil {
		return nil, fmt.Errorf("fixture_consumer: error creando consumer: %w", err)
	}

	return &FixtureConsumer{
		js:         js,
		dispatcher: dispatcher,
		consumer:   consumer,
	}, nil
}

// Start inicia el loop de consumo en una goroutine.
// Se detiene limpiamente cuando ctx es cancelado.
func (c *FixtureConsumer) Start(ctx context.Context) error {
	msgCtx, err := c.consumer.Messages(
		jetstream.PullMaxMessages(20),
	)
	if err != nil {
		return fmt.Errorf("fixture_consumer: error iniciando consumo: %w", err)
	}

	go func() {
		defer msgCtx.Stop()
		log := logger.NewDefault("notification-consumer")
		log.Info("consumer NATS iniciado", "subject", sharedevents.StreamSubjectsFilter)

		for {
			select {
			case <-ctx.Done():
				log.Info("consumer detenido por cancelación de contexto")
				return
			default:
				msg, err := msgCtx.Next()
				if err != nil {
					if ctx.Err() != nil {
						return
					}
					log.Error("error recibiendo mensaje", "error", err)
					time.Sleep(1 * time.Second)
					continue
				}

				if err := c.processMessage(ctx, msg); err != nil {
					meta, _ := msg.Metadata()
					log.Error("error procesando mensaje",
						"error", err,
						"subject", msg.Subject(),
						"num_delivered", meta.NumDelivered,
					)
					_ = msg.NakWithDelay(backoffDelay(msg))
				}
			}
		}
	}()

	return nil
}

// processMessage deserializa el envelope y lo despacha al handler correcto.
func (c *FixtureConsumer) processMessage(ctx context.Context, msg jetstream.Msg) error {
	var evt sharedevents.DomainEvent
	if err := json.Unmarshal(msg.Data(), &evt); err != nil {
		// Mensaje malformado — ack para evitar reintento infinito
		_ = msg.Ack()
		logger.FromContext(ctx).Error("mensaje malformado, descartado",
			"subject", msg.Subject(), "error", err,
		)
		return nil
	}

	if err := c.dispatcher.Dispatch(ctx, evt); err != nil {
		return fmt.Errorf("fixture_consumer: dispatch fallido para %q: %w", evt.EventType, err)
	}

	_ = msg.Ack()
	return nil
}

// backoffDelay calcula el delay de NAK según entregas previas.
func backoffDelay(msg jetstream.Msg) time.Duration {
	delays := []time.Duration{1, 5, 15, 30, 60}
	meta, err := msg.Metadata()
	if err != nil || int(meta.NumDelivered) >= len(delays) {
		return 60 * time.Second
	}
	return delays[meta.NumDelivered] * time.Second
}
