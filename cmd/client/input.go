package main

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

var trackedKeys = []ebiten.Key{
	ebiten.KeyArrowUp,
	ebiten.KeyArrowDown,
	ebiten.KeyArrowLeft,
	ebiten.KeyArrowRight,
	ebiten.KeyW,
	ebiten.KeyA,
	ebiten.KeyS,
	ebiten.KeyD,
	ebiten.KeyEnter,
	ebiten.KeySpace,
	ebiten.KeyEscape,
}

func (a *App) justPressed(k ebiten.Key) bool {
	return ebiten.IsKeyPressed(k) && !a.keys[k]
}

func (a *App) captureKeys() {
	for _, key := range trackedKeys {
		a.keys[key] = ebiten.IsKeyPressed(key)
	}
}

func timeSeed() int64 {
	return time.Now().UnixNano()
}
