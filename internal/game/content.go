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
	c.BuildIndexes()
	return &c, c.Validate()
}

func (c *Catalog) BuildIndexes() {
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
	recipeCategories := map[string]bool{}
	for _, r := range c.Recipes {
		if r.ID == "" || r.Output == "" {
			return fmt.Errorf("recipe must include id and output")
		}
		if r.Category == "" {
			return fmt.Errorf("recipe %s missing category", r.ID)
		}
		recipeCategories[r.Category] = true
		if r.Blueprint != "" {
			if _, ok := c.LootByID[r.Blueprint]; !ok {
				return fmt.Errorf("recipe %s references unknown blueprint %s", r.ID, r.Blueprint)
			}
			if r.Costs[r.Blueprint] <= 0 {
				return fmt.Errorf("recipe %s blueprint %s must be a cost", r.ID, r.Blueprint)
			}
		}
		for loot := range r.Costs {
			if _, ok := c.LootByID[loot]; !ok {
				return fmt.Errorf("recipe %s references unknown loot %s", r.ID, loot)
			}
		}
	}
	for _, category := range []string{"weapon", "armor", "hull", "drone", "module", "ability", "cosmetic", "planet_access"} {
		if !recipeCategories[category] {
			return fmt.Errorf("catalog missing recipe category %s", category)
		}
	}
	for _, p := range c.Planets {
		if _, ok := c.BossByID[p.Boss]; !ok {
			return fmt.Errorf("planet %s references unknown boss %s", p.ID, p.Boss)
		}
		for _, resource := range p.Resources {
			if _, ok := c.LootByID[resource]; !ok {
				return fmt.Errorf("planet %s references unknown resource %s", p.ID, resource)
			}
		}
	}
	return nil
}
