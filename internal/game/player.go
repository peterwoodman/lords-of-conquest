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
	AIDifficulty       AIDifficulty    `json:"aiDifficulty,omitempty"`
	Stockpile          *Stockpile      `json:"stockpile"`
	StockpileTerritory string          `json:"stockpileTerritory"` // Territory ID where stockpile is located
	AttacksRemaining   int             `json:"attacksRemaining"`
	Eliminated         bool            `json:"eliminated"`
	Alliance           AllianceSetting `json:"alliance"`     // Alliance preference
	IsOnline           bool            `json:"isOnline"`     // Connection status
	AttackCards        []CombatCard    `json:"attackCards"`   // Combat cards (card mode only)
	DefenseCards       []CombatCard    `json:"defenseCards"`  // Combat cards (card mode only)
}

// AIDifficulty is the skill level of a CPU player.
type AIDifficulty string

const (
	AIEasy   AIDifficulty = "easy"
	AIMedium AIDifficulty = "medium"
	AIHard   AIDifficulty = "hard"
)

// ParseAIDifficulty converts a string to a difficulty, defaulting to medium.
func ParseAIDifficulty(s string) AIDifficulty {
	switch AIDifficulty(s) {
	case AIEasy, AIHard:
		return AIDifficulty(s)
	default:
		return AIMedium
	}
}

// Next returns the following difficulty in the easy -> medium -> hard cycle.
func (d AIDifficulty) Next() AIDifficulty {
	switch ParseAIDifficulty(string(d)) {
	case AIEasy:
		return AIMedium
	case AIMedium:
		return AIHard
	default:
		return AIEasy
	}
}

// Label returns the display name of the difficulty.
func (d AIDifficulty) Label() string {
	switch ParseAIDifficulty(string(d)) {
	case AIEasy:
		return "Easy"
	case AIHard:
		return "Hard"
	default:
		return "Medium"
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
		AttackCards:      []CombatCard{},
		DefenseCards:     []CombatCard{},
	}
}

// ResetTurn resets per-turn counters for the player.
func (p *Player) ResetTurn() {
	p.AttacksRemaining = 2
}

