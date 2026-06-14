package venue

import "github.com/google/uuid"

// Surface es el tipo de superficie del campo de juego.
type Surface string

const (
	SurfaceNaturalGrass Surface = "natural_grass"
	SurfaceSynthetic    Surface = "synthetic"
)

// Country representa los tres países sede del Mundial 2026.
type Country string

const (
	CountryUSA    Country = "USA"
	CountryCanada Country = "CAN"
	CountryMexico Country = "MEX"
)

// Venue representa un estadio sede del torneo.
// Es la entidad raíz del bounded context venue-geo.
//
// Los 16 estadios del Mundial 2026 están distribuidos entre:
//   - EE.UU.: 11 estadios (MetLife, AT&T, SoFi, etc.)
//   - Canadá: 2 estadios (BC Place, BMO Field)
//   - México: 3 estadios (Azteca, Akron, BBVA)
type Venue struct {
	ID          uuid.UUID
	Name        string
	City        string
	Country     Country
	CountryCode string  // ISO 3166-1 alpha-3
	Capacity    int
	Surface     Surface
	Location    GeoPoint
	Timezone    string // IANA timezone: "America/New_York", "America/Mexico_City"
	AltitudeM   int    // metros sobre el nivel del mar (relevante para Ciudad de México ~2240m)
}

// Validate verifica las invariantes del venue.
func (v Venue) Validate() error {
	if v.ID == uuid.Nil {
		return &DomainError{Code: ErrCodeVenueNotFound, Message: "venue sin ID"}
	}
	if v.Name == "" {
		return &DomainError{Code: ErrCodeVenueNotFound, Message: "venue sin nombre"}
	}
	if v.Capacity <= 0 {
		return &DomainError{Code: ErrCodeInvalidCoordinates, Message: "capacidad debe ser mayor a 0"}
	}
	return v.Location.Validate()
}

// IsInCountry reporta si el venue está en el país dado.
func (v Venue) IsInCountry(country Country) bool {
	return v.Country == country
}

// DistanceTo calcula la distancia en km a otro venue usando Haversine.
func (v Venue) DistanceTo(other Venue) float64 {
	return v.Location.DistanceTo(other.Location)
}

// IsWithinRadius reporta si el venue está dentro de un radio en km de una coordenada.
func (v Venue) IsWithinRadius(center GeoPoint, radiusKm float64) bool {
	return v.Location.IsWithinRadius(center, radiusKm)
}

// NearbyVenue es un venue enriquecido con la distancia calculada al punto de búsqueda.
// Se usa exclusivamente en resultados de FindNearby.
type NearbyVenue struct {
	Venue
	DistanceKm float64
}
