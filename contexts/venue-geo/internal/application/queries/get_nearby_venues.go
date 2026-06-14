package queries

// get_nearby_venues.go
//
// GetNearbyVenuesHandler y GetNearbyVenuesQuery están definidos en
// get_distance.go junto con GetDistanceHandler y GetDistanceMatrixHandler,
// dado que los tres comparten el mismo repositorio y lógica geoespacial.
//
// Exports disponibles desde get_distance.go:
//   - GetNearbyVenuesQuery     → { Center GeoPoint, RadiusKm float64 }
//   - GetNearbyVenuesHandler   → handler con Handle(ctx, query)
//   - NearbyVenueDTO           → VenueDTO + DistanceKm
//   - NewGetNearbyVenuesHandler(repo)
