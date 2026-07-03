package game

import "fmt"

type Stack struct {
	ItemID string `json:"item_id"`
	Count  int    `json:"count"`
}

type Inventory struct {
	Items map[string]int `json:"items"`
}

func NewInventory() Inventory {
	return Inventory{Items: map[string]int{}}
}

func (i *Inventory) Add(item string, count int) {
	if i.Items == nil {
		i.Items = map[string]int{}
	}
	if count > 0 {
		i.Items[item] += count
	}
}

func (i *Inventory) Remove(item string, count int) bool {
	if count <= 0 || i.Items[item] < count {
		return false
	}
	i.Items[item] -= count
	if i.Items[item] == 0 {
		delete(i.Items, item)
	}
	return true
}

func (i Inventory) Has(cost map[string]int) bool {
	for item, count := range cost {
		if i.Items[item] < count {
			return false
		}
	}
	return true
}

func Craft(account *Account, recipe RecipeDef) error {
	if account.Credits < recipe.Credits {
		return fmt.Errorf("not enough credits")
	}
	if !account.Inventory.Has(recipe.Costs) {
		return fmt.Errorf("missing recipe components")
	}
	for item, count := range recipe.Costs {
		account.Inventory.Remove(item, count)
	}
	account.Credits -= recipe.Credits
	if account.Unlocks == nil {
		account.Unlocks = map[string]bool{}
	}
	account.Unlocks[recipe.Output] = true
	return nil
}

type MarketplaceListing struct {
	ID        string   `json:"id"`
	SellerID  PlayerID `json:"seller_id"`
	ItemID    string   `json:"item_id"`
	Quantity  int      `json:"quantity"`
	UnitPrice int      `json:"unit_price"`
	ExpiresAt int64    `json:"expires_at"`
}

type MarketStats struct {
	ItemID        string  `json:"item_id"`
	LastPrice     int     `json:"last_price"`
	MovingAverage float64 `json:"moving_average"`
	Volume24h     int     `json:"volume_24h"`
}
