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

// Toast notification timing constants (at 60 FPS)
const (
	ToastSlideInFrames  = 15  // 0.25 seconds
	ToastHoldFrames     = 90  // 1.5 seconds
	ToastSlideOutFrames = 15  // 0.25 seconds
)

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

	// Color block bounds for click detection
	myColorBlockBounds [4]int // x, y, w, h

	// Rendering
	cellSize    int
	offsetX     int
	offsetY     int
	hoveredCell [2]int

	// Pan and zoom
	zoom       float64 // 1.0 = 100%, 0.5 = 50%, 2.0 = 200%
	panX       int     // Pan offset in pixels
	panY       int
	isPanning  bool // True while right mouse button is held
	panStartX  int  // Mouse position when pan started
	panStartY  int
	panOffsetX int // Pan offset when pan started
	panOffsetY int

	// UI
	infoPanel         *Panel
	actionPanel       *Panel
	endPhaseBtn       *Button
	selectedTerritory string // For multi-step actions like moving stockpile

	// Development phase - select what to build first, then click territory
	selectedBuildType string // "city", "weapon", or "boat" (empty = none selected)
	buildUseGold      bool   // Toggle for using gold instead of resources
	devCityBtn        *Button
	devWeaponBtn      *Button
	devBoatBtn        *Button
	devUseGoldBtn     *Button

	// Water body selection for boats (when territory touches multiple water bodies)
	buildMenuTerritory string // Territory where we're building (for water body selection)

	// Water body selection for boats
	showWaterBodySelect bool
	waterBodyOptions    []string // Water body IDs to choose from
	waterBodySelectBtns []*Button

	// Combat result display
	showCombatResult  bool
	combatResult      *CombatResultData
	combatResultQueue []*CombatResultData // Queue of combat results to show
	dismissResultBtn  *Button

	// Combat animation
	showCombatAnimation   bool
	combatAnimTerritory   string                 // Territory being attacked
	combatAnimExplosions  []CombatExplosion      // Active explosions
	combatAnimTimer       int                    // Frames remaining
	combatAnimMaxDuration int                    // Total animation duration
	combatPendingResult   *CombatResultData      // Result to show after animation
	combatPendingState    map[string]interface{} // Game state to apply after animation

	// Production animation
	showProductionAnim     bool
	productionAnimData     *ProductionAnimData
	productionAnimTimer    int
	productionAnimIndex    int     // Which production item we're currently animating
	productionAnimProgress float64 // 0.0 to 1.0 for current item

	// Stockpile capture animation
	showStockpileCapture     bool
	stockpileCaptureData     *StockpileCaptureData
	stockpileCaptureTimer    int
	stockpileCaptureIndex    int     // Which resource we're animating
	stockpileCaptureProgress float64 // 0.0 to 1.0 for current resource

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
	showAllyMenu      bool
	setAllyBtn        *Button
	allyNeutralBtn    *Button
	allyDefenderBtn   *Button
	allyAskBtn        *Button
	allyPlayerBtns    []*Button // Buttons for specific player allies
	allyPlayerIDs     []string  // Player IDs corresponding to allyPlayerBtns
	cancelAllyMenuBtn *Button
	myAllianceSetting string // Current alliance setting

	// Alliance request popup (when asked to join a battle)
	showAllyRequest      bool
	allyRequest          *AllianceRequestData
	allyRequestCountdown int // Frames remaining
	supportAttackerBtn   *Button
	supportDefenderBtn   *Button
	stayNeutralBtn       *Button

	// Phase skip popup (queue to handle multiple skips)
	showPhaseSkip      bool
	phaseSkipEventID   string // Current skip's event ID for acknowledgment
	phaseSkipPhase     string
	phaseSkipReason    string
	phaseSkipCountdown int             // Frames remaining (30 seconds at 60fps = 1800)
	phaseSkipQueue     []PhaseSkipData // Queue for pending skip messages
	dismissSkipBtn     *Button

	// Victory screen
	showVictory       bool
	victoryWinnerID   string
	victoryWinnerName string
	victoryReason     string
	victoryTimer      int // Frames since victory started (for message transition)
	returnToLobbyBtn  *Button

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

	// Color picker UI
	showColorPicker bool
	colorPickerBtns []*Button
	cancelColorBtn  *Button
	usedColors      map[string]bool // Colors already used by other players

	// Trade phase UI
	proposeTradeBtn            *Button
	showTradePropose           bool               // Show propose trade popup
	showTradeIncoming          bool               // Show incoming trade popup
	showTradeResult            bool               // Show trade result popup
	waitingForTrade            bool               // Waiting for trade response
	tradeProposal              *TradeProposalData // Incoming proposal
	tradeResultAccepted        bool
	tradeResultMessage         string
	tradeTargetPlayer          string // Selected target player for trade
	tradeOfferCoal             int
	tradeOfferGold             int
	tradeOfferIron             int
	tradeOfferTimber           int
	tradeOfferHorses           int
	tradeOfferHorseTerrs       []string // Territories for horses being offered
	tradeRequestCoal           int
	tradeRequestGold           int
	tradeRequestIron           int
	tradeRequestTimber         int
	tradeRequestHorses         int
	tradeRequestHorseDestTerrs []string // Where proposer wants to receive requested horses
	tradeHorseDestTerrs        []string // Where accepter wants to place offered horses (incoming trade)
	tradeHorseSourceTerrs      []string // Which territories accepter gives horses FROM (incoming trade with RequestHorses)
	tradeSendBtn               *Button
	tradeCancelBtn             *Button
	tradeAcceptBtn             *Button
	tradeRejectBtn             *Button
	tradeResultOkBtn           *Button

	// Pending horse selection (after trade dialog closes, select on map)
	// "offer" = proposer selecting horses to give
	// "request" = proposer selecting where to receive requested horses
	// "receive" = accepter selecting where to place offered horses
	// "give" = accepter selecting which horses to give (when horses are requested from them)
	pendingHorseSelection string
	pendingHorseCount     int // How many territories to select
	horseConfirmBtn       *Button
	horseCancelBtn        *Button

	// Turn toast notification
	showTurnToast   bool   // Whether to show the toast
	turnToastTimer  int    // Frames elapsed in current phase
	turnToastPhase  string // "slide-in", "hold", "slide-out"
	initialTurnLoad bool   // Track if this is the first state load
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

// CombatExplosion represents a single explosion effect in the combat animation.
type CombatExplosion struct {
	X, Y      int     // Grid position
	OffsetX   float32 // Random offset within cell
	OffsetY   float32 // Random offset within cell
	Frame     int     // Current animation frame
	MaxFrames int     // Total frames for this explosion
}

// CombatResultData holds the result of a combat for display
type CombatResultData struct {
	EventID         string // For sync acknowledgment
	AttackerID      string
	AttackerWins    bool
	AttackStrength  int
	DefenseStrength int
	TargetTerritory string
	TargetName      string
	// Stockpile capture info
	StockpileCaptured     bool
	CapturedCoal          int
	CapturedGold          int
	CapturedIron          int
	CapturedTimber        int
	CapturedFromTerritory string
}

// ProductionAnimData holds production animation data from server
type ProductionAnimData struct {
	EventID                string
	Productions            []ProductionItem
	StockpileTerritoryID   string
	StockpileTerritoryName string
}

// StockpileCaptureData holds data for stockpile capture animation
type StockpileCaptureData struct {
	FromTerritoryID   string
	ToTerritoryID     string // Player's stockpile territory
	Resources         []CapturedResource
	PendingEventID    string // Combat event ID to acknowledge when done
	PendingCombatData *CombatResultData
}

// CapturedResource represents a resource being transferred
type CapturedResource struct {
	ResourceType string // "Coal", "Gold", "Iron", "Timber"
	Amount       int
}

// ProductionItem represents a single production event for animation
type ProductionItem struct {
	TerritoryID     string
	TerritoryName   string
	ResourceType    string // coal, gold, iron, timber, horse
	Amount          int
	DestinationID   string // For horses
	DestinationName string
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
	UnitType            string
	FromTerritory       string
	WaterBodyID         string // For boats
	StrengthBonus       int
	CanCarryWeapon      bool
	WeaponStrengthBonus int // Strength added if weapon is loaded (0 if already in range)
	CanCarryHorse       bool
	HorseStrengthBonus  int // Strength added if horse is loaded (0 if already in range)
}

// PhaseSkipData holds info about a skipped phase for the popup queue
type PhaseSkipData struct {
	EventID string
	Phase   string
	Reason  string
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
		zoom:        1.0, // Default zoom level
	}

	// End phase button (positioned in drawBottomBar)
	s.endPhaseBtn = &Button{
		X: 0, Y: 0, W: 150, H: 45,
		Text:    "End Turn",
		Primary: true,
		OnClick: func() { s.game.EndPhase() },
	}

	// Development phase build selection buttons (shown in status bar)
	s.devCityBtn = &Button{
		Text: "City",
		OnClick: func() {
			if s.selectedBuildType == "city" {
				s.selectedBuildType = "" // Deselect
			} else {
				s.selectedBuildType = "city"
			}
		},
	}
	s.devWeaponBtn = &Button{
		Text: "Weapon",
		OnClick: func() {
			if s.selectedBuildType == "weapon" {
				s.selectedBuildType = ""
			} else {
				s.selectedBuildType = "weapon"
			}
		},
	}
	s.devBoatBtn = &Button{
		Text: "Boat",
		OnClick: func() {
			if s.selectedBuildType == "boat" {
				s.selectedBuildType = ""
			} else {
				s.selectedBuildType = "boat"
			}
		},
	}
	s.devUseGoldBtn = &Button{
		Text: "[ ] Use Gold",
		OnClick: func() {
			s.buildUseGold = !s.buildUseGold
		},
	}

	// Combat result dismiss button
	s.dismissResultBtn = &Button{
		X: 0, Y: 0, W: 120, H: 40,
		Text:    "OK",
		Primary: true,
		OnClick: func() { s.dismissCombatResult() },
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

	// Horse selection buttons
	s.horseConfirmBtn = &Button{
		X: 0, Y: 0, W: 100, H: 35,
		Text:    "Confirm",
		Primary: true,
		OnClick: func() { s.confirmHorseSelection() },
	}
	s.horseCancelBtn = &Button{
		X: 0, Y: 0, W: 100, H: 35,
		Text:    "Cancel",
		OnClick: func() { s.cancelHorseSelection() },
	}

	// Color picker buttons
	s.cancelColorBtn = &Button{
		X: 0, Y: 0, W: 100, H: 35,
		Text:    "Cancel",
		OnClick: func() { s.showColorPicker = false },
	}
	s.usedColors = make(map[string]bool)

	return s
}

func (s *GameplayScene) OnEnter() {
	s.gameState = nil
	s.initialTurnLoad = true // First state load - don't show toast
	s.showTurnToast = false
}

func (s *GameplayScene) OnExit() {}

func (s *GameplayScene) Update() error {
	// Only process input if we have map data
	if s.mapData == nil {
		return nil
	}

	// Update turn toast animation (runs even during other animations)
	if s.showTurnToast {
		s.turnToastTimer++
		switch s.turnToastPhase {
		case "slide-in":
			if s.turnToastTimer >= ToastSlideInFrames {
				s.turnToastPhase = "hold"
				s.turnToastTimer = 0
			}
		case "hold":
			if s.turnToastTimer >= ToastHoldFrames {
				s.turnToastPhase = "slide-out"
				s.turnToastTimer = 0
			}
		case "slide-out":
			if s.turnToastTimer >= ToastSlideOutFrames {
				s.showTurnToast = false
			}
		}
	}

	// Always handle pan/zoom - even during animations
	s.updatePanZoom()

	// Handle victory screen (takes priority over everything)
	if s.showVictory {
		s.victoryTimer++
		s.returnToLobbyBtn.Update()
		return nil // Block all other input
	}

	// Handle production animation
	if s.showProductionAnim {
		s.updateProductionAnimation()
		return nil // Block all input during animation
	}

	// Handle stockpile capture animation
	if s.showStockpileCapture {
		s.updateStockpileCaptureAnimation()
		return nil // Block all input during animation
	}

	// Handle combat animation
	if s.showCombatAnimation {
		s.updateCombatAnimation()
		return nil // Block all input during animation
	}

	// Handle combat result dialog
	if s.showCombatResult {
		s.dismissResultBtn.Update()
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			s.dismissCombatResult()
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

	// Handle pending horse selection on map (after trade dialog closed)
	if s.pendingHorseSelection != "" {
		// Update buttons
		s.horseCancelBtn.Update()
		if s.isHorseSelectionComplete() {
			s.horseConfirmBtn.Update()
		}

		// Handle map clicks to select/deselect territories
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			mx, my := ebiten.CursorPosition()
			cell := s.screenToGrid(mx, my)
			if cell[0] >= 0 {
				terrID := s.getTerritoryAt(cell[0], cell[1])
				if terrID != "" {
					switch s.pendingHorseSelection {
					case "offer":
						s.handleOfferHorseClick(terrID)
					case "request":
						s.handleRequestHorseClick(terrID)
					case "receive":
						s.handleReceiveHorseClick(terrID)
					case "give":
						s.handleGiveHorseClick(terrID)
					}
				}
			}
		}

		// ESC to cancel
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			s.cancelHorseSelection()
		}

		// Enter to confirm when selection is complete (keyboard shortcut)
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) && s.isHorseSelectionComplete() {
			s.confirmHorseSelection()
		}

		return nil
	}

	// Handle color picker popup
	if s.showColorPicker {
		s.cancelColorBtn.Update()
		for _, btn := range s.colorPickerBtns {
			btn.Update()
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			s.showColorPicker = false
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

		// Handle clicks
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			mx, my := ebiten.CursorPosition()
			onlinePlayers := s.getOnlinePlayers()

			panelW, panelH := 560, 420
			centerX, centerY := ScreenWidth/2, ScreenHeight/2
			panelX, panelY := centerX-panelW/2, centerY-panelH/2

			// Y positions must match Draw function exactly
			// Draw: y = panelY + 50, then y += 25 for "Trade with:" label
			playerBtnY := panelY + 50 + 25 // Where player buttons start

			// Player selection
			for i, playerID := range onlinePlayers {
				btnX := panelX + 20 + (i%3)*150
				btnY := playerBtnY + (i/3)*35

				if mx >= btnX && mx < btnX+140 && my >= btnY && my < btnY+30 {
					s.tradeTargetPlayer = playerID
					break
				}
			}

			// Resource adjusters - calculate Y positions to match Draw exactly
			// Draw: y += playerRows*35 + 20 (for "I OFFER:" label), then y += 25 (for adjusters)
			playerRows := (len(onlinePlayers) + 2) / 3
			offerY := panelY + 50 + 25 + playerRows*35 + 20 + 25 // = panelY + 120 + playerRows*35

			myCoal, myGold, myIron, myTimber := s.getMyStockpile()
			myHorses := s.countPlayerHorses(s.game.config.PlayerID)

			s.handleResourceAdjusterClick(mx, my, panelX+20, offerY, &s.tradeOfferCoal, 0, myCoal)
			s.handleResourceAdjusterClick(mx, my, panelX+120, offerY, &s.tradeOfferGold, 0, myGold)
			s.handleResourceAdjusterClick(mx, my, panelX+220, offerY, &s.tradeOfferIron, 0, myIron)
			s.handleResourceAdjusterClick(mx, my, panelX+320, offerY, &s.tradeOfferTimber, 0, myTimber)
			s.handleResourceAdjusterClick(mx, my, panelX+420, offerY, &s.tradeOfferHorses, 0, myHorses)

			// "I WANT" adjusters (only if target selected)
			// Draw: y += 70 (for "I WANT:" label), then y += 25 (for adjusters)
			if s.tradeTargetPlayer != "" {
				wantY := offerY + 70 + 25 // = panelY + 215 + playerRows*35
				targetCoal, targetGold, targetIron, targetTimber := s.getPlayerStockpile(s.tradeTargetPlayer)
				targetHorses := s.countPlayerHorses(s.tradeTargetPlayer)

				s.handleResourceAdjusterClick(mx, my, panelX+20, wantY, &s.tradeRequestCoal, 0, targetCoal)
				s.handleResourceAdjusterClick(mx, my, panelX+120, wantY, &s.tradeRequestGold, 0, targetGold)
				s.handleResourceAdjusterClick(mx, my, panelX+220, wantY, &s.tradeRequestIron, 0, targetIron)
				s.handleResourceAdjusterClick(mx, my, panelX+320, wantY, &s.tradeRequestTimber, 0, targetTimber)
				s.handleResourceAdjusterClick(mx, my, panelX+420, wantY, &s.tradeRequestHorses, 0, targetHorses)
			}
		}

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

	// Handle development phase build buttons (in status bar)
	if s.currentPhase == "Development" && s.currentTurn == s.game.config.PlayerID {
		s.devCityBtn.Update()
		s.devWeaponBtn.Update()
		s.devBoatBtn.Update()
		s.devUseGoldBtn.Update()
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			s.selectedBuildType = ""
			s.buildUseGold = false
		}
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
		// Handle reinforcement selection clicks
		s.updateAttackPlanInput()

		// Only allow attack without reinforcement if base strength > 0
		if s.attackPreview != nil && s.attackPreview.AttackStrength > 0 {
			s.attackNoReinfBtn.Update()
		}
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

	// Update Set Ally button (available anytime with 3+ players)
	if len(s.playerOrder) >= 3 {
		s.setAllyBtn.Update()
	}

	// Handle click
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		// Check if clicked on color block
		if s.isClickInBounds(mx, my, s.myColorBlockBounds) {
			s.openColorPicker()
		} else if s.hoveredCell[0] >= 0 {
			s.handleCellClick(s.hoveredCell[0], s.hoveredCell[1])
		}
	}

	// ESC to cancel selection or leave game
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if s.selectedTerritory != "" || s.selectedBuildType != "" || s.shipmentMode != "" {
			// Cancel current selection/mode
			s.selectedTerritory = ""
			s.selectedBuildType = ""
			s.shipmentMode = ""
			s.shipmentFromTerritory = ""
		} else {
			// No selection - leave game and return to lobby
			s.game.LeaveGame()
			s.game.SetScene(s.game.lobbyScene)
		}
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

	// Map area first (so it goes behind UI elements when panned/zoomed)
	s.drawMapArea(screen)

	// Draw map-based animations (behind UI, on top of map)
	if s.showCombatAnimation {
		s.drawCombatAnimation(screen)
	}
	if s.showProductionAnim {
		s.drawProductionAnimation(screen)
	}
	if s.showStockpileCapture {
		s.drawStockpileCaptureAnimation(screen)
	}

	// Left sidebar on top of map and animations
	s.drawLeftSidebar(screen)

	// Bottom info bar on top of map and animations
	s.drawBottomBar(screen)

	// Draw hover info (includes attack preview during conquest)
	// Hide when attack dialogs/animations are showing
	if s.hoveredCell[0] >= 0 && !s.showAttackPlan && !s.showCombatAnimation && !s.showCombatResult {
		s.drawHoverInfo(screen)
	}

	// Draw turn toast notification (on top of UI, below modals)
	s.drawTurnToast(screen)

	// Draw modal overlays (on top of everything)
	if s.showWaterBodySelect {
		s.drawWaterBodySelect(screen)
	}
	if s.showAttackPlan {
		s.drawAttackPlan(screen)
	}
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
	// Draw color picker popup
	if s.showColorPicker {
		s.drawColorPicker(screen)
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

// updatePanZoom handles map panning and zooming input
func (s *GameplayScene) updatePanZoom() {
	mx, my := ebiten.CursorPosition()

	// Handle right mouse button for panning
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) {
		if !s.isPanning {
			// Start panning
			s.isPanning = true
			s.panStartX = mx
			s.panStartY = my
			s.panOffsetX = s.panX
			s.panOffsetY = s.panY
		} else {
			// Continue panning
			s.panX = s.panOffsetX + (mx - s.panStartX)
			s.panY = s.panOffsetY + (my - s.panStartY)
		}
	} else {
		s.isPanning = false
	}

	// Handle mouse wheel for zooming (only outside history panel)
	_, dy := ebiten.Wheel()
	if dy != 0 {
		bounds := s.historyPanelBounds
		if mx >= bounds[0] && mx <= bounds[0]+bounds[2] &&
			my >= bounds[1] && my <= bounds[1]+bounds[3] {
			// Scroll the history panel
			if dy > 0 {
				s.historyScroll--
			} else {
				s.historyScroll++
			}
		} else {
			// Zoom the map
			oldZoom := s.zoom
			if dy > 0 {
				s.zoom *= 1.1
			} else {
				s.zoom /= 1.1
			}
			// Clamp zoom
			if s.zoom < 1.0 {
				s.zoom = 1.0
			}
			if s.zoom > 3.0 {
				s.zoom = 3.0
			}
			// Adjust pan to zoom toward mouse position
			if s.zoom != oldZoom {
				sidebarWidth := 270
				mapCenterX := sidebarWidth + (ScreenWidth-sidebarWidth)/2
				mapCenterY := (ScreenHeight - 120) / 2
				zoomRatio := s.zoom / oldZoom
				s.panX = int(float64(s.panX-(mx-mapCenterX))*(zoomRatio)) + (mx - mapCenterX)
				s.panY = int(float64(s.panY-(my-mapCenterY))*(zoomRatio)) + (my - mapCenterY)
			}
		}
	}

	// Home key to reset zoom and pan
	if inpututil.IsKeyJustPressed(ebiten.KeyHome) {
		s.zoom = 1.0
		s.panX = 0
		s.panY = 0
	}
}

// handleResourceAdjusterClick handles +/- button clicks for resource adjusters.
// The y parameter is where the label is drawn; buttons are 18px below that.
func (s *GameplayScene) handleResourceAdjusterClick(mx, my, x, y int, value *int, min, max int) {
	btnY := y + 18 // Buttons are drawn 18px below the label (see drawResourceAdjuster)
	minusBtnX := x
	plusBtnX := x + 60

	if my >= btnY && my < btnY+20 {
		if mx >= minusBtnX && mx < minusBtnX+20 && *value > min {
			*value--
		}
		if mx >= plusBtnX && mx < plusBtnX+20 && *value < max {
			*value++
		}
	}
}

// isHorseSelectionComplete returns true if enough territories have been selected.
func (s *GameplayScene) isHorseSelectionComplete() bool {
	switch s.pendingHorseSelection {
	case "offer":
		return len(s.tradeOfferHorseTerrs) >= s.pendingHorseCount
	case "request":
		return len(s.tradeRequestHorseDestTerrs) >= s.pendingHorseCount
	case "receive":
		return len(s.tradeHorseDestTerrs) >= s.pendingHorseCount
	case "give":
		return len(s.tradeHorseSourceTerrs) >= s.pendingHorseCount
	}
	return false
}

// confirmHorseSelection confirms the horse selection and proceeds with the trade.
func (s *GameplayScene) confirmHorseSelection() {
	if !s.isHorseSelectionComplete() {
		return
	}
	switch s.pendingHorseSelection {
	case "offer", "request":
		s.completeSendTradeOffer()
	case "receive", "give":
		s.completeAcceptTrade()
	}
}

// cancelHorseSelection cancels the horse selection and the trade.
func (s *GameplayScene) cancelHorseSelection() {
	wasAccepting := s.pendingHorseSelection == "receive" || s.pendingHorseSelection == "give"
	s.pendingHorseSelection = ""
	s.pendingHorseCount = 0
	s.tradeOfferHorseTerrs = nil
	s.tradeRequestHorseDestTerrs = nil
	s.tradeHorseDestTerrs = nil
	s.tradeHorseSourceTerrs = nil
	// If was accepting, reject the trade
	if wasAccepting && s.tradeProposal != nil {
		s.rejectTrade()
	}
}
