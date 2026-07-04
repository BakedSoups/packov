.PHONY: test server wasm docker-up docker-down fmt ci docker-check ebitdock ebitdock-down ebitdock-doctor

test:
	go test ./...

fmt:
	gofmt -w cmd internal

server:
	go run ./cmd/server

wasm:
	GOOS=js GOARCH=wasm go build -o web/game.wasm ./cmd/client
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" web/wasm_exec.js

ci:
	go test ./internal/game ./internal/protocol ./internal/server ./cmd/server
	go run ./cmd/contentcheck content/game.json
	GOOS=js GOARCH=wasm go build -o /tmp/packov-game.wasm ./cmd/client
	GOOS=js GOARCH=wasm go build -o web/game.wasm ./cmd/client
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" web/wasm_exec.js
	go run ./cmd/websmoke web

docker-check:
	docker compose config --quiet

docker-up:
	docker compose up --build

docker-down:
	docker compose down

ebitdock:
	ebitdock dev

ebitdock-down:
	ebitdock down

ebitdock-doctor:
	ebitdock doctor
