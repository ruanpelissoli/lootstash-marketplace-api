package d2

import (
	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/games"
)

// Categories for Diablo 2 items
var Categories = []games.Category{
	{Code: "helm", Name: "Helms"},
	{Code: "armor", Name: "Body Armor"},
	{Code: "weapon", Name: "Weapons"},
	{Code: "shield", Name: "Shields"},
	{Code: "gloves", Name: "Gloves"},
	{Code: "boots", Name: "Boots"},
	{Code: "belt", Name: "Belts"},
	{Code: "amulet", Name: "Amulets"},
	{Code: "ring", Name: "Rings"},
	{Code: "charm", Name: "Charms"},
	{Code: "jewel", Name: "Jewels"},
	{Code: "rune", Name: "Runes"},
	{Code: "gem", Name: "Gems"},
	{Code: "misc", Name: "Miscellaneous"},
}

// categoryAliases maps parent category codes to additional stored values
// that should match when filtering by the parent category
var categoryAliases = map[string][]string{
	"misc": {"quest"},
}

// GetSubcategories returns additional category values that should match
// when filtering by the given category code
func GetSubcategories(category string) []string {
	if aliases, ok := categoryAliases[category]; ok {
		return aliases
	}
	return []string{category}
}

// Rarities for Diablo 2 items
var Rarities = []string{
	"normal",
	"magic",
	"rare",
	"unique",
	"set",
	"runeword",
}
