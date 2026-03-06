package maps

import "strconv"

// TerritoryIDToString converts a territory ID to a string.
func TerritoryIDToString(id int) string {
	return "t" + strconv.Itoa(id)
}

// WaterIDToString converts a water body ID (negative) to a string.
func WaterIDToString(id int) string {
	return "w" + strconv.Itoa(-id)
}

