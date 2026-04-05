package domain

// ComponentFilter specifies optional criteria for FindComponents.
// Zero values mean the field is not applied as a filter.
type ComponentFilter struct {
	Category     *Category // exact match
	Manufacturer string    // case-insensitive partial match
	MPN          string    // case-insensitive partial match
	Package      string    // exact match
	Text         string    // case-insensitive partial match over MPN, manufacturer, and description
}
