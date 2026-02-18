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
	"weapon": {
		"swords", "axes", "maces", "hammers", "polearms", "spears", "staves",
		"wands", "scepters", "bows", "crossbows", "javelins", "daggers", "knives",
		"throwing knives", "throwing axes", "claws", "orbs",
		"amazon bows", "amazon javelins", "amazon spears",
	},
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

// ServiceTypes for Diablo 2 in-game services
var ServiceTypes = []games.ServiceType{
	{Code: "rush", Name: "Rush"},
	{Code: "crush", Name: "Crush"},
	{Code: "grush", Name: "Glitch Rush"},
	{Code: "sockets", Name: "Socket Quest"},
	{Code: "waypoints", Name: "Waypoints"},
	{Code: "ubers", Name: "Ubers"},
	{Code: "colossal_ancients", Name: "Colossal Ancients"},
}
