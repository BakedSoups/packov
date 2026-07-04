package main

import (
	"fmt"
	"os"

	"packov/internal/game"
)

func main() {
	path := "content/game.json"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	catalog, err := game.LoadCatalog(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "content validation failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("content ok: %d weapons, %d enemies, %d bosses, %d loot, %d recipes, %d planets, %d events\n",
		len(catalog.Weapons),
		len(catalog.Enemies),
		len(catalog.Bosses),
		len(catalog.Loot),
		len(catalog.Recipes),
		len(catalog.Planets),
		len(catalog.Events),
	)
}
