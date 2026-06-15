package main

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/wc-fixture/notification/internal/domain/notification"
	"github.com/wc-fixture/notification/internal/domain/ports"
	"github.com/wc-fixture/shared/pkg/apperrors"
)

// inMemoryWebhookRepo implementa ports.WebhookRepository en memoria.
// Útil para desarrollo local y testing.
// En producción reemplazar por un repositorio PostgreSQL.
type inMemoryWebhookRepo struct {
	mu          sync.RWMutex
	subscribers map[uuid.UUID]notification.WebhookSubscriber // id → subscriber
}

var _ ports.WebhookRepository = (*inMemoryWebhookRepo)(nil)

func newInMemoryWebhookRepo() *inMemoryWebhookRepo {
	return &inMemoryWebhookRepo{
		subscribers: make(map[uuid.UUID]notification.WebhookSubscriber),
	}
}

func (r *inMemoryWebhookRepo) FindByTournament(_ context.Context, tournamentID uuid.UUID) ([]notification.WebhookSubscriber, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []notification.WebhookSubscriber
	for _, s := range r.subscribers {
		if s.Active && s.TournamentID == tournamentID {
			result = append(result, s)
		}
	}
	return result, nil
}

func (r *inMemoryWebhookRepo) Save(_ context.Context, s notification.WebhookSubscriber) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.subscribers[s.ID] = s
	return nil
}

func (r *inMemoryWebhookRepo) Deactivate(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	s, ok := r.subscribers[id]
	if !ok {
		return apperrors.NotFound("webhook subscriber", id.String())
	}
	s.Active = false
	r.subscribers[id] = s
	return nil
}
