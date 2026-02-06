package d2

import (
	"reflect"
	"sort"
	"testing"
)

func TestExpandStatCode(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected []string
	}{
		{
			name:     "canonical code mf returns mf and mag%",
			code:     "mf",
			expected: []string{"mf", "mag%"},
		},
		{
			name:     "game code mag% returns mag% and mf",
			code:     "mag%",
			expected: []string{"mag%", "mf"},
		},
		{
			name:     "canonical code fcr returns fcr and all cast variants",
			code:     "fcr",
			expected: []string{"fcr", "cast1", "cast2", "cast3"},
		},
		{
			name:     "game code cast2 returns cast2 and fcr",
			code:     "cast2",
			expected: []string{"cast2", "fcr"},
		},
		{
			name:     "canonical code ias returns ias and all swing variants",
			code:     "ias",
			expected: []string{"ias", "swing1", "swing2", "swing3"},
		},
		{
			name:     "canonical code fire_res returns fire_res and res-fire",
			code:     "fire_res",
			expected: []string{"fire_res", "res-fire"},
		},
		{
			name:     "game code res-fire returns res-fire and fire_res",
			code:     "res-fire",
			expected: []string{"res-fire", "fire_res"},
		},
		{
			name:     "unknown code returns itself",
			code:     "all_skills",
			expected: []string{"all_skills"},
		},
		{
			name:     "another unknown code returns itself",
			code:     "life",
			expected: []string{"life"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpandStatCode(tt.code)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("ExpandStatCode(%q) = %v, want %v", tt.code, got, tt.expected)
			}
		})
	}
}

func TestNormalizeStatCode(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name:     "canonical code stays canonical",
			code:     "mf",
			expected: "mf",
		},
		{
			name:     "game code mag% normalizes to mf",
			code:     "mag%",
			expected: "mf",
		},
		{
			name:     "game code cast1 normalizes to fcr",
			code:     "cast1",
			expected: "fcr",
		},
		{
			name:     "game code cast2 normalizes to fcr",
			code:     "cast2",
			expected: "fcr",
		},
		{
			name:     "game code swing3 normalizes to ias",
			code:     "swing3",
			expected: "ias",
		},
		{
			name:     "game code res-fire normalizes to fire_res",
			code:     "res-fire",
			expected: "fire_res",
		},
		{
			name:     "unknown code stays as-is",
			code:     "all_skills",
			expected: "all_skills",
		},
		{
			name:     "another unknown code stays as-is",
			code:     "sockets",
			expected: "sockets",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeStatCode(tt.code)
			if got != tt.expected {
				t.Errorf("NormalizeStatCode(%q) = %q, want %q", tt.code, got, tt.expected)
			}
		})
	}
}

func TestAllAliasesAreBidirectional(t *testing.T) {
	// Verify that all game codes in statCodeAliases have a reverse mapping
	for canonical, gameCodes := range statCodeAliases {
		for _, gc := range gameCodes {
			if reverseAliases[gc] != canonical {
				t.Errorf("game code %q should map back to %q, got %q", gc, canonical, reverseAliases[gc])
			}
		}
	}
}

func TestExpandStatCodeCoversAllAliases(t *testing.T) {
	// Test all canonical codes
	for canonical, gameCodes := range statCodeAliases {
		expanded := ExpandStatCode(canonical)
		if expanded[0] != canonical {
			t.Errorf("ExpandStatCode(%q) should start with canonical code, got %v", canonical, expanded)
		}

		// Sort the rest for comparison
		gotRest := expanded[1:]
		sort.Strings(gotRest)
		expectedRest := make([]string, len(gameCodes))
		copy(expectedRest, gameCodes)
		sort.Strings(expectedRest)

		if !reflect.DeepEqual(gotRest, expectedRest) {
			t.Errorf("ExpandStatCode(%q) game codes = %v, want %v", canonical, gotRest, expectedRest)
		}
	}
}
