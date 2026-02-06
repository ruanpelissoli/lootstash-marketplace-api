package d2

import (
	"fmt"
	"strings"
)

// FilterBuilder builds JSONB queries for D2 item filtering
type FilterBuilder struct {
	conditions []string
	params     []interface{}
	paramIdx   int
}

// NewFilterBuilder creates a new filter builder
func NewFilterBuilder(startIdx int) *FilterBuilder {
	return &FilterBuilder{
		conditions: make([]string, 0),
		params:     make([]interface{}, 0),
		paramIdx:   startIdx,
	}
}

// AddAffixFilter adds a filter for a specific affix
func (f *FilterBuilder) AddAffixFilter(code string, minValue, maxValue *int) {
	// Build JSONB path query
	// stats is an array like: [{"code": "all_skills", "value": 2}, ...]

	if minValue != nil && maxValue != nil {
		f.conditions = append(f.conditions, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM jsonb_array_elements(l.stats) elem WHERE elem->>'code' = $%d AND (elem->>'value')::int >= $%d AND (elem->>'value')::int <= $%d)",
			f.paramIdx, f.paramIdx+1, f.paramIdx+2,
		))
		f.params = append(f.params, code, *minValue, *maxValue)
		f.paramIdx += 3
	} else if minValue != nil {
		f.conditions = append(f.conditions, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM jsonb_array_elements(l.stats) elem WHERE elem->>'code' = $%d AND (elem->>'value')::int >= $%d)",
			f.paramIdx, f.paramIdx+1,
		))
		f.params = append(f.params, code, *minValue)
		f.paramIdx += 2
	} else if maxValue != nil {
		f.conditions = append(f.conditions, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM jsonb_array_elements(l.stats) elem WHERE elem->>'code' = $%d AND (elem->>'value')::int <= $%d)",
			f.paramIdx, f.paramIdx+1,
		))
		f.params = append(f.params, code, *maxValue)
		f.paramIdx += 2
	} else {
		// Just check if the affix exists
		f.conditions = append(f.conditions, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM jsonb_array_elements(l.stats) elem WHERE elem->>'code' = $%d)",
			f.paramIdx,
		))
		f.params = append(f.params, code)
		f.paramIdx++
	}
}

// AddTextSearch adds a text search condition
func (f *FilterBuilder) AddTextSearch(query string) {
	pattern := "%" + strings.ToLower(query) + "%"
	f.conditions = append(f.conditions, fmt.Sprintf(
		"LOWER(l.name) LIKE $%d",
		f.paramIdx,
	))
	f.params = append(f.params, pattern)
	f.paramIdx++
}

// AddCategoryFilter adds a category filter
func (f *FilterBuilder) AddCategoryFilter(category string) {
	f.conditions = append(f.conditions, fmt.Sprintf(
		"l.category = $%d",
		f.paramIdx,
	))
	f.params = append(f.params, category)
	f.paramIdx++
}

// AddRarityFilter adds a rarity filter
func (f *FilterBuilder) AddRarityFilter(rarity string) {
	f.conditions = append(f.conditions, fmt.Sprintf(
		"l.rarity = $%d",
		f.paramIdx,
	))
	f.params = append(f.params, rarity)
	f.paramIdx++
}

// Build returns the WHERE clause and parameters
func (f *FilterBuilder) Build() (string, []interface{}) {
	if len(f.conditions) == 0 {
		return "", nil
	}
	return strings.Join(f.conditions, " AND "), f.params
}

// GetNextParamIdx returns the next parameter index
func (f *FilterBuilder) GetNextParamIdx() int {
	return f.paramIdx
}
