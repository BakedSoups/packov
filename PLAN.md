# Packov Near-Term Plan

This file captures the current hands-on gameplay notes and the next improvements needed before the game feels coherent. `docs/ROADMAP.md` remains the long-term roadmap; this file is the short-term execution plan.

## Immediate Observations

- Base world enemies reach the player, stop, stare, and jitter around contact range.
- Enemy contact behavior needs to feel intentional: circling, backing off, lunging, and re-approaching instead of vibrating at the collision boundary.
- The rotating geometric objects in the menu background are a good direction, but the station/menu should feel more alive.
- A 3D base/menu backdrop with living orbs spinning around the station would improve first impression and make the station feel like a place.
- Crafting UI feels jittery, likely due to key-repeat behavior, selection feedback, and screen state updates.
- Health exists but needs to be more readable and more central to moment-to-moment play.
- The current build has a solid systems foundation, but it still needs stronger game feel, clearer feedback, and more complete run objectives.

## Priority 1: Enemy Contact and Organic Movement

- Add enemy behavior states instead of one continuous chase vector:
  - `approach`
  - `orbit`
  - `lunge`
  - `recover`
  - `retreat`
- Give each enemy a preferred engagement range and hysteresis so it does not rapidly flip between moving forward and backward.
- Add short attack windups and cooldowns so enemies visibly commit to hits.
- Add orbit slots around the player so several enemies spread around instead of all pushing into the same point.
- Add local avoidance that accounts for player radius, enemy radius, and desired attack lane.
- Add per-enemy steering personality:
  - triangle skitters strafe and lunge
  - squares hold space and shove
  - hexagons kite and shoot
  - octagons guard and rotate slowly
- Add tests for contact stability: enemies should not oscillate heavily when within attack range.

## Priority 2: Health and Combat Readability

- Add a prominent player health/shield bar near the bottom or top of the screen.
- Keep small health bars over enemies, but make boss health a large fixed UI element.
- Add damage flash on the player hull when hit.
- Add enemy hit flash when bullets connect.
- Add low-health warning styling that is readable but not noisy.
- Add clear downed/failure feedback before the result screen.
- Add floating damage numbers only if they do not clutter the Diep.io readability.
- Add a carried-loot value indicator so extraction risk is obvious.

## Priority 3: Diep.io-Style Asset Pass

- Turn current primitive helpers into a small asset API:
  - `DrawPlayerShip`
  - `DrawEnemyShape`
  - `DrawBossModule`
  - `DrawProjectile`
  - `DrawLootNode`
  - `DrawAbilityEffect`
- Make thick outlines a style token, not a hardcoded value.
- Add consistent color tokens for player, enemies, boss, bullets, loot rarity, objectives, extraction, hazards, and UI.
- Add weapon-specific projectiles:
  - machine gun: small outlined yellow rounds
  - shotgun: many small pellets
  - railgun: long dark-core slug
  - laser: solid beam with outline
  - flamethrower: expanding orange circles
  - rocket: triangle missile with trail
  - plasma: large glowing orb with outline
- Add simple procedural biome props that match the thick-outline style.

## Priority 4: Menu and Station Backdrop

- Keep all gameplay and UI in Go/Ebitengine unless there is a strong reason not to.
- Evaluate a Three.js-only decorative station backdrop for the menu:
  - living orbs spinning around a base
  - rotating ring structures
  - docking arms
  - orbiting drones
  - subtle parallax camera motion
- If Three.js is used, isolate it to the HTML host/menu backdrop and keep gameplay authoritative state and UI controls in Go.
- Alternative Go-only path: fake the 3D station with layered Ebitengine primitives, orbit math, scale changes, and shadow rings.
- Decide whether adding Three.js is worth the extra integration cost versus keeping the stack pure Go/Ebitengine.
- Add a station preview mode that shows:
  - player ship docked
  - orbiting cosmetic drones
  - menu background orbs
  - current world event beacon

## Priority 5: Crafting UI Stability

- Debounce menu input so holding a key does not cause rapid unintended state changes.
- Add a selected recipe detail panel instead of only text rows.
- Show missing components in red and available components in green.
- Show whether a recipe unlocks a weapon, ability, module, cosmetic, or planet access.
- Add confirmation for crafting expensive items.
- After crafting, refresh account state and keep selection stable.
- Add a success/failure message that does not overwrite network status permanently.

## Priority 6: Run Objectives and Extraction Loop

- Objectives need real interactions:
  - hold to activate uplink
  - mine resources
  - collect samples
  - defend beacon
  - destroy nest/reactor
- Add progress circles or bars for objective interactions.
- Add objective completion requirements before extraction is optimal.
- Add optional side objectives for better loot.
- Add extraction call UI and countdown that is more visible.
- Add enemy wave telegraphs during extraction.
- Add run result stats:
  - kills
  - mined resources
  - boss damage
  - objectives completed
  - loot extracted
  - loot lost
  - credits earned

## Priority 7: Persistence and Economy Gaps

- Replace memory fallback with real SpaceTimeDB reducers/bindings.
- Persist inventory, appearance, market listings, trades, run outcomes, daily missions, and world events.
- Add market history and price averages.
- Add listing fees and crafting fees as credit sinks.
- Add direct trade escrow.
- Add auctions after basic buy/sell feels solid.
- Add audit events for all economy mutations.

## Priority 7A: Crafting and Progression Tree

- Yes: the game needs an explicit tree that turns enemy drops and boss components into usable unlocks.
- The tree should be horizontal, not raw power creep. Crafted gear should unlock roles, playstyles, counters, utility, cosmetics, and access rather than strictly higher damage.
- Item flow should be:
  - kill enemies
  - mine resources
  - defeat bosses
  - extract successfully
  - bank components
  - discover or buy blueprints
  - craft unlocks
  - equip sidegrades
  - repeat into harder planets/events
- Normal enemies should mostly drop common/uncommon materials:
  - alien alloy
  - bio gel
  - chitin plates
  - reactor scrap
  - unstable spores
  - cryo shards
- Elite enemies should drop role-specific components:
  - targeting lenses
  - armored cores
  - venom sacs
  - phase coils
  - shield emitters
  - drone servos
- Bosses should drop exclusive components, not finished gear:
  - Hive Queen: hive egg, queen carapace, pheromone gland
  - Ancient Mech: ancient AI chip, servo heart, rail actuator
  - Void Leviathan: void crystal, gravity sac, phase membrane
  - Crystal Titan: crystal heart, prism shard, refractor plate
  - Space Worm: burrow core, worm scale, seismic tooth
- Blueprints should control major unlocks:
  - weapon blueprints
  - armor blueprints
  - hull blueprints
  - drone blueprints
  - module blueprints
  - ability blueprints
  - cosmetic blueprints
  - planet access coordinates
- Crafting branches needed:
  - Weapons: machine gun variants, shotgun, railgun, laser, flamethrower, rocket launcher, plasma cannon
  - Armor: light shield, hazard suit, reactive plating, phase mesh, medic rig
  - Hulls: scout, bruiser, interceptor, carrier, engineer, extractor
  - Drones: combat drone, mining drone, repair drone, shield drone, decoy drone
  - Modules: ammo converter, dash capacitor, extraction beacon booster, loot scanner, boss detector
  - Abilities: dash, shield, drone, turret, EMP, heal pulse, gravity anchor, phase blink
  - Cosmetics: trails, badges, outline styles, hull skins, boss trophies, guild emblems
  - Planet access: coordinates, event keys, boss lures, derelict signal maps
- Example recipes:
  - Railgun = railgun blueprint + quantum cores + rail actuator + alien alloy
  - Shield Drone = drone blueprint + shield emitters + drone servos + plasma batteries
  - Hazard Armor = armor blueprint + chitin plates + cryo shards + reactor scrap
  - Void Dash Module = module blueprint + void crystal + phase membrane + dash capacitor
  - Hive Trail Cosmetic = cosmetic blueprint + hive egg + pheromone gland
- UI needed for the tree:
  - visual recipe graph
  - locked/unlocked nodes
  - missing components
  - where-to-find hints
  - blueprint source
  - boss source
  - marketplace shortcut for tradable components
  - craft preview
  - sidegrade stat comparison
- Backend/data needed:
  - recipe graph data file
  - component tags
  - blueprint unlock table
  - account crafted unlocks
  - component provenance/audit
  - tradable versus bound item flags
  - seasonal recipe flags
- Economy rules:
  - Common components should be tradable.
  - Most boss components should be tradable unless tied to prestige unlocks.
  - Crafted gameplay unlocks should usually be account-bound.
  - Cosmetics can be tradable depending on event rarity.
  - Premium cosmetics must never be ingredients for gameplay gear.

## Priority 8: Missing Core Game Systems

- Planet selection screen with threat, biome, resources, boss, event modifiers, and party readiness.
- Daily/weekly mission persistence.
- Player profile screen with stats and unlocks.
- Friends and party invites.
- Guild creation and guild chat.
- Global chat moderation/rate limiting.
- Reconnect timeout and disconnect failure rules.
- Controller mapping screen.
- Audio: weapon shots, hits, extraction alarm, UI selection, loot pickup, boss phases.
- Accessibility: colorblind-safe rarity colors, screen shake toggle, readable outlines, scalable HUD.
- Browser smoke tests and screenshot checks for menu, station, combat, death, extraction, and marketplace.

## Priority 9: Missing Moment-to-Moment Gameplay

- Real weapon identities are missing; most weapons still behave like generic projectiles in practice.
- Abilities need readable cooldown UI, activation effects, and tactical roles.
- Enemy attacks need telegraphs, windups, cooldowns, and clear hit moments.
- Boss encounters need arenas, mechanics, weak points, phase telegraphs, and exclusive reward presentation.
- Resource mining needs an interaction, progress timing, interruption rules, and yield feedback.
- Loot needs pickup text, rarity presentation, stack value, and clear difference between carried and banked inventory.
- Hazards need readable warnings and real gameplay consequences.
- Extraction should have a visible beacon, countdown, wave intensity, and risk/reward escalation.
- Player movement needs acceleration/friction tuning, dash feel, collision response, and controller tuning.
- There is no revive/downed co-op loop yet; co-op failure is too binary.

## Priority 10: Missing Station and UX Flow

- Title/login flow needs player naming, account token visibility, and reconnect feedback.
- Station needs a clearer information hierarchy: deploy, loadout, inventory, crafting, market, profile, missions.
- Loadout screen needs stat comparisons and sidegrade tradeoff explanations.
- Crafting needs a confirmation state, missing-component routing, and blueprint discovery.
- Marketplace needs listing detail, sorting, filtering, price history, and sell quantity/price controls.
- Character editor needs callsign editing, unlock source labels, ownership locks, and cosmetic preview categories.
- Planet selection needs event banners, party readiness, boss rotation, resource preview, and threat explanation.
- Settings need persistence, rebinding, audio controls, accessibility controls, and controller testing.
- Result screen needs stat breakdown, rewards, lost loot, unlock progress, and next-action buttons.

## Priority 11: Missing Backend and Persistence

- SpaceTimeDB integration is still a contract/fallback, not real production persistence.
- Accounts need secure identity, session tokens, reconnect expiry, and abuse protection.
- Run settlement needs durable audit events, idempotency keys, and replay-safe reward application.
- Marketplace data needs persistence, listing expiration, trade history, and rollback/audit tooling.
- World events need server scheduling, persistence, notifications, and community progress reducers.
- Guilds, friends, chat, profiles, and leaderboards need real tables/reducers and moderation controls.
- Content data needs versioning so client/server catalogs cannot mismatch silently.
- Admin/live-ops tools are missing: announcements, event creation, grants, bans, economy inspection.
- Observability is missing: metrics, logs, traces, tick time, bandwidth, errors, economy movement.

## Priority 12: Missing Multiplayer Scale Work

- Full snapshots are still the main network path; this will not scale to large player counts.
- Need entity deltas: create/update/delete, quantized positions, and compressed state.
- Need interest management by camera, party, objective, extraction zone, and boss arena.
- Need reliable versus unreliable channels or equivalent message separation.
- Need server-side lag handling, input buffering, client interpolation tuning, and disconnect grace.
- Need object pools for bullets, particles, enemies, network messages, and temporary slices.
- Need load tests with simulated clients and profiling of tick CPU, allocations, and bandwidth.
- Need anti-cheat validation beyond basic loadout/input checks.

## Priority 13: Missing Content Pipeline

- Content files need schema validation and tests.
- Need separate data files for weapons, abilities, enemies, bosses, loot, recipes, planets, events, cosmetics, and missions.
- Need designer-friendly balancing fields: budget, role, counters, rarity, unlock source, event tags.
- Need hot reload or catalog version rollout for live service.
- Need content migration/versioning for accounts that own old items.
- Need debug commands to spawn enemies, grant loot, force events, and jump to boss fights.
- Need asset preview screen for all primitive art definitions.

## Priority 14: Missing Audio and Feel

- No audio layer yet.
- Need weapon sounds, enemy hit sounds, player damage sounds, loot pickup, crafting, market, menu selection, extraction alarm, boss phase stingers.
- Need screen shake with setting toggle and intensity caps.
- Need impact particles, muzzle flashes, shield impacts, explosion rings, and death bursts.
- Need music plan: station ambience, planet loop, extraction intensity, boss track, result stingers.
- Need haptic/controller rumble hooks where supported.

## Priority 15: Missing Quality and Release Infrastructure

- Need CI that runs Go tests, WASM build, Docker build, docker compose config, content validation, and linting.
- Need Playwright/Ebitdock smoke tests for title, station, character editor, deploy, combat, death, extraction, crafting, and market.
- Need screenshot/canvas nonblank checks after every visual change.
- Need browser compatibility checks for Chrome, Firefox, Safari where possible.
- Need production Docker health checks and graceful shutdown/drain.
- Need backup/restore drills once SpaceTimeDB persistence is real.
- Need staging environment and deployment documentation.
- Need error reporting from browser clients.
- Need performance budgets for FPS, memory, server tick time, and network bandwidth.

## Suggested Next Commit Order

1. Fix enemy contact states and remove jitter at melee range.
2. Add prominent player health/shield HUD and hit feedback.
3. Stabilize crafting input and recipe detail UI.
4. Add crafting/progression tree data structures and content.
5. Add crafting tree UI with missing-component hints.
6. Add primitive asset API and style tokens.
7. Prototype Go-only 3D-looking station backdrop.
8. Add weapon-specific projectile primitives and hit effects.
9. Add objective interaction system.
10. Add extraction countdown/wave telegraphs.
11. Add run result stat breakdown.
12. Add planet selection and mission panels.
13. Decide whether to integrate Three.js for menu-only decorative background.
14. Move persistence from memory fallback to real SpaceTimeDB reducers.
15. Replace full snapshots with deltas and interest management.
16. Add CI and browser smoke tests.
