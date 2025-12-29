package client

import (
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
	connectScene *ConnectScene
	lobbyScene   *LobbyScene
	waitingScene *WaitingScene

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

	g := &Game{
		config:  config,
		network: NewNetworkClient(),
	}

	// Create scenes
	g.connectScene = NewConnectScene(g)
	g.lobbyScene = NewLobbyScene(g)
	g.waitingScene = NewWaitingScene(g)

	// Start with connect scene
	g.currentScene = g.connectScene
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
func (g *Game) CreateGame(name string, isPublic bool, settings protocol.GameSettings) error {
	payload := protocol.CreateGamePayload{
		Name:     name,
		IsPublic: isPublic,
		Settings: settings,
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

// LeaveGame leaves the current game.
func (g *Game) LeaveGame() error {
	return g.network.SendPayload(protocol.TypeLeaveGame, struct{}{})
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

			// Move to lobby scene
			g.SetScene(g.lobbyScene)
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

	case protocol.TypeGameStarted:
		log.Println("Game started!")
		// TODO: Switch to gameplay scene

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
