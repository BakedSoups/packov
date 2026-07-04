package protocol

import "packov/internal/game"

type ClientMessage struct {
	Type       string            `json:"type"`
	Token      string            `json:"token,omitempty"`
	Name       string            `json:"name,omitempty"`
	PlanetID   string            `json:"planet_id,omitempty"`
	Loadout    game.Loadout      `json:"loadout,omitempty"`
	Appearance game.Appearance   `json:"appearance,omitempty"`
	Input      game.InputCommand `json:"input,omitempty"`
	RecipeID   string            `json:"recipe_id,omitempty"`
	ListingID  string            `json:"listing_id,omitempty"`
	ItemID     string            `json:"item_id,omitempty"`
	Quantity   int               `json:"quantity,omitempty"`
	UnitPrice  int               `json:"unit_price,omitempty"`
	Channel    string            `json:"channel,omitempty"`
	Body       string            `json:"body,omitempty"`
}

type ServerMessage struct {
	Type       string                    `json:"type"`
	PlayerID   game.PlayerID             `json:"player_id,omitempty"`
	Account    *game.Account             `json:"account,omitempty"`
	Snapshot   *game.Snapshot            `json:"snapshot,omitempty"`
	Catalog    *game.Catalog             `json:"catalog,omitempty"`
	WorldEvent *game.WorldEvent          `json:"world_event,omitempty"`
	Missions   []game.DailyMission       `json:"missions,omitempty"`
	Listings   []game.MarketplaceListing `json:"listings,omitempty"`
	Chat       *game.ChatMessage         `json:"chat,omitempty"`
	ChatLog    []game.ChatMessage        `json:"chat_log,omitempty"`
	Error      string                    `json:"error,omitempty"`
}
