# Database Schema

SpaceTimeDB tables are specified in `spacetime/schema.sql`. The schema is split by persistence concern:

- Accounts: `player_account`, `inventory_stack`, including cosmetic ownership and selected appearance JSON.
- Runs: `run`, `run_player`, `entity_snapshot`.
- Economy: `marketplace_listing`, `market_trade`.
- Social/MMO: `guild`, `guild_member`, `chat_message`.
- Live service: `world_event`, `leaderboard_entry`.

## Persistence Rules

- Account unlocks and station inventory persist immediately after crafting, marketplace trades, extraction, quest rewards, and cosmetic grants.
- Appearance changes persist independently from gameplay loadouts and never affect combat calculations.
- Carried loot only moves to station inventory during successful extraction.
- Entity snapshots are sampled for audit, reconnect, analytics, and later replay tooling.
- Marketplace trades are append-only for price history and fraud review.
- World event progress is additive and season-scoped.

## SpaceTimeDB Integration

The Go code uses `internal/server.Persistence`. `SpaceTimeDBAdapter` currently falls back to memory for local development while preserving method boundaries that map one-to-one to the reducer contract in `spacetime/reducers.md`.
