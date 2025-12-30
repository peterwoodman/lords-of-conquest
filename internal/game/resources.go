package game

// ResourceType represents a type of resource.
type ResourceType int

const (
	ResourceNone ResourceType = iota
	ResourceCoal
	ResourceGold
	ResourceIron
	ResourceTimber
	ResourceHorses
)

// String returns the resource name.
func (r ResourceType) String() string {
	switch r {
	case ResourceCoal:
		return "Coal"
	case ResourceGold:
		return "Gold"
	case ResourceIron:
		return "Iron"
	case ResourceTimber:
		return "Timber"
	case ResourceHorses:
		return "Horses"
	default:
		return "None"
	}
}

// IsStockpilable returns true if this resource goes to the stockpile.
// Horses are special and spread on the map instead.
func (r ResourceType) IsStockpilable() bool {
	return r != ResourceNone && r != ResourceHorses
}

// Stockpile represents a player's collected resources.
type Stockpile struct {
	Coal   int
	Gold   int
	Iron   int
	Timber int
}

// NewStockpile creates an empty stockpile.
func NewStockpile() *Stockpile {
	return &Stockpile{}
}

// Add adds resources to the stockpile.
func (s *Stockpile) Add(resource ResourceType, amount int) {
	switch resource {
	case ResourceCoal:
		s.Coal += amount
	case ResourceGold:
		s.Gold += amount
	case ResourceIron:
		s.Iron += amount
	case ResourceTimber:
		s.Timber += amount
	}
}

// Remove removes resources from the stockpile. Returns false if insufficient.
func (s *Stockpile) Remove(resource ResourceType, amount int) bool {
	switch resource {
	case ResourceCoal:
		if s.Coal < amount {
			return false
		}
		s.Coal -= amount
	case ResourceGold:
		if s.Gold < amount {
			return false
		}
		s.Gold -= amount
	case ResourceIron:
		if s.Iron < amount {
			return false
		}
		s.Iron -= amount
	case ResourceTimber:
		if s.Timber < amount {
			return false
		}
		s.Timber -= amount
	default:
		return false
	}
	return true
}

// Get returns the amount of a resource.
func (s *Stockpile) Get(resource ResourceType) int {
	switch resource {
	case ResourceCoal:
		return s.Coal
	case ResourceGold:
		return s.Gold
	case ResourceIron:
		return s.Iron
	case ResourceTimber:
		return s.Timber
	default:
		return 0
	}
}

// Total returns the total number of resources.
func (s *Stockpile) Total() int {
	return s.Coal + s.Gold + s.Iron + s.Timber
}

// Clone creates a copy of the stockpile.
func (s *Stockpile) Clone() *Stockpile {
	return &Stockpile{
		Coal:   s.Coal,
		Gold:   s.Gold,
		Iron:   s.Iron,
		Timber: s.Timber,
	}
}

// BuildCost represents the cost to build something.
type BuildCost struct {
	Coal   int
	Gold   int
	Iron   int
	Timber int
}

// CostCity is the cost to build a city.
var CostCity = BuildCost{Coal: 1, Gold: 1, Iron: 1, Timber: 1}

// CostCityGold is the gold-only cost to build a city.
var CostCityGold = BuildCost{Gold: 4}

// CostWeapon is the cost to build a weapon.
var CostWeapon = BuildCost{Coal: 1, Iron: 1}

// CostWeaponGold is the gold-only cost to build a weapon.
var CostWeaponGold = BuildCost{Gold: 2}

// CostBoat is the cost to build a boat.
var CostBoat = BuildCost{Timber: 3}

// CostBoatGold is the gold-only cost to build a boat.
var CostBoatGold = BuildCost{Gold: 3}

// CanAfford checks if a stockpile can afford a cost.
func (s *Stockpile) CanAfford(cost BuildCost) bool {
	return s.Coal >= cost.Coal &&
		s.Gold >= cost.Gold &&
		s.Iron >= cost.Iron &&
		s.Timber >= cost.Timber
}

// Spend removes resources for a build cost. Returns false if insufficient.
func (s *Stockpile) Spend(cost BuildCost) bool {
	if !s.CanAfford(cost) {
		return false
	}
	s.Coal -= cost.Coal
	s.Gold -= cost.Gold
	s.Iron -= cost.Iron
	s.Timber -= cost.Timber
	return true
}

// CanAffordStockpile checks if this stockpile can afford another stockpile's worth.
func (s *Stockpile) CanAffordStockpile(cost *Stockpile) bool {
	return s.Coal >= cost.Coal &&
		s.Gold >= cost.Gold &&
		s.Iron >= cost.Iron &&
		s.Timber >= cost.Timber
}

// Subtract removes resources based on another stockpile.
func (s *Stockpile) Subtract(cost *Stockpile) {
	s.Coal -= cost.Coal
	s.Gold -= cost.Gold
	s.Iron -= cost.Iron
	s.Timber -= cost.Timber
}

