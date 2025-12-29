# Lords of Conquest - Network Protocol

## Overview

Communication between client and server uses WebSocket with JSON-encoded messages. All messages follow a common envelope structure.

## Message Envelope

```json
{
  "type": "message_type",
  "id": "unique-message-id",
  "timestamp": 1703875200000,
  "payload": { ... }
}
```

## Connection Flow

```
Client                          Server
  |                               |
  |------ Connect (WS) --------->|
  |<----- Welcome ----------------|
  |                               |
  |------ Authenticate --------->|
  |<----- AuthResult -------------|
  |                               |
  |------ JoinLobby ------------>|
  |<----- LobbyState -------------|
  |                               |
```

---

## Lobby Messages

### Client → Server

#### `create_game`
Create a new game lobby.
```json
{
  "type": "create_game",
  "payload": {
    "name": "My Game",
    "settings": {
      "max_players": 4,
      "game_level": "expert",
      "chance_level": "medium",
      "victory_cities": 3,
      "map_id": "north_america"
    }
  }
}
```

#### `join_game`
Join an existing game.
```json
{
  "type": "join_game",
  "payload": {
    "game_id": "game-uuid",
    "player_name": "Player1",
    "preferred_color": "orange"
  }
}
```

#### `leave_game`
Leave the current game.
```json
{
  "type": "leave_game",
  "payload": {}
}
```

#### `add_ai`
Add an AI player (host only).
```json
{
  "type": "add_ai",
  "payload": {
    "personality": "aggressive",
    "name": "CPU-1"
  }
}
```

#### `update_settings`
Update game settings (host only).
```json
{
  "type": "update_settings",
  "payload": {
    "settings": {
      "victory_cities": 4
    }
  }
}
```

#### `player_ready`
Toggle ready state.
```json
{
  "type": "player_ready",
  "payload": {
    "ready": true
  }
}
```

#### `start_game`
Start the game (host only, all players must be ready).
```json
{
  "type": "start_game",
  "payload": {}
}
```

### Server → Client

#### `game_list`
List of available games.
```json
{
  "type": "game_list",
  "payload": {
    "games": [
      {
        "id": "game-uuid",
        "name": "My Game",
        "host": "Player1",
        "players": 2,
        "max_players": 4,
        "status": "waiting"
      }
    ]
  }
}
```

#### `lobby_state`
Current lobby state.
```json
{
  "type": "lobby_state",
  "payload": {
    "game_id": "game-uuid",
    "host_id": "player-uuid",
    "settings": { ... },
    "players": [
      {
        "id": "player-uuid",
        "name": "Player1",
        "color": "orange",
        "is_ai": false,
        "ai_personality": null,
        "ready": true
      }
    ]
  }
}
```

---

## Game Flow Messages

### Server → Client

#### `game_started`
Game has begun, includes initial state.
```json
{
  "type": "game_started",
  "payload": {
    "game_state": { ... },
    "your_player_id": "player-uuid"
  }
}
```

#### `phase_changed`
New phase has started.
```json
{
  "type": "phase_changed",
  "payload": {
    "phase": "production",
    "round": 3,
    "skipped": false,
    "current_player": "player-uuid"
  }
}
```

#### `turn_changed`
Active player changed within phase.
```json
{
  "type": "turn_changed",
  "payload": {
    "current_player": "player-uuid",
    "time_limit": 60
  }
}
```

#### `action_result`
Result of a player action.
```json
{
  "type": "action_result",
  "payload": {
    "action_id": "action-uuid",
    "success": true,
    "error": null,
    "state_update": { ... }
  }
}
```

#### `game_state`
Full game state synchronization.
```json
{
  "type": "game_state",
  "payload": {
    "state": { ... }
  }
}
```

#### `game_ended`
Game has concluded.
```json
{
  "type": "game_ended",
  "payload": {
    "winner": "player-uuid",
    "reason": "cities",
    "final_state": { ... }
  }
}
```

---

## Game Action Messages

### Client → Server

#### `select_territory`
(Territory Selection Phase) Claim a territory.
```json
{
  "type": "select_territory",
  "payload": {
    "territory_id": "territory-1"
  }
}
```

#### `place_stockpile`
(First Production Phase) Place initial stockpile.
```json
{
  "type": "place_stockpile",
  "payload": {
    "territory_id": "territory-5"
  }
}
```

#### `end_phase`
Voluntarily end current phase.
```json
{
  "type": "end_phase",
  "payload": {}
}
```

### Trade Phase

#### `propose_trade`
Offer a trade to another player.
```json
{
  "type": "propose_trade",
  "payload": {
    "target_player": "player-uuid",
    "offer": {
      "coal": 2,
      "gold": 0,
      "iron": 0,
      "timber": 1,
      "horses": 0
    },
    "request": {
      "coal": 0,
      "gold": 1,
      "iron": 0,
      "timber": 0,
      "horses": 0
    }
  }
}
```

#### `respond_trade`
Accept or reject a trade offer.
```json
{
  "type": "respond_trade",
  "payload": {
    "trade_id": "trade-uuid",
    "accepted": true
  }
}
```

### Shipment Phase

#### `move_stockpile`
Move stockpile to new territory.
```json
{
  "type": "move_stockpile",
  "payload": {
    "destination": "territory-10"
  }
}
```

#### `move_unit`
Move a unit (Expert level only).
```json
{
  "type": "move_unit",
  "payload": {
    "unit_type": "horse",
    "from": "territory-5",
    "to": "territory-8",
    "carry_weapon": true
  }
}
```

### Conquest Phase

#### `plan_attack`
Begin planning an attack (get strength preview).
```json
{
  "type": "plan_attack",
  "payload": {
    "target_territory": "territory-12"
  }
}
```

#### `attack_preview`
Server response with combat preview.
```json
{
  "type": "attack_preview",
  "payload": {
    "target_territory": "territory-12",
    "attack_strength": 3,
    "defense_strength": 4,
    "can_attack": false,
    "available_reinforcements": [
      {
        "unit_type": "horse",
        "from": "territory-7",
        "strength_bonus": 1,
        "can_carry_weapon": true,
        "weapon_available_at": "territory-7"
      },
      {
        "unit_type": "boat",
        "from": "territory-2",
        "strength_bonus": 2,
        "can_carry_horse": true,
        "can_carry_weapon": true
      }
    ]
  }
}
```

#### `bring_forces`
Add reinforcement to planned attack.
```json
{
  "type": "bring_forces",
  "payload": {
    "unit_type": "horse",
    "from": "territory-7",
    "pickup_weapon_at": "territory-7"
  }
}
```

#### `execute_attack`
Execute the planned attack.
```json
{
  "type": "execute_attack",
  "payload": {}
}
```

#### `cancel_attack`
Cancel planned attack.
```json
{
  "type": "cancel_attack",
  "payload": {}
}
```

### Alliance Voting (3+ players)

#### `alliance_request`
Server notifies adjacent players of attack.
```json
{
  "type": "alliance_request",
  "payload": {
    "attacker": "player-uuid",
    "defender": "player-uuid",
    "territory": "territory-12",
    "your_strength": 2,
    "time_limit": 15
  }
}
```

#### `alliance_vote`
Player chooses side.
```json
{
  "type": "alliance_vote",
  "payload": {
    "battle_id": "battle-uuid",
    "side": "attacker"  // or "defender" or "neutral"
  }
}
```

### Development Phase

#### `build`
Build a unit or city.
```json
{
  "type": "build",
  "payload": {
    "type": "city",
    "territory": "territory-15",
    "use_gold": false
  }
}
```

---

## Game State Structure

### Full State Object
```json
{
  "game_id": "game-uuid",
  "settings": {
    "game_level": "expert",
    "chance_level": "medium",
    "victory_cities": 3
  },
  "round": 3,
  "phase": "conquest",
  "current_player_index": 1,
  "player_order": ["player-1", "player-2", "player-3"],
  "players": {
    "player-1": {
      "id": "player-1",
      "name": "Alice",
      "color": "orange",
      "stockpile": {
        "coal": 3,
        "gold": 2,
        "iron": 1,
        "timber": 4,
        "horses": 0
      },
      "stockpile_territory": "territory-5",
      "attacks_remaining": 2,
      "eliminated": false
    }
  },
  "territories": {
    "territory-1": {
      "id": "territory-1",
      "name": "Alaska",
      "owner": "player-1",
      "resource": "coal",
      "has_city": false,
      "has_weapon": false,
      "has_horse": true,
      "boats": 0,
      "adjacent": ["territory-2", "territory-3"],
      "coastal_tiles": 2,
      "water_bodies": ["pacific"]
    }
  },
  "water_bodies": {
    "pacific": {
      "id": "pacific",
      "territories": ["territory-1", "territory-4", "territory-7"]
    }
  }
}
```

---

## Error Codes

| Code | Description |
|------|-------------|
| `invalid_action` | Action not allowed in current phase |
| `not_your_turn` | Not the player's turn |
| `invalid_target` | Invalid territory or player target |
| `insufficient_resources` | Not enough resources |
| `already_has_unit` | Territory already has this unit type |
| `cannot_reach` | Unit cannot reach destination |
| `attack_failed` | Attack was not successful |
| `game_not_found` | Game ID doesn't exist |
| `lobby_full` | Game is at max players |

---

## Reconnection

When a player disconnects and reconnects:

1. Client sends `reconnect` with session token
2. Server validates token and restores session
3. Server sends full `game_state`
4. If it's the player's turn, they resume with remaining time

```json
{
  "type": "reconnect",
  "payload": {
    "session_token": "token-uuid",
    "game_id": "game-uuid"
  }
}
```

Players have a configurable timeout (default 60s) before being replaced by AI or forfeiting.

