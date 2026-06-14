package queries

import (
	"context"

	"github.com/google/uuid"
	"github.com/wc-fixture/shared/pkg/apperrors"
	"github.com/wc-fixture/shared/pkg/logger"
)

// KnockoutReadModel es el puerto de lectura para el bracket eliminatorio.
type KnockoutReadModel interface {
	// GetKnockout retorna el bracket eliminatorio completo.
	// Retorna apperrors.NotFound si el bracket aún no fue generado
	// (la fase de grupos no está completa).
	GetKnockout(ctx context.Context, tournamentID uuid.UUID) (*KnockoutDTO, error)

	// GetKnockoutRound retorna solo una ronda específica del bracket.
	GetKnockoutRound(ctx context.Context, tournamentID uuid.UUID, phase string) ([]MatchDTO, error)
}

// ── GetKnockout ───────────────────────────────────────────────────────────────

// GetKnockoutQuery solicita el bracket eliminatorio completo.
type GetKnockoutQuery struct {
	TournamentID uuid.UUID
}

// GetKnockoutHandler retorna el bracket eliminatorio completo con todos los slots.
// Los slots no resueltos (SlotKindWinnerOf) se incluyen como referencias dinámicas
// para que el cliente pueda mostrar el bracket anticipado.
type GetKnockoutHandler struct {
	readModel KnockoutReadModel
}

func NewGetKnockoutHandler(rm KnockoutReadModel) *GetKnockoutHandler {
	return &GetKnockoutHandler{readModel: rm}
}

func (h *GetKnockoutHandler) Handle(ctx context.Context, q GetKnockoutQuery) (*KnockoutDTO, error) {
	if q.TournamentID == uuid.Nil {
		return nil, apperrors.Validation("tournament_id es requerido")
	}

	dto, err := h.readModel.GetKnockout(ctx, q.TournamentID)
	if err != nil {
		return nil, err
	}

	log := logger.FromContext(ctx)
	log.Debug("bracket consultado",
		"tournament_id", q.TournamentID,
		"r32_count", len(dto.RoundOf32),
		"qf_count", len(dto.Quarterfinals),
		"sf_count", len(dto.Semifinals),
	)
	return dto, nil
}

// ── GetKnockoutRound ──────────────────────────────────────────────────────────

// GetKnockoutRoundQuery solicita los partidos de una ronda eliminatoria específica.
type GetKnockoutRoundQuery struct {
	TournamentID uuid.UUID
	Phase        string // "ROUND_OF_32", "QUARTERFINAL", "SEMIFINAL", "THIRD_PLACE", "FINAL"
}

func (q GetKnockoutRoundQuery) validate() error {
	if q.TournamentID == uuid.Nil {
		return apperrors.Validation("tournament_id es requerido")
	}
	validPhases := map[string]bool{
		"ROUND_OF_32":  true,
		"QUARTERFINAL": true,
		"SEMIFINAL":    true,
		"THIRD_PLACE":  true,
		"FINAL":        true,
	}
	if !validPhases[q.Phase] {
		return apperrors.ValidationF("phase %q inválida", q.Phase)
	}
	return nil
}

// GetKnockoutRoundHandler retorna los partidos de una ronda eliminatoria.
type GetKnockoutRoundHandler struct {
	readModel KnockoutReadModel
}

func NewGetKnockoutRoundHandler(rm KnockoutReadModel) *GetKnockoutRoundHandler {
	return &GetKnockoutRoundHandler{readModel: rm}
}

func (h *GetKnockoutRoundHandler) Handle(ctx context.Context, q GetKnockoutRoundQuery) ([]MatchDTO, error) {
	if err := q.validate(); err != nil {
		return nil, err
	}

	matches, err := h.readModel.GetKnockoutRound(ctx, q.TournamentID, q.Phase)
	if err != nil {
		return nil, err
	}

	logger.FromContext(ctx).Debug("ronda eliminatoria consultada",
		"tournament_id", q.TournamentID,
		"phase", q.Phase,
		"partidos", len(matches),
	)
	return matches, nil
}
