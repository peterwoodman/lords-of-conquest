package game

import "errors"

// Game errors
var (
	ErrNotYourTurn           = errors.New("not your turn")
	ErrInvalidAction         = errors.New("invalid action for current phase")
	ErrInvalidTarget         = errors.New("invalid target")
	ErrInsufficientResources = errors.New("insufficient resources")
	ErrAlreadyHasUnit        = errors.New("territory already has this unit type")
	ErrCannotReach           = errors.New("unit cannot reach destination")
	ErrTerritoryOccupied     = errors.New("territory already claimed")
	ErrNoAttacksRemaining    = errors.New("no attacks remaining this turn")
	ErrAttackFailed          = errors.New("attack was not successful")
	ErrGameNotStarted        = errors.New("game has not started")
	ErrGameOver              = errors.New("game is over")
	ErrPlayerEliminated      = errors.New("player has been eliminated")
)

