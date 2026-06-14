package team

import "github.com/google/uuid"

// Team representa un equipo nacional participante del torneo.
// Es la entidad raíz del bounded context team-registry.
//
// team-registry es un Supporting Domain: provee datos de referencia
// a fixture-core (nombre, código, confederación) pero no contiene
// lógica de clasificación ni resultados — eso pertenece a fixture-core.
type Team struct {
	ID              uuid.UUID
	Name            string            // "Argentina", "France", etc.
	ShortName       string            // "ARG", "FRA" (3 letras FIFA)
	CountryCode     string            // ISO 3166-1 alpha-3
	Confederation   ConfederationCode // "CONMEBOL", "UEFA", etc.
	FIFARankingDate int               // ranking al momento del sorteo (para desempate)
	FlagURL         string            // URL pública de la imagen de la bandera
	Qualified       bool              // true = clasificado al Mundial 2026
}

// Validate verifica las invariantes del equipo.
func (t Team) Validate() error {
	if t.ID == uuid.Nil {
		return &DomainError{Code: ErrCodeTeamNotFound, Message: "equipo sin ID"}
	}
	if t.Name == "" {
		return &DomainError{Code: ErrCodeTeamNotFound, Message: "equipo sin nombre"}
	}
	if len(t.ShortName) != 3 {
		return &DomainError{Code: ErrCodeTeamNotFound, Message: "short_name debe tener exactamente 3 caracteres"}
	}
	if !t.Confederation.IsValid() {
		return ErrConfederationNotFound(string(t.Confederation))
	}
	if t.FIFARankingDate < 1 {
		return ErrInvalidFIFARanking(t.FIFARankingDate)
	}
	return nil
}

// IsFromConfederation reporta si el equipo pertenece a la confederación dada.
func (t Team) IsFromConfederation(code ConfederationCode) bool {
	return t.Confederation == code
}
