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
			{ID: "alien_alloy", Name: "Alien Alloy", Rarity: Common, BaseValue: 18},
			{ID: "plasma_battery", Name: "Plasma Battery", Rarity: Uncommon, BaseValue: 75},
			{ID: "quantum_core", Name: "Quantum Core", Rarity: Rare, BaseValue: 210},
			{ID: "hive_egg", Name: "Hive Egg", Rarity: Epic, BaseValue: 420},
			{ID: "boss_trophy", Name: "Boss Trophy", Rarity: Legendary, BaseValue: 1200},
		},
		Recipes: []RecipeDef{
			{ID: "craft_railgun", Output: "railgun", Costs: map[string]int{"plasma_battery": 3, "quantum_core": 2}, Credits: 1200},
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
