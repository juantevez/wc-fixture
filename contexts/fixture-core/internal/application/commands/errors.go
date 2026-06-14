package commands

import (
	"errors"

	"github.com/wc-fixture/fixture-core/internal/domain/fixture"
	"github.com/wc-fixture/shared/pkg/apperrors"
)

// mapDomainError convierte un *fixture.DomainError al *apperrors.AppError
// correspondiente. Esta conversión ocurre en el layer de aplicación para
// que el dominio no dependa de apperrors y los handlers HTTP reciban
// el tipo correcto para mapear al status HTTP.
func mapDomainError(err error) error {
	var de *fixture.DomainError
	if !errors.As(err, &de) {
		return apperrors.Internal("error inesperado de dominio", err)
	}

	switch de.Code {
	case fixture.ErrCodeMatchNotFound, fixture.ErrCodeGroupNotFound:
		return apperrors.NotFound("recurso", de.Message)

	case fixture.ErrCodeMatchAlreadyCompleted, fixture.ErrCodeDuplicateResult,
		fixture.ErrCodeGroupAlreadyCompleted, fixture.ErrCodeTournamentAlreadyEnded:
		return apperrors.Conflict(de.Message)

	case fixture.ErrCodeInvalidResult, fixture.ErrCodeInvalidPhaseTransition,
		fixture.ErrCodeInvalidTeamCount:
		return apperrors.Validation(de.Message)

	case fixture.ErrCodeSlotNotResolved, fixture.ErrCodeGroupStageNotComplete,
		fixture.ErrCodeTournamentNotStarted, fixture.ErrCodeMatchNotInProgress:
		return apperrors.Conflict(de.Message)

	default:
		return apperrors.Internal("error de dominio no mapeado: "+string(de.Code), err)
	}
}
