package d2

import "strings"

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

	// Skill tree backward compat (semantic code → old "skilltab-N" format)
	// Amazon
	"amazon-Bow and Crossbow":  {"skilltab-0"},
	"amazon-Passive and Magic": {"skilltab-1"},
	"amazon-Javelin and Spear": {"skilltab-2"},
	// Sorceress
	"sorceress-Fire":      {"skilltab-3"},
	"sorceress-Lightning": {"skilltab-4"},
	"sorceress-Cold":      {"skilltab-5"},
	// Necromancer
	"necromancer-Curses":          {"skilltab-6"},
	"necromancer-Poison and Bone": {"skilltab-7"},
	"necromancer-Summoning":       {"skilltab-8"},
	// Paladin
	"paladin-Combat Skills":    {"skilltab-9"},
	"paladin-Offensive Auras":  {"skilltab-10"},
	"paladin-Defensive Auras":  {"skilltab-11"},
	// Barbarian
	"barbarian-Combat Skills":    {"skilltab-12"},
	"barbarian-Combat Masteries": {"skilltab-13"},
	"barbarian-Warcries":         {"skilltab-14"},
	// Druid
	"druid-Summoning":     {"skilltab-15"},
	"druid-Shape Shifting": {"skilltab-16"},
	"druid-Elemental":     {"skilltab-17"},
	// Assassin
	"assassin-Traps":             {"skilltab-18"},
	"assassin-Shadow Disciplines": {"skilltab-19"},
	"assassin-Martial Arts":      {"skilltab-20"},
	// Warlock
	"warlock-Eldritch": {"skilltab-21"},
	"warlock-Demon":    {"skilltab-22"},
	"warlock-Chaos":    {"skilltab-23"},
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

// skillTabMapping maps display text keywords to semantic stat codes for all 24 D2 skill tabs.
// Each entry contains keywords that must ALL appear in the display text for a match,
// enabling disambiguation of overlapping names (e.g. Combat Skills → Paladin vs Barbarian).
var skillTabMapping = []struct {
	code     string
	keywords []string
}{
	// Amazon
	{"amazon-Bow and Crossbow", []string{"bow", "crossbow"}},
	{"amazon-Passive and Magic", []string{"passive", "magic"}},
	{"amazon-Javelin and Spear", []string{"javelin", "spear"}},
	// Sorceress
	{"sorceress-Fire", []string{"fire", "sorceress"}},
	{"sorceress-Lightning", []string{"lightning", "sorceress"}},
	{"sorceress-Cold", []string{"cold", "sorceress"}},
	// Necromancer
	{"necromancer-Curses", []string{"curses"}},
	{"necromancer-Poison and Bone", []string{"poison", "bone"}},
	{"necromancer-Summoning", []string{"summoning", "necromancer"}},
	// Paladin
	{"paladin-Combat Skills", []string{"combat", "paladin"}},
	{"paladin-Offensive Auras", []string{"offensive", "auras"}},
	{"paladin-Defensive Auras", []string{"defensive", "auras"}},
	// Barbarian (Combat Masteries must precede Combat Skills to avoid false match on "combat"+"barbarian")
	{"barbarian-Combat Masteries", []string{"masteries"}},
	{"barbarian-Combat Skills", []string{"combat", "barbarian"}},
	{"barbarian-Warcries", []string{"warcries"}},
	// Druid
	{"druid-Summoning", []string{"summoning", "druid"}},
	{"druid-Shape Shifting", []string{"shape", "shifting"}},
	{"druid-Elemental", []string{"elemental"}},
	// Assassin
	{"assassin-Traps", []string{"traps"}},
	{"assassin-Shadow Disciplines", []string{"shadow", "disciplines"}},
	{"assassin-Martial Arts", []string{"martial", "arts"}},
	// Warlock
	{"warlock-Eldritch", []string{"eldritch"}},
	{"warlock-Demon", []string{"demon"}},
	{"warlock-Chaos", []string{"chaos"}},
}

// InferSemanticStatCode maps a "skilltab" stat's displayText to its semantic code.
// Returns empty string if no match found.
// Example: "skilltab" + "+3 to Lightning Skills (Sorceress Only)" → "sorceress-Lightning"
func InferSemanticStatCode(code string, displayText string) string {
	if code != "skilltab" {
		return ""
	}
	lower := strings.ToLower(displayText)
	for _, entry := range skillTabMapping {
		allMatch := true
		for _, kw := range entry.keywords {
			if !strings.Contains(lower, kw) {
				allMatch = false
				break
			}
		}
		if allMatch {
			return entry.code
		}
	}
	return ""
}

