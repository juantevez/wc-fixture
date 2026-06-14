package venue

import "github.com/google/uuid"

// VenueDistance representa la distancia precalculada entre dos venues.
// Se persiste en la tabla venue_distances como cache de la matriz completa.
// Es simétrica: distance(A→B) == distance(B→A).
type VenueDistance struct {
	FromVenueID uuid.UUID
	ToVenueID   uuid.UUID
	DistanceKm  float64
}

// DistanceMatrix es la matriz completa de distancias entre todos los venues.
// La clave es [fromID, toID] con fromID < toID para evitar duplicados.
type DistanceMatrix map[[2]uuid.UUID]float64

// Get retorna la distancia entre dos venues en kilómetros.
// Es simétrica: Get(a, b) == Get(b, a).
// Retorna 0 si no existe la entrada (mismo venue o no calculado).
func (m DistanceMatrix) Get(from, to uuid.UUID) float64 {
	key := sortedKey(from, to)
	return m[key]
}

// Set almacena la distancia entre dos venues. Normaliza el orden de los IDs.
func (m DistanceMatrix) Set(from, to uuid.UUID, distanceKm float64) {
	key := sortedKey(from, to)
	m[key] = distanceKm
}

// sortedKey retorna la clave canónica [menor, mayor] para la matriz simétrica.
func sortedKey(a, b uuid.UUID) [2]uuid.UUID {
	if a.String() < b.String() {
		return [2]uuid.UUID{a, b}
	}
	return [2]uuid.UUID{b, a}
}

// BuildDistanceMatrix calcula la matriz completa de distancias entre venues
// usando la fórmula de Haversine implementada en GeoPoint.DistanceTo.
// Se llama una vez al inicializar el torneo y el resultado se persiste.
func BuildDistanceMatrix(venues []Venue) DistanceMatrix {
	matrix := make(DistanceMatrix)
	for i := 0; i < len(venues); i++ {
		for j := i + 1; j < len(venues); j++ {
			dist := venues[i].Location.DistanceTo(venues[j].Location)
			matrix.Set(venues[i].ID, venues[j].ID, dist)
		}
	}
	return matrix
}
