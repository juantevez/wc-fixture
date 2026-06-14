package queries

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/wc-fixture/shared/pkg/apperrors"
	"github.com/wc-fixture/shared/pkg/httputil"
	"github.com/wc-fixture/shared/pkg/logger"
)

// MatchFilters agrupa los filtros opcionales para la consulta de partidos.
// Todos los campos son opcionales — si están en zero value no se aplican.
type MatchFilters struct {
	Phase     string    // "GROUP", "ROUND_OF_32", etc. — vacío = todos
	Status    string    // "SCHEDULED", "COMPLETED", etc. — vacío = todos
	GroupName string    // "A"–"L" — vacío = todos los grupos
	VenueID   uuid.UUID // filtrar por estadio — Nil = todos
	DateFrom  time.Time // partidos desde esta fecha (inclusive)
	DateTo    time.Time // partidos hasta esta fecha (inclusive)
	TeamID    uuid.UUID // partidos donde participa el equipo — Nil = todos
}

// MatchReadModel es el puerto de lectura para consultas de partidos.
type MatchReadModel interface {
	ListMatches(ctx context.Context, tournamentID uuid.UUID, filters MatchFilters, page httputil.PageParams) ([]MatchDTO, int, error)
	GetMatch(ctx context.Context, matchID uuid.UUID) (*MatchDTO, error)
}

// ── ListMatches ───────────────────────────────────────────────────────────────

// ListMatchesQuery solicita un listado paginado de partidos con filtros opcionales.
type ListMatchesQuery struct {
	TournamentID uuid.UUID
	Filters      MatchFilters
	Page         httputil.PageParams
}

func (q ListMatchesQuery) validate() error {
	if q.TournamentID == uuid.Nil {
		return apperrors.Validation("tournament_id es requerido")
	}
	if q.Filters.GroupName != "" && !ValidGroupName(q.Filters.GroupName) {
		return apperrors.ValidationF("group_name %q inválido", q.Filters.GroupName)
	}
	if !q.Filters.DateFrom.IsZero() && !q.Filters.DateTo.IsZero() {
		if q.Filters.DateFrom.After(q.Filters.DateTo) {
			return apperrors.Validation("date_from no puede ser posterior a date_to")
		}
	}
	return nil
}

// ListMatchesHandler retorna partidos paginados con filtros opcionales.
type ListMatchesHandler struct {
	readModel MatchReadModel
}

func NewListMatchesHandler(rm MatchReadModel) *ListMatchesHandler {
	return &ListMatchesHandler{readModel: rm}
}

// ListMatchesResult agrupa los partidos y el metadata de paginación.
type ListMatchesResult struct {
	Matches []MatchDTO
	Meta    httputil.PageMeta
}

func (h *ListMatchesHandler) Handle(ctx context.Context, q ListMatchesQuery) (*ListMatchesResult, error) {
	if err := q.validate(); err != nil {
		return nil, err
	}

	matches, total, err := h.readModel.ListMatches(ctx, q.TournamentID, q.Filters, q.Page)
	if err != nil {
		return nil, err
	}

	logger.FromContext(ctx).Debug("partidos listados",
		"tournament_id", q.TournamentID,
		"total", total,
		"page", q.Page.Page,
		"filtros_activos", countActiveFilters(q.Filters),
	)

	return &ListMatchesResult{
		Matches: matches,
		Meta:    httputil.NewPageMeta(q.Page, total),
	}, nil
}

// ── GetMatch ──────────────────────────────────────────────────────────────────

// GetMatchQuery solicita el detalle de un partido por su ID.
type GetMatchQuery struct {
	MatchID uuid.UUID
}

// GetMatchHandler retorna el detalle completo de un partido.
type GetMatchHandler struct {
	readModel MatchReadModel
}

func NewGetMatchHandler(rm MatchReadModel) *GetMatchHandler {
	return &GetMatchHandler{readModel: rm}
}

func (h *GetMatchHandler) Handle(ctx context.Context, q GetMatchQuery) (*MatchDTO, error) {
	if q.MatchID == uuid.Nil {
		return nil, apperrors.Validation("match_id es requerido")
	}

	dto, err := h.readModel.GetMatch(ctx, q.MatchID)
	if err != nil {
		return nil, err
	}

	logger.FromContext(ctx).Debug("partido consultado",
		"match_id", q.MatchID,
		"phase", dto.Phase,
		"status", dto.Status,
	)
	return dto, nil
}

// countActiveFilters cuenta cuántos filtros están activos para logging.
func countActiveFilters(f MatchFilters) int {
	count := 0
	if f.Phase != "" {
		count++
	}
	if f.Status != "" {
		count++
	}
	if f.GroupName != "" {
		count++
	}
	if f.VenueID != uuid.Nil {
		count++
	}
	if !f.DateFrom.IsZero() {
		count++
	}
	if !f.DateTo.IsZero() {
		count++
	}
	if f.TeamID != uuid.Nil {
		count++
	}
	return count
}
