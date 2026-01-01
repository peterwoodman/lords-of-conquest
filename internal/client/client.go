package client

import (
	"fmt"
	"image/color"
	"log"

	"lords-of-conquest/internal/protocol"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	ScreenWidth  = 1280
	ScreenHeight = 720
)

// Scene represents a game screen/state.
type Scene interface {
	Update() error
	Draw(screen *ebiten.Image)
	OnEnter()
	OnExit()
}

// Game is the main Ebitengine game struct.
type Game struct {
	config  *Config
	network *NetworkClient

	// Current scene
	currentScene Scene
	nextScene    Scene

	// Scenes
	titleScene   *TitleScene
	connectScene *ConnectScene
	lobbyScene   *LobbyScene
	waitingScene *WaitingScene
	gameplayScene *GameplayScene

	// State
	authenticated bool
	inGame        bool
	currentGameID string
	lobbyState    *protocol.LobbyStatePayload
}

// NewGame creates a new game instance.
func NewGame() (*Game, error) {
	config, err := LoadConfig()
	if err != nil {
		log.Printf("Failed to load config: %v", err)
	}

	// Load icons (will use fallbacks if PNGs not found)
	LoadIcons()
	if len(Icons) == 0 {
		log.Printf("No PNG icons found, creating placeholder icons")
		CreatePlaceholderIcons()
	}

	// Load title screens
	LoadTitleScreens()

	// Initialize audio
	InitAudio()
	LoadAudio()

	g := &Game{
		config:  config,
		network: NewNetworkClient(),
	}

	// Create scenes
	g.titleScene = NewTitleScene(g)
	g.connectScene = NewConnectScene(g)
	g.lobbyScene = NewLobbyScene(g)
	g.waitingScene = NewWaitingScene(g)
	g.gameplayScene = NewGameplayScene(g)

	// Start with title scene
	g.currentScene = g.titleScene
	g.currentScene.OnEnter()

	// Set up network callbacks
	g.network.OnMessage = g.handleMessage
	g.network.OnDisconnect = g.handleDisconnect

	return g, nil
}

// Update handles game logic.
func (g *Game) Update() error {
	// Process scene transition
	if g.nextScene != nil {
		if g.currentScene != nil {
			g.currentScene.OnExit()
		}
		g.currentScene = g.nextScene
		g.nextScene = nil
		g.currentScene.OnEnter()
	}

	// Update current scene
	if g.currentScene != nil {
		return g.currentScene.Update()
	}
	return nil
}

// Draw renders the game.
func (g *Game) Draw(screen *ebiten.Image) {
	// Clear with dark background
	screen.Fill(color.RGBA{20, 20, 30, 255})

	if g.currentScene != nil {
		g.currentScene.Draw(screen)
	}
}

// Layout returns the game's screen dimensions.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ScreenWidth, ScreenHeight
}

// SetScene transitions to a new scene.
func (g *Game) SetScene(scene Scene) {
	g.nextScene = scene
}

// Connect attempts to connect to the server.
func (g *Game) Connect(serverAddr string) error {
	g.config.LastServer = serverAddr
	g.config.Save()
	return g.network.Connect(serverAddr)
}

// Authenticate sends authentication to the server.
func (g *Game) Authenticate(name string) error {
	payload := protocol.AuthenticatePayload{
		Token: g.config.PlayerToken,
		Name:  name,
	}
	return g.network.SendPayload(protocol.TypeAuthenticate, payload)
}

// CreateGame creates a new game on the server.
func (g *Game) CreateGame(name string, isPublic bool, settings protocol.GameSettings, mapData *protocol.MapData) error {
	payload := protocol.CreateGamePayload{
		Name:     name,
		IsPublic: isPublic,
		Settings: settings,
		MapData:  mapData,
	}
	return g.network.SendPayload(protocol.TypeCreateGame, payload)
}

// JoinGame joins a game by ID.
func (g *Game) JoinGame(gameID string) error {
	payload := protocol.JoinGamePayload{
		GameID: gameID,
	}
	return g.network.SendPayload(protocol.TypeJoinGame, payload)
}

// JoinByCode joins a game by join code.
func (g *Game) JoinByCode(code string) error {
	payload := protocol.JoinByCodePayload{
		JoinCode: code,
	}
	return g.network.SendPayload(protocol.TypeJoinByCode, payload)
}

// ListGames requests the list of public games.
func (g *Game) ListGames() error {
	return g.network.SendPayload(protocol.TypeListGames, struct{}{})
}

func (g *Game) ListYourGames() error {
	return g.network.SendPayload(protocol.TypeYourGames, struct{}{})
}

func (g *Game) DeleteGame(gameID string) error {
	payload := protocol.DeleteGamePayload{
		GameID: gameID,
	}
	return g.network.SendPayload(protocol.TypeDeleteGame, payload)
}

// SetReady sets the player's ready status.
func (g *Game) SetReady(ready bool) error {
	payload := protocol.PlayerReadyPayload{
		Ready: ready,
	}
	return g.network.SendPayload(protocol.TypePlayerReady, payload)
}

// StartGame starts the current game (host only).
func (g *Game) StartGame() error {
	return g.network.SendPayload(protocol.TypeStartGame, struct{}{})
}

// AddAI adds an AI player to the game (host only).
func (g *Game) AddAI(personality string) error {
	payload := protocol.AddAIPayload{
		Personality: personality,
	}
	return g.network.SendPayload(protocol.TypeAddAI, payload)
}

// UpdateGameSettings updates a single game setting (host only).
func (g *Game) UpdateGameSettings(key, value string) error {
	payload := protocol.UpdateSettingPayload{
		Key:   key,
		Value: value,
	}
	return g.network.SendPayload(protocol.TypeUpdateSettings, payload)
}

// LeaveGame leaves the current game.
func (g *Game) LeaveGame() error {
	return g.network.SendPayload(protocol.TypeLeaveGame, struct{}{})
}

// SelectTerritory sends a territory selection to the server.
func (g *Game) SelectTerritory(territoryID string) error {
	payload := protocol.SelectTerritoryPayload{
		TerritoryID: territoryID,
	}
	return g.network.SendPayload(protocol.TypeSelectTerritory, payload)
}

// PlaceStockpile sends a stockpile placement to the server.
func (g *Game) PlaceStockpile(territoryID string) error {
	payload := protocol.PlaceStockpilePayload{
		TerritoryID: territoryID,
	}
	return g.network.SendPayload(protocol.TypePlaceStockpile, payload)
}

// MoveStockpile moves the stockpile to a new territory during shipment.
func (g *Game) MoveStockpile(destinationID string) error {
	payload := protocol.MoveStockpilePayload{
		Destination: destinationID,
	}
	return g.network.SendPayload(protocol.TypeMoveStockpile, payload)
}

// MoveUnit moves a unit (horse, boat) during shipment phase.
func (g *Game) MoveUnit(unitType, fromID, toID, waterBodyID string, carryHorse, carryWeapon bool) error {
	payload := protocol.MoveUnitPayload{
		UnitType:    unitType,
		From:        fromID,
		To:          toID,
		WaterBodyID: waterBodyID,
		CarryHorse:  carryHorse,
		CarryWeapon: carryWeapon,
	}
	return g.network.SendPayload(protocol.TypeMoveUnit, payload)
}

// EndPhase ends the current phase for this player.
func (g *Game) EndPhase() error {
	return g.network.SendPayload(protocol.TypeEndPhase, struct{}{})
}

// ProposeTrade proposes a trade to another player.
func (g *Game) ProposeTrade(targetPlayer string, offerCoal, offerGold, offerIron, offerTimber, offerHorses int, offerHorseTerrs []string, requestCoal, requestGold, requestIron, requestTimber, requestHorses int) error {
	payload := protocol.ProposeTradePayload{
		TargetPlayer:    targetPlayer,
		OfferCoal:       offerCoal,
		OfferGold:       offerGold,
		OfferIron:       offerIron,
		OfferTimber:     offerTimber,
		OfferHorses:     offerHorses,
		OfferHorseTerrs: offerHorseTerrs,
		RequestCoal:     requestCoal,
		RequestGold:     requestGold,
		RequestIron:     requestIron,
		RequestTimber:   requestTimber,
		RequestHorses:   requestHorses,
	}
	return g.network.SendPayload(protocol.TypeProposeTrade, payload)
}

// RespondTrade responds to a trade proposal.
func (g *Game) RespondTrade(tradeID string, accepted bool, horseDestinations []string) error {
	payload := protocol.RespondTradePayload{
		TradeID:           tradeID,
		Accepted:          accepted,
		HorseDestinations: horseDestinations,
	}
	return g.network.SendPayload(protocol.TypeRespondTrade, payload)
}

// Build builds a unit or city during development phase.
func (g *Game) Build(buildType, territoryID string, useGold bool) error {
	payload := protocol.BuildPayload{
		Type:      buildType,
		Territory: territoryID,
		UseGold:   useGold,
	}
	return g.network.SendPayload(protocol.TypeBuild, payload)
}

// BuildBoatInWater builds a boat in a specific water body.
func (g *Game) BuildBoatInWater(territoryID, waterBodyID string, useGold bool) error {
	payload := protocol.BuildPayload{
		Type:        "boat",
		Territory:   territoryID,
		WaterBodyID: waterBodyID,
		UseGold:     useGold,
	}
	return g.network.SendPayload(protocol.TypeBuild, payload)
}

// PlanAttack requests an attack preview for a target territory.
func (g *Game) PlanAttack(targetTerritory string) error {
	payload := protocol.PlanAttackPayload{
		TargetTerritory: targetTerritory,
	}
	return g.network.SendPayload(protocol.TypePlanAttack, payload)
}

// ExecuteAttackWithReinforcement executes an attack with optional reinforcement.
func (g *Game) ExecuteAttackWithReinforcement(targetTerritory string, reinforcement *ReinforcementInfo) error {
	payload := protocol.ExecuteAttackPayload{
		TargetTerritory: targetTerritory,
	}
	if reinforcement != nil {
		payload.BringUnit = reinforcement.UnitType
		payload.BringFrom = reinforcement.FromTerritory
		payload.WaterBodyID = reinforcement.WaterBodyID
		payload.CarryWeapon = reinforcement.CarryWeapon
		payload.WeaponFrom = reinforcement.WeaponFrom
		payload.CarryHorse = reinforcement.CarryHorse
		payload.HorseFrom = reinforcement.HorseFrom
	}
	return g.network.SendPayload(protocol.TypeExecuteAttack, payload)
}

// ReinforcementInfo holds data about a unit to bring into battle.
type ReinforcementInfo struct {
	UnitType      string
	FromTerritory string
	WaterBodyID   string // For boats
	CarryWeapon   bool
	WeaponFrom    string
	CarryHorse    bool
	HorseFrom     string
}

// ExecuteAttack executes an attack during conquest phase.
func (g *Game) ExecuteAttack(targetTerritory string) error {
	payload := protocol.ExecuteAttackPayload{
		TargetTerritory: targetTerritory,
	}
	return g.network.SendPayload(protocol.TypeExecuteAttack, payload)
}

// SetAlliance sets the player's alliance preference.
func (g *Game) SetAlliance(setting string) error {
	payload := protocol.SetAlliancePayload{
		Setting: setting,
	}
	return g.network.SendPayload(protocol.TypeSetAlliance, payload)
}

// AllianceVote sends the player's vote for an alliance request.
func (g *Game) AllianceVote(battleID, side string) error {
	payload := protocol.AllianceVotePayload{
		BattleID: battleID,
		Side:     side,
	}
	return g.network.SendPayload(protocol.TypeAllianceVote, payload)
}

// handleMessage processes incoming server messages.
func (g *Game) handleMessage(msg *protocol.Message) {
	switch msg.Type {
	case protocol.TypeWelcome:
		log.Println("Connected to server")

	case protocol.TypeAuthResult:
		var payload protocol.AuthResultPayload
		if err := msg.ParsePayload(&payload); err != nil {
			log.Printf("Failed to parse auth result: %v", err)
			return
		}
		if payload.Success {
			g.authenticated = true
			g.config.PlayerToken = payload.Token
			g.config.PlayerID = payload.PlayerID
			g.config.PlayerName = payload.Name
			g.config.Save()
			log.Printf("Authenticated as %s", payload.Name)

			// Reset connect scene state
			g.connectScene.connecting = false
			g.connectScene.connectBtn.Disabled = false

			// Move to lobby scene
			g.SetScene(g.lobbyScene)
			
			// Request game lists
			log.Printf("Requesting game lists...")
			g.ListGames()
			g.ListYourGames()
			log.Printf("Game list requests sent")
		} else {
			log.Printf("Authentication failed: %s", payload.Error)
			// Reset connect scene state on failure
			g.connectScene.statusText = fmt.Sprintf("Auth failed: %s", payload.Error)
			g.connectScene.connecting = false
			g.connectScene.connectBtn.Disabled = false
		}

	case protocol.TypeGameCreated:
		var payload protocol.GameCreatedPayload
		if err := msg.ParsePayload(&payload); err != nil {
			return
		}
		g.currentGameID = payload.GameID
		log.Printf("Game created: %s (code: %s)", payload.GameID, payload.JoinCode)

	case protocol.TypeJoinedGame:
		var payload protocol.JoinedGamePayload
		if err := msg.ParsePayload(&payload); err != nil {
			return
		}
		g.currentGameID = payload.GameID
		log.Printf("Joined game: %s", payload.GameID)

	case protocol.TypeLobbyState:
		var payload protocol.LobbyStatePayload
		if err := msg.ParsePayload(&payload); err != nil {
			return
		}
		g.lobbyState = &payload
		g.inGame = true

		// Switch to waiting room
		if g.currentScene != g.waitingScene {
			g.SetScene(g.waitingScene)
		}

	case protocol.TypeGameList:
		var payload protocol.GameListPayload
		if err := msg.ParsePayload(&payload); err != nil {
			return
		}
		if lobby, ok := g.currentScene.(*LobbyScene); ok {
			lobby.SetGameList(payload.Games)
		}

	case protocol.TypeYourGames:
		var payload protocol.YourGamesPayload
		if err := msg.ParsePayload(&payload); err != nil {
			return
		}
		if lobby, ok := g.currentScene.(*LobbyScene); ok {
			lobby.SetYourGames(payload.Games)
		}

	case protocol.TypeGameDeleted:
		var payload protocol.GameDeletedPayload
		if err := msg.ParsePayload(&payload); err != nil {
			return
		}
		log.Printf("Game %s was deleted: %s", payload.GameID, payload.Reason)
		// If we were in that game, go back to lobby
		if g.currentGameID == payload.GameID {
			g.currentGameID = ""
			g.inGame = false
			g.lobbyState = nil
			g.SetScene(g.lobbyScene)
		}

	case protocol.TypeGameStarted:
		log.Println("Game started!")
		// Switch to gameplay scene
		g.SetScene(g.gameplayScene)

	case protocol.TypeGameState:
		log.Println("Received game state update")
		var payload protocol.GameStatePayload
		if err := msg.ParsePayload(&payload); err != nil {
			log.Printf("Failed to parse game state: %v", err)
			return
		}
		// Update gameplay scene with new state
		if stateMap, ok := payload.State.(map[string]interface{}); ok {
			// Update the gameplay scene regardless of current scene
			// (it might be transitioning)
			g.gameplayScene.SetGameState(stateMap)
			log.Println("Game state set on gameplay scene")
		} else {
			log.Printf("Game state is not a map: %T", payload.State)
		}

	case protocol.TypeActionResult:
		// Handle combat results
		var payload protocol.CombatResultPayload
		if err := msg.ParsePayload(&payload); err != nil {
			log.Printf("Failed to parse action result: %v", err)
			return
		}
		
		// Show combat result in gameplay scene
		if payload.TargetTerritory != "" {
			// Get territory name
			targetName := payload.TargetTerritory
			if g.gameplayScene.territories != nil {
				if terr, ok := g.gameplayScene.territories[payload.TargetTerritory].(map[string]interface{}); ok {
					if name, ok := terr["name"].(string); ok {
						targetName = name
					}
				}
			}
			
			result := &CombatResultData{
				AttackerWins:    payload.AttackerWins,
				AttackStrength:  payload.AttackStrength,
				DefenseStrength: payload.DefenseStrength,
				TargetTerritory: payload.TargetTerritory,
				TargetName:      targetName,
			}
			g.gameplayScene.ShowCombatResult(result)
			
			if payload.AttackerWins {
				log.Printf("Combat victory! Captured %s", targetName)
			} else {
				log.Printf("Combat defeat at %s", targetName)
			}
		}
		
	case protocol.TypeGameHistory:
		var payload protocol.GameHistoryPayload
		if err := msg.ParsePayload(&payload); err != nil {
			log.Printf("Failed to parse game history: %v", err)
			return
		}
		// Update gameplay scene with history
		g.gameplayScene.SetHistory(payload.Events)
		log.Printf("Received %d history events", len(payload.Events))

	case protocol.TypeAttackPreview:
		var payload protocol.AttackPreviewPayload
		if err := msg.ParsePayload(&payload); err != nil {
			log.Printf("Failed to parse attack preview: %v", err)
			return
		}
		// Convert to client format
		reinforcements := make([]ReinforcementData, len(payload.AvailableReinforcements))
		for i, r := range payload.AvailableReinforcements {
			reinforcements[i] = ReinforcementData{
				UnitType:       r.UnitType,
				FromTerritory:  r.From,
				WaterBodyID:    r.WaterBodyID,
				StrengthBonus:  r.StrengthBonus,
				CanCarryWeapon: r.CanCarryWeapon,
				CanCarryHorse:  r.CanCarryHorse,
			}
		}
		preview := &AttackPreviewData{
			TargetTerritory:      payload.TargetTerritory,
			AttackStrength:       payload.AttackStrength,
			DefenseStrength:      payload.DefenseStrength,
			AttackerAllyStrength: payload.AttackerAllyStrength,
			DefenderAllyStrength: payload.DefenderAllyStrength,
			CanAttack:            payload.CanAttack,
			Reinforcements:       reinforcements,
		}
		g.gameplayScene.ShowAttackPlan(preview)
		log.Printf("Attack preview: %d vs %d, %d reinforcements available",
			payload.AttackStrength, payload.DefenseStrength, len(reinforcements))

	case protocol.TypeAllianceRequest:
		var payload protocol.AllianceRequestPayload
		if err := msg.ParsePayload(&payload); err != nil {
			log.Printf("Failed to parse alliance request: %v", err)
			return
		}
		// Show alliance request popup in gameplay scene
		g.gameplayScene.ShowAllianceRequest(&payload)
		log.Printf("Alliance request: battle %s, %s vs %s at %s",
			payload.BattleID, payload.AttackerName, payload.DefenderName, payload.TerritoryName)

	case protocol.TypeAllianceResult:
		var payload protocol.AllianceResultPayload
		if err := msg.ParsePayload(&payload); err != nil {
			log.Printf("Failed to parse alliance result: %v", err)
			return
		}
		log.Printf("Alliance vote %s: accepted=%v", payload.BattleID, payload.Accepted)

	case protocol.TypeTradeProposal:
		var payload protocol.TradeProposalPayload
		if err := msg.ParsePayload(&payload); err != nil {
			log.Printf("Failed to parse trade proposal: %v", err)
			return
		}
		// Show trade proposal popup
		g.gameplayScene.ShowTradeProposal(&payload)
		log.Printf("Trade proposal from %s", payload.FromPlayerName)

	case protocol.TypeTradeResult:
		var payload protocol.TradeResultPayload
		if err := msg.ParsePayload(&payload); err != nil {
			log.Printf("Failed to parse trade result: %v", err)
			return
		}
		// Show trade result
		g.gameplayScene.ShowTradeResult(&payload)
		log.Printf("Trade result: accepted=%v - %s", payload.Accepted, payload.Message)

	case protocol.TypePhaseSkipped:
		var payload protocol.PhaseSkippedPayload
		if err := msg.ParsePayload(&payload); err != nil {
			log.Printf("Failed to parse phase skipped: %v", err)
			return
		}
		// Show phase skip popup in gameplay scene
		g.gameplayScene.ShowPhaseSkipped(payload.Phase, payload.Reason)
		log.Printf("Phase skipped: %s - %s", payload.Phase, payload.Reason)

	case protocol.TypeGameEnded:
		var payload protocol.GameEndedPayload
		if err := msg.ParsePayload(&payload); err != nil {
			log.Printf("Failed to parse game ended: %v", err)
			return
		}
		// Show victory screen
		g.gameplayScene.ShowVictory(payload.WinnerID, payload.WinnerName, payload.Reason)
		log.Printf("Game ended! Winner: %s by %s", payload.WinnerName, payload.Reason)

	case protocol.TypeError:
		var payload protocol.ErrorPayload
		if err := msg.ParsePayload(&payload); err != nil {
			return
		}
		log.Printf("Server error: %s - %s", payload.Code, payload.Message)
	}
}

// handleDisconnect handles disconnection from the server.
func (g *Game) handleDisconnect(err error) {
	log.Println("Disconnected from server")
	g.authenticated = false
	g.inGame = false
	g.SetScene(g.connectScene)
}
