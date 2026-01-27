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

### 2026-01-27 12:11:26
**Session 7 ended** - ✅ Task complete

### 2026-01-27 12:11:28
**Session 8 started** (model: opus-4.5-thinking)

### 2026-01-27 - Session 8 Progress
**Task completed:** Reset 'Use Gold' toggle to off at the start of each development phase

**Changes made:**
1. Updated `applyGameState()` in `internal/client/gameplay_util.go`:
   - Added reset logic inside the phase change detection block (lines 86-90)
   - When phase changes to "Development", resets `s.buildUseGold = false` and `s.selectedBuildType = ""`
   - Ensures consistent starting state each development phase

**User benefit:**
- Previously, if player enabled "Use Gold" toggle in one development phase, it remained enabled for the next phase
- Now the toggle resets to "off" at the start of each development phase for consistent UX
- Also clears any selected build type (city/weapon/boat) from previous phase

**Verification:**
- All 12 game unit tests pass
- Server package builds successfully

### 2026-01-27
**Session 8 ended** - ✅ Task complete

### 2026-01-27 12:13:06
**Session 8 ended** - ✅ Task complete

### 2026-01-27 12:13:08
**Session 9 started** (model: opus-4.5-thinking)

### 2026-01-27 - Session 9 Progress
**Task completed:** Change alliance behavior: specific player ally should only auto-join for that player, treat other combat as 'ask'

**Changes made:**
1. Updated `handleExecuteAttack()` in `internal/server/handlers.go` (lines 2799-2808):
   - Changed the `else` branch for specific-player alliances where the allied player isn't the attacker or defender
   - Previous behavior: logged "not participating" and skipped (treated as neutral)
   - New behavior: if online, adds player to `askPlayers` list (prompts them to choose a side); if offline, treats as neutral

**Why this matters:**
- Before: If Player A allied with Player B, and Player C attacked Player D, Player A would be neutral (not participate)
- After: Player A will be prompted to choose a side (attack/defend/neutral) in that combat
- More intuitive: "I'm allied with Bob" now means "auto-join Bob's battles, ask me about others" instead of "auto-neutral in battles Bob isn't in"

**Verification:**
- Server package builds successfully
- All game unit tests pass

### 2026-01-27
**Session 9 ended** - ✅ Task complete

### 2026-01-27 12:14:49
**Session 9 ended** - ✅ Task complete

### 2026-01-27 12:14:51
**Session 10 started** (model: opus-4.5-thinking)

### 2026-01-27 - Session 10 Progress
**Task completed:** Increase maximum resource percentage in map generation from 80% to 100%

**Changes made:**
1. Updated `GeneratorOptions.Resources` comment in `pkg/maps/generator.go` line 16:
   - Changed "10-80" to "10-100"

2. Updated `assignResources()` in `pkg/maps/generator.go` line 759-760:
   - Changed comment from "(10-80)" to "(10-100)"
   - Changed `clamp(g.options.Resources, 10, 80)` to `clamp(g.options.Resources, 10, 100)`

3. Updated `ResourcesSlider` in `internal/client/ui.go` line 848:
   - Changed `Max: 80` to `Max: 100`

**User benefit:**
- Players can now generate maps where up to 100% of territories have resources assigned
- Previous 80% cap could leave some territories without resources

**Verification:**
- All game unit tests pass
- Server package builds successfully

### 2026-01-27
**Session 10 ended** - ✅ Task complete

### 2026-01-27 12:16:02
**Session 10 ended** - ✅ Task complete

### 2026-01-27 12:16:04
**Session 11 started** (model: opus-4.5-thinking)

### 2026-01-27 - Session 11 Progress
**Task completed:** Show build costs permanently instead of in tooltips for city/weapon/boat buttons

**Changes made:**
1. Updated `drawDevelopmentControls()` in `internal/client/gameplay_ui.go`:
   - Removed tooltip assignments from city, weapon, and boat buttons (set to empty string)
   - Removed unused tooltip string variables (`cityTooltip`, `weaponTooltip`, `boatTooltip`)
   - Added cost labels below each button showing both normal and gold costs
   - City: "1 each/4G" (1 of each resource or 4 gold)
   - Weapon: "1C+1I/2G" (1 coal + 1 iron or 2 gold)
   - Boat: "3T/3G" (3 timber or 3 gold)

2. Implemented cost highlighting based on Use Gold toggle:
   - When Use Gold is OFF: normal cost uses `ColorTextMuted`, gold cost uses `ColorTextDim`
   - When Use Gold is ON: normal cost uses `ColorTextDim`, gold cost uses `ColorTextMuted`
   - This subtly highlights which cost type is currently active

**User benefit:**
- Build costs are now always visible below the buttons without requiring hover
- Both cost options (normal resources and gold-only) are shown simultaneously
- Active cost type is subtly highlighted based on the Use Gold toggle state

**Verification:**
- All 12 game unit tests pass
- Go syntax validated with gofmt (no formatting issues)
- Server-side packages build successfully

### 2026-01-27
**Session 11 ended** - ✅ Task complete

### 2026-01-27 12:20:12
**Session 11 ended** - ✅ Task complete

### 2026-01-27 12:20:14
**Session 12 started** (model: opus-4.5-thinking)

### 2026-01-27 - Session 12 Progress
**Task completed:** Show map preview inline in game lobby instead of View Map button with popup

**Changes made:**
1. Removed popup-related fields from `WaitingScene` struct:
   - Removed `viewMapBtn`, `showMapPreview`, `mapCloseBtn`

2. Updated layout constants in `NewWaitingScene()`:
   - Changed to 3-column layout: player list, map preview, actions
   - `rightPanelX`: 700 → 920
   - `btnW`: 200 → 170 (narrower buttons to fit)
   - `playerList` width: 500 → 380 (narrower to make room)

3. Updated `WaitingScene.Update()`:
   - Removed map preview dialog input handling
   - Removed `viewMapBtn.Update()` call

4. Updated `WaitingScene.Draw()`:
   - Added `mapPreviewX := 490` for center panel
   - Player list panel: narrowed to 400px
   - Added inline map preview panel (400x470) in center
   - Actions panel: moved to right edge (x=910, width=200)
   - Removed map preview dialog overlay
   - Adjusted non-host status text position

5. Replaced `drawMapPreviewDialog()` with `drawInlineMapPreview()`:
   - Takes panel coordinates as parameters
   - Draws panel with title "Map Preview"
   - Reuses same map grid rendering logic
   - Shows map info (size, territory count) at bottom

**Layout summary:**
- Player list panel: x=70, w=400 (80-470)
- Map preview panel: x=490, w=400 (490-890)
- Actions panel: x=910, w=200 (910-1110)
- All panels: y=170, h=470

**User benefit:**
- Map preview is now always visible in the lobby without requiring a click
- Both host and non-host players see the same inline map preview
- Map updates live when host changes the map (since it reads from lobby.MapData each frame)

**Verification:**
- All 12 game unit tests pass
- Go syntax validated with gofmt
- Server-side packages build successfully

### 2026-01-27 12:31:43
**Session 12 ended** - ✅ Task complete

### 2026-01-27 12:31:45
**Session 13 started** (model: opus-4.5-thinking)

### 2026-01-27 - Session 13 Progress
**Task completed:** Add 'Plan Attack' step with alliance waiting and confirmation dialog before executing attack

**Changes made:**

1. **Protocol changes** (`internal/protocol/messages.go`, `internal/protocol/payloads.go`):
   - Added `TypeRequestAttackPlan` message type for requesting alliance resolution
   - Added `TypeAttackPlanResolved` message type for server response with resolved totals
   - Added `RequestAttackPlanPayload` with attack parameters
   - Added `AttackPlanResolvedPayload` with resolved ally strengths and names
   - Added `PlanID` field to `ExecuteAttackPayload` to reference cached plans

2. **Server changes** (`internal/server/server.go`, `internal/server/handlers.go`):
   - Added `PendingAttackPlan` struct to cache resolved attack plans
   - Added `pendingAttackPlans` map to Hub for storing plans
   - Added `handleRequestAttackPlan()` handler that:
     - Triggers alliance voting (same as executeAttack used to do)
     - Waits for votes with 60 second timeout
     - Stores resolved plan with attacker/defender ally lists
     - Returns resolved totals to client
   - Updated `handleExecuteAttack()` to:
     - Check for cached plan by PlanID
     - Use pre-resolved allies if plan exists and is valid
     - Fall back to live alliance resolution if no cached plan (legacy support)

3. **Client changes** (`internal/client/gameplay.go`, `internal/client/gameplay_dialogs.go`, `internal/client/client.go`):
   - Added state fields: `showWaitingForAlliance`, `showAttackConfirmation`, `attackPlanResolved`, `confirmAttackBtn`, `cancelConfirmBtn`
   - Changed button text from "Attack"/"Attack Without" to "Plan Attack"/"Plan Without"
   - Changed "With [unit]" to "Plan w/ [unit]"
   - Updated `doAttack()` to send `RequestAttackPlan` instead of `ExecuteAttack`
   - Added `drawWaitingForAlliance()` overlay shown during alliance resolution
   - Added `drawAttackConfirmation()` dialog showing:
     - Attack forces breakdown (your forces + ally names and totals)
     - Defense forces breakdown (base defense + ally names and totals)
     - "Confirm Attack" and "Cancel" buttons
   - Added `ShowAttackConfirmation()`, `confirmAttack()`, `cancelAttackConfirmation()` methods
   - Added `ExecuteAttackWithPlan()` and `RequestAttackPlan()` network methods
   - Added handler for `TypeAttackPlanResolved` message

**User benefit:**
- Players now see a confirmation dialog with final ally contributions BEFORE committing to attack
- Attackers can see who will join their attack and who will defend
- Cancel option allows backing out without consuming attacks if odds look bad
- Alliance voting happens during "plan" phase, not during execution

**Verification:**
- All 12 game unit tests pass
- Server and protocol packages build successfully
- Go syntax validated with gofmt

### 2026-01-27 12:39:22
**Session 13 ended** - ✅ Task complete

### 2026-01-27 12:39:24
**Session 14 started** (model: opus-4.5-thinking)
