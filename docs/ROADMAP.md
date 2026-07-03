# Development Roadmap

This roadmap tracks the path from the current foundation to a production-grade browser MMO extraction shooter. Work should land in small commits that keep the game runnable after each milestone.

## Current Status

- Shared Go simulation exists for combat, extraction, loot, crafting, planet generation, bosses, enemies, events, and inventory.
- Go authoritative server exists with WebSocket sessions, matchmaking, snapshots, and a SpaceTimeDB persistence boundary.
- Ebitengine/WebAssembly browser client exists with primitive rendering, twin-stick input, HUD, and local fallback simulation.
- SpaceTimeDB schema and reducer contracts exist, but generated/native reducers are not implemented yet.
- Docker Compose, Nginx config, and architecture/database docs exist.
- The client UX will stay Go/Ebitengine-first. HTML remains only the thin WASM host shell.

## Build Order

1. Finish authoritative client-server multiplayer.
2. Implement real SpaceTimeDB persistence.
3. Build the Go/Ebitengine menu shell and station UX.
4. Build character customization and cosmetic editing.
5. Build inventory, crafting, marketplace, and settings.
6. Complete extraction objectives, mining, carried loot capacity, and run results.
7. Replace full snapshots with deltas and interest management.
8. Expand boss mechanics and procedural planet generation.
9. Add MMO social systems.
10. Add live-service operations tooling.
11. Harden for production deployment and load.

## Milestone 1: Authoritative Multiplayer

- [x] Move wire messages into a shared protocol package.
- [x] Add browser WebSocket transport.
- [x] Client sends input commands instead of trusting local state when connected.
- [x] Client applies authoritative server match/snapshot messages.
- [x] Preserve local fallback when no server is available.
- [ ] Add client-side interpolation between snapshots.
- [ ] Add reconnect resume by token and active run ID.
- [ ] Add server-side input sequence validation and rate limits.
- [ ] Add loadout validation against account unlocks.

## Milestone 2: Real SpaceTimeDB Integration

- [ ] Choose the SpaceTimeDB module language for production reducers, likely Rust or TypeScript because Go modules are not supported by CLI 2.6.
- [ ] Convert `spacetime/schema.sql` into native SpaceTimeDB table declarations.
- [ ] Implement reducers from `spacetime/reducers.md`.
- [ ] Generate bindings or bridge calls for the Go server.
- [ ] Replace the in-memory fallback in `SpaceTimeDBAdapter`.
- [ ] Persist accounts, inventory, marketplace, guilds, chat, events, run results, and leaderboard entries.
- [ ] Add migration/versioning process for schema changes.

## Milestone 3: Networking Scale

- [ ] Replace full snapshots with entity create/update/delete deltas.
- [ ] Quantize positions, rotations, velocities, HP, and phase values.
- [ ] Add interest management by viewport, party, extraction zone, objectives, and boss encounters.
- [ ] Split reliable events from unreliable high-rate state.
- [ ] Add bandwidth metrics per session.
- [ ] Add object pooling for server-side transient entities and encoded messages.
- [ ] Add load tests for hundreds of simultaneous players.

## Milestone 4: Station UX

- [x] Add a Go/Ebitengine main menu state before connecting to gameplay.
- [x] Add menu navigation states: title, character select, station, matchmaking entry, settings, and in-run escape flow.
- [x] Add keyboard/controller-style navigation for every menu.
- [x] Add station screen as the first connected screen.
- [ ] Add loadout selection for weapons, abilities, hulls, drones, and modules.
- [ ] Add inventory screen.
- [ ] Add crafting screen with component requirements.
- [ ] Add marketplace buy/sell/cancel screens.
- [ ] Add planet selection and instant matchmaking.
- [ ] Add daily/weekly mission panel.
- [ ] Add player profile and progression screen.
- [ ] Add settings for audio, graphics, controls, accessibility, and controller mapping.
- [x] Keep all interactive UI implemented in Go/Ebitengine, not DOM overlays, so desktop, WASM, and Ebitdock paths share one UI codebase.

## Milestone 5: Character Customization

- [x] Add character/ship editor as a first-class Go/Ebitengine screen.
- [x] Edit color palette, trail style, nose/module geometry, drone skin, and cosmetic badges.
- [x] Preview the character/ship in a live primitive-rendered turntable scene.
- [ ] Persist selected cosmetics to account state through SpaceTimeDB.
- [ ] Separate gameplay loadout from cosmetic appearance so cosmetics never affect power.
- [ ] Add unlock-source labels for cosmetics: boss drop, event reward, guild reward, marketplace purchase, season reward.
- [ ] Add validation so clients can only equip cosmetics owned by the account.
- [ ] Add future hooks for guild emblems and seasonal frames.

## Milestone 6: Economy

- [ ] Implement marketplace listing creation.
- [ ] Implement buy listing.
- [ ] Implement cancel listing.
- [ ] Implement auctions.
- [ ] Implement direct player trades with escrow.
- [ ] Add market price history and volume stats.
- [ ] Add credit sinks: crafting fees, listing fees, guild upgrades, cosmetics, and station services.
- [ ] Add fraud/anomaly audit tables and admin review tools.

## Milestone 7: Progression

- [ ] Add account XP and horizontal unlock tracks.
- [ ] Add weapon unlock rules.
- [ ] Add armor, hull, drone, ability, and module unlocks.
- [ ] Add planet access requirements.
- [ ] Enforce combat stat budgets so unlocks remain sidegrades.
- [ ] Add blueprint/component crafting paths.
- [ ] Add cosmetic unlocks that never affect gameplay.
- [ ] Add seasonal progression with reset-safe account identity.

## Milestone 8: Extraction Depth

- [ ] Add objective completion logic.
- [ ] Add mining/resource interaction.
- [ ] Add carried loot capacity and extraction-risk decisions.
- [ ] Add extraction defense scaling by party size, threat, and carried loot value.
- [ ] Add run success/failure result screen.
- [ ] Transfer carried loot to station inventory only on successful extraction.
- [ ] Add PvP extraction opt-in zones and anti-grief rules.
- [ ] Add enemy director with heat, noise, and extraction-response budgets.

## Milestone 9: Boss Depth

- [ ] Hive Queen: nests, swarm waves, acid zones, egg armor.
- [ ] Ancient Mech: rotating laser arms, shield generators, missile salvos.
- [ ] Void Leviathan: gravity wells, teleport dives, orbiting void mines.
- [ ] Crystal Titan: reflective shields, shard prisons, fracture phases.
- [ ] Space Worm: burrow attacks, segmented weak points, arena collapse.
- [ ] Add exclusive component drops for every boss.
- [ ] Add ultra-rare cosmetic drops.
- [ ] Add rotating world boss schedule.

## Milestone 10: Procedural Planets

- [ ] Replace simple seeded placement with reusable map pieces.
- [ ] Add biome-specific zones for forest, desert, ice, volcanic, moon, alien hive, abandoned facility, and derelict ship.
- [ ] Add objective chains that create route choices.
- [ ] Add hazards with readable telegraphs.
- [ ] Persist deterministic seeds per run.
- [ ] Apply world event modifiers to generation.
- [ ] Add extraction-site variation and boss arenas.

## Milestone 11: MMO Social

- [ ] Global chat.
- [ ] Party chat.
- [ ] Guild chat.
- [ ] Friends.
- [ ] Party invites and join-in-progress rules.
- [ ] Guild creation, roles, and membership.
- [ ] Guild stations and shared upgrades.
- [ ] Player profiles.
- [ ] Leaderboards.
- [ ] Trading.

## Milestone 12: Live Service

- [ ] Daily quests.
- [ ] Weekly quests.
- [ ] Rotating bosses.
- [ ] Global world events.
- [ ] Community progression.
- [ ] Galaxy-wide unlocks.
- [ ] Seasonal content model.
- [ ] Holiday cosmetics.
- [ ] Admin announcements.
- [ ] Content catalog versioning and hot reload.

## Milestone 13: Ebitdock and Development Workflow

- [ ] Keep the primary client as Go/Ebitengine compiled to WASM.
- [ ] Use Ebitdock as the fast browser feedback loop for the Go client when available locally.
- [ ] Add `make ebitdock` or equivalent script once the local Ebitdock command/API is confirmed.
- [ ] Document how Ebitdock serves the WASM build, reloads client changes, and captures screenshots.
- [ ] Add Ebitdock smoke checks for menu navigation, character editor, station, matchmaking, and in-run rendering.
- [ ] Ensure Ebitdock testing uses the same Go client codepath as production browser builds.
- [ ] Keep generated artifacts ignored; source remains Go plus minimal HTML host shell.

## Milestone 14: Production Hardening

- [ ] Docker health checks.
- [ ] CI for Go tests, WASM build, Docker build, and config validation.
- [ ] Browser smoke tests.
- [ ] Playwright screenshots and canvas nonblank checks.
- [ ] Structured telemetry for retention, economy, matchmaking, and encounter difficulty.
- [ ] Graceful server drain and reconnect during deploys.
- [ ] Blue/green deployment.
- [ ] Backups and restore drills for SpaceTimeDB state.

## Future Expansion Ideas

- Guild stations with upgrade rooms and shared extraction bonuses.
- Derelict fleet raids with multi-party objectives.
- Seasonal planets that change as community goals advance.
- Player-made contracts funded by credits.
- Cosmetic-only battle pass with lore logs, trails, hull skins, and emotes.
