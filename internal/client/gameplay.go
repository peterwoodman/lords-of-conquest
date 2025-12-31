package client

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// emptyImage is used for drawing triangles with solid colors
var emptyImage = func() *ebiten.Image {
	img := ebiten.NewImage(3, 3)
	img.Fill(color.White)
	return img.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)
}()

// GameplayScene handles the main game display and interaction.
type GameplayScene struct {
	game *Game

	// Game state data
	gameState    map[string]interface{}
	mapData      map[string]interface{}
	territories  map[string]interface{}
	players      map[string]interface{}
	playerOrder  []interface{}
	currentPhase string
	currentTurn  string
	round        int

	// Game history
	history       []HistoryEntry
	historyScroll int // Scroll offset for history panel

	// Rendering
	cellSize    int
	offsetX     int
	offsetY     int
	hoveredCell [2]int

	// UI
	infoPanel         *Panel
	actionPanel       *Panel
	endPhaseBtn       *Button
	selectedTerritory string // For multi-step actions like moving stockpile

	// Build menu (Development phase)
	showBuildMenu      bool
	buildMenuTerritory string
	buildCityBtn       *Button
	buildWeaponBtn     *Button
	buildBoatBtn       *Button
	cancelBuildBtn     *Button

	// Water body selection for boats
	showWaterBodySelect bool
	waterBodyOptions    []string // Water body IDs to choose from
	waterBodySelectBtns []*Button

	// Combat result display
	showCombatResult bool
	combatResult     *CombatResultData
	dismissResultBtn *Button

	// Attack planning (Conquest phase)
	showAttackPlan        bool
	attackPlanTarget      string             // Territory ID being attacked
	attackPreview         *AttackPreviewData // Preview from server
	selectedReinforcement *ReinforcementData // Selected unit to bring
	attackNoReinfBtn      *Button
	attackWithReinfBtn    *Button
	cancelAttackBtn       *Button
	loadHorseCheckbox     bool // For boats: load horse?
	loadWeaponCheckbox    bool // For boats: load weapon?
}

// HistoryEntry represents a single game history event for display.
type HistoryEntry struct {
	ID         int64
	Round      int
	Phase      string
	PlayerID   string
	PlayerName string
	EventType  string
	Message    string
}

// CombatResultData holds the result of a combat for display
type CombatResultData struct {
	AttackerWins    bool
	AttackStrength  int
	DefenseStrength int
	TargetTerritory string
	TargetName      string
}

// AttackPreviewData holds attack preview info from server
type AttackPreviewData struct {
	TargetTerritory string
	AttackStrength  int
	DefenseStrength int
	CanAttack       bool
	Reinforcements  []ReinforcementData
}

// ReinforcementData holds info about a unit that can join an attack
type ReinforcementData struct {
	UnitType       string
	FromTerritory  string
	WaterBodyID    string // For boats
	StrengthBonus  int
	CanCarryWeapon bool
	CanCarryHorse  bool
}

// Panel is a UI panel.
type Panel struct {
	X, Y, W, H int
}

// NewGameplayScene creates a new gameplay scene.
func NewGameplayScene(game *Game) *GameplayScene {
	s := &GameplayScene{
		game:        game,
		cellSize:    28,  // Slightly smaller to fit better
		offsetX:     260, // Leave room for left sidebar
		offsetY:     30,  // Top margin
		hoveredCell: [2]int{-1, -1},
	}

	// End phase button (positioned in drawBottomBar)
	s.endPhaseBtn = &Button{
		X: 0, Y: 0, W: 150, H: 45,
		Text:    "End Turn",
		Primary: true,
		OnClick: func() { s.game.EndPhase() },
	}

	// Build menu buttons
	s.buildCityBtn = &Button{
		X: 0, Y: 0, W: 200, H: 40,
		Text:    "Build City",
		OnClick: func() { s.doBuild("city") },
	}
	s.buildWeaponBtn = &Button{
		X: 0, Y: 0, W: 200, H: 40,
		Text:    "Build Weapon",
		OnClick: func() { s.doBuild("weapon") },
	}
	s.buildBoatBtn = &Button{
		X: 0, Y: 0, W: 200, H: 40,
		Text:    "Build Boat",
		OnClick: func() { s.doBuild("boat") },
	}
	s.cancelBuildBtn = &Button{
		X: 0, Y: 0, W: 200, H: 40,
		Text:    "Cancel",
		OnClick: func() { s.showBuildMenu = false },
	}

	// Combat result dismiss button
	s.dismissResultBtn = &Button{
		X: 0, Y: 0, W: 120, H: 40,
		Text:    "OK",
		Primary: true,
		OnClick: func() { s.showCombatResult = false },
	}

	// Attack planning buttons
	s.attackNoReinfBtn = &Button{
		X: 0, Y: 0, W: 160, H: 40,
		Text:    "Attack Without",
		OnClick: func() { s.doAttack(false) },
	}
	s.attackWithReinfBtn = &Button{
		X: 0, Y: 0, W: 160, H: 40,
		Text:    "Bring Unit",
		Primary: true,
		OnClick: func() { s.doAttack(true) },
	}
	s.cancelAttackBtn = &Button{
		X: 0, Y: 0, W: 100, H: 40,
		Text:    "Cancel",
		OnClick: func() { s.cancelAttackPlan() },
	}

	return s
}

func (s *GameplayScene) OnEnter() {
	s.gameState = nil
}

func (s *GameplayScene) OnExit() {}

func (s *GameplayScene) Update() error {
	// Only process input if we have map data
	if s.mapData == nil {
		return nil
	}

	// Handle combat result dialog
	if s.showCombatResult {
		s.dismissResultBtn.Update()
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			s.showCombatResult = false
		}
		return nil // Block other input while showing result
	}

	// Handle build menu
	if s.showBuildMenu {
		s.buildCityBtn.Update()
		s.buildWeaponBtn.Update()
		s.buildBoatBtn.Update()
		s.cancelBuildBtn.Update()
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			s.showBuildMenu = false
		}
		return nil // Block other input while showing menu
	}

	// Handle water body selection for boats
	if s.showWaterBodySelect {
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			s.showWaterBodySelect = false
			s.waterBodyOptions = nil
			s.buildMenuTerritory = ""
		}
		// Handle click on water cell
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			mx, my := ebiten.CursorPosition()
			cell := s.screenToGrid(mx, my)
			if cell[0] >= 0 {
				s.handleWaterBodyClick(cell[0], cell[1])
			}
		}
		return nil // Block other input while showing selection
	}

	// Handle attack planning dialog
	if s.showAttackPlan {
		s.attackNoReinfBtn.Update()
		s.cancelAttackBtn.Update()
		if s.selectedReinforcement != nil {
			s.attackWithReinfBtn.Update()
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			s.cancelAttackPlan()
		}
		return nil // Block other input while planning
	}
	// Update hovered cell
	mx, my := ebiten.CursorPosition()
	s.hoveredCell = s.screenToGrid(mx, my)

	// Update buttons
	isMyTurn := s.currentTurn == s.game.config.PlayerID
	showEndButton := isMyTurn && s.isActionPhase()
	s.endPhaseBtn.Disabled = !showEndButton
	if showEndButton {
		s.endPhaseBtn.Update()
	}

	// Handle click
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if s.hoveredCell[0] >= 0 {
			s.handleCellClick(s.hoveredCell[0], s.hoveredCell[1])
		}
	}

	// ESC to cancel selection
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		s.selectedTerritory = ""
	}

	return nil
}

// isActionPhase returns true if we're in a phase where the player can take actions and end their turn.
func (s *GameplayScene) isActionPhase() bool {
	return s.currentPhase == "Trade" || s.currentPhase == "Shipment" || s.currentPhase == "Conquest" || s.currentPhase == "Development"
}

func (s *GameplayScene) Draw(screen *ebiten.Image) {
	if s.mapData == nil {
		DrawLargeTextCentered(screen, "Loading game...", ScreenWidth/2, ScreenHeight/2, ColorText)
		return
	}

	// Background stars
	for i := 0; i < 40; i++ {
		x := float32((i * 167) % ScreenWidth)
		y := float32((i * 113) % ScreenHeight)
		alpha := uint8(30 + (i % 80))
		starColor := color.RGBA{100, 150, 255, alpha}
		vector.DrawFilledCircle(screen, x, y, 1, starColor, false)
	}

	// Left sidebar (You, Players, Resources)
	s.drawLeftSidebar(screen)

	// Map area with frame (fills remaining space)
	s.drawMapArea(screen)

	// Bottom info bar
	s.drawBottomBar(screen)

	// Draw hover info (includes attack preview during conquest)
	if s.hoveredCell[0] >= 0 {
		s.drawHoverInfo(screen)
	}

	// Draw build menu overlay
	if s.showBuildMenu {
		s.drawBuildMenu(screen)
	}

	// Draw water body selection overlay
	if s.showWaterBodySelect {
		s.drawWaterBodySelect(screen)
	}

	// Draw attack planning overlay
	if s.showAttackPlan {
		s.drawAttackPlan(screen)
	}
	// Draw combat result overlay
	if s.showCombatResult {
		s.drawCombatResult(screen)
	}
}
