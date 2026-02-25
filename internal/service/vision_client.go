package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// VisionClient handles item screenshot parsing via Claude Vision API
type VisionClient struct {
	apiKey     string
	httpClient *http.Client
}

// ParsedItemResult represents a parsed item from a screenshot
type ParsedItemResult struct {
	Name         string          `json:"name,omitempty"`
	ItemType     string          `json:"itemType"`
	Rarity       string          `json:"rarity"`
	Category     string          `json:"category"`
	Stats        json.RawMessage `json:"stats,omitempty"`
	BaseItemCode string          `json:"baseItemCode,omitempty"`
	BaseItemName string          `json:"baseItemName,omitempty"`
	Quantity     int             `json:"quantity"`
	IsStackable  bool            `json:"isStackable"`
}

// VisionIdentification represents a lightweight item identification from Vision AI
type VisionIdentification struct {
	Name     string   `json:"name"`
	TypeHint string   `json:"typeHint"`
	BaseName string   `json:"baseName,omitempty"`
	Sockets  []string `json:"sockets,omitempty"`
	Quantity int      `json:"quantity"`
}

// NewVisionClient creates a new Vision API client
func NewVisionClient(apiKey string) *VisionClient {
	return &VisionClient{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// IsConfigured returns true if the API key is set
func (v *VisionClient) IsConfigured() bool {
	return v.apiKey != ""
}

// IdentifyItem sends an image to Claude Vision API and returns a lightweight identification
// (name, type hint, base name for runewords, sockets, quantity) for catalog lookup.
func (v *VisionClient) IdentifyItem(ctx context.Context, imageData []byte, contentType string) (*VisionIdentification, error) {
	if !v.IsConfigured() {
		return nil, fmt.Errorf("vision client not configured: missing API key")
	}

	base64Image := base64.StdEncoding.EncodeToString(imageData)

	reqBody := map[string]interface{}{
		"model":      "claude-sonnet-4-5-20250929",
		"max_tokens": 512,
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "image",
						"source": map[string]interface{}{
							"type":       "base64",
							"media_type": contentType,
							"data":       base64Image,
						},
					},
					{
						"type": "text",
						"text": `Read the EXACT text from this Diablo II: Resurrected item screenshot.

IMPORTANT: You MUST transcribe the item name EXACTLY as displayed on screen. Do NOT substitute, correct, or replace the name with a different item you think it might be. Many D2 items have short or uncommon names (e.g. "Sling", "Nagelring", "Manald Heal") — always use the literal text shown.

CRITICAL - Item rarity is determined by the NAME TEXT COLOR:
- Gold/tan text = Unique (e.g. "Harlequin Crest", "Nagelring", "Sling")
- Green text = Set (e.g. "Tal Rasha's Horadric Crest")
- Gold text with rune sequence below = Runeword (e.g. "Spirit", "Enigma")
- Yellow text = Rare (random two-word name like "Eagle Mark", "Blood Knot")
- Blue text = Magic (random name like "Fine Small Charm of Vita")
- White/grey text = Normal/Superior base item

CRITICAL - DO NOT CONFUSE "Ethereal" WITH "Eth Rune":
- "Ethereal (Cannot be Repaired)" is a PROPERTY shown in red/grey text on weapons/armor. It is NOT the item name.
- "Eth Rune" is a small rune item with orange/gold "Eth" text. Runes are tiny square items.
- If you see a large weapon/armor with "Ethereal (Cannot be Repaired)" text, the item is NOT an Eth Rune. Read the actual item name at the TOP of the tooltip (e.g. "Giant Thresher", "Colossus Voulge").

Runeword identification: Runewords show the runeword NAME in gold text at the very top (e.g. "Spirit", "Enigma", "Insight"), followed by the base item type below it, then the rune sequence in quotes (e.g. 'TalThulOrtAmn'). The name field MUST be the runeword name, NOT the rune sequence.

For Normal/Superior base items: return just the base item name WITHOUT the "Superior" prefix. For example, if the item says "Superior Giant Thresher", return name as "Giant Thresher" and typeHint as "base".

Return ONLY a JSON object with:
- "name": the EXACT item name as displayed on screen. For unique/set items: the unique/set name shown in gold/green text (e.g. "Harlequin Crest", "Sling", "Nagelring"). For runewords: the runeword name shown in gold at the top (e.g. "Spirit", "Enigma"), NOT the rune sequence. For runes/gems: the full name (e.g. "Ber Rune", "Perfect Amethyst"). For magic/rare items: the full displayed name (e.g. "Fine Small Charm of Vita"). For base items: the base name WITHOUT "Superior" prefix (e.g. "Giant Thresher", not "Superior Giant Thresher").
- "typeHint": one of "unique", "set", "runeword", "rune", "gem", "base", "magic", "rare". MUST match the name text color as described above.
- "baseName": the base item type. Required for runewords (e.g. "Monarch", "Mage Plate"), magic items (e.g. "Small Charm"), and rare items (e.g. "Ring"). Null for unique/set/rune/gem.
- "sockets": if the item has visible socketed runes or gems, list their names (e.g. ["Ber Rune", "Perfect Diamond"]). Empty array if none.
- "quantity": number of items shown (default 1, may be higher for runes/gems shown in stacks)
Return ONLY the JSON object, no other text.`,
					},
				},
			},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", v.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("vision API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vision API returned status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %w", err)
	}

	if len(apiResp.Content) == 0 {
		return nil, fmt.Errorf("empty response from vision API")
	}

	text := extractJSON(apiResp.Content[0].Text)
	var result VisionIdentification
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, fmt.Errorf("failed to parse identification from response: %w", err)
	}

	if result.Quantity <= 0 {
		result.Quantity = 1
	}

	return &result, nil
}

// extractJSON strips markdown code fences and extracts the JSON object from a response string.
func extractJSON(text string) string {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "```") {
		if i := strings.Index(text, "\n"); i >= 0 {
			text = text[i+1:]
		}
		if i := strings.LastIndex(text, "```"); i >= 0 {
			text = text[:i]
		}
		text = strings.TrimSpace(text)
	}
	return text
}

// ParseItemScreenshot sends an image to Claude Vision API and returns parsed item data
func (v *VisionClient) ParseItemScreenshot(ctx context.Context, imageData []byte, contentType string) (*ParsedItemResult, error) {
	if !v.IsConfigured() {
		return nil, fmt.Errorf("vision client not configured: missing API key")
	}

	base64Image := base64.StdEncoding.EncodeToString(imageData)

	// Build the Anthropic Messages API request
	reqBody := map[string]interface{}{
		"model":      "claude-sonnet-4-5-20250929",
		"max_tokens": 1024,
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "image",
						"source": map[string]interface{}{
							"type":       "base64",
							"media_type": contentType,
							"data":       base64Image,
						},
					},
					{
						"type": "text",
						"text": `Read the stats from this Diablo II: Resurrected item screenshot.

IMPORTANT: Only extract the BLUE-colored stat lines (magical properties). Skip ALL white/grey text such as Defense, Durability, Required Level, Required Strength, Chance to Block, item class, etc. Preserve the EXACT order stats appear in the screenshot from top to bottom.

ETHEREAL: If the item displays "Ethereal (Cannot be Repaired)" (shown in red/grey text), include it as the LAST entry in the stats array with: {"code": "ethereal", "displayText": "Ethereal"}

Return a JSON object with:
- "name": the EXACT item name as displayed on screen, or null if not clearly visible
- "itemType": the base item type (e.g. "Broad Sword", "Ring", "Grand Charm", "Monarch")
- "rarity": one of "normal", "superior", "magic", "rare", "unique", "set", "runeword"
- "category": one of "helms", "body armor", "weapons", "shields", "gloves", "boots", "belts", "amulets", "rings", "charms", "jewels", "runes", "gems", "misc"
- "stats": array of ONLY the blue magical property lines, in screenshot order, each as:
  {"code": "stat_code", "value": number, "displayText": "exact blue text line as shown"}
  Use standard D2 stat codes (e.g. "ac%" for Enhanced Defense, "str" for Strength, "res-all" for All Resistances, "fcr" for Faster Cast Rate, "fhr" for Faster Hit Recovery, "skilltab" for skill bonuses)
  If the item is ethereal, append the ethereal stat entry as described above.
- "baseItemCode": the D2 base item code if known (e.g. "uap" for Shako)
- "baseItemName": the base item name (e.g. "War Hat")
- "quantity": number of items shown (default 1, may be higher for runes/gems)
- "isStackable": true if this is a rune, gem, or misc stackable item

Return ONLY the JSON object, no other text.`,
					},
				},
			},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", v.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("vision API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vision API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the Anthropic API response
	var apiResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %w", err)
	}

	if len(apiResp.Content) == 0 {
		return nil, fmt.Errorf("empty response from vision API")
	}

	// Extract JSON from the text response
	text := extractJSON(apiResp.Content[0].Text)
	var result ParsedItemResult
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, fmt.Errorf("failed to parse item data from response: %w", err)
	}

	// Default quantity to 1
	if result.Quantity <= 0 {
		result.Quantity = 1
	}

	return &result, nil
}
