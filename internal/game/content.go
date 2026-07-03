package game

import (
	"encoding/json"
	"fmt"
	"os"
)

func LoadCatalog(path string) (*Catalog, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Catalog
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	c.index()
	return &c, c.Validate()
}

func (c *Catalog) index() {
	c.WeaponByID = map[string]WeaponDef{}
	c.AbilityByID = map[string]AbilityDef{}
	c.EnemyByID = map[string]EnemyDef{}
	c.BossByID = map[string]BossDef{}
	c.LootByID = map[string]LootDef{}
	c.RecipeByID = map[string]RecipeDef{}
	c.PlanetByID = map[string]PlanetDef{}
	c.EventByID = map[string]EventDef{}
	for _, v := range c.Weapons {
		c.WeaponByID[v.ID] = v
	}
	for _, v := range c.Abilities {
		c.AbilityByID[v.ID] = v
	}
	for _, v := range c.Enemies {
		c.EnemyByID[v.ID] = v
	}
	for _, v := range c.Bosses {
		c.BossByID[v.ID] = v
	}
	for _, v := range c.Loot {
		c.LootByID[v.ID] = v
	}
	for _, v := range c.Recipes {
		c.RecipeByID[v.ID] = v
	}
	for _, v := range c.Planets {
		c.PlanetByID[v.ID] = v
	}
	for _, v := range c.Events {
		c.EventByID[v.ID] = v
	}
}

func (c *Catalog) Validate() error {
	if len(c.Weapons) == 0 || len(c.Planets) == 0 || len(c.Enemies) == 0 {
		return fmt.Errorf("catalog must include weapons, planets, and enemies")
	}
	for _, r := range c.Recipes {
		for loot := range r.Costs {
			if _, ok := c.LootByID[loot]; !ok {
				return fmt.Errorf("recipe %s references unknown loot %s", r.ID, loot)
			}
		}
	}
	for _, p := range c.Planets {
		if _, ok := c.BossByID[p.Boss]; !ok {
			return fmt.Errorf("planet %s references unknown boss %s", p.ID, p.Boss)
		}
	}
	return nil
}
