package venue

import "math"

// GeoPoint representa una coordenada geográfica en el sistema WGS84 (SRID 4326).
// Es el sistema de referencia estándar para GPS y la mayoría de los mapas digitales.
//
// Invariantes:
//   - Lat debe estar en el rango [-90, 90]
//   - Lon debe estar en el rango [-180, 180]
type GeoPoint struct {
	Lat float64 `json:"lat"` // latitud en grados decimales
	Lon float64 `json:"lon"` // longitud en grados decimales
}

// earthRadiusKm es el radio medio de la Tierra en kilómetros (WGS84).
const earthRadiusKm = 6371.0088

// Validate verifica que las coordenadas estén dentro de los rangos válidos.
func (p GeoPoint) Validate() error {
	if p.Lat < -90 || p.Lat > 90 || p.Lon < -180 || p.Lon > 180 {
		return ErrInvalidCoordinates(p.Lat, p.Lon)
	}
	return nil
}

// IsZero reporta si el GeoPoint está en la posición (0,0) — probablemente no inicializado.
func (p GeoPoint) IsZero() bool {
	return p.Lat == 0 && p.Lon == 0
}

// DistanceTo calcula la distancia geodésica entre dos puntos en kilómetros
// usando la fórmula de Haversine. Equivalente a ST_DistanceSphere de PostGIS.
//
// La fórmula de Haversine tiene un error máximo de ~0.5% debido a que
// asume la Tierra como una esfera perfecta. Para distancias entre estadios
// (máx ~5000 km) el error es menor a 25 km — aceptable para logística deportiva.
func (p GeoPoint) DistanceTo(other GeoPoint) float64 {
	lat1 := degreesToRadians(p.Lat)
	lat2 := degreesToRadians(other.Lat)
	dLat := degreesToRadians(other.Lat - p.Lat)
	dLon := degreesToRadians(other.Lon - p.Lon)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKm * c
}

// IsWithinRadius reporta si el punto está dentro de un radio dado en kilómetros.
func (p GeoPoint) IsWithinRadius(center GeoPoint, radiusKm float64) bool {
	return p.DistanceTo(center) <= radiusKm
}

func degreesToRadians(deg float64) float64 {
	return deg * math.Pi / 180
}
