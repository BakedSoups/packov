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

## Suggested Next Commit Order

1. Fix enemy contact states and remove jitter at melee range.
2. Add prominent player health/shield HUD and hit feedback.
3. Stabilize crafting input and recipe detail UI.
4. Add primitive asset API and style tokens.
5. Prototype Go-only 3D-looking station backdrop.
6. Decide whether to integrate Three.js for menu-only decorative background.
7. Add objective interaction system.
8. Add run result stat breakdown.
9. Move persistence from memory fallback to real SpaceTimeDB reducers.
