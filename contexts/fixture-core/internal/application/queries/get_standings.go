package queries

import (
	"context"

	"github.com/google/uuid"
	"github.com/wc-fixture/shared/pkg/apperrors"
	"github.com/wc-fixture/shared/pkg/logger"
)

// StandingsReadModel es el puerto de lectura para tablas de posiciones.
type StandingsReadModel interface {
	GetStandings(ctx context.Context, tournamentID uuid.UUID, groupName string) ([]StandingDTO, error)
}

// GetStandingsQuery solicita la tabla de posiciones de un grupo.
type GetStandingsQuery struct {
	TournamentID uuid.UUID
	GroupName    string
}

func (q GetStandingsQuery) validate() error {
	if q.TournamentID == uuid.Nil {
		return apperrors.Validation("tournament_id es requerido")
	}
	if !ValidGroupName(q.GroupName) {
		return apperrors.ValidationF("group_name %q inválido: debe ser A–L", q.GroupName)
	}
	return nil
}

// GetStandingsHandler retorna la tabla de posiciones de un grupo ordenada.
type GetStandingsHandler struct {
	readModel StandingsReadModel
}

func NewGetStandingsHandler(rm StandingsReadModel) *GetStandingsHandler {
	return &GetStandingsHandler{readModel: rm}
}

func (h *GetStandingsHandler) Handle(ctx context.Context, q GetStandingsQuery) ([]StandingDTO, error) {
	if err := q.validate(); err != nil {
		return nil, err
	}

	standings, err := h.readModel.GetStandings(ctx, q.TournamentID, q.GroupName)
	if err != nil {
		return nil, err
	}

	logger.FromContext(ctx).Debug("standings consultados",
		"tournament_id", q.TournamentID,
		"group", q.GroupName,
		"rows", len(standings),
	)
	return standings, nil
}

// ── BestThirds standings ──────────────────────────────────────────────────────

// BestThirdsReadModel es el puerto para consultar los mejores terceros.
type BestThirdsReadModel interface {
	GetBestThirds(ctx context.Context, tournamentID uuid.UUID) ([]BestThirdDTO, error)
}

// BestThirdDTO representa el standing de un mejor tercero clasificado.
type BestThirdDTO struct {
	GroupName  string    `json:"group_name"`
	Rank       int       `json:"rank"`       // posición entre los 8 clasificados (1–8)
	Classified bool      `json:"classified"` // true si está entre los 8 mejores
	StandingDTO
}

// GetBestThirdsQuery solicita el ranking de mejores terceros.
type GetBestThirdsQuery struct {
	TournamentID uuid.UUID
}

// GetBestThirdsHandler retorna el ranking de mejores terceros de todos los grupos.
// Disponible solo cuando la fase de grupos está completa.
type GetBestThirdsHandler struct {
	readModel BestThirdsReadModel
}

func NewGetBestThirdsHandler(rm BestThirdsReadModel) *GetBestThirdsHandler {
	return &GetBestThirdsHandler{readModel: rm}
}

func (h *GetBestThirdsHandler) Handle(ctx context.Context, q GetBestThirdsQuery) ([]BestThirdDTO, error) {
	if q.TournamentID == uuid.Nil {
		return nil, apperrors.Validation("tournament_id es requerido")
	}

	thirds, err := h.readModel.GetBestThirds(ctx, q.TournamentID)
	if err != nil {
		return nil, err
	}

	logger.FromContext(ctx).Debug("mejores terceros consultados",
		"tournament_id", q.TournamentID,
		"count", len(thirds),
	)
	return thirds, nil
}
