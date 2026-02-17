package d2

import (
	"encoding/json"
	"fmt"

	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/games"
)

// Handler implements the GameHandler interface for Diablo 2
type Handler struct{}

// NewHandler creates a new Diablo 2 handler
func NewHandler() *Handler {
	return &Handler{}
}

// GetCode returns the game code
func (h *Handler) GetCode() string {
	return "diablo2"
}

// GetName returns the game display name
func (h *Handler) GetName() string {
	return "Diablo II: Resurrected"
}

// GetCategories returns the available item categories
func (h *Handler) GetCategories() []games.Category {
	return Categories
}

// ValidateStats validates item stats
func (h *Handler) ValidateStats(stats []byte) error {
	if len(stats) == 0 {
		return nil
	}

	var statList []ItemStat
	if err := json.Unmarshal(stats, &statList); err != nil {
		return fmt.Errorf("invalid stats format: %w", err)
	}

	for _, stat := range statList {
		if stat.Code == "" {
			return fmt.Errorf("stat code is required")
		}
		// Note: We don't validate against a fixed list of codes since
		// catalog-api is the source of truth for valid stat codes
	}

	return nil
}

// GetRarities returns valid rarities for D2
func (h *Handler) GetRarities() []string {
	return Rarities
}

// GetServiceTypes returns available service types for D2
func (h *Handler) GetServiceTypes() []games.ServiceType {
	return ServiceTypes
}

// ItemStat represents a single stat on an item
type ItemStat struct {
	Code  string `json:"code"`
	Value int    `json:"value"`
	Min   *int   `json:"min,omitempty"`
	Max   *int   `json:"max,omitempty"`
}

// Register registers the D2 handler with the game registry
func Register(registry *games.Registry) {
	registry.Register(NewHandler())
}
