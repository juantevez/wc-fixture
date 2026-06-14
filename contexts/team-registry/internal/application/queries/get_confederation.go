package queries

import (
	"context"

	"github.com/wc-fixture/shared/pkg/apperrors"
	"github.com/wc-fixture/shared/pkg/logger"
	"github.com/wc-fixture/team-registry/internal/domain/ports"
	"github.com/wc-fixture/team-registry/internal/domain/team"
)

// GetConfederationQuery solicita el detalle de una confederación por código.
type GetConfederationQuery struct {
	Code string // "UEFA", "CONMEBOL", etc.
}

func (q GetConfederationQuery) validate() error {
	if q.Code == "" {
		return apperrors.Validation("code es requerido")
	}
	if !team.ConfederationCode(q.Code).IsValid() {
		return apperrors.ValidationF("confederation %q inválida", q.Code)
	}
	return nil
}

// GetConfederationHandler retorna el detalle de una confederación.
type GetConfederationHandler struct {
	repo ports.TeamRepository
}

func NewGetConfederationHandler(repo ports.TeamRepository) *GetConfederationHandler {
	return &GetConfederationHandler{repo: repo}
}

func (h *GetConfederationHandler) Handle(ctx context.Context, q GetConfederationQuery) (*ConfederationDTO, error) {
	if err := q.validate(); err != nil {
		return nil, err
	}

	confederations, err := h.repo.ListConfederations(ctx)
	if err != nil {
		return nil, err
	}

	for _, c := range confederations {
		if string(c.Code) == q.Code {
			dto := toConfederationDTO(c)
			return &dto, nil
		}
	}

	return nil, apperrors.NotFound("confederación", q.Code)
}

// ListConfederationsQuery solicita todas las confederaciones FIFA.
type ListConfederationsQuery struct{}

// ListConfederationsHandler retorna las 6 confederaciones FIFA con sus cupos WC2026.
type ListConfederationsHandler struct {
	repo ports.TeamRepository
}

func NewListConfederationsHandler(repo ports.TeamRepository) *ListConfederationsHandler {
	return &ListConfederationsHandler{repo: repo}
}

func (h *ListConfederationsHandler) Handle(ctx context.Context, _ ListConfederationsQuery) ([]ConfederationDTO, error) {
	confederations, err := h.repo.ListConfederations(ctx)
	if err != nil {
		return nil, err
	}

	dtos := make([]ConfederationDTO, len(confederations))
	for i, c := range confederations {
		dtos[i] = toConfederationDTO(c)
	}

	logger.FromContext(ctx).Debug("confederaciones listadas", "count", len(dtos))
	return dtos, nil
}
