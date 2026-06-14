package queries

// get_distance_matrix.go
//
// GetDistanceMatrixHandler y GetDistanceMatrixQuery están definidos en
// get_distance.go junto con GetDistanceHandler y GetNearbyVenuesHandler,
// dado que los tres comparten el mismo repositorio y DTOs de distancia.
//
// Exports disponibles desde get_distance.go:
//   - DistanceMatrixEntryDTO       → entrada de la matriz (from, to, km)
//   - GetDistanceMatrixQuery       → query vacío (no requiere parámetros)
//   - GetDistanceMatrixHandler     → handler con Handle(ctx, query)
//   - NewGetDistanceMatrixHandler(repo)
