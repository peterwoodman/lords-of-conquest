package client

import (
	"image"
	"image/color"
	"runtime"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.design/x/clipboard"
)

var clipboardInitialized bool

// InitClipboard initializes the clipboard for paste support.
// Call this once at startup.
func InitClipboard() {
	if err := clipboard.Init(); err == nil {
		clipboardInitialized = true
	}
}

// Colors used in the UI - Retro 8-bit inspired palette
var (
	// Dark blues and purples for backgrounds
	ColorBackground = color.RGBA{15, 15, 35, 255} // Deep space blue
	ColorPanel      = color.RGBA{25, 25, 60, 255} // Rich navy
	ColorPanelLight = color.RGBA{40, 40, 80, 255} // Lighter navy

	// Vibrant accent colors
	ColorPrimary        = color.RGBA{100, 200, 255, 255} // Bright cyan
	ColorPrimaryHover   = color.RGBA{140, 220, 255, 255} // Lighter cyan
	ColorSecondary      = color.RGBA{80, 60, 120, 255}   // Purple
	ColorSecondaryHover = color.RGBA{110, 90, 150, 255}  // Lighter purple
	ColorSuccess        = color.RGBA{100, 255, 100, 255} // Bright green
	ColorDanger         = color.RGBA{255, 80, 120, 255}  // Hot pink
	ColorWarning        = color.RGBA{255, 200, 50, 255}  // Bright yellow

	// Text colors
	ColorText       = color.RGBA{255, 255, 255, 255} // Pure white
	ColorTextMuted  = color.RGBA{160, 180, 220, 255} // Light blue-grey
	ColorTextDim    = color.RGBA{120, 140, 180, 255} // Dimmer blue-grey for details
	ColorTextShadow = color.RGBA{0, 0, 0, 180}       // Text shadow

	// UI elements
	ColorBorder     = color.RGBA{100, 200, 255, 255} // Bright cyan border
	ColorBorderDark = color.RGBA{50, 100, 150, 255}  // Darker border
	ColorInputBg    = color.RGBA{20, 20, 45, 255}    // Input background
	ColorInputFocus = color.RGBA{100, 200, 255, 255} // Focus highlight

	// Accent colors for variety
	ColorAccent1 = color.RGBA{255, 100, 200, 255} // Pink
	ColorAccent2 = color.RGBA{100, 255, 200, 255} // Mint
	ColorAccent3 = color.RGBA{255, 200, 100, 255} // Orange
)

// Player colors - 12 colors bright enough to see black icons
var PlayerColors = map[string]color.RGBA{
	"orange": {255, 140, 0, 255},
	"cyan":   {0, 200, 200, 255},
	"green":  {50, 180, 50, 255},
	"yellow": {220, 200, 50, 255},
	"purple": {160, 80, 200, 255},
	"red":    {200, 50, 50, 255},
	"blue":   {80, 100, 200, 255},
	"pink":   {255, 120, 180, 255},
	"lime":   {140, 220, 80, 255},
	"teal":   {60, 180, 150, 255},
	"coral":  {255, 100, 100, 255},
	"sky":    {100, 180, 255, 255},
}

// PlayerColorOrder defines the display order for color picker
var PlayerColorOrder = []string{
	"orange", "yellow", "lime", "green",
	"teal", "cyan", "sky", "blue",
	"purple", "pink", "coral", "red",
}

// Button represents a clickable button.
type Button struct {
	X, Y, W, H int
	Text       string
	OnClick    func()
	Disabled   bool
	Primary    bool
	hovered    bool
}

// Update handles button input.
func (b *Button) Update() {
	if b.Disabled {
		b.hovered = false
		return
	}

	mx, my := ebiten.CursorPosition()
	b.hovered = mx >= b.X && mx < b.X+b.W && my >= b.Y && my < b.Y+b.H

	if b.hovered && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if b.OnClick != nil {
			b.OnClick()
		}
	}
}

// Draw renders the button.
func (b *Button) Draw(screen *ebiten.Image) {
	var bgColor color.RGBA
	if b.Disabled {
		bgColor = ColorSecondary
	} else if b.Primary {
		if b.hovered {
			bgColor = ColorPrimaryHover
		} else {
			bgColor = ColorPrimary
		}
	} else {
		if b.hovered {
			bgColor = ColorSecondaryHover
		} else {
			bgColor = ColorSecondary
		}
	}

	// Draw background
	vector.DrawFilledRect(screen, float32(b.X), float32(b.Y), float32(b.W), float32(b.H), bgColor, false)

	// Draw border
	vector.StrokeRect(screen, float32(b.X), float32(b.Y), float32(b.W), float32(b.H), 1, ColorBorder, false)

	// Draw text centered
	textColor := ColorText
	if b.Disabled {
		textColor = ColorTextMuted
	}
	DrawTextCentered(screen, b.Text, b.X+b.W/2, b.Y+b.H/2-6, textColor)
}

// TextInput represents a text input field.
type TextInput struct {
	X, Y, W, H  int
	Placeholder string
	Text        string
	MaxLength   int
	focused     bool
	cursorBlink int
}

// Update handles text input.
func (t *TextInput) Update() {
	mx, my := ebiten.CursorPosition()
	inBounds := mx >= t.X && mx < t.X+t.W && my >= t.Y && my < t.Y+t.H

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		t.focused = inBounds
	}

	if !t.focused {
		return
	}

	t.cursorBlink++

	// Handle paste (Ctrl+V on Windows/Linux, Cmd+V on Mac)
	if t.isPastePressed() {
		t.pasteFromClipboard()
	}

	// Handle text input
	chars := ebiten.AppendInputChars(nil)
	for _, c := range chars {
		if t.MaxLength == 0 || len(t.Text) < t.MaxLength {
			t.Text += string(c)
		}
	}

	// Handle backspace
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) || inpututil.KeyPressDuration(ebiten.KeyBackspace) > 30 {
		if len(t.Text) > 0 {
			t.Text = t.Text[:len(t.Text)-1]
		}
	}
}

// isPastePressed returns true if the paste shortcut is pressed.
// Ctrl+V on Windows/Linux, Cmd+V on Mac.
func (t *TextInput) isPastePressed() bool {
	if !inpututil.IsKeyJustPressed(ebiten.KeyV) {
		return false
	}

	// On Mac, use Cmd (Meta). On Windows/Linux, use Ctrl.
	if runtime.GOOS == "darwin" {
		return ebiten.IsKeyPressed(ebiten.KeyMeta)
	}
	return ebiten.IsKeyPressed(ebiten.KeyControl)
}

// pasteFromClipboard pastes text from the clipboard into the input.
func (t *TextInput) pasteFromClipboard() {
	if !clipboardInitialized {
		return
	}

	data := clipboard.Read(clipboard.FmtText)
	if len(data) == 0 {
		return
	}

	text := string(data)
	// Remove any newlines/carriage returns for single-line input
	text = strings.ReplaceAll(text, "\r\n", "")
	text = strings.ReplaceAll(text, "\n", "")
	text = strings.ReplaceAll(text, "\r", "")

	// Append text respecting max length
	for _, c := range text {
		if t.MaxLength == 0 || len(t.Text) < t.MaxLength {
			t.Text += string(c)
		} else {
			break
		}
	}
}

// Draw renders the text input.
func (t *TextInput) Draw(screen *ebiten.Image) {
	// Draw background
	vector.DrawFilledRect(screen, float32(t.X), float32(t.Y), float32(t.W), float32(t.H), ColorInputBg, false)

	// Draw border
	borderColor := ColorBorder
	if t.focused {
		borderColor = ColorInputFocus
	}
	vector.StrokeRect(screen, float32(t.X), float32(t.Y), float32(t.W), float32(t.H), 2, borderColor, false)

	// Draw text or placeholder
	text := t.Text
	textColor := ColorText
	if text == "" {
		text = t.Placeholder
		textColor = ColorTextMuted
	}

	// Clip text to fit (debug font is 6 pixels per character)
	maxChars := (t.W - 16) / 6
	if len(text) > maxChars {
		text = text[len(text)-maxChars:]
	}

	ebitenutil.DebugPrintAt(screen, text, t.X+8, t.Y+t.H/2-6)
	_ = textColor // TODO: Use custom font rendering for colored text

	// Draw cursor (debug font is 6 pixels per character)
	if t.focused && (t.cursorBlink/30)%2 == 0 {
		cursorX := t.X + 8 + len(t.Text)*6
		if cursorX < t.X+t.W-8 {
			vector.DrawFilledRect(screen, float32(cursorX), float32(t.Y+8), 2, float32(t.H-16), ColorText, false)
		}
	}
}

// IsFocused returns true if the input is focused.
func (t *TextInput) IsFocused() bool {
	return t.focused
}

// Slider represents a draggable slider control.
type Slider struct {
	X, Y, W, H int
	Min, Max   int    // Value range
	Value      int    // Current value
	Label      string // Label to display
	OnChange   func(int)
	dragging   bool
	hovered    bool
}

// Update handles slider input.
func (s *Slider) Update() {
	mx, my := ebiten.CursorPosition()
	
	// Track area (center bar of slider)
	trackY := s.Y + s.H/2 - 4
	trackH := 8
	s.hovered = mx >= s.X && mx < s.X+s.W && my >= trackY && my < trackY+trackH+16
	
	// Handle dragging
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && s.hovered {
		s.dragging = true
	}
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		s.dragging = false
	}
	
	if s.dragging {
		// Calculate value from mouse position
		relX := mx - s.X
		if relX < 0 {
			relX = 0
		}
		if relX > s.W {
			relX = s.W
		}
		
		newValue := s.Min + (s.Max-s.Min)*relX/s.W
		if newValue != s.Value {
			s.Value = newValue
			if s.OnChange != nil {
				s.OnChange(s.Value)
			}
		}
	}
}

// Draw renders the slider.
func (s *Slider) Draw(screen *ebiten.Image) {
	// Label and value display
	labelText := s.Label
	if labelText != "" {
		labelText += ":"
	}
	DrawText(screen, labelText, s.X, s.Y, ColorText)
	
	// Value display on the right
	valueStr := ""
	switch {
	case s.Max <= 10:
		valueStr = []string{"Low", "Med-Low", "Medium", "Med-High", "High"}[(s.Value-s.Min)*4/(s.Max-s.Min)]
	default:
		valueStr = string(rune('0'+s.Value/100%10)) + string(rune('0'+s.Value/10%10)) + string(rune('0'+s.Value%10))
		// Trim leading zeros for cleaner look
		for len(valueStr) > 1 && valueStr[0] == '0' {
			valueStr = valueStr[1:]
		}
	}
	DrawText(screen, valueStr, s.X+s.W-len(valueStr)*6, s.Y, ColorPrimary)
	
	// Track background
	trackY := s.Y + 18
	trackH := 8
	vector.DrawFilledRect(screen, float32(s.X), float32(trackY), float32(s.W), float32(trackH), ColorInputBg, false)
	vector.StrokeRect(screen, float32(s.X), float32(trackY), float32(s.W), float32(trackH), 1, ColorBorderDark, false)
	
	// Filled portion
	fillW := float32(s.W) * float32(s.Value-s.Min) / float32(s.Max-s.Min)
	if fillW > 0 {
		vector.DrawFilledRect(screen, float32(s.X), float32(trackY), fillW, float32(trackH), ColorPrimary, false)
	}
	
	// Handle (knob)
	handleX := float32(s.X) + fillW - 6
	if handleX < float32(s.X) {
		handleX = float32(s.X)
	}
	handleColor := ColorPrimary
	if s.hovered || s.dragging {
		handleColor = ColorPrimaryHover
	}
	vector.DrawFilledRect(screen, handleX, float32(trackY-4), 12, float32(trackH+8), handleColor, false)
	vector.StrokeRect(screen, handleX, float32(trackY-4), 12, float32(trackH+8), 1, ColorBorder, false)
}

// Panel draws a panel background.
// DrawPanel draws a retro-style panel with decorative borders.
func DrawPanel(screen *ebiten.Image, x, y, w, h int) {
	// Main panel background
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(w), float32(h), ColorPanel, false)

	// Outer border (bright)
	vector.StrokeRect(screen, float32(x), float32(y), float32(w), float32(h), 2, ColorBorder, false)

	// Inner border (darker) for depth
	vector.StrokeRect(screen, float32(x+2), float32(y+2), float32(w-4), float32(h-4), 1, ColorBorderDark, false)

	// Corner decorations
	cornerSize := float32(6)
	// Top-left
	vector.DrawFilledRect(screen, float32(x+4), float32(y+4), cornerSize, 2, ColorBorder, false)
	vector.DrawFilledRect(screen, float32(x+4), float32(y+4), 2, cornerSize, ColorBorder, false)
	// Top-right
	vector.DrawFilledRect(screen, float32(x+w-10), float32(y+4), cornerSize, 2, ColorBorder, false)
	vector.DrawFilledRect(screen, float32(x+w-6), float32(y+4), 2, cornerSize, ColorBorder, false)
	// Bottom-left
	vector.DrawFilledRect(screen, float32(x+4), float32(y+h-6), cornerSize, 2, ColorBorder, false)
	vector.DrawFilledRect(screen, float32(x+4), float32(y+h-10), 2, cornerSize, ColorBorder, false)
	// Bottom-right
	vector.DrawFilledRect(screen, float32(x+w-10), float32(y+h-6), cornerSize, 2, ColorBorder, false)
	vector.DrawFilledRect(screen, float32(x+w-6), float32(y+h-10), 2, cornerSize, ColorBorder, false)
}

// DrawFancyPanel draws a panel with animated border effect.
func DrawFancyPanel(screen *ebiten.Image, x, y, w, h int, title string) {
	DrawPanel(screen, x, y, w, h)

	// Title bar
	if title != "" {
		titleBarH := 30
		vector.DrawFilledRect(screen, float32(x+2), float32(y+2), float32(w-4), float32(titleBarH), ColorPanelLight, false)
		vector.StrokeLine(screen, float32(x+2), float32(y+titleBarH+2), float32(x+w-2), float32(y+titleBarH+2), 2, ColorBorder, false)

		// Title text
		DrawText(screen, title, x+10, y+10, ColorText)
	}
}

// DrawText draws text at a position.
// DrawText draws text with a dark shadow for contrast.
func DrawText(screen *ebiten.Image, text string, x, y int, clr color.Color) {
	// Simple shadow
	ebitenutil.DebugPrintAt(screen, text, x+1, y+1)
	// Main text
	ebitenutil.DebugPrintAt(screen, text, x, y)
}

// DrawTextCentered draws text centered at a position.
func DrawTextCentered(screen *ebiten.Image, text string, x, y int, clr color.Color) {
	w := len(text) * 6
	DrawText(screen, text, x-w/2, y, clr)
}

// DrawLargeText draws scaled-up text (2x). Height is ~24px.
func DrawLargeText(screen *ebiten.Image, text string, x, y int, clr color.Color) {
	// Create a temporary image to render the text
	textW := len(text) * 6
	textH := 12
	tmpImg := ebiten.NewImage(textW, textH)

	// Render text to temp image
	ebitenutil.DebugPrintAt(tmpImg, text, 0, 0)

	// Get color components
	r, g, b, a := clr.RGBA()
	rf := float32(r) / 0xffff
	gf := float32(g) / 0xffff
	bf := float32(b) / 0xffff
	af := float32(a) / 0xffff

	// Draw shadow
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Scale(2.0, 2.0)
	opts.GeoM.Translate(float64(x+1), float64(y+1))
	opts.ColorScale.Scale(0, 0, 0, af*0.5)
	screen.DrawImage(tmpImg, opts)

	// Draw main scaled text with color
	opts = &ebiten.DrawImageOptions{}
	opts.GeoM.Scale(2.0, 2.0)
	opts.GeoM.Translate(float64(x), float64(y))
	opts.ColorScale.Scale(rf, gf, bf, af)
	screen.DrawImage(tmpImg, opts)
}

// DrawLargeTextCentered draws large text centered.
func DrawLargeTextCentered(screen *ebiten.Image, text string, x, y int, clr color.Color) {
	w := len(text) * 6 * 2 // 2x scale
	DrawLargeText(screen, text, x-w/2, y, clr)
}

// DrawHugeTitle draws a massive title (3x scale). Height is ~36px.
func DrawHugeTitle(screen *ebiten.Image, text string, x, y int) {
	// Create a temporary image to render the text
	textW := len(text) * 6
	textH := 12
	tmpImg := ebiten.NewImage(textW, textH)

	// Render text to temp image
	ebitenutil.DebugPrintAt(tmpImg, text, 0, 0)

	// Gold/amber color for title
	rf := float32(1.0)
	gf := float32(0.85)
	bf := float32(0.3)

	// Draw shadow
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Scale(3.0, 3.0)
	opts.GeoM.Translate(float64(x+2), float64(y+2))
	opts.ColorScale.Scale(0, 0, 0, 0.6)
	screen.DrawImage(tmpImg, opts)

	// Draw main scaled text with color
	opts = &ebiten.DrawImageOptions{}
	opts.GeoM.Scale(3.0, 3.0)
	opts.GeoM.Translate(float64(x), float64(y))
	opts.ColorScale.Scale(rf, gf, bf, 1.0)
	screen.DrawImage(tmpImg, opts)
}

// DrawHugeTitleCentered draws a huge title centered.
func DrawHugeTitleCentered(screen *ebiten.Image, text string, x, y int) {
	w := len(text) * 6 * 3 // 3x scale
	DrawHugeTitle(screen, text, x-w/2, y)
}

// DrawTitle draws a title (alias for DrawLargeText).
func DrawTitle(screen *ebiten.Image, text string, x, y int) {
	DrawLargeText(screen, text, x, y, ColorText)
}

// ListItem represents an item in a list.
type ListItem struct {
	ID       string
	Text     string
	Subtext  string
	Detail   string // Additional detail line (e.g., player names)
	Selected bool
}

// List represents a scrollable list of items.
type List struct {
	X, Y, W, H   int
	Items        []ListItem
	OnSelect     func(id string)
	selectedIdx  int
	scrollOffset int
	itemHeight   int
}

// NewList creates a new list.
func NewList(x, y, w, h int) *List {
	return &List{
		X:           x,
		Y:           y,
		W:           w,
		H:           h,
		itemHeight:  50,
		selectedIdx: -1,
	}
}

// Update handles list input.
func (l *List) Update() {
	mx, my := ebiten.CursorPosition()

	// Check if mouse is in list bounds
	if mx < l.X || mx >= l.X+l.W || my < l.Y || my >= l.Y+l.H {
		return
	}

	itemHeight := l.itemHeight
	if itemHeight <= 0 {
		itemHeight = 60 // default
	}

	// Handle scroll
	_, dy := ebiten.Wheel()
	l.scrollOffset -= int(dy * 30)
	if l.scrollOffset < 0 {
		l.scrollOffset = 0
	}
	maxScroll := len(l.Items)*itemHeight - l.H
	if maxScroll < 0 {
		maxScroll = 0
	}
	if l.scrollOffset > maxScroll {
		l.scrollOffset = maxScroll
	}

	// Handle click
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		relY := my - l.Y + l.scrollOffset
		idx := relY / itemHeight
		if idx >= 0 && idx < len(l.Items) {
			l.selectedIdx = idx
			if l.OnSelect != nil {
				l.OnSelect(l.Items[idx].ID)
			}
		}
	}
}

// Draw renders the list.
func (l *List) Draw(screen *ebiten.Image) {
	// Draw background
	DrawPanel(screen, l.X, l.Y, l.W, l.H)

	// Use configured item height
	itemHeight := l.itemHeight
	if itemHeight <= 0 {
		itemHeight = 60 // default
	}

	// Create a clipped sub-image for drawing list contents
	// This prevents items from rendering outside the list bounds
	clipRect := image.Rect(l.X, l.Y, l.X+l.W, l.Y+l.H)
	clippedScreen := screen.SubImage(clipRect).(*ebiten.Image)

	// Draw items
	visibleStart := l.scrollOffset / itemHeight
	visibleEnd := (l.scrollOffset+l.H)/itemHeight + 2 // +2 to include partially visible items

	for i := visibleStart; i < visibleEnd && i < len(l.Items); i++ {
		item := l.Items[i]
		itemY := l.Y + i*itemHeight - l.scrollOffset

		// Skip items completely outside bounds
		if itemY+itemHeight < l.Y || itemY > l.Y+l.H {
			continue
		}

		// Draw selection highlight (to clipped screen)
		if i == l.selectedIdx {
			vector.DrawFilledRect(clippedScreen, float32(l.X+4), float32(itemY+4),
				float32(l.W-8), float32(itemHeight-8), ColorPanelLight, false)
			// Bright border for selected item
			vector.StrokeRect(clippedScreen, float32(l.X+4), float32(itemY+4),
				float32(l.W-8), float32(itemHeight-8), 2, ColorBorder, false)
		}

		// Draw text to clipped screen
		DrawLargeText(clippedScreen, item.Text, l.X+15, itemY+6, ColorText)
		if item.Subtext != "" {
			DrawText(clippedScreen, item.Subtext, l.X+15, itemY+28, ColorTextMuted)
		}
		if item.Detail != "" {
			DrawText(clippedScreen, item.Detail, l.X+15, itemY+44, ColorTextDim)
		}
	}

	// Draw scrollbar if needed (on main screen, outside clip area)
	totalHeight := len(l.Items) * itemHeight
	if totalHeight > l.H {
		scrollbarHeight := float32(l.H) * float32(l.H) / float32(totalHeight)
		scrollbarY := float32(l.Y) + float32(l.scrollOffset)*float32(l.H)/float32(totalHeight)
		vector.DrawFilledRect(screen, float32(l.X+l.W-10), scrollbarY, 8, scrollbarHeight, ColorBorder, false)
	}
}

// SetItems sets the list items and resets scroll.
func (l *List) SetItems(items []ListItem) {
	l.Items = items
	l.scrollOffset = 0
	l.selectedIdx = -1
}

// SetItemsPreserve sets the list items while preserving selection and scroll position.
// If the previously selected item ID still exists, it stays selected.
func (l *List) SetItemsPreserve(items []ListItem, previousSelectedID string) {
	oldScroll := l.scrollOffset
	l.Items = items

	// Preserve selection if the item still exists
	l.selectedIdx = -1
	if previousSelectedID != "" {
		for i, item := range items {
			if item.ID == previousSelectedID {
				l.selectedIdx = i
				break
			}
		}
	}

	// Preserve scroll position (clamp to valid range)
	itemHeight := 60
	maxScroll := len(l.Items)*itemHeight - l.H
	if maxScroll < 0 {
		maxScroll = 0
	}
	if oldScroll > maxScroll {
		l.scrollOffset = maxScroll
	} else {
		l.scrollOffset = oldScroll
	}
}

// GetSelectedID returns the selected item's ID.
func (l *List) GetSelectedID() string {
	if l.selectedIdx >= 0 && l.selectedIdx < len(l.Items) {
		return l.Items[l.selectedIdx].ID
	}
	return ""
}

// ClearSelection clears the current selection.
func (l *List) ClearSelection() {
	l.selectedIdx = -1
}

// Contains checks if a string contains a substring (case insensitive).
func Contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
