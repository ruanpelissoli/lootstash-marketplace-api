package cache

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
)

// HashFilter builds a deterministic cache key from filter parameters.
// Keys are sorted before hashing to ensure identical filters always produce the same hash.
func HashFilter(params map[string]interface{}) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	ordered := make([]interface{}, 0, len(keys)*2)
	for _, k := range keys {
		ordered = append(ordered, k, params[k])
	}

	data, _ := json.Marshal(ordered)
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum[:16]) // 32-char hex
}
