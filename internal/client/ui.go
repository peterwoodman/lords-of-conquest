package client

import (
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Colors used in the UI
var (
	ColorBackground    = color.RGBA{20, 20, 30, 255}
	ColorPanel         = color.RGBA{30, 35, 50, 255}
	ColorPanelLight    = color.RGBA{45, 50, 70, 255}
	ColorPrimary       = color.RGBA{70, 130, 180, 255}  // Steel blue
	ColorPrimaryHover  = color.RGBA{100, 160, 210, 255}
	ColorSecondary     = color.RGBA{60, 60, 80, 255}
	ColorSecondaryHover= color.RGBA{80, 80, 100, 255}
	ColorSuccess       = color.RGBA{50, 150, 80, 255}
	ColorDanger        = color.RGBA{180, 60, 60, 255}
	ColorText          = color.RGBA{220, 220, 230, 255}
	ColorTextMuted     = color.RGBA{140, 140, 160, 255}
	ColorBorder        = color.RGBA{60, 65, 80, 255}
	ColorInputBg       = color.RGBA{25, 28, 40, 255}
	ColorInputFocus    = color.RGBA{70, 130, 180, 255}
)

// Player colors
var PlayerColors = map[string]color.RGBA{
	"orange": {255, 140, 0, 255},
	"cyan":   {0, 200, 200, 255},
	"green":  {50, 180, 50, 255},
	"yellow": {220, 200, 50, 255},
	"purple": {160, 80, 200, 255},
	"red":    {200, 50, 50, 255},
	"blue":   {80, 100, 200, 255},
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

	// Clip text to fit
	maxChars := (t.W - 16) / 7
	if len(text) > maxChars {
		text = text[len(text)-maxChars:]
	}

	ebitenutil.DebugPrintAt(screen, text, t.X+8, t.Y+t.H/2-6)
	_ = textColor // TODO: Use custom font rendering for colored text

	// Draw cursor
	if t.focused && (t.cursorBlink/30)%2 == 0 {
		cursorX := t.X + 8 + len(t.Text)*7
		if cursorX < t.X+t.W-8 {
			vector.DrawFilledRect(screen, float32(cursorX), float32(t.Y+8), 2, float32(t.H-16), ColorText, false)
		}
	}
}

// IsFocused returns true if the input is focused.
func (t *TextInput) IsFocused() bool {
	return t.focused
}

// Panel draws a panel background.
func DrawPanel(screen *ebiten.Image, x, y, w, h int) {
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(w), float32(h), ColorPanel, false)
	vector.StrokeRect(screen, float32(x), float32(y), float32(w), float32(h), 1, ColorBorder, false)
}

// DrawText draws text at a position.
func DrawText(screen *ebiten.Image, text string, x, y int, clr color.Color) {
	ebitenutil.DebugPrintAt(screen, text, x, y)
}

// DrawTextCentered draws text centered at a position.
func DrawTextCentered(screen *ebiten.Image, text string, x, y int, clr color.Color) {
	w := len(text) * 6
	ebitenutil.DebugPrintAt(screen, text, x-w/2, y)
}

// DrawTitle draws a large title (using multiple prints for now).
func DrawTitle(screen *ebiten.Image, text string, x, y int) {
	// Simple centered text for now
	DrawTextCentered(screen, text, x, y, ColorText)
}

// ListItem represents an item in a list.
type ListItem struct {
	ID       string
	Text     string
	Subtext  string
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
		X:          x,
		Y:          y,
		W:          w,
		H:          h,
		itemHeight: 50,
	}
}

// Update handles list input.
func (l *List) Update() {
	mx, my := ebiten.CursorPosition()

	// Check if mouse is in list bounds
	if mx < l.X || mx >= l.X+l.W || my < l.Y || my >= l.Y+l.H {
		return
	}

	// Handle scroll
	_, dy := ebiten.Wheel()
	l.scrollOffset -= int(dy * 30)
	if l.scrollOffset < 0 {
		l.scrollOffset = 0
	}
	maxScroll := len(l.Items)*l.itemHeight - l.H
	if maxScroll < 0 {
		maxScroll = 0
	}
	if l.scrollOffset > maxScroll {
		l.scrollOffset = maxScroll
	}

	// Handle click
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		relY := my - l.Y + l.scrollOffset
		idx := relY / l.itemHeight
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

	// Draw items
	visibleStart := l.scrollOffset / l.itemHeight
	visibleEnd := (l.scrollOffset + l.H) / l.itemHeight + 1

	for i := visibleStart; i < visibleEnd && i < len(l.Items); i++ {
		item := l.Items[i]
		itemY := l.Y + i*l.itemHeight - l.scrollOffset

		if itemY < l.Y-l.itemHeight || itemY > l.Y+l.H {
			continue
		}

		// Draw selection highlight
		if i == l.selectedIdx {
			vector.DrawFilledRect(screen, float32(l.X+2), float32(itemY+2),
				float32(l.W-4), float32(l.itemHeight-4), ColorPanelLight, false)
		}

		// Draw text
		DrawText(screen, item.Text, l.X+10, itemY+10, ColorText)
		if item.Subtext != "" {
			DrawText(screen, item.Subtext, l.X+10, itemY+28, ColorTextMuted)
		}
	}

	// Draw scrollbar if needed
	if len(l.Items)*l.itemHeight > l.H {
		totalHeight := len(l.Items) * l.itemHeight
		scrollbarHeight := float32(l.H) * float32(l.H) / float32(totalHeight)
		scrollbarY := float32(l.Y) + float32(l.scrollOffset)*float32(l.H)/float32(totalHeight)
		vector.DrawFilledRect(screen, float32(l.X+l.W-8), scrollbarY, 6, scrollbarHeight, ColorBorder, false)
	}
}

// SetItems sets the list items.
func (l *List) SetItems(items []ListItem) {
	l.Items = items
	l.scrollOffset = 0
}

// GetSelectedID returns the selected item's ID.
func (l *List) GetSelectedID() string {
	if l.selectedIdx >= 0 && l.selectedIdx < len(l.Items) {
		return l.Items[l.selectedIdx].ID
	}
	return ""
}

// Contains checks if a string contains a substring (case insensitive).
func Contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

