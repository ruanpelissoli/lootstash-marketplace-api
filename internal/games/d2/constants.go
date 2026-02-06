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

// Rarities for Diablo 2 items
var Rarities = []string{
	"normal",
	"magic",
	"rare",
	"unique",
	"set",
	"runeword",
}
