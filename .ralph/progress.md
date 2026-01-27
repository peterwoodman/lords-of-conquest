# Progress Log

> Updated by the agent after significant work.

---

## Session History


### 2026-01-23 08:30:30
**Session 1 started** (model: opus-4.5-thinking)

### 2026-01-23 - Session 1 Progress
**Task completed:** Fix city victory condition bug when multiple players exceed city threshold

**Changes made:**
1. Updated `IsCityVictory()` in `internal/game/state.go`:
   - Now returns true only if a player has cities >= VictoryCities AND has strictly more cities than ALL other (non-eliminated) players
   - Fixed bug where if two players both exceeded VictoryCities, the game would never end

2. Updated `GetWinner()` in `internal/game/state.go`:
   - For city victory, now returns the player with the highest city count (who is also at/above VictoryCities threshold)
   - Properly handles eliminated players

3. Created comprehensive unit tests in `internal/game/state_test.go`:
   - Tests for single player at threshold (wins)
   - Tests for two players tied at/above threshold (no winner)
   - Tests for two players above threshold with one higher (higher wins)
   - Tests for eliminated players being ignored
   - Tests for CountCities function

**Verification:**
- All 12 unit tests pass
- Server-side packages build successfully

### 2026-01-23 08:33:41
**Session 1 ended** - ✅ Task complete

### 2026-01-23 08:33:44
**Session 2 started** (model: opus-4.5-thinking)

### 2026-01-23 - Session 2 Progress
**Task completed:** Show popup when server connection fails due to sleeping server

**Changes made:**
1. Updated `ConnectScene` struct in `internal/client/scenes.go`:
   - Added `showConnectionPopup bool` field to track popup visibility
   - Added `popupOkBtn *Button` for the popup's OK button

2. Updated `NewConnectScene()`:
   - Initialized popup OK button with click handler to dismiss popup

3. Updated `OnEnter()`:
   - Reset `showConnectionPopup` to false when entering scene

4. Updated `Update()`:
   - Added check at start to handle popup input when visible
   - When popup is showing, only updates the OK button (blocks other interactions)

5. Updated `onConnect()`:
   - When connection fails to central server, shows user-friendly popup instead of status text
   - Self-hosted connections still use status text for raw error messages

6. Added `drawConnectionPopup()` method:
   - Semi-transparent overlay dims background
   - Centered popup panel with "Connection Failed" title
   - User-friendly messages: "The server may be sleeping." and "Please try again in 2 minutes."
   - Centered OK button to dismiss

**Verification:**
- Go syntax validated with gofmt
- All game tests pass
- Server-side packages build successfully (client has WSL GUI dependency issues unrelated to changes)

### 2026-01-23 08:37:20
**Session 2 ended** - ✅ Task complete

### 2026-01-23 08:37:22
**Session 3 started** (model: opus-4.5-thinking)

### 2026-01-23 - Session 3 Progress
**Task completed:** Always show attack/defense numbers in territory hover tooltip

**Changes made:**
1. Updated `drawHoverInfo()` in `internal/client/gameplay_ui.go`:
   - Removed the conditional `showAttackPreview` check that required Conquest phase, player's turn, and enemy territory
   - Combat strength preview section now always displays for all territories
   - For player's own territories: Shows "COMBAT STRENGTH" header with defense value and "(if attacked)" hint
   - For enemy/unclaimed territories: Shows "ATTACK PREVIEW" header with attack vs defense values and odds indicator

2. Simplified variable declarations:
   - Replaced `isMyTurn`, `isEnemy`, `showAttackPreview` with single `isOwnTerritory` boolean
   - Changed `attackPreviewHeight` to `strengthPreviewHeight` (always 65)

**User benefit:**
- Players can now always see combat strength information when hovering any territory
- Helps with strategic planning in all game phases, not just during Conquest
- Own territories show defensive strength; other territories show attack vs defense comparison

**Verification:**
- All 12 game unit tests pass
- Server-side packages build successfully
- Go syntax validated with gofmt

### 2026-01-23 08:40:19
**Session 3 ended** - ✅ Task complete

### 2026-01-27 11:53:44
**Session 1 started** (model: opus-4.5-thinking)

### 2026-01-27 - Session 1 Progress
**Task completed:** Fix lobby map change not applied when game starts - always uses first map

**Changes made:**
1. Updated `initializeGameState()` in `internal/server/handlers.go`:
   - Changed map loading logic to prioritize the stored `map_json` column over `Settings.MapID`
   - Now first calls `loadMapFromDatabase()` to check for host's map changes
   - Falls back to `maps.Get(Settings.MapID)` only if no stored map exists in database

**Root cause:**
- When host changed the map via UpdateMap, only the `map_json` database column was updated
- When game started, `initializeGameState()` only consulted `maps.Get(Settings.MapID)` which looked up from in-memory registry using the original map ID
- The stored `map_json` (containing the host's new map) was never used

**Verification:**
- Server package builds successfully
- All 12 game unit tests pass

### 2026-01-27 11:55:29
**Session 1 ended** - ✅ Task complete

### 2026-01-27 11:55:31
**Session 2 started** (model: opus-4.5-thinking)

### 2026-01-27 - Session 2 Progress
**Task completed:** Fix trade dialog player buttons jumping around and sometimes missing players

**Changes made:**
1. Added `sort` import to `internal/client/gameplay_handlers.go`
2. Added `sort.Strings(players)` call before returning from `getOnlinePlayers()` function

**Root cause:**
- Go map iteration order is random, so iterating over `s.players` returned different orders each call
- `getOnlinePlayers()` is called separately in Draw (rendering buttons) and Update (handling clicks)
- Different orders caused button positions to mismatch between rendering and click detection
- Adding `sort.Strings()` ensures alphabetical ordering by player ID, making the order deterministic

**Verification:**
- All 12 game unit tests pass
- Server-side packages build successfully
- Go syntax validated with gofmt

### 2026-01-27 11:57:09
**Session 2 ended** - ✅ Task complete

### 2026-01-27 11:57:11
**Session 3 started** (model: opus-4.5-thinking)

### 2026-01-27 - Session 3 Progress
**Task completed:** Fix turn order rotation - should rotate each year instead of re-randomizing

**Changes made:**
1. Added `rotatePlayerOrder()` method to PhaseManager in `internal/game/phases.go`:
   - Moves first player to end of order, shifts everyone else up
   - Properly handles eliminated players (removes them from order)
   - Includes logging for debugging

2. Added standalone `rotatePlayerOrder(state *GameState)` function in `internal/game/initialize.go`:
   - Same logic as PhaseManager method for use by GameState methods

3. Updated `NextPhase()` in `internal/game/phases.go` (line 210):
   - Changed from `shufflePlayerOrder()` to `rotatePlayerOrder()` when advancing from Conquest to Development (Year 2+)

4. Updated `startNewRound()` in `internal/game/development.go` (line 352):
   - Changed from `shufflePlayerOrder(g)` to `rotatePlayerOrder(g)`

5. Kept `shufflePlayerOrder()` for Year 1:
   - Territory selection → Production still uses shuffle (line 157 in phases.go)
   - `startFirstRound()` still uses shuffle (line 75 in selection.go)

**Behavior:**
- Year 1: Random player order (fair starting conditions)
- Year 2+: First player rotates to last position (ensures fairness across rounds)

**Verification:**
- All 12 game unit tests pass
- Server package builds successfully

### 2026-01-27 12:01:01
**Session 3 ended** - ✅ Task complete

### 2026-01-27 12:01:03
**Session 4 started** (model: opus-4.5-thinking)

### 2026-01-27 - Session 4 Progress
**Task completed:** Fix boats being able to attack landlocked territories or territories not in the same water body

**Changes made:**
1. Updated `canBoatReachTargetViaWater()` in `internal/game/conquest.go`:
   - Added `attackerID` parameter to the function signature
   - Modified the second check (lines 170-175) to verify the attacker owns the coastal territory
   - Previous code allowed attacking if target was adjacent to ANY coastal territory in the water body
   - New code requires target to be adjacent to a coastal territory the ATTACKER owns

2. Updated call sites:
   - `GetAttackPlan()` at line 85: passes `attackerID` to the function
   - `AttackWithAllies()` at line 212: passes `attackerID` to the function

**Root cause:**
- The `canBoatReachTargetViaWater` function's second check allowed boats to attack any territory adjacent to any coastal territory in the water body, regardless of ownership
- This allowed boats to effectively "teleport" to attack inland territories they shouldn't be able to reach
- The fix now correctly requires the attacker to own the coastal territory being used as a "landing point"

**Valid boat attack scenarios (after fix):**
1. Direct water attack: Target is coastal and shares the same water body
2. Landing attack: Target is adjacent to a coastal territory the attacker owns in that water body

**Verification:**
- All 12 game unit tests pass
- Server package builds successfully

### 2026-01-27
**Session 4 ended** - ✅ Task complete

### 2026-01-27 12:03:20
**Session 4 ended** - ✅ Task complete

### 2026-01-27 12:03:22
**Session 5 started** (model: opus-4.5-thinking)

### 2026-01-27 - Session 5 Progress
**Task completed:** Fix lobby: cannot join public game after selecting from Active games list

**Changes made:**
1. Updated `yourGameList.OnSelect` callback in `internal/client/scenes.go` (line 664-667):
   - Now calls `s.gameList.ClearSelection()` when an item is selected
   - This immediately clears the other list's visual selection

2. Updated `gameList.OnSelect` callback in `internal/client/scenes.go` (line 671-674):
   - Now calls `s.yourGameList.ClearSelection()` when an item is selected
   - This immediately clears the other list's visual selection

3. Removed buggy code at lines 809-816 that ran every frame:
   - The old code always prioritized `yourGameList` in its if-else structure
   - When `yourGameList` had any selection, it would clear `gameList`'s selection every frame
   - This prevented clicks on `gameList` from registering when `yourGameList` had a selection

**Root cause:**
- Both lists maintained their own `selectedIdx` independently
- The frame-by-frame synchronization code had a flawed if-else structure that always favored `yourGameList`
- When clicking `gameList`, the click would set its selection, but the next line of code would immediately clear it

**Fix approach:**
- Handle mutual exclusivity at the moment of click (in OnSelect callbacks)
- Each list clears the OTHER list's selection when clicked
- Removed the broken every-frame synchronization code

**Verification:**
- All 12 game unit tests pass
- Server package builds successfully
- Go syntax validated with gofmt

### 2026-01-27 12:05:44
**Session 5 ended** - ✅ Task complete

### 2026-01-27 12:05:46
**Session 6 started** (model: opus-4.5-thinking)

### 2026-01-27 - Session 6 Progress
**Task completed:** Fix territory hover popup not showing defense from boats

**Changes made:**
1. Updated `calculateCombatStrength()` in `internal/client/gameplay_handlers.go`:
   - Added boat counting for target territory defense after line 415
   - Iterates over `target["boats"]` map and adds `count * 2` for each boat
   - This matches server behavior in `combat.go` line 120: `strength += target.TotalBoats() * 2`

2. Updated comment on `getTerritoryStrength()`:
   - Clarified that boats are NOT counted here for attack (must be brought as reinforcements)
   - Clarified that boats ARE counted for defense but handled separately in calculateCombatStrength

**Root cause:**
- Server's `CalculateDefenseStrength` in combat.go adds +2 per boat in the target territory
- Client's `calculateCombatStrength` was calling `getTerritoryStrength` which explicitly skipped boats
- This caused the hover popup to show lower defense than actual combat would use

**Verification:**
- All 12 game unit tests pass
- Server package builds successfully
- Go syntax validated with gofmt (no formatting issues)

### 2026-01-27
**Session 6 ended** - ✅ Task complete

### 2026-01-27 12:08:16
**Session 6 ended** - ✅ Task complete

### 2026-01-27 12:08:18
**Session 7 started** (model: opus-4.5-thinking)

### 2026-01-27 - Session 7 Progress
**Task completed:** Add toast notification when it becomes your turn

**Changes made:**
1. Added toast state fields to `GameplayScene` in `internal/client/gameplay.go`:
   - `showTurnToast`, `turnToastTimer`, `turnToastPhase`, `initialTurnLoad`

2. Added timing constants at top of gameplay.go:
   - `ToastSlideInFrames = 15` (0.25s)
   - `ToastHoldFrames = 90` (1.5s)
   - `ToastSlideOutFrames = 15` (0.25s)

3. Updated `OnEnter()` to initialize `initialTurnLoad = true` to prevent toast on initial game load

4. Updated `applyGameState()` in `internal/client/gameplay_util.go`:
   - Added detection for when turn changes TO the local player
   - Triggers toast only when turn changes, not on initial load
   - Clears `initialTurnLoad` flag after first state update

5. Added toast animation update logic in `Update()`:
   - Progresses through slide-in → hold → slide-out phases
   - Runs independently of other animations (non-blocking)

6. Created `drawTurnToast()` function in `internal/client/gameplay_ui.go`:
   - Draws horizontal banner at top-center of screen
   - Smooth slide animation from off-screen (-60) to visible (20)
   - Shows "YOUR TURN!" in large text with current phase below
   - Uses player's color as accent border/glow
   - 400px wide, 60px tall

7. Added `drawTurnToast()` call in `Draw()` after hover info but before modal overlays

**Verification:**
- Server package builds successfully
- All 12 game unit tests pass
- Go syntax validated with gofmt

### 2026-01-27
**Session 7 ended** - ✅ Task complete
