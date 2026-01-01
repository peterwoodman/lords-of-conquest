package client

import (
	"fmt"
	"image/color"

	"lords-of-conquest/internal/protocol"
	"lords-of-conquest/pkg/maps"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// ==================== Title Scene ====================

// TitleScene shows the game's title screen with a transition.
type TitleScene struct {
	game *Game

	timer       int     // Frame counter
	phase       int     // Current phase
	fadeAlpha   float64 // 0.0 to 1.0 for fade transition
	zoomLevel   float64 // Current zoom level (1.0 = normal, 5.0 = 500%)
	skipPressed bool
}

// Title screen timing (at 60fps)
const (
	titlePhase8Bit    = 0
	titlePhaseZoomIn  = 1
	titlePhaseFading  = 2
	titlePhaseZoomOut = 3
	titlePhaseModern  = 4
	titlePhaseDone    = 5

	title8BitDuration    = 10 // 1 second of 8-bit before zoom
	titleZoomInDuration  = 160 // 1.5 seconds to zoom in
	titleFadeDuration    = 90 // 1 second fade while zoomed
	titleZoomOutDuration = 160 // 1.5 seconds to zoom out
	titleModernDuration  = 60 // 1 second of modern before connect

	// Zoom parameters
	titleZoomFocusX = 0.75 // 80% on X axis
	titleZoomFocusY = 0.35 // 39% on Y axis
	titleZoomMax    = 5.0  // 500% zoom
)

// NewTitleScene creates a new title scene.
func NewTitleScene(game *Game) *TitleScene {
	return &TitleScene{game: game}
}

func (s *TitleScene) OnEnter() {
	s.timer = 0
	s.phase = titlePhase8Bit
	s.fadeAlpha = 0
	s.zoomLevel = 1.0
	s.skipPressed = false
}

func (s *TitleScene) OnExit() {}

func (s *TitleScene) Update() error {
	s.timer++

	// Allow skipping with any key or click
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) ||
		inpututil.IsKeyJustPressed(ebiten.KeySpace) ||
		inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
		inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		s.skipPressed = true
	}

	// If skipped, go straight to connect with title background
	if s.skipPressed {
		s.game.connectScene.showTitleBackground = true
		s.game.SetScene(s.game.connectScene)
		return nil
	}

	// Phase transitions based on timer
	switch s.phase {
	case titlePhase8Bit:
		// Static 8-bit display before zoom
		if s.timer >= title8BitDuration {
			s.phase = titlePhaseZoomIn
			s.timer = 0
		}
	case titlePhaseZoomIn:
		// Zoom in on the 8-bit image
		progress := float64(s.timer) / float64(titleZoomInDuration)
		// Ease-out curve for smooth zoom
		eased := 1 - (1-progress)*(1-progress)
		s.zoomLevel = 1.0 + (titleZoomMax-1.0)*eased
		if s.timer >= titleZoomInDuration {
			s.zoomLevel = titleZoomMax
			s.phase = titlePhaseFading
			s.timer = 0
		}
	case titlePhaseFading:
		// Crossfade while zoomed
		s.fadeAlpha = float64(s.timer) / float64(titleFadeDuration)
		if s.fadeAlpha >= 1.0 {
			s.fadeAlpha = 1.0
			s.phase = titlePhaseZoomOut
			s.timer = 0
		}
	case titlePhaseZoomOut:
		// Zoom out on the modern image
		progress := float64(s.timer) / float64(titleZoomOutDuration)
		// Ease-in curve for smooth zoom out
		eased := progress * progress
		s.zoomLevel = titleZoomMax - (titleZoomMax-1.0)*eased
		if s.timer >= titleZoomOutDuration {
			s.zoomLevel = 1.0
			s.phase = titlePhaseModern
			s.timer = 0
		}
	case titlePhaseModern:
		if s.timer >= titleModernDuration {
			s.phase = titlePhaseDone
			s.game.connectScene.showTitleBackground = true
			s.game.SetScene(s.game.connectScene)
		}
	}

	return nil
}

func (s *TitleScene) Draw(screen *ebiten.Image) {
	// Get title images
	img8Bit := GetTitleScreen8Bit()
	imgModern := GetTitleScreenModern()

	// Calculate scaling to fill screen while maintaining aspect ratio
	screenW := float64(ScreenWidth)
	screenH := float64(ScreenHeight)

	switch s.phase {
	case titlePhase8Bit:
		// Show only 8-bit version (no zoom yet)
		if img8Bit != nil {
			s.drawImageFullScreen(screen, img8Bit)
		} else {
			// Fallback if image not loaded
			screen.Fill(color.RGBA{0, 0, 0, 255})
			DrawLargeTextCentered(screen, "LORDS OF CONQUEST", int(screenW)/2, int(screenH)/2-20, ColorPrimary)
			DrawTextCentered(screen, "(8-bit title screen)", int(screenW)/2, int(screenH)/2+20, ColorTextMuted)
		}

	case titlePhaseZoomIn:
		// Zooming in on 8-bit image
		if img8Bit != nil {
			s.drawImageZoomed(screen, img8Bit, s.zoomLevel, 1.0)
		}

	case titlePhaseFading:
		// Crossfade between 8-bit and modern while zoomed
		if img8Bit != nil {
			s.drawImageZoomed(screen, img8Bit, s.zoomLevel, 1.0)
		}
		if imgModern != nil {
			// Draw modern on top with increasing alpha
			s.drawImageZoomed(screen, imgModern, s.zoomLevel, s.fadeAlpha)
		}

	case titlePhaseZoomOut:
		// Zooming out on modern image
		if imgModern != nil {
			s.drawImageZoomed(screen, imgModern, s.zoomLevel, 1.0)
		}

	case titlePhaseModern, titlePhaseDone:
		// Show only modern version (no zoom)
		if imgModern != nil {
			s.drawImageFullScreen(screen, imgModern)
		} else {
			screen.Fill(color.RGBA{20, 20, 40, 255})
			DrawLargeTextCentered(screen, "LORDS OF CONQUEST", int(screenW)/2, int(screenH)/2-20, ColorPrimary)
		}
	}

	// Skip hint at bottom
	DrawTextCentered(screen, "Press any key to skip", int(screenW)/2, int(screenH)-30, ColorTextMuted)
}

func (s *TitleScene) drawImageFullScreen(screen *ebiten.Image, img *ebiten.Image) {
	if img == nil {
		return
	}

	imgW := float64(img.Bounds().Dx())
	imgH := float64(img.Bounds().Dy())
	screenW := float64(ScreenWidth)
	screenH := float64(ScreenHeight)

	// Scale to cover screen
	scaleX := screenW / imgW
	scaleY := screenH / imgH
	scale := scaleX
	if scaleY > scaleX {
		scale = scaleY
	}

	// Center the image
	offsetX := (screenW - imgW*scale) / 2
	offsetY := (screenH - imgH*scale) / 2

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(offsetX, offsetY)
	screen.DrawImage(img, op)
}

func (s *TitleScene) drawImageFullScreenWithAlpha(screen *ebiten.Image, img *ebiten.Image, alpha float64) {
	if img == nil {
		return
	}

	imgW := float64(img.Bounds().Dx())
	imgH := float64(img.Bounds().Dy())
	screenW := float64(ScreenWidth)
	screenH := float64(ScreenHeight)

	// Scale to cover screen
	scaleX := screenW / imgW
	scaleY := screenH / imgH
	scale := scaleX
	if scaleY > scaleX {
		scale = scaleY
	}

	// Center the image
	offsetX := (screenW - imgW*scale) / 2
	offsetY := (screenH - imgH*scale) / 2

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(offsetX, offsetY)
	op.ColorScale.ScaleAlpha(float32(alpha))
	screen.DrawImage(img, op)
}

// drawImageZoomed draws an image zoomed around a focal point (titleZoomFocusX, titleZoomFocusY)
// The focal point in the image will be centered on screen when zoomed.
func (s *TitleScene) drawImageZoomed(screen *ebiten.Image, img *ebiten.Image, zoom float64, alpha float64) {
	if img == nil {
		return
	}

	imgW := float64(img.Bounds().Dx())
	imgH := float64(img.Bounds().Dy())
	screenW := float64(ScreenWidth)
	screenH := float64(ScreenHeight)

	// Base scale to cover screen
	scaleX := screenW / imgW
	scaleY := screenH / imgH
	baseScale := scaleX
	if scaleY > scaleX {
		baseScale = scaleY
	}

	// Total scale including zoom
	totalScale := baseScale * zoom

	// The focal point in image coordinates
	focalImgX := imgW * titleZoomFocusX
	focalImgY := imgH * titleZoomFocusY

	// Where the focal point ends up after base scaling (centered image)
	baseOffsetX := (screenW - imgW*baseScale) / 2
	baseOffsetY := (screenH - imgH*baseScale) / 2
	focalScreenX := baseOffsetX + focalImgX*baseScale
	focalScreenY := baseOffsetY + focalImgY*baseScale

	// For zoom > 1, we want the focal point to move toward screen center
	// Interpolate the focal point position from its base position to screen center
	targetX := screenW / 2
	targetY := screenH / 2

	// How much to interpolate (0 = no zoom effect, 1 = fully centered)
	t := (zoom - 1.0) / (titleZoomMax - 1.0)

	// Current focal point target position (interpolated)
	currentFocalX := focalScreenX + (targetX-focalScreenX)*t
	currentFocalY := focalScreenY + (targetY-focalScreenY)*t

	// Calculate offset so that after scaling, the focal point lands at currentFocalX/Y
	// After scaling: focalImgX * totalScale + offsetX = currentFocalX
	offsetX := currentFocalX - focalImgX*totalScale
	offsetY := currentFocalY - focalImgY*totalScale

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(totalScale, totalScale)
	op.GeoM.Translate(offsetX, offsetY)
	if alpha < 1.0 {
		op.ColorScale.ScaleAlpha(float32(alpha))
	}
	screen.DrawImage(img, op)
}

// ==================== Connect Scene ====================

// ConnectScene handles server connection and player name entry.
type ConnectScene struct {
	game *Game

	serverInput *TextInput
	nameInput   *TextInput
	connectBtn  *Button
	statusText  string
	connecting  bool

	// If true, show the modern title screen as background
	showTitleBackground bool
}

// NewConnectScene creates a new connect scene.
func NewConnectScene(game *Game) *ConnectScene {
	s := &ConnectScene{game: game}

	s.serverInput = &TextInput{
		X: ScreenWidth/2 - 150, Y: 280,
		W: 300, H: 40,
		Placeholder: "Server address",
		MaxLength:   100,
	}

	s.nameInput = &TextInput{
		X: ScreenWidth/2 - 150, Y: 340,
		W: 300, H: 40,
		Placeholder: "Your name",
		MaxLength:   20,
	}

	s.connectBtn = &Button{
		X: ScreenWidth/2 - 100, Y: 410,
		W: 200, H: 45,
		Text:    "Connect",
		Primary: true,
	}

	s.connectBtn.OnClick = s.onConnect

	return s
}

func (s *ConnectScene) OnEnter() {
	// Load saved values
	if s.game.config.LastServer != "" {
		s.serverInput.Text = s.game.config.LastServer
	}
	if s.game.config.PlayerName != "" {
		s.nameInput.Text = s.game.config.PlayerName
	}
	s.connecting = false
	s.connectBtn.Disabled = false
	s.statusText = ""
}

func (s *ConnectScene) OnExit() {}

func (s *ConnectScene) Update() error {
	if s.connecting {
		return nil
	}

	s.serverInput.Update()
	s.nameInput.Update()
	s.connectBtn.Update()

	// Enter key to connect
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) && !s.nameInput.IsFocused() {
		s.onConnect()
	}

	return nil
}

func (s *ConnectScene) Draw(screen *ebiten.Image) {
	// Background - either title screen or starfield
	if s.showTitleBackground {
		imgModern := GetTitleScreenModern()
		if imgModern != nil {
			s.drawTitleBackground(screen, imgModern)
		}
	} else {
		// Starfield background effect (simple dots)
		for i := 0; i < 50; i++ {
			x := float32((i * 137) % ScreenWidth)
			y := float32((i * 97) % ScreenHeight)
			size := float32(1 + (i % 3))
			alpha := uint8(100 + (i % 155))
			starColor := color.RGBA{100, 150, 255, alpha}
			vector.DrawFilledCircle(screen, x, y, size, starColor, false)
		}
	}

	// Main panel
	panelW := 600
	panelH := 450
	panelX := ScreenWidth/2 - panelW/2
	panelY := ScreenHeight/2 - panelH/2
	DrawFancyPanel(screen, panelX, panelY, panelW, panelH, "")

	// Huge title - centered
	titleY := panelY + 40
	DrawHugeTitleCentered(screen, "LORDS OF CONQUEST", ScreenWidth/2, titleY)

	// Subtitle
	DrawTextCentered(screen, "Again", ScreenWidth/2, titleY+55, ColorTextMuted)

	// Server input
	inputY := panelY + 170
	DrawText(screen, "Server:", panelX+30, inputY-20, ColorTextMuted)
	s.serverInput.Y = inputY
	s.serverInput.H = 45
	s.serverInput.Draw(screen)

	// Name input
	inputY += 80
	DrawText(screen, "Your Name:", panelX+30, inputY-20, ColorTextMuted)
	s.nameInput.Y = inputY
	s.nameInput.H = 45
	s.nameInput.Draw(screen)

	// Connect button - bigger
	s.connectBtn.Y = panelY + panelH - 80
	s.connectBtn.H = 50
	s.connectBtn.Draw(screen)

	// Status text
	if s.statusText != "" {
		statusColor := ColorText
		if s.connecting {
			statusColor = ColorWarning
		}
		DrawLargeTextCentered(screen, s.statusText, ScreenWidth/2, panelY+panelH-25, statusColor)
	}

	// Version
	DrawText(screen, "v0.1.0", 10, ScreenHeight-30, ColorTextMuted)
}

func (s *ConnectScene) drawTitleBackground(screen *ebiten.Image, img *ebiten.Image) {
	if img == nil {
		return
	}

	imgW := float64(img.Bounds().Dx())
	imgH := float64(img.Bounds().Dy())
	screenW := float64(ScreenWidth)
	screenH := float64(ScreenHeight)

	// Scale to cover screen
	scaleX := screenW / imgW
	scaleY := screenH / imgH
	scale := scaleX
	if scaleY > scaleX {
		scale = scaleY
	}

	// Center the image
	offsetX := (screenW - imgW*scale) / 2
	offsetY := (screenH - imgH*scale) / 2

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(offsetX, offsetY)
	screen.DrawImage(img, op)
}

func (s *ConnectScene) onConnect() {
	server := s.serverInput.Text
	name := s.nameInput.Text

	if server == "" {
		s.statusText = "Please enter a server address"
		return
	}
	if name == "" {
		s.statusText = "Please enter your name"
		return
	}

	s.statusText = "Connecting..."
	s.connecting = true
	s.connectBtn.Disabled = true

	go func() {
		err := s.game.Connect(server)
		if err != nil {
			s.statusText = fmt.Sprintf("Connection failed: %v", err)
			s.connecting = false
			s.connectBtn.Disabled = false
			return
		}

		// Authenticate
		s.game.Authenticate(name)
	}()
}

// ==================== Lobby Scene ====================

// LobbyScene shows available games and allows creating/joining.
type LobbyScene struct {
	game *Game

	gameList     *List
	yourGameList *List
	codeInput    *TextInput
	createBtn    *Button
	joinBtn      *Button
	joinCodeBtn  *Button
	refreshBtn   *Button
	deleteBtn    *Button
	games        []protocol.GameListItem
	yourGames    []protocol.GameListItem
	selectedGame string

	// Create game dialog
	showCreate       bool
	createNameInput  *TextInput
	createPublicBtn  *Button
	createPrivateBtn *Button
	createConfirmBtn *Button
	createCancelBtn  *Button
	createPublic     bool

	// Map generation options
	mapSizeBtn     [3]*Button // S, M, L
	mapSize        int        // 0=S, 1=M, 2=L
	territoriesBtn [3]*Button // L, M, H
	territories    int        // 0=L, 1=M, 2=H
	waterBorderBtn *Button
	waterBorder    bool
	islandsBtn     [3]*Button // L, M, H
	islands        int        // 0=L, 1=M, 2=H
	resourcesBtn   [3]*Button // L, M, H
	resources      int        // 0=L, 1=M, 2=H
	generateBtn    *Button
	regenerateBtn  *Button

	// Map preview
	generatedMap   *maps.Map
	generatedSteps []maps.GeneratorStep
	previewGrid    [][]int
	previewWidth   int
	previewHeight  int
	animStep       int
	animTicker     int
	animating      bool

	// Auto-refresh timer (5 seconds at 60fps = 300 frames)
	refreshTimer int
}

// NewLobbyScene creates a new lobby scene.
func NewLobbyScene(game *Game) *LobbyScene {
	s := &LobbyScene{game: game}

	// Your games list (top half)
	s.yourGameList = NewList(50, 100, 500, 180)
	s.yourGameList.OnSelect = func(id string) {
		s.selectedGame = id
	}

	// Public game list (bottom half)
	s.gameList = NewList(50, 320, 500, 280)
	s.gameList.OnSelect = func(id string) {
		s.selectedGame = id
	}

	// Code input
	s.codeInput = &TextInput{
		X: 600, Y: 150,
		W: 200, H: 40,
		Placeholder: "Join code",
		MaxLength:   9,
	}

	// Buttons
	s.createBtn = &Button{
		X: 600, Y: 100, W: 200, H: 40,
		Text: "Create Game", Primary: true,
		OnClick: func() { s.showCreate = true; s.initMapGeneration() },
	}

	s.joinBtn = &Button{
		X: 600, Y: 250, W: 200, H: 40,
		Text:    "Join Selected",
		OnClick: s.onJoinSelected,
	}

	s.deleteBtn = &Button{
		X: 600, Y: 300, W: 200, H: 40,
		Text:    "Delete Selected",
		OnClick: s.onDeleteGame,
	}

	s.joinCodeBtn = &Button{
		X: 810, Y: 150, W: 100, H: 40,
		Text:    "Join",
		OnClick: s.onJoinByCode,
	}

	s.refreshBtn = &Button{
		X: 600, Y: 200, W: 200, H: 40,
		Text:    "Refresh",
		OnClick: s.onRefresh,
	}

	// Create dialog
	s.createNameInput = &TextInput{
		X: ScreenWidth/2 - 150, Y: 280,
		W: 300, H: 40,
		Placeholder: "Game name",
		MaxLength:   30,
	}

	s.createPublicBtn = &Button{
		X: ScreenWidth/2 - 150, Y: 340, W: 140, H: 40,
		Text: "Public", Primary: true,
		OnClick: func() { s.createPublic = true },
	}

	s.createPrivateBtn = &Button{
		X: ScreenWidth/2 + 10, Y: 340, W: 140, H: 40,
		Text:    "Private",
		OnClick: func() { s.createPublic = false },
	}

	s.createConfirmBtn = &Button{
		X: ScreenWidth/2 - 150, Y: 410, W: 140, H: 40,
		Text: "Create", Primary: true,
		OnClick: s.onCreateConfirm,
	}

	s.createCancelBtn = &Button{
		X: ScreenWidth/2 + 10, Y: 410, W: 140, H: 40,
		Text:    "Cancel",
		OnClick: func() { s.showCreate = false; s.animating = false },
	}

	s.createPublic = true

	// Map size buttons
	sizes := []string{"S", "M", "L"}
	for i := 0; i < 3; i++ {
		idx := i
		s.mapSizeBtn[i] = &Button{
			Text:    sizes[i],
			OnClick: func() { s.mapSize = idx },
		}
	}
	s.mapSize = 1 // Default medium

	// Territory count buttons
	counts := []string{"Low", "Med", "High"}
	for i := 0; i < 3; i++ {
		idx := i
		s.territoriesBtn[i] = &Button{
			Text:    counts[i],
			OnClick: func() { s.territories = idx },
		}
	}
	s.territories = 1 // Default medium

	// Water border button
	s.waterBorderBtn = &Button{
		Text:    "Water Border",
		OnClick: func() { s.waterBorder = !s.waterBorder },
	}
	s.waterBorder = true // Default on

	// Islands buttons
	for i := 0; i < 3; i++ {
		idx := i
		s.islandsBtn[i] = &Button{
			Text:    counts[i],
			OnClick: func() { s.islands = idx },
		}
	}
	s.islands = 1 // Default medium

	// Resources buttons
	for i := 0; i < 3; i++ {
		idx := i
		s.resourcesBtn[i] = &Button{
			Text:    counts[i],
			OnClick: func() { s.resources = idx },
		}
	}
	s.resources = 1 // Default medium

	// Generate buttons
	s.generateBtn = &Button{
		Text:    "Generate Map",
		Primary: true,
		OnClick: s.onGenerateMap,
	}

	s.regenerateBtn = &Button{
		Text:    "Regenerate",
		OnClick: s.onGenerateMap,
	}

	return s
}

func (s *LobbyScene) OnEnter() {
	s.showCreate = false
	s.refreshTimer = 0 // Reset auto-refresh timer
	s.game.ListGames()
	s.game.ListYourGames()
}

func (s *LobbyScene) OnExit() {}

func (s *LobbyScene) Update() error {
	if s.showCreate {
		s.createNameInput.Update()
		s.createPublicBtn.Update()
		s.createPrivateBtn.Update()
		s.createConfirmBtn.Update()
		s.createCancelBtn.Update()

		// Update button states
		s.createPublicBtn.Primary = s.createPublic
		s.createPrivateBtn.Primary = !s.createPublic

		// Update map generation buttons
		for i := 0; i < 3; i++ {
			s.mapSizeBtn[i].Update()
			s.mapSizeBtn[i].Primary = s.mapSize == i
			s.territoriesBtn[i].Update()
			s.territoriesBtn[i].Primary = s.territories == i
			s.islandsBtn[i].Update()
			s.islandsBtn[i].Primary = s.islands == i
			s.resourcesBtn[i].Update()
			s.resourcesBtn[i].Primary = s.resources == i
		}
		s.waterBorderBtn.Update()
		s.waterBorderBtn.Primary = s.waterBorder
		s.generateBtn.Update()
		s.regenerateBtn.Update()

		// Animate map generation (one territory at a time)
		if s.animating && s.generatedSteps != nil {
			s.animTicker++
			if s.animTicker >= 8 { // Show each territory for ~8 frames
				s.animTicker = 0
				if s.animStep < len(s.generatedSteps) {
					step := s.generatedSteps[s.animStep]
					if step.IsComplete {
						s.animating = false
					} else {
						// Apply all cells for this territory at once
						for _, cell := range step.Cells {
							x, y := cell[0], cell[1]
							if s.previewGrid != nil && y < len(s.previewGrid) && x < len(s.previewGrid[0]) {
								s.previewGrid[y][x] = step.TerritoryID
							}
						}
						s.animStep++
					}
				}
			}
		}

		// Disable create button until map is generated
		s.createConfirmBtn.Disabled = s.generatedMap == nil

		// Escape to close
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			s.showCreate = false
			s.animating = false
		}
		return nil
	}

	s.yourGameList.Update()
	s.gameList.Update()
	s.codeInput.Update()
	s.createBtn.Update()
	s.joinBtn.Update()
	s.deleteBtn.Update()
	s.joinCodeBtn.Update()
	s.refreshBtn.Update()

	// Auto-refresh every 5 seconds (300 frames at 60fps)
	s.refreshTimer++
	if s.refreshTimer >= 300 {
		s.refreshTimer = 0
		s.game.ListGames()
		s.game.ListYourGames()
	}

	// Update selected game from either list
	if id := s.yourGameList.GetSelectedID(); id != "" {
		s.selectedGame = id
		s.gameList.ClearSelection()
	} else if id := s.gameList.GetSelectedID(); id != "" {
		s.selectedGame = id
		s.yourGameList.ClearSelection()
	}

	// Disable join/delete if nothing selected
	s.joinBtn.Disabled = s.selectedGame == ""
	s.deleteBtn.Disabled = s.selectedGame == "" || !s.isCreator(s.selectedGame)

	return nil
}

func (s *LobbyScene) Draw(screen *ebiten.Image) {
	// Background stars
	for i := 0; i < 30; i++ {
		x := float32((i * 173) % ScreenWidth)
		y := float32((i * 127) % ScreenHeight)
		alpha := uint8(50 + (i % 100))
		starColor := color.RGBA{100, 150, 255, alpha}
		vector.DrawFilledCircle(screen, x, y, 1, starColor, false)
	}

	// Title panel
	titlePanel := DrawFancyPanel
	titlePanel(screen, 20, 20, ScreenWidth-40, 80, "")

	// Large title
	DrawLargeText(screen, "GAME LOBBY", 45, 35, ColorText)

	DrawText(screen, fmt.Sprintf("Welcome, %s!", s.game.config.PlayerName), 45, 65, ColorTextMuted)

	listY := 120
	hasYourGames := len(s.yourGames) > 0

	// Your games list (only if there are games)
	if hasYourGames {
		DrawFancyPanel(screen, 20, listY, 600, 220, "Your Active Games")
		s.yourGameList.Y = listY + 40
		s.yourGameList.H = 170
		s.yourGameList.Draw(screen)
		listY += 240
	}

	// Public games list
	publicGamesH := ScreenHeight - listY - 20
	if !hasYourGames {
		publicGamesH = ScreenHeight - listY - 20
	}

	DrawFancyPanel(screen, 20, listY, 600, publicGamesH, "Public Games")
	s.gameList.Y = listY + 40
	s.gameList.H = publicGamesH - 50
	s.gameList.Draw(screen)

	// Right side panel
	rightX := 640
	rightY := 120
	DrawFancyPanel(screen, rightX, rightY, 340, ScreenHeight-rightY-20, "Actions")

	// Buttons inside right panel
	buttonX := rightX + 20
	buttonY := rightY + 50

	s.createBtn.X = buttonX
	s.createBtn.Y = buttonY
	s.createBtn.W = 300
	s.createBtn.H = 50
	s.createBtn.Draw(screen)

	buttonY += 80
	DrawText(screen, "Join by code:", buttonX, buttonY, ColorTextMuted)
	s.codeInput.X = buttonX
	s.codeInput.Y = buttonY + 20
	s.codeInput.W = 200
	s.codeInput.H = 45
	s.codeInput.Draw(screen)

	s.joinCodeBtn.X = buttonX + 210
	s.joinCodeBtn.Y = buttonY + 20
	s.joinCodeBtn.W = 90
	s.joinCodeBtn.H = 45
	s.joinCodeBtn.Draw(screen)

	buttonY += 100
	s.refreshBtn.X = buttonX
	s.refreshBtn.Y = buttonY
	s.refreshBtn.W = 300
	s.refreshBtn.H = 50
	s.refreshBtn.Draw(screen)

	buttonY += 70
	s.joinBtn.X = buttonX
	s.joinBtn.Y = buttonY
	s.joinBtn.W = 300
	s.joinBtn.H = 50
	s.joinBtn.Draw(screen)

	buttonY += 70
	s.deleteBtn.X = buttonX
	s.deleteBtn.Y = buttonY
	s.deleteBtn.W = 300
	s.deleteBtn.H = 50
	s.deleteBtn.Draw(screen)

	// Create dialog overlay
	if s.showCreate {
		// Semi-transparent overlay
		vector.DrawFilledRect(screen, 0, 0, float32(ScreenWidth), float32(ScreenHeight),
			color.RGBA{0, 0, 0, 200}, false)

		// Larger dialog for map generation
		dialogW := 900
		dialogH := 600
		dialogX := (ScreenWidth - dialogW) / 2
		dialogY := (ScreenHeight - dialogH) / 2

		DrawFancyPanel(screen, dialogX, dialogY, dialogW, dialogH, "Create New Game")

		// Left side: Options
		optX := dialogX + 30
		optY := dialogY + 50
		optW := 280

		// Game name
		DrawText(screen, "Game Name:", optX, optY, ColorTextMuted)
		s.createNameInput.X = optX
		s.createNameInput.Y = optY + 20
		s.createNameInput.W = optW
		s.createNameInput.H = 35
		s.createNameInput.Draw(screen)

		optY += 70
		// Visibility
		DrawText(screen, "Visibility:", optX, optY, ColorTextMuted)
		btnW := (optW - 10) / 2
		s.createPublicBtn.X = optX
		s.createPublicBtn.Y = optY + 20
		s.createPublicBtn.W = btnW
		s.createPublicBtn.H = 32
		s.createPrivateBtn.X = optX + btnW + 10
		s.createPrivateBtn.Y = optY + 20
		s.createPrivateBtn.W = btnW
		s.createPrivateBtn.H = 32
		s.createPublicBtn.Draw(screen)
		s.createPrivateBtn.Draw(screen)

		optY += 65
		// Map Size
		DrawText(screen, "Map Size:", optX, optY, ColorTextMuted)
		btn3W := (optW - 20) / 3
		for i := 0; i < 3; i++ {
			s.mapSizeBtn[i].X = optX + i*(btn3W+10)
			s.mapSizeBtn[i].Y = optY + 20
			s.mapSizeBtn[i].W = btn3W
			s.mapSizeBtn[i].H = 32
			s.mapSizeBtn[i].Draw(screen)
		}

		optY += 65
		// Territories
		DrawText(screen, "Territories:", optX, optY, ColorTextMuted)
		for i := 0; i < 3; i++ {
			s.territoriesBtn[i].X = optX + i*(btn3W+10)
			s.territoriesBtn[i].Y = optY + 20
			s.territoriesBtn[i].W = btn3W
			s.territoriesBtn[i].H = 32
			s.territoriesBtn[i].Draw(screen)
		}

		optY += 65
		// Water Border
		s.waterBorderBtn.X = optX
		s.waterBorderBtn.Y = optY
		s.waterBorderBtn.W = optW
		s.waterBorderBtn.H = 32
		s.waterBorderBtn.Draw(screen)

		optY += 50
		// Islands
		DrawText(screen, "Islands:", optX, optY, ColorTextMuted)
		for i := 0; i < 3; i++ {
			s.islandsBtn[i].X = optX + i*(btn3W+10)
			s.islandsBtn[i].Y = optY + 20
			s.islandsBtn[i].W = btn3W
			s.islandsBtn[i].H = 32
			s.islandsBtn[i].Draw(screen)
		}

		optY += 65
		// Resources
		DrawText(screen, "Resources:", optX, optY, ColorTextMuted)
		for i := 0; i < 3; i++ {
			s.resourcesBtn[i].X = optX + i*(btn3W+10)
			s.resourcesBtn[i].Y = optY + 20
			s.resourcesBtn[i].W = btn3W
			s.resourcesBtn[i].H = 32
			s.resourcesBtn[i].Draw(screen)
		}

		optY += 65
		// Generate button
		s.generateBtn.X = optX
		s.generateBtn.Y = optY
		s.generateBtn.W = optW
		s.generateBtn.H = 40
		if s.generatedMap != nil {
			s.generateBtn.Text = "Regenerate"
		} else {
			s.generateBtn.Text = "Generate Map"
		}
		s.generateBtn.Draw(screen)

		// Right side: Map preview
		previewX := dialogX + 340
		previewY := dialogY + 50
		previewW := dialogW - 370
		previewH := dialogH - 150

		// Preview frame
		DrawPanel(screen, previewX, previewY, previewW, previewH)

		// Draw map preview
		if s.previewGrid != nil && s.previewWidth > 0 && s.previewHeight > 0 {
			cellSize := previewW / s.previewWidth
			cellSizeH := previewH / s.previewHeight
			if cellSizeH < cellSize {
				cellSize = cellSizeH
			}
			if cellSize < 2 {
				cellSize = 2
			}
			if cellSize > 15 {
				cellSize = 15
			}

			mapDrawW := s.previewWidth * cellSize
			mapDrawH := s.previewHeight * cellSize
			mapOffX := previewX + (previewW-mapDrawW)/2
			mapOffY := previewY + (previewH-mapDrawH)/2

			for y := 0; y < s.previewHeight; y++ {
				for x := 0; x < s.previewWidth; x++ {
					tid := s.previewGrid[y][x]
					var cellColor color.RGBA
					if tid == 0 {
						// Water
						cellColor = color.RGBA{30, 70, 140, 255}
					} else {
						// Territory - use ID for color
						cellColor = s.territoryColor(tid)
					}

					cx := float32(mapOffX + x*cellSize)
					cy := float32(mapOffY + y*cellSize)
					vector.DrawFilledRect(screen, cx, cy, float32(cellSize), float32(cellSize), cellColor, false)
				}
			}

			// Show territory count
			if s.generatedMap != nil {
				infoText := fmt.Sprintf("%d territories", len(s.generatedMap.Territories))
				DrawText(screen, infoText, previewX+10, previewY+previewH-25, ColorTextMuted)
			}
		} else {
			DrawTextCentered(screen, "Click 'Generate Map' to preview", previewX+previewW/2, previewY+previewH/2, ColorTextMuted)
		}

		// Bottom buttons
		bottomY := dialogY + dialogH - 60
		s.createConfirmBtn.X = dialogX + dialogW - 310
		s.createConfirmBtn.Y = bottomY
		s.createConfirmBtn.W = 140
		s.createConfirmBtn.H = 45
		s.createCancelBtn.X = dialogX + dialogW - 160
		s.createCancelBtn.Y = bottomY
		s.createCancelBtn.W = 140
		s.createCancelBtn.H = 45
		s.createConfirmBtn.Draw(screen)
		s.createCancelBtn.Draw(screen)
	}
}

// initMapGeneration resets map generation state
func (s *LobbyScene) initMapGeneration() {
	s.generatedMap = nil
	s.generatedSteps = nil
	s.previewGrid = nil
	s.animating = false
	s.animStep = 0
}

// onGenerateMap generates a new map with current options
func (s *LobbyScene) onGenerateMap() {
	opts := maps.GeneratorOptions{
		Size:        maps.MapSize(s.mapSize),
		Territories: maps.TerritoryCount(s.territories),
		WaterBorder: s.waterBorder,
		Islands:     maps.IslandAmount(s.islands),
		Resources:   maps.ResourceAmount(s.resources),
	}

	gen := maps.NewGenerator(opts)
	s.generatedMap, s.generatedSteps = gen.Generate()

	// Initialize preview grid for animation - starts as all water (0)
	s.previewWidth = s.generatedMap.Width
	s.previewHeight = s.generatedMap.Height
	s.previewGrid = make([][]int, s.previewHeight)
	for y := range s.previewGrid {
		s.previewGrid[y] = make([]int, s.previewWidth)
		// All cells start as water (0) - territories will be added on top
	}

	// Start animation
	s.animStep = 0
	s.animTicker = 0
	s.animating = true
}

// territoryColor returns a color for a territory based on its ID
func (s *LobbyScene) territoryColor(id int) color.RGBA {
	// Generate a nice color palette based on territory ID
	colors := []color.RGBA{
		{180, 100, 100, 255}, // Red
		{100, 180, 100, 255}, // Green
		{100, 100, 180, 255}, // Blue
		{180, 180, 100, 255}, // Yellow
		{180, 100, 180, 255}, // Purple
		{100, 180, 180, 255}, // Cyan
		{200, 140, 100, 255}, // Orange
		{140, 100, 200, 255}, // Violet
		{100, 200, 140, 255}, // Teal
		{200, 100, 140, 255}, // Pink
		{140, 200, 100, 255}, // Lime
		{100, 140, 200, 255}, // Sky
		{170, 130, 90, 255},  // Tan
		{130, 170, 90, 255},  // Olive
		{90, 130, 170, 255},  // Steel
		{170, 90, 130, 255},  // Rose
	}

	return colors[id%len(colors)]
}

func (s *LobbyScene) SetGameList(games []protocol.GameListItem) {
	s.games = games
	items := make([]ListItem, len(games))
	for i, g := range games {
		items[i] = ListItem{
			ID:      g.ID,
			Text:    g.Name,
			Subtext: fmt.Sprintf("%d/%d players", g.PlayerCount, g.MaxPlayers),
		}
	}
	// Preserve selection and scroll if this is an auto-refresh
	s.gameList.SetItemsPreserve(items, s.selectedGame)
}

func (s *LobbyScene) SetYourGames(games []protocol.GameListItem) {
	s.yourGames = games
	items := make([]ListItem, len(games))
	for i, g := range games {
		status := g.Status
		if g.IsYourTurn {
			status += " - YOUR TURN!"
		}
		items[i] = ListItem{
			ID:      g.ID,
			Text:    g.Name,
			Subtext: fmt.Sprintf("%s (%d/%d)", status, g.PlayerCount, g.MaxPlayers),
		}
	}
	// Preserve selection and scroll if this is an auto-refresh
	s.yourGameList.SetItemsPreserve(items, s.selectedGame)
}

func (s *LobbyScene) isCreator(gameID string) bool {
	// Check in your games list
	for _, g := range s.yourGames {
		if g.ID == gameID && g.HostName == s.game.config.PlayerName {
			return true
		}
	}
	return false
}

func (s *LobbyScene) onRefresh() {
	s.refreshTimer = 0 // Reset auto-refresh timer
	s.game.ListGames()
	s.game.ListYourGames()
}

func (s *LobbyScene) onJoinSelected() {
	if s.selectedGame != "" {
		s.game.JoinGame(s.selectedGame)
	}
}

func (s *LobbyScene) onDeleteGame() {
	if s.selectedGame != "" && s.isCreator(s.selectedGame) {
		s.game.DeleteGame(s.selectedGame)
		s.selectedGame = ""
		// Refresh lists
		s.onRefresh()
	}
}

func (s *LobbyScene) onJoinByCode() {
	code := s.codeInput.Text
	if code != "" {
		s.game.JoinByCode(code)
	}
}

func (s *LobbyScene) onCreateConfirm() {
	// Must have a generated map
	if s.generatedMap == nil {
		return
	}

	name := s.createNameInput.Text
	if name == "" {
		name = s.game.config.PlayerName + "'s Game"
	}

	settings := protocol.GameSettings{
		MaxPlayers:    4,
		GameLevel:     "expert",
		ChanceLevel:   "medium",
		VictoryCities: 3,
		MapID:         s.generatedMap.ID,
	}

	// Convert generated map to protocol MapData
	mapData := &protocol.MapData{
		ID:          s.generatedMap.ID,
		Name:        s.generatedMap.Name,
		Width:       s.generatedMap.Width,
		Height:      s.generatedMap.Height,
		Grid:        s.generatedMap.Grid,
		Territories: make(map[string]protocol.TerritoryInfo),
	}

	// Add territory info
	for id, t := range s.generatedMap.Territories {
		mapData.Territories[fmt.Sprintf("%d", id)] = protocol.TerritoryInfo{
			Name:     t.Name,
			Resource: t.Resource.String(),
		}
	}

	s.game.CreateGame(name, s.createPublic, settings, mapData)
	s.showCreate = false
	s.animating = false
}

// ==================== Waiting Scene ====================

// WaitingScene shows the game lobby while waiting for players.
type WaitingScene struct {
	game *Game

	playerList  *List
	addAIBtn    *Button
	readyBtn    *Button
	startBtn    *Button
	leaveBtn    *Button
	copyCodeBtn *Button

	// Host-only settings
	mapBtn      *Button
	settingsBtn *Button

	// Settings dialog
	showSettings      bool
	chanceLevelBtns   [3]*Button // Low, Medium, High
	victoryCitiesBtns [4]*Button // 3, 4, 5, 6
	maxPlayersBtns    [3]*Button // 2, 3, 4
	settingsCloseBtn  *Button
}

// NewWaitingScene creates a new waiting scene.
func NewWaitingScene(game *Game) *WaitingScene {
	s := &WaitingScene{game: game}

	s.playerList = NewList(50, 150, 400, 300)

	s.addAIBtn = &Button{
		X: 500, Y: 150, W: 180, H: 40,
		Text:    "Add AI Player",
		OnClick: func() { s.game.AddAI("aggressive") },
	}

	s.readyBtn = &Button{
		X: 500, Y: 200, W: 180, H: 40,
		Text:    "Ready",
		Primary: true,
		OnClick: s.onToggleReady,
	}

	s.startBtn = &Button{
		X: 500, Y: 260, W: 180, H: 40,
		Text:    "Start Game",
		Primary: true,
		OnClick: func() { s.game.StartGame() },
	}

	s.leaveBtn = &Button{
		X: 500, Y: 400, W: 180, H: 40,
		Text: "Leave Game",
		OnClick: func() {
			s.game.LeaveGame()
			s.game.SetScene(s.game.lobbyScene)
		},
	}

	s.copyCodeBtn = &Button{
		X: 50, Y: 520, W: 200, H: 40,
		Text: "Copy Join Code",
	}

	// Host-only buttons for map and settings
	s.mapBtn = &Button{
		X: 500, Y: 320, W: 180, H: 40,
		Text:    "Change Map",
		OnClick: func() { s.onChangeMap() },
	}

	s.settingsBtn = &Button{
		X: 500, Y: 370, W: 180, H: 40,
		Text:    "Settings",
		OnClick: func() { s.showSettings = true },
	}

	// Settings dialog buttons
	chanceLevels := []string{"Low", "Medium", "High"}
	for i, label := range chanceLevels {
		idx := i
		s.chanceLevelBtns[i] = &Button{
			Text: label,
			OnClick: func() {
				s.game.UpdateGameSettings("chanceLevel", chanceLevels[idx])
			},
		}
	}

	victoryCities := []string{"3", "4", "5", "6"}
	for i, label := range victoryCities {
		idx := i
		s.victoryCitiesBtns[i] = &Button{
			Text: label,
			OnClick: func() {
				s.game.UpdateGameSettings("victoryCities", victoryCities[idx])
			},
		}
	}

	maxPlayers := []string{"2", "3", "4"}
	for i, label := range maxPlayers {
		idx := i
		s.maxPlayersBtns[i] = &Button{
			Text: label,
			OnClick: func() {
				s.game.UpdateGameSettings("maxPlayers", maxPlayers[idx])
			},
		}
	}

	s.settingsCloseBtn = &Button{
		Text:    "Close",
		OnClick: func() { s.showSettings = false },
	}

	return s
}

func (s *WaitingScene) OnEnter() {}
func (s *WaitingScene) OnExit()  {}

func (s *WaitingScene) Update() error {
	// Handle settings dialog
	if s.showSettings {
		for _, btn := range s.chanceLevelBtns {
			btn.Update()
		}
		for _, btn := range s.victoryCitiesBtns {
			btn.Update()
		}
		for _, btn := range s.maxPlayersBtns {
			btn.Update()
		}
		s.settingsCloseBtn.Update()
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			s.showSettings = false
		}
		return nil
	}

	s.playerList.Update()
	s.readyBtn.Update()
	s.leaveBtn.Update()
	s.copyCodeBtn.Update()

	// Update based on lobby state
	if lobby := s.game.lobbyState; lobby != nil {
		// Update player list
		items := make([]ListItem, len(lobby.Players))
		for i, p := range lobby.Players {
			status := ""
			if p.IsAI {
				status = fmt.Sprintf("AI (%s)", p.AIPersonality)
			} else if p.Ready {
				status = "Ready"
			} else if p.IsConnected {
				status = "Connected"
			} else {
				status = "Disconnected"
			}
			items[i] = ListItem{
				ID:      p.ID,
				Text:    p.Name,
				Subtext: fmt.Sprintf("%s - %s", p.Color, status),
			}
		}
		s.playerList.SetItems(items)

		// Show/hide host-only buttons
		isHost := lobby.HostID == s.game.config.PlayerID
		s.startBtn.Disabled = !isHost || !s.canStart()
		s.addAIBtn.Disabled = !isHost || len(lobby.Players) >= lobby.Settings.MaxPlayers
		s.mapBtn.Disabled = !isHost
		s.settingsBtn.Disabled = !isHost

		s.addAIBtn.Update()
		s.startBtn.Update()
		s.mapBtn.Update()
		s.settingsBtn.Update()

		// Update ready button text
		for _, p := range lobby.Players {
			if p.ID == s.game.config.PlayerID {
				if p.Ready {
					s.readyBtn.Text = "Not Ready"
				} else {
					s.readyBtn.Text = "Ready"
				}
				break
			}
		}
	}

	return nil
}

func (s *WaitingScene) Draw(screen *ebiten.Image) {
	lobby := s.game.lobbyState
	if lobby == nil {
		DrawTextCentered(screen, "Loading...", ScreenWidth/2, ScreenHeight/2, ColorText)
		return
	}

	// Title
	DrawText(screen, lobby.GameName, 50, 50, ColorText)
	DrawText(screen, fmt.Sprintf("Join Code: %s", lobby.JoinCode), 50, 75, ColorPrimary)

	// Settings summary
	DrawText(screen, fmt.Sprintf("Players: %d/%d | Cities to win: %d",
		len(lobby.Players), lobby.Settings.MaxPlayers, lobby.Settings.VictoryCities),
		50, 100, ColorTextMuted)
	DrawText(screen, fmt.Sprintf("Level: %s | Chance: %s",
		lobby.Settings.GameLevel, lobby.Settings.ChanceLevel),
		50, 120, ColorTextMuted)

	// Player list
	DrawText(screen, "Players:", 50, 135, ColorTextMuted)
	s.playerList.Draw(screen)

	// Buttons
	isHost := lobby.HostID == s.game.config.PlayerID
	if isHost {
		s.addAIBtn.Draw(screen)
		s.startBtn.Draw(screen)
		s.mapBtn.Draw(screen)
		s.settingsBtn.Draw(screen)
	}
	s.readyBtn.Draw(screen)
	s.leaveBtn.Draw(screen)
	s.copyCodeBtn.Draw(screen)

	// Host indicator
	if isHost {
		DrawText(screen, "(You are the host)", 500, 130, ColorTextMuted)
	}

	// Settings dialog overlay
	if s.showSettings {
		s.drawSettingsDialog(screen, lobby)
	}
}

func (s *WaitingScene) drawSettingsDialog(screen *ebiten.Image, lobby *protocol.LobbyStatePayload) {
	// Semi-transparent overlay
	vector.DrawFilledRect(screen, 0, 0, float32(ScreenWidth), float32(ScreenHeight),
		color.RGBA{0, 0, 0, 200}, false)

	// Dialog panel
	dialogW := 400
	dialogH := 350
	dialogX := (ScreenWidth - dialogW) / 2
	dialogY := (ScreenHeight - dialogH) / 2

	DrawFancyPanel(screen, dialogX, dialogY, dialogW, dialogH, "Game Settings")

	y := dialogY + 50
	btnW := 80
	btnH := 35

	// Chance Level
	DrawText(screen, "Chance Level:", dialogX+20, y, ColorText)
	y += 25
	for i, btn := range s.chanceLevelBtns {
		btn.X = dialogX + 20 + i*(btnW+10)
		btn.Y = y
		btn.W = btnW
		btn.H = btnH
		btn.Primary = lobby.Settings.ChanceLevel == btn.Text
		btn.Draw(screen)
	}

	y += 55
	// Victory Cities
	DrawText(screen, "Cities to Win:", dialogX+20, y, ColorText)
	y += 25
	for i, btn := range s.victoryCitiesBtns {
		btn.X = dialogX + 20 + i*(60+10)
		btn.Y = y
		btn.W = 60
		btn.H = btnH
		btn.Primary = fmt.Sprintf("%d", lobby.Settings.VictoryCities) == btn.Text
		btn.Draw(screen)
	}

	y += 55
	// Max Players
	DrawText(screen, "Max Players:", dialogX+20, y, ColorText)
	y += 25
	for i, btn := range s.maxPlayersBtns {
		btn.X = dialogX + 20 + i*(60+10)
		btn.Y = y
		btn.W = 60
		btn.H = btnH
		btn.Primary = fmt.Sprintf("%d", lobby.Settings.MaxPlayers) == btn.Text
		btn.Draw(screen)
	}

	// Close button
	s.settingsCloseBtn.X = dialogX + dialogW/2 - 60
	s.settingsCloseBtn.Y = dialogY + dialogH - 55
	s.settingsCloseBtn.W = 120
	s.settingsCloseBtn.H = 40
	s.settingsCloseBtn.Draw(screen)
}

func (s *WaitingScene) onChangeMap() {
	// Go back to lobby scene with create dialog open
	s.game.LeaveGame()
	s.game.SetScene(s.game.lobbyScene)
	// Trigger the create game dialog
	s.game.lobbyScene.showCreate = true
}

func (s *WaitingScene) onToggleReady() {
	if lobby := s.game.lobbyState; lobby != nil {
		for _, p := range lobby.Players {
			if p.ID == s.game.config.PlayerID {
				s.game.SetReady(!p.Ready)
				break
			}
		}
	}
}

func (s *WaitingScene) canStart() bool {
	lobby := s.game.lobbyState
	if lobby == nil || len(lobby.Players) < 2 {
		return false
	}
	for _, p := range lobby.Players {
		if !p.IsAI && !p.Ready {
			return false
		}
	}
	return true
}
