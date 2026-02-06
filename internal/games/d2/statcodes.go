package d2

// statCodeAliases maps canonical (user-friendly) codes to all game data variants.
// This allows filtering to work with either the simplified frontend codes or
// the raw game data codes from catalog-api.
var statCodeAliases = map[string][]string{
	// Magic Find / Gold Find
	"mf": {"mag%"},
	"gf": {"gold%"},

	// Speed stats (multiple variants per type)
	"fcr": {"cast1", "cast2", "cast3"},
	"ias": {"swing1", "swing2", "swing3"},
	"fhr": {"balance1", "balance2", "balance3"},
	"frw": {"move1", "move2", "move3"},

	// Damage/Attack
	"ed": {"dmg%"},
	"ar": {"att", "att%"},

	// Resistances
	"fire_res":   {"res-fire"},
	"cold_res":   {"res-cold"},
	"light_res":  {"res-ltng"},
	"poison_res": {"res-pois"},
	"all_res":    {"res-all"},

	// Leech
	"life_steal": {"lifesteal"},
	"mana_steal": {"manasteal"},

	// Combat
	"crushing_blow": {"crush"},
	"deadly_strike": {"deadly"},
	"open_wounds":   {"openwounds"},
}

// reverseAliases maps game codes back to canonical codes
var reverseAliases map[string]string

func init() {
	reverseAliases = make(map[string]string)
	for canonical, gameCodes := range statCodeAliases {
		for _, gc := range gameCodes {
			reverseAliases[gc] = canonical
		}
	}
}

// ExpandStatCode returns all codes to search for (canonical + game codes).
// This allows filtering to match listings regardless of which code system was used.
func ExpandStatCode(code string) []string {
	// If it's a canonical code, return it plus all game aliases
	if gameCodes, ok := statCodeAliases[code]; ok {
		result := make([]string, 0, len(gameCodes)+1)
		result = append(result, code)
		result = append(result, gameCodes...)
		return result
	}

	// If it's a game code, return it plus its canonical code
	if canonical, ok := reverseAliases[code]; ok {
		return []string{code, canonical}
	}

	// Unknown code, return as-is
	return []string{code}
}

// NormalizeStatCode converts any code variant to its canonical form.
// If the code is unknown, it returns the code as-is.
func NormalizeStatCode(code string) string {
	if _, ok := statCodeAliases[code]; ok {
		return code // Already canonical
	}
	if canonical, ok := reverseAliases[code]; ok {
		return canonical
	}
	return code // Unknown, return as-is
}

// skillTabMappings maps semantic skill tree codes to skilltab param values.
// Items store skill tree bonuses as {code: "skilltab", param: "N", value: X}
// where N is the tab number (0-20). This mapping allows filtering by
// user-friendly codes like "sor-fire" instead of raw param values.
var skillTabMappings = map[string]string{
	// Amazon
	"ama-bow":     "0",
	"ama-passive": "1",
	"ama-javelin": "2",
	// Sorceress
	"sor-fire":      "3",
	"sor-lightning": "4",
	"sor-cold":      "5",
	// Necromancer
	"nec-curses":     "6",
	"nec-poisonbone": "7",
	"nec-summon":     "8",
	// Paladin
	"pal-combat":    "9",
	"pal-offensive": "10",
	"pal-defensive": "11",
	// Barbarian
	"bar-combat":    "12",
	"bar-masteries": "13",
	"bar-warcries":  "14",
	// Druid
	"dru-summon":       "15",
	"dru-shapeshifting": "16",
	"dru-elemental":    "17",
	// Assassin
	"ass-traps":   "18",
	"ass-shadow":  "19",
	"ass-martial": "20",
}

// GetSkillTabParam returns the skilltab param value if code is a skill tree code,
// or empty string if it's not a skill tree code.
func GetSkillTabParam(code string) string {
	return skillTabMappings[code]
}
