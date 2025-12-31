package client

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// drawIconOnCell draws an icon centered on a grid cell
func (s *GameplayScene) drawIconOnCell(screen *ebiten.Image, iconType, param string, cellX, cellY float32) {
	cellSize := float32(s.cellSize)
	iconSize := cellSize * 0.7 // Icon takes up 70% of cell

	// Center the icon in the cell
	offsetX := (cellSize - iconSize) / 2
	offsetY := (cellSize - iconSize) / 2
	x := cellX + offsetX
	y := cellY + offsetY

	// Try to use PNG icon first
	var iconImg *ebiten.Image
	switch iconType {
	case "resource":
		// Map resource name to icon name
		switch param {
		case "Coal":
			iconImg = GetIcon("coal")
		case "Gold":
			iconImg = GetIcon("gold")
		case "Iron":
			iconImg = GetIcon("iron")
		case "Timber":
			iconImg = GetIcon("timber")
		case "Grassland":
			iconImg = GetIcon("grassland")
		}
	case "stockpile":
		iconImg = GetIcon("stockpile")
	case "city":
		iconImg = GetIcon("city")
	case "weapon":
		iconImg = GetIcon("weapon")
	case "horse":
		iconImg = GetIcon("horse")
	case "boat":
		iconImg = GetIcon("boat")
	}

	if iconImg != nil {
		// Draw the PNG icon scaled to fit
		op := &ebiten.DrawImageOptions{}
		imgW := float32(iconImg.Bounds().Dx())
		imgH := float32(iconImg.Bounds().Dy())
		scaleX := iconSize / imgW
		scaleY := iconSize / imgH
		op.GeoM.Scale(float64(scaleX), float64(scaleY))
		op.GeoM.Translate(float64(x), float64(y))
		screen.DrawImage(iconImg, op)
	} else {
		// Fallback to drawing shapes
		s.drawIconFallback(screen, iconType, param, x, y, iconSize)
	}
}

// drawIconFallback draws a fallback shape when PNG icon isn't available
func (s *GameplayScene) drawIconFallback(screen *ebiten.Image, iconType, param string, x, y, size float32) {
	switch iconType {
	case "resource":
		s.drawResourceIconFallback(screen, param, x, y, size)
	case "city":
		s.drawCityIconFallback(screen, x, y, size)
	case "weapon":
		s.drawWeaponIconFallback(screen, x, y, size)
	case "horse":
		s.drawHorseIconFallback(screen, x, y, size)
	case "boat":
		count := 1
		fmt.Sscanf(param, "%d", &count)
		s.drawBoatIconFallback(screen, x, y, size, count)
	case "stockpile":
		s.drawStockpileIconFallback(screen, param, x, y, size)
	}
}

// drawResourceIconFallback draws a resource indicator
func (s *GameplayScene) drawResourceIconFallback(screen *ebiten.Image, resource string, x, y, size float32) {
	var iconColor color.RGBA
	var symbol string

	switch resource {
	case "Coal":
		iconColor = color.RGBA{40, 40, 40, 255}
		symbol = "C"
	case "Gold":
		iconColor = color.RGBA{255, 200, 50, 255}
		symbol = "G"
	case "Iron":
		iconColor = color.RGBA{140, 140, 160, 255}
		symbol = "I"
	case "Timber":
		iconColor = color.RGBA{100, 70, 40, 255}
		symbol = "W"
	case "Grassland":
		iconColor = color.RGBA{120, 180, 80, 255}
		symbol = "G"
	default:
		return
	}

	// Draw diamond shape for resources
	cx, cy := x+size/2, y+size/2
	halfSize := size * 0.6

	// Diamond points
	path := vector.Path{}
	path.MoveTo(cx, cy-halfSize) // Top
	path.LineTo(cx+halfSize, cy) // Right
	path.LineTo(cx, cy+halfSize) // Bottom
	path.LineTo(cx-halfSize, cy) // Left
	path.Close()

	// Fill
	vs, is := path.AppendVerticesAndIndicesForFilling(nil, nil)
	for i := range vs {
		vs[i].ColorR = float32(iconColor.R) / 255
		vs[i].ColorG = float32(iconColor.G) / 255
		vs[i].ColorB = float32(iconColor.B) / 255
		vs[i].ColorA = 1
	}
	screen.DrawTriangles(vs, is, emptyImage, nil)

	// Border
	vector.StrokeLine(screen, cx, cy-halfSize, cx+halfSize, cy, 1, color.RGBA{0, 0, 0, 200}, false)
	vector.StrokeLine(screen, cx+halfSize, cy, cx, cy+halfSize, 1, color.RGBA{0, 0, 0, 200}, false)
	vector.StrokeLine(screen, cx, cy+halfSize, cx-halfSize, cy, 1, color.RGBA{0, 0, 0, 200}, false)
	vector.StrokeLine(screen, cx-halfSize, cy, cx, cy-halfSize, 1, color.RGBA{0, 0, 0, 200}, false)

	_ = symbol // Could draw letter if needed
}

// drawCityIconFallback draws a city building icon
func (s *GameplayScene) drawCityIconFallback(screen *ebiten.Image, x, y, size float32) {
	// Draw a small house/castle shape
	buildingColor := color.RGBA{220, 200, 180, 255}
	roofColor := color.RGBA{150, 80, 60, 255}

	// Building body (rectangle)
	bodyH := size * 0.6
	bodyY := y + size - bodyH
	vector.DrawFilledRect(screen, x+1, bodyY, size-2, bodyH, buildingColor, false)
	vector.StrokeRect(screen, x+1, bodyY, size-2, bodyH, 1, color.RGBA{100, 80, 60, 255}, false)

	// Roof (triangle)
	cx := x + size/2
	path := vector.Path{}
	path.MoveTo(cx, y)         // Top point
	path.LineTo(x+size, bodyY) // Right
	path.LineTo(x, bodyY)      // Left
	path.Close()

	vs, is := path.AppendVerticesAndIndicesForFilling(nil, nil)
	for i := range vs {
		vs[i].ColorR = float32(roofColor.R) / 255
		vs[i].ColorG = float32(roofColor.G) / 255
		vs[i].ColorB = float32(roofColor.B) / 255
		vs[i].ColorA = 1
	}
	screen.DrawTriangles(vs, is, emptyImage, nil)
}

// drawWeaponIconFallback draws a sword/weapon icon
func (s *GameplayScene) drawWeaponIconFallback(screen *ebiten.Image, x, y, size float32) {
	bladeColor := color.RGBA{180, 180, 200, 255}
	hiltColor := color.RGBA{120, 80, 40, 255}

	cx := x + size/2
	cy := y + size/2

	// Blade (vertical line)
	bladeLen := size * 0.7
	vector.StrokeLine(screen, cx, cy-bladeLen/2, cx, cy+bladeLen/3, 2, bladeColor, false)

	// Crossguard (horizontal line)
	guardLen := size * 0.5
	guardY := cy + bladeLen/6
	vector.StrokeLine(screen, cx-guardLen/2, guardY, cx+guardLen/2, guardY, 2, hiltColor, false)

	// Handle
	vector.StrokeLine(screen, cx, guardY, cx, cy+bladeLen/2, 2, hiltColor, false)
}

// drawHorseIconFallback draws a horse icon
func (s *GameplayScene) drawHorseIconFallback(screen *ebiten.Image, x, y, size float32) {
	horseColor := color.RGBA{140, 100, 60, 255}

	// Simplified horse shape: body oval + head + legs
	cx := x + size/2
	cy := y + size/2

	// Body (oval)
	bodyW := size * 0.7
	bodyH := size * 0.4
	vector.DrawFilledCircle(screen, cx, cy, bodyW/2, horseColor, false)

	// Head (smaller circle offset to right)
	headX := cx + bodyW/3
	headY := cy - bodyH/2
	vector.DrawFilledCircle(screen, headX, headY, size*0.2, horseColor, false)

	// Legs (simple lines)
	legColor := color.RGBA{100, 70, 40, 255}
	legY := cy + bodyH/2
	vector.StrokeLine(screen, cx-bodyW/4, cy, cx-bodyW/4, legY+size*0.2, 1.5, legColor, false)
	vector.StrokeLine(screen, cx+bodyW/4, cy, cx+bodyW/4, legY+size*0.2, 1.5, legColor, false)
}

// drawBoatIconFallback draws a boat icon with count
func (s *GameplayScene) drawBoatIconFallback(screen *ebiten.Image, x, y, size float32, count int) {
	boatColor := color.RGBA{100, 80, 60, 255}
	sailColor := color.RGBA{240, 240, 230, 255}

	cx := x + size/2
	cy := y + size/2

	// Hull (curved bottom)
	hullW := size * 0.8
	hullH := size * 0.3
	hullY := cy + size*0.2

	// Simple hull rectangle
	vector.DrawFilledRect(screen, cx-hullW/2, hullY, hullW, hullH, boatColor, false)

	// Sail (triangle)
	mastX := cx
	mastTop := cy - size*0.3
	path := vector.Path{}
	path.MoveTo(mastX, mastTop)
	path.LineTo(mastX+size*0.3, hullY)
	path.LineTo(mastX, hullY)
	path.Close()

	vs, is := path.AppendVerticesAndIndicesForFilling(nil, nil)
	for i := range vs {
		vs[i].ColorR = float32(sailColor.R) / 255
		vs[i].ColorG = float32(sailColor.G) / 255
		vs[i].ColorB = float32(sailColor.B) / 255
		vs[i].ColorA = 1
	}
	screen.DrawTriangles(vs, is, emptyImage, nil)

	// Mast
	vector.StrokeLine(screen, mastX, mastTop, mastX, hullY, 1, boatColor, false)

	// Count indicator if more than 1
	if count > 1 {
		countText := fmt.Sprintf("%d", count)
		DrawTextCentered(screen, countText, int(x+size+2), int(cy-2), color.RGBA{255, 255, 255, 255})
	}
}

// drawStockpileIconFallback draws a stockpile crate icon
func (s *GameplayScene) drawStockpileIconFallback(screen *ebiten.Image, playerID string, x, y, size float32) {
	// Stockpile color (golden brown)
	stockpileColor := color.RGBA{200, 160, 80, 255}
	borderColor := color.RGBA{150, 100, 30, 255}

	// Draw crate
	vector.DrawFilledRect(screen, x, y, size, size, stockpileColor, false)
	vector.StrokeRect(screen, x, y, size, size, 1.5, borderColor, false)

	// Cross-hatching for crate appearance
	vector.StrokeLine(screen, x, y, x+size, y+size, 1, borderColor, false)
	vector.StrokeLine(screen, x+size, y, x, y+size, 1, borderColor, false)
}

// findTerritoryCenter finds the center cell of a territory
func (s *GameplayScene) findTerritoryCenter(territoryID string, grid []interface{}) (int, int) {
	// Extract numeric ID from "t1", "t2", etc.
	if len(territoryID) < 2 || territoryID[0] != 't' {
		return -1, -1
	}

	var numID int
	fmt.Sscanf(territoryID[1:], "%d", &numID)

	width := int(s.mapData["width"].(float64))
	height := int(s.mapData["height"].(float64))

	// Find all cells of this territory and compute average position
	sumX, sumY, count := 0, 0, 0
	for y := 0; y < height; y++ {
		row := grid[y].([]interface{})
		for x := 0; x < width; x++ {
			if int(row[x].(float64)) == numID {
				sumX += x
				sumY += y
				count++
			}
		}
	}

	if count == 0 {
		return -1, -1
	}

	return sumX / count, sumY / count
}
