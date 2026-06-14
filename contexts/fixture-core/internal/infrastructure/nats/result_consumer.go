package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/wc-fixture/fixture-core/internal/application/commands"
	sharedevents "github.com/wc-fixture/shared/pkg/events"
	"github.com/wc-fixture/shared/pkg/logger"
)

const (
	consumerName    = "fixture-core-results"
	maxDeliver      = 5
	ackWaitDuration = 30 * time.Second
)

// ResultConsumer consume eventos fixture.result.ingested publicados por
// result-ingestion y los procesa con RegisterMatchResultHandler.
//
// Garantías:
//   - At-least-once delivery: si el handler falla, el mensaje se reintenta
//   - maxDeliver = 5: tras 5 fallos el mensaje queda en dead-letter
//   - AckWait = 30s: tiempo máximo para procesar antes de redelivery
//   - BackOff progresivo: 1s → 5s → 15s → 30s → 60s entre reintentos
type ResultConsumer struct {
	js       jetstream.JetStream
	handler  *commands.RegisterMatchResultHandler
	consumer jetstream.Consumer
}

// NewResultConsumer crea el consumer durable de JetStream para resultados.
func NewResultConsumer(
	ctx context.Context,
	js jetstream.JetStream,
	handler *commands.RegisterMatchResultHandler,
	streamName string,
) (*ResultConsumer, error) {

	consumerCfg := jetstream.ConsumerConfig{
		Name:          consumerName,
		Durable:       consumerName,
		FilterSubject: string(sharedevents.SubjectResultIngested),
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       ackWaitDuration,
		MaxDeliver:    maxDeliver,
		DeliverPolicy: jetstream.DeliverAllPolicy,
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
		return nil, fmt.Errorf("result_consumer: error creando consumer: %w", err)
	}

	return &ResultConsumer{
		js:       js,
		handler:  handler,
		consumer: consumer,
	}, nil
}

// Start inicia el loop de consumo en una goroutine.
// Se detiene limpiamente cuando ctx es cancelado.
func (c *ResultConsumer) Start(ctx context.Context) error {
	msgCtx, err := c.consumer.Messages(
		jetstream.PullMaxMessages(10),
	)
	if err != nil {
		return fmt.Errorf("result_consumer: error iniciando consumo: %w", err)
	}

	go func() {
		defer msgCtx.Stop()
		log := logger.NewDefault("result-consumer")
		log.Info("consumer iniciado", "subject", sharedevents.SubjectResultIngested)

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
					log.Error("error procesando mensaje — se reintentará",
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

// processMessage deserializa el envelope, construye el comando y lo ejecuta.
func (c *ResultConsumer) processMessage(ctx context.Context, msg jetstream.Msg) error {
	var evt sharedevents.DomainEvent
	if err := json.Unmarshal(msg.Data(), &evt); err != nil {
		// Mensaje malformado — ack para evitar reintento infinito
		_ = msg.Ack()
		return fmt.Errorf("result_consumer: mensaje malformado, descartado: %w", err)
	}

	log := logger.WithFields(ctx,
		"event_id", evt.EventID,
		"event_type", evt.EventType,
		"aggregate_id", evt.AggregateID,
	)

	cmd, err := buildRegisterMatchResultCmd(evt)
	if err != nil {
		_ = msg.Ack()
		log.Error("payload inválido, descartando mensaje", "error", err)
		return nil
	}

	if err := c.handler.Handle(ctx, cmd); err != nil {
		return fmt.Errorf("result_consumer: error ejecutando handler: %w", err)
	}

	_ = msg.Ack()
	log.Info("resultado procesado exitosamente", "match_id", cmd.MatchID)
	return nil
}

// ResultIngestedPayload es el schema del payload que publica result-ingestion.
// Debe mantenerse sincronizado con el schema de result-ingestion.
type ResultIngestedPayload struct {
	TournamentID string `json:"tournament_id"`
	MatchID      string `json:"match_id"`
	HomeTeamID   string `json:"home_team_id"`
	AwayTeamID   string `json:"away_team_id"`
	HomeGoals    int    `json:"home_goals"`
	AwayGoals    int    `json:"away_goals"`
	HomeGoalsET  *int   `json:"home_goals_et,omitempty"`
	AwayGoalsET  *int   `json:"away_goals_et,omitempty"`
	HomeGoalsPen *int   `json:"home_goals_pen,omitempty"`
	AwayGoalsPen *int   `json:"away_goals_pen,omitempty"`
	CompletedAt  string `json:"completed_at"` // RFC3339 UTC
	RegisteredBy string `json:"registered_by"`
}

// buildRegisterMatchResultCmd deserializa el payload del evento en el comando.
func buildRegisterMatchResultCmd(evt sharedevents.DomainEvent) (commands.RegisterMatchResultCmd, error) {
	var payload ResultIngestedPayload
	if err := evt.DecodePayload(&payload); err != nil {
		return commands.RegisterMatchResultCmd{}, fmt.Errorf("error deserializando payload: %w", err)
	}

	tournamentID, err := uuid.Parse(payload.TournamentID)
	if err != nil {
		return commands.RegisterMatchResultCmd{}, fmt.Errorf("tournament_id inválido: %w", err)
	}
	matchID, err := uuid.Parse(payload.MatchID)
	if err != nil {
		return commands.RegisterMatchResultCmd{}, fmt.Errorf("match_id inválido: %w", err)
	}
	homeTeamID, err := uuid.Parse(payload.HomeTeamID)
	if err != nil {
		return commands.RegisterMatchResultCmd{}, fmt.Errorf("home_team_id inválido: %w", err)
	}
	awayTeamID, err := uuid.Parse(payload.AwayTeamID)
	if err != nil {
		return commands.RegisterMatchResultCmd{}, fmt.Errorf("away_team_id inválido: %w", err)
	}
	completedAt, err := time.Parse(time.RFC3339, payload.CompletedAt)
	if err != nil {
		return commands.RegisterMatchResultCmd{}, fmt.Errorf("completed_at inválido: %w", err)
	}

	return commands.RegisterMatchResultCmd{
		TournamentID: tournamentID,
		MatchID:      matchID,
		HomeTeamID:   homeTeamID,
		AwayTeamID:   awayTeamID,
		HomeGoals:    payload.HomeGoals,
		AwayGoals:    payload.AwayGoals,
		HomeGoalsET:  payload.HomeGoalsET,
		AwayGoalsET:  payload.AwayGoalsET,
		HomeGoalsPen: payload.HomeGoalsPen,
		AwayGoalsPen: payload.AwayGoalsPen,
		CompletedAt:  completedAt,
		RegisteredBy: payload.RegisteredBy,
	}, nil
}

// backoffDelay calcula el delay de NAK según el número de entregas previas.
func backoffDelay(msg jetstream.Msg) time.Duration {
	delays := []time.Duration{1, 5, 15, 30, 60}
	meta, err := msg.Metadata()
	if err != nil || int(meta.NumDelivered) >= len(delays) {
		return 60 * time.Second
	}
	return delays[meta.NumDelivered] * time.Second
}
