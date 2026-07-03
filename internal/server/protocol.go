package server

import "packov/internal/game"

type ClientMessage struct {
	Type      string            `json:"type"`
	Token     string            `json:"token,omitempty"`
	Name      string            `json:"name,omitempty"`
	PlanetID  string            `json:"planet_id,omitempty"`
	Loadout   game.Loadout      `json:"loadout,omitempty"`
	Input     game.InputCommand `json:"input,omitempty"`
	RecipeID  string            `json:"recipe_id,omitempty"`
	ListingID string            `json:"listing_id,omitempty"`
}

type ServerMessage struct {
	Type       string              `json:"type"`
	PlayerID   game.PlayerID       `json:"player_id,omitempty"`
	Account    *game.Account       `json:"account,omitempty"`
	Snapshot   *game.Snapshot      `json:"snapshot,omitempty"`
	Catalog    *game.Catalog       `json:"catalog,omitempty"`
	WorldEvent *game.WorldEvent    `json:"world_event,omitempty"`
	Missions   []game.DailyMission `json:"missions,omitempty"`
	Error      string              `json:"error,omitempty"`
}
