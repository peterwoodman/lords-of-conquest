package maps

import "lords-of-conquest/internal/game"

// ToGameData converts a map to game initialization data.
func ToGameData(m *Map) game.MapData {
	territories := make(map[string]game.TerritoryData)
	for id, t := range m.Territories {
		adjacent := make([]string, len(t.AdjacentTerritories))
		for i, adj := range t.AdjacentTerritories {
			adjacent[i] = TerritoryIDToString(adj)
		}

		waters := make([]string, len(t.AdjacentWaters))
		for i, w := range t.AdjacentWaters {
			waters[i] = WaterIDToString(w)
		}

		territories[TerritoryIDToString(id)] = game.TerritoryData{
			Name:         t.Name,
			Resource:     t.Resource,
			Adjacent:     adjacent,
			CoastalTiles: t.CoastalCells,
			WaterBodies:  waters,
		}
	}

	waterBodies := make(map[string]game.WaterBodyData)
	for id, wb := range m.WaterBodies {
		coastal := make([]string, len(wb.CoastalTerritories))
		for i, t := range wb.CoastalTerritories {
			coastal[i] = TerritoryIDToString(t)
		}

		waterBodies[WaterIDToString(id)] = game.WaterBodyData{
			Territories: coastal,
		}
	}

	return game.MapData{
		ID:          m.ID,
		Name:        m.Name,
		Territories: territories,
		WaterBodies: waterBodies,
	}
}
