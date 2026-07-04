package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	root := "web"
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	index, err := os.ReadFile(filepath.Join(root, "index.html"))
	if err != nil {
		fail("read index.html: %v", err)
	}
	for _, ref := range [][]byte{[]byte("wasm_exec.js"), []byte("game.wasm")} {
		if !bytes.Contains(index, ref) {
			fail("index.html missing %s", ref)
		}
	}
	wasm, err := os.ReadFile(filepath.Join(root, "game.wasm"))
	if err != nil {
		fail("read game.wasm: %v", err)
	}
	if len(wasm) < 4 || !bytes.Equal(wasm[:4], []byte{0x00, 0x61, 0x73, 0x6d}) {
		fail("game.wasm does not have a wasm module header")
	}
	if _, err := os.Stat(filepath.Join(root, "wasm_exec.js")); err != nil {
		fail("missing wasm_exec.js: %v", err)
	}
	fmt.Printf("web smoke ok: %s\n", root)
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
