package queries

import (
	"context"

	"github.com/google/uuid"
	"github.com/wc-fixture/shared/pkg/apperrors"
	"github.com/wc-fixture/shared/pkg/logger"
)

// GroupReadModel es el puerto de lectura para consultas de grupos.
type GroupReadModel interface {
	GetGroup(ctx context.Context, tournamentID uuid.UUID, groupName string) (*GroupDTO, error)
	ListGroups(ctx context.Context, tournamentID uuid.UUID) ([]GroupDTO, error)
}

// ── GetGroup ──────────────────────────────────────────────────────────────────

// GetGroupQuery solicita el detalle de un grupo específico (A–L).
type GetGroupQuery struct {
	TournamentID uuid.UUID
	GroupName    string
}

func (q GetGroupQuery) validate() error {
	if q.TournamentID == uuid.Nil {
		return apperrors.Validation("tournament_id es requerido")
	}
	if q.GroupName == "" {
		return apperrors.Validation("group_name es requerido")
	}
	if !ValidGroupName(q.GroupName) {
		return apperrors.ValidationF("group_name %q inválido: debe ser A–L", q.GroupName)
	}
	return nil
}

// GetGroupHandler retorna el detalle completo de un grupo.
type GetGroupHandler struct {
	readModel GroupReadModel
}

func NewGetGroupHandler(rm GroupReadModel) *GetGroupHandler {
	return &GetGroupHandler{readModel: rm}
}

func (h *GetGroupHandler) Handle(ctx context.Context, q GetGroupQuery) (*GroupDTO, error) {
	if err := q.validate(); err != nil {
		return nil, err
	}

	dto, err := h.readModel.GetGroup(ctx, q.TournamentID, q.GroupName)
	if err != nil {
		return nil, err
	}

	logger.FromContext(ctx).Debug("grupo consultado",
		"tournament_id", q.TournamentID,
		"group", q.GroupName,
		"status", dto.Status,
	)
	return dto, nil
}

// ── ListGroups ────────────────────────────────────────────────────────────────

// ListGroupsQuery solicita todos los grupos del torneo.
type ListGroupsQuery struct {
	TournamentID uuid.UUID
}

// ListGroupsHandler retorna los 12 grupos con sus standings y partidos.
type ListGroupsHandler struct {
	readModel GroupReadModel
}

func NewListGroupsHandler(rm GroupReadModel) *ListGroupsHandler {
	return &ListGroupsHandler{readModel: rm}
}

func (h *ListGroupsHandler) Handle(ctx context.Context, q ListGroupsQuery) ([]GroupDTO, error) {
	if q.TournamentID == uuid.Nil {
		return nil, apperrors.Validation("tournament_id es requerido")
	}

	groups, err := h.readModel.ListGroups(ctx, q.TournamentID)
	if err != nil {
		return nil, err
	}

	logger.FromContext(ctx).Debug("grupos listados",
		"tournament_id", q.TournamentID,
		"count", len(groups),
	)
	return groups, nil
}

// ValidGroupName reporta si el nombre de grupo es válido para el Mundial 2026.
func ValidGroupName(name string) bool {
	valid := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L"}
	for _, n := range valid {
		if n == name {
			return true
		}
	}
	return false
}
