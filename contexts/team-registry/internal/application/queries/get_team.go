// Package queries contiene los query handlers del bounded context team-registry.
package queries

import (
	"context"

	"github.com/google/uuid"
	"github.com/wc-fixture/shared/pkg/apperrors"
	"github.com/wc-fixture/shared/pkg/logger"
	"github.com/wc-fixture/team-registry/internal/domain/ports"
	"github.com/wc-fixture/team-registry/internal/domain/team"
)

// ── DTOs ──────────────────────────────────────────────────────────────────────

// TeamDTO es la representación serializable de un equipo para la API REST.
type TeamDTO struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	ShortName     string    `json:"short_name"`
	CountryCode   string    `json:"country_code"`
	Confederation string    `json:"confederation"`
	FIFARanking   int       `json:"fifa_ranking"`
	FlagURL       string    `json:"flag_url"`
	Qualified     bool      `json:"qualified"`
}

// ConfederationDTO es la representación serializable de una confederación.
type ConfederationDTO struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	ShortName string `json:"short_name"`
	Region    string `json:"region"`
	WC2026Slots int  `json:"wc2026_slots"`
}

func toTeamDTO(t team.Team) TeamDTO {
	return TeamDTO{
		ID:            t.ID,
		Name:          t.Name,
		ShortName:     t.ShortName,
		CountryCode:   t.CountryCode,
		Confederation: string(t.Confederation),
		FIFARanking:   t.FIFARankingDate,
		FlagURL:       t.FlagURL,
		Qualified:     t.Qualified,
	}
}

func toConfederationDTO(c team.Confederation) ConfederationDTO {
	return ConfederationDTO{
		Code:        string(c.Code),
		Name:        c.Name,
		ShortName:   c.ShortName,
		Region:      c.Region,
		WC2026Slots: c.Code.SlotsInWC2026(),
	}
}

// ── GetTeam ───────────────────────────────────────────────────────────────────

// GetTeamQuery solicita un equipo por su UUID.
type GetTeamQuery struct {
	TeamID uuid.UUID
}

// GetTeamHandler retorna el detalle de un equipo por ID.
type GetTeamHandler struct {
	repo ports.TeamRepository
}

func NewGetTeamHandler(repo ports.TeamRepository) *GetTeamHandler {
	return &GetTeamHandler{repo: repo}
}

func (h *GetTeamHandler) Handle(ctx context.Context, q GetTeamQuery) (*TeamDTO, error) {
	if q.TeamID == uuid.Nil {
		return nil, apperrors.Validation("team_id es requerido")
	}

	t, err := h.repo.GetByID(ctx, q.TeamID)
	if err != nil {
		return nil, err
	}

	dto := toTeamDTO(*t)
	logger.FromContext(ctx).Debug("equipo consultado", "team_id", q.TeamID, "name", t.Name)
	return &dto, nil
}
