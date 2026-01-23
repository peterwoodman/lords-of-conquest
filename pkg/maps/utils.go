package maps

import "strconv"

// TerritoryIDToString converts a territory ID to a string.
func TerritoryIDToString(id int) string {
	return "t" + strconv.Itoa(id)
}

// StringToTerritoryID converts a string back to a territory ID.
func StringToTerritoryID(s string) int {
	if len(s) > 1 && s[0] == 't' {
		id, _ := strconv.Atoi(s[1:])
		return id
	}
	return 0
}

// WaterIDToString converts a water body ID (negative) to a string.
func WaterIDToString(id int) string {
	return "w" + strconv.Itoa(-id)
}

// StringToWaterID converts a string back to a water body ID.
func StringToWaterID(s string) int {
	if len(s) > 1 && s[0] == 'w' {
		id, _ := strconv.Atoi(s[1:])
		return -id
	}
	return 0
}

