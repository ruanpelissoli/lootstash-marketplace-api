package games

// GameHandler defines the interface for game-specific operations
type GameHandler interface {
	// GetCode returns the game's unique code (e.g., "diablo2")
	GetCode() string

	// GetName returns the game's display name
	GetName() string

	// GetCategories returns the available item categories for this game
	GetCategories() []Category

	// ValidateStats validates item stats for this game
	ValidateStats(stats []byte) error

	// GetRarities returns the valid rarities for this game
	GetRarities() []string

	// GetServiceTypes returns the available service types for this game
	GetServiceTypes() []ServiceType
}

// Category represents an item category
type Category struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// ServiceType represents a type of in-game service
type ServiceType struct {
	Code string `json:"code"`
	Name string `json:"name"`
}
