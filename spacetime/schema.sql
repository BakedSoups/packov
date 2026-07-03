-- Packov SpaceTimeDB persistence model.
-- Go is not a SpaceTimeDB module language in CLI 2.6, so this schema is the
-- contract consumed by the Go adapter and mirrored by future generated module bindings.

CREATE TABLE player_account (
  player_id TEXT PRIMARY KEY,
  display_name TEXT NOT NULL,
  credits BIGINT NOT NULL,
  level INT NOT NULL,
  unlocks JSON NOT NULL,
  cosmetics JSON NOT NULL,
  appearance JSON NOT NULL,
  current_run TEXT,
  last_seen_utc TIMESTAMP NOT NULL
);

CREATE TABLE inventory_stack (
  player_id TEXT NOT NULL,
  item_id TEXT NOT NULL,
  quantity BIGINT NOT NULL,
  PRIMARY KEY (player_id, item_id)
);

CREATE TABLE run (
  run_id TEXT PRIMARY KEY,
  planet_id TEXT NOT NULL,
  seed BIGINT NOT NULL,
  phase TEXT NOT NULL,
  tick BIGINT NOT NULL,
  started_utc TIMESTAMP NOT NULL,
  ended_utc TIMESTAMP
);

CREATE TABLE run_player (
  run_id TEXT NOT NULL,
  player_id TEXT NOT NULL,
  loadout JSON NOT NULL,
  extracted BOOLEAN NOT NULL,
  downed BOOLEAN NOT NULL,
  carried_loot JSON NOT NULL,
  PRIMARY KEY (run_id, player_id)
);

CREATE TABLE entity_snapshot (
  run_id TEXT NOT NULL,
  tick BIGINT NOT NULL,
  entity_id BIGINT NOT NULL,
  kind TEXT NOT NULL,
  owner_id TEXT,
  def_id TEXT,
  x DOUBLE PRECISION NOT NULL,
  y DOUBLE PRECISION NOT NULL,
  vx DOUBLE PRECISION NOT NULL,
  vy DOUBLE PRECISION NOT NULL,
  hp DOUBLE PRECISION NOT NULL,
  phase INT NOT NULL,
  PRIMARY KEY (run_id, tick, entity_id)
);

CREATE TABLE marketplace_listing (
  listing_id TEXT PRIMARY KEY,
  seller_id TEXT NOT NULL,
  item_id TEXT NOT NULL,
  quantity BIGINT NOT NULL,
  unit_price BIGINT NOT NULL,
  created_utc TIMESTAMP NOT NULL,
  expires_utc TIMESTAMP NOT NULL
);

CREATE TABLE market_trade (
  trade_id TEXT PRIMARY KEY,
  listing_id TEXT NOT NULL,
  buyer_id TEXT NOT NULL,
  seller_id TEXT NOT NULL,
  item_id TEXT NOT NULL,
  quantity BIGINT NOT NULL,
  unit_price BIGINT NOT NULL,
  traded_utc TIMESTAMP NOT NULL
);

CREATE TABLE guild (
  guild_id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  founder_id TEXT NOT NULL,
  station_level INT NOT NULL,
  bank JSON NOT NULL,
  created_utc TIMESTAMP NOT NULL
);

CREATE TABLE guild_member (
  guild_id TEXT NOT NULL,
  player_id TEXT NOT NULL,
  role TEXT NOT NULL,
  joined_utc TIMESTAMP NOT NULL,
  PRIMARY KEY (guild_id, player_id)
);

CREATE TABLE chat_message (
  message_id TEXT PRIMARY KEY,
  channel TEXT NOT NULL,
  sender_id TEXT NOT NULL,
  guild_id TEXT,
  body TEXT NOT NULL,
  sent_utc TIMESTAMP NOT NULL
);

CREATE TABLE world_event (
  event_instance_id TEXT PRIMARY KEY,
  event_id TEXT NOT NULL,
  planet_id TEXT NOT NULL,
  starts_utc TIMESTAMP NOT NULL,
  ends_utc TIMESTAMP NOT NULL,
  progress DOUBLE PRECISION NOT NULL
);

CREATE TABLE leaderboard_entry (
  board_id TEXT NOT NULL,
  player_id TEXT NOT NULL,
  score BIGINT NOT NULL,
  season_id TEXT NOT NULL,
  updated_utc TIMESTAMP NOT NULL,
  PRIMARY KEY (board_id, player_id, season_id)
);
