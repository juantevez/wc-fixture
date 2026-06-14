package team

// Confederation representa una confederación FIFA.
// Los 32 equipos del Mundial 2026 provienen de las 6 confederaciones.
type Confederation struct {
	Code      ConfederationCode
	Name      string
	ShortName string
	Region    string
}

// ConfederationCode es el código oficial de la confederación FIFA.
type ConfederationCode string

const (
	ConfederationUEFA    ConfederationCode = "UEFA"    // Europa
	ConfederationCONMEBOL ConfederationCode = "CONMEBOL" // Sudamérica
	ConfederationCONCACAF ConfederationCode = "CONCACAF" // Norte/Centro América y Caribe
	ConfederationCAF     ConfederationCode = "CAF"     // África
	ConfederationAFC     ConfederationCode = "AFC"     // Asia
	ConfederationOFC     ConfederationCode = "OFC"     // Oceanía
)

// AllConfederations retorna las 6 confederaciones FIFA con sus datos oficiales.
func AllConfederations() []Confederation {
	return []Confederation{
		{
			Code:      ConfederationUEFA,
			Name:      "Union of European Football Associations",
			ShortName: "UEFA",
			Region:    "Europe",
		},
		{
			Code:      ConfederationCONMEBOL,
			Name:      "Confederación Sudamericana de Fútbol",
			ShortName: "CONMEBOL",
			Region:    "South America",
		},
		{
			Code:      ConfederationCONCACAF,
			Name:      "Confederation of North, Central America and Caribbean Association Football",
			ShortName: "CONCACAF",
			Region:    "North/Central America & Caribbean",
		},
		{
			Code:      ConfederationCAF,
			Name:      "Confederation of African Football",
			ShortName: "CAF",
			Region:    "Africa",
		},
		{
			Code:      ConfederationAFC,
			Name:      "Asian Football Confederation",
			ShortName: "AFC",
			Region:    "Asia",
		},
		{
			Code:      ConfederationOFC,
			Name:      "Oceania Football Confederation",
			ShortName: "OFC",
			Region:    "Oceania",
		},
	}
}

// IsValid reporta si el código de confederación es válido.
func (c ConfederationCode) IsValid() bool {
	switch c {
	case ConfederationUEFA, ConfederationCONMEBOL, ConfederationCONCACAF,
		ConfederationCAF, ConfederationAFC, ConfederationOFC:
		return true
	}
	return false
}

// SlotsInWC2026 retorna la cantidad de cupos que tiene la confederación
// en el Mundial 2026 (48 equipos total).
func (c ConfederationCode) SlotsInWC2026() int {
	slots := map[ConfederationCode]int{
		ConfederationUEFA:     16,
		ConfederationCAF:       9,
		ConfederationCONMEBOL:  6,
		ConfederationCONCACAF:  6,
		ConfederationAFC:        8,
		ConfederationOFC:        1,
		// 2 cupos inter-confederaciones (playoff)
	}
	return slots[c]
}
