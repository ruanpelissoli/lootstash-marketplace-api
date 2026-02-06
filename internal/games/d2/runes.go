package d2

import (
	"fmt"
	"strings"
)

// RuneData contains information about a rune
type RuneData struct {
	Code   string
	Name   string
	Number int
}

// RuneCodes maps rune codes to their display information
var RuneCodes = map[string]RuneData{
	"r01": {Code: "r01", Name: "El", Number: 1},
	"r02": {Code: "r02", Name: "Eld", Number: 2},
	"r03": {Code: "r03", Name: "Tir", Number: 3},
	"r04": {Code: "r04", Name: "Nef", Number: 4},
	"r05": {Code: "r05", Name: "Eth", Number: 5},
	"r06": {Code: "r06", Name: "Ith", Number: 6},
	"r07": {Code: "r07", Name: "Tal", Number: 7},
	"r08": {Code: "r08", Name: "Ral", Number: 8},
	"r09": {Code: "r09", Name: "Ort", Number: 9},
	"r10": {Code: "r10", Name: "Thul", Number: 10},
	"r11": {Code: "r11", Name: "Amn", Number: 11},
	"r12": {Code: "r12", Name: "Sol", Number: 12},
	"r13": {Code: "r13", Name: "Shael", Number: 13},
	"r14": {Code: "r14", Name: "Dol", Number: 14},
	"r15": {Code: "r15", Name: "Hel", Number: 15},
	"r16": {Code: "r16", Name: "Io", Number: 16},
	"r17": {Code: "r17", Name: "Lum", Number: 17},
	"r18": {Code: "r18", Name: "Ko", Number: 18},
	"r19": {Code: "r19", Name: "Fal", Number: 19},
	"r20": {Code: "r20", Name: "Lem", Number: 20},
	"r21": {Code: "r21", Name: "Pul", Number: 21},
	"r22": {Code: "r22", Name: "Um", Number: 22},
	"r23": {Code: "r23", Name: "Mal", Number: 23},
	"r24": {Code: "r24", Name: "Ist", Number: 24},
	"r25": {Code: "r25", Name: "Gul", Number: 25},
	"r26": {Code: "r26", Name: "Vex", Number: 26},
	"r27": {Code: "r27", Name: "Ohm", Number: 27},
	"r28": {Code: "r28", Name: "Lo", Number: 28},
	"r29": {Code: "r29", Name: "Sur", Number: 29},
	"r30": {Code: "r30", Name: "Ber", Number: 30},
	"r31": {Code: "r31", Name: "Jah", Number: 31},
	"r32": {Code: "r32", Name: "Cham", Number: 32},
	"r33": {Code: "r33", Name: "Zod", Number: 33},
}

// GetRuneImageURL returns the Supabase storage URL for a rune image
func GetRuneImageURL(code string) string {
	rune, ok := RuneCodes[code]
	if !ok {
		return ""
	}
	// Format: lowercase rune name for the image filename
	return fmt.Sprintf("http://127.0.0.1:54321/storage/v1/object/public/d2-items/d2/rune/%s.png",
		strings.ToLower(rune.Name))
}

// GetRuneName returns the display name for a rune code
func GetRuneName(code string) string {
	if rune, ok := RuneCodes[code]; ok {
		return rune.Name
	}
	return code
}
