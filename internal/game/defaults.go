package game

func DefaultCatalogForClient() *Catalog {
	c := &Catalog{
		Weapons: []WeaponDef{
			{ID: "machine_gun", Name: "Machine Gun", Damage: 7, CooldownMS: 80, Speed: 720, Pellets: 1, Spread: 0.02, Range: 760},
			{ID: "shotgun", Name: "Shotgun", Damage: 9, CooldownMS: 520, Speed: 590, Pellets: 7, Spread: 0.22, Range: 420},
			{ID: "railgun", Name: "Railgun", Damage: 48, CooldownMS: 980, Speed: 1180, Pellets: 1, Spread: 0.005, Range: 1150, Energy: 18},
			{ID: "plasma_cannon", Name: "Plasma Cannon", Damage: 34, CooldownMS: 430, Speed: 520, Pellets: 1, Spread: 0.04, Range: 710, Energy: 9},
		},
		Abilities: []AbilityDef{
			{ID: "dash", Name: "Dash", CooldownMS: 2400, DurationMS: 160, Power: 390},
			{ID: "shield", Name: "Shield", CooldownMS: 11000, DurationMS: 4200, Power: 70},
			{ID: "emp", Name: "EMP", CooldownMS: 15000, DurationMS: 1200, Power: 210},
			{ID: "heal_pulse", Name: "Heal Pulse", CooldownMS: 13000, Power: 34},
		},
		Enemies: []EnemyDef{
			{ID: "skitter", Shape: "triangle", HP: 24, Speed: 145, Damage: 12, Sense: 560, Rarity: Common},
			{ID: "bulwark", Shape: "square", HP: 88, Speed: 70, Damage: 22, Sense: 420, Rarity: Uncommon},
			{ID: "hex_spitter", Shape: "hexagon", HP: 42, Speed: 95, Damage: 16, Sense: 690, Rarity: Rare},
		},
		Bosses: []BossDef{
			{ID: "hive_queen", Name: "Hive Queen", HP: 1600, PhaseHP: []float64{0.7, 0.35}, ExclusiveLoot: []string{"hive_egg", "boss_trophy"}, CosmeticDropPPM: 90},
		},
		Loot: []LootDef{
			{ID: "alien_alloy", Name: "Alien Alloy", Rarity: Common, BaseValue: 18, Tags: []string{"component", "common"}, Source: "skitters and salvage nodes", Tradable: true},
			{ID: "plasma_battery", Name: "Plasma Battery", Rarity: Uncommon, BaseValue: 75, Tags: []string{"component", "energy"}, Source: "reactor caches and elite patrols", Tradable: true},
			{ID: "quantum_core", Name: "Quantum Core", Rarity: Rare, BaseValue: 210, Tags: []string{"component", "elite"}, Source: "elite enemies and derelicts", Tradable: true},
			{ID: "hive_egg", Name: "Hive Egg", Rarity: Epic, BaseValue: 420, Tags: []string{"boss_component", "hive"}, Source: "Hive Queen", Tradable: true},
			{ID: "rail_actuator", Name: "Rail Actuator", Rarity: Epic, BaseValue: 520, Tags: []string{"boss_component", "weapon"}, Source: "Ancient Mech", Tradable: true},
			{ID: "shield_emitter", Name: "Shield Emitter", Rarity: Rare, BaseValue: 190, Tags: []string{"component", "drone"}, Source: "elite guardians", Tradable: true},
			{ID: "drone_servo", Name: "Drone Servo", Rarity: Rare, BaseValue: 180, Tags: []string{"component", "drone"}, Source: "elite sentries", Tradable: true},
			{ID: "cryo_shard", Name: "Cryo Shard", Rarity: Uncommon, BaseValue: 65, Tags: []string{"component", "hazard"}, Source: "ice planets", Tradable: true},
			{ID: "chitin_plate", Name: "Chitin Plate", Rarity: Uncommon, BaseValue: 58, Tags: []string{"component", "armor"}, Source: "hive enemies", Tradable: true},
			{ID: "void_crystal", Name: "Void Crystal", Rarity: Legendary, BaseValue: 900, Tags: []string{"boss_component", "void"}, Source: "Void Leviathan", Tradable: true},
			{ID: "phase_membrane", Name: "Phase Membrane", Rarity: Epic, BaseValue: 540, Tags: []string{"boss_component", "void"}, Source: "Void Leviathan", Tradable: true},
			{ID: "railgun_blueprint", Name: "Railgun Blueprint", Rarity: Rare, BaseValue: 350, Tags: []string{"blueprint", "weapon"}, Source: "Ancient Mech event caches", Tradable: true},
			{ID: "drone_blueprint", Name: "Drone Blueprint", Rarity: Rare, BaseValue: 330, Tags: []string{"blueprint", "drone"}, Source: "derelict fleet events", Tradable: true},
			{ID: "armor_blueprint", Name: "Armor Blueprint", Rarity: Rare, BaseValue: 310, Tags: []string{"blueprint", "armor"}, Source: "hazard objectives", Tradable: true},
			{ID: "hull_blueprint", Name: "Hull Blueprint", Rarity: Rare, BaseValue: 330, Tags: []string{"blueprint", "hull"}, Source: "station contracts", Tradable: true},
			{ID: "module_blueprint", Name: "Module Blueprint", Rarity: Rare, BaseValue: 340, Tags: []string{"blueprint", "module"}, Source: "boss side objectives", Tradable: true},
			{ID: "ability_blueprint", Name: "Ability Blueprint", Rarity: Rare, BaseValue: 360, Tags: []string{"blueprint", "ability"}, Source: "world event objectives", Tradable: true},
			{ID: "cosmetic_blueprint", Name: "Cosmetic Blueprint", Rarity: Epic, BaseValue: 500, Tags: []string{"blueprint", "cosmetic"}, Source: "rare boss drops", Tradable: true},
			{ID: "coordinate_map", Name: "Coordinate Map", Rarity: Epic, BaseValue: 650, Tags: []string{"blueprint", "planet_access"}, Source: "derelict command rooms", Tradable: true},
		},
		Recipes: []RecipeDef{
			{ID: "craft_railgun", Output: "railgun", OutputName: "Railgun", Category: "weapon", UnlockType: "account_bound", Blueprint: "railgun_blueprint", BlueprintSource: "Ancient Mech event caches", Source: "Ancient Mech", Role: "precision burst", TradeRule: "crafted unlock is account-bound", StatHint: "Long-range burst sidegrade with high aim demand.", Costs: map[string]int{"alien_alloy": 8, "plasma_battery": 3, "quantum_core": 2, "rail_actuator": 1, "railgun_blueprint": 1}, Credits: 1200},
			{ID: "craft_hazard_armor", Output: "hazard_armor", OutputName: "Hazard Armor", Category: "armor", UnlockType: "account_bound", Blueprint: "armor_blueprint", BlueprintSource: "hazard objectives", Source: "Ice and hive planets", Role: "hazard counter", TradeRule: "crafted unlock is account-bound", StatHint: "Reduces hazard pressure without increasing weapon damage.", Costs: map[string]int{"armor_blueprint": 1, "chitin_plate": 5, "cryo_shard": 4, "alien_alloy": 8}, Credits: 900},
			{ID: "craft_interceptor_hull", Output: "interceptor_hull", OutputName: "Interceptor Hull", Category: "hull", UnlockType: "account_bound", Blueprint: "hull_blueprint", BlueprintSource: "station contracts", Source: "Verdant-9 speed trials", Role: "mobility hull", TradeRule: "crafted unlock is account-bound", StatHint: "Faster repositioning with lighter cargo handling.", Costs: map[string]int{"hull_blueprint": 1, "alien_alloy": 14, "quantum_core": 1}, Credits: 1000},
			{ID: "craft_shield_drone", Output: "shield_drone", OutputName: "Shield Drone", Category: "drone", UnlockType: "account_bound", Blueprint: "drone_blueprint", BlueprintSource: "derelict fleet events", Source: "elite guardians", Role: "defensive support", TradeRule: "crafted unlock is account-bound", StatHint: "Adds protection windows, not passive damage.", Costs: map[string]int{"drone_blueprint": 1, "shield_emitter": 2, "drone_servo": 2, "plasma_battery": 2}, Credits: 1400},
			{ID: "craft_void_dash_module", Output: "void_dash_module", OutputName: "Void Dash Module", Category: "module", UnlockType: "account_bound", Blueprint: "module_blueprint", BlueprintSource: "boss side objectives", Source: "Void Leviathan", Role: "evasive utility", TradeRule: "crafted unlock is account-bound", StatHint: "Changes dash timing and escape routes.", Costs: map[string]int{"module_blueprint": 1, "void_crystal": 1, "phase_membrane": 1, "quantum_core": 2}, Credits: 2200},
			{ID: "craft_gravity_anchor", Output: "gravity_anchor", OutputName: "Gravity Anchor", Category: "ability", UnlockType: "account_bound", Blueprint: "ability_blueprint", BlueprintSource: "world event objectives", Source: "black hole events", Role: "crowd control", TradeRule: "crafted unlock is account-bound", StatHint: "Controls enemy lanes for team play.", Costs: map[string]int{"ability_blueprint": 1, "void_crystal": 1, "quantum_core": 2}, Credits: 1600},
			{ID: "craft_hive_trail", Output: "hive_trail", OutputName: "Hive Trail", Category: "cosmetic", UnlockType: "cosmetic", Blueprint: "cosmetic_blueprint", BlueprintSource: "rare boss drops", Source: "Hive Queen", Role: "prestige cosmetic", TradeRule: "cosmetic unlock; event variants may trade", StatHint: "Visual only. No gameplay power.", Costs: map[string]int{"cosmetic_blueprint": 1, "hive_egg": 1}, Credits: 750},
			{ID: "craft_umbra_coordinates", Output: "umbra_coordinates", OutputName: "Umbra Coordinates", Category: "planet_access", UnlockType: "account_bound", Blueprint: "coordinate_map", BlueprintSource: "derelict command rooms", Source: "Fractus Belt", Role: "planet access", TradeRule: "crafted access is account-bound", StatHint: "Unlocks travel to higher-risk void content.", Costs: map[string]int{"coordinate_map": 1, "quantum_core": 2, "plasma_battery": 3}, Credits: 1300},
		},
		Planets: []PlanetDef{
			{ID: "verdant", Name: "Verdant-9", Type: "forest", Threat: 1, Resources: []string{"alien_alloy", "hive_egg"}, Boss: "hive_queen", Hazards: []string{"spore_cloud"}},
		},
		Events: []EventDef{
			{ID: "alien_invasion", Name: "Alien Invasion", DurationMin: 60, Modifiers: []string{"enemy_density", "guild_objective"}},
		},
	}
	c.BuildIndexes()
	return c
}
