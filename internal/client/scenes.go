package client

import (
	"fmt"

	"lords-of-conquest/internal/protocol"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ==================== Connect Scene ====================

// ConnectScene handles server connection and player name entry.
type ConnectScene struct {
	game *Game

	serverInput *TextInput
	nameInput   *TextInput
	connectBtn  *Button
	statusText  string
	connecting  bool
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
	// Title
	DrawTextCentered(screen, "LORDS OF CONQUEST", ScreenWidth/2, 150, ColorText)
	DrawTextCentered(screen, "A Modern Remake", ScreenWidth/2, 180, ColorTextMuted)

	// Server input
	DrawText(screen, "Server:", ScreenWidth/2-150, 258, ColorTextMuted)
	s.serverInput.Draw(screen)

	// Name input
	DrawText(screen, "Your Name:", ScreenWidth/2-150, 318, ColorTextMuted)
	s.nameInput.Draw(screen)

	// Connect button
	s.connectBtn.Draw(screen)

	// Status text
	if s.statusText != "" {
		DrawTextCentered(screen, s.statusText, ScreenWidth/2, 480, ColorTextMuted)
	}

	// Version
	ebitenutil.DebugPrintAt(screen, "v0.1.0", 10, ScreenHeight-20)
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

	gameList    *List
	codeInput   *TextInput
	createBtn   *Button
	joinBtn     *Button
	joinCodeBtn *Button
	refreshBtn  *Button
	games       []protocol.GameListItem

	// Create game dialog
	showCreate       bool
	createNameInput  *TextInput
	createPublicBtn  *Button
	createPrivateBtn *Button
	createConfirmBtn *Button
	createCancelBtn  *Button
	createPublic     bool
}

// NewLobbyScene creates a new lobby scene.
func NewLobbyScene(game *Game) *LobbyScene {
	s := &LobbyScene{game: game}

	// Game list
	s.gameList = NewList(50, 100, 500, 400)
	s.gameList.OnSelect = func(id string) {}

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
		OnClick: func() { s.showCreate = true },
	}

	s.joinBtn = &Button{
		X: 600, Y: 520, W: 200, H: 40,
		Text:    "Join Selected",
		OnClick: s.onJoinSelected,
	}

	s.joinCodeBtn = &Button{
		X: 810, Y: 150, W: 100, H: 40,
		Text:    "Join",
		OnClick: s.onJoinByCode,
	}

	s.refreshBtn = &Button{
		X: 50, Y: 520, W: 150, H: 40,
		Text:    "Refresh",
		OnClick: func() { s.game.ListGames() },
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
		OnClick: func() { s.showCreate = false },
	}

	s.createPublic = true

	return s
}

func (s *LobbyScene) OnEnter() {
	s.showCreate = false
	s.game.ListGames()
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

		// Escape to close
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			s.showCreate = false
		}
		return nil
	}

	s.gameList.Update()
	s.codeInput.Update()
	s.createBtn.Update()
	s.joinBtn.Update()
	s.joinCodeBtn.Update()
	s.refreshBtn.Update()

	// Disable join if nothing selected
	s.joinBtn.Disabled = s.gameList.GetSelectedID() == ""

	return nil
}

func (s *LobbyScene) Draw(screen *ebiten.Image) {
	// Title
	DrawText(screen, "Game Lobby", 50, 50, ColorText)
	DrawText(screen, fmt.Sprintf("Welcome, %s!", s.game.config.PlayerName), 50, 70, ColorTextMuted)

	// Public games list
	DrawText(screen, "Public Games:", 50, 85, ColorTextMuted)
	s.gameList.Draw(screen)

	// Right side
	s.createBtn.Draw(screen)

	DrawText(screen, "Join by code:", 600, 130, ColorTextMuted)
	s.codeInput.Draw(screen)
	s.joinCodeBtn.Draw(screen)

	// Bottom buttons
	s.refreshBtn.Draw(screen)
	s.joinBtn.Draw(screen)

	// Create dialog overlay
	if s.showCreate {
		// Dim background
		DrawPanel(screen, ScreenWidth/2-200, 200, 400, 300)

		DrawTextCentered(screen, "Create Game", ScreenWidth/2, 230, ColorText)

		DrawText(screen, "Game Name:", ScreenWidth/2-150, 258, ColorTextMuted)
		s.createNameInput.Draw(screen)

		DrawText(screen, "Visibility:", ScreenWidth/2-150, 320, ColorTextMuted)
		s.createPublicBtn.Draw(screen)
		s.createPrivateBtn.Draw(screen)

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
	s.gameList.SetItems(items)
}

func (s *LobbyScene) onJoinSelected() {
	id := s.gameList.GetSelectedID()
	if id != "" {
		s.game.JoinGame(id)
	}
}

func (s *LobbyScene) onJoinByCode() {
	code := s.codeInput.Text
	if code != "" {
		s.game.JoinByCode(code)
	}
}

func (s *LobbyScene) onCreateConfirm() {
	name := s.createNameInput.Text
	if name == "" {
		name = s.game.config.PlayerName + "'s Game"
	}

	settings := protocol.GameSettings{
		MaxPlayers:    4,
		GameLevel:     "expert",
		ChanceLevel:   "medium",
		VictoryCities: 3,
	}

	s.game.CreateGame(name, s.createPublic, settings)
	s.showCreate = false
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

	return s
}

func (s *WaitingScene) OnEnter() {}
func (s *WaitingScene) OnExit()  {}

func (s *WaitingScene) Update() error {
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

		s.addAIBtn.Update()
		s.startBtn.Update()

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
	}
	s.readyBtn.Draw(screen)
	s.leaveBtn.Draw(screen)
	s.copyCodeBtn.Draw(screen)

	// Host indicator
	if isHost {
		DrawText(screen, "(You are the host)", 500, 130, ColorTextMuted)
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
