package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"lords-of-conquest/internal/database"
	"lords-of-conquest/internal/protocol"
	"lords-of-conquest/pkg/maps"

	"github.com/coder/websocket"
)

// testClient is a minimal scripted player talking to a real server over WebSocket.
type testClient struct {
	t        *testing.T
	ws       *websocket.Conn
	playerID string
}

func (c *testClient) send(msgType protocol.MessageType, payload interface{}) {
	c.t.Helper()
	msg, err := protocol.NewMessage(msgType, payload)
	if err != nil {
		c.t.Fatal(err)
	}
	data, _ := json.Marshal(msg)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.ws.Write(ctx, websocket.MessageText, data); err != nil {
		c.t.Fatalf("write %s: %v", msgType, err)
	}
}

func (c *testClient) read(ctx context.Context) (*protocol.Message, error) {
	_, data, err := c.ws.Read(ctx)
	if err != nil {
		return nil, err
	}
	var msg protocol.Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// waitFor reads until a message of the given type arrives, failing on server errors.
func (c *testClient) waitFor(ctx context.Context, msgType protocol.MessageType) *protocol.Message {
	c.t.Helper()
	for {
		msg, err := c.read(ctx)
		if err != nil {
			c.t.Fatalf("waiting for %s: %v", msgType, err)
		}
		if msg.Type == protocol.TypeError {
			c.t.Fatalf("server error while waiting for %s: %s", msgType, msg.Payload)
		}
		if msg.Type == msgType {
			return msg
		}
	}
}

func startTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	db, err := database.New(filepath.Join(t.TempDir(), "lords.db"))
	if err != nil {
		t.Fatal(err)
	}
	s := &Server{db: db}
	s.hub = NewHub(s)

	ts := httptest.NewServer(http.HandlerFunc(s.handleWebSocket))
	t.Cleanup(func() {
		ts.Close()
		db.Close()
	})
	return ts
}

func dialTestClient(t *testing.T, ts *httptest.Server, name string) *testClient {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ws, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(ts.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	ws.SetReadLimit(8 * 1024 * 1024)
	t.Cleanup(func() { ws.Close(websocket.StatusNormalClosure, "") })

	c := &testClient{t: t, ws: ws}
	c.send(protocol.TypeAuthenticate, protocol.AuthenticatePayload{Name: name})
	var auth protocol.AuthResultPayload
	if err := c.waitFor(ctx, protocol.TypeAuthResult).ParsePayload(&auth); err != nil || !auth.Success {
		t.Fatalf("authentication failed: %v %+v", err, auth)
	}
	c.playerID = auth.PlayerID
	return c
}

func generatedMapData() *protocol.MapData {
	m, _ := maps.NewGenerator(maps.GeneratorOptions{Width: 30, Territories: 24, Islands: 1, Resources: 50}).Generate()
	data := &protocol.MapData{
		ID:          m.ID,
		Name:        m.Name,
		Width:       m.Width,
		Height:      m.Height,
		Grid:        m.Grid,
		Territories: make(map[string]protocol.TerritoryInfo),
	}
	for id, terr := range m.Territories {
		data.Territories[fmt.Sprintf("%d", id)] = protocol.TerritoryInfo{Name: terr.Name, Resource: terr.Resource.String()}
	}
	return data
}

// TestGameAgainstAIReachesRoundThree plays a full game loop against two AI players. It fails
// on a timeout if any turn, production, or acknowledgment step deadlocks.
func TestGameAgainstAIReachesRoundThree(t *testing.T) {
	for _, mode := range []string{"classic", "cards"} {
		t.Run(mode, func(t *testing.T) {
			ts := startTestServer(t)
			c := dialTestClient(t, ts, "Tester")

			ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
			defer cancel()

			c.send(protocol.TypeCreateGame, protocol.CreateGamePayload{
				Name:     "Test",
				Settings: protocol.GameSettings{MaxPlayers: 4, VictoryCities: 99, ChanceLevel: "low", CombatMode: mode},
				MapData:  generatedMapData(),
			})
			// Sent back to back: the server must handle one client's messages in order
			c.send(protocol.TypeAddAI, protocol.AddAIPayload{Personality: "aggressive"})
			c.send(protocol.TypeAddAI, protocol.AddAIPayload{Personality: "aggressive"})
			c.send(protocol.TypePlayerReady, protocol.PlayerReadyPayload{Ready: true})
			c.send(protocol.TypeStartGame, struct{}{})

			playUntilRound(t, c, ctx, 3)
		})
	}
}

func playUntilRound(t *testing.T, c *testClient, ctx context.Context, targetRound int) {
	t.Helper()
	lastActedOn := ""
	for {
		msg, err := c.read(ctx)
		if err != nil {
			t.Fatalf("game stalled before round %d: %v", targetRound, err)
		}

		var generic map[string]interface{}
		_ = json.Unmarshal(msg.Payload, &generic)
		if eventID, ok := generic["event_id"].(string); ok && eventID != "" {
			c.send(protocol.TypeClientReady, protocol.ClientReadyPayload{EventID: eventID})
		}

		switch msg.Type {
		case protocol.TypeError:
			t.Logf("server error: %s", msg.Payload)
		case protocol.TypeDefenseCardRequest:
			battleID, _ := generic["battle_id"].(string)
			c.send(protocol.TypeSelectDefenseCards, protocol.SelectCardsPayload{BattleID: battleID})
		case protocol.TypeGameEnded:
			return
		case protocol.TypeGameState:
			state := generic["state"].(map[string]interface{})
			round := int(state["round"].(float64))
			if round >= targetRound {
				return
			}

			phase := state["phase"].(string)
			current := state["currentPlayerId"].(string)
			territories := state["territories"].(map[string]interface{})
			me := state["players"].(map[string]interface{})[c.playerID].(map[string]interface{})

			if phase == "Production" && state["stockpilePlacementPending"] == true && me["stockpileTerritory"] == nil {
				for id, terr := range territories {
					if terr.(map[string]interface{})["owner"] == c.playerID {
						c.send(protocol.TypePlaceStockpile, protocol.PlaceStockpilePayload{TerritoryID: id})
						break
					}
				}
				continue
			}

			if current != c.playerID {
				continue
			}
			// Avoid repeating an action for the same turn when several states arrive
			turnKey := fmt.Sprintf("%d/%s/%v/%d", round, phase, state["commerceSubPhase"], ownedCount(territories, c.playerID))
			if turnKey == lastActedOn {
				continue
			}
			lastActedOn = turnKey

			switch phase {
			case "Territory Selection":
				for id, terr := range territories {
					if terr.(map[string]interface{})["owner"] == "" {
						c.send(protocol.TypeSelectTerritory, protocol.SelectTerritoryPayload{TerritoryID: id})
						break
					}
				}
			case "Development", "Commerce", "Conquest":
				c.send(protocol.TypeEndPhase, struct{}{})
			}
		}
	}
}

func ownedCount(territories map[string]interface{}, playerID string) int {
	n := 0
	for _, terr := range territories {
		if terr.(map[string]interface{})["owner"] == playerID {
			n++
		}
	}
	return n
}
