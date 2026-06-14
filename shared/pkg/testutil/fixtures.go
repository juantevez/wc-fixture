package testutil

import (
	"time"

	"github.com/google/uuid"
)

// UUIDs fijos para tests — evitan generar UUIDs random en cada corrida,
// haciendo los mensajes de error más legibles y reproducibles.
var (
	FixtureID   = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	TournamentID = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	GroupAID    = uuid.MustParse("00000000-0000-0000-0000-000000000010")
	GroupBID    = uuid.MustParse("00000000-0000-0000-0000-000000000011")

	// Equipos de ejemplo
	ArgentinaID = uuid.MustParse("00000000-0000-0000-0001-000000000001")
	BrasilID    = uuid.MustParse("00000000-0000-0000-0001-000000000002")
	FranciaID   = uuid.MustParse("00000000-0000-0000-0001-000000000003")
	AlemaniaID  = uuid.MustParse("00000000-0000-0000-0001-000000000004")

	// Venues de ejemplo
	SoFiStadiumID = uuid.MustParse("00000000-0000-0000-0002-000000000001")
	AztecaID      = uuid.MustParse("00000000-0000-0000-0002-000000000002")
)

// ReferenceTime es una fecha fija para tests que requieren tiempo determinístico.
var ReferenceTime = time.Date(2026, 6, 15, 20, 0, 0, 0, time.UTC)
