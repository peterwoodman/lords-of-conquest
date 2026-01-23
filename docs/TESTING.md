# Testing Territory Selection

## What Should Work

1. **Start Server**
   ```bash
   go run ./cmd/server
   ```
   Should see: "Loaded 1 maps - Test Map: 20x12, 8 territories"

2. **Start Client**
   ```bash
   go run ./cmd/client
   ```
   
3. **Connect & Create Game**
   - Enter server: `localhost:8080`
   - Enter your name
   - Click "Create Game"
   - Name it, choose public/private
   - Click "Create"

4. **Add a Second Player** (for testing)
   - Click "Add AI Player"
   - Set ready
   - Click "Start Game"

5. **Territory Selection Should Begin**
   - Map renders (8 colored territories, blue water)
   - Info panel shows "Round 0 - Territory Selection"
   - Shows current player's turn
   - "YOUR TURN" indicator when it's your turn

6. **Select Territories**
   - Hover over territories to see info tooltip
   - Click an unclaimed (gray) territory when it's your turn
   - Territory changes to your player color
   - Turn automatically advances to next player
   - AI takes its turn automatically

7. **Complete Selection**
   - After all territories claimed (except <2), moves to Production phase
   - Would show "Round 1 - Production" 

## What To Look For

✅ **Working:**
- Map renders with correct size
- Territories show in different colors
- Water is blue
- Hover shows territory name, owner, resource
- Click selects territory
- Turn indicator updates
- Your territories show in your color

❌ **Potential Issues:**
- If map doesn't show: Check console for errors
- If click doesn't work: Make sure it's your turn
- If AI doesn't move: Need to implement AI (not done yet)

## Current Limitation

**AI players don't actually play yet** - you'll need at least 2 human players to test fully, OR manually control both players by:
1. Open client twice
2. Join same game with both
3. Take turns clicking

## Next Steps After This Works

1. AI territory selection
2. Stockpile placement UI
3. Production phase display
4. Development phase (build cities/weapons)

