package client

import (
	"fmt"
	"image/color"

	"lords-of-conquest/internal/protocol"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
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
	// Starfield background effect (simple dots)
	for i := 0; i < 50; i++ {
		x := float32((i * 137) % ScreenWidth)
		y := float32((i * 97) % ScreenHeight)
		size := float32(1 + (i%3))
		alpha := uint8(100 + (i % 155))
		starColor := color.RGBA{100, 150, 255, alpha}
		vector.DrawFilledCircle(screen, x, y, size, starColor, false)
	}
	
	// Main panel
	panelW := 600
	panelH := 450
	panelX := ScreenWidth/2 - panelW/2
	panelY := ScreenHeight/2 - panelH/2
	DrawFancyPanel(screen, panelX, panelY, panelW, panelH, "")
	
	// Huge title - centered
	titleY := panelY + 50
	DrawHugeTitleCentered(screen, "LORDS OF CONQUEST", ScreenWidth/2, titleY)
	
	// Subtitle - larger
	DrawLargeTextCentered(screen, "A Modern Remake", ScreenWidth/2, titleY+45, ColorTextMuted)

	// Server input
	inputY := panelY + 180
	DrawLargeText(screen, "Server:", panelX+30, inputY-30, ColorTextMuted)
	s.serverInput.Y = inputY
	s.serverInput.H = 45
	s.serverInput.Draw(screen)

	// Name input
	inputY += 90
	DrawLargeText(screen, "Your Name:", panelX+30, inputY-30, ColorTextMuted)
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
		OnClick: func() { s.showCreate = true },
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

	s.yourGameList.Update()
	s.gameList.Update()
	s.codeInput.Update()
	s.createBtn.Update()
	s.joinBtn.Update()
	s.deleteBtn.Update()
	s.joinCodeBtn.Update()
	s.refreshBtn.Update()

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
	DrawLargeText(screen, "GAME LOBBY", 45, 38, ColorText)
	
	DrawLargeText(screen, fmt.Sprintf("Welcome, %s!", s.game.config.PlayerName), 45, 58, ColorTextMuted)

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
	DrawLargeText(screen, "Join by code:", buttonX, buttonY, ColorTextMuted)
	s.codeInput.X = buttonX
	s.codeInput.Y = buttonY + 25
	s.codeInput.W = 200
	s.codeInput.H = 45
	s.codeInput.Draw(screen)
	
	s.joinCodeBtn.X = buttonX + 210
	s.joinCodeBtn.Y = buttonY + 25
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
			color.RGBA{0, 0, 0, 180}, false)
		
		// Dialog panel
		DrawFancyPanel(screen, ScreenWidth/2-300, 180, 600, 360, "Create New Game")

		dialogX := ScreenWidth/2 - 250
		dialogY := 250
		
		DrawLargeText(screen, "Game Name:", dialogX, dialogY, ColorTextMuted)
		s.createNameInput.X = dialogX
		s.createNameInput.Y = dialogY + 25
		s.createNameInput.H = 45
		s.createNameInput.Draw(screen)

		dialogY += 100
		DrawLargeText(screen, "Visibility:", dialogX, dialogY, ColorTextMuted)
		s.createPublicBtn.Y = dialogY + 25
		s.createPublicBtn.H = 50
		s.createPrivateBtn.Y = dialogY + 25
		s.createPrivateBtn.H = 50
		s.createPublicBtn.Draw(screen)
		s.createPrivateBtn.Draw(screen)

		s.createConfirmBtn.Y = dialogY + 110
		s.createConfirmBtn.H = 50
		s.createCancelBtn.Y = dialogY + 110
		s.createCancelBtn.H = 50
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
	s.yourGameList.SetItems(items)
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
	name := s.createNameInput.Text
	if name == "" {
		name = s.game.config.PlayerName + "'s Game"
	}

	settings := protocol.GameSettings{
		MaxPlayers:    4,
		GameLevel:     "expert",
		ChanceLevel:   "medium",
		VictoryCities: 3,
		MapID:         "test", // Use the test map
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
