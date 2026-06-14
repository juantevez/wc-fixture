package queries

// get_best_thirds.go
//
// El query GetBestThirds y su handler están definidos en get_standings.go
// junto con BestThirdsReadModel y BestThirdDTO, dado que la tabla de mejores
// terceros es una extensión directa de los standings de grupo.
//
// Este archivo existe para mantener la consistencia del scaffold y documenta
// la decisión de co-ubicación.
//
// Exports disponibles desde get_standings.go:
//   - BestThirdDTO         → DTO del mejor tercero con su ranking
//   - BestThirdsReadModel  → puerto de lectura
//   - GetBestThirdsQuery   → query struct
//   - GetBestThirdsHandler → handler con Handle(ctx, query)
//   - NewGetBestThirdsHandler(rm BestThirdsReadModel)
