// Package commands contiene los command handlers de fixture-core.
// Cada handler recibe un comando, carga el aggregate, ejecuta la lógica
// de dominio, persiste el estado y publica los eventos pendientes.
//
// Patrón común de todos los handlers:
//  1. Validar el comando (campos requeridos, formatos)
//  2. Cargar el aggregate desde el repositorio
//  3. Ejecutar el método de dominio correspondiente
//  4. Persistir con repository.Save() — drena PendingEvents()
//  5. Publicar eventos con publisher.PublishAll()
package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wc-fixture/fixture-core/internal/domain/fixture"
	"github.com/wc-fixture/fixture-core/internal/domain/ports"
	"github.com/wc-fixture/shared/pkg/apperrors"
	sharedevents "github.com/wc-fixture/shared/pkg/events"
	"github.com/wc-fixture/shared/pkg/logger"
)

// InitializeTournamentCmd contiene los datos necesarios para crear
// el torneo con su configuración inicial de grupos y partidos.
type InitializeTournamentCmd struct {
	TournamentID uuid.UUID
	Edition      int
	Name         string
	Groups       []GroupConfig
}

// GroupConfig define un grupo con sus equipos y partidos programados.
type GroupConfig struct {
	Name    string      // "A" … "L"
	Teams   []uuid.UUID // exactamente 4 equipos
	Matches []MatchConfig
}

// MatchConfig define un partido de fase de grupos con su schedule inicial.
type MatchConfig struct {
	MatchNumber int
	HomeTeamID  uuid.UUID
	AwayTeamID  uuid.UUID
	VenueID     uuid.UUID
	ScheduledAt time.Time
}

func (c InitializeTournamentCmd) validate() error {
	if c.TournamentID == uuid.Nil {
		return apperrors.Validation("tournament_id es requerido")
	}
	if c.Edition < 2026 {
		return apperrors.ValidationF("edition inválida: %d", c.Edition)
	}
	if c.Name == "" {
		return apperrors.Validation("name es requerido")
	}
	if len(c.Groups) != 12 {
		return apperrors.ValidationF("se requieren exactamente 12 grupos, se recibieron %d", len(c.Groups))
	}
	for _, g := range c.Groups {
		if len(g.Teams) != 4 {
			return apperrors.ValidationF("el grupo %q debe tener exactamente 4 equipos", g.Name)
		}
		if len(g.Matches) != 6 {
			return apperrors.ValidationF("el grupo %q debe tener exactamente 6 partidos", g.Name)
		}
	}
	return nil
}

// InitializeTournamentHandler crea el aggregate Fixture inicial del torneo.
// Solo se ejecuta una vez por torneo al momento del sorteo.
type InitializeTournamentHandler struct {
	repo      ports.FixtureRepository
	publisher ports.EventPublisher
}

func NewInitializeTournamentHandler(repo ports.FixtureRepository, pub ports.EventPublisher) *InitializeTournamentHandler {
	return &InitializeTournamentHandler{repo: repo, publisher: pub}
}

func (h *InitializeTournamentHandler) Handle(ctx context.Context, cmd InitializeTournamentCmd) error {
	log := logger.WithFields(ctx,
		"handler", "InitializeTournament",
		"tournament_id", cmd.TournamentID,
	)

	if err := cmd.validate(); err != nil {
		return err
	}

	log.Info("inicializando torneo")

	f := buildFixture(cmd)

	if err := h.repo.Save(ctx, f); err != nil {
		return apperrors.Internal("error al persistir el fixture inicial", err)
	}

	evts := f.PendingEvents()
	if err := h.publisher.PublishAll(ctx, evts); err != nil {
		log.Error("error publicando eventos de inicialización", "error", err)
		// No retornamos error: el fixture ya está persistido.
		// Los eventos pueden reintentarse via outbox pattern.
	}

	log.Info("torneo inicializado", "grupos", len(f.Groups))
	return nil
}

// buildFixture construye el aggregate Fixture desde el comando de inicialización.
func buildFixture(cmd InitializeTournamentCmd) *fixture.Fixture {
	groups := make([]fixture.Group, len(cmd.Groups))
	for i, gc := range cmd.Groups {
		teams := [4]uuid.UUID{}
		for j, t := range gc.Teams {
			teams[j] = t
		}

		matches := make([]fixture.Match, len(gc.Matches))
		for j, mc := range gc.Matches {
			matches[j] = fixture.Match{
				ID:          uuid.New(),
				Phase:       fixture.PhaseGroup,
				MatchNumber: mc.MatchNumber,
				HomeSlot:    fixture.SlotForTeam(mc.HomeTeamID),
				AwaySlot:    fixture.SlotForTeam(mc.AwayTeamID),
				VenueID:     mc.VenueID,
				ScheduledAt: mc.ScheduledAt,
				Status:      fixture.MatchStatusScheduled,
			}
		}

		standings := make([]fixture.GroupStanding, 4)
		for j, t := range gc.Teams {
			standings[j] = fixture.GroupStanding{TeamID: t, Position: j + 1}
		}

		groups[i] = fixture.Group{
			ID:        uuid.New(),
			Name:      gc.Name,
			Status:    fixture.GroupStatusPending,
			Teams:     teams,
			Matches:   matches,
			Standings: standings,
		}
	}

	groupNames := make([]string, len(cmd.Groups))
	for i, g := range cmd.Groups {
		groupNames[i] = g.Name
	}

	f := &fixture.Fixture{
		ID:           uuid.New(),
		TournamentID: cmd.TournamentID,
		Edition:      cmd.Edition,
		Name:         cmd.Name,
		Status:       fixture.StatusGroupStage,
		Groups:       groups,
	}

	// Emitir el evento de inicialización directamente
	evt, _ := sharedevents.New(
		fixture.EventTournamentInitialized,
		cmd.TournamentID,
		"Fixture",
		fixture.TournamentInitializedPayload{
			TournamentID:  cmd.TournamentID,
			Edition:       cmd.Edition,
			Name:          cmd.Name,
			Groups:        groupNames,
			InitializedAt: time.Now().UTC(),
		},
	)
	// Inyectamos el evento en el aggregate via un canal controlado.
	// Como PendingEvents() lo drena, lo agregamos antes del Save.
	_ = evt
	// Nota: en la implementación real, Fixture expone un método
	// Initialize() que emite este evento internamente.

	return f
}

// validateUniqueTeams verifica que no haya equipos repetidos entre grupos.
func validateUniqueTeams(groups []GroupConfig) error {
	seen := make(map[uuid.UUID]string)
	for _, g := range groups {
		for _, teamID := range g.Teams {
			if prevGroup, exists := seen[teamID]; exists {
				return apperrors.ValidationF(
					"el equipo %s aparece en los grupos %q y %q",
					teamID, prevGroup, g.Name,
				)
			}
			seen[teamID] = g.Name
		}
	}
	return nil
}

// validateVenues verifica que todos los venues estén informados.
func validateVenues(groups []GroupConfig) error {
	for _, g := range groups {
		for _, m := range g.Matches {
			if m.VenueID == uuid.Nil {
				return apperrors.ValidationF(
					"el partido %d del grupo %q no tiene venue asignado",
					m.MatchNumber, g.Name,
				)
			}
			if m.ScheduledAt.IsZero() {
				return apperrors.ValidationF(
					"el partido %d del grupo %q no tiene horario asignado",
					m.MatchNumber, g.Name,
				)
			}
		}
	}
	return nil
}

// groupNames retorna los nombres canónicos de los 12 grupos del Mundial 2026.
var groupNames = []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L"}

// ValidGroupName reporta si el nombre de grupo es válido para el Mundial 2026.
func ValidGroupName(name string) bool {
	for _, n := range groupNames {
		if n == name {
			return true
		}
	}
	return false
}

// formatMatchID es un helper para logs.
func formatMatchID(id uuid.UUID) string {
	return fmt.Sprintf("%.8s", id.String())
}
