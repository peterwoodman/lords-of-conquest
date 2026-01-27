package client

import (
	"fmt"
	"image/color"
	"math"
	"sort"
	"strings"

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

// Title screen timing (at 60fps) - using absolute frame positions on timeline
const (
	titlePhaseAnimating = 0
	titlePhaseDone      = 1

	// Timeline: [8bit] -> [zoom in] -> [fade+hold] -> [zoom out] -> [modern] -> done
	// With overlap, fade starts before zoom-in ends, zoom-out starts before fade ends
	titleStart8Bit    = 0
	titleStartZoomIn  = 10  // Start zooming in
	titleStartFade    = 130 // Start fading (overlaps with end of zoom-in)
	titleStartZoomOut = 190 // Start zooming out (overlaps with end of fade)
	titleStartModern  = 320 // Fully zoomed out, show modern
	titleEnd          = 350 // Transition to connect scene

	// Durations for smooth interpolation
	titleZoomInFrames  = 160 // How long zoom-in takes
	titleFadeFrames    = 100 // How long fade takes
	titleZoomOutFrames = 160 // How long zoom-out takes

	// Zoom parameters
	titleZoomFocusX = 0.75 // 75% on X axis
	titleZoomFocusY = 0.35 // 35% on Y axis
	titleZoomMax    = 5.0  // 500% zoom
)

// NewTitleScene creates a new title scene.
func NewTitleScene(game *Game) *TitleScene {
	return &TitleScene{game: game}
}

func (s *TitleScene) OnEnter() {
	s.timer = 0
	s.phase = titlePhaseAnimating
	s.fadeAlpha = 0
	s.zoomLevel = 1.0
	s.skipPressed = false

	// Start playing intro music
	PlayIntroMusic()
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

	// Single timeline approach - calculate zoom and fade based on current frame
	t := s.timer
	logMin := math.Log(1.0)
	logMax := math.Log(titleZoomMax)

	// Calculate zoom level based on timeline position
	if t < titleStartZoomIn {
		// Before zoom starts
		s.zoomLevel = 1.0
	} else if t < titleStartZoomIn+titleZoomInFrames {
		// Zooming in
		progress := float64(t-titleStartZoomIn) / float64(titleZoomInFrames)
		eased := progress * progress * (3 - 2*progress) // smoothstep
		s.zoomLevel = math.Exp(logMin + (logMax-logMin)*eased)
	} else if t < titleStartZoomOut {
		// Holding at max zoom
		s.zoomLevel = titleZoomMax
	} else if t < titleStartZoomOut+titleZoomOutFrames {
		// Zooming out
		progress := float64(t-titleStartZoomOut) / float64(titleZoomOutFrames)
		eased := progress * progress * (3 - 2*progress) // smoothstep
		s.zoomLevel = math.Exp(logMax - (logMax-logMin)*eased)
	} else {
		// After zoom ends
		s.zoomLevel = 1.0
	}

	// Calculate fade alpha based on timeline position
	if t < titleStartFade {
		// Before fade starts
		s.fadeAlpha = 0
	} else if t < titleStartFade+titleFadeFrames {
		// Fading
		progress := float64(t-titleStartFade) / float64(titleFadeFrames)
		s.fadeAlpha = progress * progress * (3 - 2*progress) // smoothstep for fade too
	} else {
		// After fade ends
		s.fadeAlpha = 1.0
	}

	// End of animation
	if t >= titleEnd {
		s.phase = titlePhaseDone
		s.game.connectScene.showTitleBackground = true
		s.game.SetScene(s.game.connectScene)
	}

	return nil
}

func (s *TitleScene) Draw(screen *ebiten.Image) {
	// Get title images
	img8Bit := GetTitleScreen8Bit()
	imgModern := GetTitleScreenModern()

	screenW := float64(ScreenWidth)
	screenH := float64(ScreenHeight)

	// Simple drawing based on current zoom and fade values
	// Draw 8-bit image (underneath, fades out as fadeAlpha increases)
	if img8Bit != nil && s.fadeAlpha < 1.0 {
		if s.zoomLevel > 1.01 {
			s.drawImageZoomed(screen, img8Bit, s.zoomLevel, 1.0-s.fadeAlpha)
		} else {
			s.drawImageFullScreenWithAlpha(screen, img8Bit, 1.0-s.fadeAlpha)
		}
	} else if img8Bit == nil && s.fadeAlpha < 1.0 {
		// Fallback if 8-bit image not loaded
		screen.Fill(color.RGBA{0, 0, 0, 255})
		DrawLargeTextCentered(screen, "LORDS OF CONQUEST", int(screenW)/2, int(screenH)/2-20, ColorPrimary)
		DrawTextCentered(screen, "(8-bit title screen)", int(screenW)/2, int(screenH)/2+20, ColorTextMuted)
	}

	// Draw modern image (on top, fades in as fadeAlpha increases)
	if imgModern != nil && s.fadeAlpha > 0 {
		if s.zoomLevel > 1.01 {
			s.drawImageZoomed(screen, imgModern, s.zoomLevel, s.fadeAlpha)
		} else {
			s.drawImageFullScreenWithAlpha(screen, imgModern, s.fadeAlpha)
		}
	} else if imgModern == nil && s.fadeAlpha >= 1.0 {
		screen.Fill(color.RGBA{20, 20, 40, 255})
		DrawLargeTextCentered(screen, "LORDS OF CONQUEST", int(screenW)/2, int(screenH)/2-20, ColorPrimary)
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

// Central server address
const CentralServerAddress = "lords-of-conquest.onrender.com"

// ConnectScene handles server connection and player name entry.
type ConnectScene struct {
	game *Game

	// Server selection
	useCentralServer bool
	centralBtn       *Button
	selfHostedBtn    *Button

	serverInput *TextInput
	nameInput   *TextInput
	connectBtn  *Button
	statusText  string
	connecting  bool

	// If true, show the modern title screen as background
	showTitleBackground bool

	// Connection error popup
	showConnectionPopup bool
	popupOkBtn          *Button
}

// NewConnectScene creates a new connect scene.
func NewConnectScene(game *Game) *ConnectScene {
	s := &ConnectScene{game: game, useCentralServer: true}

	s.centralBtn = &Button{
		Text: "Central Server",
		OnClick: func() {
			s.useCentralServer = true
		},
	}

	s.selfHostedBtn = &Button{
		Text: "Self-Hosted",
		OnClick: func() {
			s.useCentralServer = false
		},
	}

	s.serverInput = &TextInput{
		X: ScreenWidth/2 - 150, Y: 280,
		W: 300, H: 40,
		MaxLength: 100,
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

	// Initialize connection error popup button
	s.popupOkBtn = &Button{
		Text:    "OK",
		Primary: true,
		OnClick: func() {
			s.showConnectionPopup = false
			s.statusText = ""
		},
	}

	return s
}

func (s *ConnectScene) OnEnter() {
	// Load saved values
	if s.game.config.LastServer != "" {
		// Determine if last server was central or custom
		if s.game.config.LastServer == CentralServerAddress {
			s.useCentralServer = true
		} else {
			s.useCentralServer = false
			s.serverInput.Text = s.game.config.LastServer
		}
	} else {
		// Default to localhost if no saved server
		s.serverInput.Text = "localhost:30000"
	}
	if s.game.config.PlayerName != "" {
		s.nameInput.Text = s.game.config.PlayerName
	}
	s.connecting = false
	s.connectBtn.Disabled = false
	s.statusText = ""
	s.showConnectionPopup = false
}

func (s *ConnectScene) OnExit() {}

func (s *ConnectScene) Update() error {
	// Handle connection error popup
	if s.showConnectionPopup {
		s.popupOkBtn.Update()
		return nil
	}

	if s.connecting {
		return nil
	}

	s.centralBtn.Update()
	s.selfHostedBtn.Update()
	if !s.useCentralServer {
		s.serverInput.Update()
	}
	s.nameInput.Update()
	s.connectBtn.Update()

	// Enter key to connect
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) && !s.nameInput.IsFocused() && !s.serverInput.IsFocused() {
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

	// Server selection buttons
	inputY := panelY + 140
	DrawText(screen, "Server:", panelX+30, inputY, ColorTextMuted)

	// Central Server button
	s.centralBtn.X = panelX + 120
	s.centralBtn.Y = inputY - 5
	s.centralBtn.W = 150
	s.centralBtn.H = 35
	s.centralBtn.Primary = s.useCentralServer
	s.centralBtn.Draw(screen)

	// Self-Hosted button
	s.selfHostedBtn.X = panelX + 280
	s.selfHostedBtn.Y = inputY - 5
	s.selfHostedBtn.W = 150
	s.selfHostedBtn.H = 35
	s.selfHostedBtn.Primary = !s.useCentralServer
	s.selfHostedBtn.Draw(screen)

	// Server address input (only for self-hosted)
	inputY += 50
	if !s.useCentralServer {
		DrawText(screen, "Address:", panelX+30, inputY, ColorTextMuted)
		s.serverInput.X = panelX + 120
		s.serverInput.Y = inputY - 5
		s.serverInput.W = 310
		s.serverInput.H = 40
		s.serverInput.Draw(screen)
		inputY += 55
	}

	// Name input
	DrawText(screen, "Your Name:", panelX+30, inputY, ColorTextMuted)
	s.nameInput.X = panelX + 120
	s.nameInput.Y = inputY - 5
	s.nameInput.W = 310
	s.nameInput.H = 40
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

	// Connection error popup
	if s.showConnectionPopup {
		s.drawConnectionPopup(screen)
	}
}

func (s *ConnectScene) drawConnectionPopup(screen *ebiten.Image) {
	// Semi-transparent overlay
	vector.DrawFilledRect(screen, 0, 0, float32(ScreenWidth), float32(ScreenHeight),
		color.RGBA{0, 0, 0, 200}, false)

	// Popup panel
	panelW := 450
	panelH := 200
	panelX := (ScreenWidth - panelW) / 2
	panelY := (ScreenHeight - panelH) / 2

	DrawFancyPanel(screen, panelX, panelY, panelW, panelH, "Connection Failed")

	// Message lines
	msg1 := "The server may be sleeping."
	msg2 := "Please try again in 2 minutes."
	DrawTextCentered(screen, msg1, ScreenWidth/2, panelY+70, ColorText)
	DrawTextCentered(screen, msg2, ScreenWidth/2, panelY+95, ColorTextMuted)

	// OK button
	btnW := 100
	btnH := 40
	s.popupOkBtn.X = panelX + (panelW-btnW)/2
	s.popupOkBtn.Y = panelY + panelH - 60
	s.popupOkBtn.W = btnW
	s.popupOkBtn.H = btnH
	s.popupOkBtn.Draw(screen)
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
	var server string
	if s.useCentralServer {
		server = CentralServerAddress
	} else {
		server = s.serverInput.Text
		if server == "" {
			s.statusText = "Please enter a server address"
			return
		}
	}
	name := s.nameInput.Text

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
			s.connecting = false
			s.connectBtn.Disabled = false
			// Show popup for central server (may be sleeping), regular status for self-hosted
			if s.useCentralServer {
				s.showConnectionPopup = true
			} else {
				s.statusText = fmt.Sprintf("Connection failed: %v", err)
			}
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
	deleteBtn    *Button
	games        []protocol.GameListItem
	yourGames    []protocol.GameListItem
	selectedGame string

	// Map generation dialog (step 1)
	mapGenDialog *MapGenDialog

	// Create game details dialog (step 2)
	showCreateDetails bool
	createNameInput   *TextInput
	createPublicBtn   *Button
	createPrivateBtn  *Button
	createConfirmBtn  *Button
	createCancelBtn   *Button
	createPublic      bool

	// Auto-refresh timer (5 seconds at 60fps = 300 frames)
	refreshTimer int
}

// NewLobbyScene creates a new lobby scene.
func NewLobbyScene(game *Game) *LobbyScene {
	s := &LobbyScene{game: game}

	// Your games list (top half) - taller items for more info
	s.yourGameList = NewList(50, 100, 500, 180)
	s.yourGameList.itemHeight = 70 // Taller items to show player names
	s.yourGameList.OnSelect = func(id string) {
		s.selectedGame = id
		s.gameList.ClearSelection() // Clear the other list to avoid conflicting selections
	}

	// Public game list (bottom half)
	s.gameList = NewList(50, 320, 500, 280)
	s.gameList.OnSelect = func(id string) {
		s.selectedGame = id
		s.yourGameList.ClearSelection() // Clear the other list to avoid conflicting selections
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
		OnClick: func() {
			s.mapGenDialog.Show()
		},
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

	// Map generation dialog (step 1)
	s.mapGenDialog = NewMapGenDialog()
	s.mapGenDialog.WidthSlider.Value = 30
	s.mapGenDialog.TerritoriesSlider.Value = 40
	s.mapGenDialog.IslandsSlider.Value = 3
	s.mapGenDialog.ResourcesSlider.Value = 45
	s.mapGenDialog.OnConfirm = func(m *maps.Map) {
		// Move to step 2: hide map dialog, show game details dialog
		s.mapGenDialog.Hide()
		s.showCreateDetails = true
	}
	s.mapGenDialog.OnCancel = func() {
		s.mapGenDialog.Hide()
	}

	// Game details dialog (step 2)
	s.createNameInput = &TextInput{
		W: 300, H: 40,
		Placeholder: "Game name",
		MaxLength:   30,
	}

	s.createPublicBtn = &Button{
		Text: "Public", Primary: true,
		OnClick: func() { s.createPublic = true },
	}

	s.createPrivateBtn = &Button{
		Text:    "Private",
		OnClick: func() { s.createPublic = false },
	}

	s.createConfirmBtn = &Button{
		Text: "Create", Primary: true,
		OnClick: s.onCreateConfirm,
	}

	s.createCancelBtn = &Button{
		Text: "Cancel",
		OnClick: func() {
			s.showCreateDetails = false
			// Go back to map gen dialog
		},
	}

	s.createPublic = true

	return s
}

func (s *LobbyScene) OnEnter() {
	s.showCreateDetails = false
	s.mapGenDialog.Hide()
	s.refreshTimer = 0 // Reset auto-refresh timer
	s.game.ListGames()
	s.game.ListYourGames()
}

func (s *LobbyScene) OnExit() {}

func (s *LobbyScene) Update() error {
	// Step 1: Map generation dialog
	if s.mapGenDialog.Visible {
		s.mapGenDialog.Update()
		return nil
	}

	// Step 2: Game details dialog
	if s.showCreateDetails {
		s.createNameInput.Update()
		s.createPublicBtn.Update()
		s.createPrivateBtn.Update()
		s.createConfirmBtn.Update()
		s.createCancelBtn.Update()

		// Update button states
		s.createPublicBtn.Primary = s.createPublic
		s.createPrivateBtn.Primary = !s.createPublic

		// Escape to go back
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			s.showCreateDetails = false
		}
		return nil
	}

	s.yourGameList.Update()
	s.gameList.Update()
	s.codeInput.Update()

	// Auto-refresh every 5 seconds (300 frames at 60fps)
	s.refreshTimer++
	if s.refreshTimer >= 300 {
		s.refreshTimer = 0
		s.game.ListGames()
		s.game.ListYourGames()
	}

	// Selection synchronization is handled by OnSelect callbacks (mutual exclusion)

	// Update button disabled state BEFORE button Update() so they respond correctly
	s.joinBtn.Disabled = s.selectedGame == ""
	s.deleteBtn.Disabled = s.selectedGame == "" || !s.isCreator(s.selectedGame)

	// Now update buttons with correct disabled state
	s.createBtn.Update()
	s.joinBtn.Update()
	s.deleteBtn.Update()
	s.joinCodeBtn.Update()

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

	// Only show Join button if a game is selected
	if s.selectedGame != "" {
		buttonY += 100
		s.joinBtn.X = buttonX
		s.joinBtn.Y = buttonY
		s.joinBtn.W = 300
		s.joinBtn.H = 50
		s.joinBtn.Draw(screen)

		// Only show Delete button if user owns the selected game
		if s.isCreator(s.selectedGame) {
			buttonY += 70
			s.deleteBtn.X = buttonX
			s.deleteBtn.Y = buttonY
			s.deleteBtn.W = 300
			s.deleteBtn.H = 50
			s.deleteBtn.Draw(screen)
		}
	}

	// Step 1: Map generation dialog
	s.mapGenDialog.Draw(screen, "Create Game - Choose Map")

	// Step 2: Game details dialog
	if s.showCreateDetails {
		// Semi-transparent overlay
		vector.DrawFilledRect(screen, 0, 0, float32(ScreenWidth), float32(ScreenHeight),
			color.RGBA{0, 0, 0, 200}, false)

		dialogW := 400
		dialogH := 280
		dialogX := (ScreenWidth - dialogW) / 2
		dialogY := (ScreenHeight - dialogH) / 2

		DrawFancyPanel(screen, dialogX, dialogY, dialogW, dialogH, "Game Details")

		optX := dialogX + 30
		optY := dialogY + 50
		optW := dialogW - 60

		// Game name
		DrawText(screen, "Game Name:", optX, optY, ColorTextMuted)
		s.createNameInput.X = optX
		s.createNameInput.Y = optY + 25
		s.createNameInput.W = optW
		s.createNameInput.H = 40
		s.createNameInput.Draw(screen)

		optY += 85
		// Visibility
		DrawText(screen, "Visibility:", optX, optY, ColorTextMuted)
		btnW := (optW - 10) / 2
		s.createPublicBtn.X = optX
		s.createPublicBtn.Y = optY + 25
		s.createPublicBtn.W = btnW
		s.createPublicBtn.H = 40
		s.createPrivateBtn.X = optX + btnW + 10
		s.createPrivateBtn.Y = optY + 25
		s.createPrivateBtn.W = btnW
		s.createPrivateBtn.H = 40
		s.createPublicBtn.Draw(screen)
		s.createPrivateBtn.Draw(screen)

		// Bottom buttons
		bottomY := dialogY + dialogH - 60
		s.createConfirmBtn.X = dialogX + 30
		s.createConfirmBtn.Y = bottomY
		s.createConfirmBtn.W = (optW - 10) / 2
		s.createConfirmBtn.H = 45
		s.createCancelBtn.X = dialogX + 30 + (optW-10)/2 + 10
		s.createCancelBtn.Y = bottomY
		s.createCancelBtn.W = (optW - 10) / 2
		s.createCancelBtn.H = 45
		s.createConfirmBtn.Draw(screen)
		s.createCancelBtn.Draw(screen)
	}
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
		// Build status line
		statusLine := ""
		if g.Status == "started" && g.Round > 0 {
			// Show year and phase for started games
			phaseName := g.Phase
			// Make phase names more readable
			switch g.Phase {
			case "production":
				phaseName = "Production"
			case "trade":
				phaseName = "Trade"
			case "shipment":
				phaseName = "Shipment"
			case "conquest":
				phaseName = "Conquest"
			case "development":
				phaseName = "Development"
			case "selection":
				phaseName = "Selection"
			}
			statusLine = fmt.Sprintf("Year %d - %s", g.Round, phaseName)
			if g.IsYourTurn {
				statusLine += " (YOUR TURN!)"
			}
		} else {
			statusLine = fmt.Sprintf("%s (%d/%d)", g.Status, g.PlayerCount, g.MaxPlayers)
			if g.IsYourTurn {
				statusLine += " - YOUR TURN!"
			}
		}

		// Build player names string (sorted alphabetically for consistent display)
		playerStr := ""
		if len(g.PlayerNames) > 0 {
			sortedNames := make([]string, len(g.PlayerNames))
			copy(sortedNames, g.PlayerNames)
			sort.Strings(sortedNames)
			playerStr = strings.Join(sortedNames, ", ")
		}

		items[i] = ListItem{
			ID:      g.ID,
			Text:    g.Name,
			Subtext: statusLine,
			Detail:  playerStr,
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
		s.refreshTimer = 0
		s.game.ListGames()
		s.game.ListYourGames()
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
	generatedMap := s.mapGenDialog.GeneratedMap
	if generatedMap == nil {
		return
	}

	name := s.createNameInput.Text
	if name == "" {
		name = s.game.config.PlayerName + "'s Game"
	}

	settings := protocol.GameSettings{
		MaxPlayers:    protocol.DefaultMaxPlayers,
		ChanceLevel:   protocol.DefaultChanceLevel,
		VictoryCities: protocol.DefaultVictoryCities,
		MapID:         generatedMap.ID,
	}

	// Convert generated map to protocol MapData
	mapData := &protocol.MapData{
		ID:          generatedMap.ID,
		Name:        generatedMap.Name,
		Width:       generatedMap.Width,
		Height:      generatedMap.Height,
		Grid:        generatedMap.Grid,
		Territories: make(map[string]protocol.TerritoryInfo),
	}

	// Add territory info
	for id, t := range generatedMap.Territories {
		mapData.Territories[fmt.Sprintf("%d", id)] = protocol.TerritoryInfo{
			Name:     t.Name,
			Resource: t.Resource.String(),
		}
	}

	s.game.CreateGame(name, s.createPublic, settings, mapData)
	s.showCreateDetails = false
	s.mapGenDialog.Hide()
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

	// Map generation dialog (host only)
	mapGenDialog *MapGenDialog

	// Settings dialog
	showSettings        bool
	chanceLevelBtns     [3]*Button // Low, Medium, High
	victoryCitiesSlider *Slider
	maxPlayersSlider    *Slider
	settingsCloseBtn    *Button
}

// NewWaitingScene creates a new waiting scene.
func NewWaitingScene(game *Game) *WaitingScene {
	s := &WaitingScene{game: game}

	// Layout constants - 3-column layout: player list, map preview, actions
	leftMargin := 80
	rightPanelX := 920
	btnW := 170
	btnH := 45

	s.playerList = NewList(leftMargin, 210, 380, 410)

	s.addAIBtn = &Button{
		X: rightPanelX, Y: 210, W: btnW, H: btnH,
		Text:    "Add CPU",
		OnClick: func() { s.game.AddAI("aggressive") },
	}

	s.readyBtn = &Button{
		X: rightPanelX, Y: 270, W: btnW, H: btnH,
		Text:    "Ready",
		Primary: true,
		OnClick: s.onToggleReady,
	}

	s.startBtn = &Button{
		X: rightPanelX, Y: 330, W: btnW, H: btnH,
		Text:    "Start Game",
		Primary: true,
		OnClick: func() { s.game.StartGame() },
	}

	s.mapBtn = &Button{
		X: rightPanelX, Y: 390, W: btnW, H: btnH,
		Text: "Change Map",
		OnClick: func() {
			s.mapGenDialog.Show()
		},
	}

	s.settingsBtn = &Button{
		X: rightPanelX, Y: 450, W: btnW, H: btnH,
		Text: "Settings",
		OnClick: func() {
			// Sync slider values with current server state when opening dialog
			if lobby := s.game.lobbyState; lobby != nil {
				s.victoryCitiesSlider.Value = lobby.Settings.VictoryCities
				s.maxPlayersSlider.Value = lobby.Settings.MaxPlayers
			}
			s.showSettings = true
		},
	}

	s.leaveBtn = &Button{
		X: rightPanelX, Y: 560, W: btnW, H: btnH,
		Text: "Leave Game",
		OnClick: func() {
			s.game.LeaveGame()
			s.game.SetScene(s.game.lobbyScene)
		},
	}

	s.copyCodeBtn = &Button{
		X: leftMargin + 320, Y: 95, W: 120, H: 35,
		Text: "Copy Code",
		OnClick: func() {
			if lobby := s.game.lobbyState; lobby != nil {
				CopyToClipboard(lobby.JoinCode)
			}
		},
	}

	// Settings dialog buttons
	chanceLevels := []string{"low", "medium", "high"}
	chanceLevelLabels := []string{"Low", "Medium", "High"}
	for i, label := range chanceLevelLabels {
		idx := i
		s.chanceLevelBtns[i] = &Button{
			Text: label,
			OnClick: func() {
				s.game.UpdateGameSettings("chanceLevel", chanceLevels[idx])
			},
		}
	}

	s.victoryCitiesSlider = &Slider{
		Min:   protocol.MinVictoryCities,
		Max:   protocol.MaxVictoryCities,
		Value: protocol.DefaultVictoryCities,
		Label: "Cities to Win",
		OnChange: func(val int) {
			s.game.UpdateGameSettings("victoryCities", fmt.Sprintf("%d", val))
		},
	}

	s.maxPlayersSlider = &Slider{
		Min:   protocol.MinPlayers,
		Max:   protocol.MaxPlayers,
		Value: protocol.DefaultMaxPlayers,
		Label: "Max Players",
		OnChange: func(val int) {
			s.game.UpdateGameSettings("maxPlayers", fmt.Sprintf("%d", val))
		},
	}

	s.settingsCloseBtn = &Button{
		Text:    "Close",
		OnClick: func() { s.showSettings = false },
	}

	// Map generation dialog
	s.mapGenDialog = NewMapGenDialog()
	s.mapGenDialog.OnConfirm = func(m *maps.Map) {
		s.game.UpdateMap(m)
		s.mapGenDialog.Hide()
	}
	s.mapGenDialog.OnCancel = func() {
		s.mapGenDialog.Hide()
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
		s.victoryCitiesSlider.Update()
		s.maxPlayersSlider.Update()
		s.settingsCloseBtn.Update()
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			s.showSettings = false
		}
		return nil
	}

	// Handle map generation dialog
	if s.mapGenDialog.Visible {
		s.mapGenDialog.Update()
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

	leftMargin := 80
	mapPreviewX := 490
	rightPanelX := 920

	// Game name as title
	DrawLargeText(screen, lobby.GameName, leftMargin, 50, ColorText)

	// Join code - prominent display
	DrawLargeText(screen, fmt.Sprintf("Join Code: %s", lobby.JoinCode), leftMargin, 95, ColorPrimary)
	s.copyCodeBtn.Draw(screen)

	// Settings summary
	DrawText(screen, fmt.Sprintf("Players: %d/%d  |  Cities to win: %d  |  Chance: %s",
		len(lobby.Players), lobby.Settings.MaxPlayers, lobby.Settings.VictoryCities,
		lobby.Settings.ChanceLevel),
		leftMargin, 140, ColorTextMuted)

	// Player list panel (narrower to make room for map preview)
	DrawFancyPanel(screen, leftMargin-10, 170, 400, 470, "Players")
	s.playerList.Draw(screen)

	// Map preview panel (center, visible to all players)
	s.drawInlineMapPreview(screen, mapPreviewX, 170, 400, 470, lobby.MapData)

	// Right side - Actions panel
	isHost := lobby.HostID == s.game.config.PlayerID

	actionsPanelTitle := "Actions"
	if isHost {
		actionsPanelTitle = "Host Actions"
	}
	DrawFancyPanel(screen, rightPanelX-10, 170, 200, 470, actionsPanelTitle)

	if isHost {
		s.addAIBtn.Draw(screen)
		s.startBtn.Draw(screen)
		s.mapBtn.Draw(screen)
		s.settingsBtn.Draw(screen)
	} else {
		// Check if current player is ready
		playerReady := false
		for _, p := range lobby.Players {
			if p.ID == s.game.config.PlayerID {
				playerReady = p.Ready
				break
			}
		}
		if playerReady {
			DrawText(screen, "Waiting for host...", rightPanelX, 220, ColorTextMuted)
		} else {
			DrawText(screen, "Click Ready", rightPanelX, 220, ColorTextMuted)
		}
	}
	s.readyBtn.Draw(screen)
	s.leaveBtn.Draw(screen)

	// Settings dialog overlay
	if s.showSettings {
		s.drawSettingsDialog(screen, lobby)
	}

	// Map generation dialog overlay (host only)
	s.mapGenDialog.Draw(screen, "Change Map")
}

func (s *WaitingScene) drawSettingsDialog(screen *ebiten.Image, lobby *protocol.LobbyStatePayload) {
	// Semi-transparent overlay
	vector.DrawFilledRect(screen, 0, 0, float32(ScreenWidth), float32(ScreenHeight),
		color.RGBA{0, 0, 0, 200}, false)

	// Dialog panel
	dialogW := 400
	dialogH := 300
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
		btn.Primary = strings.EqualFold(lobby.Settings.ChanceLevel, btn.Text)
		btn.Draw(screen)
	}

	y += 55
	// Victory Cities slider
	s.victoryCitiesSlider.X = dialogX + 20
	s.victoryCitiesSlider.Y = y
	s.victoryCitiesSlider.W = dialogW - 40
	s.victoryCitiesSlider.H = 40
	s.victoryCitiesSlider.Draw(screen)

	y += 60
	// Max Players slider
	s.maxPlayersSlider.X = dialogX + 20
	s.maxPlayersSlider.Y = y
	s.maxPlayersSlider.W = dialogW - 40
	s.maxPlayersSlider.H = 40
	s.maxPlayersSlider.Draw(screen)

	// Close button
	s.settingsCloseBtn.X = dialogX + dialogW/2 - 60
	s.settingsCloseBtn.Y = dialogY + dialogH - 55
	s.settingsCloseBtn.W = 120
	s.settingsCloseBtn.H = 40
	s.settingsCloseBtn.Draw(screen)
}

func (s *WaitingScene) drawInlineMapPreview(screen *ebiten.Image, panelX, panelY, panelW, panelH int, mapData *protocol.MapData) {
	DrawFancyPanel(screen, panelX, panelY, panelW, panelH, "Map Preview")

	// Draw the map if we have map data
	if mapData != nil {
		// Calculate scale to fit the map in the panel
		mapAreaW := panelW - 40
		mapAreaH := panelH - 100
		mapAreaX := panelX + 20
		mapAreaY := panelY + 50

		// Calculate cell size based on map dimensions
		cellW := mapAreaW / mapData.Width
		cellH := mapAreaH / mapData.Height
		cellSize := cellW
		if cellH < cellW {
			cellSize = cellH
		}
		if cellSize < 2 {
			cellSize = 2
		}
		if cellSize > 12 {
			cellSize = 12
		}

		// Center the map
		actualW := cellSize * mapData.Width
		actualH := cellSize * mapData.Height
		offsetX := mapAreaX + (mapAreaW-actualW)/2
		offsetY := mapAreaY + (mapAreaH-actualH)/2

		// Draw the map grid
		for y := 0; y < mapData.Height; y++ {
			for x := 0; x < mapData.Width; x++ {
				cell := mapData.Grid[y][x]
				var c color.RGBA
				if cell == 0 {
					// Water
					c = color.RGBA{30, 60, 120, 255}
				} else {
					// Land - use territory ID to vary color slightly
					base := byte(80 + (cell*17)%40)
					c = color.RGBA{base, base + 30, base, 255}
				}
				vector.DrawFilledRect(screen,
					float32(offsetX+x*cellSize),
					float32(offsetY+y*cellSize),
					float32(cellSize),
					float32(cellSize),
					c, false)
			}
		}

		// Draw map info at bottom of panel
		DrawText(screen, fmt.Sprintf("Size: %dx%d  Territories: %d",
			mapData.Width, mapData.Height, len(mapData.Territories)),
			panelX+20, panelY+panelH-40, ColorTextMuted)
	} else {
		DrawTextCentered(screen, "Map data not available",
			panelX+panelW/2, panelY+panelH/2, ColorTextMuted)
	}
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
