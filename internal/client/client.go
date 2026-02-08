package client

import (
	"fmt"
	"image/color"
	"log"

	"lords-of-conquest/internal/protocol"
	"lords-of-conquest/pkg/maps"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
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
	titleScene    *TitleScene
	connectScene  *ConnectScene
	lobbyScene    *LobbyScene
	waitingScene  *WaitingScene
	gameplayScene *GameplayScene

	// State
	authenticated bool
	inGame        bool
	currentGameID string
	lobbyState    *protocol.LobbyStatePayload

	// Music control UI
	showMusicControl  bool
	musicVolumeSlider *Slider
	musicMuteBtn      *Button
	musicCloseBtn     *Button
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

	// Initialize clipboard for paste support
	InitClipboard()

	// Initialize audio
	InitAudio()
	LoadAudio()

	g := &Game{
		config:  config,
		network: NewNetworkClient(),
	}

	// Initialize music volume from config
	SetMusicVolume(config.MusicVolume)
	SetMusicMuted(!config.SoundEnabled)

	// Create music control UI
	g.musicVolumeSlider = &Slider{
		Min:   0,
		Max:   100,
		Value: int(config.MusicVolume * 100),
		Label: "Volume",
		OnChange: func(val int) {
			SetMusicVolume(float64(val) / 100.0)
			g.config.MusicVolume = float64(val) / 100.0
		},
	}
	g.musicMuteBtn = &Button{
		Text: "Mute",
		OnClick: func() {
			ToggleMusicMute()
			g.config.SoundEnabled = !IsMusicMuted()
		},
	}
	g.musicCloseBtn = &Button{
		Text:    "Close",
		Primary: true,
		OnClick: func() {
			g.showMusicControl = false
			g.config.Save() // Save volume preference
		},
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
	// Handle music control dialog (takes priority)
	if g.showMusicControl {
		g.musicVolumeSlider.Update()
		g.musicMuteBtn.Update()
		g.musicCloseBtn.Update()
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			g.showMusicControl = false
			g.config.Save()
		}
		return nil // Block scene input while dialog is open
	}

	// Check for click on music icon (bottom right)
	if IsMusicPlaying() && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		iconX, iconY := ScreenWidth-50, ScreenHeight-50
		if mx >= iconX && mx <= iconX+40 && my >= iconY && my <= iconY+40 {
			g.showMusicControl = true
			return nil
		}
	}

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

	// Draw music icon when music is playing (bottom right)
	if IsMusicPlaying() {
		g.drawMusicIcon(screen)
	}

	// Draw music control dialog
	if g.showMusicControl {
		g.drawMusicControlDialog(screen)
	}
}

// drawMusicIcon draws a speaker icon in the bottom right corner
func (g *Game) drawMusicIcon(screen *ebiten.Image) {
	iconX := ScreenWidth - 50
	iconY := ScreenHeight - 50
	iconSize := 40

	// Background
	vector.DrawFilledRect(screen, float32(iconX), float32(iconY), float32(iconSize), float32(iconSize), color.RGBA{40, 40, 50, 200}, false)
	vector.StrokeRect(screen, float32(iconX), float32(iconY), float32(iconSize), float32(iconSize), 1, ColorBorder, false)

	// Try to use speaker PNG icon
	if speakerIcon, ok := Icons["speaker"]; ok {
		// Draw the speaker icon centered in the button
		imgW := speakerIcon.Bounds().Dx()
		imgH := speakerIcon.Bounds().Dy()
		scale := float64(iconSize-8) / float64(max(imgW, imgH))

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(scale, scale)
		op.GeoM.Translate(float64(iconX)+4+(float64(iconSize-8)-float64(imgW)*scale)/2,
			float64(iconY)+4+(float64(iconSize-8)-float64(imgH)*scale)/2)

		// Dim if muted
		if IsMusicMuted() {
			op.ColorScale.Scale(0.4, 0.4, 0.4, 1)
		}
		screen.DrawImage(speakerIcon, op)

		// Draw X over icon if muted
		if IsMusicMuted() {
			cx := float32(iconX + iconSize/2)
			cy := float32(iconY + iconSize/2)
			vector.StrokeLine(screen, cx-10, cy-10, cx+10, cy+10, 3, ColorDanger, false)
			vector.StrokeLine(screen, cx+10, cy-10, cx-10, cy+10, 3, ColorDanger, false)
		}
	}
}

// drawMusicControlDialog draws the music volume control dialog
func (g *Game) drawMusicControlDialog(screen *ebiten.Image) {
	// Semi-transparent overlay
	vector.DrawFilledRect(screen, 0, 0, float32(ScreenWidth), float32(ScreenHeight),
		color.RGBA{0, 0, 0, 180}, false)

	// Dialog panel
	dialogW := 300
	dialogH := 180
	dialogX := (ScreenWidth - dialogW) / 2
	dialogY := (ScreenHeight - dialogH) / 2

	DrawFancyPanel(screen, dialogX, dialogY, dialogW, dialogH, "Music Volume")

	// Volume slider
	g.musicVolumeSlider.X = dialogX + 20
	g.musicVolumeSlider.Y = dialogY + 55
	g.musicVolumeSlider.W = dialogW - 40
	g.musicVolumeSlider.H = 35
	g.musicVolumeSlider.Value = int(GetMusicVolume() * 100)
	g.musicVolumeSlider.Draw(screen)

	// Mute button
	if IsMusicMuted() {
		g.musicMuteBtn.Text = "Unmute"
	} else {
		g.musicMuteBtn.Text = "Mute"
	}
	g.musicMuteBtn.X = dialogX + 20
	g.musicMuteBtn.Y = dialogY + 110
	g.musicMuteBtn.W = 100
	g.musicMuteBtn.H = 35
	g.musicMuteBtn.Draw(screen)

	// Close button
	g.musicCloseBtn.X = dialogX + dialogW - 120
	g.musicCloseBtn.Y = dialogY + 110
	g.musicCloseBtn.W = 100
	g.musicCloseBtn.H = 35
	g.musicCloseBtn.Draw(screen)
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

// ChangeColor changes the player's color.
func (g *Game) ChangeColor(color string) error {
	payload := protocol.ChangeColorPayload{
		Color: color,
	}
	return g.network.SendPayload(protocol.TypeChangeColor, payload)
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

// UpdateMap sends a new map to the server (host only).
func (g *Game) UpdateMap(m *maps.Map) error {
	// Convert to protocol MapData
	mapData := &protocol.MapData{
		ID:          m.ID,
		Name:        m.Name,
		Width:       m.Width,
		Height:      m.Height,
		Grid:        m.Grid,
		Territories: make(map[string]protocol.TerritoryInfo),
	}
	for id, t := range m.Territories {
		mapData.Territories[fmt.Sprintf("%d", id)] = protocol.TerritoryInfo{
			Name:     t.Name,
			Resource: t.Resource.String(),
		}
	}

	payload := protocol.UpdateMapPayload{
		MapData: mapData,
	}
	return g.network.SendPayload(protocol.TypeUpdateMap, payload)
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
func (g *Game) ProposeTrade(targetPlayer string, offerCoal, offerGold, offerIron, offerTimber, offerHorses int, offerHorseTerrs []string, requestCoal, requestGold, requestIron, requestTimber, requestHorses int, requestHorseDestTerrs []string) error {
	payload := protocol.ProposeTradePayload{
		TargetPlayer:          targetPlayer,
		OfferCoal:             offerCoal,
		OfferGold:             offerGold,
		OfferIron:             offerIron,
		OfferTimber:           offerTimber,
		OfferHorses:           offerHorses,
		OfferHorseTerrs:       offerHorseTerrs,
		RequestCoal:           requestCoal,
		RequestGold:           requestGold,
		RequestIron:           requestIron,
		RequestTimber:         requestTimber,
		RequestHorses:         requestHorses,
		RequestHorseDestTerrs: requestHorseDestTerrs,
	}
	return g.network.SendPayload(protocol.TypeProposeTrade, payload)
}

// RespondTrade responds to a trade proposal.
func (g *Game) RespondTrade(tradeID string, accepted bool, horseDestinations, horseSources []string) error {
	payload := protocol.RespondTradePayload{
		TradeID:           tradeID,
		Accepted:          accepted,
		HorseDestinations: horseDestinations,
		HorseSources:      horseSources,
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

// RenameTerritory sends a territory rename request to the server.
func (g *Game) RenameTerritory(territoryID, name string) error {
	payload := protocol.RenameTerritoryPayload{
		TerritoryID: territoryID,
		Name:        name,
	}
	return g.network.SendPayload(protocol.TypeRenameTerritory, payload)
}

// DrawTerritory sends territory drawing data (and optionally a name change) to the server.
// Both are saved atomically to avoid race conditions.
func (g *Game) DrawTerritory(territoryID string, drawing map[string]int, name string) error {
	payload := protocol.DrawTerritoryPayload{
		TerritoryID: territoryID,
		Drawing:     drawing,
		Name:        name,
	}
	return g.network.SendPayload(protocol.TypeDrawTerritory, payload)
}

// SendClientReady tells the server we're ready to proceed after an event.
func (g *Game) SendClientReady(eventID, eventType string) error {
	if eventID == "" {
		return nil // No event to acknowledge
	}
	log.Printf("Sending client ready for event %s (%s)", eventID, eventType)
	payload := protocol.ClientReadyPayload{
		EventID:   eventID,
		EventType: eventType,
	}
	return g.network.SendPayload(protocol.TypeClientReady, payload)
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

// ExecuteAttackWithPlan executes an attack using a cached plan from RequestAttackPlan.
func (g *Game) ExecuteAttackWithPlan(targetTerritory, planID string, reinforcement *ReinforcementInfo) error {
	payload := protocol.ExecuteAttackPayload{
		TargetTerritory: targetTerritory,
		PlanID:          planID,
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

// RequestAttackPlan requests alliance resolution before committing to attack.
func (g *Game) RequestAttackPlan(targetTerritory string, reinforcement *ReinforcementInfo) error {
	payload := protocol.RequestAttackPlanPayload{
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
	return g.network.SendPayload(protocol.TypeRequestAttackPlan, payload)
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

// BuyCard purchases a combat card during the Development phase.
func (g *Game) BuyCard(cardType, resource string) error {
	payload := protocol.BuyCardPayload{
		CardType: cardType,
		Resource: resource,
	}
	return g.network.SendPayload(protocol.TypeBuyCard, payload)
}

// SelectDefenseCards sends the defender's card selection for card combat.
func (g *Game) SelectDefenseCards(cardIDs []string) error {
	payload := protocol.SelectCardsPayload{
		CardIDs: cardIDs,
	}
	return g.network.SendPayload(protocol.TypeSelectDefenseCards, payload)
}

// ExecuteAttackWithCards executes an attack with card combat card selection.
func (g *Game) ExecuteAttackWithCards(targetTerritory string, reinforcement *ReinforcementInfo, planID string, attackCardIDs []string) error {
	payload := protocol.ExecuteAttackPayload{
		TargetTerritory: targetTerritory,
		PlanID:          planID,
		AttackCardIDs:   attackCardIDs,
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

// SetAlliance sets the player's alliance preference.
func (g *Game) SetAlliance(setting string) error {
	payload := protocol.SetAlliancePayload{
		Setting: setting,
	}
	return g.network.SendPayload(protocol.TypeSetAlliance, payload)
}

// Surrender surrenders to another player, giving them all territories and resources.
func (g *Game) Surrender(targetPlayerID string) error {
	payload := protocol.SurrenderPayload{
		TargetPlayerID: targetPlayerID,
	}
	return g.network.SendPayload(protocol.TypeSurrender, payload)
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

			// Fade out intro music when connected
			FadeOutIntroMusic(2000) // 2 second fade

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
				EventID:               payload.EventID,
				AttackerID:            payload.AttackerID,
				AttackerWins:          payload.AttackerWins,
				AttackStrength:        payload.AttackStrength,
				DefenseStrength:       payload.DefenseStrength,
				TargetTerritory:       payload.TargetTerritory,
				TargetName:            targetName,
				StockpileCaptured:     payload.StockpileCaptured,
				CapturedCoal:          payload.CapturedCoal,
				CapturedGold:          payload.CapturedGold,
				CapturedIron:          payload.CapturedIron,
				CapturedTimber:        payload.CapturedTimber,
				CapturedFromTerritory: payload.CapturedFromTerritory,
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
				UnitType:            r.UnitType,
				FromTerritory:       r.From,
				WaterBodyID:         r.WaterBodyID,
				StrengthBonus:       r.StrengthBonus,
				CanCarryWeapon:      r.CanCarryWeapon,
				WeaponStrengthBonus: r.WeaponStrengthBonus,
				CanCarryHorse:       r.CanCarryHorse,
				HorseStrengthBonus:  r.HorseStrengthBonus,
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

	case protocol.TypeAttackPlanResolved:
		var payload protocol.AttackPlanResolvedPayload
		if err := msg.ParsePayload(&payload); err != nil {
			log.Printf("Failed to parse attack plan resolved: %v", err)
			return
		}
		// Show the attack confirmation dialog with resolved alliance totals
		g.gameplayScene.ShowAttackConfirmation(&payload)
		log.Printf("Attack plan resolved: %d vs %d (allies: +%d vs +%d)",
			payload.BaseAttackStrength, payload.BaseDefenseStrength,
			payload.AttackerAllyStrength, payload.DefenderAllyStrength)

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
		g.gameplayScene.ShowPhaseSkipped(payload.EventID, payload.Phase, payload.Reason)
		log.Printf("Phase skipped: %s - %s (event: %s)", payload.Phase, payload.Reason, payload.EventID)

	case protocol.TypeProductionResults:
		var payload protocol.ProductionResultsPayload
		if err := msg.ParsePayload(&payload); err != nil {
			log.Printf("Failed to parse production results: %v", err)
			return
		}
		// Start production animation in gameplay scene
		g.gameplayScene.StartProductionAnimation(&payload)
		log.Printf("Production results received: %d items (event: %s)", len(payload.Productions), payload.EventID)

	case protocol.TypeGameEnded:
		var payload protocol.GameEndedPayload
		if err := msg.ParsePayload(&payload); err != nil {
			log.Printf("Failed to parse game ended: %v", err)
			return
		}
		// Show victory screen
		g.gameplayScene.ShowVictory(payload.WinnerID, payload.WinnerName, payload.Reason)
		log.Printf("Game ended! Winner: %s by %s", payload.WinnerName, payload.Reason)

	case protocol.TypeSurrenderResult:
		var payload protocol.SurrenderResultPayload
		if err := msg.ParsePayload(&payload); err != nil {
			log.Printf("Failed to parse surrender result: %v", err)
			return
		}
		log.Printf("Surrender: %s surrendered to %s (%d territories)",
			payload.SurrenderedPlayerName, payload.TargetPlayerName, payload.TerritoriesGained)
		// The game state update will handle showing the changes

	case protocol.TypeTerritoryDrawing:
		var payload protocol.DrawTerritoryPayload
		if err := msg.ParsePayload(&payload); err != nil {
			log.Printf("Failed to parse territory drawing: %v", err)
			return
		}
		// Update local territory drawing data
		g.gameplayScene.UpdateTerritoryDrawing(payload.TerritoryID, payload.Drawing)
		log.Printf("Territory drawing update for %s (%d pixels)", payload.TerritoryID, len(payload.Drawing))

	case protocol.TypeCardDrawn:
		var payload protocol.CardDrawnPayload
		if err := msg.ParsePayload(&payload); err != nil {
			log.Printf("Failed to parse card drawn: %v", err)
			return
		}
		log.Printf("Drew card: %s (%s, %s)", payload.Card.Name, payload.Card.Rarity, payload.Card.CardType)
		g.gameplayScene.ShowCardDrawn(payload.Card.Name, payload.Card.Description, payload.Card.Rarity, payload.Card.CardType)

	case protocol.TypeDefenseCardRequest:
		var payload protocol.DefenseCardRequestPayload
		if err := msg.ParsePayload(&payload); err != nil {
			log.Printf("Failed to parse defense card request: %v", err)
			return
		}
		log.Printf("Defense card request: %s attacking %s with %d cards", payload.AttackerName, payload.TerritoryName, payload.AttackerCardCount)
		g.gameplayScene.ShowDefenseCardRequest(payload.BattleID, payload.AttackerName, payload.TerritoryName, payload.AttackerCardCount, payload.BaseAttackStr, payload.BaseDefenseStr)

	case protocol.TypeCardReveal:
		var payload protocol.CardRevealPayload
		if err := msg.ParsePayload(&payload); err != nil {
			log.Printf("Failed to parse card reveal: %v", err)
			return
		}
		log.Printf("Card reveal: Attack %d vs Defense %d, attacker wins: %v", payload.FinalAttackStr, payload.FinalDefenseStr, payload.AttackerWins)
		g.gameplayScene.ShowCardReveal(&payload)

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
