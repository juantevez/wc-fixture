// Package webhook contiene el adaptador de entrega de notificaciones via HTTP POST.
package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/wc-fixture/notification/internal/domain/notification"
	"github.com/wc-fixture/notification/internal/domain/ports"
	"github.com/wc-fixture/shared/pkg/logger"
)

const (
	webhookTimeout = 10 * time.Second
	signatureHeader = "X-WC-Signature-256"
	deliveryIDHeader = "X-WC-Delivery"
)

// Dispatcher implementa ports.Notifier entregando notificaciones
// via HTTP POST a los webhooks registrados para cada torneo.
//
// Cada entrega incluye:
//   - Header X-WC-Signature-256: HMAC-SHA256 del payload firmado con el secret del suscriptor
//   - Header X-WC-Delivery: UUID de la notificación para idempotencia en el receptor
//   - Content-Type: application/json
//   - Timeout de 10s para no bloquear el procesamiento
type Dispatcher struct {
	repo   ports.WebhookRepository
	client *http.Client
}

var _ ports.Notifier = (*Dispatcher)(nil)

func NewDispatcher(repo ports.WebhookRepository) *Dispatcher {
	return &Dispatcher{
		repo: repo,
		client: &http.Client{
			Timeout: webhookTimeout,
		},
	}
}

// Notify busca los suscriptores activos del torneo y entrega la notificación
// a todos los que están suscritos al tipo de evento de la notificación.
// Los errores de entrega individual se loguean pero no detienen las entregas restantes.
func (d *Dispatcher) Notify(ctx context.Context, n *notification.Notification) error {
	log := logger.WithFields(ctx,
		"notification_id", n.ID,
		"kind", n.Kind,
		"tournament_id", n.TournamentID,
	)

	subscribers, err := d.repo.FindByTournament(ctx, n.TournamentID)
	if err != nil {
		return fmt.Errorf("webhook_dispatcher: error buscando suscriptores: %w", err)
	}

	if len(subscribers) == 0 {
		log.Debug("sin suscriptores para el torneo — notificación omitida")
		return nil
	}

	var lastErr error
	delivered := 0

	for _, sub := range subscribers {
		if !sub.IsSubscribedTo(n.Kind) {
			continue
		}

		if err := d.deliver(ctx, n, sub); err != nil {
			log.Error("error entregando webhook",
				"subscriber_id", sub.ID,
				"url", sub.URL,
				"error", err,
			)
			lastErr = err
			continue
		}
		delivered++
	}

	log.Info("notificación entregada",
		"total_subscribers", len(subscribers),
		"delivered", delivered,
	)

	// Retornamos el último error solo si ninguna entrega fue exitosa
	if delivered == 0 && lastErr != nil {
		return lastErr
	}
	return nil
}

// deliver realiza el HTTP POST hacia un suscriptor específico.
func (d *Dispatcher) deliver(ctx context.Context, n *notification.Notification, sub notification.WebhookSubscriber) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sub.URL, bytes.NewReader(n.Payload))
	if err != nil {
		return fmt.Errorf("error construyendo request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(deliveryIDHeader, n.ID.String())

	// Firmar el payload con HMAC-SHA256 si el suscriptor tiene secret
	if sub.Secret != "" {
		sig := computeHMAC(n.Payload, sub.Secret)
		req.Header.Set(signatureHeader, "sha256="+sig)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("error en HTTP POST a %s: %w", sub.URL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return notification.ErrWebhookDeliveryFailed(sub.URL, resp.StatusCode)
	}

	return nil
}

// computeHMAC genera la firma HMAC-SHA256 del payload usando el secret del suscriptor.
// El receptor puede verificar la autenticidad del webhook comparando:
//   expected = "sha256=" + hex(HMAC-SHA256(secret, payload))
func computeHMAC(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}
