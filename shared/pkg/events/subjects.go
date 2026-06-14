package events

// Subject define el tipo para los subjects de NATS JetStream.
// Usar el tipo en lugar de strings literales evita typos entre productor
// y consumidor.
type Subject string

// Subjects del stream FIXTURE_EVENTS.
// Convención: <dominio>.<aggregate>.<evento_en_snake_case>
const (
	// fixture-core → todos los consumidores
	SubjectMatchResultRegistered    Subject = "fixture.match.result_registered"
	SubjectGroupStageCompleted      Subject = "fixture.group.stage_completed"
	SubjectKnockoutBracketGenerated Subject = "fixture.knockout.bracket_generated"
	SubjectKnockoutMatchAdvanced    Subject = "fixture.knockout.match_advanced"
	SubjectMatchScheduleUpdated     Subject = "fixture.match.schedule_updated"
	SubjectTournamentFinished       Subject = "fixture.tournament.finished"

	// result-ingestion → fixture-core
	SubjectResultIngested Subject = "fixture.result.ingested"
)

// Stream es el nombre del JetStream stream que agrupa todos los subjects.
const Stream = "FIXTURE_EVENTS"

// StreamSubjects es el filtro de subjects que pertenecen al stream.
// Se usa al crear el stream en NATS: nats.stream.subjects = ["fixture.>"]
const StreamSubjectsFilter = "fixture.>"

// String implementa Stringer para logs y debugging.
func (s Subject) String() string { return string(s) }
