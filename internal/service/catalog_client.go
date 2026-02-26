package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ruanpelissoli/lootstash-marketplace-api/internal/api/dto"
)

// CatalogClient handles item lookups against the catalog API
type CatalogClient struct {
	baseURL    string
	httpClient *http.Client
}

// CatalogSearchResponse represents the catalog search API response
type CatalogSearchResponse struct {
	Items      []CatalogSearchResult `json:"items"`
	TotalCount int                   `json:"totalCount"`
}

// CatalogSearchResult represents a single search result from the catalog
type CatalogSearchResult struct {
	ID          json.Number `json:"id"`
	Name        string      `json:"name"`
	Type        string      `json:"type"`        // "Unique", "Set", "Runeword", "Rune", "Gem", "Base"
	Category    string      `json:"category"`
	Subcategory []string    `json:"subcategory"`
	ImageURL    string      `json:"imageUrl"`
	BaseName    string      `json:"baseName"`
}

// Catalog detail parsing types (internal)

type catalogAffix struct {
	Name     string `json:"name"`
	Code     string `json:"code"`
	HasRange bool   `json:"hasRange"`
	MinValue *int   `json:"minValue,omitempty"`
	MaxValue *int   `json:"maxValue,omitempty"`
}

type catalogBaseInfo struct {
	Code        string   `json:"code"`
	Name        string   `json:"name"`
	Category    string   `json:"category"`
	Subcategory []string `json:"subcategory"`
	ItemType    string   `json:"itemType"`
}

type catalogRuneInfo struct {
	ID       int    `json:"id"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	ImageURL string `json:"imageUrl"`
}

// Type-specific detail structs

type catalogUniqueDetail struct {
	ID          int              `json:"id"`
	Name        string           `json:"name"`
	Rarity      string           `json:"rarity"`
	Category    string           `json:"category"`
	Subcategory []string         `json:"subcategory"`
	Base        *catalogBaseInfo `json:"base"`
	Affixes     []catalogAffix   `json:"affixes"`
	ImageURL    string           `json:"imageUrl"`
}

type catalogSetDetail struct {
	ID           int              `json:"id"`
	Name         string           `json:"name"`
	SetName      string           `json:"setName"`
	Rarity       string           `json:"rarity"`
	Category     string           `json:"category"`
	Subcategory  []string         `json:"subcategory"`
	Base         *catalogBaseInfo `json:"base"`
	Affixes      []catalogAffix   `json:"affixes"`
	BonusAffixes []catalogAffix   `json:"bonusAffixes"`
	ImageURL     string           `json:"imageUrl"`
}

type catalogRunewordDetail struct {
	ID          int               `json:"id"`
	Name        string            `json:"name"`
	DisplayName string            `json:"displayName"`
	Rarity      string            `json:"rarity"`
	Category    string            `json:"category"`
	Subcategory []string          `json:"subcategory"`
	Runes       []catalogRuneInfo `json:"runes"`
	RuneOrder   string            `json:"runeOrder"`
	Affixes     []catalogAffix    `json:"affixes"`
	ImageURL    string            `json:"imageUrl"`
}

type catalogRuneDetail struct {
	ID       int    `json:"id"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	ImageURL string `json:"imageUrl"`
}

type catalogGemDetail struct {
	ID       int    `json:"id"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	GemType  string `json:"gemType"`
	Quality  string `json:"quality"`
	ImageURL string `json:"imageUrl"`
}

type catalogBaseDetail struct {
	Code        string   `json:"code"`
	Name        string   `json:"name"`
	Category    string   `json:"category"`
	Subcategory []string `json:"subcategory"`
	ItemType    string   `json:"itemType"`
	ImageURL    string   `json:"imageUrl"`
}

// NewCatalogClient creates a new catalog API client
func NewCatalogClient(baseURL string) *CatalogClient {
	return &CatalogClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// IsConfigured returns true if the base URL is set
func (c *CatalogClient) IsConfigured() bool {
	return c.baseURL != ""
}

// Search queries the catalog API for items matching the given query
func (c *CatalogClient) Search(ctx context.Context, query string) (*CatalogSearchResponse, error) {
	u := fmt.Sprintf("%s/api/v1/d2/items/search?q=%s&limit=5", c.baseURL, url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create catalog search request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("catalog search request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read catalog search response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("catalog search returned status %d: %s", resp.StatusCode, string(body))
	}

	var result CatalogSearchResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse catalog search response: %w", err)
	}

	return &result, nil
}

// SearchBases queries the catalog API for base items matching the given query
func (c *CatalogClient) SearchBases(ctx context.Context, query string) (*CatalogSearchResponse, error) {
	u := fmt.Sprintf("%s/api/v1/d2/items/search?q=%s&limit=5&type=base", c.baseURL, url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create catalog search request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("catalog search request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read catalog search response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("catalog search returned status %d: %s", resp.StatusCode, string(body))
	}

	var result CatalogSearchResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse catalog search response: %w", err)
	}

	return &result, nil
}

// GetItemDetail fetches the full detail for an item from the catalog
func (c *CatalogClient) GetItemDetail(ctx context.Context, itemType string, id string) (json.RawMessage, error) {
	u := fmt.Sprintf("%s/api/v1/d2/items/%s/%s", c.baseURL, url.PathEscape(itemType), url.PathEscape(id))

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create catalog detail request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("catalog detail request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read catalog detail response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("catalog detail returned status %d: %s", resp.StatusCode, string(body))
	}

	return json.RawMessage(body), nil
}

// mapDetailToParsedItem converts a catalog detail response into a ParsedStashItem
func (c *CatalogClient) mapDetailToParsedItem(itemType string, detail json.RawMessage, searchResult CatalogSearchResult) (*dto.ParsedStashItem, error) {
	// The catalog detail API returns a wrapper: {"itemType": "unique", "unique": {...}}
	var wrapper map[string]json.RawMessage
	if err := json.Unmarshal(detail, &wrapper); err != nil {
		return nil, fmt.Errorf("failed to parse detail wrapper: %w", err)
	}

	// Extract the inner detail object
	innerKey := strings.ToLower(itemType)
	innerData, ok := wrapper[innerKey]
	if !ok {
		// Fallback: try the full detail as-is (some endpoints may not wrap)
		innerData = detail
	}

	parsed := &dto.ParsedStashItem{
		CatalogItemID: searchResult.ID.String(),
		CatalogType:   strings.ToLower(itemType),
	}

	switch strings.ToLower(itemType) {
	case "unique":
		return c.mapUniqueDetail(innerData, parsed)
	case "set":
		return c.mapSetDetail(innerData, parsed)
	case "runeword":
		return c.mapRunewordDetail(innerData, parsed)
	case "rune":
		return c.mapRuneDetail(innerData, parsed)
	case "gem":
		return c.mapGemDetail(innerData, parsed)
	case "base":
		return c.mapBaseDetail(innerData, parsed)
	default:
		return nil, fmt.Errorf("unsupported catalog item type: %s", itemType)
	}
}

func (c *CatalogClient) mapUniqueDetail(data json.RawMessage, parsed *dto.ParsedStashItem) (*dto.ParsedStashItem, error) {
	var d catalogUniqueDetail
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, fmt.Errorf("failed to parse unique detail: %w", err)
	}

	parsed.Name = d.Name
	parsed.Rarity = strings.ToLower(d.Rarity)
	if parsed.Rarity == "" {
		parsed.Rarity = "unique"
	}
	parsed.Category = resolveCategory(d.Category, d.Subcategory)
	parsed.ImageURL = d.ImageURL
	parsed.Stats = affixesToStats(d.Affixes)
	parsed.Quantity = 1

	if d.Base != nil {
		parsed.ItemType = d.Base.Name
		parsed.BaseItemCode = d.Base.Code
		parsed.BaseItemName = d.Base.Name
	}

	return parsed, nil
}

func (c *CatalogClient) mapSetDetail(data json.RawMessage, parsed *dto.ParsedStashItem) (*dto.ParsedStashItem, error) {
	var d catalogSetDetail
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, fmt.Errorf("failed to parse set detail: %w", err)
	}

	parsed.Name = d.Name
	parsed.Rarity = strings.ToLower(d.Rarity)
	if parsed.Rarity == "" {
		parsed.Rarity = "set"
	}
	parsed.Category = resolveCategory(d.Category, d.Subcategory)
	parsed.ImageURL = d.ImageURL
	parsed.Stats = affixesToStats(d.Affixes)
	parsed.Quantity = 1

	if len(d.BonusAffixes) > 0 {
		parsed.Suffixes = affixesToStats(d.BonusAffixes)
	}

	if d.Base != nil {
		parsed.ItemType = d.Base.Name
		parsed.BaseItemCode = d.Base.Code
		parsed.BaseItemName = d.Base.Name
	}

	return parsed, nil
}

func (c *CatalogClient) mapRunewordDetail(data json.RawMessage, parsed *dto.ParsedStashItem) (*dto.ParsedStashItem, error) {
	var d catalogRunewordDetail
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, fmt.Errorf("failed to parse runeword detail: %w", err)
	}

	parsed.Name = d.Name
	parsed.Rarity = "runeword"
	parsed.Category = resolveCategory(d.Category, d.Subcategory)
	parsed.ImageURL = d.ImageURL
	parsed.Stats = affixesToStats(d.Affixes)
	parsed.RuneOrder = d.RuneOrder
	parsed.Quantity = 1

	// Map runes to RuneInfo format
	if len(d.Runes) > 0 {
		runes := make([]dto.RuneInfo, len(d.Runes))
		for i, r := range d.Runes {
			runes[i] = dto.RuneInfo{
				Code:     r.Code,
				Name:     r.Name,
				ImageURL: r.ImageURL,
			}
		}
		runesJSON, err := json.Marshal(runes)
		if err == nil {
			parsed.Runes = runesJSON
		}
	}

	return parsed, nil
}

func (c *CatalogClient) mapRuneDetail(data json.RawMessage, parsed *dto.ParsedStashItem) (*dto.ParsedStashItem, error) {
	var d catalogRuneDetail
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, fmt.Errorf("failed to parse rune detail: %w", err)
	}

	parsed.Name = d.Name
	parsed.ItemType = d.Name
	parsed.Rarity = "normal"
	parsed.Category = "runes"
	parsed.ImageURL = d.ImageURL
	parsed.IsStackable = true
	parsed.Quantity = 1

	return parsed, nil
}

func (c *CatalogClient) mapGemDetail(data json.RawMessage, parsed *dto.ParsedStashItem) (*dto.ParsedStashItem, error) {
	var d catalogGemDetail
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, fmt.Errorf("failed to parse gem detail: %w", err)
	}

	parsed.Name = d.Name
	parsed.ItemType = d.Name
	parsed.Rarity = "normal"
	parsed.Category = "gems"
	parsed.ImageURL = d.ImageURL
	parsed.IsStackable = true
	parsed.Quantity = 1

	return parsed, nil
}

func (c *CatalogClient) mapBaseDetail(data json.RawMessage, parsed *dto.ParsedStashItem) (*dto.ParsedStashItem, error) {
	var d catalogBaseDetail
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, fmt.Errorf("failed to parse base detail: %w", err)
	}

	parsed.Name = d.Name
	parsed.ItemType = d.Name
	parsed.Rarity = "normal"
	parsed.Category = resolveCategory(d.Category, d.Subcategory)
	parsed.ImageURL = d.ImageURL
	parsed.BaseItemCode = d.Code
	parsed.BaseItemName = d.Name
	parsed.Quantity = 1

	return parsed, nil
}

// catalogStatEntry is the stat format written into ParsedStashItem.Stats
type catalogStatEntry struct {
	Code        string `json:"code"`
	Value       *int   `json:"value,omitempty"`
	DisplayText string `json:"displayText"`
	IsVariable  bool   `json:"isVariable"`
	Min         *int   `json:"min,omitempty"`
	Max         *int   `json:"max,omitempty"`
}

// affixesToStats converts catalog affixes to the listing-compatible stats JSON format
func affixesToStats(affixes []catalogAffix) json.RawMessage {
	if len(affixes) == 0 {
		return nil
	}

	stats := make([]catalogStatEntry, len(affixes))
	for i, a := range affixes {
		stats[i] = catalogStatEntry{
			Code:        a.Code,
			DisplayText: a.Name,
			IsVariable:  a.HasRange,
		}
		if a.HasRange {
			stats[i].Min = a.MinValue
			stats[i].Max = a.MaxValue
		}
	}

	data, err := json.Marshal(stats)
	if err != nil {
		return nil
	}
	return data
}

// resolveCategory maps catalog category/subcategory to marketplace category
// Uses subcategory[0] when available (maps directly), falls back to lowercase category
func resolveCategory(category string, subcategory []string) string {
	if len(subcategory) > 0 && subcategory[0] != "" {
		return strings.ToLower(subcategory[0])
	}
	return strings.ToLower(category)
}
