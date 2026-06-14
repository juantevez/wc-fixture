package fixture

import (
	"time"

	"github.com/google/uuid"
	sharedevents "github.com/wc-fixture/shared/pkg/events"
)

// TournamentStatus representa el estado global del torneo.
type TournamentStatus string

const (
	StatusDraft       TournamentStatus = "DRAFT"
	StatusGroupStage  TournamentStatus = "GROUP_STAGE"
	StatusKnockout    TournamentStatus = "KNOCKOUT"
	StatusFinished    TournamentStatus = "FINISHED"
)

const (
	totalGroups        = 12
	classifiedPerGroup = 2  // 1° y 2° de cada grupo → 24 equipos
	bestThirdsCount    = 8  // mejores terceros → 8 equipos
	totalKnockoutTeams = 32 // 24 + 8
	roundOf32Matches   = 16
)

// Fixture es el aggregate root del bounded context fixture-core.
// Gestiona el estado completo del torneo: grupos, partidos, clasificación
// y bracket eliminatorio. Es la única entidad que puede mutar el estado
// del torneo — toda modificación pasa por sus métodos de comando.
//
// Patrón de eventos: cada comando que modifica el estado acumula eventos
// de dominio en el slice pendingEvents. El repositorio los persiste y el
// command handler los publica a NATS JetStream.
type Fixture struct {
	ID           uuid.UUID
	TournamentID uuid.UUID
	Edition      int
	Name         string
	Status       TournamentStatus

	Groups         []Group  // 12 grupos A–L
	KnockoutRounds []Round  // rondas eliminatorias

	BestThirdsPolicy BestThirdsPolicy

	// Optimistic locking: se incrementa en cada comando.
	Version int64

	// Eventos acumulados durante el comando actual.
	// El repositorio los drena y persiste; el command handler los publica.
	pendingEvents []sharedevents.DomainEvent
}

// Round representa una ronda del bracket eliminatorio.
type Round struct {
	Phase   MatchPhase
	Matches []Match
}

// ── Comandos del aggregate ────────────────────────────────────────────────────

// RegisterMatchResult registra el resultado de un partido.
// Si el partido pertenece a un grupo, actualiza la tabla de posiciones.
// Si el grupo queda completo, verifica si toda la fase de grupos terminó.
// Si la fase de grupos terminó, genera el bracket eliminatorio automáticamente.
//
// Este es el comando más ejecutado del sistema — ocurre 104 veces por torneo.
func (f *Fixture) RegisterMatchResult(matchID uuid.UUID, result MatchResult) error {
	if f.Status == StatusFinished {
		return &DomainError{Code: ErrCodeTournamentAlreadyEnded, Message: "el torneo ya finalizó"}
	}

	// Buscar en fase de grupos
	if group := f.findGroupByMatch(matchID); group != nil {
		if err := group.applyResult(matchID, result); err != nil {
			return err
		}
		f.appendEvent(EventMatchResultRegistered, f.TournamentID, buildResultPayload(matchID, group.Name, result))

		if group.IsComplete() {
			f.appendEvent(EventGroupStageCompleted, f.TournamentID, buildGroupCompletedPayload(group))
		}

		if f.allGroupsComplete() {
			if err := f.generateKnockoutBracket(); err != nil {
				return err
			}
			f.Status = StatusKnockout
		}
		f.Version++
		return nil
	}

	// Buscar en fase eliminatoria
	match, round := f.findKnockoutMatch(matchID)
	if match == nil {
		return errMatchNotFound(matchID.String())
	}
	if err := match.validateResult(result); err != nil {
		return err
	}

	match.Result = &result
	match.Status = MatchStatusCompleted
	f.appendEvent(EventMatchResultRegistered, f.TournamentID, buildResultPayload(matchID, "", result))

	if err := f.advanceKnockoutBracket(match, round, result); err != nil {
		return err
	}

	f.Version++
	return nil
}

// UpdateMatchSchedule modifica el horario y/o venue de un partido programado.
func (f *Fixture) UpdateMatchSchedule(matchID uuid.UUID, newScheduledAt time.Time, newVenueID uuid.UUID) error {
	match := f.findAnyMatch(matchID)
	if match == nil {
		return errMatchNotFound(matchID.String())
	}
	if match.IsCompleted() {
		return errMatchAlreadyCompleted(matchID.String())
	}

	payload := MatchScheduleUpdatedPayload{
		MatchID:        matchID,
		OldScheduledAt: match.ScheduledAt,
		NewScheduledAt: newScheduledAt,
		OldVenueID:     match.VenueID,
		NewVenueID:     newVenueID,
		UpdatedAt:      time.Now().UTC(),
	}

	match.ScheduledAt = newScheduledAt
	match.VenueID = newVenueID
	f.appendEvent(EventMatchScheduleUpdated, f.TournamentID, payload)
	f.Version++
	return nil
}

// ── Consultas del aggregate ───────────────────────────────────────────────────

// GroupByName retorna el grupo por su nombre (A–L).
func (f *Fixture) GroupByName(name string) (*Group, bool) {
	for i := range f.Groups {
		if f.Groups[i].Name == name {
			return &f.Groups[i], true
		}
	}
	return nil, false
}

// AllGroupsCompleted reporta si todos los grupos han finalizado.
func (f *Fixture) AllGroupsCompleted() bool {
	return f.allGroupsComplete()
}

// BestThirdCandidates retorna los standings de todos los terceros clasificados.
// Solo válido cuando AllGroupsCompleted() == true.
func (f *Fixture) BestThirdCandidates() []BestThirdCandidate {
	candidates := make([]BestThirdCandidate, 0, totalGroups)
	for _, g := range f.Groups {
		if s, ok := g.ThirdPlace(); ok {
			candidates = append(candidates, BestThirdCandidate{
				GroupStanding: s,
				GroupName:     g.Name,
			})
		}
	}
	return candidates
}

// PendingEvents retorna los eventos acumulados y limpia el slice interno.
// Llamado por el repositorio después de persistir el aggregate.
func (f *Fixture) PendingEvents() []sharedevents.DomainEvent {
	evts := f.pendingEvents
	f.pendingEvents = nil
	return evts
}

// ── Lógica interna ────────────────────────────────────────────────────────────

func (f *Fixture) allGroupsComplete() bool {
	if len(f.Groups) < totalGroups {
		return false
	}
	for _, g := range f.Groups {
		if g.Status != GroupStatusCompleted {
			return false
		}
	}
	return true
}

// generateKnockoutBracket construye los 16 partidos de octavos de final
// resolviendo los slots según los clasificados de cada grupo y los mejores terceros.
func (f *Fixture) generateKnockoutBracket() error {
	candidates := f.BestThirdCandidates()
	bestThirds := f.BestThirdsPolicy.Classify(candidates)

	// Resolver clasificados de cada grupo
	classified := f.resolveGroupClassified()

	// Construir los 16 partidos de octavos según el cuadro FIFA 2026
	matches := buildRoundOf32(classified, bestThirds)

	f.KnockoutRounds = []Round{
		{Phase: PhaseRoundOf32, Matches: matches},
		{Phase: PhaseQuarterfinal, Matches: buildEmptyRound(PhaseQuarterfinal, 8, 49+16)},
		{Phase: PhaseSemifinal, Matches: buildEmptyRound(PhaseSemifinal, 4, 49+16+8)},
		{Phase: PhaseThirdPlace, Matches: buildEmptyRound(PhaseThirdPlace, 1, 49+16+8+4)},
		{Phase: PhaseFinal, Matches: buildEmptyRound(PhaseFinal, 1, 49+16+8+4+1)},
	}

	matchSnaps := make([]MatchSnap, len(matches))
	for i, m := range matches {
		matchSnaps[i] = toMatchSnap(m)
	}

	bestThirdIDs := make([]uuid.UUID, len(bestThirds))
	copy(bestThirdIDs, bestThirds)

	f.appendEvent(EventKnockoutBracketGenerated, f.TournamentID, KnockoutBracketGeneratedPayload{
		TournamentID:       f.TournamentID,
		BestThirdsSelected: bestThirdIDs,
		RoundOf32Matches:   matchSnaps,
		GeneratedAt:        time.Now().UTC(),
	})

	return nil
}

// advanceKnockoutBracket resuelve el slot del siguiente partido
// tras un resultado eliminatorio.
func (f *Fixture) advanceKnockoutBracket(completedMatch *Match, completedRound *Round, result MatchResult) error {
	winner := result.Winner()
	loser := result.Loser()

	nextMatch := f.findNextMatch(completedMatch)

	if nextMatch != nil {
		// Resolver el slot correspondiente en el siguiente partido
		if completedMatch.ParentHomeMatchID != nil && *completedMatch.ParentHomeMatchID == completedMatch.ID {
			nextMatch.HomeSlot = nextMatch.HomeSlot.Resolve(winner)
		} else {
			nextMatch.AwaySlot = nextMatch.AwaySlot.Resolve(winner)
		}
	}

	// Tercer puesto: el perdedor de semifinal
	if completedRound != nil && completedRound.Phase == PhaseSemifinal {
		f.assignThirdPlaceSlot(completedMatch.ID, loser)
	}

	// Verificar si es la final → torneo terminado
	if completedRound != nil && completedRound.Phase == PhaseFinal {
		f.Status = StatusFinished
		f.appendEvent(EventTournamentFinished, f.TournamentID, TournamentFinishedPayload{
			TournamentID: f.TournamentID,
			ChampionID:   winner,
			RunnerUpID:   loser,
			FinalMatchID: completedMatch.ID,
			FinishedAt:   time.Now().UTC(),
		})
	}

	var nextMatchID *uuid.UUID
	if nextMatch != nil {
		id := nextMatch.ID
		nextMatchID = &id
	}
	f.appendEvent(EventKnockoutMatchAdvanced, f.TournamentID, KnockoutMatchAdvancedPayload{
		CompletedMatchID: completedMatch.ID,
		WinnerTeamID:     winner,
		LoserTeamID:      loser,
		NextMatchID:      nextMatchID,
		Phase:            completedRound.Phase,
		AdvancedAt:       time.Now().UTC(),
	})

	return nil
}

// ── Helpers de búsqueda ───────────────────────────────────────────────────────

func (f *Fixture) findGroupByMatch(matchID uuid.UUID) *Group {
	for i := range f.Groups {
		if _, ok := f.Groups[i].findMatch(matchID); ok {
			return &f.Groups[i]
		}
	}
	return nil
}

func (f *Fixture) findKnockoutMatch(matchID uuid.UUID) (*Match, *Round) {
	for i := range f.KnockoutRounds {
		for j := range f.KnockoutRounds[i].Matches {
			if f.KnockoutRounds[i].Matches[j].ID == matchID {
				return &f.KnockoutRounds[i].Matches[j], &f.KnockoutRounds[i]
			}
		}
	}
	return nil, nil
}

func (f *Fixture) findAnyMatch(matchID uuid.UUID) *Match {
	if g := f.findGroupByMatch(matchID); g != nil {
		m, _ := g.findMatch(matchID)
		return m
	}
	m, _ := f.findKnockoutMatch(matchID)
	return m
}

func (f *Fixture) findNextMatch(completed *Match) *Match {
	for i := range f.KnockoutRounds {
		for j := range f.KnockoutRounds[i].Matches {
			m := &f.KnockoutRounds[i].Matches[j]
			if (m.ParentHomeMatchID != nil && *m.ParentHomeMatchID == completed.ID) ||
				(m.ParentAwayMatchID != nil && *m.ParentAwayMatchID == completed.ID) {
				return m
			}
		}
	}
	return nil
}

func (f *Fixture) assignThirdPlaceSlot(semifinalMatchID uuid.UUID, loserID uuid.UUID) {
	for i := range f.KnockoutRounds {
		if f.KnockoutRounds[i].Phase != PhaseThirdPlace {
			continue
		}
		for j := range f.KnockoutRounds[i].Matches {
			m := &f.KnockoutRounds[i].Matches[j]
			if m.ParentHomeMatchID != nil && *m.ParentHomeMatchID == semifinalMatchID {
				m.HomeSlot = m.HomeSlot.Resolve(loserID)
			} else if m.ParentAwayMatchID != nil && *m.ParentAwayMatchID == semifinalMatchID {
				m.AwaySlot = m.AwaySlot.Resolve(loserID)
			}
		}
	}
}

// resolveGroupClassified retorna un mapa groupName → [firstID, secondID].
func (f *Fixture) resolveGroupClassified() map[string][2]uuid.UUID {
	result := make(map[string][2]uuid.UUID, totalGroups)
	for _, g := range f.Groups {
		first, second := g.ClassifiedTeams()
		result[g.Name] = [2]uuid.UUID{first, second}
	}
	return result
}

// ── Acumulación de eventos ────────────────────────────────────────────────────

func (f *Fixture) appendEvent(eventType string, aggregateID uuid.UUID, payload any) {
	evt, err := sharedevents.New(eventType, aggregateID, "Fixture", payload)
	if err != nil {
		// En producción esto no debería ocurrir — el payload siempre es serializable.
		// Si ocurre es un bug en el código del aggregate.
		panic("fixture: error serializando evento de dominio: " + err.Error())
	}
	f.pendingEvents = append(f.pendingEvents, evt)
}

// ── Helpers de construcción del bracket ──────────────────────────────────────

// buildRoundOf32 construye los 16 partidos de octavos según el cuadro oficial
// FIFA 2026. La numeración sigue el formato M49–M64 (M1–M48 son fase de grupos).
//
// Cuadro simplificado — la asignación exacta de mejores terceros por combinación
// de grupos se determina post-sorteo y puede variar. Esta implementación usa
// la distribución estándar publicada por FIFA para la edición 2026.
func buildRoundOf32(classified map[string][2]uuid.UUID, bestThirds []uuid.UUID) []Match {
	now := time.Now().UTC()
	matches := make([]Match, roundOf32Matches)

	// Asignaciones directas de primeros y segundos por grupo
	// según el cuadro oficial FIFA 2026
	type slotDef struct {
		home MatchSlot
		away MatchSlot
	}

	defs := []slotDef{
		{SlotForTeam(classified["A"][0]), SlotForTeam(classified["B"][1])},  // M49: 1A vs 2B
		{SlotForTeam(classified["C"][0]), SlotForTeam(classified["D"][1])},  // M50: 1C vs 2D
		{SlotForTeam(classified["E"][0]), SlotForTeam(classified["F"][1])},  // M51: 1E vs 2F
		{SlotForTeam(classified["G"][0]), SlotForTeam(classified["H"][1])},  // M52: 1G vs 2H
		{SlotForTeam(classified["I"][0]), SlotForTeam(classified["J"][1])},  // M53: 1I vs 2J
		{SlotForTeam(classified["K"][0]), SlotForTeam(classified["L"][1])},  // M54: 1K vs 2L
		{SlotForTeam(classified["B"][0]), SlotForTeam(classified["A"][1])},  // M55: 1B vs 2A
		{SlotForTeam(classified["D"][0]), SlotForTeam(classified["C"][1])},  // M56: 1D vs 2C
		// Mejores terceros — asignación según grupos de origen
		{SlotForTeam(classified["F"][0]), SlotForTeam(bestThirdsOrNil(bestThirds, 0))}, // M57
		{SlotForTeam(classified["H"][0]), SlotForTeam(bestThirdsOrNil(bestThirds, 1))}, // M58
		{SlotForTeam(classified["J"][0]), SlotForTeam(bestThirdsOrNil(bestThirds, 2))}, // M59
		{SlotForTeam(classified["L"][0]), SlotForTeam(bestThirdsOrNil(bestThirds, 3))}, // M60
		{SlotForTeam(classified["E"][1]), SlotForTeam(bestThirdsOrNil(bestThirds, 4))}, // M61
		{SlotForTeam(classified["G"][1]), SlotForTeam(bestThirdsOrNil(bestThirds, 5))}, // M62
		{SlotForTeam(classified["I"][1]), SlotForTeam(bestThirdsOrNil(bestThirds, 6))}, // M63
		{SlotForTeam(classified["K"][1]), SlotForTeam(bestThirdsOrNil(bestThirds, 7))}, // M64
	}

	for i, def := range defs {
		matches[i] = Match{
			ID:          uuid.New(),
			Phase:       PhaseRoundOf32,
			MatchNumber: 49 + i,
			HomeSlot:    def.home,
			AwaySlot:    def.away,
			Status:      MatchStatusScheduled,
			ScheduledAt: now, // el schedule real lo asigna el command handler
		}
	}
	return matches
}

func bestThirdsOrNil(bestThirds []uuid.UUID, i int) uuid.UUID {
	if i < len(bestThirds) {
		return bestThirds[i]
	}
	return uuid.Nil
}

// buildEmptyRound crea partidos vacíos (slots no resueltos) para las rondas
// posteriores del bracket. Los slots se resuelven a medida que avanzan los clasificados.
func buildEmptyRound(phase MatchPhase, count, startNumber int) []Match {
	matches := make([]Match, count)
	for i := range count {
		matches[i] = Match{
			ID:          uuid.New(),
			Phase:       phase,
			MatchNumber: startNumber + i,
			HomeSlot:    MatchSlot{Kind: SlotKindWinnerOf},
			AwaySlot:    MatchSlot{Kind: SlotKindWinnerOf},
			Status:      MatchStatusScheduled,
		}
	}
	return matches
}

// ── Helpers de construcción de payloads ──────────────────────────────────────

func buildResultPayload(matchID uuid.UUID, groupName string, result MatchResult) MatchResultRegisteredPayload {
	winner := result.Winner()
	var winnerPtr *uuid.UUID
	if winner != uuid.Nil {
		winnerPtr = &winner
	}
	return MatchResultRegisteredPayload{
		MatchID:      matchID,
		GroupName:    groupName,
		HomeTeamID:   result.HomeTeamID,
		AwayTeamID:   result.AwayTeamID,
		HomeGoals:    result.HomeGoals,
		AwayGoals:    result.AwayGoals,
		HomeGoalsET:  result.HomeGoalsET,
		AwayGoalsET:  result.AwayGoalsET,
		HomeGoalsPen: result.HomeGoalsPen,
		AwayGoalsPen: result.AwayGoalsPen,
		WinnerTeamID: winnerPtr,
		CompletedAt:  result.CompletedAt,
	}
}

func buildGroupCompletedPayload(g *Group) GroupStageCompletedPayload {
	snaps := make([]StandingSnap, len(g.Standings))
	for i, s := range g.Standings {
		snaps[i] = toStandingSnap(s)
	}
	return GroupStageCompletedPayload{
		GroupName:   g.Name,
		Standings:   snaps,
		CompletedAt: time.Now().UTC(),
	}
}
