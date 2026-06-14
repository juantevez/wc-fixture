package queries

import (
	"context"

	"github.com/wc-fixture/shared/pkg/apperrors"
	"github.com/wc-fixture/shared/pkg/logger"
	"github.com/wc-fixture/team-registry/internal/domain/ports"
	"github.com/wc-fixture/team-registry/internal/domain/team"
)

// ListTeamsQuery solicita todos los equipos con filtros opcionales.
type ListTeamsQuery struct {
	Confederation string // "" = todas | "UEFA" | "CONMEBOL" | etc.
	QualifiedOnly bool   // true = solo los 48 clasificados al WC2026
}

func (q ListTeamsQuery) validate() error {
	if q.Confederation != "" {
		code := team.ConfederationCode(q.Confederation)
		if !code.IsValid() {
			return apperrors.ValidationF(
				"confederation %q inválida: use UEFA, CONMEBOL, CONCACAF, CAF, AFC u OFC",
				q.Confederation,
			)
		}
	}
	return nil
}

// ListTeamsHandler retorna equipos filtrados por confederación y/o clasificación.
type ListTeamsHandler struct {
	repo ports.TeamRepository
}

func NewListTeamsHandler(repo ports.TeamRepository) *ListTeamsHandler {
	return &ListTeamsHandler{repo: repo}
}

func (h *ListTeamsHandler) Handle(ctx context.Context, q ListTeamsQuery) ([]TeamDTO, error) {
	if err := q.validate(); err != nil {
		return nil, err
	}

	teams, err := h.repo.List(ctx, ports.TeamFilters{
		Confederation: team.ConfederationCode(q.Confederation),
		QualifiedOnly: q.QualifiedOnly,
	})
	if err != nil {
		return nil, err
	}

	dtos := make([]TeamDTO, len(teams))
	for i, t := range teams {
		dtos[i] = toTeamDTO(t)
	}

	logger.FromContext(ctx).Debug("equipos listados",
		"count", len(dtos),
		"confederation", q.Confederation,
		"qualified_only", q.QualifiedOnly,
	)
	return dtos, nil
}
