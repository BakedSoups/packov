package game

import "time"

type EntityID uint64
type PlayerID string

type Rarity string

const (
	Common    Rarity = "common"
	Uncommon  Rarity = "uncommon"
	Rare      Rarity = "rare"
	Epic      Rarity = "epic"
	Legendary Rarity = "legendary"
	Relic     Rarity = "relic"
)

type WeaponDef struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Damage     float64 `json:"damage"`
	CooldownMS int     `json:"cooldown_ms"`
	Speed      float64 `json:"speed"`
	Pellets    int     `json:"pellets"`
	Spread     float64 `json:"spread"`
	Range      float64 `json:"range"`
	Energy     int     `json:"energy"`
}

type AbilityDef struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	CooldownMS int     `json:"cooldown_ms"`
	DurationMS int     `json:"duration_ms"`
	Power      float64 `json:"power"`
}

type EnemyDef struct {
	ID     string  `json:"id"`
	Shape  string  `json:"shape"`
	HP     float64 `json:"hp"`
	Speed  float64 `json:"speed"`
	Damage float64 `json:"damage"`
	Sense  float64 `json:"sense"`
	Rarity Rarity  `json:"rarity"`
}

type BossDef struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	HP              float64   `json:"hp"`
	PhaseHP         []float64 `json:"phase_hp"`
	ExclusiveLoot   []string  `json:"exclusive_loot"`
	CosmeticDropPPM int       `json:"cosmetic_drop_ppm"`
}

type LootDef struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Rarity    Rarity   `json:"rarity"`
	BaseValue int      `json:"base_value"`
	Tags      []string `json:"tags,omitempty"`
	Source    string   `json:"source,omitempty"`
	Tradable  bool     `json:"tradable"`
	Bound     bool     `json:"bound,omitempty"`
}

type RecipeDef struct {
	ID              string         `json:"id"`
	Output          string         `json:"output"`
	OutputName      string         `json:"output_name,omitempty"`
	Category        string         `json:"category"`
	UnlockType      string         `json:"unlock_type,omitempty"`
	Blueprint       string         `json:"blueprint,omitempty"`
	BlueprintSource string         `json:"blueprint_source,omitempty"`
	Source          string         `json:"source,omitempty"`
	Role            string         `json:"role,omitempty"`
	TradeRule       string         `json:"trade_rule,omitempty"`
	StatHint        string         `json:"stat_hint,omitempty"`
	Costs           map[string]int `json:"costs"`
	Credits         int            `json:"credits"`
}

type PlanetDef struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	Threat    int      `json:"threat"`
	Resources []string `json:"resources"`
	Boss      string   `json:"boss"`
	Hazards   []string `json:"hazards"`
}

type EventDef struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	DurationMin int      `json:"duration_min"`
	Modifiers   []string `json:"modifiers"`
}

type Catalog struct {
	Weapons   []WeaponDef  `json:"weapons"`
	Abilities []AbilityDef `json:"abilities"`
	Enemies   []EnemyDef   `json:"enemies"`
	Bosses    []BossDef    `json:"bosses"`
	Loot      []LootDef    `json:"loot"`
	Recipes   []RecipeDef  `json:"recipes"`
	Planets   []PlanetDef  `json:"planets"`
	Events    []EventDef   `json:"events"`

	WeaponByID  map[string]WeaponDef  `json:"-"`
	AbilityByID map[string]AbilityDef `json:"-"`
	EnemyByID   map[string]EnemyDef   `json:"-"`
	BossByID    map[string]BossDef    `json:"-"`
	LootByID    map[string]LootDef    `json:"-"`
	RecipeByID  map[string]RecipeDef  `json:"-"`
	PlanetByID  map[string]PlanetDef  `json:"-"`
	EventByID   map[string]EventDef   `json:"-"`
}

type Loadout struct {
	WeaponID  string `json:"weapon_id"`
	AbilityID string `json:"ability_id"`
	HullID    string `json:"hull_id"`
	DroneID   string `json:"drone_id"`
}

type Appearance struct {
	Callsign   string  `json:"callsign"`
	HullID     string  `json:"hull_id"`
	Primary    string  `json:"primary"`
	Secondary  string  `json:"secondary"`
	TrailID    string  `json:"trail_id"`
	NoseID     string  `json:"nose_id"`
	DroneSkin  string  `json:"drone_skin"`
	BadgeID    string  `json:"badge_id"`
	SpinOffset float64 `json:"spin_offset"`
}

type Account struct {
	ID          PlayerID        `json:"id"`
	Name        string          `json:"name"`
	Credits     int             `json:"credits"`
	Level       int             `json:"level"`
	Unlocks     map[string]bool `json:"unlocks"`
	Inventory   Inventory       `json:"inventory"`
	Cosmetics   []string        `json:"cosmetics"`
	Appearance  Appearance      `json:"appearance"`
	CurrentRun  string          `json:"current_run"`
	LastSeenUTC time.Time       `json:"last_seen_utc"`
}

type InputCommand struct {
	PlayerID PlayerID `json:"player_id"`
	Seq      uint64   `json:"seq"`
	Move     Vec2     `json:"move"`
	Aim      Vec2     `json:"aim"`
	Fire     bool     `json:"fire"`
	Ability  bool     `json:"ability"`
	Extract  bool     `json:"extract"`
	Trade    bool     `json:"trade"`
}
