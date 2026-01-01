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
	history            []HistoryEntry
	historyScroll      int    // Scroll offset for history panel
	historyPanelBounds [4]int // x, y, w, h for scroll detection

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

	// Alliance UI
	showAllyMenu        bool
	setAllyBtn          *Button
	allyNeutralBtn      *Button
	allyDefenderBtn     *Button
	allyAskBtn          *Button
	allyPlayerBtns      []*Button // Buttons for specific player allies
	allyPlayerIDs       []string  // Player IDs corresponding to allyPlayerBtns
	cancelAllyMenuBtn   *Button
	myAllianceSetting   string // Current alliance setting

	// Alliance request popup (when asked to join a battle)
	showAllyRequest       bool
	allyRequest           *AllianceRequestData
	allyRequestCountdown  int // Frames remaining
	supportAttackerBtn    *Button
	supportDefenderBtn    *Button
	stayNeutralBtn        *Button

	// Phase skip popup (queue to handle multiple skips)
	showPhaseSkip      bool
	phaseSkipPhase     string
	phaseSkipReason    string
	phaseSkipCountdown int // Frames remaining (30 seconds at 60fps = 1800)
	phaseSkipQueue     []PhaseSkipData // Queue for pending skip messages
	dismissSkipBtn     *Button

	// Victory screen
	showVictory      bool
	victoryWinnerID  string
	victoryWinnerName string
	victoryReason    string
	victoryTimer     int // Frames since victory started (for message transition)
	returnToLobbyBtn *Button

	// Shipment phase UI
	shipmentMode          string // "", "stockpile", "horse", "boat"
	shipmentFromTerritory string // Source territory for unit movement
	shipmentWaterBodyID   string // For boats: which water body
	shipmentCarryHorse    bool   // For boats: carry horse?
	shipmentCarryWeapon   bool   // For boats/horses: carry weapon?
	moveStockpileBtn      *Button
	moveHorseBtn          *Button
	moveBoatBtn           *Button
	cancelShipmentBtn     *Button
	shipmentConfirmBtn    *Button

	// Trade phase UI
	proposeTradeBtn     *Button
	showTradePropose    bool                         // Show propose trade popup
	showTradeIncoming   bool                         // Show incoming trade popup
	showTradeResult     bool                         // Show trade result popup
	waitingForTrade     bool                         // Waiting for trade response
	tradeProposal       *TradeProposalData           // Incoming proposal
	tradeResultAccepted bool
	tradeResultMessage  string
	tradeTargetPlayer   string                       // Selected target player for trade
	tradeOfferCoal      int
	tradeOfferGold      int
	tradeOfferIron      int
	tradeOfferTimber    int
	tradeOfferHorses    int
	tradeOfferHorseTerrs []string                    // Territories for horses being offered
	tradeRequestCoal    int
	tradeRequestGold    int
	tradeRequestIron    int
	tradeRequestTimber  int
	tradeRequestHorses  int
	tradeHorseDestTerrs []string                     // Where to place received horses
	tradeSendBtn        *Button
	tradeCancelBtn      *Button
	tradeAcceptBtn      *Button
	tradeRejectBtn      *Button
	tradeResultOkBtn    *Button
}

// TradeProposalData holds data for an incoming trade proposal.
type TradeProposalData struct {
	TradeID        string
	FromPlayerID   string
	FromPlayerName string
	OfferCoal      int
	OfferGold      int
	OfferIron      int
	OfferTimber    int
	OfferHorses    int
	RequestCoal    int
	RequestGold    int
	RequestIron    int
	RequestTimber  int
	RequestHorses  int
}

// AllianceRequestData holds data for an incoming alliance request.
type AllianceRequestData struct {
	BattleID      string
	AttackerID    string
	AttackerName  string
	DefenderID    string
	DefenderName  string
	TerritoryID   string
	TerritoryName string
	YourStrength  int
	TimeLimit     int
	ExpiresAt     int64
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
	TargetTerritory      string
	AttackStrength       int
	DefenseStrength      int
	AttackerAllyStrength int
	DefenderAllyStrength int
	CanAttack            bool
	Reinforcements       []ReinforcementData
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

// PhaseSkipData holds info about a skipped phase for the popup queue
type PhaseSkipData struct {
	Phase  string
	Reason string
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

	// Alliance menu buttons
	s.setAllyBtn = &Button{
		X: 0, Y: 0, W: 180, H: 30,
		Text:    "Set Ally",
		OnClick: func() { s.showAllyMenu = true },
	}
	s.allyNeutralBtn = &Button{
		X: 0, Y: 0, W: 200, H: 35,
		Text:    "Always Neutral",
		OnClick: func() { s.setAlliance("neutral") },
	}
	s.allyDefenderBtn = &Button{
		X: 0, Y: 0, W: 200, H: 35,
		Text:    "Always Defender",
		OnClick: func() { s.setAlliance("defender") },
	}
	s.allyAskBtn = &Button{
		X: 0, Y: 0, W: 200, H: 35,
		Text:    "Ask Each Time",
		OnClick: func() { s.setAlliance("ask") },
	}
	s.cancelAllyMenuBtn = &Button{
		X: 0, Y: 0, W: 200, H: 35,
		Text:    "Cancel",
		OnClick: func() { s.showAllyMenu = false },
	}
	s.myAllianceSetting = "ask" // Default

	// Alliance request popup buttons
	s.supportAttackerBtn = &Button{
		X: 0, Y: 0, W: 140, H: 40,
		Text:    "Support Attacker",
		OnClick: func() { s.voteAlliance("attacker") },
	}
	s.supportDefenderBtn = &Button{
		X: 0, Y: 0, W: 140, H: 40,
		Text:    "Support Defender",
		OnClick: func() { s.voteAlliance("defender") },
	}
	s.stayNeutralBtn = &Button{
		X: 0, Y: 0, W: 140, H: 40,
		Text:    "Stay Neutral",
		OnClick: func() { s.voteAlliance("neutral") },
	}

	// Phase skip popup button
	s.dismissSkipBtn = &Button{
		X: 0, Y: 0, W: 100, H: 40,
		Text: "OK",
		OnClick: func() {
			s.showNextPhaseSkip() // Show next in queue or close
		},
	}

	// Victory screen button
	s.returnToLobbyBtn = &Button{
		X: 0, Y: 0, W: 200, H: 50,
		Text:    "Return to Lobby",
		Primary: true,
		OnClick: func() {
			StopWinnerMusic()
			s.showVictory = false
			s.game.LeaveGame()
			s.game.SetScene(s.game.lobbyScene)
		},
	}

	// Shipment phase buttons
	s.moveStockpileBtn = &Button{
		X: 0, Y: 0, W: 200, H: 40,
		Text:    "Move Stockpile",
		OnClick: func() { s.startShipmentMode("stockpile") },
	}
	s.moveHorseBtn = &Button{
		X: 0, Y: 0, W: 200, H: 40,
		Text:    "Move Horse",
		OnClick: func() { s.startShipmentMode("horse") },
	}
	s.moveBoatBtn = &Button{
		X: 0, Y: 0, W: 200, H: 40,
		Text:    "Move Boat",
		OnClick: func() { s.startShipmentMode("boat") },
	}
	s.cancelShipmentBtn = &Button{
		X: 0, Y: 0, W: 200, H: 40,
		Text:    "Cancel",
		OnClick: func() { s.cancelShipmentMode() },
	}
	s.shipmentConfirmBtn = &Button{
		X: 0, Y: 0, W: 200, H: 40,
		Text:    "Confirm Move",
		Primary: true,
		OnClick: func() { s.confirmShipment() },
	}

	// Trade phase buttons
	s.proposeTradeBtn = &Button{
		X: 0, Y: 0, W: 150, H: 40,
		Text:    "Propose Trade",
		OnClick: func() { s.showTradePropose = true; s.resetTradeForm() },
	}
	s.tradeSendBtn = &Button{
		X: 0, Y: 0, W: 120, H: 40,
		Text:    "Send Offer",
		Primary: true,
		OnClick: func() { s.sendTradeOffer() },
	}
	s.tradeCancelBtn = &Button{
		X: 0, Y: 0, W: 100, H: 40,
		Text:    "Cancel",
		OnClick: func() { s.showTradePropose = false },
	}
	s.tradeAcceptBtn = &Button{
		X: 0, Y: 0, W: 100, H: 40,
		Text:    "Accept",
		Primary: true,
		OnClick: func() { s.acceptTrade() },
	}
	s.tradeRejectBtn = &Button{
		X: 0, Y: 0, W: 100, H: 40,
		Text:    "Reject",
		OnClick: func() { s.rejectTrade() },
	}
	s.tradeResultOkBtn = &Button{
		X: 0, Y: 0, W: 100, H: 40,
		Text:    "OK",
		Primary: true,
		OnClick: func() { s.showTradeResult = false },
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

	// Handle victory screen (takes priority over everything)
	if s.showVictory {
		s.victoryTimer++
		s.returnToLobbyBtn.Update()
		return nil // Block all other input
	}

	// Handle combat result dialog
	if s.showCombatResult {
		s.dismissResultBtn.Update()
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			s.showCombatResult = false
		}
		return nil // Block other input while showing result
	}

	// Handle phase skip popup
	if s.showPhaseSkip {
		s.dismissSkipBtn.Update()
		s.phaseSkipCountdown--
		if s.phaseSkipCountdown <= 0 {
			s.showNextPhaseSkip() // Show next in queue or close
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			s.showNextPhaseSkip() // Show next in queue or close
		}
		return nil // Block other input while showing popup
	}

	// Handle trade result popup
	if s.showTradeResult {
		s.tradeResultOkBtn.Update()
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			s.showTradeResult = false
		}
		return nil
	}

	// Handle incoming trade proposal popup
	if s.showTradeIncoming {
		s.tradeAcceptBtn.Update()
		s.tradeRejectBtn.Update()
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			s.rejectTrade()
		}
		return nil
	}

	// Handle trade proposal popup
	if s.showTradePropose {
		s.tradeSendBtn.Update()
		s.tradeCancelBtn.Update()
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			s.showTradePropose = false
		}
		return nil
	}

	// Handle waiting for trade response
	if s.waitingForTrade {
		// Block all input while waiting
		return nil
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

	// Handle alliance menu
	if s.showAllyMenu {
		s.allyNeutralBtn.Update()
		s.allyDefenderBtn.Update()
		s.allyAskBtn.Update()
		s.cancelAllyMenuBtn.Update()
		for _, btn := range s.allyPlayerBtns {
			btn.Update()
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			s.showAllyMenu = false
		}
		return nil // Block other input while showing menu
	}

	// Handle alliance request popup
	if s.showAllyRequest {
		s.supportAttackerBtn.Update()
		s.supportDefenderBtn.Update()
		s.stayNeutralBtn.Update()
		// Update countdown
		s.allyRequestCountdown--
		if s.allyRequestCountdown <= 0 {
			// Timeout - auto neutral
			s.voteAlliance("neutral")
		}
		return nil // Block other input while showing request
	}

	// Handle shipment phase controls (no blocking - controls are in status bar)
	if s.currentPhase == "Shipment" && s.currentTurn == s.game.config.PlayerID {
		s.moveStockpileBtn.Update()
		s.moveHorseBtn.Update()
		s.moveBoatBtn.Update()
		if s.shipmentMode != "" {
			s.shipmentConfirmBtn.Update()
			s.cancelShipmentBtn.Update()
			if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
				s.cancelShipmentMode()
			}
		}
	}

	// Update hovered cell
	mx, my := ebiten.CursorPosition()
	s.hoveredCell = s.screenToGrid(mx, my)

	// Handle mouse wheel scrolling for history panel
	_, dy := ebiten.Wheel()
	if dy != 0 {
		// Check if mouse is over history panel
		bounds := s.historyPanelBounds
		if mx >= bounds[0] && mx <= bounds[0]+bounds[2] &&
			my >= bounds[1] && my <= bounds[1]+bounds[3] {
			// Scroll the history panel
			if dy > 0 {
				s.historyScroll-- // Scroll up (show newer)
			} else {
				s.historyScroll++ // Scroll down (show older)
			}
			// Bounds clamping is done in drawHistoryPanel
		}
	}

	// Update buttons
	isMyTurn := s.currentTurn == s.game.config.PlayerID
	showEndButton := isMyTurn && s.isActionPhase()
	s.endPhaseBtn.Disabled = !showEndButton
	if showEndButton {
		s.endPhaseBtn.Update()
	}

	// Update trade button during trade phase
	if isMyTurn && s.currentPhase == "Trade" {
		s.proposeTradeBtn.Update()
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
	// Draw alliance menu overlay
	if s.showAllyMenu {
		s.drawAllyMenu(screen)
	}
	// Draw alliance request popup overlay
	if s.showAllyRequest {
		s.drawAllyRequest(screen)
	}
	// Draw trade popups
	if s.showTradePropose {
		s.drawTradePropose(screen)
	}
	if s.showTradeIncoming {
		s.drawTradeIncoming(screen)
	}
	if s.showTradeResult {
		s.drawTradeResult(screen)
	}
	if s.waitingForTrade {
		s.drawTradeWaiting(screen)
	}
	// Draw phase skip popup overlay
	if s.showPhaseSkip {
		s.drawPhaseSkip(screen)
	}
	// Draw victory screen overlay (last, on top of everything)
	if s.showVictory {
		s.drawVictoryScreen(screen)
	}
}
