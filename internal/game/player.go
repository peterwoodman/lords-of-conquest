package game

// PlayerColor represents a player's color.
type PlayerColor string

const (
	ColorOrange PlayerColor = "orange"
	ColorCyan   PlayerColor = "cyan"
	ColorGreen  PlayerColor = "green"
	ColorYellow PlayerColor = "yellow"
	ColorPurple PlayerColor = "purple"
	ColorRed    PlayerColor = "red"
	ColorBlue   PlayerColor = "blue"
)

// AllColors returns all available player colors.
func AllColors() []PlayerColor {
	return []PlayerColor{
		ColorOrange,
		ColorCyan,
		ColorGreen,
		ColorYellow,
		ColorPurple,
		ColorRed,
		ColorBlue,
	}
}

// AllianceSetting represents a player's alliance preference.
type AllianceSetting string

const (
	AllianceAsk      AllianceSetting = "ask"      // Ask each time (default)
	AllianceNeutral  AllianceSetting = "neutral"  // Never participate
	AllianceDefender AllianceSetting = "defender" // Always support defender
	// Or a player ID to always support that player
)

// Player represents a player in the game.
type Player struct {
	ID                 string          `json:"id"`
	Name               string          `json:"name"`
	Color              PlayerColor     `json:"color"`
	IsAI               bool            `json:"isAI"`
	AIPersonality      AIPersonality   `json:"aiPersonality"`
	Stockpile          *Stockpile      `json:"stockpile"`
	StockpileTerritory string          `json:"stockpileTerritory"` // Territory ID where stockpile is located
	AttacksRemaining   int             `json:"attacksRemaining"`
	Eliminated         bool            `json:"eliminated"`
	Alliance           AllianceSetting `json:"alliance"`   // Alliance preference
	IsOnline           bool            `json:"isOnline"`   // Connection status
}

// AIPersonality defines AI behavior type.
type AIPersonality int

const (
	AIPersonalityNone AIPersonality = iota
	AIAggressive
	AIDefensive
	AIPassive
)

// String returns the personality name.
func (p AIPersonality) String() string {
	switch p {
	case AIAggressive:
		return "Aggressive"
	case AIDefensive:
		return "Defensive"
	case AIPassive:
		return "Passive"
	default:
		return "Human"
	}
}

// NewPlayer creates a new player.
func NewPlayer(id, name string, color PlayerColor) *Player {
	return &Player{
		ID:               id,
		Name:             name,
		Color:            color,
		IsAI:             false,
		Stockpile:        NewStockpile(),
		AttacksRemaining: 2,
		Eliminated:       false,
		Alliance:         AllianceAsk, // Default to ask
		IsOnline:         true,
	}
}

// NewAIPlayer creates a new AI player.
func NewAIPlayer(id, name string, color PlayerColor, personality AIPersonality) *Player {
	return &Player{
		ID:               id,
		Name:             name,
		Color:            color,
		IsAI:             true,
		AIPersonality:    personality,
		Stockpile:        NewStockpile(),
		AttacksRemaining: 2,
		Eliminated:       false,
		Alliance:         AllianceNeutral, // AI is neutral by default
		IsOnline:         true,            // AI is always "online"
	}
}

// ResetTurn resets per-turn counters for the player.
func (p *Player) ResetTurn() {
	p.AttacksRemaining = 2
}

