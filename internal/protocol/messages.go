// Package protocol defines the network message types for client-server communication.
package protocol

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// MessageType identifies the type of message.
type MessageType string

// Authentication message types
const (
	TypeAuthenticate MessageType = "authenticate"
	TypeAuthResult   MessageType = "auth_result"
)

// Lobby message types
const (
	TypeCreateGame     MessageType = "create_game"
	TypeGameCreated    MessageType = "game_created"
	TypeJoinGame       MessageType = "join_game"
	TypeJoinByCode     MessageType = "join_by_code"
	TypeJoinedGame     MessageType = "joined_game"
	TypeLeaveGame      MessageType = "leave_game"
	TypeDeleteGame     MessageType = "delete_game"
	TypeGameDeleted    MessageType = "game_deleted"
	TypeAddAI          MessageType = "add_ai"
	TypeRemovePlayer   MessageType = "remove_player"
	TypeUpdateSettings MessageType = "update_settings"
	TypePlayerReady    MessageType = "player_ready"
	TypeStartGame      MessageType = "start_game"
	TypeListGames      MessageType = "list_games"
	TypeGameList       MessageType = "game_list"
	TypeYourGames      MessageType = "your_games"
	TypeLobbyState     MessageType = "lobby_state"
	TypePlayerJoined   MessageType = "player_joined"
	TypePlayerLeft     MessageType = "player_left"
)

// Game flow message types
const (
	TypeGameStarted   MessageType = "game_started"
	TypePhaseChanged  MessageType = "phase_changed"
	TypeTurnChanged   MessageType = "turn_changed"
	TypeActionResult  MessageType = "action_result"
	TypeGameState     MessageType = "game_state"
	TypeGameEnded     MessageType = "game_ended"
	TypeGameHistory   MessageType = "game_history"
)

// Action message types
const (
	TypeSelectTerritory MessageType = "select_territory"
	TypePlaceStockpile  MessageType = "place_stockpile"
	TypeEndPhase        MessageType = "end_phase"
	TypeProposeTrade    MessageType = "propose_trade"
	TypeRespondTrade    MessageType = "respond_trade"
	TypeMoveStockpile   MessageType = "move_stockpile"
	TypeMoveUnit        MessageType = "move_unit"
	TypePlanAttack      MessageType = "plan_attack"
	TypeAttackPreview   MessageType = "attack_preview"
	TypeBringForces     MessageType = "bring_forces"
	TypeExecuteAttack   MessageType = "execute_attack"
	TypeCancelAttack    MessageType = "cancel_attack"
	TypeAllianceRequest MessageType = "alliance_request"
	TypeAllianceVote    MessageType = "alliance_vote"
	TypeBuild           MessageType = "build"
)

// System message types
const (
	TypeWelcome     MessageType = "welcome"
	TypeError       MessageType = "error"
	TypeReconnect   MessageType = "reconnect"
	TypeDisconnect  MessageType = "disconnect"
	TypePing        MessageType = "ping"
	TypePong        MessageType = "pong"
)

// Message is the envelope for all messages.
type Message struct {
	Type      MessageType     `json:"type"`
	ID        string          `json:"id"`
	Timestamp int64           `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

// NewMessage creates a new message with the given type and payload.
func NewMessage(msgType MessageType, payload interface{}) (*Message, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &Message{
		Type:      msgType,
		ID:        uuid.New().String(),
		Timestamp: time.Now().UnixMilli(),
		Payload:   data,
	}, nil
}

// ParsePayload unmarshals the payload into the given type.
func (m *Message) ParsePayload(v interface{}) error {
	return json.Unmarshal(m.Payload, v)
}

// ErrorCode represents an error type.
type ErrorCode string

const (
	ErrCodeInvalidAction         ErrorCode = "invalid_action"
	ErrCodeNotYourTurn           ErrorCode = "not_your_turn"
	ErrCodeInvalidTarget         ErrorCode = "invalid_target"
	ErrCodeInsufficientResources ErrorCode = "insufficient_resources"
	ErrCodeAlreadyHasUnit        ErrorCode = "already_has_unit"
	ErrCodeCannotReach           ErrorCode = "cannot_reach"
	ErrCodeAttackFailed          ErrorCode = "attack_failed"
	ErrCodeGameNotFound          ErrorCode = "game_not_found"
	ErrCodeLobbyFull             ErrorCode = "lobby_full"
	ErrCodeNotAuthenticated      ErrorCode = "not_authenticated"
	ErrCodeInternalError         ErrorCode = "internal_error"
)

// ErrorPayload is the payload for error messages.
type ErrorPayload struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}
